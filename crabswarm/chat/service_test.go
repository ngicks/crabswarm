package chat

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/resolver"
)

// fakeProvider resolves tokens from a table. A token missing from it is
// unknown — permanently unresolvable; err, when set, is returned for every
// token instead and stands in for a cmdman that could not be asked at all.
type fakeProvider struct {
	mu    sync.Mutex
	infos map[string]resolver.TeamInfo
	err   error
	calls int
}

func (p *fakeProvider) Resolve(_ context.Context, token string) (resolver.TeamInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	if p.err != nil {
		return resolver.TeamInfo{}, p.err
	}
	info, ok := p.infos[token]
	if !ok {
		return resolver.TeamInfo{}, fmt.Errorf("%w: %q", resolver.ErrUnknownToken, token)
	}
	return info, nil
}

// vouch makes the provider place token in room/team.
func (p *fakeProvider) vouch(token, room, team string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.infos[token] = resolver.TeamInfo{Room: room, Team: team}
}

// vouchNamed is vouch for a provider that also knows what to call the token's
// holder, the way the compose provider reads a name off the command's labels.
func (p *fakeProvider) vouchNamed(token, room, team, name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.infos[token] = resolver.TeamInfo{Room: room, Team: team, Name: name}
}

// forget makes the provider stop knowing token, the way cmdman stops knowing a
// command once it is gone.
func (p *fakeProvider) forget(token string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.infos, token)
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
	svc, provider, notifier, _ := newTestServiceWithMirror(t)
	return svc, provider, notifier
}

// newTestServiceWithMirror is [newTestService] with the recording status mirror
// handed back too, for the cases that assert on what the service published.
func newTestServiceWithMirror(
	t *testing.T,
) (*Service, *fakeProvider, *fakeNotifier, *fakeStatusMirror) {
	t.Helper()
	store, _ := newTestStore(t)
	provider := &fakeProvider{infos: map[string]resolver.TeamInfo{}}
	notifier := &fakeNotifier{}
	mirror := &fakeStatusMirror{}
	return NewService(store, provider, notifier, mirror, nil), provider, notifier, mirror
}

// published is one call the service made on its [StatusMirror]. A cleared
// member carries no state, which is what tells the two apart.
type published struct {
	member  Member
	state   MemberState
	cleared bool
}

// fakeStatusMirror records what the service published, in order.
type fakeStatusMirror struct {
	mu  sync.Mutex
	got []published
	err error
}

var _ StatusMirror = (*fakeStatusMirror)(nil)

func (m *fakeStatusMirror) Set(_ context.Context, member Member, state MemberState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.got = append(m.got, published{member: member, state: state})
	return m.err
}

func (m *fakeStatusMirror) Clear(_ context.Context, member Member) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.got = append(m.got, published{member: member, cleared: true})
	return m.err
}

func (m *fakeStatusMirror) calls() []published {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.got)
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

// expireVerified back-dates the provider's verdict on token, which is how a
// test reaches [providerCheckTTL] without waiting it out.
func expireVerified(t *testing.T, svc *Service, token string) {
	t.Helper()
	svc.mu.Lock()
	defer svc.mu.Unlock()
	svc.verified[token] = time.Now().Add(-providerCheckTTL - time.Second)
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

// Past the TTL the cached verdict no longer answers for anyone: the provider is
// asked again, and its current answer decides. An agent it has forgotten is
// reaped; one it still vouches for keeps its place.
func TestService_ProviderCheckIsRedoneAfterTTL(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouch("tok-a", "/work", "alpha")
	provider.vouch("tok-b", "/work", "alpha")

	_, err := svc.Join(callCtx(t, "tok-a"), &chatv1.JoinRequest{Name: "ana"})
	assert.NilError(t, err)
	_, err = svc.Join(callCtx(t, "tok-b"), &chatv1.JoinRequest{Name: "bob"})
	assert.NilError(t, err)
	assert.Equal(t, provider.callCount(), 2)

	expireVerified(t, svc, "tok-a")
	expireVerified(t, svc, "tok-b")
	provider.forget("tok-a")

	// ana's command is gone: the call is refused and the attendance goes with
	// it, rather than being answered from the verdict the join cached.
	_, err = svc.ListMembers(callCtx(t, "tok-a"), &chatv1.ListMembersRequest{})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
	assert.Equal(t, provider.callCount(), 3)
	_, err = svc.store.Member(t.Context(), "tok-a")
	assert.ErrorIs(t, err, ErrNotFound)

	// bob's is still running, so the re-check costs it nothing but the lookup.
	res, err := svc.ListMembers(callCtx(t, "tok-b"), &chatv1.ListMembersRequest{})
	assert.NilError(t, err)
	assert.Equal(t, provider.callCount(), 4)
	assert.Equal(t, len(res.GetMembers()), 1)
	assert.Equal(t, res.GetMembers()[0].GetName(), "bob")
}
