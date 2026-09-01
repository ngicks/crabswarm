package mcpserver

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// eventTimeout bounds how long a test waits on something the bridge does off
// its own goroutines — announcing a change, watching the room again. Generous
// enough to ride out a loaded machine and the retry backoff, short enough that
// a bridge that never does it fails instead of hanging the suite.
const eventTimeout = 5 * time.Second

func memberInState(
	team, name, room string, state chatv1.HarnessState,
) *chatv1.Member {
	m := member(team, name, room)
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

// watchedUpdates connects a harness that records what the bridge announces, and
// returns the session beside the URIs as they arrive.
func watchedUpdates(
	t *testing.T, svc *fakeChatService,
) (*mcp.ClientSession, <-chan string) {
	t.Helper()

	updated := make(chan string, 8)
	session := startSessionWith(t, svc, &mcp.ClientOptions{
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
// nothing is watching.
func pushEvent(t *testing.T, svc *fakeChatService, ev *chatv1.RoomEvent) {
	t.Helper()

	select {
	case svc.events <- ev:
	case <-time.After(eventTimeout):
		t.Fatal("nothing is watching the room")
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

// resourcesByURI is what the harness was offered, keyed by the URI it would ask
// for. Keyed rather than indexed: the order a listing comes back in is the
// SDK's business, not something a harness may depend on.
func resourcesByURI(t *testing.T, session *mcp.ClientSession) map[string]*mcp.Resource {
	t.Helper()

	listed, err := session.ListResources(t.Context(), nil)
	assert.NilError(t, err)
	byURI := map[string]*mcp.Resource{}
	for _, r := range listed.Resources {
		byURI[r.URI] = r
	}
	return byURI
}

// The harness is offered the room in two documents and no others: who is in it,
// and what has been said in it.
func TestServer_ServesTheRoomAsResources(t *testing.T) {
	session := startSession(t, &fakeChatService{self: member("backend", "alice", testRoom)})

	offered := resourcesByURI(t, session)
	assert.Equal(t, len(offered), 2)
	assert.Equal(t, offered[membersURI].MIMEType, membersMIMEType)
	assert.Equal(t, offered[historyURI].MIMEType, historyMIMEType)
}

// The roster is a resource rather than a fifth tool, and answers as structured
// data: its reader is the harness, and a member's state is not in the listing
// the CLI prints at all.
func TestServer_ServesTheRoster(t *testing.T) {
	fake := &fakeChatService{
		self: member("backend", "alice", testRoom),
		members: []*chatv1.Member{
			memberInState("backend", "alice", testRoom,
				chatv1.HarnessState_HARNESS_STATE_WORKING),
			memberInState("frontend", "bob", testRoom,
				chatv1.HarnessState_HARNESS_STATE_WAITING),
			// A member the daemon reported no state for still belongs on the
			// roster: it attends the room either way.
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
      "state": "working"
    },
    {
      "address": "frontend/bob",
      "team": "frontend",
      "name": "bob",
      "state": "waiting"
    },
    {
      "address": "ops/carol",
      "team": "ops",
      "name": "carol",
      "state": "unknown"
    }
  ]
}`)

	// Reading the roster attends the room first, the way a tool call does:
	// listing a room from outside it would be asking for a refusal.
	assert.Assert(t, fake.lastJoin() != nil)
}

// The transcript is handed over in the words `crabswarm chat history` prints,
// down to the trailing newline — the same promise the tools make. It is pinned
// against the renderer rather than against a transcript spelled out here: the
// point is that the two never drift, not what today's wording happens to be.
func TestServer_ServesTheTranscript(t *testing.T) {
	entries := []*chatv1.HistoryEntry{{
		From:   member("frontend", "bob", testRoom),
		To:     member("backend", "alice", testRoom),
		Text:   "rebased onto main",
		SentAt: timestamppb.New(time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)),
	}, {
		From:   member("backend", "alice", testRoom),
		Text:   "pulling now",
		SentAt: timestamppb.New(time.Date(2026, 8, 31, 12, 1, 0, 0, time.UTC)),
	}}
	fake := &fakeChatService{
		self:    member("backend", "alice", testRoom),
		entries: entries,
	}
	session := startSession(t, fake)

	res, err := session.ReadResource(t.Context(),
		&mcp.ReadResourceParams{URI: historyURI})
	assert.NilError(t, err)
	assert.Equal(t, len(res.Contents), 1)
	assert.Equal(t, res.Contents[0].URI, historyURI)
	assert.Equal(t, res.Contents[0].MIMEType, historyMIMEType)

	var rendered strings.Builder
	assert.NilError(t, cli.RenderHistory(&rendered, entries))
	assert.Equal(t, res.Contents[0].Text, rendered.String())

	// A read carries no window to ask for, so it asks for none and takes the
	// one the daemon defaults to.
	assert.Equal(t, fake.lastHistory().GetLimit(), int32(0))
	assert.Assert(t, fake.lastJoin() != nil)
}

// A room nobody has spoken in answers in the CLI's words too. The resource is a
// read like any other: content saying so beats content that is empty, which a
// reader cannot tell from a read that never happened.
func TestServer_ServesAnEmptyTranscript(t *testing.T) {
	session := startSession(t, &fakeChatService{self: member("backend", "alice", testRoom)})

	res, err := session.ReadResource(t.Context(),
		&mcp.ReadResourceParams{URI: historyURI})
	assert.NilError(t, err)
	assert.Equal(t, len(res.Contents), 1)

	var rendered strings.Builder
	assert.NilError(t, cli.RenderHistory(&rendered, nil))
	assert.Equal(t, res.Contents[0].Text, rendered.String())
}

// A subscribed harness is told to look again whenever the room's attendance or
// anyone's state changes, and left alone otherwise.
func TestServer_AnnouncesTheRosterWhenTheRoomChanges(t *testing.T) {
	fake := &fakeChatService{
		self:    member("backend", "alice", testRoom),
		members: []*chatv1.Member{member("backend", "alice", testRoom)},
		events:  make(chan *chatv1.RoomEvent),
	}
	session, updated := watchedUpdates(t, fake)

	assert.NilError(t, session.Subscribe(t.Context(),
		&mcp.SubscribeParams{URI: membersURI}))

	pushEvent(t, fake, stateChangedEvent(
		member("frontend", "bob", testRoom),
		chatv1.HarnessState_HARNESS_STATE_WAITING))
	assert.Equal(t, nextUpdate(t, updated), membersURI)

	// A message being appended leaves the same members in the same states. It
	// changes the transcript, but nothing may subscribe to that, so there is
	// nobody to tell. The join behind it is what the harness hears about, and
	// only once.
	pushEvent(t, fake, messageAppendedEvent(
		member("frontend", "bob", testRoom), "rebased onto main"))
	pushEvent(t, fake, joinedEvent(member("ops", "carol", testRoom)))
	assert.Equal(t, nextUpdate(t, updated), membersURI)
	noMoreUpdates(t, updated)
}

// Nothing is watched until something asks to be told. A harness that lists what
// the bridge offers and never subscribes should cost the daemon no stream.
func TestServer_WatchesNothingUntilSomethingSubscribes(t *testing.T) {
	fake := &fakeChatService{
		self:    member("backend", "alice", testRoom),
		members: []*chatv1.Member{member("backend", "alice", testRoom)},
		events:  make(chan *chatv1.RoomEvent),
	}
	session := startSession(t, fake)

	_, err := session.ReadResource(t.Context(),
		&mcp.ReadResourceParams{URI: membersURI})
	assert.NilError(t, err)
	assert.Equal(t, fake.watchCount(), 0)

	assert.NilError(t, session.Subscribe(t.Context(),
		&mcp.SubscribeParams{URI: membersURI}))
	// The feed is up once it can carry an event.
	pushEvent(t, fake, joinedEvent(member("ops", "carol", testRoom)))
	assert.Equal(t, fake.watchCount(), 1)
}

// Only the roster may be subscribed to. A URI the bridge does not serve is
// refused because the SDK would otherwise leave the harness waiting on news
// that could never come; the transcript is refused because the room's feed
// carries nothing that would announce it, which is the same waiting arrived at
// from the other side. Neither starts a feed.
//
// The handler is exercised directly because the protocol the SDK negotiates
// here opens a subscription without waiting for the answer, so a refusal never
// reaches the client as the error of a call.
func TestServer_RefusesToWatchWhatItCannotAnnounce(t *testing.T) {
	bridge, err := New(slog.New(slog.DiscardHandler),
		serveTestDaemon(t, &fakeChatService{}), testToken)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = bridge.client.Close() })

	for _, uri := range []string{"crabswarm://chat/rooms", historyURI} {
		err = bridge.subscribed(t.Context(), &mcp.SubscribeRequest{
			Params: &mcp.SubscribeParams{URI: uri},
		})
		assert.Assert(t, err != nil, "subscribing to %s was accepted", uri)
		select {
		case <-bridge.watchWanted:
			t.Fatalf("subscribing to %s started the room feed", uri)
		default:
		}
	}

	assert.NilError(t, bridge.subscribed(t.Context(), &mcp.SubscribeRequest{
		Params: &mcp.SubscribeParams{URI: membersURI},
	}))
	<-bridge.watchWanted
}

// The daemon drops a watcher that falls behind, so the bridge watches again —
// and says the roster changed as soon as it is back, because whatever happened
// while nothing was watching went unannounced and only a re-read can find it.
func TestServer_WatchesAgainAfterTheFeedEnds(t *testing.T) {
	fake := &fakeChatService{
		self:          member("backend", "alice", testRoom),
		members:       []*chatv1.Member{member("backend", "alice", testRoom)},
		events:        make(chan *chatv1.RoomEvent),
		watchFailures: 1,
	}
	session, updated := watchedUpdates(t, fake)

	assert.NilError(t, session.Subscribe(t.Context(),
		&mcp.SubscribeParams{URI: membersURI}))

	assert.Equal(t, nextUpdate(t, updated), membersURI)

	// The new feed carries what the dropped one would have. Taking the event is
	// also what proves a second watcher is up: the first was refused before it
	// could read anything.
	pushEvent(t, fake, stateChangedEvent(
		member("frontend", "bob", testRoom),
		chatv1.HarnessState_HARNESS_STATE_DONE))
	assert.Equal(t, fake.watchCount(), 2)
	assert.Equal(t, nextUpdate(t, updated), membersURI)
}
