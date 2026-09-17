package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// The stream is the attendance: it appears in the room as the stream opens and
// is gone as the stream closes, while the role it attended under stays behind
// with its read position.
func TestService_AttendInsertsOnOpenAndDeletesOnClose(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouch("tok-a", testRoom, "alpha")

	ana := attendStream(t, svc, "tok-a", "ana", chatv1.MemberKind_MEMBER_KIND_AGENT)
	assert.Equal(t, ana.self.GetRoom(), testRoom)
	assert.Equal(t, ana.self.GetTeam(), "alpha")
	assert.Equal(t, ana.self.GetKind(), chatv1.MemberKind_MEMBER_KIND_AGENT)

	stored, err := svc.store.Member(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, addressOf(stored), "alpha/ana")

	assert.Assert(t, errors.Is(ana.close(t), context.Canceled))

	_, err = svc.store.Member(t.Context(), "tok-a")
	assert.ErrorIs(t, err, ErrNotAttending)
	members, err := svc.store.ListMembers(t.Context(), testRoom)
	assert.NilError(t, err)
	assert.Equal(t, len(members), 0)

	// The role outlives the session that declared it: its read position is what
	// keeps the room's backlog from being handed to it all over again.
	assert.Equal(t, countRows(t, svc.store, `SELECT COUNT(*) FROM read_positions`), 1)
}

func TestService_AttendNamesTheAttendee(t *testing.T) {
	for _, tc := range []struct {
		name      string
		requested string
		derived   string
		want      string
	}{
		{"what the request asked for", "ana", "compose-1", "ana"},
		{"what the provider derived", "", "compose-1", "compose-1"},
		{"its kind and token, as a last resort", "", "", "agent-tok-long"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, provider, _ := newTestService(t)
			provider.vouchNamed("tok-long-enough", testRoom, "alpha", tc.derived)

			s := attendStream(t, svc, "tok-long-enough", tc.requested,
				chatv1.MemberKind_MEMBER_KIND_AGENT)
			assert.Equal(t, s.self.GetName(), tc.want)
		})
	}
}

// What the attendance declared about its harness reaches the room: the member
// it opened with says it, and so does every roster drawn afterwards. Both
// values stand for the life of the attendance, so a reader deciding whether to
// wait for an answer reads them off the roster rather than asking.
func TestService_AttendCarriesTheHarnessAndDeliveryOntoTheRoster(t *testing.T) {
	svc, provider, _ := newTestService(t)
	nina := nativeAgent(t, svc, provider, "tok-n", testRoom, "alpha", "nina")
	assert.Equal(t, nina.self.GetHarness(), chatv1.Harness_HARNESS_CLAUDE_CODE)
	assert.Equal(t, nina.self.GetNudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)

	// An agent that declared no delivery is typed at through its terminal, and
	// the roster says so rather than leaving the column blank.
	agent(t, svc, provider, "tok-s", testRoom, "alpha", "sam")
	human(t, svc, provider, "tok-h", testRoom, "alpha", "hana")

	res, err := svc.ListMembers(callCtx(t, "tok-n"), &chatv1.ListMembersRequest{})
	assert.NilError(t, err)
	got := map[string][2]string{}
	for _, m := range res.GetMembers() {
		got[address(m)] = [2]string{m.GetHarness().String(), m.GetNudge().String()}
	}
	assert.DeepEqual(t, got, map[string][2]string{
		"alpha/nina": {"HARNESS_CLAUDE_CODE", "NUDGE_DELIVERY_NATIVE"},
		"alpha/sam":  {"HARNESS_UNSPECIFIED", "NUDGE_DELIVERY_TERMINAL"},
		"alpha/hana": {"HARNESS_UNSPECIFIED", "NUDGE_DELIVERY_UNSPECIFIED"},
	})
}

// A value outside the schema is a client speaking one this daemon does not
// have, which is worth refusing rather than recording as something else.
func TestService_AttendRejectsAnUnknownHarnessOrDelivery(t *testing.T) {
	for _, tc := range []struct {
		name    string
		harness chatv1.Harness
		nudge   chatv1.NudgeDelivery
	}{
		{"harness", chatv1.Harness(99), chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL},
		{"delivery", chatv1.Harness_HARNESS_CODEX, chatv1.NudgeDelivery(99)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, provider, _ := newTestService(t)
			provider.vouch("tok-a", testRoom, "alpha")

			s := newSession(t, "tok-a")
			s.harness, s.nudge = tc.harness, tc.nudge
			s.run(svc, "ana", chatv1.MemberKind_MEMBER_KIND_AGENT)
			assert.Equal(t, status.Code(s.wait(t)), codes.InvalidArgument)

			_, err := svc.store.Member(t.Context(), "tok-a")
			assert.ErrorIs(t, err, ErrNotAttending)
		})
	}
}

// The daemon never guesses what attends: a harness and the shell a person types
// in look the same to the team-info provider, and nudging the wrong one types
// keystrokes into somebody's session.
func TestService_AttendRejectsAnUndeclaredKind(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouch("tok-a", testRoom, "alpha")

	s := startAttend(t, svc, "tok-a", "ana",
		chatv1.MemberKind_MEMBER_KIND_UNSPECIFIED)
	assert.Equal(t, status.Code(s.wait(t)), codes.InvalidArgument)

	_, err := svc.store.Member(t.Context(), "tok-a")
	assert.ErrorIs(t, err, ErrNotAttending)
}

func TestService_AttendRejectsAnUnknownToken(t *testing.T) {
	svc, _, _ := newTestService(t)

	// Nothing places the token, so there is nowhere to put its holder and no
	// reason to take the token for an identity.
	s := startAttend(t, svc, "tok-nobody", "ana",
		chatv1.MemberKind_MEMBER_KIND_AGENT)
	assert.Equal(t, status.Code(s.wait(t)), codes.Unauthenticated)
}

func TestService_AttendOnProviderFailureIsUnavailable(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.err = errors.New("cmdman: connection refused")

	// Turning a caller away because cmdman was busy would read as "you do not
	// belong here", which is not what happened.
	s := startAttend(t, svc, "tok-a", "ana", chatv1.MemberKind_MEMBER_KIND_AGENT)
	err := s.wait(t)
	assert.Equal(t, status.Code(err), codes.Unavailable)
	assert.Assert(t,
		strings.Contains(status.Convert(err).Message(), ProviderUnavailableMessage))
}

// One token is one session, and one role is one read position: a second stream
// for either is a client that lost track of the first.
func TestService_AttendRefusesASecondStream(t *testing.T) {
	svc, provider, _ := newTestService(t)
	agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")

	again := startAttend(t, svc, "tok-a", "ana", chatv1.MemberKind_MEMBER_KIND_AGENT)
	assert.Equal(t, status.Code(again.wait(t)), codes.AlreadyExists)

	// Another session of the same role, refused for the read position it would
	// have had to share.
	provider.vouch("tok-a2", testRoom, "alpha")
	other := startAttend(t, svc, "tok-a2", "ana", chatv1.MemberKind_MEMBER_KIND_AGENT)
	assert.Equal(t, status.Code(other.wait(t)), codes.AlreadyExists)

	// The first stream is untouched by either refusal.
	stored, err := svc.store.Member(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, stored.Token, "tok-a")
}

// The role is free again once the stream that held it closes, so a harness that
// restarts comes back under the name it left with.
func TestService_AttendAgainAfterTheStreamClosed(t *testing.T) {
	svc, provider, _ := newTestService(t)
	ana := agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")
	assert.Assert(t, ana.close(t) != nil)

	provider.vouch("tok-a2", testRoom, "alpha")
	back := attendStream(t, svc, "tok-a2", "ana", chatv1.MemberKind_MEMBER_KIND_AGENT)
	assert.Equal(t, back.self.GetName(), "ana")
}

func TestService_AttendStreamsWhatHappensInTheRoom(t *testing.T) {
	svc, provider, _ := newTestService(t)
	ana := agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")

	bob := agent(t, svc, provider, "tok-b", testRoom, "alpha", "bob")
	_, err := svc.ReportState(callCtx(t, "tok-b"), &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_WORKING,
	})
	assert.NilError(t, err)
	_, err = svc.Send(callCtx(t, "tok-b"), &chatv1.SendRequest{
		Target: everyone(), Text: "morning",
	})
	assert.NilError(t, err)
	assert.Assert(t, bob.close(t) != nil)

	assert.Equal(t, describeEvent(nextEvent(t, ana.sent)), "joined:alpha/bob")
	assert.Equal(t, describeEvent(nextEvent(t, ana.sent)),
		"state:alpha/bob:HARNESS_STATE_WORKING")
	assert.Equal(t, describeEvent(nextEvent(t, ana.sent)), "message:alpha/bob:morning")
	assert.Equal(t, describeEvent(nextEvent(t, ana.sent)), "left:alpha/bob")
}

// A room's news never leaves it: rooms are what members can see of each other.
func TestService_AttendIsRoomScoped(t *testing.T) {
	svc, provider, _ := newTestService(t)
	ana := agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")
	agent(t, svc, provider, "tok-far", "/work/elsewhere", "alpha", "stranger")

	noMoreEvents(t, ana.sent)
}

// An attendee that stops reading is dropped rather than served a feed with
// holes in it, and the mutation that dropped it was never held up.
func TestService_SlowAttendeeIsDropped(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouch("tok-a", testRoom, "alpha")

	stalled := newSession(t, "tok-a")
	stalled.release = make(chan struct{})
	stalled.run(svc, "ana", chatv1.MemberKind_MEMBER_KIND_AGENT)
	// Attending and announced by the time this is on the wire; everything below
	// is ordered behind it, and the feed is what the stream stops reading.
	assert.Equal(t, describeEvent(nextEvent(t, stalled.sent)), "attended:alpha/ana")

	// One event past the buffer, published by somebody the stalled stream is
	// not blocking: every send returns while it sits there.
	provider.vouch("tok-b", testRoom, "alpha")
	bob := attendStream(t, svc, "tok-b", "bob", chatv1.MemberKind_MEMBER_KIND_AGENT)
	for i := range roomEventBuffer + 1 {
		_, err := svc.Send(callCtx(t, "tok-b"), &chatv1.SendRequest{
			Target: everyone(), Text: fmt.Sprintf("note-%d", i),
		})
		assert.NilError(t, err)
	}
	assert.Equal(t, describeEvent(nextEvent(t, bob.sent)), "message:alpha/bob:note-0")

	close(stalled.release)
	assert.Equal(t, status.Code(stalled.wait(t)), codes.ResourceExhausted)

	// Dropped from the feed is dropped from the room: the stream was the
	// attendance.
	_, err := svc.store.Member(t.Context(), "tok-a")
	assert.ErrorIs(t, err, ErrNotAttending)
}

func TestService_ListMembersIsRoomScoped(t *testing.T) {
	svc, provider, _ := newTestService(t)
	agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")
	human(t, svc, provider, "tok-b", testRoom, "beta", "bob")
	agent(t, svc, provider, "tok-far", "/work/elsewhere", "alpha", "stranger")

	res, err := svc.ListMembers(callCtx(t, "tok-a"), &chatv1.ListMembersRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(res.GetMembers()), 2)
	assert.Equal(t, address(res.GetMembers()[0]), "alpha/ana")
	assert.Equal(t, address(res.GetMembers()[1]), "beta/bob")
	assert.Equal(t, res.GetMembers()[1].GetKind(), chatv1.MemberKind_MEMBER_KIND_HUMAN)
}

func TestService_ListMembersCarriesReportedState(t *testing.T) {
	svc, provider, _ := newTestService(t)
	agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")

	// Attendance starts done: the stream opens before the session has work.
	res, err := svc.ListMembers(callCtx(t, "tok-a"), &chatv1.ListMembersRequest{})
	assert.NilError(t, err)
	assert.Equal(t, res.GetMembers()[0].GetState(),
		chatv1.HarnessState_HARNESS_STATE_DONE)

	_, err = svc.ReportState(callCtx(t, "tok-a"), &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_WAITING,
	})
	assert.NilError(t, err)
	res, err = svc.ListMembers(callCtx(t, "tok-a"), &chatv1.ListMembersRequest{})
	assert.NilError(t, err)
	assert.Equal(t, res.GetMembers()[0].GetState(),
		chatv1.HarnessState_HARNESS_STATE_WAITING)
}

func TestService_ReportState(t *testing.T) {
	svc, provider, _ := newTestService(t)
	ana := agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")

	_, err := svc.ReportState(callCtx(t, "tok-a"), &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED,
	})
	assert.Equal(t, status.Code(err), codes.InvalidArgument)

	_, err = svc.ReportState(callCtx(t, "tok-a"), &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_WORKING,
	})
	assert.NilError(t, err)
	assert.Equal(t, describeEvent(nextEvent(t, ana.sent)),
		"state:alpha/ana:HARNESS_STATE_WORKING")

	// Hooks report working after every tool call. The state is stored again —
	// it carries when the harness was last seen in it — but the room is not
	// told a roster it already has.
	_, err = svc.ReportState(callCtx(t, "tok-a"), &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_WORKING,
	})
	assert.NilError(t, err)
	noMoreEvents(t, ana.sent)
}

func TestService_PublishesMemberStateOnAttendReportAndClose(t *testing.T) {
	svc, provider, _, mirror := newTestServiceWithMirror(t)
	provider.vouch("tok-a", testRoom, "alpha")

	ana := attendStream(t, svc, "tok-a", "ana", chatv1.MemberKind_MEMBER_KIND_AGENT)
	_, err := svc.ReportState(callCtx(t, "tok-a"), &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_WORKING,
	})
	assert.NilError(t, err)
	// Attend withdraws the attendance before it returns, so its return is the
	// barrier for everything the withdrawal did.
	assert.Assert(t, ana.close(t) != nil)

	calls := mirror.calls()
	assert.Equal(t, len(calls), 3)
	assert.Equal(t, calls[0].state, StateDone)
	assert.Equal(t, calls[1].state, StateWorking)
	assert.Assert(t, calls[2].cleared)
	assert.Equal(t, addressOf(calls[2].member), "alpha/ana")
}

// A watcher recording what it read off a terminal goes through the same path as
// a hook's report: the store keeps it, the display follows it, the room hears
// about the change and hears nothing about a repeat.
func TestService_RecordStateFollowsTheReportPath(t *testing.T) {
	svc, provider, _, mirror := newTestServiceWithMirror(t)
	provider.vouch("tok-a", testRoom, "alpha")

	ana := attendStream(t, svc, "tok-a", "ana", chatv1.MemberKind_MEMBER_KIND_AGENT)

	assert.NilError(t, svc.RecordState(t.Context(), "tok-a", StateWaiting))
	assert.Equal(t, describeEvent(nextEvent(t, ana.sent)),
		"state:alpha/ana:HARNESS_STATE_WAITING")

	assert.NilError(t, svc.RecordState(t.Context(), "tok-a", StateWaiting))
	noMoreEvents(t, ana.sent)

	m, err := svc.store.Member(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, m.State, StateWaiting)

	calls := mirror.calls()
	assert.Equal(t, len(calls), 3)
	assert.Equal(t, calls[2].state, StateWaiting)

	// A session that ended between the listing and the reading is refused rather
	// than recorded against nobody.
	assert.Assert(t, errors.Is(svc.RecordState(t.Context(), "tok-z", StateDone), ErrNotAttending))
	assert.Assert(t, ana.close(t) != nil)
}

// The store is authoritative by the time the mirror is asked, so a display that
// cannot be written costs an operator a stale screen and the member nothing.
func TestService_PublishFailureDoesNotFailTheRPC(t *testing.T) {
	svc, provider, _, mirror := newTestServiceWithMirror(t)
	mirror.err = errors.New("cmdman: no such command")
	provider.vouch("tok-a", testRoom, "alpha")

	ana := attendStream(t, svc, "tok-a", "ana", chatv1.MemberKind_MEMBER_KIND_AGENT)
	_, err := svc.ReportState(callCtx(t, "tok-a"), &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_DONE,
	})
	assert.NilError(t, err)
	assert.Assert(t, len(mirror.calls()) > 0)
	assert.Assert(t, ana.close(t) != nil)
}
