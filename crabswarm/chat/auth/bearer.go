package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
)

// AuthorizationMetadataKey is the metadata key an admin credential travels in.
// It is the standard HTTP one rather than a crabswarm-specific header: a
// TLS-terminating reverse proxy, and every OIDC client that would sit behind
// one, already speak it, and a credential nobody has to be taught to send is
// one less thing between an operator and their daemon.
const AuthorizationMetadataKey = "authorization"

// bearerScheme is the credential type prefix of [AuthorizationMetadataKey],
// matched case-insensitively the way RFC 6750 requires.
const bearerScheme = "bearer"

// BearerFromContext returns the credential of the incoming call's
// "authorization: Bearer <credential>" metadata, and whether there was one.
//
// It reports only presence, never why a header was rejected: a caller that
// cannot present a credential learns nothing from being told which part of it
// was wrong.
func BearerFromContext(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	// gRPC lowercases incoming keys, so the lookup needs no folding of its own.
	for _, v := range md.Get(AuthorizationMetadataKey) {
		scheme, credential, found := strings.Cut(v, " ")
		if !found || !strings.EqualFold(scheme, bearerScheme) {
			continue
		}
		if credential = strings.TrimSpace(credential); credential != "" {
			return credential, true
		}
	}
	return "", false
}

// ContextWithBearer returns ctx carrying credential as the outgoing call's
// bearer credential. It is the client-side counterpart of
// [BearerFromContext].
func ContextWithBearer(ctx context.Context, credential string) context.Context {
	return metadata.AppendToOutgoingContext(ctx,
		AuthorizationMetadataKey, "Bearer "+credential)
}
