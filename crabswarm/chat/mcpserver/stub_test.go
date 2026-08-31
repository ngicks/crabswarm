package mcpserver

import (
	"context"
	"net"
	"path/filepath"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat"
)

func member(team, name, room string) *chatv1.Member {
	return &chatv1.Member{Team: team, Name: name, Room: room}
}

// fakeChatService answers the member RPCs with canned data and keeps the
// requests it received, so a test can assert what the bridge put on the wire.
//
// It is the stub crabswarm/chat/cli's tests drive, with a mutex added: the
// bridge attends from its startup goroutine while tool calls run, so two
// goroutines reach these fields at once. Copied rather than shared, because
// sharing would mean exporting a test double from non-test code.
type fakeChatService struct {
	chatv1.UnimplementedChatServiceServer

	// err, when set, fails every RPC — the daemon rejecting what the caller
	// asked for rather than being unreachable. It is set before the bridge
	// starts and never written again.
	err error

	self      *chatv1.Member
	recipient *chatv1.Member
	delivered int32
	messages  []*chatv1.Message
	members   []*chatv1.Member

	mu        sync.Mutex
	join      *chatv1.JoinRequest
	send      *chatv1.SendRequest
	broadcast *chatv1.BroadcastRequest
	reads     int
}

func (f *fakeChatService) Join(
	_ context.Context, req *chatv1.JoinRequest,
) (*chatv1.JoinResponse, error) {
	f.mu.Lock()
	f.join = req
	f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return &chatv1.JoinResponse{Self: f.self}, nil
}

func (f *fakeChatService) Send(
	_ context.Context, req *chatv1.SendRequest,
) (*chatv1.SendResponse, error) {
	f.mu.Lock()
	f.send = req
	f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return &chatv1.SendResponse{Recipient: f.recipient}, nil
}

func (f *fakeChatService) Broadcast(
	_ context.Context, req *chatv1.BroadcastRequest,
) (*chatv1.BroadcastResponse, error) {
	f.mu.Lock()
	f.broadcast = req
	f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return &chatv1.BroadcastResponse{DeliveredCount: f.delivered}, nil
}

func (f *fakeChatService) Read(
	_ context.Context, _ *chatv1.ReadRequest,
) (*chatv1.ReadResponse, error) {
	f.mu.Lock()
	f.reads++
	f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return &chatv1.ReadResponse{Messages: f.messages}, nil
}

func (f *fakeChatService) ListMembers(
	_ context.Context, _ *chatv1.ListMembersRequest,
) (*chatv1.ListMembersResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &chatv1.ListMembersResponse{Members: f.members}, nil
}

func (f *fakeChatService) lastJoin() *chatv1.JoinRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.join
}

func (f *fakeChatService) lastSend() *chatv1.SendRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.send
}

func (f *fakeChatService) lastBroadcast() *chatv1.BroadcastRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.broadcast
}

func (f *fakeChatService) readCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reads
}

// serveTestDaemon starts the stub on a Unix socket behind the daemon's own
// [chat.UnaryTokenInterceptor] and returns the socket path. A real socket
// rather than a bufconn because [New] takes a path and dials it itself, which
// is the half of startup worth exercising.
func serveTestDaemon(t *testing.T, svc chatv1.ChatServiceServer) string {
	t.Helper()

	sock := filepath.Join(t.TempDir(), "chat.sock")
	lis, err := net.Listen("unix", sock)
	assert.NilError(t, err)

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(chat.UnaryTokenInterceptor()))
	chatv1.RegisterChatServiceServer(srv, svc)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	return sock
}
