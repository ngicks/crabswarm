package issues

import (
	"sync"

	issuesv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/issues/v1"
)

// eventKind classifies a watch [event].
type eventKind int

const (
	// issuesChanged reports that a source's issues moved: the event carries
	// the source and the changed issue ids, empty meaning "refetch all".
	issuesChanged eventKind = iota
	// sourcesChanged reports that the source registry itself changed. It
	// carries neither a source nor ids.
	sourcesChanged
)

// event is one notification fanned out to WatchIssues subscribers.
type event struct {
	kind     eventKind
	sourceID string
	issueIDs []string
}

// subBuffer is the per-subscriber channel depth, deep enough to absorb a
// burst of per-source polls landing together.
const subBuffer = 32

// eventHub fans one [event] out to every current subscriber. It is safe for
// concurrent use and shared by every poller, so a subscriber registers once
// and receives every source's events; sources come and go without
// subscribers re-subscribing.
//
// A slow subscriber never blocks a poller: each subscriber's channel is
// buffered and delivery is non-blocking, dropping when the buffer is full.
// Dropped events are harmless — they are invalidation hints, and a client
// that reconnects after a drop refetches anyway.
type eventHub struct {
	mu   sync.Mutex
	next int
	subs map[int]chan event
}

// newEventHub returns an empty hub ready for use.
func newEventHub() *eventHub {
	return &eventHub{subs: make(map[int]chan event)}
}

// Subscribe registers a subscriber and returns its receive channel plus an
// unsubscribe function. Unsubscribing removes the subscriber and closes the
// channel; it is idempotent.
func (h *eventHub) Subscribe() (<-chan event, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	id := h.next
	h.next++
	ch := make(chan event, subBuffer)
	h.subs[id] = ch

	var once sync.Once
	unsub := func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			if c, ok := h.subs[id]; ok {
				delete(h.subs, id)
				close(c)
			}
		})
	}
	return ch, unsub
}

// Publish fans ev out to every current subscriber without blocking. Both
// Publish and the unsubscribe closure hold h.mu, so a send never races a
// channel close.
func (h *eventHub) Publish(ev event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ch := range h.subs {
		select {
		case ch <- ev:
		default:
			// Slow subscriber: drop rather than stall the poller.
		}
	}
}

// eventToProto renders an event as one WatchIssues stream message.
func eventToProto(ev event) *issuesv1.WatchIssuesResponse {
	switch ev.kind {
	case sourcesChanged:
		return &issuesv1.WatchIssuesResponse{
			Event: &issuesv1.WatchIssuesResponse_SourcesChanged{
				SourcesChanged: &issuesv1.SourcesChanged{},
			},
		}
	default:
		return &issuesv1.WatchIssuesResponse{
			Event: &issuesv1.WatchIssuesResponse_IssuesChanged{
				IssuesChanged: &issuesv1.IssuesChanged{
					SourceId: ev.sourceID,
					IssueIds: ev.issueIDs,
				},
			},
		}
	}
}
