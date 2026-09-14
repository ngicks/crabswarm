package mcpserver

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// eventTimeout bounds how long a test waits on something the bridge does off
// its own goroutines — announcing a change, attending again. Generous enough to
// ride out a loaded machine and the retry backoff, short enough that a bridge
// that never does it fails instead of hanging the suite.
const eventTimeout = 5 * time.Second

// memberOnRoster is member() with the two things the roster carries beside an
// address: what attends, and what its harness last reported.
func memberOnRoster(
	team, name, room string, kind chatv1.MemberKind, state chatv1.HarnessState,
) *chatv1.Member {
	m := member(team, name, room)
	m.Kind = kind
	m.State = state
	return m
}

func stateChangedEvent(m *chatv1.Member, state chatv1.HarnessState) *chatv1.RoomEvent {
	return &chatv1.RoomEvent{
		Event: &chatv1.RoomEvent_MemberStateChanged{
			MemberStateChanged: &chatv1.MemberStateChanged{Member: m, State: state},
		},
	}
}

func joinedEvent(m *chatv1.Member) *chatv1.RoomEvent {
	return &chatv1.RoomEvent{
		Event: &chatv1.RoomEvent_MemberJoined{
			MemberJoined: &chatv1.MemberJoined{Member: m},
		},
	}
}

func messageAppendedEvent(from *chatv1.Member, text string) *chatv1.RoomEvent {
	return &chatv1.RoomEvent{
		Event: &chatv1.RoomEvent_MessageAppended{
			MessageAppended: &chatv1.MessageAppended{
				Message: &chatv1.Message{From: from, Text: text},
			},
		},
	}
}

// watchedUpdates runs bridge under a harness that records what it announces,
// and returns the session beside the URIs as they arrive. The bridge is the
// caller's to build, since a case about attending again has to set the pace it
// retries at.
func watchedUpdates(
	t *testing.T, bridge *Server,
) (*mcp.ClientSession, <-chan string) {
	t.Helper()

	updated := make(chan string, 8)
	session := serveBridge(t, bridge, &mcp.ClientOptions{
		ResourceUpdatedHandler: func(
			_ context.Context, req *mcp.ResourceUpdatedNotificationRequest,
		) {
			updated <- req.Params.URI
		},
	})
	return session, updated
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

// subscribeToRoster subscribes to the roster and returns once the bridge is
// announcing to the subscription.
//
// Waiting is what makes the cases below deterministic. The SDK's Subscribe
// returns before the server has recorded the subscription, and the feed runs
// from the moment the bridge attends, so an event pushed straight after
// subscribing is regularly announced to nobody. The helper keeps pushing until
// one lands, then waits for the channel to go quiet, so the case that follows
// starts from a subscription that works and an inbox with nothing left in it.
func subscribeToRoster(
	t *testing.T, session *mcp.ClientSession, svc *fakeChatService, updated <-chan string,
) {
	t.Helper()

	assert.NilError(t, session.Subscribe(t.Context(),
		&mcp.SubscribeParams{URI: membersURI}))

	deadline := time.Now().Add(eventTimeout)
	for announced := false; !announced; {
		if time.Now().After(deadline) {
			t.Fatalf("the subscription to %s never took effect", membersURI)
		}
		pushEvent(t, svc, joinedEvent(member("ops", "settling", testRoom)))
		select {
		case <-updated:
			announced = true
		case <-time.After(50 * time.Millisecond):
		}
	}
	for {
		select {
		case <-updated:
		case <-time.After(50 * time.Millisecond):
			return
		}
	}
}

// dropFeed ends the attendance the bridge is holding, the way the daemon does
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

func nextUpdate(t *testing.T, updated <-chan string) string {
	t.Helper()

	select {
	case uri := <-updated:
		return uri
	case <-time.After(eventTimeout):
		t.Fatal("the bridge announced nothing")
		return ""
	}
}

// noMoreUpdates asserts that nothing further is announced. The wait is short:
// it is bounding a notification the bridge would already have sent, not one it
// is expected to get around to.
func noMoreUpdates(t *testing.T, updated <-chan string) {
	t.Helper()

	select {
	case uri := <-updated:
		t.Fatalf("announced %q with nothing to announce", uri)
	case <-time.After(50 * time.Millisecond):
	}
}

// The harness is offered the room's attendance and nothing else. What has been
// said in it is read through chat_read, which moves the read position: a
// document a harness could re-read whenever it liked would consume the unread
// without anybody asking.
func TestServer_ServesTheRoomAsResources(t *testing.T) {
	session := startSession(t, &fakeChatService{self: member("backend", "alice", testRoom)})

	listed, err := session.ListResources(t.Context(), nil)
	assert.NilError(t, err)
	assert.Equal(t, len(listed.Resources), 1)
	assert.Equal(t, listed.Resources[0].URI, membersURI)
	assert.Equal(t, listed.Resources[0].MIMEType, membersMIMEType)
}

// The roster is a resource rather than a fourth tool, and answers as structured
// data: its reader is the harness, which re-reads the room as it changes rather
// than parsing the columns the CLI prints.
func TestServer_ServesTheRoster(t *testing.T) {
	fake := &fakeChatService{
		self: member("backend", "alice", testRoom),
		members: []*chatv1.Member{
			memberOnRoster("backend", "alice", testRoom,
				chatv1.MemberKind_MEMBER_KIND_AGENT,
				chatv1.HarnessState_HARNESS_STATE_WORKING),
			memberOnRoster("frontend", "bob", testRoom,
				chatv1.MemberKind_MEMBER_KIND_HUMAN,
				chatv1.HarnessState_HARNESS_STATE_WAITING),
			// A member the daemon reported neither a kind nor a state for still
			// belongs on the roster: it attends the room either way.
			member("ops", "carol", testRoom),
		},
	}
	session := startSession(t, fake)

	res, err := session.ReadResource(t.Context(),
		&mcp.ReadResourceParams{URI: membersURI})
	assert.NilError(t, err)
	assert.Equal(t, len(res.Contents), 1)
	assert.Equal(t, res.Contents[0].URI, membersURI)
	assert.Equal(t, res.Contents[0].MIMEType, membersMIMEType)

	// Pinned as the text it is rather than as a structure it decodes to: the
	// reader of a resource is whatever the harness hands the document to, so
	// the keys and the words are the interface, not the Go type behind them.
	assert.Equal(t, res.Contents[0].Text, `{
  "room": "/work/proj",
  "members": [
    {
      "address": "backend/alice",
      "team": "backend",
      "name": "alice",
      "kind": "agent",
      "state": "working"
    },
    {
      "address": "frontend/bob",
      "team": "frontend",
      "name": "bob",
      "kind": "human",
      "state": "waiting"
    },
    {
      "address": "ops/carol",
      "team": "ops",
      "name": "carol",
      "kind": "unknown",
      "state": "unknown"
    }
  ]
}`)

	// Reading the roster waits for the attendance the way a tool call does:
	// listing a room from outside it would be asking for a refusal.
	assert.Assert(t, fake.lastAttend() != nil)
}

// A subscribed harness is told to look again whenever the room's attendance or
// anyone's state changes, and left alone otherwise.
func TestServer_AnnouncesTheRosterWhenTheRoomChanges(t *testing.T) {
	fake := &fakeChatService{
		self:    member("backend", "alice", testRoom),
		members: []*chatv1.Member{member("backend", "alice", testRoom)},
		events:  make(chan *chatv1.RoomEvent),
	}
	session, updated := watchedUpdates(t, newTestBridge(t, fake))

	subscribeToRoster(t, session, fake, updated)

	pushEvent(t, fake, stateChangedEvent(
		member("frontend", "bob", testRoom),
		chatv1.HarnessState_HARNESS_STATE_WAITING))
	assert.Equal(t, nextUpdate(t, updated), membersURI)

	// A message being appended leaves the same members in the same states. It
	// changes the room's conversation, but nothing may subscribe to that, so
	// there is nobody to tell. The arrival behind it is what the harness hears
	// about, and only once.
	pushEvent(t, fake, messageAppendedEvent(
		member("frontend", "bob", testRoom), "rebased onto main"))
	pushEvent(t, fake, joinedEvent(member("ops", "carol", testRoom)))
	assert.Equal(t, nextUpdate(t, updated), membersURI)
	noMoreUpdates(t, updated)
}

// The feed runs from the moment the bridge attends, because it is the
// attendance: holding the stream is what makes this member one. A session that
// has not subscribed is told nothing about it, which is the SDK's business, and
// it does not.
func TestServer_HoldsTheFeedBeforeAnythingSubscribes(t *testing.T) {
	fake := &fakeChatService{
		self:    member("backend", "alice", testRoom),
		members: []*chatv1.Member{member("backend", "alice", testRoom)},
		events:  make(chan *chatv1.RoomEvent),
	}
	session, updated := watchedUpdates(t, newTestBridge(t, fake))

	// The feed is up once it can carry an event, and nothing was subscribed to
	// carry it to. One attendance, not several: the bridge attends from the
	// start and the stream it opened is still the one it is reading.
	pushEvent(t, fake, joinedEvent(member("ops", "carol", testRoom)))
	assert.Equal(t, fake.attendCount(), 1)
	noMoreUpdates(t, updated)

	// Subscribing to the feed that was already running is what makes the same
	// event reach the harness.
	subscribeToRoster(t, session, fake, updated)
	pushEvent(t, fake, joinedEvent(member("ops", "dave", testRoom)))
	assert.Equal(t, nextUpdate(t, updated), membersURI)
}

// unwatchable is a URI the bridge does not serve, spelled as one a harness
// might plausibly have reached for.
const unwatchable = "crabswarm://chat/history"

// assertNotFound asserts uri was refused as the missing resource the SDK
// spells. Pinned as that error rather than as any error at all: it is what
// tells a harness the URI is not one this bridge has, so it stops waiting on
// news that could never come.
func assertNotFound(t *testing.T, uri string, err error) {
	t.Helper()

	assert.Assert(t, errors.Is(err, mcp.ResourceNotFoundError(uri)),
		"%s was not refused as a missing resource: %v", uri, err)
}

// Only the roster may be subscribed to. A URI the bridge does not serve is
// refused because the SDK would otherwise record a subscription for whatever it
// was handed and leave the harness waiting.
//
// The handler is exercised directly because the protocol the SDK negotiates
// here opens a subscription without waiting for the answer, so a refusal never
// reaches the client as the error of a call.
func TestServer_RefusesToWatchWhatItCannotAnnounce(t *testing.T) {
	bridge, err := New(slog.New(slog.DiscardHandler),
		serveTestDaemon(t, &fakeChatService{}), testToken)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = bridge.client.Close() })

	assertNotFound(t, unwatchable, bridge.subscribed(t.Context(), &mcp.SubscribeRequest{
		Params: &mcp.SubscribeParams{URI: unwatchable},
	}))
	assert.NilError(t, bridge.subscribed(t.Context(), &mcp.SubscribeRequest{
		Params: &mcp.SubscribeParams{URI: membersURI},
	}))
}

// Withdrawing is answered by the same gate as asking: a harness that was
// refused a subscription has none to withdraw, so telling it the withdrawal
// succeeded would say it had had one.
//
// Exercised directly for the reason the subscribe side is: the SDK does not
// hand either refusal back as the error of a client call.
func TestServer_RefusesToUnwatchWhatItCannotAnnounce(t *testing.T) {
	bridge, err := New(slog.New(slog.DiscardHandler),
		serveTestDaemon(t, &fakeChatService{}), testToken)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = bridge.client.Close() })

	unsubscribe := func(uri string) error {
		return bridge.unsubscribed(t.Context(), &mcp.UnsubscribeRequest{
			Params: &mcp.UnsubscribeParams{URI: uri},
		})
	}

	assert.NilError(t, unsubscribe(membersURI))
	assertNotFound(t, unwatchable, unsubscribe(unwatchable))
}

// A daemon that ended the attendance is answered by attending again — and by
// saying the roster changed as soon as it is back, because whatever happened
// while the bridge was away went unannounced and only a re-read can find it.
func TestServer_AnnouncesTheRosterAfterAttendingAgain(t *testing.T) {
	fake := &fakeChatService{
		self:    member("backend", "alice", testRoom),
		members: []*chatv1.Member{member("backend", "alice", testRoom)},
		events:  make(chan *chatv1.RoomEvent),
		drops:   make(chan error),
	}
	// The case is about the attendance coming back, so the loop that brings it
	// back runs at a pace the case can wait for.
	bridge := newTestBridge(t, fake)
	runsAt(bridge, 10*time.Millisecond, 20*time.Millisecond)
	session, updated := watchedUpdates(t, bridge)

	subscribeToRoster(t, session, fake, updated)

	dropFeed(t, fake, status.Error(codes.ResourceExhausted, "reader fell behind"))

	// The roster is announced as changed as soon as the new attendance is up,
	// without an event to announce: what happened while nobody was reading is
	// exactly what nobody can be told about.
	assert.Equal(t, nextUpdate(t, updated), membersURI)

	// The new feed carries what the dropped one would have. Two attendances in
	// all: the one that ended and the one reading this event.
	pushEvent(t, fake, stateChangedEvent(
		member("frontend", "bob", testRoom),
		chatv1.HarnessState_HARNESS_STATE_DONE))
	assert.Equal(t, nextUpdate(t, updated), membersURI)
	assert.Equal(t, fake.attendCount(), 2)
}
