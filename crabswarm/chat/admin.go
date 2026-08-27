package chat

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"sync"
	"time"

	"filippo.io/age"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// nonceTTL is how long an issued challenge stays acceptable. Long enough for a
// human-driven CLI to decrypt the nonce and make its call, short enough that a
// nonce leaked from a shell history or a process listing is already dead.
const nonceTTL = 2 * time.Minute

// maxOutstandingNonces caps the challenges kept at once. GetNonce answers
// anyone who reaches the socket — only the identity holder can read the reply,
// but anyone can ask — so the set needs a ceiling. Reaching it evicts the
// oldest challenge instead of refusing to issue: refusing would let a spammer
// pin the set full and lock the operator out, while eviction costs at most a
// retry, and the CLI uses a nonce milliseconds after asking for it.
const maxOutstandingNonces = 128

// AdminService is the host-facing half of the chat broker: the ChatAdminService
// gRPC implementation over the [Store]. It carries the operations a participant
// must not be able to perform — reading every room, editing team formation, and
// minting tokens for humans no provider vouches for.
//
// Its caller is not identified by a token: an agent holds one of those. It is
// identified by possession of the age identity file matching the configured
// admin recipient, proven per call by the nonce challenge — age encrypts but
// does not sign, so the proof is "you could read what only you can read".
// GetNonce hands out a nonce encrypted to the recipient and every other RPC
// carries it back in plaintext.
//
// With no recipient configured nothing can prove that possession, so every RPC
// here fails; see [NewAdminService].
type AdminService struct {
	chatv1.UnimplementedChatAdminServiceServer

	store     *Store
	recipient age.Recipient
	logger    *slog.Logger

	mu sync.Mutex
	// nonces maps each outstanding challenge to when it stops being accepted.
	nonces map[string]time.Time
}

var _ chatv1.ChatAdminServiceServer = (*AdminService)(nil)

// NewAdminService returns the ChatAdminService implementation over store,
// gated by the age recipient named by adminRecipient — the "age1..." public
// key of the identity file the host operator keeps outside the mounts
// participants see. A nil logger discards logs.
//
// An empty adminRecipient is not an error: it is a daemon that was never given
// an admin key, and it leaves every admin RPC failing with FailedPrecondition
// rather than refusing to serve chat at all. A non-empty one that does not
// parse IS an error, so a typo in the config is caught at startup instead of at
// the first admin call.
//
// Only native age recipients are accepted, not the ssh keys the age CLI also
// understands: the config names one recipient string, and an ssh recipient
// would drag in its own key-format parsing for no gain here.
func NewAdminService(
	store *Store,
	adminRecipient string,
	logger *slog.Logger,
) (*AdminService, error) {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	var recipient age.Recipient
	if adminRecipient != "" {
		parsed, err := age.ParseX25519Recipient(adminRecipient)
		if err != nil {
			return nil, fmt.Errorf("parsing chat admin recipient %q: %w", adminRecipient, err)
		}
		recipient = parsed
	}
	return &AdminService{
		store:     store,
		recipient: recipient,
		logger:    logger,
		nonces:    make(map[string]time.Time),
	}, nil
}

// GetNonce issues a challenge encrypted to the configured admin recipient. It
// is the one admin RPC that proves nothing: the challenge is readable only by
// the identity file, so handing it to a caller who cannot decrypt it tells them
// nothing they did not already know.
//
// The payload is a raw age file, not ASCII armor: the field is bytes on a
// binary transport, and armor exists for the channels that cannot carry
// arbitrary bytes. The nonce inside is [rand.Text] — 128 unguessable bits in an
// alphabet that survives a proto string field and a shell argument unescaped,
// which is where it travels next.
func (a *AdminService) GetNonce(
	_ context.Context,
	_ *chatv1.GetNonceRequest,
) (*chatv1.GetNonceResponse, error) {
	if a.recipient == nil {
		return nil, adminNotConfigured()
	}
	nonce := rand.Text()
	payload, err := encryptNonce(a.recipient, nonce)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	expiresAt := time.Now().Add(nonceTTL)
	a.issue(nonce, expiresAt)
	return &chatv1.GetNonceResponse{
		EncryptedNonce: payload,
		ExpiresAt:      timestamppb.New(expiresAt),
	}, nil
}

// issue records nonce as outstanding until expiresAt, sweeping the expired
// challenges and evicting the oldest survivor when the set is at
// [maxOutstandingNonces].
func (a *AdminService) issue(nonce string, expiresAt time.Time) {
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	maps.DeleteFunc(a.nonces, func(_ string, exp time.Time) bool { return now.After(exp) })
	for len(a.nonces) >= maxOutstandingNonces {
		var (
			oldest    string
			oldestExp time.Time
		)
		for n, exp := range a.nonces {
			if oldestExp.IsZero() || exp.Before(oldestExp) {
				oldest, oldestExp = n, exp
			}
		}
		delete(a.nonces, oldest)
	}
	a.nonces[nonce] = expiresAt
}

// verifyNonce accepts one outstanding challenge and consumes it, so a nonce
// works exactly once even within its TTL — a nonce seen by anything that can
// read the caller's arguments is spent by the time it is seen.
//
// Consumption happens here, before the RPC does its work, so an RPC that then
// fails on its own terms still costs a fresh GetNonce. Tying it to the outcome
// would buy nothing: the round trip is one call in the CLI.
//
// Unknown, expired and empty all read as the same PermissionDenied: which one
// it was is not information a caller who cannot decrypt a nonce should get.
func (a *AdminService) verifyNonce(nonce string) error {
	if a.recipient == nil {
		return adminNotConfigured()
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	expiresAt, ok := a.nonces[nonce]
	delete(a.nonces, nonce)
	if !ok || time.Now().After(expiresAt) {
		return status.Error(codes.PermissionDenied,
			"admin nonce is unknown or expired; get a fresh one with GetNonce")
	}
	return nil
}

// adminNotConfigured is the refusal every admin RPC gives when the daemon has
// no admin recipient. FailedPrecondition rather than PermissionDenied: there is
// nothing the caller could have presented, the host has to configure a
// recipient first.
func adminNotConfigured() error {
	return status.Error(codes.FailedPrecondition, "chat admin recipient not configured")
}

// DecryptNonce reads the nonce out of a GetNonce response's encrypted payload
// with the operator's age identities, which [age.ParseIdentities] parses out of
// the identity file. It is the client half of the admin challenge, kept beside
// the server half so the two cannot drift apart on the payload format.
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

// encryptNonce encrypts nonce to recipient as a raw age file.
func encryptNonce(recipient age.Recipient, nonce string) ([]byte, error) {
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
