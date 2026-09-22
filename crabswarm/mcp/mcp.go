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
// Reading that stream is also how a mention reaches the agent. The server knows
// which harness it serves, because the harness names itself in the MCP
// handshake; where that harness has a channel of its own, the member attends as
// one the daemon never types at and the server pushes the notice through the
// channel instead, off the same feed everything else is read from.
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
	"os"
	"sync"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
	"github.com/ngicks/crabswarm/internal/libver"
	"github.com/ngicks/crabswarm/pkg/harnessctl"
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

	// getenv reads the environment the harness started this server in, which is
	// where a deliverer finds what its channel needs. It is a field rather than
	// a call to [os.Getenv] so a test can play a harness without setting a
	// variable on the process running it. [New] takes the real one.
	getenv func(string) string

	// detect names the harness behind the client that handshook. It is a field
	// for the reason getenv is one: a harness that carries a channel or a state
	// feed of its own speaks to a CLI this process cannot start, so a test hands
	// the server one that plays the part. [New] takes [harnessctl.Detect].
	detect func(clientName string, getenv func(string) string) harnessctl.Harness

	// initialized is closed once the harness has finished the MCP handshake and
	// [Server.harness] names what it runs. Attendance waits on it: what the
	// member declares about itself is read off that handshake, and attending
	// before it landed would put the wrong answer in the room for the rest of
	// the session.
	initialized chan struct{}
	initOnce    sync.Once
	// harness is the CLI this server serves and the channel a mention takes to
	// it. Written once, before initialized is closed, and read only after.
	harness harnessctl.Harness

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
	// self is the member the daemon attended as, which is what an event of the
	// room is matched against: a member is named by room, team and name.
	self *chatv1.Member
	// state is what this member's harness last reported about itself, as the
	// attendance said it and every report since has changed it. A mention is
	// only ever delivered while it is done.
	state chatv1.HarnessState
	// pending says a mention arrived that nothing delivered — the agent was
	// mid-turn, or the delivery failed. It is what the next done report and the
	// next attendance answer by counting the unread and saying how much waits.
	pending bool

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
	return newServer(logger, sockPath, token, os.Getenv)
}

// newServer is [New] with the environment handed in. What the server declares
// about a channel is read out of it before the session starts, so a test plays
// a launcher through this rather than setting a variable on the process running
// the suite — which would point a real channel at whatever is listening on the
// plugin's own port.
func newServer(
	logger *slog.Logger, sockPath, token string, getenv func(string) string,
) (*Server, error) {
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
		getenv:            getenv,
		detect:            harnessctl.Detect,
		subscribable:      map[string]struct{}{},
		initialized:       make(chan struct{}),
		settled:           make(chan struct{}),
		attendBackoffBase: attendBackoffBase,
		attendBackoffMax:  attendBackoffMax,
	}
	opts := &mcpsdk.ServerOptions{
		Logger:             logger,
		SubscribeHandler:   s.subscribed,
		UnsubscribeHandler: s.unsubscribed,
	}
	// What the server declares about itself is settled here, because the
	// handshake carries it and the handshake is what tells the server which
	// harness it is serving — the answer arrives after the question.
	if harnessctl.ClaudeChannelEnabled(s.getenv) {
		opts.Instructions = claudeChannelInstructions
	}
	s.mcp = mcpsdk.NewServer(
		&mcpsdk.Implementation{Name: serverName, Version: libver.Version},
		opts,
	)
	s.mcp.AddReceivingMiddleware(s.readsHandshake)
	return s, nil
}

// claudeChannelInstructions tells the model what the event a room notice reaches
// it as is, and which tool answers it.
//
// The harness shows the model an event another plugin's channel pushed, and that
// plugin's own instructions tell the model to answer with its reply tool — which
// reaches a browser tab nobody is watching, while the room goes on waiting. So
// the event is named as precisely as this end can name it: the source the plugin
// labels it with, the message id this server numbers its uploads by, and the
// opening of the line the room worded.
const claudeChannelInstructions = "Room notices from " + serverName +
	` reach you as <channel source="fakechat" chat_id="web" ` +
	`message_id="crabswarm-..."> events whose content opens with ` +
	"[crabswarm chat]. Each one says a chat message is waiting for you: " +
	"read it with the chat_read tool and answer with chat_send. " +
	"Never answer such an event with fakechat's reply tool."

// readsHandshake watches what the harness sends for the point at which it has
// said who it is.
//
// The middleware runs after the handler rather than on one named method,
// because which frame carries the client's name depends on the protocol the two
// sides settled on: the handshake a harness opens with today names it in the
// initialize request, and the newer one that replaces that handshake carries the
// same name in the metadata of whatever the client asks first. Reading it off
// the session afterwards covers both, where waiting for the initialized
// notification would leave a member on the newer protocol attending nothing at
// all — it never sends one.
func (s *Server) readsHandshake(next mcpsdk.MethodHandler) mcpsdk.MethodHandler {
	return func(
		ctx context.Context, method string, req mcpsdk.Request,
	) (mcpsdk.Result, error) {
		res, err := next(ctx, method, req)
		if session, ok := req.GetSession().(*mcpsdk.ServerSession); ok {
			s.handshook(session)
		}
		return res, err
	}
}

// handshook reads the harness off session once it names a client, which is
// where the member learns what it runs and how a mention can reach it. It is
// the first naming that counts: a session names one client for its whole life,
// and a member cannot change what it attends as without attending again.
func (s *Server) handshook(session *mcpsdk.ServerSession) {
	params := session.InitializeParams()
	if params == nil {
		return
	}
	s.initOnce.Do(func() {
		var client string
		if params.ClientInfo != nil {
			client = params.ClientInfo.Name
		}
		s.harness = s.detect(client, s.getenv)
		close(s.initialized)
		s.logger.Info("the harness named itself",
			"client", client,
			"harness", cli.HarnessName(s.harness.Kind()),
			"nudge", cli.NudgeDeliveryName(s.harness.Nudge()))
	})
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
		// The feed is followed under the same identity the attendance uses: a
		// state is reported about the member this server attends as, so a
		// server with none to resolve has nobody to report about and would hold
		// a feed open for nothing.
		g.Go(func() error {
			s.watchHarnessState(gctx)
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
	// Nothing is declared before the harness has said what it is: the kind and
	// the delivery an attendance carries stand for its whole life, and the
	// daemon stops typing at a member that claimed to deliver its own mentions.
	select {
	case <-s.initialized:
	case <-ctx.Done():
		return
	}
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
		// An attendance that never opened — the daemon refused it, or the
		// harness's channel was not there to probe — is a different report from
		// one that ended, and the operator reading the log should not be told a
		// stream ended when none was ever up.
		msg := "the chat attendance ended; attending again"
		if !landed {
			msg = "the chat attendance could not open; trying again"
		}
		s.warnRetry(msg, failures, backoff, err)
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
	// A harness whose channel is somewhere other than this session is asked
	// whether it is there before the member attends, and the member stays out of
	// the room for as long as the answer is no.
	//
	// Attending as a terminal member instead — leaving the daemon to type at a
	// session whose channel never came up — would paper the broken launch over:
	// the room would go on working, and nobody would learn that the plugin the
	// launch promised is not running. A member missing from the roster is noticed,
	// and the refusal it is missing over says which channel was looked for and
	// where.
	//
	// Asked on every attempt rather than once: a plugin the launcher started
	// beside this process may well come up after it, and then this is the loop
	// that picks it up.
	if prober, ok := s.harness.(harnessctl.Prober); ok {
		if err := prober.Probe(ctx); err != nil {
			s.attendanceEnded(err)
			return false, err
		}
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
	// The harness and the delivery are what the handshake said: a harness whose
	// channel this server can push a mention through attends as native, and the
	// daemon then types nothing at it, leaving the waking to the deliverer
	// below.
	attendance, err := s.client.Attend(actx, token, "",
		chatv1.MemberKind_MEMBER_KIND_AGENT,
		s.harness.Kind(),
		s.harness.Nudge())
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
	// Whatever was said while nothing was attending was said to nobody here, so
	// every attendance asks what is waiting rather than only the ones that
	// followed a dropped stream: a server that started after its agent was
	// mentioned has the same gap to close.
	s.deliverWaiting(ctx)
	// The first event of the feed is this member's own arrival, which the
	// daemon publishes to the room it just joined. It is announced like any
	// other roster change: the room did gain a member, and every attendee sees
	// the same feed.
	err = attendance.Forward(actx, func(ev *chatv1.RoomEvent) error {
		if rosterChanged(ev) {
			s.announceRoster(ctx)
		}
		s.observe(ctx, ev)
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

// watchHarnessState reports what this member's harness says it is doing for as
// long as the session runs, for a harness that says it without being asked.
//
// It is the whole of such a member's state: its CLI carries a feed of its own,
// its hooks report nothing beside it, and a room hearing neither would never
// learn when the agent's turn is over — which is the moment a mention waiting
// for it may be delivered. A harness with no feed leaves the reporting to its
// hooks, and there is nothing to run here.
func (s *Server) watchHarnessState(ctx context.Context) {
	// Nothing before the handshake, for the reason attendance waits on it: the
	// harness is what the handshake names, and there is nothing to ask for a
	// feed until it has.
	select {
	case <-s.initialized:
	case <-ctx.Done():
		return
	}
	source, ok := s.harness.(harnessctl.StateSource)
	if !ok {
		return
	}
	source.Watch(ctx, func(state chatv1.HarnessState) {
		s.reportHarnessState(ctx, state)
	})
}

// reportHarnessState hands the daemon one state the feed carried, which the
// daemon publishes to the room — this server included, where it is what decides
// whether a mention may be delivered now.
//
// A report that failed is said once and left. The next state the feed carries
// replaces it whatever became of this one, and the member meanwhile keeps
// whatever it last reported, which is the answer that interrupts nobody
// mid-turn. A session on its way out says nothing at all: what the feed had
// queued is drained as it closes, against a daemon connection closing with it.
func (s *Server) reportHarnessState(ctx context.Context, state chatv1.HarnessState) {
	token, err := s.ResolveToken()
	if err == nil {
		err = s.client.ReportHarnessState(ctx, token, state)
	}
	if err != nil && ctx.Err() == nil {
		s.logger.Warn("reporting what the harness says it is doing failed",
			"state", cli.HarnessStateName(state), "error", err)
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
//
// The member comes back with the state the daemon holds for it, which is what
// decides whether a mention may be delivered: a fresh attendance is done, and
// one taken up again carries whatever the agent's hooks last reported.
func (s *Server) attendanceLanded(self *chatv1.Member) {
	s.mu.Lock()
	s.attended = true
	s.attendErr = nil
	s.self = self
	s.state = self.GetState()
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
