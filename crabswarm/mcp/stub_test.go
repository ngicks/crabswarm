package mcp

import (
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

// fakeChatService answers the attendance RPC with canned data and keeps what it
// received, so a test can assert what the server put on the wire. The verbs a
// tool family calls are left unimplemented: what this package does on its own
// is attend.
//
// It is the stub crabswarm/chat/cli's tests drive, trimmed to attendance and
// with a mutex added: the server holds its attendance on a goroutine of its own
// while the session runs, so two goroutines reach these fields at once. Copied
// rather than shared, because sharing would mean exporting a test double from
// non-test code.
type fakeChatService struct {
	chatv1.UnimplementedChatServiceServer

	self *chatv1.Member

	// events is the room feed a held attendance forwards. Unbuffered on
	// purpose: a test that handed over an event knows the stub took it, which
	// is the only synchronisation either side needs.
	events chan *chatv1.RoomEvent
	// drops carries the error a held attendance ends with, which is how a test
	// plays a daemon that closed the stream while the server was reading it.
	// Unbuffered for the reason events is.
	drops chan error

	mu sync.Mutex
	// err, when set, fails every RPC — the daemon rejecting what the caller
	// asked for rather than being unreachable. It is guarded because a test may
	// flip it mid-session with [fakeChatService.setErr], which is how a daemon
	// that refuses a server until it is ready is played.
	err     error
	attend  *chatv1.AttendRequest
	opens   int
	attends int
}

// Attend opens the attendance the way the daemon does: the Attended event
// naming the member, then the room's own feed — which carries this attendance's
// arrival as its first entry, since every attendee sees the same room.
func (f *fakeChatService) Attend(
	req *chatv1.AttendRequest,
	stream grpc.ServerStreamingServer[chatv1.RoomEvent],
) error {
	f.mu.Lock()
	f.attend = req
	f.opens++
	err := f.err
	if err == nil {
		f.attends++
	}
	f.mu.Unlock()
	if err != nil {
		return err
	}
	opening := []*chatv1.RoomEvent{
		{Event: &chatv1.RoomEvent_Attended{Attended: &chatv1.Attended{Self: f.self}}},
		{Event: &chatv1.RoomEvent_MemberJoined{
			MemberJoined: &chatv1.MemberJoined{Member: f.self},
		}},
	}
	for _, ev := range opening {
		if err := stream.Send(ev); err != nil {
			return err
		}
	}
	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-f.drops:
			return err
		case ev := <-f.events:
			if err := stream.Send(ev); err != nil {
				return err
			}
		}
	}
}

// setErr changes what every RPC answers with from here on, so a test can play a
// daemon that refuses a server and then admits it.
func (f *fakeChatService) setErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

// openCount is how many times attendance was asked for, refusals included,
// which is what pins a server that keeps asking.
func (f *fakeChatService) openCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.opens
}

// attendCount is how many of those were admitted, which is what pins the server
// being a member rather than trying to become one.
func (f *fakeChatService) attendCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.attends
}

// serveTestDaemon starts the stub on a Unix socket behind the daemon's own
// token interceptors and returns the socket path. A real socket rather than a
// bufconn because [New] takes a path and dials it itself, which is the half of
// startup worth exercising. Both interceptors, because attendance is a stream
// and a unary interceptor never sees one.
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
