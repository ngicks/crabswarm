// Package auth proves that a caller holds the operator's age identity.
//
// age encrypts but does not sign, so the proof is "you could read what only
// you can read": the daemon hands out a nonce encrypted to the configured
// recipient and accepts it back in plaintext, once. This package owns the
// challenge itself — issuing, expiring and spending nonces, and the two
// halves of the encryption — while the transport that carries it decides what
// a refusal looks like on the wire.
package auth

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"maps"
	"sync"
	"time"

	"filippo.io/age"
)

// NonceTTL is how long an issued challenge stays acceptable. Long enough for a
// human-driven CLI to decrypt the nonce and make its call, short enough that a
// nonce leaked from a shell history or a process listing is already dead.
const NonceTTL = 2 * time.Minute

// maxOutstanding caps the challenges kept at once. Issuing answers anyone who
// reaches the socket — only the identity holder can read the reply, but anyone
// can ask — so the set needs a ceiling. Reaching it evicts the oldest
// challenge instead of refusing to issue: refusing would let a spammer pin the
// set full and lock the operator out, while eviction costs at most a retry,
// and the CLI uses a nonce milliseconds after asking for it.
const maxOutstanding = 128

// Nonces is the set of challenges currently outstanding. It is safe for
// concurrent use, and the zero value is not: use [NewNonces].
type Nonces struct {
	mu sync.Mutex
	// nonces maps each outstanding challenge to when it stops being accepted.
	nonces map[string]time.Time
}

// NewNonces returns an empty challenge set.
func NewNonces() *Nonces {
	return &Nonces{nonces: make(map[string]time.Time)}
}

// Issue mints a challenge, records it as outstanding, and returns it with the
// moment it stops being accepted. It sweeps the expired challenges and evicts
// the oldest survivor when the set is already at its ceiling.
//
// The nonce is [rand.Text] — 128 unguessable bits in an alphabet that survives
// a proto string field and a shell argument unescaped, which is where it
// travels next.
func (n *Nonces) Issue() (nonce string, expiresAt time.Time) {
	nonce = rand.Text()
	now := time.Now()
	expiresAt = now.Add(NonceTTL)

	n.mu.Lock()
	defer n.mu.Unlock()
	maps.DeleteFunc(n.nonces, func(_ string, exp time.Time) bool { return now.After(exp) })
	for len(n.nonces) >= maxOutstanding {
		var (
			oldest    string
			oldestExp time.Time
		)
		for candidate, exp := range n.nonces {
			if oldestExp.IsZero() || exp.Before(oldestExp) {
				oldest, oldestExp = candidate, exp
			}
		}
		delete(n.nonces, oldest)
	}
	n.nonces[nonce] = expiresAt
	return nonce, expiresAt
}

// Verify reports whether nonce was outstanding, consuming it either way, so a
// nonce works exactly once even within its TTL — a nonce seen by anything that
// can read the caller's arguments is spent by the time it is seen.
//
// Unknown, expired and empty are not told apart: which one it was is not
// information a caller who cannot decrypt a nonce should get.
func (n *Nonces) Verify(nonce string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	expiresAt, ok := n.nonces[nonce]
	delete(n.nonces, nonce)
	return ok && !time.Now().After(expiresAt)
}

// DecryptNonce reads the nonce out of an encrypted challenge with the
// operator's age identities, which [age.ParseIdentities] parses out of the
// identity file. It is the client half of the challenge, kept beside the
// server half so the two cannot drift apart on the payload format.
func DecryptNonce(payload []byte, identities ...age.Identity) (string, error) {
	r, err := age.Decrypt(bytes.NewReader(payload), identities...)
	if err != nil {
		return "", fmt.Errorf("decrypting admin nonce: %w", err)
	}
	nonce, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("decrypting admin nonce: %w", err)
	}
	return string(nonce), nil
}

// EncryptNonce encrypts nonce to recipient as a raw age file, not ASCII armor:
// the challenge travels as bytes on a binary transport, and armor exists for
// the channels that cannot carry arbitrary bytes.
func EncryptNonce(recipient age.Recipient, nonce string) ([]byte, error) {
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipient)
	if err != nil {
		return nil, fmt.Errorf("encrypting admin nonce: %w", err)
	}
	if _, err := io.WriteString(w, nonce); err != nil {
		return nil, fmt.Errorf("encrypting admin nonce: %w", err)
	}
	// Close writes the final chunk: the buffer read before it holds a
	// truncated file that no identity can open.
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("encrypting admin nonce: %w", err)
	}
	return buf.Bytes(), nil
}
