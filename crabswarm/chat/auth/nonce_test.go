package auth

import (
	"testing"
	"time"

	"filippo.io/age"
	"gotest.tools/v3/assert"
)

func TestNonces_IssueThenVerify(t *testing.T) {
	n := NewNonces()

	before := time.Now()
	nonce, expiresAt := n.Issue()
	assert.Assert(t, nonce != "")
	assert.Assert(t, expiresAt.After(before))
	assert.Assert(t, !expiresAt.After(time.Now().Add(NonceTTL)))

	assert.Assert(t, n.Verify(nonce))
}

func TestNonces_VerifyRefusesWhatItShould(t *testing.T) {
	n := NewNonces()

	t.Run("a used nonce does not work twice", func(t *testing.T) {
		nonce, _ := n.Issue()
		assert.Assert(t, n.Verify(nonce))
		assert.Assert(t, !n.Verify(nonce))
	})

	t.Run("a made-up nonce is refused", func(t *testing.T) {
		assert.Assert(t, !n.Verify("guessed"))
	})

	t.Run("an empty nonce is refused", func(t *testing.T) {
		assert.Assert(t, !n.Verify(""))
	})

	t.Run("an expired nonce is refused", func(t *testing.T) {
		nonce, _ := n.Issue()
		// Stamping the entry is how a test reaches the TTL without waiting it
		// out.
		n.mu.Lock()
		n.nonces[nonce] = time.Now().Add(-time.Second)
		n.mu.Unlock()
		assert.Assert(t, !n.Verify(nonce))
	})
}

// An expired challenge is spent even though it was refused, so the set cannot
// be filled with nonces that no longer work.
func TestNonces_VerifyConsumesAnExpiredNonce(t *testing.T) {
	n := NewNonces()

	nonce, _ := n.Issue()
	n.mu.Lock()
	n.nonces[nonce] = time.Now().Add(-time.Second)
	n.mu.Unlock()

	assert.Assert(t, !n.Verify(nonce))
	n.mu.Lock()
	_, still := n.nonces[nonce]
	n.mu.Unlock()
	assert.Assert(t, !still)
}

func TestNonces_OutstandingAreBounded(t *testing.T) {
	n := NewNonces()

	for range maxOutstanding + 10 {
		n.Issue()
	}

	n.mu.Lock()
	outstanding := len(n.nonces)
	n.mu.Unlock()
	assert.Equal(t, outstanding, maxOutstanding)
}

func TestNonceRoundTrip(t *testing.T) {
	id, err := age.GenerateX25519Identity()
	assert.NilError(t, err)

	payload, err := EncryptNonce(id.Recipient(), "nonce-abc123")
	assert.NilError(t, err)
	assert.Assert(t, len(payload) > 0)

	got, err := DecryptNonce(payload, id)
	assert.NilError(t, err)
	assert.Equal(t, got, "nonce-abc123")
}

func TestDecryptNonce_RefusesWhatItCannotRead(t *testing.T) {
	id, err := age.GenerateX25519Identity()
	assert.NilError(t, err)
	other, err := age.GenerateX25519Identity()
	assert.NilError(t, err)

	payload, err := EncryptNonce(id.Recipient(), "nonce-abc123")
	assert.NilError(t, err)

	_, err = DecryptNonce(payload, other)
	assert.ErrorContains(t, err, "decrypting admin nonce")

	_, err = DecryptNonce([]byte("not an age file"), other)
	assert.ErrorContains(t, err, "decrypting admin nonce")
}
