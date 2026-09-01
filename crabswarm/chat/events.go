package chat

import (
	"sync"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// roomEventBuffer is how far behind a watcher may fall before it is dropped.
// Deep enough to ride out a burst — a room full of agents reporting state at
// the end of a turn — while the watcher is busy with the previous event, and
// shallow enough that a wedged watcher is noticed instead of being carried.
const roomEventBuffer = 64

// roomBroadcaster fans events out to whoever watches the room they happened in.
//
// Publishing never blocks: a mutation is already recorded by the time it is
// announced, so a watcher that stopped reading must not be able to hold up the
// RPC that produced the news. Such a watcher is dropped rather than served a
// thinned feed — a silently missing member-left leaves its member list wrong
// with nothing to notice, while a dropped stream tells the client to list the
// room again and resubscribe.
type roomBroadcaster struct {
	mu sync.Mutex
	// subs holds the live subscriptions per room. A room with none is dropped
	// from the map: rooms come and go with the sessions attending them.
	subs map[string]map[*roomSubscription]struct{}
}

func newRoomBroadcaster() *roomBroadcaster {
	return &roomBroadcaster{subs: make(map[string]map[*roomSubscription]struct{})}
}

// roomSubscription is one watcher's feed of a room.
type roomSubscription struct {
	room string
	// events is closed when the broadcaster drops the watcher for falling
	// behind. A watcher that leaves on its own leaves the channel open, so a
	// closed one means that and only that.
	events chan *chatv1.RoomEvent
}

// subscribe opens a feed of room. The caller ends it with
// [roomBroadcaster.unsubscribe].
func (b *roomBroadcaster) subscribe(room string) *roomSubscription {
	sub := &roomSubscription{
		room:   room,
		events: make(chan *chatv1.RoomEvent, roomEventBuffer),
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.subs[room] == nil {
		b.subs[room] = make(map[*roomSubscription]struct{})
	}
	b.subs[room][sub] = struct{}{}
	return sub
}

// unsubscribe ends sub's feed. Ending one the broadcaster already dropped is
// fine: a watcher unsubscribes on its way out either way.
func (b *roomBroadcaster) unsubscribe(sub *roomSubscription) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.remove(sub)
}

// publish announces ev to everyone watching room, dropping the watchers whose
// buffer is full. It must be called only once the mutation ev reports has
// actually persisted.
func (b *roomBroadcaster) publish(room string, ev *chatv1.RoomEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for sub := range b.subs[room] {
		select {
		case sub.events <- ev:
		default:
			b.remove(sub)
			close(sub.events)
		}
	}
}

// remove forgets sub. The caller holds the mutex.
func (b *roomBroadcaster) remove(sub *roomSubscription) {
	subs := b.subs[sub.room]
	delete(subs, sub)
	if len(subs) == 0 {
		delete(b.subs, sub.room)
	}
}

// memberJoinedEvent announces that m now attends its room.
func memberJoinedEvent(m Member) *chatv1.RoomEvent {
	return &chatv1.RoomEvent{
		Event: &chatv1.RoomEvent_MemberJoined{
			MemberJoined: &chatv1.MemberJoined{Member: memberProto(m)},
		},
	}
}

// memberLeftEvent announces that m no longer attends its room.
func memberLeftEvent(m Member) *chatv1.RoomEvent {
	return &chatv1.RoomEvent{
		Event: &chatv1.RoomEvent_MemberLeft{
			MemberLeft: &chatv1.MemberLeft{Member: memberProto(m)},
		},
	}
}

// memberStateChangedEvent announces the state m now reports. The state is an
// argument rather than something read off m: m is the member as it stood
// before the report landed, which is the one state the event must not carry.
func memberStateChangedEvent(m Member, state chatv1.HarnessState) *chatv1.RoomEvent {
	member := memberProto(m)
	// m was read before the new state was recorded, so the state it carries is
	// the one being replaced. Announcing the member as still being in it, beside
	// the state it just moved to, would make the event contradict itself.
	member.State = state
	return &chatv1.RoomEvent{
		Event: &chatv1.RoomEvent_MemberStateChanged{
			MemberStateChanged: &chatv1.MemberStateChanged{
				Member: member,
				State:  state,
			},
		},
	}
}
