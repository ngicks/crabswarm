package chat

import (
	"context"
	"net"
	"testing"
	"time"

	"filippo.io/age"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/auth"
)

// newTestAdminService wires an admin service over a fresh store, gated by a
// throwaway age identity standing in for the operator's identity file.
func newTestAdminService(t *testing.T) (*AdminService, *age.X25519Identity) {
	t.Helper()
	svc, id, _ := newTestAdminServiceWithNotifier(t)
	return svc, id
}

// newTestAdminServiceWithNotifier is [newTestAdminService] with the recording
// notifier handed back too, for the cases that assert on what a delivery
// reported.
func newTestAdminServiceWithNotifier(
	t *testing.T,
) (*AdminService, *age.X25519Identity, *fakeNotifier) {
	t.Helper()
	id, err := age.GenerateX25519Identity()
	assert.NilError(t, err)
	store, _ := newTestStore(t)
	ageAuth, err := auth.NewAgeNonce(id.Recipient().String())
	assert.NilError(t, err)
	notifier := &fakeNotifier{}
	return NewAdminService(store, ageAuth, notifier, nil), id, notifier
}

// adminCtx is the context an admin RPC sees when the caller sent credential as
// its bearer credential. The service methods are called directly here, so the
// metadata has to be put on the incoming side by hand; a call over a real
// connection uses [auth.ContextWithBearer] instead.
func adminCtx(t *testing.T, credential string) context.Context {
	t.Helper()
	return metadata.NewIncomingContext(t.Context(),
		metadata.Pairs(auth.AuthorizationMetadataKey, "Bearer "+credential))
}

// adminNonce runs the challenge the admin CLI runs: ask for a nonce, decrypt it
// with the identity file's identity, hand back the plaintext.
func adminNonce(t *testing.T, svc *AdminService, id age.Identity) string {
	t.Helper()
	res, err := svc.GetNonce(t.Context(), &chatv1.GetNonceRequest{})
	assert.NilError(t, err)
	nonce, err := auth.DecryptNonce(res.GetEncryptedNonce(), id)
	assert.NilError(t, err)
	return nonce
}

func TestNewAdminService_WithoutAnAuthenticator(t *testing.T) {
	store, _ := newTestStore(t)

	svc := NewAdminService(store, nil, nil, nil)
	assert.Assert(t, svc.auth == nil)
}

func TestAdminService_NonceRoundTrip(t *testing.T) {
	svc, id := newTestAdminService(t)

	before := time.Now()
	res, err := svc.GetNonce(t.Context(), &chatv1.GetNonceRequest{})
	assert.NilError(t, err)
	assert.Assert(t, len(res.GetEncryptedNonce()) > 0)

	expiresAt := res.GetExpiresAt().AsTime()
	assert.Assert(t, expiresAt.After(before))
	assert.Assert(t, !expiresAt.After(time.Now().Add(auth.NonceTTL)))

	nonce, err := auth.DecryptNonce(res.GetEncryptedNonce(), id)
	assert.NilError(t, err)
	assert.Assert(t, nonce != "")

	// The decrypted nonce is what the admin RPCs accept.
	_, err = svc.ListRooms(adminCtx(t, nonce), &chatv1.ListRoomsRequest{})
	assert.NilError(t, err)
}

func TestAdminService_NonceIsNotReadableByAnotherIdentity(t *testing.T) {
	svc, _ := newTestAdminService(t)
	other, err := age.GenerateX25519Identity()
	assert.NilError(t, err)

	res, err := svc.GetNonce(t.Context(), &chatv1.GetNonceRequest{})
	assert.NilError(t, err)

	_, err = auth.DecryptNonce(res.GetEncryptedNonce(), other)
	assert.ErrorContains(t, err, "decrypting admin nonce")

	_, err = auth.DecryptNonce([]byte("not an age file"), other)
	assert.ErrorContains(t, err, "decrypting admin nonce")
}

func TestAdminService_RejectsSpentNonce(t *testing.T) {
	svc, id := newTestAdminService(t)

	t.Run("a used nonce does not work twice", func(t *testing.T) {
		nonce := adminNonce(t, svc, id)
		_, err := svc.ListRooms(adminCtx(t, nonce), &chatv1.ListRoomsRequest{})
		assert.NilError(t, err)

		_, err = svc.ListRooms(adminCtx(t, nonce), &chatv1.ListRoomsRequest{})
		assert.Equal(t, status.Code(err), codes.PermissionDenied)
	})

	t.Run("a made-up nonce is refused", func(t *testing.T) {
		_, err := svc.ListRooms(adminCtx(t, "guessed"), &chatv1.ListRoomsRequest{})
		assert.Equal(t, status.Code(err), codes.PermissionDenied)
	})

	t.Run("a call carrying no credential at all is refused", func(t *testing.T) {
		_, err := svc.ListRooms(t.Context(), &chatv1.ListRoomsRequest{})
		assert.Equal(t, status.Code(err), codes.PermissionDenied)
	})

	t.Run("a nonce is refused by the RPC that consumed it", func(t *testing.T) {
		// A failed RPC still spends its nonce.
		nonce := adminNonce(t, svc, id)
		_, err := svc.MoveMember(adminCtx(t, nonce), &chatv1.MoveMemberRequest{
			Room: "/work", Team: "alpha", Name: "nobody", ToTeam: "beta",
		})
		assert.Equal(t, status.Code(err), codes.NotFound)

		_, err = svc.ListRooms(adminCtx(t, nonce), &chatv1.ListRoomsRequest{})
		assert.Equal(t, status.Code(err), codes.PermissionDenied)
	})
}

func TestAdminService_WithoutRecipientEveryRPCIsRefused(t *testing.T) {
	store, _ := newTestStore(t)
	svc := NewAdminService(store, nil, nil, nil)

	_, err := svc.GetNonce(t.Context(), &chatv1.GetNonceRequest{})
	assert.Equal(t, status.Code(err), codes.FailedPrecondition)

	// Even carrying a credential: there is nothing to verify it against, so the
	// refusal is the configuration's, not the caller's.
	_, err = svc.ListRooms(adminCtx(t, "anything"), &chatv1.ListRoomsRequest{})
	assert.Equal(t, status.Code(err), codes.FailedPrecondition)

	_, err = svc.MoveMember(adminCtx(t, "anything"), &chatv1.MoveMemberRequest{
		Room: "/work", Team: "alpha", Name: "ana", ToTeam: "beta",
	})
	assert.Equal(t, status.Code(err), codes.FailedPrecondition)

	_, err = svc.RegisterMember(adminCtx(t, "anything"), &chatv1.RegisterMemberRequest{
		Room: "/work", Team: "hosts", Name: "hana",
	})
	assert.Equal(t, status.Code(err), codes.FailedPrecondition)

	_, err = svc.Send(adminCtx(t, "anything"), &chatv1.AdminSendRequest{
		Room: "/work", Target: "alpha/ana", Text: "hi",
	})
	assert.Equal(t, status.Code(err), codes.FailedPrecondition)
}

// TestAdminService_OverGRPC exercises the daemon's wiring: the admin service
// shares the socket and the interceptor with ChatService, yet its calls carry
// no token — the bearer credential in the call's metadata is the whole of it.
func TestAdminService_OverGRPC(t *testing.T) {
	svc, id := newTestAdminService(t)
	join(t, svc.store, "tok-a", "/work", "alpha", "ana")

	lis := bufconn.Listen(1 << 16)
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(UnaryTokenInterceptor()))
	chatv1.RegisterChatAdminServiceServer(srv, svc)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NilError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	client := chatv1.NewChatAdminServiceClient(conn)

	challenge, err := client.GetNonce(t.Context(), &chatv1.GetNonceRequest{})
	assert.NilError(t, err)
	nonce, err := auth.DecryptNonce(challenge.GetEncryptedNonce(), id)
	assert.NilError(t, err)

	rooms, err := client.ListRooms(
		auth.ContextWithBearer(t.Context(), nonce), &chatv1.ListRoomsRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(rooms.GetRooms()), 1)
	assert.Equal(t, rooms.GetRooms()[0].GetName(), "/work")
}

// noChallengeAuth stands in for an authenticator whose credentials are minted
// elsewhere — an OIDC issuer, say — so there is nothing for the daemon to hand
// out and nothing for a caller to fetch.
type noChallengeAuth struct{ credential string }

var _ AdminAuthenticator = (*noChallengeAuth)(nil)

func (a *noChallengeAuth) Challenge(context.Context) (AdminChallenge, error) {
	return AdminChallenge{}, status.Error(codes.Unimplemented,
		"this daemon takes credentials from an external issuer; there is no challenge")
}

func (a *noChallengeAuth) Authenticate(ctx context.Context) error {
	credential, ok := auth.BearerFromContext(ctx)
	if !ok || credential != a.credential {
		return status.Error(codes.PermissionDenied, "unknown credential")
	}
	return nil
}

// An authenticator with no challenge step says so through GetNonce, and the
// service passes that answer along untouched rather than reporting a failure of
// its own. The RPCs that do the work keep working: a credential the issuer
// minted needs no challenge to become usable.
func TestAdminService_ChallengelessAuthenticator(t *testing.T) {
	store, _ := newTestStore(t)
	svc := NewAdminService(store, &noChallengeAuth{credential: "issued-token"}, nil, nil)

	_, err := svc.GetNonce(t.Context(), &chatv1.GetNonceRequest{})
	assert.Equal(t, status.Code(err), codes.Unimplemented)

	_, err = svc.ListRooms(adminCtx(t, "issued-token"), &chatv1.ListRoomsRequest{})
	assert.NilError(t, err)

	_, err = svc.ListRooms(adminCtx(t, "some other token"), &chatv1.ListRoomsRequest{})
	assert.Equal(t, status.Code(err), codes.PermissionDenied)
}

// A credential that is not a bearer credential at all is refused the same way a
// wrong one is, and never reaches the authenticator as something to check.
func TestAdminService_RefusesMalformedAuthorization(t *testing.T) {
	svc, _ := newTestAdminService(t)

	for _, tc := range []struct {
		name   string
		header string
	}{
		{"another scheme", "Basic dXNlcjpwYXNz"},
		{"a scheme and nothing else", "Bearer"},
		{"an empty credential", "Bearer "},
		{"no scheme at all", "just-the-nonce"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := metadata.NewIncomingContext(t.Context(),
				metadata.Pairs(auth.AuthorizationMetadataKey, tc.header))

			_, err := svc.ListRooms(ctx, &chatv1.ListRoomsRequest{})
			assert.Equal(t, status.Code(err), codes.PermissionDenied)
		})
	}
}
