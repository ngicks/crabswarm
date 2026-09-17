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
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/mcp/harness"
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
func newTestBridge(t *testing.T, svc *fakeChatService) *Server {
	t.Helper()

	bridge, err := New(slog.New(slog.DiscardHandler), serveTestDaemon(t, svc), testToken)
	assert.NilError(t, err)
	return bridge
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
		if name == harness.SinkEnv {
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

	pushEvent(t, fake, mentionOf(member("frontend", "bob", testRoom), self, "rebase please"))
	waitNotices(t, sink,
		"[crabswarm chat] new message from frontend/bob — read it with the chat_read tool")

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
	waitNotices(t, sink,
		"[crabswarm chat] new message from frontend/bob — read it with the chat_read tool",
		"[crabswarm chat] new message from frontend/bob — read it with the chat_read tool")
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

	bob := member("frontend", "bob", testRoom)
	pushEvent(t, fake, stateOf(self, chatv1.HarnessState_HARNESS_STATE_WORKING))
	pushEvent(t, fake, mentionOf(bob, self, "the migration needs you"))
	pushEvent(t, fake, mentionOf(bob, self, "and the rebase after it"))
	fake.setUnread(2)

	// The turn ends, and what waited is delivered as the one thing worth saying.
	pushEvent(t, fake, stateOf(self, chatv1.HarnessState_HARNESS_STATE_DONE))
	waitNotices(t, sink,
		"[crabswarm chat] 2 unread messages mention you — read them with the chat_read tool")

	// Nothing waits any more, so the next report that ends a turn says nothing.
	fake.setUnread(0)
	pushEvent(t, fake, stateOf(self, chatv1.HarnessState_HARNESS_STATE_WORKING))
	pushEvent(t, fake, stateOf(self, chatv1.HarnessState_HARNESS_STATE_DONE))
	waitNotices(t, sink,
		"[crabswarm chat] 2 unread messages mention you — read them with the chat_read tool")
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
		"[crabswarm chat] 1 unread message mentions you — read it with the chat_read tool")
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

// declaresChannel reports whether the handshake res carries the capability
// Claude Code registers a channel listener on.
func declaresChannel(res *mcpsdk.InitializeResult) bool {
	if res.Capabilities == nil {
		return false
	}
	_, declared := res.Capabilities.Experimental[harness.ClaudeChannelCapability]
	return declared
}

// A server its launcher registered as a channel says so in the handshake, since
// the capability is what makes Claude Code listen — and then attends as a
// member the daemon leaves to its own server.
//
// The environment is read before the handshake rather than after: the
// capability has to be in the answer the server gives, and the answer is what
// tells the server which harness it was serving all along.
func TestServer_DeclaresTheChannelItsLauncherRegistered(t *testing.T) {
	t.Setenv(harness.ClaudeChannelEnv, "1")
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	bridge := newTestBridge(t, fake)
	session := serveBridgeAs(t, bridge, "claude-code")

	res := session.InitializeResult()
	assert.Assert(t, declaresChannel(res), "the handshake declares no channel")
	// The model is shown a channel event and nothing else, so the instructions
	// are where it reads what to do with one.
	for _, want := range []string{serverName, "chat_read", "chat_send"} {
		assert.Assert(t, strings.Contains(res.Instructions, want),
			"the instructions do not name %q:\n%s", want, res.Instructions)
	}
	// Above this revision the events would be dropped unseen, so the server
	// offers nothing above it.
	assert.Equal(t, res.ProtocolVersion, maxProtocolVersion)

	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })
	assert.Equal(t, fake.lastAttend().GetHarness(), chatv1.Harness_HARNESS_CLAUDE_CODE)
	assert.Equal(t, fake.lastAttend().GetNudge(),
		chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)
}

// A Claude Code nobody registered the server with declares nothing, and is
// woken through its terminal like any harness with no channel.
//
// Declaring the capability anyway would have Claude Code listening on a channel
// its session never registered, and every notice pushed at it would be dropped
// unseen while the daemon typed at nobody.
func TestServer_LeavesTheChannelOutUntilItIsRegistered(t *testing.T) {
	t.Setenv(harness.ClaudeChannelEnv, "")
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	bridge := newTestBridge(t, fake)
	session := serveBridgeAs(t, bridge, "claude-code")

	res := session.InitializeResult()
	assert.Assert(t, !declaresChannel(res), "the handshake declares a channel nobody registered")
	assert.Equal(t, res.Instructions, "")

	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })
	assert.Equal(t, fake.lastAttend().GetNudge(),
		chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL)
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
