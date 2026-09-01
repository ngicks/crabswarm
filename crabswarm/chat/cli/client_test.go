package cli

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat"
)

// testDaemon is an in-process stand-in for the daemon: the services a test
// registers, behind the same token interceptor the real server installs, plus
// the credentials each call arrived with.
type testDaemon struct {
	client *Client

	mu     sync.Mutex
	tokens []string
}

// serveTestDaemon starts the in-process server and returns a client dialed to
// it. Either service may be nil when a test does not exercise that half.
//
// The daemon's own [chat.UnaryTokenInterceptor] is installed rather than a stub
// check, so a client that forgot to attach its token fails here exactly as it
// would against a real daemon.
func serveTestDaemon(
	t *testing.T,
	chatSvc chatv1.ChatServiceServer,
	adminSvc chatv1.ChatAdminServiceServer,
) *testDaemon {
	t.Helper()

	d := &testDaemon{}
	lis := bufconn.Listen(1 << 16)
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(
		d.recordToken,
		chat.UnaryTokenInterceptor(),
	))
	if chatSvc != nil {
		chatv1.RegisterChatServiceServer(srv, chatSvc)
	}
	if adminSvc != nil {
		chatv1.RegisterChatAdminServiceServer(srv, adminSvc)
	}
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NilError(t, err)

	d.client = newClient(conn)
	t.Cleanup(func() { _ = d.client.Close() })
	return d
}

// recordToken notes the identity metadata of every call, including the calls
// that carry none.
func (d *testDaemon) recordToken(
	ctx context.Context,
	req any,
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	seen := ""
	if v := md.Get(chat.TokenMetadataKey); len(v) > 0 {
		seen = v[0]
	}
	d.mu.Lock()
	d.tokens = append(d.tokens, seen)
	d.mu.Unlock()
	return handler(ctx, req)
}

func (d *testDaemon) seenTokens() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.tokens...)
}

// fakeChatService answers the member RPCs with canned data and keeps the
// requests it received, so a test can assert what the client put on the wire.
type fakeChatService struct {
	chatv1.UnimplementedChatServiceServer

	// err, when set, fails every RPC — the daemon rejecting what the caller
	// asked for rather than being unreachable.
	err error

	self      *chatv1.Member
	recipient *chatv1.Member
	delivered int32
	messages  []*chatv1.Message
	members   []*chatv1.Member
	entries   []*chatv1.HistoryEntry

	join       *chatv1.JoinRequest
	send       *chatv1.SendRequest
	broadcast  *chatv1.BroadcastRequest
	state      *chatv1.ReportStateRequest
	history    *chatv1.HistoryRequest
	leaveCalls int
}

func (f *fakeChatService) Join(
	_ context.Context, req *chatv1.JoinRequest,
) (*chatv1.JoinResponse, error) {
	f.join = req
	if f.err != nil {
		return nil, f.err
	}
	return &chatv1.JoinResponse{Self: f.self}, nil
}

func (f *fakeChatService) Send(
	_ context.Context, req *chatv1.SendRequest,
) (*chatv1.SendResponse, error) {
	f.send = req
	if f.err != nil {
		return nil, f.err
	}
	return &chatv1.SendResponse{Recipient: f.recipient}, nil
}

func (f *fakeChatService) Broadcast(
	_ context.Context, req *chatv1.BroadcastRequest,
) (*chatv1.BroadcastResponse, error) {
	f.broadcast = req
	if f.err != nil {
		return nil, f.err
	}
	return &chatv1.BroadcastResponse{DeliveredCount: f.delivered}, nil
}

func (f *fakeChatService) Read(
	_ context.Context, _ *chatv1.ReadRequest,
) (*chatv1.ReadResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &chatv1.ReadResponse{Messages: f.messages}, nil
}

func (f *fakeChatService) History(
	_ context.Context, req *chatv1.HistoryRequest,
) (*chatv1.HistoryResponse, error) {
	f.history = req
	if f.err != nil {
		return nil, f.err
	}
	return &chatv1.HistoryResponse{Entries: f.entries}, nil
}

func (f *fakeChatService) ListMembers(
	_ context.Context, _ *chatv1.ListMembersRequest,
) (*chatv1.ListMembersResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &chatv1.ListMembersResponse{Members: f.members}, nil
}

func (f *fakeChatService) Leave(
	_ context.Context, _ *chatv1.LeaveRequest,
) (*chatv1.LeaveResponse, error) {
	f.leaveCalls++
	if f.err != nil {
		return nil, f.err
	}
	return &chatv1.LeaveResponse{}, nil
}

func (f *fakeChatService) ReportState(
	_ context.Context, req *chatv1.ReportStateRequest,
) (*chatv1.ReportStateResponse, error) {
	f.state = req
	if f.err != nil {
		return nil, f.err
	}
	return &chatv1.ReportStateResponse{}, nil
}

// The token has to travel as request metadata: the daemon reads it there and
// nowhere else, so a context value would arrive as no token at all.
func TestClient_CarriesTokenAsMetadata(t *testing.T) {
	fake := &fakeChatService{self: member("backend", "alice", "/work/proj")}
	d := serveTestDaemon(t, fake, nil)

	var out strings.Builder
	assert.NilError(t, d.client.Join(t.Context(), &out, "tok-a", "alice", false))
	assert.Equal(t, out.String(), "joined /work/proj as backend/alice\n")
	assert.DeepEqual(t, d.seenTokens(), []string{"tok-a"})
}

// A caller with no token never gets past the interceptor, and the refusal is
// reported in the daemon's own words.
func TestClient_EmptyTokenIsRejected(t *testing.T) {
	d := serveTestDaemon(t, &fakeChatService{}, nil)

	err := d.client.Join(t.Context(), &strings.Builder{}, "", "alice", false)
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), chat.TokenMetadataKey))
	assert.Equal(t, status.Code(errors.Unwrap(err)), codes.Unauthenticated)
}

// An error the daemon returns is surfaced as the message it wrote, without the
// "rpc error: code = ..." envelope: an ambiguous address names the qualified
// form to retry with, and that sentence is the whole point of the error.
func TestClient_SurfacesServerMessageVerbatim(t *testing.T) {
	const msg = `"alice" is attended by teams "backend" and "frontend": address one as "team/name"`
	fake := &fakeChatService{err: status.Error(codes.InvalidArgument, msg)}
	d := serveTestDaemon(t, fake, nil)

	err := d.client.Send(t.Context(), &strings.Builder{}, "tok-a", "alice", "hi")
	assert.Assert(t, err != nil)
	assert.Equal(t, err.Error(), msg)
	assert.Equal(t, status.Code(errors.Unwrap(err)), codes.InvalidArgument)
}

// A running daemon answers Unavailable when its team-info provider could not be
// asked, and that is not the daemon being absent: telling the operator to start
// one would send them to restart what is already running. The answer reaches
// them as the daemon wrote it.
func TestClient_ProviderUnavailableIsNotTheDaemonBeingDown(t *testing.T) {
	const msg = chat.ProviderUnavailableMessage + ": cmdman: connection refused"
	fake := &fakeChatService{err: status.Error(codes.Unavailable, msg)}
	d := serveTestDaemon(t, fake, nil)

	err := d.client.Join(t.Context(), &strings.Builder{}, "tok-a", "alice", false)
	assert.Assert(t, err != nil)
	assert.Equal(t, err.Error(), msg)
	assert.Assert(t, !errors.Is(err, ErrDaemonUnreachable))
	assert.Equal(t, status.Code(errors.Unwrap(err)), codes.Unavailable)
}

// Nothing listening on the socket is a different failure from a refused
// request, and the CLI says how to fix it.
func TestClient_UnreachableDaemonHint(t *testing.T) {
	client, err := Dial(filepath.Join(t.TempDir(), "absent.sock"))
	assert.NilError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	err = client.Read(t.Context(), &strings.Builder{}, "tok-a", ReadOptions{})
	assert.ErrorIs(t, err, ErrDaemonUnreachable)
	assert.Assert(t, strings.Contains(err.Error(), "crabswarm serve"))
}

func TestDial_RejectsEmptySocketPath(t *testing.T) {
	_, err := Dial("")
	assert.Assert(t, err != nil)
}
