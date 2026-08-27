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
	id, err := age.GenerateX25519Identity()
	assert.NilError(t, err)
	store, _ := newTestStore(t)
	svc, err := NewAdminService(store, id.Recipient().String(), nil)
	assert.NilError(t, err)
	return svc, id
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

func TestNewAdminService_RecipientParsing(t *testing.T) {
	store, _ := newTestStore(t)

	t.Run("a malformed recipient fails construction", func(t *testing.T) {
		_, err := NewAdminService(store, "age1-not-a-key", nil)
		assert.ErrorContains(t, err, "chat admin recipient")
	})

	t.Run("no recipient constructs an admin-less service", func(t *testing.T) {
		svc, err := NewAdminService(store, "", nil)
		assert.NilError(t, err)
		assert.Assert(t, svc.recipient == nil)
	})
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
	_, err = svc.ListRooms(t.Context(), &chatv1.ListRoomsRequest{Nonce: nonce})
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
		_, err := svc.ListRooms(t.Context(), &chatv1.ListRoomsRequest{Nonce: nonce})
		assert.NilError(t, err)

		_, err = svc.ListRooms(t.Context(), &chatv1.ListRoomsRequest{Nonce: nonce})
		assert.Equal(t, status.Code(err), codes.PermissionDenied)
	})

	t.Run("a made-up nonce is refused", func(t *testing.T) {
		_, err := svc.ListRooms(t.Context(), &chatv1.ListRoomsRequest{Nonce: "guessed"})
		assert.Equal(t, status.Code(err), codes.PermissionDenied)
	})

	t.Run("a missing nonce is refused", func(t *testing.T) {
		_, err := svc.ListRooms(t.Context(), &chatv1.ListRoomsRequest{})
		assert.Equal(t, status.Code(err), codes.PermissionDenied)
	})

	t.Run("a nonce is refused by the RPC that consumed it", func(t *testing.T) {
		// A failed RPC still spends its nonce.
		nonce := adminNonce(t, svc, id)
		_, err := svc.MoveMember(t.Context(), &chatv1.MoveMemberRequest{
			Nonce: nonce, Room: "/work", Team: "alpha", Name: "nobody", ToTeam: "beta",
		})
		assert.Equal(t, status.Code(err), codes.NotFound)

		_, err = svc.ListRooms(t.Context(), &chatv1.ListRoomsRequest{Nonce: nonce})
		assert.Equal(t, status.Code(err), codes.PermissionDenied)
	})
}

func TestAdminService_WithoutRecipientEveryRPCIsRefused(t *testing.T) {
	store, _ := newTestStore(t)
	svc, err := NewAdminService(store, "", nil)
	assert.NilError(t, err)

	_, err = svc.GetNonce(t.Context(), &chatv1.GetNonceRequest{})
	assert.Equal(t, status.Code(err), codes.FailedPrecondition)

	// Even carrying a nonce-shaped argument: there is nothing to verify it
	// against, so the refusal is the configuration's, not the caller's.
	_, err = svc.ListRooms(t.Context(), &chatv1.ListRoomsRequest{Nonce: "anything"})
	assert.Equal(t, status.Code(err), codes.FailedPrecondition)

	_, err = svc.MoveMember(t.Context(), &chatv1.MoveMemberRequest{
		Nonce: "anything", Room: "/work", Team: "alpha", Name: "ana", ToTeam: "beta",
	})
	assert.Equal(t, status.Code(err), codes.FailedPrecondition)

	_, err = svc.RegisterMember(t.Context(), &chatv1.RegisterMemberRequest{
		Nonce: "anything", Room: "/work", Team: "hosts", Name: "hana",
	})
	assert.Equal(t, status.Code(err), codes.FailedPrecondition)
}

// TestAdminService_OverGRPC exercises the daemon's wiring: the admin service
// shares the socket and the interceptor with ChatService, yet its calls carry
// no token — the nonce in the request is the whole credential.
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

	rooms, err := client.ListRooms(t.Context(), &chatv1.ListRoomsRequest{Nonce: nonce})
	assert.NilError(t, err)
	assert.Equal(t, len(rooms.GetRooms()), 1)
	assert.Equal(t, rooms.GetRooms()[0].GetName(), "/work")
}
