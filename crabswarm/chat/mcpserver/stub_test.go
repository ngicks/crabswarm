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

	self      *chatv1.Member
	recipient *chatv1.Member
	delivered int32
	messages  []*chatv1.Message
	entries   []*chatv1.HistoryEntry
	members   []*chatv1.Member

	// watchFailures is how many WatchRoom calls are refused before one is
	// served, which is how a test plays the daemon dropping a watcher that fell
	// behind. It is set before the bridge starts and never written again.
	watchFailures int
	// events is the room feed a served WatchRoom forwards. Unbuffered on
	// purpose: a test that handed over an event knows the stub took it, which
	// is the only synchronisation either side needs.
	events chan *chatv1.RoomEvent

	mu sync.Mutex
	// err, when set, fails every unary RPC — the daemon rejecting what the
	// caller asked for rather than being unreachable. It is guarded because a
	// test may flip it mid-session with [fakeChatService.setErr], which is how
	// a daemon that forgot a member it had admitted is played.
	err       error
	join      *chatv1.JoinRequest
	joins     int
	send      *chatv1.SendRequest
	broadcast *chatv1.BroadcastRequest
	history   *chatv1.HistoryRequest
	reads     int
	watches   int
}

// failure is the canned error as it stands, read under the lock so a test that
// flips it mid-session races with nothing.
func (f *fakeChatService) failure() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.err
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
	f.joins++
	f.mu.Unlock()
	if err := f.failure(); err != nil {
		return nil, err
	}
	return &chatv1.JoinResponse{Self: f.self}, nil
}

func (f *fakeChatService) Send(
	_ context.Context, req *chatv1.SendRequest,
) (*chatv1.SendResponse, error) {
	f.mu.Lock()
	f.send = req
	f.mu.Unlock()
	if err := f.failure(); err != nil {
		return nil, err
	}
	return &chatv1.SendResponse{Recipient: f.recipient}, nil
}

func (f *fakeChatService) Broadcast(
	_ context.Context, req *chatv1.BroadcastRequest,
) (*chatv1.BroadcastResponse, error) {
	f.mu.Lock()
	f.broadcast = req
	f.mu.Unlock()
	if err := f.failure(); err != nil {
		return nil, err
	}
	return &chatv1.BroadcastResponse{DeliveredCount: f.delivered}, nil
}

func (f *fakeChatService) Read(
	_ context.Context, _ *chatv1.ReadRequest,
) (*chatv1.ReadResponse, error) {
	f.mu.Lock()
	f.reads++
	f.mu.Unlock()
	if err := f.failure(); err != nil {
		return nil, err
	}
	return &chatv1.ReadResponse{Messages: f.messages}, nil
}

// History answers with the canned transcript whatever window was asked for.
// The daemon's own limit handling is [chat.Service]'s to get right; what the
// bridge owes is the request it sends, which [fakeChatService.lastHistory]
// keeps.
func (f *fakeChatService) History(
	_ context.Context, req *chatv1.HistoryRequest,
) (*chatv1.HistoryResponse, error) {
	f.mu.Lock()
	f.history = req
	f.mu.Unlock()
	if err := f.failure(); err != nil {
		return nil, err
	}
	return &chatv1.HistoryResponse{Entries: f.entries}, nil
}

func (f *fakeChatService) ListMembers(
	_ context.Context, _ *chatv1.ListMembersRequest,
) (*chatv1.ListMembersResponse, error) {
	if err := f.failure(); err != nil {
		return nil, err
	}
	return &chatv1.ListMembersResponse{Members: f.members}, nil
}

// setErr changes what every unary RPC answers with from here on, so a test can
// play a daemon that stops recognising a member it had already admitted — and
// then recognises it again once the bridge attends afresh.
func (f *fakeChatService) setErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func (f *fakeChatService) lastJoin() *chatv1.JoinRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.join
}

// joinCount is how many times attendance was declared, which is what pins the
// bridge asking once per session rather than once per call.
func (f *fakeChatService) joinCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.joins
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

func (f *fakeChatService) lastHistory() *chatv1.HistoryRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.history
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
