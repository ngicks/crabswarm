// Package mcpserver is the per-agent bridge between a harness that speaks MCP
// and the chat broker the crabswarm daemon hosts.
//
// A harness starts this as a stdio subprocess of its own, so an agent gets the
// chat verbs as tools it is offered rather than as commands it has to remember
// to type — and, because the bridge attends the room the moment it starts and
// holds that attendance for the rest of the session, it is a member before its
// first turn instead of whenever it first thinks to say something, and a member
// again once a daemon that went away comes back.
//
// Attendance is one open stream, and holding it is the whole of what membership
// means. The bridge opens it at startup and opens it again every time it ends,
// so there is nothing for a tool to declare: a tool called while the stream is
// down reports that instead of acting as somebody the room does not have.
//
// No chat logic lives here. Every tool forwards to the same ChatService call
// the matching `crabswarm chat` subcommand makes, through the same
// [cli.Client], and hands back the text that client rendered. That is what
// keeps a tool result and a CLI verb's output the same words: a room reads the
// same whether a member is wired to it through MCP or through a shell.
//
// Beside the tools, the room's attendance is offered as a resource a harness
// can hold open and subscribe to, so a view of who is around and what each of
// them is doing costs no turn. It is answered as structured data rather than in
// the CLI's words, since its reader is the harness rather than the model.
package mcpserver

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"

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

	// mu guards what the attendance loop reports about itself, which is what
	// every tool call reads before acting.
	mu sync.Mutex
	// attended says whether the attendance stream is open right now.
	attended bool
	// attendErr is why it is not, as the last attempt ended — the daemon's own
	// refusal where there was one, which is what a failed tool call hands the
	// agent to read.
	attendErr error
	// settled is closed once the first attempt has either landed or failed, so
	// a tool call that arrived during startup waits for an answer instead of
	// reporting an attendance that simply has not happened yet.
	settled    chan struct{}
	settleOnce sync.Once

	// The loop's schedule, held here rather than read from the constants below
	// so a test can drive it at a pace it can wait for. [New] takes the
	// constants.
	attendBackoffBase time.Duration
	attendBackoffMax  time.Duration
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
		logger:            logger,
		client:            client,
		token:             token,
		settled:           make(chan struct{}),
		attendBackoffBase: attendBackoffBase,
		attendBackoffMax:  attendBackoffMax,
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
		// so a bridge with no identity to resolve has nothing to attend with,
		// and retrying would only say so again. Recording the refusal as the
		// answer to attendance is what keeps a tool call from waiting on a
		// loop that will never run; it still serves, and the tools report this
		// same refusal, which is where the agent reads what is missing.
		s.attendanceEnded(err)
		s.logger.Error("no chat identity; the tools will report it", "error", err)
	} else {
		g.Go(func() error {
			s.attend(gctx)
			return nil
		})
	}
	g.Go(func() error {
		// The session ending ends the attendance too: a bridge whose harness is
		// gone has nobody left to attend for, and without this the loop would
		// hold the session open for the rest of its backoff.
		defer cancel()
		return s.mcp.Run(gctx, transport)
	})
	return g.Wait()
}

// How long the bridge waits before opening the attendance stream again. The
// first retries are quick, for the ordinary case of a bridge that started while
// the daemon was still binding its socket; the ceiling is what keeps a daemon
// that is down from being asked in a loop.
const (
	attendBackoffBase = 200 * time.Millisecond
	attendBackoffMax  = 2 * time.Second
)

// How many consecutive failures pass between the lines the loop logs. The
// first of a run is always reported, since that is the one that says what
// broke; after it a loop that keeps trying for the rest of the session would
// bury everything else on the harness's stderr, and the second identical line
// says nothing the first did not. Against the ceiling the loop retries at, this
// is a line every half minute.
const warnEvery = 15

// warnRetry reports a failed attempt — the first of a run, and every warnEvery
// after it.
func (s *Server) warnRetry(msg string, failures int, backoff time.Duration, err error) {
	if failures > 1 && failures%warnEvery != 0 {
		return
	}
	s.logger.Warn(msg, "failures", failures, "backoff", backoff, "error", err)
}

// attendTimeout bounds the opening of one attendance. The stream is lazy, so a
// daemon that accepted the connection and then answered nothing would leave the
// loop waiting for the rest of the session and every tool call waiting with it.
// A local socket answers in microseconds; this is only the point past which no
// answer is coming.
const attendTimeout = 10 * time.Second

// errNotAttending is what a tool call is refused with when the attendance is
// down for a reason nothing recorded. Every path that ends an attendance
// records why, so this stands in for nothing that happens today; it exists so a
// refusal is never the empty error.
var errNotAttending = errors.New("not attending the chat room")

// attend keeps this member attending for the whole session: one open stream,
// opened again every time it ends.
//
// It never gives up, because attendance is not something the agent asked for
// and so not something it will notice missing. A bridge starts with its
// harness, which is regularly before the daemon is up at all, and the daemon
// can go away and come back underneath it; either way a message addressed to
// this member needs a member to be addressed to, and a hook reporting harness
// state needs one to report about, both before the agent takes its first turn.
// A loop that stopped after a handful of tries would leave the room a member
// short until the agent happened to call a tool, which is exactly the moment it
// is too late.
func (s *Server) attend(ctx context.Context) {
	backoff := s.attendBackoffBase
	failures := 0
	for reopened := false; ; reopened = true {
		landed, err := s.holdAttendance(ctx, reopened)
		if ctx.Err() != nil {
			return
		}
		// An attendance that stood for a while and then ended is not the
		// trouble one that never opened is, so it starts its retries over
		// rather than inheriting the wait the previous failure had climbed to.
		if landed {
			backoff = s.attendBackoffBase
			failures = 0
		}
		failures++
		s.warnRetry("the chat attendance ended; attending again", failures, backoff, err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(2*backoff, s.attendBackoffMax)
	}
}

// holdAttendance opens one attendance and holds it until it ends, reporting
// whether it ever opened and why it is over.
//
// reopened says an earlier attendance had already ended, which makes the roster
// stale by definition: whatever happened while nothing was attending went
// unannounced. Saying so as soon as the new stream is up closes that gap — the
// subscriber reads the resource, and the read lists the room afresh.
//
// The stream gets a context of its own so giving up on an opening that never
// answers ends nothing but that attempt. The parent context lives as long as
// the session.
func (s *Server) holdAttendance(ctx context.Context, reopened bool) (bool, error) {
	token, err := cli.ResolveToken(s.token)
	if err != nil {
		s.attendanceEnded(err)
		return false, err
	}
	actx, cancel := context.WithCancel(ctx)
	defer cancel()
	// A deadline on the context would end the attendance itself ten seconds in,
	// so the wait is bounded by cancelling the attempt instead and only until
	// the daemon has answered.
	opening := time.AfterFunc(attendTimeout, cancel)
	// The empty name takes the one the daemon derives from the token: an agent
	// is named by whoever registered it, not by the harness it happens to run.
	//
	// Always as an agent: this bridge is started by a harness and serves
	// nothing else, so the terminal behind it is one a nudge belongs in.
	attendance, err := s.client.Attend(actx, token, "", chatv1.MemberKind_MEMBER_KIND_AGENT)
	if !opening.Stop() && err != nil {
		err = errors.New("the daemon did not answer the attendance within " +
			attendTimeout.String())
	}
	if err != nil {
		s.attendanceEnded(err)
		return false, err
	}
	s.attendanceLanded(attendance.Self())
	if reopened {
		s.membersChanged(ctx)
	}
	// The first event of the feed is this bridge's own arrival, which the
	// daemon publishes to the room it just joined. It is announced like any
	// other roster change: the room did gain a member, and every attendee sees
	// the same feed.
	err = attendance.Forward(actx, func(ev *chatv1.RoomEvent) error {
		if rosterChanged(ev) {
			s.membersChanged(ctx)
		}
		return nil
	})
	s.attendanceEnded(err)
	return true, err
}

// attendanceLanded records an open attendance, which is what lets a tool act on
// behalf of this member.
func (s *Server) attendanceLanded(self *chatv1.Member) {
	s.mu.Lock()
	s.attended = true
	s.attendErr = nil
	s.mu.Unlock()
	s.settle()
	s.logger.Info("attending the chat room",
		"member", cli.Address(self), "room", self.GetRoom())
}

// attendanceEnded records that there is no attendance and why, which is what a
// tool call is refused with until the loop opens the next one.
func (s *Server) attendanceEnded(err error) {
	s.mu.Lock()
	s.attended = false
	s.attendErr = err
	s.mu.Unlock()
	s.settle()
}

// settle releases the tool calls waiting for the first attempt to finish. Every
// attempt after that finds the gate already open and answers from what the last
// one recorded.
func (s *Server) settle() {
	s.settleOnce.Do(func() { close(s.settled) })
}

// awaitAttendance blocks until the bridge has an answer about its attendance
// and returns nil when it has one.
//
// The wait is only ever for the first attempt: a harness starts its MCP
// subprocesses before the services they talk to, so a tool called in the first
// moments of a session would otherwise be refused for no better reason than
// having arrived first.
//
// After that a call is answered from what the loop last recorded. A tool called
// in the gap between an attendance ending and the next one opening is refused
// with why the last one ended — the gap is a backoff wide and the agent can
// call again, where waiting it out would hand the model a tool that sometimes
// blocks for seconds with nothing to show for it.
func (s *Server) awaitAttendance(ctx context.Context) error {
	select {
	case <-s.settled:
	case <-ctx.Done():
		return ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.attended:
		return nil
	case s.attendErr != nil:
		return s.attendErr
	default:
		return errNotAttending
	}
}
