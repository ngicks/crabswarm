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
