package chat

import (
	"errors"
	"strings"
	"testing"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"
)

func TestService_JoinDerivesRoomAndTeamFromProvider(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouch("tok-a", "/work/repo", "alpha")

	res, err := svc.Join(callCtx(t, "tok-a"), &chatv1.JoinRequest{Name: "ana"})
	assert.NilError(t, err)
	assert.Equal(t, res.GetSelf().GetName(), "ana")
	assert.Equal(t, res.GetSelf().GetTeam(), "alpha")
	assert.Equal(t, res.GetSelf().GetRoom(), "/work/repo")

	// The member is in the store as a provider-originated agent.
	stored, err := svc.store.Member(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, stored.Kind, KindAgent)
	assert.Equal(t, stored.State, StateDone)
}

func TestService_JoinRejectsUnknownToken(t *testing.T) {
	svc, _, _ := newTestService(t)

	_, err := svc.Join(callCtx(t, "stranger"), &chatv1.JoinRequest{Name: "who"})
	assert.Equal(t, status.Code(err), codes.NotFound)

	_, err = svc.store.Member(t.Context(), "stranger")
	assert.ErrorIs(t, err, ErrNotFound)
}

// A lookup that merely failed must not read as "you do not belong here".
func TestService_JoinOnProviderFailureIsUnavailable(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.err = errors.New("cmdman: connection refused")

	_, err := svc.Join(callCtx(t, "tok-a"), &chatv1.JoinRequest{Name: "ana"})
	assert.Equal(t, status.Code(err), codes.Unavailable)
	// The wording is the CLI's only way to tell this apart from the Unavailable
	// gRPC reports when no daemon is listening at all.
	assert.Assert(t, strings.Contains(status.Convert(err).Message(),
		ProviderUnavailableMessage))
}

func TestService_JoinIsIdempotent(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouch("tok-a", "/work", "alpha")

	first, err := svc.Join(callCtx(t, "tok-a"), &chatv1.JoinRequest{Name: "ana"})
	assert.NilError(t, err)
	// A second join under another name keeps the attendance already declared.
	second, err := svc.Join(callCtx(t, "tok-a"), &chatv1.JoinRequest{Name: "renamed"})
	assert.NilError(t, err)
	assert.Equal(t, second.GetSelf().GetName(), first.GetSelf().GetName())
	assert.Equal(t, second.GetSelf().GetName(), "ana")
}

func TestService_JoinDefaultsNameToTokenPrefix(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouch("0123456789abcdef", "/work", "alpha")

	res, err := svc.Join(callCtx(t, "0123456789abcdef"), &chatv1.JoinRequest{})
	assert.NilError(t, err)
	assert.Equal(t, res.GetSelf().GetName(), "agent-01234567")
}

// A joiner that reported no name is better named by whatever the provider knows
// it as than by its own token.
func TestService_JoinDefaultsNameToProviderName(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouchNamed("0123456789abcdef", "/work", "alpha", "worker-1")

	res, err := svc.Join(callCtx(t, "0123456789abcdef"), &chatv1.JoinRequest{})
	assert.NilError(t, err)
	assert.Equal(t, res.GetSelf().GetName(), "worker-1")
}

func TestService_JoinPrefersRequestedNameOverProviderName(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouchNamed("tok-a", "/work", "alpha", "worker-1")

	res, err := svc.Join(callCtx(t, "tok-a"), &chatv1.JoinRequest{Name: "ana"})
	assert.NilError(t, err)
	assert.Equal(t, res.GetSelf().GetName(), "ana")
}

func TestService_JoinAnswersRegisteredHumanWithoutProvider(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.err = errors.New("cmdman: connection refused")
	_, err := svc.store.Join(t.Context(), Member{
		Token: "human-tok", Name: "hana", Team: "hosts", Room: "/work", Kind: KindHuman,
	})
	assert.NilError(t, err)

	res, err := svc.Join(callCtx(t, "human-tok"), &chatv1.JoinRequest{Name: "ignored"})
	assert.NilError(t, err)
	assert.Equal(t, res.GetSelf().GetName(), "hana")
	assert.Equal(t, provider.callCount(), 0)
}

func TestService_JoinRejectsNameTakenInTeam(t *testing.T) {
	svc, provider, _ := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")
	provider.vouch("tok-b", "/work", "alpha")

	_, err := svc.Join(callCtx(t, "tok-b"), &chatv1.JoinRequest{Name: "ana"})
	assert.Equal(t, status.Code(err), codes.AlreadyExists)

	// The provider still places the member carrying the name, so it keeps it:
	// only a name nobody is left to answer for is handed over.
	stored, err := svc.store.Member(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, stored.Name, "ana")
}

// A recreated command derives the name its predecessor left behind. Nothing
// else would ever free it, so the collision does: the predecessor is gone with
// the token the provider stopped knowing, and the newcomer takes the name.
func TestService_JoinReclaimsTheNameOfAGoneMember(t *testing.T) {
	svc, provider, _ := newTestService(t)
	// Seeded straight into the store: what the predecessor left behind is a
	// member row, not a declaration this service witnessed.
	join(t, svc.store, "tok-old", "/work", "alpha", "worker-1")
	provider.vouchNamed("tok-new", "/work", "alpha", "worker-1")

	res, err := svc.Join(callCtx(t, "tok-new"), &chatv1.JoinRequest{})
	assert.NilError(t, err)
	assert.Equal(t, res.GetSelf().GetName(), "worker-1")
	assert.Equal(t, res.GetSelf().GetTeam(), "alpha")

	// One member of that name, and it is the one that just joined.
	_, err = svc.store.Member(t.Context(), "tok-old")
	assert.ErrorIs(t, err, ErrNotFound)
	members, err := svc.store.ListMembers(t.Context(), "/work")
	assert.NilError(t, err)
	assert.Equal(t, len(members), 1)
	assert.Equal(t, members[0].Token, "tok-new")
}

// The holder was vouched for moments ago, by its own join, and that verdict is
// cached for the TTL. The collision asks the provider again regardless: a
// recreated replica arrives exactly while its predecessor's verdict is fresh.
func TestService_JoinReclaimLooksPastTheCachedVerdict(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouchNamed("tok-old", "/work", "alpha", "worker-1")
	_, err := svc.Join(callCtx(t, "tok-old"), &chatv1.JoinRequest{})
	assert.NilError(t, err)

	provider.forget("tok-old")
	provider.vouchNamed("tok-new", "/work", "alpha", "worker-1")

	res, err := svc.Join(callCtx(t, "tok-new"), &chatv1.JoinRequest{})
	assert.NilError(t, err)
	assert.Equal(t, res.GetSelf().GetName(), "worker-1")
	_, err = svc.store.Member(t.Context(), "tok-old")
	assert.ErrorIs(t, err, ErrNotFound)
}

// A human's token was minted by the daemon, so no provider can vouch for it and
// none is asked: their name is theirs until an operator says otherwise.
func TestService_JoinNeverReclaimsAHumanName(t *testing.T) {
	svc, provider, _ := newTestService(t)
	_, err := svc.store.Join(t.Context(), Member{
		Token: "human-tok", Name: "hana", Team: "alpha", Room: "/work", Kind: KindHuman,
	})
	assert.NilError(t, err)
	provider.vouch("tok-a", "/work", "alpha")

	_, err = svc.Join(callCtx(t, "tok-a"), &chatv1.JoinRequest{Name: "hana"})
	assert.Equal(t, status.Code(err), codes.AlreadyExists)

	stored, err := svc.store.Member(t.Context(), "human-tok")
	assert.NilError(t, err)
	assert.Equal(t, stored.Name, "hana")
	// The only lookup was the joiner's own admission.
	assert.Equal(t, provider.callCount(), 1)
}

// A cmdman that could not be asked says nothing about the holder, and nothing
// is not enough to take a name away from it.
func TestService_JoinKeepsAHolderTheProviderCouldNotJudge(t *testing.T) {
	svc, provider, _ := newTestService(t)
	join(t, svc.store, "tok-old", "/work", "alpha", "worker-1")
	provider.vouchNamed("tok-new", "/work", "alpha", "worker-1")
	provider.failLookup("tok-old", errors.New("cmdman: connection refused"))

	_, err := svc.Join(callCtx(t, "tok-new"), &chatv1.JoinRequest{})
	assert.Equal(t, status.Code(err), codes.AlreadyExists)

	stored, err := svc.store.Member(t.Context(), "tok-old")
	assert.NilError(t, err)
	assert.Equal(t, stored.Name, "worker-1")
	_, err = svc.store.Member(t.Context(), "tok-new")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_JoinRejectsNameWithTeamSeparator(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouch("tok-a", "/work", "alpha")

	// "/" separates team from name in an address, so it cannot be part of one.
	_, err := svc.Join(callCtx(t, "tok-a"), &chatv1.JoinRequest{Name: "beta/bob"})
	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestService_ListMembersIsRoomScoped(t *testing.T) {
	svc, provider, _ := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")
	seedAgent(t, svc, provider, "tok-b", "/work", "beta", "bob")
	seedAgent(t, svc, provider, "tok-c", "/elsewhere", "alpha", "cid")

	res, err := svc.ListMembers(callCtx(t, "tok-a"), &chatv1.ListMembersRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(res.GetMembers()), 2)
	assert.Equal(t, res.GetMembers()[0].GetName(), "ana")
	assert.Equal(t, res.GetMembers()[1].GetTeam(), "beta")
}

// The listing carries every member's harness state, so a caller holding the
// roster knows who can be interrupted without asking about them one at a time.
func TestService_ListMembersCarriesReportedState(t *testing.T) {
	svc, provider, _ := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")
	seedAgent(t, svc, provider, "tok-b", "/work", "alpha", "bob")

	_, err := svc.ReportState(callCtx(t, "tok-a"), &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_WAITING,
	})
	assert.NilError(t, err)

	res, err := svc.ListMembers(callCtx(t, "tok-a"), &chatv1.ListMembersRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(res.GetMembers()), 2)
	assert.Equal(t, res.GetMembers()[0].GetName(), "ana")
	assert.Equal(t, res.GetMembers()[0].GetState(),
		chatv1.HarnessState_HARNESS_STATE_WAITING)
	// A member whose harness has reported nothing yet is listed as done, which
	// is the state the store admits a joiner in: it has taken no turn, so
	// nothing about it is in the way.
	assert.Equal(t, res.GetMembers()[1].GetName(), "bob")
	assert.Equal(t, res.GetMembers()[1].GetState(),
		chatv1.HarnessState_HARNESS_STATE_DONE)
}

func TestService_LeaveWithdrawsAttendance(t *testing.T) {
	svc, provider, _ := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")

	_, err := svc.Leave(callCtx(t, "tok-a"), &chatv1.LeaveRequest{})
	assert.NilError(t, err)

	_, err = svc.store.Member(t.Context(), "tok-a")
	assert.ErrorIs(t, err, ErrNotFound)

	_, err = svc.ListMembers(callCtx(t, "tok-a"), &chatv1.ListMembersRequest{})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestService_ReportState(t *testing.T) {
	svc, provider, _ := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")

	_, err := svc.ReportState(callCtx(t, "tok-a"), &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_WAITING,
	})
	assert.NilError(t, err)
	stored, err := svc.store.Member(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, stored.State, StateWaiting)

	// An unfilled state must not silently mark the harness done.
	_, err = svc.ReportState(callCtx(t, "tok-a"), &chatv1.ReportStateRequest{})
	assert.Equal(t, status.Code(err), codes.InvalidArgument)
	stored, err = svc.store.Member(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, stored.State, StateWaiting)
}

// The state an operator sees on a command follows the state the store holds,
// so every RPC that changes one publishes the other.
func TestService_PublishesMemberStateOnJoinReportAndLeave(t *testing.T) {
	svc, provider, _, mirror := newTestServiceWithMirror(t)
	provider.vouch("tok-a", "/work", "alpha")

	_, err := svc.Join(callCtx(t, "tok-a"), &chatv1.JoinRequest{Name: "ana"})
	assert.NilError(t, err)

	_, err = svc.ReportState(callCtx(t, "tok-a"), &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_WORKING,
	})
	assert.NilError(t, err)

	_, err = svc.Leave(callCtx(t, "tok-a"), &chatv1.LeaveRequest{})
	assert.NilError(t, err)

	calls := mirror.calls()
	assert.Equal(t, len(calls), 3, "published: %v", calls)
	// A fresh member starts done, which is what a session that has not been
	// given work yet is.
	assert.Equal(t, calls[0].state, StateDone)
	assert.Equal(t, calls[0].member.Token, "tok-a")
	assert.Equal(t, calls[1].state, StateWorking)
	assert.Assert(t, calls[2].cleared)
	assert.Equal(t, calls[2].member.Team+"/"+calls[2].member.Name, "alpha/ana")
}

// Re-declared attendance republishes what the store already holds: the session
// starting again is often one whose display was reset under it.
func TestService_JoinAgainRepublishesTheStoredState(t *testing.T) {
	svc, provider, _, mirror := newTestServiceWithMirror(t)
	provider.vouch("tok-a", "/work", "alpha")

	_, err := svc.Join(callCtx(t, "tok-a"), &chatv1.JoinRequest{Name: "ana"})
	assert.NilError(t, err)
	_, err = svc.ReportState(callCtx(t, "tok-a"), &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_WAITING,
	})
	assert.NilError(t, err)

	_, err = svc.Join(callCtx(t, "tok-a"), &chatv1.JoinRequest{Name: "ana"})
	assert.NilError(t, err)

	calls := mirror.calls()
	assert.Equal(t, len(calls), 3, "published: %v", calls)
	assert.Equal(t, calls[2].state, StateWaiting)
}

// A reaped member's command is gone with the token the provider stopped
// knowing, so there is no status left to withdraw.
func TestService_ReapingPublishesNothing(t *testing.T) {
	svc, provider, _, mirror := newTestServiceWithMirror(t)
	provider.vouch("tok-a", "/work", "alpha")
	_, err := svc.Join(callCtx(t, "tok-a"), &chatv1.JoinRequest{Name: "ana"})
	assert.NilError(t, err)

	provider.forget("tok-a")
	svc.forgetVerified("tok-a")

	_, err = svc.ListMembers(callCtx(t, "tok-a"), &chatv1.ListMembersRequest{})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)

	// Only the join published; the reap added nothing.
	calls := mirror.calls()
	assert.Equal(t, len(calls), 1, "published: %v", calls)
	assert.Equal(t, calls[0].state, StateDone)
}

// A mirror that cannot publish never fails the RPC: the store is authoritative
// and a stale display costs only the display.
func TestService_PublishFailureDoesNotFailTheRPC(t *testing.T) {
	svc, provider, _, mirror := newTestServiceWithMirror(t)
	provider.vouch("tok-a", "/work", "alpha")
	mirror.err = errors.New("cmdman: command is not running")

	_, err := svc.Join(callCtx(t, "tok-a"), &chatv1.JoinRequest{Name: "ana"})
	assert.NilError(t, err)
	_, err = svc.ReportState(callCtx(t, "tok-a"), &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_WORKING,
	})
	assert.NilError(t, err)
	_, err = svc.Leave(callCtx(t, "tok-a"), &chatv1.LeaveRequest{})
	assert.NilError(t, err)
}
