package chat

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// followStream is one open Follow stream: the events the service wrote down it,
// and the cancel that closes it the way a client going away does.
type followStream struct {
	grpc.ServerStream
	ctx    context.Context
	cancel context.CancelFunc
	// release, when set, holds every event after Followed until it is closed,
	// which is how a test plays a client that stopped reading.
	release chan struct{}
	// opened records that Followed is through. Written by the Follow goroutine
	// alone, which is the only caller of Send.
	opened bool
	sent   chan *chatv1.FollowEvent
	// done carries what Follow returned, selectable against a timeout so a
	// stream that never ends fails the test instead of hanging it.
	done chan error
}

var _ grpc.ServerStreamingServer[chatv1.FollowEvent] = (*followStream)(nil)

func (s *followStream) Context() context.Context { return s.ctx }

func (s *followStream) Send(ev *chatv1.FollowEvent) error {
	if s.release != nil && s.opened {
		<-s.release
	}
	s.opened = true
	s.sent <- ev
	return nil
}

// newFollowStream builds the stream a test hands Follow, over the request
// context ctx, not yet running. A test that plays a stalled client fills in
// release before starting it.
func newFollowStream(t *testing.T, ctx context.Context) *followStream {
	t.Helper()
	ctx, cancel := context.WithCancel(ctx)
	s := &followStream{
		ctx:    ctx,
		cancel: cancel,
		// Room for a stalled stream to drain everything a dropped subscription
		// had buffered without Send being what blocks.
		sent: make(chan *chatv1.FollowEvent, 4*roomEventBuffer),
		done: make(chan error, 1),
	}
	t.Cleanup(func() {
		cancel()
		select {
		case <-s.done:
		case <-time.After(eventTimeout):
		}
	})
	return s
}

// member starts [Service.Follow] on the stream and returns it, still running.
// A nil since starts it live.
func (s *followStream) member(svc *Service, since *int64) *followStream {
	go func() {
		s.done <- svc.Follow(&chatv1.FollowRequest{Since: since}, s)
	}()
	return s
}

// admin starts [AdminService.Follow] on the stream and returns it, still
// running. A nil since starts it live.
func (s *followStream) admin(svc *AdminService, room string, since *int64) *followStream {
	go func() {
		s.done <- svc.Follow(&chatv1.AdminFollowRequest{Room: room, Since: since}, s)
	}()
	return s
}

// next is the next event on the stream, rendered by [describeFollowEvent].
func (s *followStream) next(t *testing.T) string {
	t.Helper()
	select {
	case ev := <-s.sent:
		return describeFollowEvent(ev)
	case <-time.After(eventTimeout):
		t.Fatal("timed out waiting for a follow event")
		return ""
	}
}

// close ends the stream the way a client going away does and returns what
// Follow returned.
func (s *followStream) close(t *testing.T) error {
	t.Helper()
	s.cancel()
	return s.wait(t)
}

// wait returns what Follow returned, failing the test rather than hanging when
// the stream is still running.
func (s *followStream) wait(t *testing.T) error {
	t.Helper()
	select {
	case err := <-s.done:
		s.done <- err
		return err
	case <-time.After(eventTimeout):
		t.Fatal("the follow stream did not return")
		return nil
	}
}

// describeFollowEvent renders an event as "followed:<room>:<last seq>" or
// "message:<seq>:<team/name>:<text>", with " (mentioned)" on a message that
// mentions the follower. One comparison covers the order, the seq and the flag.
func describeFollowEvent(ev *chatv1.FollowEvent) string {
	switch e := ev.GetEvent().(type) {
	case *chatv1.FollowEvent_Followed:
		return fmt.Sprintf("followed:%s:%d", e.Followed.GetRoom(), e.Followed.GetLastSeq())
	case *chatv1.FollowEvent_Message:
		msg := e.Message
		out := fmt.Sprintf("message:%d:%s:%s", msg.GetSeq(), address(msg.GetFrom()), msg.GetText())
		if ev.GetMentionedYou() {
			out += " (mentioned)"
		}
		return out
	default:
		return fmt.Sprintf("unknown:%v", ev)
	}
}

// say sends text the way both services do, stored and then announced, failing
// the test on error.
func say(t *testing.T, d deliverer, from Sender, target Target, text string) Sent {
	t.Helper()
	sent, err := d.send(t.Context(), from, target, text, sentAt)
	assert.NilError(t, err)
	return sent
}

// announce publishes a message the store already holds, the way a send that
// lost a race to another one announces it late.
func announce(s *Store, sent Sent) {
	s.events.publish(sent.Message.Room, messageAppendedEvent(sent.Message))
}

// Two sends can be announced in the opposite order from their seqs, since each
// announces after the store let go of it. The stream reads the gap out of the
// store before the message that revealed it, and the late announcement of what
// it already sent is dropped.
func TestFollow_RepairsAnOutOfOrderAnnouncement(t *testing.T) {
	svc, id := newTestAdminService(t)
	ana := senderOf(attend(t, svc.store, "tok-a", testRoom, "alpha", "ana"))

	f := newFollowStream(t, adminCtx(t, adminNonce(t, svc, id))).admin(svc, testRoom, nil)
	assert.Equal(t, f.next(t), "followed:/work/repo:0")

	first := send(t, svc.store, ana, Target{}, "first")
	second := send(t, svc.store, ana, Target{}, "second")
	announce(svc.store, second)
	announce(svc.store, first)
	// The next message is the barrier: had the late announcement gone out, it
	// would be on the stream ahead of this one.
	say(t, svc.deliver, ana, Target{}, "third")

	assert.Equal(t, f.next(t), "message:1:alpha/ana:first")
	assert.Equal(t, f.next(t), "message:2:alpha/ana:second")
	assert.Equal(t, f.next(t), "message:3:alpha/ana:third")
}

// A message appended before the stream read the room and announced after it is
// one the stream already holds, by the replay or by starting past it.
func TestFollow_DropsWhatItAlreadySent(t *testing.T) {
	svc, id := newTestAdminService(t)
	ana := senderOf(attend(t, svc.store, "tok-a", testRoom, "alpha", "ana"))
	var held []Sent
	for i := range 4 {
		held = append(held, send(t, svc.store, ana, Target{}, fmt.Sprintf("note-%d", i)))
	}

	f := newFollowStream(t, adminCtx(t, adminNonce(t, svc, id))).admin(svc, testRoom, new(int64(2)))
	assert.Equal(t, f.next(t), "followed:/work/repo:4")
	assert.Equal(t, f.next(t), "message:3:alpha/ana:note-2")
	assert.Equal(t, f.next(t), "message:4:alpha/ana:note-3")

	for _, sent := range held {
		announce(svc.store, sent)
	}
	say(t, svc.deliver, ana, Target{}, "live")
	assert.Equal(t, f.next(t), "message:5:alpha/ana:live")
}

// A replay longer than one page is fed page by page and still arrives whole and
// in order.
func TestFollow_ReplaysPastOnePage(t *testing.T) {
	svc, id := newTestAdminService(t)
	ana := senderOf(attend(t, svc.store, "tok-a", testRoom, "alpha", "ana"))
	total := 2*followPage + 5
	for i := range total {
		send(t, svc.store, ana, Target{}, fmt.Sprintf("note-%d", i))
	}

	f := newFollowStream(t, adminCtx(t, adminNonce(t, svc, id))).admin(svc, testRoom, new(int64(1)))
	assert.Equal(t, f.next(t), fmt.Sprintf("followed:/work/repo:%d", total))
	for seq := 2; seq <= total; seq++ {
		assert.Equal(t, f.next(t), fmt.Sprintf("message:%d:alpha/ana:note-%d", seq, seq-1))
	}
}

// The feed is the room's, so it carries arrivals and departures too; the stream
// is the conversation alone.
func TestFollow_CarriesMessagesOnly(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouchNamed("tok-f", testRoom, "alpha", "fay")

	f := newFollowStream(t, callCtx(t, "tok-f")).member(svc, nil)
	assert.Equal(t, f.next(t), "followed:/work/repo:0")

	bob := agent(t, svc, provider, "tok-b", testRoom, "alpha", "bob")
	_, err := svc.ReportState(callCtx(t, "tok-b"), &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_WORKING,
	})
	assert.NilError(t, err)
	_, err = svc.Send(callCtx(t, "tok-b"), &chatv1.SendRequest{Text: "hello"})
	assert.NilError(t, err)
	assert.Assert(t, bob.close(t) != nil)
	// Bob's departure is on the feed by now; the next message is the barrier
	// that would sit behind anything the stream made of it.
	say(t, svc.deliver, Sender{Name: "bob", Team: "alpha", Room: testRoom}, Target{}, "after")

	assert.Equal(t, f.next(t), "message:1:alpha/bob:hello")
	assert.Equal(t, f.next(t), "message:2:alpha/bob:after")
}

// A follower resuming after a seq the room no longer reaches holds one from
// before the room's numbering started over. The stream starts from the room's
// newest seq, so the messages said from then on reach it.
func TestFollow_SincePastTheNewestStartsFromTheNewest(t *testing.T) {
	svc, id := newTestAdminService(t)
	host := adminSender(testRoom)
	for i := range 3 {
		say(t, svc.deliver, host, Target{}, fmt.Sprintf("old-%d", i))
	}
	_, err := svc.store.DeleteRoom(t.Context(), testRoom)
	assert.NilError(t, err)
	say(t, svc.deliver, host, Target{}, "restarted")

	f := newFollowStream(t, adminCtx(t, adminNonce(t, svc, id))).admin(svc, testRoom, new(int64(3)))
	assert.Equal(t, f.next(t), "followed:/work/repo:1")

	say(t, svc.deliver, host, Target{}, "next")
	assert.Equal(t, f.next(t), "message:2:/"+adminName+":next")
}

// A room deleted under an open stream and spoken in again numbers from one
// again. The stream opens over with a Followed carrying zero and carries the
// restarted room from its first message.
func TestFollow_StartsOverWithARoomDeletedUnderIt(t *testing.T) {
	svc, id := newTestAdminService(t)
	host := adminSender(testRoom)

	f := newFollowStream(t, adminCtx(t, adminNonce(t, svc, id))).admin(svc, testRoom, nil)
	assert.Equal(t, f.next(t), "followed:/work/repo:0")
	say(t, svc.deliver, host, Target{}, "old-1")
	say(t, svc.deliver, host, Target{}, "old-2")
	assert.Equal(t, f.next(t), "message:1:/"+adminName+":old-1")
	assert.Equal(t, f.next(t), "message:2:/"+adminName+":old-2")

	_, err := svc.store.DeleteRoom(t.Context(), testRoom)
	assert.NilError(t, err)
	say(t, svc.deliver, host, Target{}, "new-1")
	assert.Equal(t, f.next(t), "followed:/work/repo:0")
	assert.Equal(t, f.next(t), "message:1:/"+adminName+":new-1")

	say(t, svc.deliver, host, Target{}, "new-2")
	assert.Equal(t, f.next(t), "message:2:/"+adminName+":new-2")
}

// A restarted room that grew back past the stream's seq before its first
// message was announced holds a message at that seq the stream never sent,
// which tells the restart apart from a late announcement.
func TestFollow_StartsOverWithARoomThatGrewBackPastIt(t *testing.T) {
	svc, id := newTestAdminService(t)
	host := adminSender(testRoom)

	f := newFollowStream(t, adminCtx(t, adminNonce(t, svc, id))).admin(svc, testRoom, nil)
	assert.Equal(t, f.next(t), "followed:/work/repo:0")
	say(t, svc.deliver, host, Target{}, "old-1")
	say(t, svc.deliver, host, Target{}, "old-2")
	assert.Equal(t, f.next(t), "message:1:/"+adminName+":old-1")
	assert.Equal(t, f.next(t), "message:2:/"+adminName+":old-2")

	_, err := svc.store.DeleteRoom(t.Context(), testRoom)
	assert.NilError(t, err)
	var regrown []Sent
	for i := range 3 {
		regrown = append(regrown, send(t, svc.store, host, Target{}, fmt.Sprintf("new-%d", i+1)))
	}
	announce(svc.store, regrown[0])
	announce(svc.store, regrown[2])

	assert.Equal(t, f.next(t), "followed:/work/repo:0")
	assert.Equal(t, f.next(t), "message:1:/"+adminName+":new-1")
	assert.Equal(t, f.next(t), "message:2:/"+adminName+":new-2")
	assert.Equal(t, f.next(t), "message:3:/"+adminName+":new-3")
}

// A follower that stops reading is dropped rather than served a conversation
// with holes in it, and the sends that dropped it were never held up. The answer
// is to follow again from the last seq it saw.
func TestFollow_SlowFollowerIsDropped(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouchNamed("tok-f", testRoom, "alpha", "fay")
	bob := senderOf(attend(t, svc.store, "tok-b", testRoom, "alpha", "bob"))

	stalled := newFollowStream(t, callCtx(t, "tok-f"))
	stalled.release = make(chan struct{})
	stalled.member(svc, nil)
	assert.Equal(t, stalled.next(t), "followed:/work/repo:0")

	// Two past the buffer: the stream holds at most one message in the Send it
	// is stuck in, and the buffer behind it fills either way.
	for i := range roomEventBuffer + 2 {
		say(t, svc.deliver, bob, Target{}, fmt.Sprintf("note-%d", i))
	}

	close(stalled.release)
	assert.Equal(t, status.Code(stalled.wait(t)), codes.ResourceExhausted)
}
