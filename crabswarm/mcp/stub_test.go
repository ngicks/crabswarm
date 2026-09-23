package mcp

import (
	"context"
	"net"
	"path/filepath"
	"slices"
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
	// cancels counts the held attendances the reader let go of, which is the
	// only way a case sees the server end one from its own side.
	cancels int
	// unread is what CountUnread answers with, and counts how many times it was
	// asked. A test moves the first with [fakeChatService.setUnread] and reads
	// the second to know the server has asked once already, which is what makes
	// "it was told afterwards" an ordering a case can rely on.
	unread int32
	counts int
	// reported is every state the member said it was in, oldest first. The whole
	// trail rather than the last of it: what a harness feed says is a sequence,
	// and a member that went working and came back is not one that never moved.
	reported []chatv1.HarnessState
}

// ReportState records what the member says it is doing, the way the daemon does
// before publishing it to the room.
func (f *fakeChatService) ReportState(
	_ context.Context, req *chatv1.ReportStateRequest,
) (*chatv1.ReportStateResponse, error) {
	f.mu.Lock()
	err := f.err
	if err == nil {
		f.reported = append(f.reported, req.GetState())
	}
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return &chatv1.ReportStateResponse{}, nil
}

// reportedStates is what the member reported about itself, oldest first.
func (f *fakeChatService) reportedStates() []chatv1.HarnessState {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.reported)
}

// CountUnread answers with the canned count and moves nothing, the way the
// daemon's does: a caller may ask as often as it likes.
func (f *fakeChatService) CountUnread(
	_ context.Context, _ *chatv1.CountUnreadRequest,
) (*chatv1.CountUnreadResponse, error) {
	f.mu.Lock()
	f.counts++
	unread, err := f.unread, f.err
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return &chatv1.CountUnreadResponse{UnreadMentions: unread}, nil
}

// setUnread changes what the room has waiting for the caller from here on.
func (f *fakeChatService) setUnread(n int32) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unread = n
}

// countCount is how many times the server asked what was waiting.
func (f *fakeChatService) countCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.counts
}

// lastAttend is the attendance request as the server put it on the wire, which
// is where a case reads what the member declared about itself.
func (f *fakeChatService) lastAttend() *chatv1.AttendRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.attend
}

// Attend opens the attendance the way the daemon does: the Attended event
// naming the member, then the room's own feed — which carries this attendance's
// arrival as its first entry, since every attendee sees the same room.
//
// The state on that member is the daemon's answer rather than the caller's
// claim, and every attendance it admits — the first and each one after a stream
// it dropped — reports the member done, because the daemon records an
// attendance it has no state for as one whose turn is over.
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
			f.mu.Lock()
			f.cancels++
			f.mu.Unlock()
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

// cancelCount is how many held attendances the server let go of, which is what
// pins one it ended itself — the daemon is the one that learns a member has
// stopped attending.
func (f *fakeChatService) cancelCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cancels
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
