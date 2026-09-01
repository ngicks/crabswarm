package chat

import (
	"fmt"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// eventTimeout is how long a test waits for an event that should already be on
// its way. Generous: it is only ever spent by a test that is about to fail.
const eventTimeout = 5 * time.Second

// subscriberCount reports how many watchers room has. It exists for the tests:
// a stream's subscription lands somewhere between the client's call returning
// and the handler running, and this is what lets a test wait for it.
func (b *roomBroadcaster) subscriberCount(room string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs[room])
}

// nextEvent returns the next event on the feed, failing the test instead of
// hanging when none arrives.
func nextEvent(t *testing.T, events <-chan *chatv1.RoomEvent) *chatv1.RoomEvent {
	t.Helper()
	select {
	case ev, ok := <-events:
		assert.Assert(t, ok, "the watcher was dropped instead of served an event")
		return ev
	case <-time.After(eventTimeout):
		t.Fatal("timed out waiting for a room event")
		return nil
	}
}

// noMoreEvents asserts that nothing else was announced. A mutation publishes
// inside the RPC that made it, so whatever a returned call announced is already
// on the feed and nothing has to be waited for.
func noMoreEvents(t *testing.T, events <-chan *chatv1.RoomEvent) {
	t.Helper()
	select {
	case ev, ok := <-events:
		if !ok {
			t.Fatal("the watcher was dropped")
		}
		t.Fatalf("unexpected room event: %s", describeEvent(ev))
	default:
	}
}

// describeEvent renders an event as "kind:team/name[:state]", which is what the
// tests assert on: one comparison covers the kind and who it is about, and an
// event of the wrong kind fails as a wrong kind rather than as an empty name.
func describeEvent(ev *chatv1.RoomEvent) string {
	switch e := ev.GetEvent().(type) {
	case *chatv1.RoomEvent_MemberJoined:
		return "joined:" + address(e.MemberJoined.GetMember())
	case *chatv1.RoomEvent_MemberLeft:
		return "left:" + address(e.MemberLeft.GetMember())
	case *chatv1.RoomEvent_MemberStateChanged:
		return "state:" + address(e.MemberStateChanged.GetMember()) +
			":" + e.MemberStateChanged.GetState().String()
	default:
		return fmt.Sprintf("unknown:%v", ev)
	}
}

func address(m *chatv1.Member) string {
	return m.GetTeam() + "/" + m.GetName()
}

// testEvent is a stand-in event; the broadcaster carries whatever it is given,
// so its own tests need nothing more specific.
func testEvent(name string) *chatv1.RoomEvent {
	return memberJoinedEvent(Member{Name: name, Team: "alpha", Room: "/work"})
}

func TestRoomBroadcaster_FansOutWithinOneRoom(t *testing.T) {
	b := newRoomBroadcaster()
	first := b.subscribe("/work")
	second := b.subscribe("/work")
	elsewhere := b.subscribe("/other")

	b.publish("/work", testEvent("ana"))

	assert.Equal(t, describeEvent(nextEvent(t, first.events)), "joined:alpha/ana")
	assert.Equal(t, describeEvent(nextEvent(t, second.events)), "joined:alpha/ana")
	// A room's news never leaves it: rooms are what members can see of each
	// other.
	noMoreEvents(t, elsewhere.events)

	// A room nobody watches costs nothing to publish to.
	b.publish("/nowhere", testEvent("nobody"))
}

func TestRoomBroadcaster_UnsubscribeStopsDelivery(t *testing.T) {
	b := newRoomBroadcaster()
	sub := b.subscribe("/work")
	other := b.subscribe("/work")

	b.unsubscribe(sub)
	b.publish("/work", testEvent("ana"))

	// The feed goes quiet without being closed: a closed one is how a dropped
	// watcher is told it was dropped, and this one left of its own accord.
	noMoreEvents(t, sub.events)
	assert.Equal(t, b.subscriberCount("/work"), 1)
	assert.Equal(t, describeEvent(nextEvent(t, other.events)), "joined:alpha/ana")

	// Unsubscribing twice is what a watcher dropped mid-stream does on its way
	// out.
	b.unsubscribe(sub)
	b.unsubscribe(other)
	assert.Equal(t, b.subscriberCount("/work"), 0)
}

func TestRoomBroadcaster_DropsAWatcherThatFallsBehind(t *testing.T) {
	b := newRoomBroadcaster()
	slow := b.subscribe("/work")
	keeping := b.subscribe("/work")

	// One more than fits: the last publish has nowhere to put the event.
	for i := range roomEventBuffer + 1 {
		b.publish("/work", testEvent(fmt.Sprintf("ana-%d", i)))
		// The other watcher keeps up, which is what makes this a test of one
		// slow watcher rather than of two.
		assert.Equal(t, describeEvent(nextEvent(t, keeping.events)),
			fmt.Sprintf("joined:alpha/ana-%d", i))
	}

	// The slow one is gone, and its closed feed is what tells its stream so.
	assert.Equal(t, b.subscriberCount("/work"), 1)
	for range roomEventBuffer {
		assert.Assert(t, <-slow.events != nil)
	}
	_, open := <-slow.events
	assert.Assert(t, !open, "a dropped watcher's feed should be closed")

	// Publishing after the drop still reaches the watcher that kept up.
	b.publish("/work", testEvent("bob"))
	assert.Equal(t, describeEvent(nextEvent(t, keeping.events)), "joined:alpha/bob")
}
