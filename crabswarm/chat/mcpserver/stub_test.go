package mcpserver

import (
	"context"
	"net"
	"path/filepath"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	// watchFailures is how many WatchRoom calls are refused before one is
	// served, which is how a test plays the daemon dropping a watcher that fell
	// behind. It is set before the bridge starts and never written again.
	watchFailures int
	// events is the room feed a served WatchRoom forwards. Unbuffered on
	// purpose: a test that handed over an event knows the stub took it, which
	// is the only synchronisation either side needs.
	events chan *chatv1.RoomEvent

	mu        sync.Mutex
	join      *chatv1.JoinRequest
	send      *chatv1.SendRequest
	broadcast *chatv1.BroadcastRequest
	reads     int
	watches   int
}

func (f *fakeChatService) WatchRoom(
	_ *chatv1.WatchRoomRequest,
	stream grpc.ServerStreamingServer[chatv1.RoomEvent],
) error {
	f.mu.Lock()
	f.watches++
	refuse := f.watches <= f.watchFailures
	f.mu.Unlock()
	if refuse {
		return status.Error(codes.ResourceExhausted, "watcher fell behind")
	}
	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev := <-f.events:
			if err := stream.Send(ev); err != nil {
				return err
			}
		}
	}
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

func (f *fakeChatService) watchCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.watches
}

// serveTestDaemon starts the stub on a Unix socket behind the daemon's own
// token interceptors and returns the socket path. A real socket rather than a
// bufconn because [New] takes a path and dials it itself, which is the half of
// startup worth exercising. Both interceptors, because the bridge watches the
// room over the streaming half as well.
func serveTestDaemon(t *testing.T, svc chatv1.ChatServiceServer) string {
	t.Helper()

	sock := filepath.Join(t.TempDir(), "chat.sock")
	lis, err := net.Listen("unix", sock)
	assert.NilError(t, err)

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(chat.UnaryTokenInterceptor()),
		grpc.ChainStreamInterceptor(chat.StreamTokenInterceptor()),
	)
	chatv1.RegisterChatServiceServer(srv, svc)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	return sock
}
