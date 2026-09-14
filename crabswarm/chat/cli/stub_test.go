package cli

import (
	"context"
	"net"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
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
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			d.recordToken,
			chat.UnaryTokenInterceptor(),
		),
		// Attend is a stream, and a unary interceptor never sees one: without
		// the streaming half installed the way the daemon installs it, a client
		// that forgot its token would be served here and refused in production.
		grpc.ChainStreamInterceptor(
			d.recordStreamToken,
			chat.StreamTokenInterceptor(),
		),
	)
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
	d.noteToken(ctx)
	return handler(ctx, req)
}

// recordStreamToken is [testDaemon.recordToken] for the streaming half.
func (d *testDaemon) recordStreamToken(
	srv any,
	ss grpc.ServerStream,
	_ *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	d.noteToken(ss.Context())
	return handler(srv, ss)
}

func (d *testDaemon) noteToken(ctx context.Context) {
	md, _ := metadata.FromIncomingContext(ctx)
	seen := ""
	if v := md.Get(chat.TokenMetadataKey); len(v) > 0 {
		seen = v[0]
	}
	d.mu.Lock()
	d.tokens = append(d.tokens, seen)
	d.mu.Unlock()
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
	mentioned []*chatv1.Member
	absent    []*chatv1.Member
	messages  []*chatv1.Message
	remaining int32
	members   []*chatv1.Member
	// events follow the opening one down an Attend stream, which then stays
	// open until its client goes away.
	events []*chatv1.RoomEvent
	// openWith replaces the Attended event a stream opens with, for the case
	// where the daemon says something else first.
	openWith *chatv1.RoomEvent

	send  *chatv1.SendRequest
	read  *chatv1.ReadRequest
	state *chatv1.ReportStateRequest

	// mu guards what the Attend handler records: that stream is still running
	// while the case that opened it makes its assertions.
	mu     sync.Mutex
	attend *chatv1.AttendRequest
}

func (f *fakeChatService) Attend(
	req *chatv1.AttendRequest,
	stream grpc.ServerStreamingServer[chatv1.RoomEvent],
) error {
	f.mu.Lock()
	f.attend = req
	f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	opening := f.openWith
	if opening == nil {
		opening = &chatv1.RoomEvent{
			Event: &chatv1.RoomEvent_Attended{Attended: &chatv1.Attended{Self: f.self}},
		}
	}
	if err := stream.Send(opening); err != nil {
		return err
	}
	for _, ev := range f.events {
		if err := stream.Send(ev); err != nil {
			return err
		}
	}
	<-stream.Context().Done()
	return stream.Context().Err()
}

// attendRequest is the request the open stream declared, read under the lock
// the handler writes it with.
func (f *fakeChatService) attendRequest() *chatv1.AttendRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.attend
}

func (f *fakeChatService) Send(
	_ context.Context, req *chatv1.SendRequest,
) (*chatv1.SendResponse, error) {
	f.send = req
	if f.err != nil {
		return nil, f.err
	}
	return &chatv1.SendResponse{Mentioned: f.mentioned, Absent: f.absent}, nil
}

func (f *fakeChatService) Read(
	_ context.Context, req *chatv1.ReadRequest,
) (*chatv1.ReadResponse, error) {
	f.read = req
	if f.err != nil {
		return nil, f.err
	}
	return &chatv1.ReadResponse{
		Messages:        f.messages,
		RemainingUnread: f.remaining,
	}, nil
}

func (f *fakeChatService) ListMembers(
	_ context.Context, _ *chatv1.ListMembersRequest,
) (*chatv1.ListMembersResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &chatv1.ListMembersResponse{Members: f.members}, nil
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
