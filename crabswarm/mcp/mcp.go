// Package mcp is the crabswarm MCP server one harness spawns per agent; chat is
// its first tool family.
//
// A harness starts it as a stdio subprocess of its own, so an agent gets
// crabswarm's verbs as tools it is offered rather than as commands it has to
// remember to type — and, because the server attends the room the moment it
// starts and holds that attendance for the rest of the session, the agent is a
// member before its first turn instead of whenever it first thinks to say
// something, and a member again once a daemon that went away comes back.
//
// Attendance is one open stream, and holding it is the whole of what membership
// means. The server opens it at startup and opens it again every time it ends,
// so there is nothing for a tool to declare: a tool called while the stream is
// down reports that instead of acting as somebody the room does not have.
//
// What the server offers is not its own. A tool family is a sub-package that
// registers its tools and its resources onto a [Server] before it runs, so this
// package knows the room it attends and the harness it answers, and each family
// knows the verbs it serves.
package mcp

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
	"github.com/ngicks/crabswarm/internal/libver"
)

// serverName identifies the server to the harness, which lists it beside every
// other MCP server it was configured with. It names crabswarm rather than the
// family it happens to carry: one agent gets one of these, and what it offers
// grows without the harness's configuration changing.
const serverName = "crabswarm-mcp"

// Server bridges one agent's MCP stdio session to the crabswarm daemon.
type Server struct {
	logger *slog.Logger
	client *cli.Client
	// token is the value the caller was configured with, kept as it was given.
	// It is resolved against the environment where it is used rather than here,
	// so a harness that starts the server with no identity at all gets one whose
	// tools say what is missing instead of a subprocess that exited before the
	// handshake.
	token string
	mcp   *mcpsdk.Server

	// subscribable are the resource URIs a family registered, which are the ones
	// a harness may ask to be told about. Written while the families register
	// and only read once the session is up.
	subscribable map[string]struct{}
	// rosterResources are the resources a change in who attends the room makes
	// stale. Written and read like subscribable.
	rosterResources []string

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
// It comes back with no tools and no resources: a family registers its own onto
// it, and everything registered before [Server.Run] is advertised during the
// handshake rather than announced as a change the harness has to notice.
//
// token is resolved the way every member verb resolves it — the value given
// here first, then the environment — but not until something needs it, so a
// server started with no identity at all still answers the handshake and
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
		subscribable:      map[string]struct{}{},
		settled:           make(chan struct{}),
		attendBackoffBase: attendBackoffBase,
		attendBackoffMax:  attendBackoffMax,
	}
	s.mcp = mcpsdk.NewServer(
		&mcpsdk.Implementation{Name: serverName, Version: libver.Version},
		&mcpsdk.ServerOptions{
			Logger:             logger,
			SubscribeHandler:   s.subscribed,
			UnsubscribeHandler: s.unsubscribed,
		},
	)
	return s, nil
}

// MCP is the SDK server a family adds its tools to. It is handed over rather
// than wrapped because the SDK adds a tool through a generic function, which a
// method here could not be.
func (s *Server) MCP() *mcpsdk.Server {
	return s.mcp
}

// Client is the connection to the daemon every tool acts through.
func (s *Server) Client() *cli.Client {
	return s.client
}

// ResolveToken resolves the caller's identity the way every member verb does:
// the value the server was configured with first, then the environment. A
// family resolves it per call rather than once, since a server that resolves
// none still serves and answers with what is missing.
func (s *Server) ResolveToken() (string, error) {
	return cli.ResolveToken(s.token)
}

// AddResource registers a resource and lets a harness subscribe to it.
//
// Subscribing goes through the server because the SDK takes one handler for the
// whole session: a URI nothing registered is refused rather than accepted
// quietly, which is what keeps a harness from waiting on news that could never
// come.
func (s *Server) AddResource(res *mcpsdk.Resource, handler mcpsdk.ResourceHandler) {
	s.subscribable[res.URI] = struct{}{}
	s.mcp.AddResource(res, handler)
}

// AnnounceOnRosterChange has the subscribed sessions told to read uri again
// whenever the room's attendance or anyone's state changes — including after an
// attendance the server had to open again, since whatever happened while
// nothing was attending went unannounced.
func (s *Server) AnnounceOnRosterChange(uri string) {
	s.rosterResources = append(s.rosterResources, uri)
}

// Run serves MCP on stdin/stdout until ctx is done.
//
// It serves the one session its harness starts it for and closes the
// connection to the daemon on the way out, so a Server is not reusable: a
// harness that reconnects starts a new process, which is the only thing a
// stdio server can mean by a new session.
func (s *Server) Run(ctx context.Context) error {
	return s.Serve(ctx, &mcpsdk.StdioTransport{})
}

// Serve is [Server.Run] with the transport given, so a caller can drive a
// session over something other than the process's own stdio.
func (s *Server) Serve(ctx context.Context, transport mcpsdk.Transport) error {
	defer func() { _ = s.client.Close() }()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	g, gctx := errgroup.WithContext(ctx)
	if _, err := s.ResolveToken(); err != nil {
		// Neither the flag nor the environment changes while the process runs,
		// so a server with no identity to resolve has nothing to attend with,
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
		// The session ending ends the attendance too: a server whose harness is
		// gone has nobody left to attend for, and without this the loop would
		// hold the session open for the rest of its backoff.
		defer cancel()
		return s.mcp.Run(gctx, transport)
	})
	return g.Wait()
}

// How long the server waits before opening the attendance stream again. The
// first retries are quick, for the ordinary case of one that started while the
// daemon was still binding its socket; the ceiling is what keeps a daemon that
// is down from being asked in a loop.
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
// and so not something it will notice missing. A server starts with its
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
	token, err := s.ResolveToken()
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
	// Always as an agent: this server is started by a harness and serves
	// nothing else, so the terminal behind it is one a nudge belongs in.
	//
	// The harness goes unnamed and the delivery is the terminal one, which is
	// how every agent has been reached so far. Reading the harness off the MCP
	// handshake and delivering a mention through its own channel is work this
	// server does not do yet, and declaring either before it does would have the
	// daemon stop typing at a member nothing else would reach.
	attendance, err := s.client.Attend(actx, token, "",
		chatv1.MemberKind_MEMBER_KIND_AGENT,
		chatv1.Harness_HARNESS_UNSPECIFIED,
		chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL)
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
		s.announceRoster(ctx)
	}
	// The first event of the feed is this member's own arrival, which the
	// daemon publishes to the room it just joined. It is announced like any
	// other roster change: the room did gain a member, and every attendee sees
	// the same feed.
	err = attendance.Forward(actx, func(ev *chatv1.RoomEvent) error {
		if rosterChanged(ev) {
			s.announceRoster(ctx)
		}
		return nil
	})
	s.attendanceEnded(err)
	return true, err
}

// announceRoster tells the subscribed sessions to read the resources a roster
// change made stale.
func (s *Server) announceRoster(ctx context.Context) {
	for _, uri := range s.rosterResources {
		err := s.mcp.ResourceUpdated(ctx,
			&mcpsdk.ResourceUpdatedNotificationParams{URI: uri})
		if err != nil {
			s.logger.Warn("announcing the changed room roster failed",
				"uri", uri, "error", err)
		}
	}
}

// rosterChanged reports whether ev changes who attends the room or what state
// they are in. A message being appended does not: the same members are there in
// the same states. It changes the room's conversation, but nothing subscribes
// to that — the chat family's read verb is what a member reads its messages
// with, and a subscription the server accepted would be one it could not serve.
func rosterChanged(ev *chatv1.RoomEvent) bool {
	switch ev.GetEvent().(type) {
	case *chatv1.RoomEvent_MemberStateChanged,
		*chatv1.RoomEvent_MemberJoined,
		*chatv1.RoomEvent_MemberLeft:
		return true
	default:
		return false
	}
}

// subscribed accepts a request to be told about a resource. The room's feed is
// already up — it is the attendance the server holds for the whole session — so
// there is nothing to start here; what is left is deciding whether the server
// can keep the promise a subscription asks for.
func (s *Server) subscribed(_ context.Context, req *mcpsdk.SubscribeRequest) error {
	return s.announceable(req.Params.URI)
}

// unsubscribed acknowledges the withdrawal and leaves the feed running.
//
// The feed is not the subscription's to end: it is the attendance itself, which
// runs for as long as the session does whether or not anything is listening.
// Nothing is announced to a harness that withdrew — the SDK sends a resource
// update to the sessions that subscribed and to no others. The SDK also
// requires this handler as soon as [Server.subscribed] exists.
func (s *Server) unsubscribed(_ context.Context, req *mcpsdk.UnsubscribeRequest) error {
	return s.announceable(req.Params.URI)
}

// announceable returns nil when the server can tell a harness that uri changed,
// and the refusal otherwise.
//
// A URI no family registered is refused rather than accepted quietly. The SDK
// records a subscription for whatever URI it is handed, so accepting a typo
// would leave the harness waiting on news that could never come.
func (s *Server) announceable(uri string) error {
	if _, ok := s.subscribable[uri]; ok {
		return nil
	}
	return mcpsdk.ResourceNotFoundError(uri)
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

// AwaitAttendance blocks until the server has an answer about its attendance
// and returns nil when it has one. A family waits on it before acting, since
// none of the daemon's member calls mean anything from outside the room.
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
func (s *Server) AwaitAttendance(ctx context.Context) error {
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
