package chat

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

func TestUnaryTokenInterceptor(t *testing.T) {
	interceptor := UnaryTokenInterceptor()
	chatCall := &grpc.UnaryServerInfo{FullMethod: chatv1.ChatService_Send_FullMethodName}

	var seen string
	handler := func(ctx context.Context, _ any) (any, error) {
		seen, _ = ctx.Value(tokenContextKey{}).(string)
		return "handled", nil
	}

	t.Run("token reaches the handler", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(t.Context(),
			metadata.Pairs(TokenMetadataKey, "tok-a"))
		got, err := interceptor(ctx, nil, chatCall, handler)
		assert.NilError(t, err)
		assert.Equal(t, got, "handled")
		assert.Equal(t, seen, "tok-a")
	})

	t.Run("missing metadata is rejected", func(t *testing.T) {
		_, err := interceptor(t.Context(), nil, chatCall, handler)
		assert.Equal(t, status.Code(err), codes.Unauthenticated)
	})

	t.Run("empty token is rejected", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(t.Context(),
			metadata.Pairs(TokenMetadataKey, ""))
		_, err := interceptor(ctx, nil, chatCall, handler)
		assert.Equal(t, status.Code(err), codes.Unauthenticated)
	})

	t.Run("other services pass through untouched", func(t *testing.T) {
		seen = "unset"
		other := &grpc.UnaryServerInfo{
			FullMethod: "/ngicks.crabswarm.hook.v1.AuditService/ReportHookInputEvent",
		}
		got, err := interceptor(t.Context(), nil, other, handler)
		assert.NilError(t, err)
		assert.Equal(t, got, "handled")
		assert.Equal(t, seen, "")
	})

	t.Run("the admin service passes through untouched", func(t *testing.T) {
		// Its caller holds an age identity, not a token; the nonce in the
		// request is the credential.
		seen = "unset"
		admin := &grpc.UnaryServerInfo{
			FullMethod: chatv1.ChatAdminService_GetNonce_FullMethodName,
		}
		got, err := interceptor(t.Context(), nil, admin, handler)
		assert.NilError(t, err)
		assert.Equal(t, got, "handled")
		assert.Equal(t, seen, "")
	})
}

// fakeStream is a [grpc.ServerStream] carrying nothing but a context, which is
// all the stream interceptor touches.
type fakeStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s fakeStream) Context() context.Context { return s.ctx }

func TestStreamTokenInterceptor(t *testing.T) {
	interceptor := StreamTokenInterceptor()
	chatCall := &grpc.StreamServerInfo{
		FullMethod: chatv1.ChatService_Attend_FullMethodName,
	}

	var seen string
	handler := func(_ any, ss grpc.ServerStream) error {
		seen, _ = ss.Context().Value(tokenContextKey{}).(string)
		return nil
	}

	t.Run("token reaches the handler", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(t.Context(),
			metadata.Pairs(TokenMetadataKey, "tok-a"))
		assert.NilError(t, interceptor(nil, fakeStream{ctx: ctx}, chatCall, handler))
		assert.Equal(t, seen, "tok-a")
	})

	t.Run("missing metadata is rejected", func(t *testing.T) {
		err := interceptor(nil, fakeStream{ctx: t.Context()}, chatCall, handler)
		assert.Equal(t, status.Code(err), codes.Unauthenticated)
	})

	t.Run("other services pass through untouched", func(t *testing.T) {
		seen = "unset"
		other := &grpc.StreamServerInfo{
			FullMethod: "/ngicks.crabswarm.preview.v1.PreviewService/Watch",
		}
		assert.NilError(t, interceptor(nil, fakeStream{ctx: t.Context()}, other, handler))
		assert.Equal(t, seen, "")
	})
}

// dialTestService serves svc over an in-memory listener with the interceptors
// the daemon installs, and returns a client of it. Both interceptors: a stream
// carries no token without the streaming one, so an Attend test without it
// would exercise a server the daemon never runs.
func dialTestService(t *testing.T, svc *Service) chatv1.ChatServiceClient {
	t.Helper()
	lis := bufconn.Listen(1 << 16)
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(UnaryTokenInterceptor()),
		grpc.ChainStreamInterceptor(StreamTokenInterceptor()),
	)
	chatv1.RegisterChatServiceServer(srv, svc)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NilError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return chatv1.NewChatServiceClient(conn)
}

// attendOverGRPC opens an attendance stream for token and reads the Attended
// event that opens it, so the caller knows the attendance has landed. The
// cancel closes the stream the way a client going away does.
func attendOverGRPC(
	t *testing.T,
	client chatv1.ChatServiceClient,
	token, name string,
) (chatv1.ChatService_AttendClient, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(
		metadata.AppendToOutgoingContext(t.Context(), TokenMetadataKey, token))
	t.Cleanup(cancel)

	stream, err := client.Attend(ctx, &chatv1.AttendRequest{
		Name: name, Kind: chatv1.MemberKind_MEMBER_KIND_AGENT,
	})
	assert.NilError(t, err)
	ev, err := stream.Recv()
	assert.NilError(t, err)
	assert.Equal(t, describeEvent(ev), "attended:alpha/"+name)
	return stream, cancel
}

// TestService_OverGRPC exercises the wiring the daemon uses: request metadata
// through the interceptor into the service, over both a unary call and the
// attendance stream.
func TestService_OverGRPC(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouch("tok-a", testRoom, "alpha")
	provider.vouch("tok-b", testRoom, "alpha")

	client := dialTestService(t, svc)
	asBob := metadata.AppendToOutgoingContext(t.Context(), TokenMetadataKey, "tok-b")

	ana, _ := attendOverGRPC(t, client, "tok-a", "ana")
	_, stopBob := attendOverGRPC(t, client, "tok-b", "bob")

	_, err := client.Send(asBob, &chatv1.SendRequest{Target: to("ana"), Text: "ping"})
	assert.NilError(t, err)
	read, err := client.Read(
		metadata.AppendToOutgoingContext(t.Context(), TokenMetadataKey, "tok-a"),
		&chatv1.ReadRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(read.GetMessages()), 1)
	assert.Equal(t, read.GetMessages()[0].GetText(), "ping")

	// The feed carries what happened after ana's own arrival, which opens it.
	assert.Equal(t, describeEvent(recvEvent(t, ana)), "joined:alpha/ana")
	assert.Equal(t, describeEvent(recvEvent(t, ana)), "joined:alpha/bob")
	assert.Equal(t, describeEvent(recvEvent(t, ana)), "message:alpha/bob:ping")

	// Closing the stream is leaving, and the token means nothing again.
	stopBob()
	waitDetached(t, svc.store, "tok-b")
	_, err = client.Send(asBob, &chatv1.SendRequest{Target: to("ana"), Text: "again"})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)

	// A call carrying no token never reaches the service.
	_, err = client.ListMembers(t.Context(), &chatv1.ListMembersRequest{})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

// recvEvent is the next event of a stream, failing the test on a stream that
// ended instead of hanging the run on one that went quiet.
func recvEvent(t *testing.T, stream chatv1.ChatService_AttendClient) *chatv1.RoomEvent {
	t.Helper()
	ev, err := stream.Recv()
	assert.NilError(t, err)
	return ev
}
