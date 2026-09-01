// Package mcpserver is the per-agent bridge between a harness that speaks MCP
// and the chat broker the crabswarm daemon hosts.
//
// A harness starts this as a stdio subprocess of its own, so an agent gets the
// chat verbs as tools it is offered rather than as commands it has to remember
// to type — and, because the bridge attends the room the moment it starts, it
// is a member before its first turn instead of whenever it first thinks to say
// something.
//
// No chat logic lives here. Every tool forwards to the same ChatService call
// the matching `crabswarm chat` subcommand makes, through the same
// [cli.Client], and hands back the text that client rendered. That is what
// keeps a tool result and a CLI verb's output the same words: a room reads the
// same whether a member is wired to it through MCP or through a shell.
//
// Beside the tools, the room itself is offered as resources a harness can hold
// open: its attendance, which is subscribable, so a view of who is around and
// what each of them is doing costs no turn, and its transcript, for catching up
// on what the room has been saying. The roster is answered as structured data
// rather than in the CLI's words, since its reader is the harness; the
// transcript is answered in them, since every line already carries everything
// an entry holds.
package mcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
	"github.com/ngicks/crabswarm/internal/libver"
)

// serverName identifies the bridge to the harness. It names the room it
// bridges to rather than the process it runs in: the harness lists it beside
// every other MCP server it was configured with.
const serverName = "crabswarm-chat"

// Server bridges one agent's MCP stdio session to the chat daemon.
type Server struct {
	logger *slog.Logger
	client *cli.Client
	token  string
	mcp    *mcp.Server

	// joinMu serializes attendance: the startup retry and a tool call that
	// arrived before it succeeded would otherwise both ask, and the second
	// answer would tell the first nothing it did not already know.
	joinMu sync.Mutex
	joined bool

	// watchWanted is closed by the first subscription to the members resource,
	// which is what starts the room's event feed. A channel rather than a
	// context held on the struct: the feed belongs to the session Run serves,
	// and this is how the subscription reaches the goroutine that owns it.
	watchWanted chan struct{}
	watchOnce   sync.Once
}

// New dials sockPath, resolves identity for token, and prepares the MCP
// server. Join happens in Run so failures surface as MCP tool errors, not a
// dead harness.
//
// token is resolved the way every member verb resolves it — the value given
// here first, then the environment — so a bridge configured with no token at
// all still inherits the identity cmdman gave the agent. A nil logger
// discards logs; a logger writing to stdout would corrupt the MCP stream, so
// the caller owns that choice.
func New(logger *slog.Logger, sockPath, token string) (*Server, error) {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	token, err := cli.ResolveToken(token)
	if err != nil {
		return nil, err
	}
	client, err := cli.Dial(sockPath)
	if err != nil {
		return nil, err
	}
	s := &Server{
		logger:      logger,
		client:      client,
		token:       token,
		watchWanted: make(chan struct{}),
	}
	s.mcp = mcp.NewServer(
		&mcp.Implementation{Name: serverName, Version: libver.Version},
		&mcp.ServerOptions{
			Logger:             logger,
			SubscribeHandler:   s.subscribed,
			UnsubscribeHandler: s.unsubscribed,
		},
	)
	s.addTools()
	s.addResources()
	return s, nil
}

// Run serves MCP on stdin/stdout until ctx is done.
//
// It serves the one session its harness starts it for and closes the
// connection to the daemon on the way out, so a Server is not reusable: a
// harness that reconnects starts a new process, which is the only thing a
// stdio server can mean by a new session.
func (s *Server) Run(ctx context.Context) error {
	return s.serve(ctx, &mcp.StdioTransport{})
}

// serve is [Server.Run] with the transport injected, so a test can drive a
// session over an in-memory pipe instead of the process's own stdio.
func (s *Server) serve(ctx context.Context, transport mcp.Transport) error {
	defer func() { _ = s.client.Close() }()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		s.joinWithRetry(gctx)
		return nil
	})
	g.Go(func() error {
		// Waits for a subscription before it watches anything, so the feed
		// exists only for a session that asked for it and ends with that
		// session rather than with the process.
		s.watchMembers(gctx)
		return nil
	})
	g.Go(func() error {
		// The session ending ends the attendance retry too: a bridge whose
		// harness is gone has nobody left to attend for, and without this the
		// retry would hold the session open for the rest of its backoff.
		defer cancel()
		return s.mcp.Run(gctx, transport)
	})
	return g.Wait()
}

// How long the bridge keeps asking to attend before it settles for serving
// tools that report why they cannot. A handful of tries covers a bridge that
// started while the daemon was still binding its socket; past that the daemon
// is not coming up on its own, and each tool call retries anyway.
const (
	joinAttempts    = 5
	joinBackoffBase = 200 * time.Millisecond
	joinBackoffMax  = 2 * time.Second
)

// joinTimeout bounds one attempt at attending. The lock is held across the
// call, so a daemon that accepted the connection and then never answered would
// otherwise wedge every tool call behind it — including the ones whose own
// deadline has already passed, which would be waiting on an answer nobody is
// left to read. A local socket answers in microseconds; this is only the point
// past which no answer is coming.
const joinTimeout = 10 * time.Second

// joinWithRetry declares attendance as soon as the process starts, so a
// message addressed to this member has an inbox to land in before its harness
// takes a turn.
//
// Running out of attempts is not fatal. A bridge that exited on a daemon it
// could not reach would leave the harness holding a subprocess that died
// during startup, with no chat and nothing to read about why; a bridge that
// stays up answers the same question through every tool the agent calls.
func (s *Server) joinWithRetry(ctx context.Context) {
	backoff := joinBackoffBase
	for attempt := 1; ; attempt++ {
		err := s.ensureJoined(ctx)
		if err == nil {
			return
		}
		if attempt >= joinAttempts {
			s.logger.Error("giving up on attending the chat room; the tools will report it",
				"attempts", attempt, "error", err)
			return
		}
		s.logger.Warn("attending the chat room failed; retrying",
			"attempt", attempt, "backoff", backoff, "error", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(2*backoff, joinBackoffMax)
	}
}

// ensureJoined makes sure the caller is attending before a tool acts on its
// behalf. Join is idempotent for a known token, so asking again costs one
// round trip — and it is what lets a tool succeed against a daemon that came
// back after the startup attempts ran out.
//
// Attendance already declared is remembered rather than re-declared per call,
// so the round trip is spent once; [Server.forgetJoined] is what puts the
// memory back when the daemon stops counting this member as one.
func (s *Server) ensureJoined(ctx context.Context) error {
	s.joinMu.Lock()
	defer s.joinMu.Unlock()
	if s.joined {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, joinTimeout)
	defer cancel()
	// The empty name takes the one the daemon derives from the token: an agent
	// is named by whoever registered it, not by the harness it happens to run.
	var identity strings.Builder
	if err := s.client.Join(ctx, &identity, s.token, ""); err != nil {
		return fmt.Errorf("attending the chat room: %w", err)
	}
	s.joined = true
	s.logger.Info("attending the chat room", "identity", strings.TrimSpace(identity.String()))
	return nil
}

// forgetJoined drops the remembered attendance when err is the daemon refusing
// a caller it does not count as a member, so the next call declares it again.
// It hands err back unchanged, so the call site returns it in place.
//
// Attendance outlives the bridge's memory of it in more than one way: the
// daemon reaps a member whose command the team-info provider stopped knowing,
// a human types `crabswarm chat leave`, a restarted daemon comes back on a
// fresh database. Without this the bridge would keep acting on a membership
// that no longer exists — every tool failing the same way forever, and the
// watch loop spending its backoff on the same refusal — with a join it is
// already sure it made standing in the way of the one that would fix it.
//
// Unauthenticated alone: that is the code the daemon answers a caller it
// cannot resolve to a member with. NotFound means the member the caller
// addressed does not exist, which attending again would not change.
func (s *Server) forgetJoined(err error) error {
	if status.Code(err) != codes.Unauthenticated {
		return err
	}
	s.joinMu.Lock()
	defer s.joinMu.Unlock()
	s.joined = false
	return err
}
