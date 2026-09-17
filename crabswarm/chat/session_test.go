package chat

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// session is one open Attend stream: the events the service wrote down it, and
// the cancel that closes it the way a client going away does.
type session struct {
	grpc.ServerStream
	ctx     context.Context
	cancel  context.CancelFunc
	release chan struct{}
	sent    chan *chatv1.RoomEvent
	// done carries what Attend returned. A buffered channel rather than an
	// errgroup: there is one call to wait for, and waiting for it has to be
	// selectable against a timeout so a stream that never ends fails the test
	// instead of hanging it.
	done chan error
	self *chatv1.Member
	// handshaken records that the opening Attended event is through. Written by
	// the Attend goroutine alone, which is the only caller of Send.
	handshaken bool
	// harness and nudge are what the attendance declares about itself. Both are
	// left unspecified by every case but the ones about who gets typed at, since
	// that is what a caller declaring nothing sends.
	harness chatv1.Harness
	nudge   chatv1.NudgeDelivery
}

var _ grpc.ServerStreamingServer[chatv1.RoomEvent] = (*session)(nil)

func (s *session) Context() context.Context { return s.ctx }

func (s *session) Send(ev *chatv1.RoomEvent) error {
	// A stalled client is one that stopped reading the feed, not one that never
	// got its handshake, so the opening event goes through either way. That is
	// also what gives such a test a barrier: Attended is written after the
	// arrival is announced, so anything the test then does is ordered behind it.
	if s.release != nil && s.handshaken {
		<-s.release
	}
	s.handshaken = true
	s.sent <- ev
	return nil
}

// newSession builds the stream a test hands Attend, not yet running. A test
// that plays a stalled client fills in release before starting it.
func newSession(t *testing.T, token string) *session {
	t.Helper()
	ctx, cancel := context.WithCancel(callCtx(t, token))
	s := &session{
		ctx:    ctx,
		cancel: cancel,
		// Twice the drop threshold: a test may let a stalled stream drain the
		// whole buffer at once, and Send must not be what blocks then.
		sent: make(chan *chatv1.RoomEvent, 2*roomEventBuffer),
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

// run starts Attend on the stream and returns it, still running. What the
// session declares about its harness goes out with the request.
func (s *session) run(svc *Service, name string, kind chatv1.MemberKind) *session {
	go func() {
		s.done <- svc.Attend(&chatv1.AttendRequest{
			Name:    name,
			Kind:    kind,
			Harness: s.harness,
			Nudge:   s.nudge,
		}, s)
	}()
	return s
}

// startAttend opens an attendance stream and returns without waiting for it to
// get anywhere, for the cases that assert on how it is refused.
func startAttend(
	t *testing.T,
	svc *Service,
	token, name string,
	kind chatv1.MemberKind,
) *session {
	t.Helper()
	return newSession(t, token).run(svc, name, kind)
}

// attendStream opens an attendance stream and waits for the Attended event that
// opens it, so the caller can rely on the attendance having landed.
func attendStream(
	t *testing.T,
	svc *Service,
	token, name string,
	kind chatv1.MemberKind,
) *session {
	t.Helper()
	return awaitAttended(t, startAttend(t, svc, token, name, kind))
}

// awaitAttended waits for the two events that open an attendance, so what the
// caller does next runs against a room that already has the member.
func awaitAttended(t *testing.T, s *session) *session {
	t.Helper()
	ev := nextEvent(t, s.sent)
	self := ev.GetAttended().GetSelf()
	assert.Assert(t, self != nil, "the first event was %s", describeEvent(ev))
	s.self = self
	// Its own arrival follows on the feed: the attendance is announced to the
	// room, and this stream is one of the room's by then. Consumed here so
	// every case reads only the events it caused.
	assert.Equal(t, describeEvent(nextEvent(t, s.sent)), "joined:"+address(self))
	return s
}

// agent puts an agent in room/team through its own attendance stream, the way
// a harness reaches the daemon.
func agent(
	t *testing.T,
	svc *Service,
	provider *fakeProvider,
	token, room, team, name string,
) *session {
	t.Helper()
	provider.vouch(token, room, team)
	return attendStream(t, svc, token, name, chatv1.MemberKind_MEMBER_KIND_AGENT)
}

// nativeAgent is [agent] for one whose own MCP server delivers its mentions
// through the harness. It declares the harness too, since a server that can
// deliver a mention is one that read which client it is talking to.
func nativeAgent(
	t *testing.T,
	svc *Service,
	provider *fakeProvider,
	token, room, team, name string,
) *session {
	t.Helper()
	provider.vouch(token, room, team)
	s := newSession(t, token)
	s.harness = chatv1.Harness_HARNESS_CLAUDE_CODE
	s.nudge = chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE
	return awaitAttended(t, s.run(svc, name, chatv1.MemberKind_MEMBER_KIND_AGENT))
}

// human is [agent] for someone whose terminal is never typed into.
func human(
	t *testing.T,
	svc *Service,
	provider *fakeProvider,
	token, room, team, name string,
) *session {
	t.Helper()
	provider.vouch(token, room, team)
	return attendStream(t, svc, token, name, chatv1.MemberKind_MEMBER_KIND_HUMAN)
}

// close ends the stream the way a client going away does and returns what
// Attend returned. It is also the barrier for the withdrawal: the attendance is
// given up before Attend returns.
func (s *session) close(t *testing.T) error {
	t.Helper()
	s.cancel()
	return s.wait(t)
}

// wait returns what Attend returned, failing the test rather than hanging when
// the stream is still running.
func (s *session) wait(t *testing.T) error {
	t.Helper()
	select {
	case err := <-s.done:
		// Put back, so a cleanup that waits again is not the one that hangs.
		s.done <- err
		return err
	case <-time.After(eventTimeout):
		t.Fatal("the attendance stream did not return")
		return nil
	}
}
