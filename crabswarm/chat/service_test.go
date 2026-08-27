package chat

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// fakeProvider resolves tokens from a table. A token missing from it is
// unknown — permanently unresolvable; err, when set, is returned for every
// token instead and stands in for a cmdman that could not be asked at all.
type fakeProvider struct {
	mu    sync.Mutex
	infos map[string]TeamInfo
	err   error
	calls int
}

func (p *fakeProvider) Resolve(_ context.Context, token string) (TeamInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	if p.err != nil {
		return TeamInfo{}, p.err
	}
	info, ok := p.infos[token]
	if !ok {
		return TeamInfo{}, fmt.Errorf("%w: %q", ErrUnknownToken, token)
	}
	return info, nil
}

// vouch makes the provider place token in room/team.
func (p *fakeProvider) vouch(token, room, team string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.infos[token] = TeamInfo{Room: room, Team: team}
}

func (p *fakeProvider) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

// notification is one recorded [Notifier] call.
type notification struct {
	recipient Member
	from      Sender
	text      string
}

type fakeNotifier struct {
	mu   sync.Mutex
	got  []notification
	err  error
	fail bool
}

func (n *fakeNotifier) Notify(_ context.Context, recipient Member, from Sender, text string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.got = append(n.got, notification{recipient: recipient, from: from, text: text})
	if n.fail {
		return n.err
	}
	return nil
}

func (n *fakeNotifier) notified() []notification {
	n.mu.Lock()
	defer n.mu.Unlock()
	return slices.Clone(n.got)
}

// newTestService wires a service over a fresh store, a table-driven provider
// and a recording notifier.
func newTestService(t *testing.T) (*Service, *fakeProvider, *fakeNotifier) {
	t.Helper()
	store, _ := newTestStore(t)
	provider := &fakeProvider{infos: map[string]TeamInfo{}}
	notifier := &fakeNotifier{}
	return NewService(store, provider, notifier, nil), provider, notifier
}

// seedAgent puts an agent in the store the way a past join left it, with the
// provider still vouching for its token.
func seedAgent(
	t *testing.T,
	svc *Service,
	provider *fakeProvider,
	token, room, team, name string,
) Member {
	t.Helper()
	provider.vouch(token, room, team)
	return join(t, svc.store, token, room, team, name)
}

// callCtx is a request context carrying token, as the interceptor would leave
// it for a service method.
func callCtx(t *testing.T, token string) context.Context {
	t.Helper()
	return ContextWithToken(t.Context(), token)
}

func TestService_MissingTokenIsUnauthenticated(t *testing.T) {
	svc, _, _ := newTestService(t)

	_, err := svc.Join(t.Context(), &chatv1.JoinRequest{Name: "ana"})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)

	_, err = svc.Send(t.Context(), &chatv1.SendRequest{To: "bob", Text: "hi"})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestService_NonMemberIsUnauthenticated(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouch("tok-a", "/work", "alpha")

	// A token the provider knows but that never joined is still not a member.
	_, err := svc.Send(callCtx(t, "tok-a"), &chatv1.SendRequest{To: "bob", Text: "hi"})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
	assert.Equal(t, provider.callCount(), 0)
}

func TestService_ReapsMemberTheProviderForgot(t *testing.T) {
	svc, _, _ := newTestService(t)
	// Seeded straight into the store: joining through the service would vouch
	// for the token and the reap check would be skipped for the TTL.
	join(t, svc.store, "gone", "/work", "alpha", "ghost")
	join(t, svc.store, "tok-b", "/work", "alpha", "bob")

	_, err := svc.Send(callCtx(t, "gone"), &chatv1.SendRequest{To: "bob", Text: "hi"})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)

	_, err = svc.store.Member(t.Context(), "gone")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestService_TransientProviderFailureKeepsMember(t *testing.T) {
	svc, provider, _ := newTestService(t)
	join(t, svc.store, "tok-a", "/work", "alpha", "ana")
	join(t, svc.store, "tok-b", "/work", "alpha", "bob")
	provider.err = errors.New("cmdman: connection refused")

	res, err := svc.Send(callCtx(t, "tok-a"), &chatv1.SendRequest{To: "bob", Text: "hi"})
	assert.NilError(t, err)
	assert.Equal(t, res.GetRecipient().GetName(), "bob")

	_, err = svc.store.Member(t.Context(), "tok-a")
	assert.NilError(t, err)
}

func TestService_HumanMemberIsNeverProviderChecked(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.err = errors.New("cmdman: connection refused")
	_, err := svc.store.Join(t.Context(), Member{
		Token: "human-tok", Name: "hana", Team: "hosts", Room: "/work", Kind: KindHuman,
	})
	assert.NilError(t, err)

	res, err := svc.ListMembers(callCtx(t, "human-tok"), &chatv1.ListMembersRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(res.GetMembers()), 1)
	assert.Equal(t, provider.callCount(), 0)
}

func TestService_ProviderCheckIsCachedAcrossCalls(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouch("tok-a", "/work", "alpha")

	_, err := svc.Join(callCtx(t, "tok-a"), &chatv1.JoinRequest{Name: "ana"})
	assert.NilError(t, err)
	assert.Equal(t, provider.callCount(), 1)

	// Within the TTL the following calls ride the join's verification.
	_, err = svc.ListMembers(callCtx(t, "tok-a"), &chatv1.ListMembersRequest{})
	assert.NilError(t, err)
	_, err = svc.Read(callCtx(t, "tok-a"), &chatv1.ReadRequest{})
	assert.NilError(t, err)
	assert.Equal(t, provider.callCount(), 1)
}
