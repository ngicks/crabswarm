package auth

import (
	"context"
	"testing"
	"time"

	"filippo.io/age"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"
)

// newTestAgeNonce returns an authenticator gated by two throwaway identities
// standing in for the operators' identity files.
func newTestAgeNonce(t *testing.T) (*AgeNonce, *age.X25519Identity, *age.X25519Identity) {
	t.Helper()
	first, err := age.GenerateX25519Identity()
	assert.NilError(t, err)
	second, err := age.GenerateX25519Identity()
	assert.NilError(t, err)
	a, err := NewAgeNonce(first.Recipient().String(), second.Recipient().String())
	assert.NilError(t, err)
	return a, first, second
}

// bearerCtx is the context an RPC sees when the caller sent credential.
func bearerCtx(t *testing.T, credential string) context.Context {
	t.Helper()
	return metadata.NewIncomingContext(t.Context(),
		metadata.Pairs(AuthorizationMetadataKey, "Bearer "+credential))
}

func TestNewAgeNonce_RecipientParsing(t *testing.T) {
	id, err := age.GenerateX25519Identity()
	assert.NilError(t, err)

	// A typo in the config is caught where the daemon starts, not at the first
	// admin call — and the offending key is named, not just the count.
	_, err = NewAgeNonce(id.Recipient().String(), "age1-not-a-key")
	assert.ErrorContains(t, err, "age1-not-a-key")

	// No recipients is not an authenticator that refuses everything; it is no
	// authenticator, which the caller has to notice and act on.
	_, err = NewAgeNonce()
	assert.ErrorContains(t, err, "no chat admin recipients")
}

func TestAgeNonce_ChallengeThenAuthenticate(t *testing.T) {
	a, first, second := newTestAgeNonce(t)

	before := time.Now()
	challenge, err := a.Challenge(t.Context())
	assert.NilError(t, err)
	assert.Assert(t, len(challenge.Payload) > 0)
	assert.Assert(t, challenge.ExpiresAt.After(before))
	assert.Assert(t, !challenge.ExpiresAt.After(time.Now().Add(NonceTTL)))

	nonce, err := DecryptNonce(challenge.Payload, first)
	assert.NilError(t, err)
	assert.NilError(t, a.Authenticate(bearerCtx(t, nonce)))

	// The second operator answers a challenge of their own with the same
	// standing: one challenge is readable by every configured identity.
	challenge, err = a.Challenge(t.Context())
	assert.NilError(t, err)
	nonce, err = DecryptNonce(challenge.Payload, second)
	assert.NilError(t, err)
	assert.NilError(t, a.Authenticate(bearerCtx(t, nonce)))
}

// Every way of not presenting a usable credential is the same refusal, so a
// caller who cannot decrypt a challenge learns nothing from which one it was.
func TestAgeNonce_AuthenticateRefusals(t *testing.T) {
	a, id, _ := newTestAgeNonce(t)

	spend := func(t *testing.T) string {
		t.Helper()
		challenge, err := a.Challenge(t.Context())
		assert.NilError(t, err)
		nonce, err := DecryptNonce(challenge.Payload, id)
		assert.NilError(t, err)
		return nonce
	}

	t.Run("no metadata at all", func(t *testing.T) {
		assertRefused(t, a.Authenticate(t.Context()))
	})

	t.Run("a header of another scheme", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(t.Context(),
			metadata.Pairs(AuthorizationMetadataKey, "Basic dXNlcjpwYXNz"))
		assertRefused(t, a.Authenticate(ctx))
	})

	t.Run("a made-up credential", func(t *testing.T) {
		assertRefused(t, a.Authenticate(bearerCtx(t, "guessed")))
	})

	t.Run("a credential already spent", func(t *testing.T) {
		nonce := spend(t)
		assert.NilError(t, a.Authenticate(bearerCtx(t, nonce)))
		assertRefused(t, a.Authenticate(bearerCtx(t, nonce)))
	})
}

func assertRefused(t *testing.T, err error) {
	t.Helper()
	assert.Equal(t, status.Code(err), codes.PermissionDenied)
}
