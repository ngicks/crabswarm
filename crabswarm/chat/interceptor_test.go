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
}

// TestService_OverGRPC exercises the wiring the daemon uses: request metadata
// through the interceptor into the service.
func TestService_OverGRPC(t *testing.T) {
	svc, provider, _ := newTestService(t)
	provider.vouch("tok-a", "/work", "alpha")
	provider.vouch("tok-b", "/work", "alpha")

	lis := bufconn.Listen(1 << 16)
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(UnaryTokenInterceptor()))
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
	client := chatv1.NewChatServiceClient(conn)

	asAna := metadata.AppendToOutgoingContext(t.Context(), TokenMetadataKey, "tok-a")
	asBob := metadata.AppendToOutgoingContext(t.Context(), TokenMetadataKey, "tok-b")

	joined, err := client.Join(asAna, &chatv1.JoinRequest{Name: "ana"})
	assert.NilError(t, err)
	assert.Equal(t, joined.GetSelf().GetRoom(), "/work")
	_, err = client.Join(asBob, &chatv1.JoinRequest{Name: "bob"})
	assert.NilError(t, err)

	_, err = client.Send(asAna, &chatv1.SendRequest{To: "bob", Text: "ping"})
	assert.NilError(t, err)
	read, err := client.Read(asBob, &chatv1.ReadRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(read.GetMessages()), 1)
	assert.Equal(t, read.GetMessages()[0].GetText(), "ping")

	// A call carrying no token never reaches the service.
	_, err = client.Join(t.Context(), &chatv1.JoinRequest{Name: "nobody"})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}
