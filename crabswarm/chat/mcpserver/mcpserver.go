// Package mcpserver is the per-agent bridge between a harness that speaks MCP
// and the chat broker the crabswarm daemon hosts.
//
// A harness starts this as a stdio subprocess of its own, so an agent gets the
// chat verbs as tools it is offered rather than as commands it has to remember
// to type — and, because the bridge attends the room the moment it starts and
// keeps attending for the rest of the session, it is a member before its first
// turn instead of whenever it first thinks to say something, and a member again
// once a daemon that went away comes back.
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

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
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
	// token is the value the caller was configured with, kept as it was given.
	// It is resolved against the environment where it is used rather than here,
	// so a harness that starts the bridge with no identity at all gets a server
	// whose tools say what is missing instead of a subprocess that exited
	// before the handshake.
	token string
	mcp   *mcp.Server

	// joinMu serializes attendance: the attend loop and a tool call that
	// arrived before it succeeded would otherwise both ask, and the second
	// answer would tell the first nothing it did not already know.
	joinMu sync.Mutex
	joined bool

	// The attend loop's schedule, held here rather than read from the constants
	// below so a test can drive the loop at a pace it can wait for. [New] takes
	// the constants.
	joinBackoffBase time.Duration
	joinBackoffMax  time.Duration
}

// New dials sockPath and prepares the MCP server. Attendance is declared from
// Run, so nothing about the daemon — being down, refusing this caller, not
// existing yet — keeps the harness from getting a server it can talk to.
//
// token is resolved the way every member verb resolves it — the value given
// here first, then the environment — but not until something needs it, so a
// bridge started with no identity at all still answers the handshake and
// reports the missing token through every tool the agent calls. A nil logger
// discards logs; a logger writing to stdout would corrupt the MCP stream, so
// the caller owns that choice.
func New(logger *slog.Logger, sockPath, token string) (*Server, error) {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	client, err := cli.Dial(sockPath)
	if err != nil {
		return nil, err
	}
	s := &Server{
		logger:          logger,
		client:          client,
		token:           token,
		joinBackoffBase: joinBackoffBase,
		joinBackoffMax:  joinBackoffMax,
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
	if _, err := cli.ResolveToken(s.token); err != nil {
		// Neither the flag nor the environment changes while the process runs,
		// so a bridge with no identity to resolve has nothing to attend with
		// and nothing to watch for, and retrying would only say so again. It
		// still serves: the tools and the resources answer with this same
		// refusal, which is where the agent reads what is missing.
		s.logger.Error("no chat identity; the tools will report it", "error", err)
	} else {
		g.Go(func() error {
			s.attend(gctx)
			return nil
		})
		g.Go(func() error {
			s.watchMembers(gctx)
			return nil
		})
	}
	g.Go(func() error {
		// The session ending ends the attendance loop and the feed too: a
		// bridge whose harness is gone has nobody left to attend for, and
		// without this they would hold the session open for the rest of their
		// backoff.
		defer cancel()
		return s.mcp.Run(gctx, transport)
	})
	return g.Wait()
}

// How long the bridge waits between attempts at attending, and — since asking
// again costs nothing once it is a member — how often it looks. The first
// retries are quick, for the ordinary case of a bridge that started while the
// daemon was still binding its socket; the ceiling is what keeps a daemon that
// is down from being asked in a loop.
const (
	joinBackoffBase = 200 * time.Millisecond
	joinBackoffMax  = 2 * time.Second
)

// How many consecutive failures pass between the lines a retry loop logs. The
// first of a run is always reported, since that is the one that says what
// broke; after it a loop that keeps trying for the rest of the session would
// bury everything else on the harness's stderr, and the second identical line
// says nothing the first did not. Against the ceilings the two loops retry at,
// this is a line every half minute or so.
const warnEvery = 15

// warnRetry reports a failed attempt — the first of a run, and every warnEvery
// after it.
func (s *Server) warnRetry(msg string, failures int, backoff time.Duration, err error) {
	if failures > 1 && failures%warnEvery != 0 {
		return
	}
	s.logger.Warn(msg, "failures", failures, "backoff", backoff, "error", err)
}

// joinTimeout bounds one attempt at attending. The lock is held across the
// call, so a daemon that accepted the connection and then never answered would
// otherwise wedge every tool call behind it — including the ones whose own
// deadline has already passed, which would be waiting on an answer nobody is
// left to read. A local socket answers in microseconds; this is only the point
// past which no answer is coming.
const joinTimeout = 10 * time.Second

// attend keeps this member attending for the whole session.
//
// It never gives up, because attendance is not something the agent asked for
// and so not something it will notice missing. A bridge starts with its
// harness, which is regularly before the daemon is up at all, and the daemon
// can go away and come back underneath it; either way a message addressed to
// this member needs an inbox to land in and a hook reporting its state needs a
// member to report about, both before the agent takes its first turn. A loop
// that stopped after a handful of tries would leave the room a member short
// until the agent happened to call a tool, which is exactly the moment it is
// too late.
//
// Attendance is asked for on a cadence rather than only when something clears
// it: the refusal that says the daemon has forgotten this member arrives on
// whichever goroutine happened to make a call, and a cadence needs no wiring
// between them. Asking again costs nothing while the membership stands —
// [Server.ensureJoined] answers from what it remembers, without a round trip.
func (s *Server) attend(ctx context.Context) {
	backoff := s.joinBackoffBase
	failures := 0
	for {
		wait := s.joinBackoffMax
		if _, err := s.ensureJoined(ctx); err != nil {
			// A session ending while the join was in flight fails it, and that
			// is the shutdown rather than something to report.
			if ctx.Err() != nil {
				return
			}
			failures++
			s.warnRetry("attending the chat room failed; retrying", failures, backoff, err)
			wait = backoff
			backoff = min(2*backoff, s.joinBackoffMax)
		} else {
			backoff = s.joinBackoffBase
			failures = 0
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

// ensureJoined makes sure the caller is attending before a tool acts on its
// behalf, and hands back the identity token it attends with. Join is
// idempotent for a known token, so asking again costs one round trip — and it
// is what lets a tool succeed against a daemon that has only just come up.
//
// The token is resolved here rather than held from startup, and its refusal is
// returned as it stands: it names the flag and the two variables an identity
// can come from, which is the whole of what the reader of a failed tool call
// can do about it.
//
// Attendance already declared is remembered rather than re-declared per call,
// so the round trip is spent once; [Server.forgetJoined] is what puts the
// memory back when the daemon stops counting this member as one.
func (s *Server) ensureJoined(ctx context.Context) (string, error) {
	token, err := cli.ResolveToken(s.token)
	if err != nil {
		return "", err
	}
	s.joinMu.Lock()
	defer s.joinMu.Unlock()
	if s.joined {
		return token, nil
	}
	ctx, cancel := context.WithTimeout(ctx, joinTimeout)
	defer cancel()
	// The empty name takes the one the daemon derives from the token: an agent
	// is named by whoever registered it, not by the harness it happens to run.
	//
	// Always as an agent: this bridge is started by a harness and serves
	// nothing else, so the terminal behind it is one a nudge belongs in.
	var identity strings.Builder
	if err := s.client.Join(ctx, &identity, token, "",
		chatv1.MemberKind_MEMBER_KIND_AGENT); err != nil {
		return "", fmt.Errorf("attending the chat room: %w", err)
	}
	s.joined = true
	s.logger.Info("attending the chat room", "identity", strings.TrimSpace(identity.String()))
	return token, nil
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
