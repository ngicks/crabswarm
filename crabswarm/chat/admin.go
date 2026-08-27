package chat

import (
	"context"
	"fmt"
	"log/slog"

	"filippo.io/age"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/auth"
)

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
	nonces    *auth.Nonces
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
		nonces:    auth.NewNonces(),
	}, nil
}

// GetNonce issues a challenge encrypted to the configured admin recipient. It
// is the one admin RPC that proves nothing: the challenge is readable only by
// the identity file, so handing it to a caller who cannot decrypt it tells them
// nothing they did not already know.
func (a *AdminService) GetNonce(
	_ context.Context,
	_ *chatv1.GetNonceRequest,
) (*chatv1.GetNonceResponse, error) {
	if a.recipient == nil {
		return nil, adminNotConfigured()
	}
	nonce, expiresAt := a.nonces.Issue()
	payload, err := auth.EncryptNonce(a.recipient, nonce)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &chatv1.GetNonceResponse{
		EncryptedNonce: payload,
		ExpiresAt:      timestamppb.New(expiresAt),
	}, nil
}

// authenticate spends the caller's credential on behalf of an RPC and turns a
// refusal into the status the wire carries.
//
// It runs before the RPC does its work, so an RPC that then fails on its own
// terms still costs a fresh credential. Tying it to the outcome would buy
// nothing: the round trip is one call in the CLI.
//
// A missing credential, a malformed one and a wrong one are all the same
// PermissionDenied: which it was is not information a caller who cannot present
// one should get.
func (a *AdminService) authenticate(ctx context.Context) error {
	if a.recipient == nil {
		return adminNotConfigured()
	}
	credential, ok := auth.BearerFromContext(ctx)
	if !ok || !a.nonces.Verify(credential) {
		return status.Error(codes.PermissionDenied,
			"admin credential is missing, unknown or expired; "+
				"get a fresh nonce with GetNonce and send it as "+
				"\"authorization: Bearer <nonce>\"")
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
