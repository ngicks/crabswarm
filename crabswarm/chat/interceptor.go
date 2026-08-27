package chat

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// TokenMetadataKey is the gRPC metadata key carrying the caller's identity on
// every ChatService call: the provider-reported session id for an agent, the
// daemon-issued token for an admin-registered human.
const TokenMetadataKey = "x-crabswarm-token"

// tokenContextKey types the context value the interceptor hands to the
// service, so nothing else in a request context can be mistaken for a token.
type tokenContextKey struct{}

// ContextWithToken returns ctx carrying token as the caller's identity. It is
// what [UnaryTokenInterceptor] does to a request context, exported for
// in-process callers and tests that invoke a [Service] method directly instead
// of going through a gRPC server.
func ContextWithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenContextKey{}, token)
}

// tokenFromContext reads the token [UnaryTokenInterceptor] put in ctx. A
// missing one is Unauthenticated rather than Internal: the service is reachable
// without the interceptor only through a misconfigured server, and failing
// every call closed is the safe direction.
func tokenFromContext(ctx context.Context) (string, error) {
	token, _ := ctx.Value(tokenContextKey{}).(string)
	if token == "" {
		return "", status.Errorf(codes.Unauthenticated,
			"missing %q metadata", TokenMetadataKey)
	}
	return token, nil
}

// UnaryTokenInterceptor lifts the [TokenMetadataKey] metadata of a ChatService
// call into its context and rejects a call that carries none with
// Unauthenticated. A [Service] reads the caller identity from there and nowhere
// else, so a server hosting one must install this interceptor.
//
// Only ChatService methods are touched. The interceptor is installed on the
// whole gRPC server — grpc-go has no per-service interceptors — and the other
// services on the daemon socket carry no token, so every one of their calls
// would otherwise be rejected.
func UnaryTokenInterceptor() grpc.UnaryServerInterceptor {
	prefix := "/" + chatv1.ChatService_ServiceDesc.ServiceName + "/"
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if !strings.HasPrefix(info.FullMethod, prefix) {
			return handler(ctx, req)
		}
		md, _ := metadata.FromIncomingContext(ctx)
		values := md.Get(TokenMetadataKey)
		if len(values) == 0 || values[0] == "" {
			return nil, status.Errorf(codes.Unauthenticated,
				"missing %q metadata", TokenMetadataKey)
		}
		return handler(ContextWithToken(ctx, values[0]), req)
	}
}
