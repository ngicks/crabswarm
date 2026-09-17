package chat

import (
	"context"
	"net"
	"path/filepath"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	chatsvc "github.com/ngicks/crabswarm/crabswarm/chat"
)

func member(team, name, room string) *chatv1.Member {
	return &chatv1.Member{Team: team, Name: name, Room: room}
}

// rolesTarget is a message addressed to the given members, which is the shape a
// mention arrives in.
func rolesTarget(members ...*chatv1.Member) *chatv1.Target {
	roles := make([]*chatv1.MemberTarget, len(members))
	for i, m := range members {
		roles[i] = &chatv1.MemberTarget{Team: m.GetTeam(), Name: m.GetName()}
	}
	return &chatv1.Target{
		Target: &chatv1.Target_Roles{Roles: &chatv1.Roles{Roles: roles}},
	}
}

// fakeChatService answers the member RPCs with canned data and keeps the
// requests it received, so a test can assert what the tools put on the wire.
//
// It is the stub crabswarm/chat/cli's tests drive, with a mutex added: the
// server holds its attendance on a goroutine of its own while tool calls run,
// so two goroutines reach these fields at once. Copied rather than shared,
// because sharing would mean exporting a test double from non-test code.
type fakeChatService struct {
	chatv1.UnimplementedChatServiceServer

	self      *chatv1.Member
	mentioned []*chatv1.Member
	absent    []*chatv1.Member
	messages  []*chatv1.Message
	remaining int32
	members   []*chatv1.Member

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
	// flip it mid-session, which is how a daemon that refuses a caller until it
	// is ready is played.
	err     error
	attend  *chatv1.AttendRequest
	opens   int
	attends int
	send    *chatv1.SendRequest
	read    *chatv1.ReadRequest
	reads   int
}

// failure is the canned error as it stands, read under the lock so a test that
// flips it mid-session races with nothing.
func (f *fakeChatService) failure() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.err
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

func (f *fakeChatService) Send(
	_ context.Context, req *chatv1.SendRequest,
) (*chatv1.SendResponse, error) {
	f.mu.Lock()
	f.send = req
	f.mu.Unlock()
	if err := f.failure(); err != nil {
		return nil, err
	}
	return &chatv1.SendResponse{Mentioned: f.mentioned, Absent: f.absent}, nil
}

func (f *fakeChatService) Read(
	_ context.Context, req *chatv1.ReadRequest,
) (*chatv1.ReadResponse, error) {
	f.mu.Lock()
	f.read = req
	f.reads++
	f.mu.Unlock()
	if err := f.failure(); err != nil {
		return nil, err
	}
	return &chatv1.ReadResponse{Messages: f.messages, RemainingUnread: f.remaining}, nil
}

func (f *fakeChatService) ListMembers(
	_ context.Context, _ *chatv1.ListMembersRequest,
) (*chatv1.ListMembersResponse, error) {
	if err := f.failure(); err != nil {
		return nil, err
	}
	return &chatv1.ListMembersResponse{Members: f.members}, nil
}

func (f *fakeChatService) lastAttend() *chatv1.AttendRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.attend
}

// openCount is how many times attendance was asked for, refusals included.
func (f *fakeChatService) openCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.opens
}

// attendCount is how many of those were admitted, which is what pins the tools
// acting as a member rather than as somebody trying to become one.
func (f *fakeChatService) attendCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.attends
}

func (f *fakeChatService) lastSend() *chatv1.SendRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.send
}

func (f *fakeChatService) lastRead() *chatv1.ReadRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.read
}

func (f *fakeChatService) readCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reads
}

// serveTestDaemon starts the stub on a Unix socket behind the daemon's own
// token interceptors and returns the socket path. A real socket rather than a
// bufconn because the server takes a path and dials it itself, which is the
// half of startup worth exercising. Both interceptors, because attendance is a
// stream and a unary interceptor never sees one.
func serveTestDaemon(t *testing.T, svc chatv1.ChatServiceServer) string {
	t.Helper()

	sock := filepath.Join(t.TempDir(), "chat.sock")
	lis, err := net.Listen("unix", sock)
	assert.NilError(t, err)

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(chatsvc.UnaryTokenInterceptor()),
		grpc.ChainStreamInterceptor(chatsvc.StreamTokenInterceptor()),
	)
	chatv1.RegisterChatServiceServer(srv, svc)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	return sock
}
