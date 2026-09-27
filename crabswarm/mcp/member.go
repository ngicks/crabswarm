package mcp

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// Member is one chat identity this process attends the room as: the attendance
// it holds, the harness a mention reaches it through, and the state that
// harness last reported.
//
// A stdio server holds one for its whole session. An HTTP server holds one per
// identity token, for as long as that token has a session open, so several
// agents behind one harness server are each a member of their own.
type Member struct {
	logger *slog.Logger
	client *cli.Client
	// token is the identity this member attends as. It is resolved before the
	// member exists: a session with no identity has no member at all, and its
	// tools report what is missing instead.
	token string
	// announce tells the subscribed sessions to read the roster again. It is the
	// server's, since the sessions and the resources are.
	announce func(ctx context.Context)
	pace     pace

	// initialized is closed once a session of this member has finished the MCP
	// handshake and [Member.harness] names what it runs. Attendance waits on it:
	// what the member declares about itself is read off that handshake, and
	// attending before it landed would put the wrong answer in the room for the
	// rest of the attendance.
	initialized chan struct{}
	initOnce    sync.Once

	// mu guards what the attendance loop reports about itself, which is what
	// every tool call reads before acting, and the harness a later session may
	// replace.
	mu sync.Mutex
	// harness is the CLI the member's most recently handshaken session runs,
	// and the channel a mention takes to it. An attendance declares the one it
	// finds when it opens and keeps that declaration until it opens again.
	harness harnessctl.Harness
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

	// reportMu serializes what this member tells the daemon about its harness.
	// It is held across the send, so a repeat that read the last state cannot
	// land after a newer one the feed sent meanwhile and leave the daemon a
	// state behind. It is not mu: mu is what every tool call reads before
	// acting, and holding that across an RPC would stall them.
	reportMu sync.Mutex
	// reportedState is what a feed last handed over, which is what gets said
	// again. Unspecified means no feed has spoken — the state the attendance
	// came back with is not one, since it is what the daemon already holds.
	reportedState chatv1.HarnessState
}

// pace is the schedule a member's loops run at. It is held by the server and
// copied into every member it makes, so a test can drive the loops at a pace it
// can wait for. [New] and [NewHTTP] take the constants.
type pace struct {
	attendBackoffBase  time.Duration
	attendBackoffMax   time.Duration
	harnessStateResend time.Duration
	// reprobeInterval is how often a held attendance asks its harness's channel
	// whether it is still there.
	reprobeInterval time.Duration
}

// defaultPace is the schedule a server nobody tuned runs its members at.
var defaultPace = pace{
	attendBackoffBase:  attendBackoffBase,
	attendBackoffMax:   attendBackoffMax,
	harnessStateResend: harnessStateResend,
	reprobeInterval:    reprobeInterval,
}

func newMember(
	logger *slog.Logger,
	client *cli.Client,
	token string,
	announce func(ctx context.Context),
	p pace,
) *Member {
	return &Member{
		logger:      logger,
		client:      client,
		token:       token,
		announce:    announce,
		pace:        p,
		initialized: make(chan struct{}),
		settled:     make(chan struct{}),
	}
}

// Token is the identity token this member attends as, which is what every call
// a tool makes on its behalf carries.
func (m *Member) Token() string {
	return m.token
}

// run holds the attendance and follows the harness until ctx is done. The
// server that made the member ends ctx once no session is left to attend for.
func (m *Member) run(ctx context.Context) {
	var g errgroup.Group
	g.Go(func() error {
		m.attend(ctx)
		return nil
	})
	// The feed is followed under the same identity the attendance uses: a state
	// is reported about the member this is, so a session with no identity has
	// no member and holds no feed open for nothing.
	g.Go(func() error {
		m.watchHarnessState(ctx)
		return nil
	})
	g.Go(func() error {
		m.resendHarnessState(ctx)
		return nil
	})
	_ = g.Wait()
}

// handshook records the harness a session of this member runs, which is how a
// mention reaches the member from here on. The first one is also what lets the
// attendance open.
//
// A later session replaces the harness rather than being ignored: several
// sessions share one token when a harness server hosts one agent in several
// windows, and the one handshaken last is the one a mention is handed to.
func (m *Member) handshook(harness harnessctl.Harness) {
	m.mu.Lock()
	m.harness = harness
	m.mu.Unlock()
	m.initOnce.Do(func() { close(m.initialized) })
}

// currentHarness is the harness of the most recently handshaken session, or nil
// before any session has handshaken.
func (m *Member) currentHarness() harnessctl.Harness {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.harness
}

// How long a member waits before opening the attendance stream again. The
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
func (m *Member) warnRetry(msg string, failures int, backoff time.Duration, err error) {
	if failures > 1 && failures%warnEvery != 0 {
		return
	}
	m.logger.Warn(msg, "failures", failures, "backoff", backoff, "error", err)
}

// attendTimeout bounds the opening of one attendance. The stream is lazy, so a
// daemon that accepted the connection and then answered nothing would leave the
// loop waiting for the rest of the session and every tool call waiting with it.
// A local socket answers in microseconds; this is only the point past which no
// answer is coming.
const attendTimeout = 10 * time.Second

// reprobeInterval is how often a held attendance asks its harness's channel
// whether it is still there.
//
// The probe gates the opening, and on its own it gates nothing after that: a
// plugin that dies mid-session would leave a member listed as one the daemon
// types nothing at, with every mention it is handed dropped. Asking again is a
// loopback request the plugin answers out of memory, so what this number buys
// is only how long a channel that went away may go unnoticed — a few seconds of
// one late mention, where a probe every second would cost the same and answer
// no sooner than the next mention needs it.
const reprobeInterval = 5 * time.Second

// errNotAttending is what a tool call is refused with when the attendance is
// down for a reason nothing recorded. Every path that ends an attendance
// records why, so this stands in for nothing that happens today; it exists so a
// refusal is never the empty error.
var errNotAttending = errors.New("not attending the chat room")

// attend keeps this member attending for as long as it runs: one open stream,
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
func (m *Member) attend(ctx context.Context) {
	// Nothing is declared before the harness has said what it is: the kind and
	// the delivery an attendance carries stand for its whole life, and the
	// daemon stops typing at a member that claimed to deliver its own mentions.
	select {
	case <-m.initialized:
	case <-ctx.Done():
		return
	}
	backoff := m.pace.attendBackoffBase
	// failures counts the consecutive attempts that did not open, and nothing
	// else: it is what decides which of them is worth a line, and an attendance
	// that stood in between means whatever kept the previous opening from
	// working is over.
	failures := 0
	for reopened := false; ; reopened = true {
		landed, err := m.holdAttendance(ctx, reopened)
		if ctx.Err() != nil {
			return
		}
		// An attendance that stood for a while and then ended is not the
		// trouble one that never opened is, so it starts its retries over
		// rather than inheriting the wait the previous failure had climbed to —
		// and it is reported every time rather than thinned out the way a run
		// of failures is, since it resets the run and so is never one of them.
		//
		// The operator reading the log should not be told a stream ended when
		// none was ever up either, which is the other half of why the two are
		// reported apart: a refused opening — the daemon turned it down, or the
		// harness's channel was not there to probe — says what to go and fix,
		// and the first of a run is the one that says it.
		if landed {
			backoff = m.pace.attendBackoffBase
			failures = 0
			m.logger.Warn("the chat attendance ended; attending again",
				"backoff", backoff, "error", err)
		} else {
			failures++
			m.warnRetry("the chat attendance could not open; trying again",
				failures, backoff, err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(2*backoff, m.pace.attendBackoffMax)
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
// the member.
func (m *Member) holdAttendance(ctx context.Context, reopened bool) (bool, error) {
	// One harness for the whole attempt: what is probed is what is declared, and
	// a session handshaking meanwhile is picked up by the next opening.
	harness := m.currentHarness()
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
	// that picks it up. It is asked again for as long as the stream is held, for
	// the same reason in the other direction — see [Member.reprobe].
	prober, probed := harness.(harnessctl.Prober)
	if probed {
		if err := prober.Probe(ctx); err != nil {
			m.attendanceEnded(err)
			return false, err
		}
	}
	actx, cancel := context.WithCancel(ctx)
	defer cancel()
	// A held attendance has two halves: the room's feed, which this member acts
	// on, and the question of whether the channel it acts through is still
	// there. Either one ending ends the attendance — so a plugin that died takes
	// its member out of the room the same way a daemon that went away does, and
	// the loop above backs off, keeps asking, and attends again once the plugin
	// answers.
	//
	// The group is made before the stream is opened, because the stream is then
	// opened on the group's own context: the half that failed is the answer the
	// attendance ends with, and the cancel the group makes on its way out is what
	// ends the other half. Opening the stream on the context above instead would
	// have the two race to report, and the feed's own cancellation could stand in
	// front of what actually went wrong.
	g, gctx := errgroup.WithContext(actx)
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
	attendance, err := m.client.Attend(gctx, m.token, "",
		chatv1.MemberKind_MEMBER_KIND_AGENT,
		harness.Kind(),
		harness.Nudge())
	if !opening.Stop() && err != nil {
		err = errors.New("the daemon did not answer the attendance within " +
			attendTimeout.String())
	}
	if err != nil {
		m.attendanceEnded(err)
		return false, err
	}
	m.attendanceLanded(attendance.Self())
	if reopened {
		m.announce(ctx)
	}
	// Whatever was said while nothing was attending was said to nobody here, so
	// every attendance asks what is waiting rather than only the ones that
	// followed a dropped stream: a server that started after its agent was
	// mentioned has the same gap to close.
	m.deliverWaiting(ctx)
	// The first event of the feed is this member's own arrival, which the
	// daemon publishes to the room it just joined. It is announced like any
	// other roster change: the room did gain a member, and every attendee sees
	// the same feed.
	g.Go(func() error {
		return attendance.Forward(gctx, func(ev *chatv1.RoomEvent) error {
			if rosterChanged(ev) {
				m.announce(ctx)
			}
			m.observe(ctx, ev)
			return nil
		})
	})
	if probed {
		g.Go(func() error { return m.reprobe(gctx, prober) })
	}
	err = g.Wait()
	m.attendanceEnded(err)
	return true, err
}

// reprobe asks the channel whether it is still there for as long as the
// attendance is held, and answers with the first refusal.
//
// A failed delivery does not ask on the spot. It says the notice is still
// unread, which the outstanding count already carries to the next report that
// ends a turn, and it says nothing about whether the plugin is coming back —
// while a plugin that is genuinely gone is caught here within the interval
// anyway. Wiring one into the other would buy a few seconds at the price of a
// path that only ever runs when something is already broken.
func (m *Member) reprobe(ctx context.Context, prober harnessctl.Prober) error {
	ticker := time.NewTicker(m.pace.reprobeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// Nothing to report: the attendance is ending for whatever the other
			// half answered with, and a context this half only read would stand
			// in front of it as the reason.
			return nil
		case <-ticker.C:
			if err := prober.Probe(ctx); err != nil {
				return err
			}
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
// long as the member runs, for a harness that says it without being asked.
//
// It is the whole of such a member's state: its CLI carries a feed of its own,
// its hooks report nothing beside it, and a room hearing neither would never
// learn when the agent's turn is over — which is the moment a mention waiting
// for it may be delivered. A harness with no feed leaves the reporting to its
// hooks, and there is nothing to run here.
//
// The harness watched is the first session's. A session handshaking later takes
// over delivery, and the feed stays where it is: the feed is the harness server
// itself, which every session of one token shares.
func (m *Member) watchHarnessState(ctx context.Context) {
	// Nothing before the handshake, for the reason attendance waits on it: the
	// harness is what the handshake names, and there is nothing to ask for a
	// feed until it has.
	select {
	case <-m.initialized:
	case <-ctx.Done():
		return
	}
	source, ok := m.currentHarness().(harnessctl.StateSource)
	if !ok {
		return
	}
	source.Watch(ctx, func(state chatv1.HarnessState) {
		m.reportHarnessState(ctx, state)
	})
}

// reportHarnessState hands the daemon one state the feed carried, which the
// daemon publishes to the room — this member included, where it is what decides
// whether a mention may be delivered now.
//
// The state is remembered as it is sent, and [Member.resendHarnessState] says
// what was remembered again; a report that failed is therefore not left. The
// member meanwhile keeps whatever the daemon last recorded, which is the answer
// that interrupts nobody mid-turn. A member on its way out says nothing at all:
// what the feed had queued is drained as it closes, against a daemon connection
// closing with it.
func (m *Member) reportHarnessState(ctx context.Context, state chatv1.HarnessState) {
	m.reportMu.Lock()
	defer m.reportMu.Unlock()
	m.reportedState = state
	m.sendHarnessState(ctx, state)
}

// harnessStateResend is how often the bridge re-sends the last state a feed
// reported, so a daemon that restarted or refused a report hears it again.
const harnessStateResend = 10 * time.Second

// resendHarnessState says again, every harnessStateResend, what a feed last
// reported about this member, for as long as the member runs.
//
// A feed speaks on change alone, and a member the daemon has wrong therefore
// stays wrong until the agent's next transition — which may be the end of a
// turn nobody was told had begun. A restarted daemon records a reopened
// attendance as done, a refused report changes nothing, and a change-only
// watcher repeats neither. Saying it again bounds the gap to one interval.
//
// Once an interval rather than on every poll of a feed, because the daemon runs
// `cmdman status set` as a child process for every report it records. What the
// room hears costs nothing: the daemon publishes an event only where the state
// moved. Which feed said it does not matter — the Codex app server and the
// Claude Code listing are repeated alike — and nothing is repeated at all until
// one of them has spoken.
func (m *Member) resendHarnessState(ctx context.Context) {
	ticker := time.NewTicker(m.pace.harnessStateResend)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.resendLastHarnessState(ctx)
		}
	}
}

// resendLastHarnessState says the last reported state again, where there is one
// to say and somebody to say it to.
//
// Nothing is sent while the attendance is down: the daemon refuses a state
// reported about a member the room does not have, so the attempt would fail and
// say so every interval for as long as the daemon stayed away. The next tick
// catches the attendance that comes back, which is the case this exists for.
func (m *Member) resendLastHarnessState(ctx context.Context) {
	if !m.attending() {
		return
	}
	m.reportMu.Lock()
	defer m.reportMu.Unlock()
	if m.reportedState == chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED {
		return
	}
	m.sendHarnessState(ctx, m.reportedState)
}

// sendHarnessState puts one state on the wire and reports a refusal. The caller
// holds reportMu.
func (m *Member) sendHarnessState(ctx context.Context, state chatv1.HarnessState) {
	err := m.client.ReportHarnessState(ctx, m.token, state)
	if err != nil && ctx.Err() == nil {
		m.logger.Warn("reporting what the harness says it is doing failed",
			"state", cli.HarnessStateName(state), "error", err)
	}
}

// attending reports whether the attendance stream is open right now.
func (m *Member) attending() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.attended
}

// attendanceLanded records an open attendance, which is what lets a tool act on
// behalf of this member.
//
// The member comes back with the state the daemon holds for it, which is what
// decides whether a mention may be delivered: a fresh attendance is done, and
// one taken up again carries whatever the agent's hooks last reported.
func (m *Member) attendanceLanded(self *chatv1.Member) {
	m.mu.Lock()
	m.attended = true
	m.attendErr = nil
	m.self = self
	m.state = self.GetState()
	m.mu.Unlock()
	m.settle()
	m.logger.Info("attending the chat room",
		"member", cli.Address(self), "room", self.GetRoom())
}

// attendanceEnded records that there is no attendance and why, which is what a
// tool call is refused with until the loop opens the next one.
func (m *Member) attendanceEnded(err error) {
	m.mu.Lock()
	m.attended = false
	m.attendErr = err
	m.mu.Unlock()
	m.settle()
}

// settle releases the tool calls waiting for the first attempt to finish. Every
// attempt after that finds the gate already open and answers from what the last
// one recorded.
func (m *Member) settle() {
	m.settleOnce.Do(func() { close(m.settled) })
}

// AwaitAttendance blocks until the member has an answer about its attendance
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
func (m *Member) AwaitAttendance(ctx context.Context) error {
	select {
	case <-m.settled:
	case <-ctx.Done():
		return ctx.Err()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	switch {
	case m.attended:
		return nil
	case m.attendErr != nil:
		return m.attendErr
	default:
		return errNotAttending
	}
}
