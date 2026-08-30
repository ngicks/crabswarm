package auth

import (
	"testing"

	"google.golang.org/grpc/metadata"
	"gotest.tools/v3/assert"
)

func TestBearerFromContext(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header string
		want   string
	}{
		{"a bearer credential", "Bearer nonce-abc123", "nonce-abc123"},
		// RFC 6750 makes the scheme case-insensitive, and clients disagree
		// about how they spell it.
		{"a lowercase scheme", "bearer nonce-abc123", "nonce-abc123"},
		{"a shouted scheme", "BEARER nonce-abc123", "nonce-abc123"},
		{"padding around the credential", "Bearer   nonce-abc123  ", "nonce-abc123"},
		{"another scheme", "Basic dXNlcjpwYXNz", ""},
		{"a scheme and nothing else", "Bearer", ""},
		{"an empty credential", "Bearer ", ""},
		{"a bare credential with no scheme", "nonce-abc123", ""},
		{"an empty header", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := metadata.NewIncomingContext(t.Context(),
				metadata.Pairs(AuthorizationMetadataKey, tc.header))

			got, ok := BearerFromContext(ctx)
			assert.Equal(t, got, tc.want)
			assert.Equal(t, ok, tc.want != "")
		})
	}
}

// A call that carries no metadata at all reads the same as one carrying no
// credential: there is nothing to authenticate either way.
func TestBearerFromContext_NoMetadata(t *testing.T) {
	got, ok := BearerFromContext(t.Context())
	assert.Equal(t, got, "")
	assert.Assert(t, !ok)
}

func TestContextWithBearer_RoundTrips(t *testing.T) {
	// The client puts it on the outgoing side; the server reads the incoming
	// one, which is what the transport turns it into.
	md, ok := metadata.FromOutgoingContext(
		ContextWithBearer(t.Context(), "nonce-abc123"))
	assert.Assert(t, ok)

	got, ok := BearerFromContext(metadata.NewIncomingContext(t.Context(), md))
	assert.Assert(t, ok)
	assert.Equal(t, got, "nonce-abc123")
}
