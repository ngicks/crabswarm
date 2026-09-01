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
// every ChatService call: the session id the team-info provider reports for a
// command it knows — an agent harness and the shell a person types in alike —
// or the token `chat admin register` printed for someone attending from
// outside.
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
// else, so a server hosting one must install this interceptor and
// [StreamTokenInterceptor] beside it.
//
// Only ChatService methods are touched. The interceptor is installed on the
// whole gRPC server — grpc-go has no per-service interceptors — and the other
// services on the daemon socket carry no token, so every one of their calls
// would otherwise be rejected.
//
// ChatAdminService is one of those others, deliberately: an operator holds no
// member token, and [AdminService] authenticates them from the bearer
// credential in the call's authorization metadata instead. Requiring a token
// there would lock out the very caller the service exists for.
func UnaryTokenInterceptor() grpc.UnaryServerInterceptor {
	prefix := chatServicePrefix()
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if !strings.HasPrefix(info.FullMethod, prefix) {
			return handler(ctx, req)
		}
		token, err := tokenFromMetadata(ctx)
		if err != nil {
			return nil, err
		}
		return handler(ContextWithToken(ctx, token), req)
	}
}

// StreamTokenInterceptor is [UnaryTokenInterceptor] for the streaming half of
// ChatService. A unary interceptor never sees a stream, so without this one
// WatchRoom would reach the service with no token at all and refuse every
// caller.
func StreamTokenInterceptor() grpc.StreamServerInterceptor {
	prefix := chatServicePrefix()
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if !strings.HasPrefix(info.FullMethod, prefix) {
			return handler(srv, ss)
		}
		ctx := ss.Context()
		token, err := tokenFromMetadata(ctx)
		if err != nil {
			return err
		}
		return handler(srv, tokenStream{
			ServerStream: ss,
			ctx:          ContextWithToken(ctx, token),
		})
	}
}

// tokenStream carries the token-bearing context into a stream handler. A
// stream's context is read from the stream rather than passed in, so the only
// way to add to it is to hand the handler a stream that answers differently.
type tokenStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s tokenStream) Context() context.Context { return s.ctx }

// tokenFromMetadata reads the caller's token off an incoming call, refusing one
// that carries none.
func tokenFromMetadata(ctx context.Context) (string, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get(TokenMetadataKey)
	if len(values) == 0 || values[0] == "" {
		return "", status.Errorf(codes.Unauthenticated,
			"missing %q metadata", TokenMetadataKey)
	}
	return values[0], nil
}

// chatServicePrefix is what a ChatService method's full name starts with, which
// is how the interceptors tell the calls they gate from the ones they let past.
func chatServicePrefix() string {
	return "/" + chatv1.ChatService_ServiceDesc.ServiceName + "/"
}
