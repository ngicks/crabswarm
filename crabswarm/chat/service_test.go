package chat

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
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
}

func (p *fakeProvider) Resolve(_ context.Context, token string) (resolver.TeamInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
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

// nudged is who the notifier was asked to wake, by address, in order.
func (n *fakeNotifier) nudged() []string {
	var out []string
	for _, got := range n.notified() {
		out = append(out, got.recipient.Team+"/"+got.recipient.Name)
	}
	return out
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

// callCtx is a request context carrying token, as the interceptor would leave
// it for a service method.
func callCtx(t *testing.T, token string) context.Context {
	t.Helper()
	return ContextWithToken(t.Context(), token)
}

// addressOf renders a member the way the tests name one.
func addressOf(m Member) string { return m.Team + "/" + m.Name }

// everyone is the whole-room target.
func everyone() *chatv1.Target {
	return &chatv1.Target{Target: &chatv1.Target_Everyone{Everyone: &chatv1.Everyone{}}}
}

// to is the explicit-list target, each role written the way a caller writes
// one: "team/name", or a bare "name" for the daemon to resolve.
func to(addrs ...string) *chatv1.Target {
	roles := make([]*chatv1.MemberTarget, len(addrs))
	for i, a := range addrs {
		team, name, qualified := strings.Cut(a, "/")
		if !qualified {
			team, name = "", a
		}
		roles[i] = &chatv1.MemberTarget{Team: team, Name: name}
	}
	return &chatv1.Target{Target: &chatv1.Target_Roles{Roles: &chatv1.Roles{Roles: roles}}}
}

// waitDetached waits until token is attending nothing, for a stream closed by a
// client over a connection: the cancellation reaches the handler some moments
// after the client asked for it, so a call made straight away would race it.
// A stream closed by the handler returning needs no such wait — the withdrawal
// is done before Attend returns.
func waitDetached(t *testing.T, store *Store, token string) {
	t.Helper()
	deadline := time.Now().Add(eventTimeout)
	for {
		if _, err := store.Member(t.Context(), token); errors.Is(err, ErrNotAttending) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("token %q is still attending", token)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestService_MissingTokenIsUnauthenticated(t *testing.T) {
	svc, _, _ := newTestService(t)

	_, err := svc.Send(t.Context(), &chatv1.SendRequest{Text: "hi"})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)

	_, err = svc.ListMembers(t.Context(), &chatv1.ListMembersRequest{})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)

	// Attend establishes the attendance rather than requiring one, but it still
	// needs to know who is attending.
	stream := &session{ctx: t.Context(), sent: make(chan *chatv1.RoomEvent, 1)}
	err = svc.Attend(&chatv1.AttendRequest{
		Name: "ana", Kind: chatv1.MemberKind_MEMBER_KIND_AGENT,
	}, stream)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestService_NotAttendingIsUnauthenticated(t *testing.T) {
	svc, provider, _ := newTestService(t)
	// Known to the provider, but holding no stream: the provider says where the
	// token would go, not that anybody took it there.
	provider.vouch("tok-a", testRoom, "alpha")

	_, err := svc.Send(callCtx(t, "tok-a"), &chatv1.SendRequest{Text: "hi"})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

// A member RPC is answered for exactly as long as the stream that declared the
// attendance: once it closes, the token means nothing again.
func TestService_MemberRPCAfterTheStreamClosedIsUnauthenticated(t *testing.T) {
	svc, provider, _ := newTestService(t)
	ana := agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")

	_, err := svc.ListMembers(callCtx(t, "tok-a"), &chatv1.ListMembersRequest{})
	assert.NilError(t, err)

	assert.Assert(t, ana.close(t) != nil)

	_, err = svc.ListMembers(callCtx(t, "tok-a"), &chatv1.ListMembersRequest{})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
	_, err = svc.Send(callCtx(t, "tok-a"), &chatv1.SendRequest{Text: "hi"})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
	_, err = svc.Read(callCtx(t, "tok-a"), &chatv1.ReadRequest{})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
	_, err = svc.ReportState(callCtx(t, "tok-a"), &chatv1.ReportStateRequest{
		State: chatv1.HarnessState_HARNESS_STATE_DONE,
	})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}
