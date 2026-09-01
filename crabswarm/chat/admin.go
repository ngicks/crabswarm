package chat

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/auth"
)

// AdminService is the host-facing half of the chat broker: the ChatAdminService
// gRPC implementation over the [Store]. It carries the operations a participant
// must not be able to perform — reading every room, editing team formation,
// minting tokens for humans no provider vouches for, and sending into any room
// without attending it.
//
// Its caller is not identified by a token: an agent holds one of those. It is
// identified per call by the credential its [AdminAuthenticator] accepts, which
// the caller sends as the bearer credential of the call's authorization
// metadata. How that credential is obtained and judged is entirely the
// authenticator's business; this service only asks.
//
// With no authenticator configured nothing can prove anything, so every RPC
// here fails; see [NewAdminService].
type AdminService struct {
	chatv1.UnimplementedChatAdminServiceServer

	store    *Store
	provider TeamInfoProvider
	auth     AdminAuthenticator
	deliver  deliverer
	logger   *slog.Logger
}

// AdminChallenge is what [AdminAuthenticator.Challenge] hands a caller to
// answer.
type AdminChallenge = auth.Challenge

// AdminAuthenticator decides whether the caller of an admin RPC may perform it.
//
// Authenticate reads the caller's credential out of ctx itself — it arrives as
// the incoming call's bearer metadata — and returns nil when the caller may
// proceed. A refusal must be a gRPC status the service can return as it is,
// since only the authenticator knows what a caller could have presented
// instead; anything else is reported as Internal.
//
// Challenge issues whatever the caller must answer to obtain a credential. An
// authenticator whose credentials are minted elsewhere has no challenge step
// and returns a status carrying codes.Unimplemented, which reaches the caller
// as the answer that there is nothing to fetch.
type AdminAuthenticator interface {
	Challenge(ctx context.Context) (AdminChallenge, error)
	Authenticate(ctx context.Context) error
}

var _ chatv1.ChatAdminServiceServer = (*AdminService)(nil)

// NewAdminService returns the ChatAdminService implementation over store,
// gated by authenticator and reporting the deliveries of [AdminService.Send] to
// notifier. A nil notifier means [NopNotifier]; a nil logger discards logs.
//
// The notifier is the same seam the member half is given, and normally the same
// instance: a recipient is nudged for an operator's message the way it is for a
// peer's, since from where it sits both are mail. The provider is the same one
// too, and is consulted for one thing only: whether the member holding a name
// an operator's move collides with is still there. A nil provider leaves every
// such collision a refusal, since nothing can then show the name to be free.
//
// A nil authenticator is not an error: it is a daemon that was never given a
// way to recognise its operator, and it leaves every admin RPC failing with
// FailedPrecondition rather than refusing to serve chat at all.
func NewAdminService(
	store *Store,
	provider TeamInfoProvider,
	authenticator AdminAuthenticator,
	notifier Notifier,
	logger *slog.Logger,
) *AdminService {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &AdminService{
		store:    store,
		provider: provider,
		auth:     authenticator,
		deliver:  newDeliverer(store, notifier, logger),
		logger:   logger,
	}
}

// GetNonce hands back whatever the configured authenticator wants the caller to
// answer. It is the one admin RPC that proves nothing, which is why it is the
// one that may be called without a credential: a challenge only its intended
// reader can use tells anyone else nothing they did not already know.
//
// Unimplemented from the authenticator reaches the caller unchanged: it is the
// answer that this daemon has no challenge step, not a failure.
func (a *AdminService) GetNonce(
	ctx context.Context,
	_ *chatv1.GetNonceRequest,
) (*chatv1.GetNonceResponse, error) {
	if a.auth == nil {
		return nil, adminNotConfigured()
	}
	challenge, err := a.auth.Challenge(ctx)
	if err != nil {
		return nil, authStatus(err)
	}
	return &chatv1.GetNonceResponse{
		EncryptedNonce: challenge.Payload,
		ExpiresAt:      timestamppb.New(challenge.ExpiresAt),
	}, nil
}

// authenticate spends the caller's credential on behalf of an RPC and turns a
// refusal into the status the wire carries.
//
// It runs before the RPC does its work, so an RPC that then fails on its own
// terms still costs a fresh credential. Tying it to the outcome would buy
// nothing: the round trip is one call in the CLI.
//
// What counts as a refusal is the authenticator's to say; this only makes sure
// an unconfigured daemon answers before one is consulted.
func (a *AdminService) authenticate(ctx context.Context) error {
	if a.auth == nil {
		return adminNotConfigured()
	}
	if err := a.auth.Authenticate(ctx); err != nil {
		return authStatus(err)
	}
	return nil
}

// authStatus passes an authenticator's own status through untouched — it is the
// only party that knows what the caller could have presented instead — and
// reports anything else as Internal.
func authStatus(err error) error {
	if _, ok := status.FromError(err); ok {
		return err
	}
	return status.Error(codes.Internal, err.Error())
}

// adminNotConfigured is the refusal every admin RPC gives when the daemon has
// no admin recipient. FailedPrecondition rather than PermissionDenied: there is
// nothing the caller could have presented, the host has to configure a
// recipient first.
func adminNotConfigured() error {
	return status.Error(codes.FailedPrecondition, "chat admin recipient not configured")
}
