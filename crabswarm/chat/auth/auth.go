// Package auth proves that a caller may perform the host-side chat operations.
//
// It holds the ways a daemon can be configured to authenticate its operator.
// Today that is [AgeNonce], which proves possession of an age identity file:
// age encrypts but does not sign, so the proof is "you could read what only you
// can read" — the daemon issues a nonce encrypted to the configured recipient
// and takes the plaintext back as the caller's credential. A setup whose
// credentials are minted elsewhere needs no challenge step and says so.
//
// A credential always reaches the daemon the same way, as the bearer credential
// of the call's authorization metadata; see [BearerFromContext].
package auth

import "time"

// Challenge is something a caller must answer to authenticate, handed to them
// as opaque bytes. What is inside depends on who issued it — [AgeNonce] puts an
// encrypted nonce there — so a caller passes it to the matching client half
// rather than reading it.
type Challenge struct {
	// Payload is the challenge itself.
	Payload []byte
	// ExpiresAt is when answering it stops working.
	ExpiresAt time.Time
}
