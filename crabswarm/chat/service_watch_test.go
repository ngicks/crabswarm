package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// watchStream stands in for the gRPC stream WatchRoom writes to, handing the
// test what the service sent. A non-nil release makes Send wait for it, which
// is how a test plays a watcher that stopped reading.
type watchStream struct {
	grpc.ServerStream
	ctx     context.Context
	release chan struct{}
	sent    chan *chatv1.RoomEvent
}

var _ grpc.ServerStreamingServer[chatv1.RoomEvent] = (*watchStream)(nil)

func newWatchStream(ctx context.Context) *watchStream {
	return &watchStream{
		ctx: ctx,
		// Twice the drop threshold: a test may let a stalled stream drain the
		// whole buffer at once, and Send must not be what blocks then.
		sent: make(chan *chatv1.RoomEvent, 2*roomEventBuffer),
	}
}

func (s *watchStream) Context() context.Context { return s.ctx }

func (s *watchStream) Send(ev *chatv1.RoomEvent) error {
	if s.release != nil {
		<-s.release
	}
	s.sent <- ev
	return nil
}

// watch runs WatchRoom for stream and returns the channel its outcome arrives
// on, which is also how a test knows the stream ended.
func watch(svc *Service, stream *watchStream) <-chan error {
	done := make(chan error, 1)
	go func() { done <- svc.WatchRoom(&chatv1.WatchRoomRequest{}, stream) }()
	return done
}

// waitWatching waits until a watcher of room has reached the broadcaster.
// WatchRoom subscribes on its own goroutine — over gRPC, after the client's
// call has already returned — and an event published before that is announced
// to nobody.
func waitWatching(t *testing.T, store *Store, room string) {
	t.Helper()
	deadline := time.Now().Add(eventTimeout)
	for store.events.subscriberCount(room) == 0 {
		if time.Now().After(deadline) {
			t.Fatalf("no watcher of room %q", room)
		}
		time.Sleep(time.Millisecond)
	}
}

// waitUnwatched is [waitWatching] the other way round, for asserting that a
// finished stream left nothing behind.
func waitUnwatched(t *testing.T, store *Store, room string) {
	t.Helper()
	deadline := time.Now().Add(eventTimeout)
	for store.events.subscriberCount(room) != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("room %q still has a watcher", room)
		}
		time.Sleep(time.Millisecond)
	}
}

// awaitOutcome returns what WatchRoom returned, failing the test rather than
// hanging when it is still running.
func awaitOutcome(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(eventTimeout):
		t.Fatal("WatchRoom did not return")
		return nil
	}
}

func TestService_WatchRoomRequiresMembership(t *testing.T) {
	svc, provider, _ := newTestService(t)
	// Known to the provider but attending nothing: there is no room to watch,
	// and the refusal reads as it does for every other member verb.
	provider.vouch("tok-a", "/work", "alpha")

	err := svc.WatchRoom(&chatv1.WatchRoomRequest{},
		newWatchStream(callCtx(t, "tok-a")))
	assert.Equal(t, status.Code(err), codes.Unauthenticated)

	err = svc.WatchRoom(&chatv1.WatchRoomRequest{}, newWatchStream(t.Context()))
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestService_WatchRoomStreamsWhatHappensInTheRoom(t *testing.T) {
	svc, provider, _ := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")
	provider.vouch("tok-b", "/work", "alpha")

	ctx, cancel := context.WithCancel(callCtx(t, "tok-a"))
	defer cancel()
	stream := newWatchStream(ctx)
	done := watch(svc, stream)
	waitWatching(t, svc.store, "/work")

	_, err := svc.Join(callCtx(t, "tok-b"), &chatv1.JoinRequest{Name: "bob"})
	assert.NilError(t, err)
	_, err = svc.ReportState(callCtx(t, "tok-b"), &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_WORKING,
	})
	assert.NilError(t, err)
	_, err = svc.Leave(callCtx(t, "tok-b"), &chatv1.LeaveRequest{})
	assert.NilError(t, err)

	assert.Equal(t, describeEvent(nextEvent(t, stream.sent)), "joined:alpha/bob")
	assert.Equal(t, describeEvent(nextEvent(t, stream.sent)),
		"state:alpha/bob:HARNESS_STATE_WORKING")
	assert.Equal(t, describeEvent(nextEvent(t, stream.sent)), "left:alpha/bob")

	// The client going away ends the stream and takes the subscription with it.
	cancel()
	assert.Assert(t, errors.Is(awaitOutcome(t, done), context.Canceled))
	waitUnwatched(t, svc.store, "/work")
}

// Re-declared attendance is not news: nothing about the room changed.
func TestService_WatchRoomIgnoresARepeatedJoin(t *testing.T) {
	svc, provider, _ := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")

	sub := svc.store.events.subscribe("/work")
	defer svc.store.events.unsubscribe(sub)

	_, err := svc.Join(callCtx(t, "tok-a"), &chatv1.JoinRequest{Name: "ana"})
	assert.NilError(t, err)
	noMoreEvents(t, sub.events)
}

func TestService_WatchRoomIsRoomScoped(t *testing.T) {
	svc, provider, _ := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")
	provider.vouch("tok-c", "/elsewhere", "alpha")

	sub := svc.store.events.subscribe("/work")
	defer svc.store.events.unsubscribe(sub)

	_, err := svc.Join(callCtx(t, "tok-c"), &chatv1.JoinRequest{Name: "cid"})
	assert.NilError(t, err)
	_, err = svc.ReportState(callCtx(t, "tok-c"), &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_WORKING,
	})
	assert.NilError(t, err)
	noMoreEvents(t, sub.events)
}

// A member the provider has forgotten is dropped from the room by whatever call
// noticed, and the room hears about it the same as any other departure — the
// watchers are the other members, still running.
func TestService_WatchRoomSeesAReapedMemberLeave(t *testing.T) {
	svc, _, _ := newTestService(t)
	// Seeded straight into the store: joining through the service would vouch
	// for the token and the reap check would be skipped for the TTL.
	join(t, svc.store, "gone", "/work", "beta", "ghost")

	sub := svc.store.events.subscribe("/work")
	defer svc.store.events.unsubscribe(sub)

	_, err := svc.ListMembers(callCtx(t, "gone"), &chatv1.ListMembersRequest{})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
	assert.Equal(t, describeEvent(nextEvent(t, sub.events)), "left:beta/ghost")
}

// A watcher that stops reading is the daemon's problem only until it is
// dropped: the mutations it is not keeping up with must not slow down for it.
func TestService_SlowWatcherIsDroppedWithoutBlockingMutations(t *testing.T) {
	svc, provider, _ := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")

	stream := newWatchStream(callCtx(t, "tok-a"))
	stream.release = make(chan struct{})
	done := watch(svc, stream)
	waitWatching(t, svc.store, "/work")

	// One event more than the watcher can be behind by, counting the one it is
	// stuck sending.
	states := []chatv1.HarnessState{
		chatv1.HarnessState_HARNESS_STATE_WORKING,
		chatv1.HarnessState_HARNESS_STATE_DONE,
	}
	for i := range roomEventBuffer + 2 {
		_, err := svc.ReportState(callCtx(t, "tok-a"), &chatv1.ReportStateRequest{
			State: states[i%len(states)],
		})
		assert.NilError(t, err, "mutation %d waited for the watcher", i)
	}

	// Let it read again: what it gets is the end of its stream, not a feed with
	// holes it cannot see.
	close(stream.release)
	assert.Equal(t, status.Code(awaitOutcome(t, done)), codes.ResourceExhausted)
	waitUnwatched(t, svc.store, "/work")
}

// TestService_WatchRoomOverGRPC exercises the wiring the daemon uses: a stream
// carries no token without the stream interceptor, so this is what proves the
// RPC works at all for a real client.
func TestService_WatchRoomOverGRPC(t *testing.T) {
	svc, provider, _ := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")
	provider.vouch("tok-b", "/work", "alpha")
	client := dialTestService(t, svc)

	asAna := metadata.AppendToOutgoingContext(t.Context(), TokenMetadataKey, "tok-a")
	// A deadline on the watch is what a missing event runs into: Recv has
	// nothing else to give up against, and a hung test says far less than a
	// failed one.
	watchCtx, cancel := context.WithTimeout(asAna, eventTimeout)
	defer cancel()
	stream, err := client.WatchRoom(watchCtx, &chatv1.WatchRoomRequest{})
	assert.NilError(t, err)
	waitWatching(t, svc.store, "/work")

	asBob := metadata.AppendToOutgoingContext(t.Context(), TokenMetadataKey, "tok-b")
	_, err = client.Join(asBob, &chatv1.JoinRequest{Name: "bob"})
	assert.NilError(t, err)
	_, err = client.ReportState(asBob, &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_WAITING,
	})
	assert.NilError(t, err)
	_, err = client.Leave(asBob, &chatv1.LeaveRequest{})
	assert.NilError(t, err)

	joined, err := stream.Recv()
	assert.NilError(t, err)
	assert.Equal(t, describeEvent(joined), "joined:alpha/bob")
	assert.Equal(t, joined.GetMemberJoined().GetMember().GetRoom(), "/work")

	reported, err := stream.Recv()
	assert.NilError(t, err)
	assert.Equal(t, describeEvent(reported), "state:alpha/bob:HARNESS_STATE_WAITING")

	left, err := stream.Recv()
	assert.NilError(t, err)
	assert.Equal(t, describeEvent(left), "left:alpha/bob")

	// Hanging up ends the stream and unsubscribes it.
	cancel()
	waitUnwatched(t, svc.store, "/work")

	t.Run("a token in no room is refused", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), eventTimeout)
		defer cancel()
		stranger := metadata.AppendToOutgoingContext(
			ctx, TokenMetadataKey, "stranger")
		refused, err := client.WatchRoom(stranger, &chatv1.WatchRoomRequest{})
		if err == nil {
			// A refused stream is answered on the first read: opening one only
			// hands the request to the transport.
			_, err = refused.Recv()
		}
		assert.Equal(t, status.Code(err), codes.Unauthenticated)
	})

	t.Run("a stream carrying no token never reaches the service", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), eventTimeout)
		defer cancel()
		refused, err := client.WatchRoom(ctx, &chatv1.WatchRoomRequest{})
		if err == nil {
			_, err = refused.Recv()
		}
		assert.Equal(t, status.Code(err), codes.Unauthenticated)
	})
}
