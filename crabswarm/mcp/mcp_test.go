package mcp

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// testToken is spelled out at every call site rather than left to resolution:
// this test binary may itself run under cmdman, and an empty token would
// quietly resolve to $CMDMAN_CMD_ID and talk as whoever is running the suite.
const testToken = "tok-a"

const testRoom = "/work/proj"

// eventTimeout bounds how long a test waits on something the server does off
// its own goroutines — attending again, ending an attendance. Generous enough
// to ride out a loaded machine and the retry backoff, short enough that a
// server that never does it fails instead of hanging the suite.
const eventTimeout = 5 * time.Second

// newTestBridge builds a server onto the stub. It carries no tool family: what
// this package does on its own is attend, and every case here is about that.
//
// The environment it was launched in is empty, so nothing a case does turns on
// what the process running the suite carries. A launcher variable in that
// environment would otherwise have a case build a real channel and go looking
// for it on the plugin's own loopback port, where a session belonging to
// whoever runs the suite may well be listening.
func newTestBridge(t *testing.T, svc *fakeChatService) *Server {
	t.Helper()

	return newTestBridgeIn(t, svc, func(string) string { return "" })
}

// newTestBridgeIn is [newTestBridge] launched in the environment getenv answers
// for, which is how a case plays a launcher that wired something up beside the
// server.
func newTestBridgeIn(
	t *testing.T, svc *fakeChatService, getenv func(string) string,
) *Server {
	t.Helper()

	bridge, err := newServer(
		slog.New(slog.DiscardHandler), serveTestDaemon(t, svc), testToken, getenv)
	assert.NilError(t, err)
	return bridge
}

// launchedWithTheChannel is the environment of a server whose launcher
// registered the fakechat plugin as a channel of the session beside it.
//
// It carries the claim and no port, and a case using it plays its harness as
// well, because nothing here may build the real channel: the real one is probed
// on the plugin's own loopback port, which on a developer's machine is a session
// of their own.
func launchedWithTheChannel(name string) string {
	if name == harnessctl.ClaudeChannelEnv {
		return "1"
	}
	return ""
}

// runsAt sets the loop going at a pace a case can wait for, for the cases that
// are about it attending again rather than about what one attendance did.
func runsAt(bridge *Server, base, max time.Duration) {
	bridge.attendBackoffBase = base
	bridge.attendBackoffMax = max
}

// serveBridge runs bridge over an in-memory pipe and returns the session a
// harness would hold. The client names itself something no harness is called,
// which is what a server this suite starts is: the cases about what a name
// decides spell one with [serveBridgeAs].
func serveBridge(t *testing.T, bridge *Server) *mcpsdk.ClientSession {
	t.Helper()
	return serveBridgeAs(t, bridge, "test-harness")
}

// serveBridgeAs is [serveBridge] under the name a harness gives itself in the
// handshake, which is all the server has to go on when it decides what it is
// serving.
func serveBridgeAs(t *testing.T, bridge *Server, clientName string) *mcpsdk.ClientSession {
	t.Helper()

	serverSide, clientSide := mcpsdk.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(t.Context())
	var served errgroup.Group
	served.Go(func() error { return bridge.Serve(ctx, serverSide) })

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: clientName, Version: "v0"}, nil)
	session, err := client.Connect(t.Context(), clientSide, nil)
	assert.NilError(t, err)

	// Waiting for Serve to return keeps the server's own goroutines from
	// outliving the test that owns the stub they are calling.
	t.Cleanup(func() {
		_ = session.Close()
		cancel()
		_ = served.Wait()
	})
	return session
}

// waitFor blocks until want reports true, failing with why when it never does.
// The wait is what a case spends on something the server does off its own
// goroutines; the poll is short because everything here is local.
func waitFor(t *testing.T, why string, want func() bool) {
	t.Helper()

	deadline := time.Now().Add(eventTimeout)
	for time.Now().Before(deadline) {
		if want() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("%s within %s", why, eventTimeout)
}

// dropFeed ends the attendance the server is holding, the way the daemon does
// when it goes away or drops a reader that fell behind. It returns once the
// stub has taken the error, so the attendance the next assertion is about is
// the one that comes after this.
func dropFeed(t *testing.T, svc *fakeChatService, err error) {
	t.Helper()

	select {
	case svc.drops <- err:
	case <-time.After(eventTimeout):
		t.Fatal("nothing is attending the room")
	}
}

// A server whose daemon refuses it keeps asking, past any handful of attempts.
// A harness starts its MCP subprocesses before the services they talk to, so
// the daemon regularly arrives late; a server that stopped asking would leave
// the room a member short until the agent happened to call a tool.
func TestServer_KeepsAttendingUntilTheDaemonAdmitsIt(t *testing.T) {
	fake := &fakeChatService{
		self: member("backend", "alice", testRoom),
		err:  status.Error(codes.Unauthenticated, "unknown identity token"),
	}
	bridge := newTestBridge(t, fake)
	runsAt(bridge, 10*time.Millisecond, 20*time.Millisecond)
	serveBridge(t, bridge)

	// More attempts than a bounded retry would have made.
	waitFor(t, "the server stopped asking to attend", func() bool {
		return fake.openCount() > 5
	})

	fake.setErr(nil)
	waitFor(t, "the server never attended once the daemon admitted it", func() bool {
		return fake.attendCount() == 1
	})
}

// The attendance is the stream, so a daemon that closes it has taken the
// membership with it — a restart, a reaped command, a watcher dropped for
// falling behind. The server opens it again with nothing asking it to, which is
// what the agent's hooks need: they report harness state through the CLI and
// are refused for as long as the room is missing the member.
func TestServer_AttendsAgainAfterTheDaemonEndsTheStream(t *testing.T) {
	fake := &fakeChatService{
		self:   member("backend", "alice", testRoom),
		events: make(chan *chatv1.RoomEvent),
		drops:  make(chan error),
	}
	bridge := newTestBridge(t, fake)
	runsAt(bridge, 10*time.Millisecond, 20*time.Millisecond)
	serveBridge(t, bridge)

	waitFor(t, "the server never attended", func() bool {
		return fake.attendCount() == 1
	})

	dropFeed(t, fake, status.Error(codes.Unavailable, "the daemon is going away"))

	waitFor(t, "the server never attended again", func() bool {
		return fake.attendCount() == 2
	})
}

// watched is a resource registered the way a tool family registers one, so the
// gate below has something it may announce.
const watched = "crabswarm://test/watched"

// unwatchable is a URI nothing registered, spelled as one a harness might
// plausibly have reached for.
const unwatchable = "crabswarm://test/missing"

// registerWatched adds the resource a family would, which is what makes the
// server willing to announce it.
func registerWatched(bridge *Server) {
	bridge.AddResource(
		&mcpsdk.Resource{Name: "watched", URI: watched, MIMEType: "application/json"},
		func(context.Context, *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
			return &mcpsdk.ReadResourceResult{}, nil
		},
	)
}

// assertNotFound asserts uri was refused as the missing resource the SDK
// spells. Pinned as that error rather than as any error at all: it is what
// tells a harness the URI is not one this server has, so it stops waiting on
// news that could never come.
func assertNotFound(t *testing.T, uri string, err error) {
	t.Helper()

	assert.Assert(t, errors.Is(err, mcpsdk.ResourceNotFoundError(uri)),
		"%s was not refused as a missing resource: %v", uri, err)
}

// Only a registered resource may be subscribed to. A URI no family registered
// is refused because the SDK would otherwise record a subscription for whatever
// it was handed and leave the harness waiting.
//
// The handler is exercised directly because the case is about the gate rather
// than about how a refusal travels back to whoever asked.
func TestServer_RefusesToWatchWhatItCannotAnnounce(t *testing.T) {
	bridge, err := New(slog.New(slog.DiscardHandler),
		serveTestDaemon(t, &fakeChatService{}), testToken)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = bridge.client.Close() })
	registerWatched(bridge)

	assertNotFound(t, unwatchable, bridge.subscribed(t.Context(), &mcpsdk.SubscribeRequest{
		Params: &mcpsdk.SubscribeParams{URI: unwatchable},
	}))
	assert.NilError(t, bridge.subscribed(t.Context(), &mcpsdk.SubscribeRequest{
		Params: &mcpsdk.SubscribeParams{URI: watched},
	}))
}

// Withdrawing is answered by the same gate as asking: a harness that was
// refused a subscription has none to withdraw, so telling it the withdrawal
// succeeded would say it had had one.
//
// Exercised directly for the reason the subscribe side is.
func TestServer_RefusesToUnwatchWhatItCannotAnnounce(t *testing.T) {
	bridge, err := New(slog.New(slog.DiscardHandler),
		serveTestDaemon(t, &fakeChatService{}), testToken)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = bridge.client.Close() })
	registerWatched(bridge)

	unsubscribe := func(uri string) error {
		return bridge.unsubscribed(t.Context(), &mcpsdk.UnsubscribeRequest{
			Params: &mcpsdk.UnsubscribeParams{URI: uri},
		})
	}

	assert.NilError(t, unsubscribe(watched))
	assertNotFound(t, unwatchable, unsubscribe(unwatchable))
}

// deliversTo has the server find the harness sink in its environment, which is
// what makes it a member that delivers its own mentions, and returns the file
// every notice lands in.
func deliversTo(t *testing.T, bridge *Server) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "notices.log")
	bridge.getenv = func(name string) string {
		if name == harnessctl.SinkEnv {
			return path
		}
		return ""
	}
	return path
}

// notices is what the harness was handed, oldest first. A file that is not
// there is one nothing was delivered to.
func notices(t *testing.T, path string) []string {
	t.Helper()

	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	assert.NilError(t, err)
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

// waitNotices blocks until the harness has been handed exactly want, so a case
// pins both what was delivered and that nothing else was.
func waitNotices(t *testing.T, path string, want ...string) {
	t.Helper()

	var got []string
	deadline := time.Now().Add(eventTimeout)
	for time.Now().Before(deadline) {
		got = notices(t, path)
		if slices.Equal(got, want) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("the harness was handed\n%q\nwant\n%q", got, want)
}

// doneSelf is the member a fresh attendance names: an agent that finished its
// last turn, which is the state a mention may be delivered in.
func doneSelf(team, name, room string) *chatv1.Member {
	m := member(team, name, room)
	m.Kind = chatv1.MemberKind_MEMBER_KIND_AGENT
	m.State = chatv1.HarnessState_HARNESS_STATE_DONE
	return m
}

// mentionOf is a message addressed to one member, as the daemon publishes it to
// the whole room once it has resolved the target.
func mentionOf(from, to *chatv1.Member, text string) *chatv1.RoomEvent {
	return &chatv1.RoomEvent{
		Event: &chatv1.RoomEvent_MessageAppended{
			MessageAppended: &chatv1.MessageAppended{
				Message: &chatv1.Message{
					From: from,
					Target: &chatv1.Target{
						Target: &chatv1.Target_Roles{Roles: &chatv1.Roles{
							Roles: []*chatv1.MemberTarget{
								{Team: to.GetTeam(), Name: to.GetName()},
							},
						}},
					},
					Text: text,
				},
			},
		},
	}
}

// stateOf is a member reporting what its harness is doing now.
func stateOf(m *chatv1.Member, state chatv1.HarnessState) *chatv1.RoomEvent {
	reported := &chatv1.Member{
		Team: m.GetTeam(), Name: m.GetName(), Room: m.GetRoom(), State: state,
	}
	return &chatv1.RoomEvent{
		Event: &chatv1.RoomEvent_MemberStateChanged{
			MemberStateChanged: &chatv1.MemberStateChanged{
				Member: reported, State: state,
			},
		},
	}
}

// pushEvent hands one event to the stub's feed. The channel is unbuffered, so
// this returns once the stub has taken it — and fails rather than hanging when
// nothing is attending.
func pushEvent(t *testing.T, svc *fakeChatService, ev *chatv1.RoomEvent) {
	t.Helper()

	select {
	case svc.events <- ev:
	case <-time.After(eventTimeout):
		t.Fatal("nothing is attending the room")
	}
}

// A member whose harness carries a channel of its own attends as one the daemon
// never types at, and the mention it is no longer typed at for is handed to that
// channel instead — once, and while it is idle, which is the only state a
// harness may be interrupted in.
func TestServer_DeliversAMentionItsHarnessCanTake(t *testing.T) {
	self := doneSelf("backend", "alice", testRoom)
	fake := &fakeChatService{self: self, events: make(chan *chatv1.RoomEvent)}
	bridge := newTestBridge(t, fake)
	sink := deliversTo(t, bridge)
	serveBridgeAs(t, bridge, "claude-code")

	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })
	// What it declared is what stops the daemon typing at it as well.
	assert.Equal(t, fake.lastAttend().GetHarness(), chatv1.Harness_HARNESS_CLAUDE_CODE)
	assert.Equal(t, fake.lastAttend().GetNudge(),
		chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)

	arrival := "[crabswarm chat] new message from frontend/bob" +
		" — read it with the chat_read tool and respond with chat_send," +
		" both from crabswarm-mcp"
	pushEvent(t, fake, mentionOf(member("frontend", "bob", testRoom), self, "rebase please"))
	waitNotices(t, sink, arrival)

	// A message it wrote itself and one addressed to nobody are both left alone:
	// neither is unread for it, so a notice would buy it an empty read.
	pushEvent(t, fake, mentionOf(self, self, "a note to myself"))
	pushEvent(t, fake, &chatv1.RoomEvent{
		Event: &chatv1.RoomEvent_MessageAppended{
			MessageAppended: &chatv1.MessageAppended{
				Message: &chatv1.Message{
					From: member("frontend", "bob", testRoom), Text: "main is red",
				},
			},
		},
	})
	pushEvent(t, fake, mentionOf(member("frontend", "bob", testRoom), self, "and this one"))
	waitNotices(t, sink, arrival, arrival)
}

// A harness mid-turn is not interrupted, however long the turn runs: the report
// is the only thing that speaks for it, and a notice pushed into a turn in
// progress is an interruption nobody asked for. What arrived meanwhile is
// delivered as a count once the turn ends — one line rather than one per
// message, since by then what matters is that something waits.
func TestServer_HoldsAMentionUntilTheTurnEnds(t *testing.T) {
	self := doneSelf("backend", "alice", testRoom)
	fake := &fakeChatService{self: self, events: make(chan *chatv1.RoomEvent)}
	bridge := newTestBridge(t, fake)
	sink := deliversTo(t, bridge)
	serveBridgeAs(t, bridge, "claude-code")

	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })

	waiting := "[crabswarm chat] 2 unread messages mention you" +
		" — read them with the chat_read tool and respond with chat_send," +
		" both from crabswarm-mcp"

	bob := member("frontend", "bob", testRoom)
	pushEvent(t, fake, stateOf(self, chatv1.HarnessState_HARNESS_STATE_WORKING))
	pushEvent(t, fake, mentionOf(bob, self, "the migration needs you"))
	pushEvent(t, fake, mentionOf(bob, self, "and the rebase after it"))
	fake.setUnread(2)

	// The turn ends, and what waited is delivered as the one thing worth saying.
	pushEvent(t, fake, stateOf(self, chatv1.HarnessState_HARNESS_STATE_DONE))
	waitNotices(t, sink, waiting)

	// Nothing waits any more, so the next report that ends a turn says nothing.
	fake.setUnread(0)
	pushEvent(t, fake, stateOf(self, chatv1.HarnessState_HARNESS_STATE_WORKING))
	pushEvent(t, fake, stateOf(self, chatv1.HarnessState_HARNESS_STATE_DONE))
	waitNotices(t, sink, waiting)
}

// Whatever was said while nothing was attending was said to nobody here, so
// every attendance asks what is waiting. A daemon that went away and came back
// is the case that makes it visible; a server that started after its agent was
// mentioned closes the same gap.
func TestServer_DeliversWhatWaitedWhenItAttendsAgain(t *testing.T) {
	self := doneSelf("backend", "alice", testRoom)
	fake := &fakeChatService{
		self:   self,
		events: make(chan *chatv1.RoomEvent),
		drops:  make(chan error),
	}
	bridge := newTestBridge(t, fake)
	runsAt(bridge, 10*time.Millisecond, 20*time.Millisecond)
	sink := deliversTo(t, bridge)
	serveBridgeAs(t, bridge, "claude-code")

	// The first attendance asked and was told nothing waits, which is what makes
	// the mention below one that arrived while nobody was reading.
	waitFor(t, "the server never asked what was waiting",
		func() bool { return fake.countCount() == 1 })
	assert.Assert(t, notices(t, sink) == nil, "delivered with nothing waiting")

	fake.setUnread(1)
	dropFeed(t, fake, status.Error(codes.Unavailable, "the daemon is going away"))

	waitNotices(t, sink,
		"[crabswarm chat] 1 unread message mentions you"+
			" — read it with the chat_read tool and respond with chat_send,"+
			" both from crabswarm-mcp")
}

// A harness this server has no channel for attends as one the daemon types at,
// and is named all the same: what it runs is worth showing on a roster even
// where nothing about the waking turns on it.
func TestServer_AttendsAsTheHarnessThatNamedItself(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	bridge := newTestBridge(t, fake)
	serveBridgeAs(t, bridge, "codex-mcp-client")

	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })
	assert.Equal(t, fake.lastAttend().GetHarness(), chatv1.Harness_HARNESS_CODEX)
	assert.Equal(t, fake.lastAttend().GetNudge(),
		chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL)
	// Nothing is asked of a member the daemon wakes: the count is what a
	// delivery is decided on, and there is no delivery to decide.
	assert.Equal(t, fake.countCount(), 0)
}

// feedingHarness is a harness whose CLI says what its agent is doing without
// being asked, which is what makes the server watch it. It stands in for Codex,
// whose app server this process cannot start; what the case is about is the
// server's half of that — the watch it runs, and the report it makes of every
// state the watch carries.
//
// It is woken through its terminal, so nothing it reports turns into a delivery
// and the trail below is the reporting alone.
type feedingHarness struct {
	// states is what the watch reports. Unbuffered: a case that handed a state
	// over knows the watch took it.
	states chan chatv1.HarnessState
	// watches counts the watches started, which is what pins one for the
	// session rather than one per state.
	watches atomic.Int64
}

func (*feedingHarness) Kind() chatv1.Harness { return chatv1.Harness_HARNESS_CODEX }

func (*feedingHarness) Nudge() chatv1.NudgeDelivery {
	return chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL
}

func (*feedingHarness) Deliver(context.Context, harnessctl.Notice) error {
	return errors.New("this harness delivers nothing")
}

func (h *feedingHarness) Watch(ctx context.Context, report func(chatv1.HarnessState)) {
	h.watches.Add(1)
	for {
		select {
		case <-ctx.Done():
			return
		case state := <-h.states:
			report(state)
		}
	}
}

// says hands one state to the watch, and fails rather than hanging when nothing
// is watching.
func (h *feedingHarness) says(t *testing.T, state chatv1.HarnessState) {
	t.Helper()

	select {
	case h.states <- state:
	case <-time.After(eventTimeout):
		t.Fatalf("nothing is watching the harness to hear %s", state)
	}
}

// runsOn has the server serve h whatever name the client handshakes under,
// which is how a case plays a CLI this process cannot start.
func runsOn(bridge *Server, h harnessctl.Harness) {
	bridge.detect = func(string, func(string) string) harnessctl.Harness { return h }
}

// waitReported blocks until the member has reported exactly want, so a case
// pins both the states that reached the daemon and that nothing else did.
func waitReported(t *testing.T, svc *fakeChatService, want ...chatv1.HarnessState) {
	t.Helper()

	var got []chatv1.HarnessState
	deadline := time.Now().Add(eventTimeout)
	for time.Now().Before(deadline) {
		got = svc.reportedStates()
		if slices.Equal(got, want) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("the member reported\n%v\nwant\n%v", got, want)
}

// A harness that carries a feed of its own is where its member's state comes
// from: the server watches it for the whole session and hands the daemon every
// state it hears, so the room follows the agent's turns with no hook reporting
// anything.
func TestServer_ReportsWhatTheHarnessFeedSays(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	bridge := newTestBridge(t, fake)
	feed := &feedingHarness{states: make(chan chatv1.HarnessState)}
	runsOn(bridge, feed)
	serveBridge(t, bridge)

	// The watch is started once the handshake has named the harness, which is
	// also what the attendance waits for.
	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })

	feed.says(t, chatv1.HarnessState_HARNESS_STATE_WORKING)
	feed.says(t, chatv1.HarnessState_HARNESS_STATE_DONE)
	waitReported(t, fake,
		chatv1.HarnessState_HARNESS_STATE_WORKING,
		chatv1.HarnessState_HARNESS_STATE_DONE)
	assert.Equal(t, feed.watches.Load(), int64(1))
}

// A harness with no feed is left to its hooks: nothing is watched, and the
// member's state is what those hooks report. Watching one that says nothing
// would hold a connection open for a feed that does not exist.
func TestServer_WatchesNothingForAHarnessWithoutAFeed(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	bridge := newTestBridge(t, fake)
	serveBridgeAs(t, bridge, "codex-mcp-client")

	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })
	assert.Assert(t, bridge.harness != nil)
	_, watchable := bridge.harness.(harnessctl.StateSource)
	assert.Assert(t, !watchable, "a Codex with no app server carries a state feed")
	assert.Equal(t, len(fake.reportedStates()), 0)
}

// experimental is whatever the handshake res declared beyond the capabilities
// the SDK fills in for every server, which for this one is nothing at all.
func experimental(res *mcpsdk.InitializeResult) map[string]any {
	if res.Capabilities == nil {
		return nil
	}
	return res.Capabilities.Experimental
}

// plainRevision is the MCP revision an SDK server with nothing added to it
// settles on with the SDK's own client, which is the revision a client gets when
// the server turns none down.
func plainRevision(t *testing.T) string {
	t.Helper()

	serverSide, clientSide := mcpsdk.NewInMemoryTransports()
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "probe", Version: "v0"}, nil)
	ctx, cancel := context.WithCancel(t.Context())
	var served errgroup.Group
	served.Go(func() error { return server.Run(ctx, serverSide) })

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "probe-client", Version: "v0"}, nil)
	session, err := client.Connect(t.Context(), clientSide, nil)
	assert.NilError(t, err)
	t.Cleanup(func() {
		_ = session.Close()
		cancel()
		_ = served.Wait()
	})
	return session.InitializeResult().ProtocolVersion
}

// The server declares no channel of its own and caps no revision: a room notice
// reaches a Claude Code through the plugin's loopback server, which is nothing
// this session negotiates. A capability or a cap left over from one that was
// would cost the agent whatever the revision above it carries, for a channel
// nobody reads.
func TestServer_DeclaresNoChannelOfItsOwn(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	bridge := newTestBridge(t, fake)
	session := serveBridgeAs(t, bridge, "claude-code")

	res := session.InitializeResult()
	assert.Equal(t, len(experimental(res)), 0,
		"the handshake declares %v", experimental(res))
	assert.Equal(t, res.ProtocolVersion, plainRevision(t))

	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })
	assert.Equal(t, fake.lastAttend().GetNudge(),
		chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL)
}

// A server the plugin's channel was wired up beside says in its instructions
// which tool answers a room notice.
//
// The event the model is shown is one that plugin pushed, and the plugin's own
// instructions have it answer with the reply tool — which reaches a browser tab
// nobody is watching while the room goes on waiting. So this says what such an
// event is, which tools the room is read and answered with, and that the reply
// tool is not one of them.
func TestServer_TellsTheModelWhichToolAnswersARoomNotice(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	bridge := newTestBridgeIn(t, fake, launchedWithTheChannel)
	// The harness is played rather than detected: a real one built from this
	// environment would be probed on the plugin's own loopback port.
	runsOn(bridge, &probingHarness{})
	session := serveBridgeAs(t, bridge, "claude-code")

	got := session.InitializeResult().Instructions
	for _, want := range []string{"fakechat", "chat_read", "chat_send", "reply"} {
		assert.Assert(t, strings.Contains(got, want),
			"the instructions do not name %q:\n%s", want, got)
	}
}

// A server nobody wired a channel up beside says nothing. The agent has no
// channel event coming, and instructions about one would be instructions about
// something that never arrives.
func TestServer_SaysNothingAboutAChannelNobodyWiredUp(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	session := serveBridgeAs(t, newTestBridge(t, fake), "claude-code")

	assert.Equal(t, session.InitializeResult().Instructions, "")
}

// probingHarness is a harness whose channel is something other than the session
// this server holds, which is what makes the server ask whether the channel is
// there before the member attends.
//
// It stands in for Claude Code's, whose plugin this process must not go looking
// for: one belonging to whoever runs the suite may well be listening where the
// real probe would look, and a notice posted at it would land in their session.
type probingHarness struct {
	// probes counts the questions asked, which is what pins one per attempt
	// rather than one for the session.
	probes atomic.Int64

	// mu guards err, which a case clears while the attendance loop is asking.
	mu sync.Mutex
	// err is what the channel answers with, or nil for one that is listening.
	err error
}

func (*probingHarness) Kind() chatv1.Harness { return chatv1.Harness_HARNESS_CLAUDE_CODE }

func (*probingHarness) Nudge() chatv1.NudgeDelivery {
	return chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE
}

func (*probingHarness) Deliver(context.Context, harnessctl.Notice) error { return nil }

func (h *probingHarness) Probe(context.Context) error {
	h.probes.Add(1)
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.err
}

// listens has the channel answer from here on, the way a plugin that came up
// after the process beside it does.
func (h *probingHarness) listens() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.err = nil
}

// addMembersTool registers the roster tool in the shape every chat tool has: it
// waits on the attendance before it acts, so what a refused attendance says is
// what the call answers with. The family that registers the real one lives in a
// package that imports this one, so the case spells the one line of it this is
// about.
func addMembersTool(bridge *Server) {
	mcpsdk.AddTool(bridge.MCP(),
		&mcpsdk.Tool{Name: "chat_members", Description: "list everyone attending your room"},
		func(
			ctx context.Context, _ *mcpsdk.CallToolRequest, _ struct{},
		) (*mcpsdk.CallToolResult, any, error) {
			if err := bridge.AwaitAttendance(ctx); err != nil {
				return nil, nil, err
			}
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "attending"}},
			}, nil, nil
		})
}

// toolText unwraps the one text block a tool answered with.
func toolText(t *testing.T, res *mcpsdk.CallToolResult) string {
	t.Helper()

	assert.Equal(t, len(res.Content), 1)
	text, ok := res.Content[0].(*mcpsdk.TextContent)
	assert.Assert(t, ok, "the tool answered with %T, not text", res.Content[0])
	return text.Text
}

// A channel that was declared and is not listening is a member that never
// appears. What the launch variables say is a claim about a plugin nothing in
// this session can see, so the server asks the channel first: a member attending
// on a broken claim attends as one the daemon stops typing at, and every mention
// it is then handed is dropped.
//
// The refusal is what a tool call reads meanwhile, and the question is asked
// again on every attempt, so a plugin that came up late is one the loop picks up.
func TestServer_WaitsForItsChannelBeforeAttending(t *testing.T) {
	fake := &fakeChatService{
		self:   doneSelf("backend", "alice", testRoom),
		events: make(chan *chatv1.RoomEvent),
	}
	bridge := newTestBridge(t, fake)
	runsAt(bridge, 10*time.Millisecond, 20*time.Millisecond)
	const refused = "reaching the fakechat plugin on port 8787: connection refused"
	missing := &probingHarness{err: errors.New(refused)}
	runsOn(bridge, missing)
	addMembersTool(bridge)
	session := serveBridgeAs(t, bridge, "claude-code")

	waitFor(t, "the server never asked its channel twice", func() bool {
		return missing.probes.Load() >= 2
	})
	// The daemon was never asked at all, so the room is a member short — which is
	// the failure somebody reads.
	assert.Equal(t, fake.openCount(), 0)

	// What the agent reads meanwhile is why, down to where the channel was looked
	// for.
	res, err := session.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "chat_members"})
	assert.NilError(t, err)
	assert.Assert(t, res.IsError)
	answer := toolText(t, res)
	assert.Assert(t, strings.Contains(answer, refused), "the tool answered %q", answer)

	missing.listens()
	waitFor(t, "the server never attended once its channel answered", func() bool {
		return fake.attendCount() == 1
	})
	// One attendance and no more: it is held for the rest of the session, and it
	// is the one the daemon leaves to this server.
	assert.Equal(t, fake.openCount(), 1)
	assert.Equal(t, fake.lastAttend().GetHarness(), chatv1.Harness_HARNESS_CLAUDE_CODE)
	assert.Equal(t, fake.lastAttend().GetNudge(),
		chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)
}

func TestNew_RejectsEmptySocketPath(t *testing.T) {
	_, err := New(nil, "", testToken)
	assert.Assert(t, err != nil)
}

// The retry loop comes out of New with a pace of its own. The fields exist so a
// test can slow it down or hold it still, and a server nobody tuned would
// otherwise run it at a zero wait — asking the daemon as fast as it can answer
// for the whole session.
func TestNew_SeedsTheRetryPace(t *testing.T) {
	bridge, err := New(nil, serveTestDaemon(t, &fakeChatService{}), testToken)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = bridge.client.Close() })

	assert.Assert(t, bridge.attendBackoffBase > 0,
		"attending starts at %s", bridge.attendBackoffBase)
	assert.Assert(t, bridge.attendBackoffMax >= bridge.attendBackoffBase,
		"attending climbs to %s, below its first wait of %s",
		bridge.attendBackoffMax, bridge.attendBackoffBase)
}
