package chat

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/auth"
)

// An operator follows as nobody, so nothing is marked as mentioning them, not
// even what is addressed to everyone, in the replay or live.
func TestAdminService_FollowMentionsNobody(t *testing.T) {
	svc, id := newTestAdminService(t)
	ana := senderOf(seedConversation(t, svc, 3))

	f := newFollowStream(t, adminCtx(t, adminNonce(t, svc, id))).admin(svc, testRoom, new(int64(1)))
	assert.Equal(t, f.next(t), "followed:/work/repo:3")
	assert.Equal(t, f.next(t), "message:2:alpha/ana:note-1")
	assert.Equal(t, f.next(t), "message:3:alpha/ana:note-2")

	say(t, svc.deliver, ana, Target{Kind: TargetEveryone}, "all")
	adminSend(t, svc, id, testRoom, everyone(), "from the host")
	assert.Equal(t, f.next(t), "message:4:alpha/ana:all")
	assert.Equal(t, f.next(t), "message:5:/"+adminName+":from the host")
}

// An operator may open the stream before anybody arrives, so a room nothing has
// been said in follows from zero rather than failing.
func TestAdminService_FollowARoomNobodyUsedYet(t *testing.T) {
	svc, id := newTestAdminService(t)

	f := newFollowStream(t, adminCtx(t, adminNonce(t, svc, id))).admin(svc, "/work/later", nil)
	assert.Equal(t, f.next(t), "followed:/work/later:0")

	ana := senderOf(attend(t, svc.store, "tok-a", "/work/later", "alpha", "ana"))
	say(t, svc.deliver, ana, Target{}, "first")
	assert.Equal(t, f.next(t), "message:1:alpha/ana:first")
}

// A since of zero replays an operator's stream from the room's first message,
// as it does a member's.
func TestAdminService_FollowSinceZeroReplaysFromTheFirstMessage(t *testing.T) {
	svc, id := newTestAdminService(t)
	seedConversation(t, svc, 2)

	ctx := adminCtx(t, adminNonce(t, svc, id))
	f := newFollowStream(t, ctx).admin(svc, testRoom, new(int64(0)))
	assert.Equal(t, f.next(t), "followed:/work/repo:2")
	assert.Equal(t, f.next(t), "message:1:alpha/ana:note-0")
	assert.Equal(t, f.next(t), "message:2:alpha/ana:note-1")
}

func TestAdminService_FollowRefuses(t *testing.T) {
	t.Run("a call carrying no credential", func(t *testing.T) {
		svc, _ := newTestAdminService(t)
		f := newFollowStream(t, t.Context()).admin(svc, testRoom, nil)
		assert.Equal(t, status.Code(f.wait(t)), codes.PermissionDenied)
	})

	t.Run("a daemon with no way to recognise its operator", func(t *testing.T) {
		store, _ := newTestStore(t)
		svc := NewAdminService(store, nil, nil, nil)
		f := newFollowStream(t, adminCtx(t, "anything")).admin(svc, testRoom, nil)
		assert.Equal(t, status.Code(f.wait(t)), codes.FailedPrecondition)
	})

	t.Run("an empty room", func(t *testing.T) {
		svc, id := newTestAdminService(t)
		f := newFollowStream(t, adminCtx(t, adminNonce(t, svc, id))).admin(svc, "", nil)
		assert.Equal(t, status.Code(f.wait(t)), codes.InvalidArgument)
	})

	t.Run("a since below zero", func(t *testing.T) {
		svc, id := newTestAdminService(t)
		ctx := adminCtx(t, adminNonce(t, svc, id))
		f := newFollowStream(t, ctx).admin(svc, testRoom, new(int64(-1)))
		assert.Equal(t, status.Code(f.wait(t)), codes.InvalidArgument)
	})
}

// TestAdminService_FollowOverGRPC exercises the daemon's wiring: the stream
// interceptor is installed on the whole server, and the admin stream passes it
// carrying no token, proven by the bearer credential alone.
func TestAdminService_FollowOverGRPC(t *testing.T) {
	svc, id := newTestAdminService(t)

	lis := bufconn.Listen(1 << 16)
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(UnaryTokenInterceptor()),
		grpc.ChainStreamInterceptor(StreamTokenInterceptor()),
	)
	chatv1.RegisterChatAdminServiceServer(srv, svc)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NilError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	client := chatv1.NewChatAdminServiceClient(conn)

	challenge, err := client.GetNonce(t.Context(), &chatv1.GetNonceRequest{})
	assert.NilError(t, err)
	nonce, err := auth.DecryptNonce(challenge.GetEncryptedNonce(), id)
	assert.NilError(t, err)

	stream, err := client.Follow(auth.ContextWithBearer(t.Context(), nonce),
		&chatv1.AdminFollowRequest{Room: testRoom})
	assert.NilError(t, err)
	ev, err := stream.Recv()
	assert.NilError(t, err)
	assert.Equal(t, describeFollowEvent(ev), "followed:/work/repo:0")

	ana := senderOf(attend(t, svc.store, "tok-a", testRoom, "alpha", "ana"))
	say(t, svc.deliver, ana, Target{Kind: TargetEveryone}, "hello")
	ev, err = stream.Recv()
	assert.NilError(t, err)
	assert.Equal(t, describeFollowEvent(ev), "message:1:alpha/ana:hello")
}
