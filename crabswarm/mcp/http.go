package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"

	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// TokenHeader is the HTTP request header an MCP session names its chat
// identity in. A launcher that runs one harness server for several agents sets
// it per agent, which is what makes each of them a member of its own.
const TokenHeader = "X-Crabswarm-Token"

// SharedServerName is the name a harness's configuration declares
// [HTTPServer] under, which is the name a call through the harness server
// addresses it by. apm takes two packages declaring a server of the same name
// for one server, so the name differs from the stdio server's.
const SharedServerName = "crabswarm-mcp-shared"

// HTTPPath is where [HTTPServer] serves MCP.
const HTTPPath = "/mcp"

// errNoTokenHeader is what every tool of a session that named no identity
// answers with. It names the header and nothing else: a session on a shared
// server never falls back to the process's own identity, since every session
// that forgot the header would otherwise act as one and the same member.
var errNoTokenHeader = errors.New("no chat identity token: this MCP session sent no " +
	TokenHeader + " header, which is what names the member a session acts as")

// readHeaderTimeout bounds how long a client may take to send its request
// headers. A local harness sends them at once; this only keeps a connection
// that never does from being held open for nothing.
const readHeaderTimeout = 10 * time.Second

// httpKeepAlive is how often every HTTP session is pinged. A client that goes
// away without the DELETE that ends its session — killed, or its host gone —
// leaves nothing else behind that says so, and its member would attend for as
// long as the server runs. A session that leaves [keepAliveFailures] pings in a
// row unanswered is closed, which releases it like a DELETE does, so such a
// member leaves about a minute and a half after its client did.
//
// A ping travels on the event stream a client holds open for its session, and
// both Codex and OpenCode hold one open and answer every ping on it, so a live
// session is never closed by this.
const httpKeepAlive = 30 * time.Second

// HTTPServer serves MCP sessions over streamable HTTP, many at once, and
// attends the room once per identity token among them.
//
// A session names the member it acts as in the [TokenHeader] of its initialize
// request, and keeps that member for its whole life. Sessions naming one token
// share one member, which attends while at least one of them is open and
// leaves when the last one closes. A session naming none still serves, and its
// tools answer with what is missing.
//
// A member with several sessions is reached through the one it was last seen
// in. It is seen in a session when the session opens, when a focus call
// arrives on it, and when the app server reports a turn starting on the thread
// the session serves. A harness server hosting several agents' threads, as a
// shared Codex app server does, is scoped to each member's threads, so a
// mention reaches the thread of the agent it names.
//
// A shared OpenCode server is the exception to a session naming its member: it
// opens one session for every TUI attached to it. Each TUI's plugin registers
// over the routes beside [HTTPPath] instead, and every tool call on an OpenCode
// session acts as the member that owns the OpenCode session the call names:
// the one whose TUI registered it first among the TUIs showing it now. See
// opencode.go.
type HTTPServer struct {
	host *host

	// mu guards the registry below, which every request of every session reads.
	mu sync.Mutex
	// closed is set once the server is shutting down. A session that arrives
	// after it acts as nobody: its member would outlive the server that made it.
	closed bool
	// members are the members attending, by the token they attend as.
	members map[string]*attendee
	// sessions are the sessions bound so far, and what each acts as.
	sessions map[*mcpsdk.ServerSession]*binding
	// seen counts the moments a member is seen somewhere, so that of two
	// moments the later one has the greater number.
	seen uint64
	// turns is the moment a turn last started on each thread the app server
	// reported one on.
	turns map[string]uint64
	// pings are the pings in flight, by nonce, and the thread each was sent to.
	pings map[string]string
	// running holds every member's loops and every session's watch, so a
	// shutdown waits for them before the connection to the daemon closes.
	running errgroup.Group
}

// attendee is one member and what keeps it in the room: the MCP sessions naming
// its token and the notice streams an OpenCode TUI holds open with it. It
// leaves once neither is left.
type attendee struct {
	member   *Member
	stop     context.CancelFunc
	sessions int
	// streams counts the notice streams open, which openCode writes to.
	streams  int
	openCode *harnessctl.OpenCodeNotices
	// openCodeSession is the OpenCode session the member's TUI last said it
	// shows, "" before it said one and once its last notice stream closed.
	openCodeSession string
	// openCodeShownAt is the moment the TUI registered openCodeSession, which
	// orders the TUIs showing one session: the earliest owns it.
	openCodeShownAt uint64
	// scoped is the harness the member is reached through on each harness
	// server hosting several agents, by that server's unscoped harness. It is
	// made once per server, so the member keeps one feed there however often
	// it moves between its sessions.
	scoped map[harnessctl.ThreadScoper]scopedHarness
}

// scopedHarness is a harness scoped to one member's threads, and those threads
// as the harness asks about them.
type scopedHarness struct {
	harness harnessctl.Harness
	threads *memberThreads
}

// binding is what one session acts as. A nil attendee is a session that named
// no identity.
type binding struct {
	attendee *attendee
	// handshook says the session's harness has been read off its handshake.
	handshook bool
	// harness is the harness the handshake named, nil before it landed.
	harness harnessctl.Harness
	// thread is the thread the session serves on a harness server hosting
	// several, once a tool call named it.
	thread string
	// seen is the moment the member was last seen in this session itself: when
	// the session opened, or when a focus call arrived on it.
	seen uint64
}

// NewHTTP dials sockPath and prepares a server for many MCP sessions. Nothing
// attends until a session names a token, so the daemon being down or refusing
// that token keeps no harness from getting a server it can talk to.
//
// getenv is the environment the harness server started this process in. It is
// where a deliverer finds what its channel needs, the same for every session;
// it is never where a session's identity comes from. A nil getenv reads the
// process environment, and a nil logger discards logs.
//
// It comes back with no tools and no resources: a family registers its own onto
// it before [HTTPServer.Serve], as onto a stdio [Server].
func NewHTTP(
	logger *slog.Logger,
	sockPath string,
	getenv func(string) string,
) (*HTTPServer, error) {
	return newHTTP(logger, sockPath, getenv, httpKeepAlive)
}

// newHTTP is [NewHTTP] pinging its sessions every keepAlive, so a test can
// watch a session whose client vanished being closed in milliseconds.
func newHTTP(
	logger *slog.Logger,
	sockPath string,
	getenv func(string) string,
	keepAlive time.Duration,
) (*HTTPServer, error) {
	h, err := newHost(logger, sockPath, getenv, keepAlive)
	if err != nil {
		return nil, err
	}
	s := &HTTPServer{
		host:     h,
		members:  map[string]*attendee{},
		sessions: map[*mcpsdk.ServerSession]*binding{},
		turns:    map[string]uint64{},
		pings:    map[string]string{},
	}
	// One detector for every session, so the sessions hosted by one app server
	// share one connection to it, and the turns it reports reach the members.
	h.detect = harnessctl.NewDetector(logger, s.turnStarted).Detect
	h.mcp.AddReceivingMiddleware(s.bindsSessions, answersReservedCalls(s), takesOpenCodeSession)
	return s, nil
}

// MCP is the SDK server a family adds its tools to. Every session is served by
// this one server, so a tool registered once is offered to all of them.
func (s *HTTPServer) MCP() *mcpsdk.Server {
	return s.host.mcp
}

// Client is the connection to the daemon every tool acts through, whichever
// member it acts as.
func (s *HTTPServer) Client() *cli.Client {
	return s.host.client
}

// AddResource registers a resource and lets a harness subscribe to it.
func (s *HTTPServer) AddResource(res *mcpsdk.Resource, handler mcpsdk.ResourceHandler) {
	s.host.addResource(res, handler)
}

// AnnounceOnRosterChange has the subscribed sessions told to read uri again
// whenever the room's attendance or anyone's state changes, as any member of
// this server sees it.
func (s *HTTPServer) AnnounceOnRosterChange(uri string) {
	s.host.announceOnRosterChange(uri)
}

// MemberOf is the member a tool called over session acts as: the one its token
// names. A session that named no token answers with what is missing.
//
// A call on an OpenCode session acts as the member that owns the OpenCode
// session the call named, which ctx carries, and never as the session's own
// token: one such session carries the calls of every TUI attached to the
// server.
func (s *HTTPServer) MemberOf(ctx context.Context, session *mcpsdk.ServerSession) (*Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.sessions[session]
	switch {
	case b == nil:
		return nil, errNotAttending
	case b.servesOpenCode():
		return s.openCodeCaller(ctx)
	case b.attendee == nil:
		return nil, errNoTokenHeader
	default:
		return b.attendee.member, nil
	}
}

// methodInitialize is the request a session opens with, which is where it names
// its identity and its client.
const methodInitialize = "initialize"

// bindsSessions ties every session to its member at its initialize request, and
// reads its harness off that handshake.
//
// Only initialize binds. A client may probe the server before it initializes —
// the SDK's own client asks server/discover first — and the handler gives that
// probe a session of its own that its client never closes. The server closes it
// once it has left [keepAliveFailures] pings unanswered, which the SDK logs as
// a failure. A probe binding its token would keep the member in the room until
// then, after every real session was gone.
//
// The binding comes before the handler, so a member exists by the time the
// session's first call needs one. The harness is read after it, because the
// initialize request is what hands the session its client's name.
func (s *HTTPServer) bindsSessions(next mcpsdk.MethodHandler) mcpsdk.MethodHandler {
	return func(
		ctx context.Context, method string, req mcpsdk.Request,
	) (mcpsdk.Result, error) {
		session, ok := req.GetSession().(*mcpsdk.ServerSession)
		if !ok || method != methodInitialize {
			return next(ctx, method, req)
		}
		s.bind(session, req.GetExtra())
		res, err := next(ctx, method, req)
		s.handshook(session)
		return res, err
	}
}

// bind ties session to the member the token in extra names, making that member
// when it is the token's first session, and watches the session for its end.
// A session already bound keeps what it was bound to.
func (s *HTTPServer) bind(session *mcpsdk.ServerSession, extra *mcpsdk.RequestExtra) {
	var token string
	if extra != nil {
		token = extra.Header.Get(TokenHeader)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	if _, ok := s.sessions[session]; ok {
		return
	}
	b := &binding{}
	if token != "" {
		a := s.attendeeOf(token)
		a.sessions++
		b.attendee = a
		// A session opening is the member being seen in it: a person started a
		// TUI, or a new thread in one.
		b.seen = s.nextSeen()
	}
	s.sessions[session] = b
	s.running.Go(func() error {
		// A session ends however its client left — a DELETE, pings left
		// unanswered, or the server closing it — and Wait returns on each.
		_ = session.Wait()
		s.release(session)
		return nil
	})
}

// attendeeOf is the member attending as token, made and started when it is the
// token's first. Its caller counts what keeps it in the room. It runs with mu
// held.
func (s *HTTPServer) attendeeOf(token string) *attendee {
	if a := s.members[token]; a != nil {
		return a
	}
	// The member runs on a context of its own rather than on any request's: it
	// lives as long as its sessions and streams do, which is longer than any one
	// request and shorter than the server.
	ctx, stop := context.WithCancel(context.Background())
	a := &attendee{
		member: s.host.newMember(token),
		stop:   stop,
		scoped: map[harnessctl.ThreadScoper]scopedHarness{},
	}
	s.members[token] = a
	s.running.Go(func() error {
		a.member.run(ctx)
		return nil
	})
	return a
}

// unkept is called with mu held whenever a loses a session or a stream. It
// reports whether nothing keeps a in the room any more, having taken it off the
// registry, and otherwise points it at what is left. A true answer leaves the
// caller to stop the member once mu is released.
func (s *HTTPServer) unkept(a *attendee) bool {
	if a.sessions > 0 || a.streams > 0 {
		// The member may have been last seen in what closed.
		s.retarget(a)
		return false
	}
	if s.members[a.member.token] == a {
		delete(s.members, a.member.token)
	}
	return true
}

// handshook reads the harness off session once it names a client, and points
// the member the session acts as at the session it was last seen in. Each
// session is read once: it names one client for its whole life.
func (s *HTTPServer) handshook(session *mcpsdk.ServerSession) {
	params := session.InitializeParams()
	if params == nil {
		return
	}
	s.mu.Lock()
	b := s.sessions[session]
	first := b != nil && !b.handshook
	if first {
		b.handshook = true
	}
	s.mu.Unlock()
	if !first {
		return
	}
	harness := s.host.harnessOf(params)
	s.mu.Lock()
	defer s.mu.Unlock()
	b.harness = harness
	switch {
	case b.attendee != nil:
		s.retarget(b.attendee)
	case !b.servesOpenCode():
		// An OpenCode session names no identity by design: its calls name one
		// each. Any other harness was meant to send the header.
		s.host.logger.Warn("an MCP session named no chat identity; its tools will report it",
			"session", session.ID(), "header", TokenHeader)
	}
}

// release unbinds a session that ended, and takes its member out of the room
// when it was the last thing keeping it there.
//
// A token whose next session arrives before the daemon has let go of the old
// attendance gets a new member that is refused as already attending; its loop
// retries until the daemon has caught up, which is what it does for any other
// refusal.
func (s *HTTPServer) release(session *mcpsdk.ServerSession) {
	s.mu.Lock()
	b := s.sessions[session]
	delete(s.sessions, session)
	var left *attendee
	if b != nil && b.attendee != nil {
		b.attendee.sessions--
		if s.unkept(b.attendee) {
			left = b.attendee
		}
	}
	if b != nil && b.thread != "" && !s.threadBound(b.thread) {
		delete(s.turns, b.thread)
	}
	s.mu.Unlock()
	if left != nil {
		left.stop()
		s.host.logger.Info("the last session of a chat member closed; leaving the room",
			"member", cli.Address(left.member.selfMember()))
	}
}

// Serve serves MCP at [HTTPPath] on ln until ctx is done, and closes the
// connection to the daemon on the way out, so an HTTPServer is not reusable.
// The routes an OpenCode TUI's plugin registers over are served beside it.
//
// On the way out every session is closed and every member leaves the room.
// Like [Server.Serve], it answers a shutdown ctx asked for with ctx's error.
func (s *HTTPServer) Serve(ctx context.Context, ln net.Listener) error {
	defer func() { _ = s.host.client.Close() }()

	mux := http.NewServeMux()
	mux.Handle(HTTPPath, mcpsdk.NewStreamableHTTPHandler(
		func(*http.Request) *mcpsdk.Server { return s.host.mcp },
		&mcpsdk.StreamableHTTPOptions{Logger: s.host.logger},
	))
	s.routeOpenCode(mux)
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: readHeaderTimeout}

	s.host.logger.Info("serving MCP over HTTP", "addr", ln.Addr().String(), "path", HTTPPath)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serving MCP over HTTP on %s: %w", ln.Addr(), err)
		}
		return nil
	})
	g.Go(func() error {
		<-gctx.Done()
		// Closed rather than shut down: a session's event stream is a request
		// that never finishes, and a graceful shutdown would wait on it forever.
		err := srv.Close()
		s.shutdown()
		return err
	})
	if err := g.Wait(); err != nil {
		return err
	}
	return ctx.Err()
}

// shutdown takes every member out of the room and closes every session, then
// waits for what they were running.
//
// The members go first. A tool call still waiting on its member's attendance
// is answered once that member stops, and closing a session waits for the
// calls it has in flight.
func (s *HTTPServer) shutdown() {
	s.mu.Lock()
	s.closed = true
	stops := make([]context.CancelFunc, 0, len(s.members))
	for _, a := range s.members {
		stops = append(stops, a.stop)
	}
	s.mu.Unlock()
	for _, stop := range stops {
		stop()
	}
	for session := range s.host.mcp.Sessions() {
		_ = session.Close()
	}
	_ = s.running.Wait()
}
