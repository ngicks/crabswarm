package auth

import (
	"context"
	"fmt"

	"filippo.io/age"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AgeNonce authenticates the holder of an age identity file by challenge and
// response. It issues a nonce encrypted to every configured recipient and
// accepts that nonce back, once, as the caller's credential.
//
// The identity files are the host operators', kept outside the mounts
// participants see. Only native age recipients are accepted, not the ssh keys
// the age CLI also understands: a daemon names recipient strings, and an ssh
// recipient would drag in its own key-format parsing for no gain here.
type AgeNonce struct {
	recipients []age.Recipient
	nonces     *Nonces
}

// NewAgeNonce returns an authenticator that challenges the holders of the
// identities matching recipients, the "age1..." public keys of the operators'
// identity files. Any one of those identities can answer a challenge.
//
// No recipient at all, or one that does not parse, is an error: a daemon with
// no operator key configured has no authenticator at all rather than one that
// refuses everything, so that a caller is told to configure a key instead of
// being told their credential was wrong.
func NewAgeNonce(recipients ...string) (*AgeNonce, error) {
	if len(recipients) == 0 {
		return nil, fmt.Errorf("no chat admin recipients")
	}
	parsed := make([]age.Recipient, len(recipients))
	for i, recipient := range recipients {
		r, err := age.ParseX25519Recipient(recipient)
		if err != nil {
			return nil, fmt.Errorf("parsing chat admin recipient %q: %w", recipient, err)
		}
		parsed[i] = r
	}
	return &AgeNonce{recipients: parsed, nonces: NewNonces()}, nil
}

// Challenge issues a nonce encrypted to every configured recipient. It proves
// nothing on its own: the challenge is readable only by the identity files, so
// handing it to a caller who cannot decrypt it tells them nothing they did not
// already know.
//
// The payload is a raw age file, not ASCII armor: the challenge travels as
// bytes on a binary transport, and armor exists for the channels that cannot
// carry arbitrary bytes.
func (a *AgeNonce) Challenge(context.Context) (Challenge, error) {
	nonce, expiresAt := a.nonces.Issue()
	payload, err := EncryptNonce(nonce, a.recipients...)
	if err != nil {
		return Challenge{}, err
	}
	return Challenge{Payload: payload, ExpiresAt: expiresAt}, nil
}

// Authenticate accepts the nonce the caller sent as its bearer credential,
// spending it: a nonce works exactly once even within its TTL, so one seen by
// anything that can read the caller's arguments is already spent.
//
// A missing credential, a malformed one, an unknown one and an expired one are
// one and the same refusal. Which it was is not information a caller who cannot
// decrypt a challenge should get.
func (a *AgeNonce) Authenticate(ctx context.Context) error {
	credential, ok := BearerFromContext(ctx)
	if !ok || !a.nonces.Verify(credential) {
		return status.Error(codes.PermissionDenied,
			"admin credential is missing, unknown or expired; get a fresh nonce with "+
				`GetNonce and send it as "authorization: Bearer <nonce>"`)
	}
	return nil
}
