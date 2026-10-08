package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// forget puts a role in room and takes it out again, which leaves it a role the
// room can address with nobody attending under it.
func forget(t *testing.T, s *Store, room, team, name string) {
	t.Helper()
	attend(t, s, "tok-gone-"+team+"-"+name, room, team, name)
	_, err := s.Detach(t.Context(), "tok-gone-"+team+"-"+name)
	assert.NilError(t, err)
}

// The stream opens on the room as it stands, then carries what is said from
// then on.
func TestService_FollowOpensOnTheNewestSeqThenGoesLive(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouchNamed("tok-f", testRoom, "alpha", "fay")
	bob := senderOf(attend(t, svc.store, "tok-b", testRoom, "alpha", "bob"))
	for i := range 3 {
		say(t, svc.deliver, bob, Target{}, fmt.Sprintf("old-%d", i))
	}

	f := newFollowStream(t, callCtx(t, "tok-f")).member(svc, nil)
	assert.Equal(t, f.next(t), "followed:/work/repo:3")

	say(t, svc.deliver, bob, Target{}, "new-0")
	say(t, svc.deliver, bob, Target{}, "new-1")
	assert.Equal(t, f.next(t), "message:4:alpha/bob:new-0")
	assert.Equal(t, f.next(t), "message:5:alpha/bob:new-1")
}

// Following again from the last seq seen hands over what was said in between
// and then goes live, each message once.
func TestService_FollowSinceReplaysThenGoesLive(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouchNamed("tok-f", testRoom, "alpha", "fay")
	bob := senderOf(attend(t, svc.store, "tok-b", testRoom, "alpha", "bob"))
	for i := range 5 {
		say(t, svc.deliver, bob, Target{}, fmt.Sprintf("note-%d", i))
	}

	f := newFollowStream(t, callCtx(t, "tok-f")).member(svc, new(int64(3)))
	assert.Equal(t, f.next(t), "followed:/work/repo:5")
	assert.Equal(t, f.next(t), "message:4:alpha/bob:note-3")
	assert.Equal(t, f.next(t), "message:5:alpha/bob:note-4")

	say(t, svc.deliver, bob, Target{}, "live")
	assert.Equal(t, f.next(t), "message:6:alpha/bob:live")
}

// A since of zero is a seq like any other: it replays the room from its first
// message. A follower that first saw the room empty resumes after zero this way
// and misses nothing said while it was away.
func TestService_FollowSinceZeroReplaysFromTheFirstMessage(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouchNamed("tok-f", testRoom, "alpha", "fay")
	bob := senderOf(attend(t, svc.store, "tok-b", testRoom, "alpha", "bob"))
	for i := range 2 {
		say(t, svc.deliver, bob, Target{}, fmt.Sprintf("note-%d", i))
	}

	f := newFollowStream(t, callCtx(t, "tok-f")).member(svc, new(int64(0)))
	assert.Equal(t, f.next(t), "followed:/work/repo:2")
	assert.Equal(t, f.next(t), "message:1:alpha/bob:note-0")
	assert.Equal(t, f.next(t), "message:2:alpha/bob:note-1")

	say(t, svc.deliver, bob, Target{}, "live")
	assert.Equal(t, f.next(t), "message:3:alpha/bob:live")
}

// mentioned_you says whether a message is for the follower: addressed to it or
// to everyone, and not its own.
func TestService_FollowMarksWhatIsForTheFollower(t *testing.T) {
	svc, provider, _ := newTestService(t)
	forget(t, svc.store, testRoom, "alpha", "ana")
	provider.vouchNamed("tok-a", testRoom, "alpha", "ana")
	bob := senderOf(attend(t, svc.store, "tok-b", testRoom, "alpha", "bob"))
	attend(t, svc.store, "tok-c", testRoom, "alpha", "carol")
	ana := Sender{Name: "ana", Team: "alpha", Room: testRoom}

	f := newFollowStream(t, callCtx(t, "tok-a")).member(svc, nil)
	assert.Equal(t, f.next(t), "followed:/work/repo:0")

	say(t, svc.deliver, bob, Target{Kind: TargetEveryone}, "all")
	say(t, svc.deliver, bob, toRoles(role("alpha", "ana")), "you")
	say(t, svc.deliver, bob, toRoles(role("alpha", "carol")), "her")
	say(t, svc.deliver, bob, Target{}, "post")
	say(t, svc.deliver, ana, Target{Kind: TargetEveryone}, "mine")

	assert.Equal(t, f.next(t), "message:1:alpha/bob:all (mentioned)")
	assert.Equal(t, f.next(t), "message:2:alpha/bob:you (mentioned)")
	assert.Equal(t, f.next(t), "message:3:alpha/bob:her")
	assert.Equal(t, f.next(t), "message:4:alpha/bob:post")
	assert.Equal(t, f.next(t), "message:5:alpha/ana:mine")
}

// The replayed stretch is marked the same way as the live one: the flag is
// about who a message is for, not about when the stream saw it.
func TestService_FollowMarksTheReplayToo(t *testing.T) {
	svc, provider, _ := newTestService(t)
	forget(t, svc.store, testRoom, "alpha", "ana")
	provider.vouchNamed("tok-a", testRoom, "alpha", "ana")
	bob := senderOf(attend(t, svc.store, "tok-b", testRoom, "alpha", "bob"))
	say(t, svc.deliver, bob, Target{}, "seen")
	say(t, svc.deliver, bob, toRoles(role("alpha", "ana")), "you")
	say(t, svc.deliver, bob, Target{}, "post")

	f := newFollowStream(t, callCtx(t, "tok-a")).member(svc, new(int64(1)))
	assert.Equal(t, f.next(t), "followed:/work/repo:3")
	assert.Equal(t, f.next(t), "message:2:alpha/bob:you (mentioned)")
	assert.Equal(t, f.next(t), "message:3:alpha/bob:post")
}

// Following is not attending: it puts nobody in the room, announces nothing,
// and moves no read position, so it takes nothing from the agent beside it. A
// client going away ends it with the cancellation and announces nothing either.
func TestService_FollowIsNotAttendance(t *testing.T) {
	svc, provider, _ := newTestService(t)
	olga := agent(t, svc, provider, "tok-o", testRoom, "alpha", "olga")
	forget(t, svc.store, testRoom, "alpha", "ana")
	provider.vouchNamed("tok-a", testRoom, "alpha", "ana")
	ana := Sender{Name: "ana", Team: "alpha", Room: testRoom}
	before := readPosition(t, svc.store, testRoom, "alpha", "ana")

	f := newFollowStream(t, callCtx(t, "tok-a")).member(svc, nil)
	assert.Equal(t, f.next(t), "followed:/work/repo:0")
	noMoreEvents(t, olga.sent)

	_, err := svc.Send(callCtx(t, "tok-o"), &chatv1.SendRequest{Target: to("ana"), Text: "ping"})
	assert.NilError(t, err)
	assert.Equal(t, f.next(t), "message:1:alpha/olga:ping (mentioned)")
	assert.Equal(t, describeEvent(nextEvent(t, olga.sent)), "message:alpha/olga:ping")

	_, err = svc.store.Member(t.Context(), "tok-a")
	assert.ErrorIs(t, err, ErrNotAttending)
	members, err := svc.store.ListMembers(t.Context(), testRoom)
	assert.NilError(t, err)
	assert.Equal(t, len(members), 1)
	assert.Equal(t, readPosition(t, svc.store, testRoom, "alpha", "ana"), before)
	unread, err := svc.store.CountUnread(t.Context(), ana)
	assert.NilError(t, err)
	assert.Equal(t, unread, 1)

	assert.Assert(t, errors.Is(f.close(t), context.Canceled))
	noMoreEvents(t, olga.sent)
}

// A token that never attended is followed without a read position being made
// for it.
func TestService_FollowSeedsNoReadPosition(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouchNamed("tok-f", testRoom, "alpha", "fay")

	f := newFollowStream(t, callCtx(t, "tok-f")).member(svc, nil)
	assert.Equal(t, f.next(t), "followed:/work/repo:0")
	assert.Equal(t, countRows(t, svc.store,
		`SELECT COUNT(*) FROM read_positions WHERE room = ? AND team = ? AND name = ?`,
		testRoom, "alpha", "fay"), 0)
}

func TestService_FollowRefuses(t *testing.T) {
	t.Run("a call carrying no token", func(t *testing.T) {
		svc, _, _ := newTestService(t)
		f := newFollowStream(t, t.Context()).member(svc, nil)
		assert.Equal(t, status.Code(f.wait(t)), codes.Unauthenticated)
	})

	t.Run("a token nothing places", func(t *testing.T) {
		svc, _, _ := newTestService(t)
		f := newFollowStream(t, callCtx(t, "tok-nobody")).member(svc, nil)
		assert.Equal(t, status.Code(f.wait(t)), codes.Unauthenticated)
	})

	t.Run("a provider that could not be asked", func(t *testing.T) {
		svc, provider, _ := newTestService(t)
		provider.err = errors.New("cmdman: connection refused")
		f := newFollowStream(t, callCtx(t, "tok-a")).member(svc, nil)
		err := f.wait(t)
		assert.Equal(t, status.Code(err), codes.Unavailable)
		assert.Assert(t,
			strings.Contains(status.Convert(err).Message(), ProviderUnavailableMessage))
	})

	t.Run("a since below zero", func(t *testing.T) {
		svc, provider, _ := newTestService(t)
		provider.vouchNamed("tok-f", testRoom, "alpha", "fay")
		f := newFollowStream(t, callCtx(t, "tok-f")).member(svc, new(int64(-1)))
		assert.Equal(t, status.Code(f.wait(t)), codes.InvalidArgument)
	})
}

// TestService_FollowOverGRPC exercises the daemon's wiring: the stream
// interceptor lifts the token off the call for Follow as it does for Attend.
func TestService_FollowOverGRPC(t *testing.T) {
	svc, provider, _ := newTestService(t)
	forget(t, svc.store, testRoom, "alpha", "fay")
	provider.vouchNamed("tok-f", testRoom, "alpha", "fay")
	bob := senderOf(attend(t, svc.store, "tok-b", testRoom, "alpha", "bob"))
	client := dialTestService(t, svc)

	stream, err := client.Follow(
		metadata.AppendToOutgoingContext(t.Context(), TokenMetadataKey, "tok-f"),
		&chatv1.FollowRequest{})
	assert.NilError(t, err)
	ev, err := stream.Recv()
	assert.NilError(t, err)
	assert.Equal(t, describeFollowEvent(ev), "followed:/work/repo:0")

	say(t, svc.deliver, bob, toRoles(role("alpha", "fay")), "ping")
	ev, err = stream.Recv()
	assert.NilError(t, err)
	assert.Equal(t, describeFollowEvent(ev), "message:1:alpha/bob:ping (mentioned)")

	// A call carrying no token never reaches the service.
	stream, err = client.Follow(t.Context(), &chatv1.FollowRequest{})
	assert.NilError(t, err)
	_, err = stream.Recv()
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

// The follower is the role its token stands for: the one it attends under when
// it attends, else the one an attendance under the token would get.
func TestService_FollowNamesTheFollowerTheWayAttendWould(t *testing.T) {
	for _, tc := range []struct {
		name   string
		setup  func(t *testing.T, svc *Service, provider *fakeProvider)
		follow string
	}{
		{
			name: "the role an attendance under the token holds",
			setup: func(t *testing.T, svc *Service, provider *fakeProvider) {
				provider.vouchNamed("tok-long-enough", testRoom, "alpha", "derived")
				attend(t, svc.store, "tok-long-enough", testRoom, "alpha", "asked")
			},
			follow: "alpha/asked",
		},
		{
			name: "a person the operator registered, whom no provider places",
			setup: func(t *testing.T, svc *Service, _ *fakeProvider) {
				attendHuman(t, svc.store, "tok-long-enough", testRoom, "hosts", "hana")
			},
			follow: "hosts/hana",
		},
		{
			name: "the name the provider derives",
			setup: func(t *testing.T, svc *Service, provider *fakeProvider) {
				provider.vouchNamed("tok-long-enough", testRoom, "alpha", "derived")
				forget(t, svc.store, testRoom, "alpha", "derived")
			},
			follow: "alpha/derived",
		},
		{
			name: "the agent name an unnamed token attends under",
			setup: func(t *testing.T, svc *Service, provider *fakeProvider) {
				provider.vouch("tok-long-enough", testRoom, "alpha")
				forget(t, svc.store, testRoom, "alpha", "agent-tok-long")
			},
			follow: "alpha/agent-tok-long",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, provider, _ := newTestService(t)
			tc.setup(t, svc, provider)
			bob := senderOf(attend(t, svc.store, "tok-b", testRoom, "alpha", "bob"))
			team, name, _ := strings.Cut(tc.follow, "/")

			f := newFollowStream(t, callCtx(t, "tok-long-enough")).member(svc, nil)
			assert.Equal(t, f.next(t), "followed:/work/repo:0")

			say(t, svc.deliver, bob, toRoles(role(team, name)), "you")
			assert.Equal(t, f.next(t), "message:1:alpha/bob:you (mentioned)")
		})
	}
}
