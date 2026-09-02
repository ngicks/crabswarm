package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"

	"github.com/ngicks/crabswarm/crabswarm/chat/auth"
)

// newIdentityFile writes a fresh age identity the way the age CLI does — one
// private key per line — and returns its path and matching recipient.
func newIdentityFile(t *testing.T) (path string, recipient *age.X25519Recipient) {
	t.Helper()
	id, err := age.GenerateX25519Identity()
	assert.NilError(t, err)
	path = filepath.Join(t.TempDir(), "chat_admin.key")
	assert.NilError(t, os.WriteFile(
		path,
		[]byte("# created by a test\n"+id.String()+"\n"),
		0o600,
	))
	return path, id.Recipient()
}

func encryptTo(t *testing.T, recipient age.Recipient, plaintext string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipient)
	assert.NilError(t, err)
	_, err = io.WriteString(w, plaintext)
	assert.NilError(t, err)
	assert.NilError(t, w.Close())
	return buf.Bytes()
}

func TestDecryptNonce_RoundTrip(t *testing.T) {
	path, recipient := newIdentityFile(t)

	got, err := DecryptNonce(path, encryptTo(t, recipient, "nonce-abc123"))
	assert.NilError(t, err)
	assert.Equal(t, got, "nonce-abc123")
}

// Holding the wrong identity is the whole failure mode the challenge exists to
// catch, and the error has to point at the file rather than at age internals.
func TestDecryptNonce_WrongIdentity(t *testing.T) {
	path, _ := newIdentityFile(t)
	_, other := newIdentityFile(t)

	_, err := DecryptNonce(path, encryptTo(t, other, "nonce-abc123"))
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), path))
}

func TestDecryptNonce_MissingFile(t *testing.T) {
	_, err := DecryptNonce(filepath.Join(t.TempDir(), "absent.key"), nil)
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "admin identity file"))
}

func TestResolveIdentityPath(t *testing.T) {
	got, err := ResolveIdentityPath("/from/flag.key", "/from/config.key")
	assert.NilError(t, err)
	assert.Equal(t, got, "/from/flag.key")

	got, err = ResolveIdentityPath("", "/from/config.key")
	assert.NilError(t, err)
	assert.Equal(t, got, "/from/config.key")

	_, err = ResolveIdentityPath("", "")
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "--identity"))
}

// A configured path is written with "~" and printed back that way, so it is
// expanded where it is opened rather than where it is configured.
func TestResolveIdentityPath_ExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := ResolveIdentityPath("", "~/.config/crabswarm/chat_admin.key")
	assert.NilError(t, err)
	assert.Equal(t, got, filepath.Join(home, ".config", "crabswarm", "chat_admin.key"))
}

// fakeAdminService issues a real age-encrypted challenge and records the nonce
// echoed by each following call, which is what the CLI's half of the
// challenge-response has to get right.
type fakeAdminService struct {
	chatv1.UnimplementedChatAdminServiceServer

	recipient age.Recipient
	nonce     string

	nonceCalls int
	bearer     string
	move       *chatv1.MoveMemberRequest
	register   *chatv1.RegisterMemberRequest
	send       *chatv1.AdminSendRequest
	history    *chatv1.AdminHistoryRequest

	rooms     []*chatv1.Room
	delivered int32
	entries   []*chatv1.AdminHistoryEntry
}

func (f *fakeAdminService) GetNonce(
	_ context.Context, _ *chatv1.GetNonceRequest,
) (*chatv1.GetNonceResponse, error) {
	f.nonceCalls++
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, f.recipient)
	if err != nil {
		return nil, err
	}
	if _, err := io.WriteString(w, f.nonce); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return &chatv1.GetNonceResponse{EncryptedNonce: buf.Bytes()}, nil
}

func (f *fakeAdminService) ListRooms(
	ctx context.Context, _ *chatv1.ListRoomsRequest,
) (*chatv1.ListRoomsResponse, error) {
	f.bearer, _ = auth.BearerFromContext(ctx)
	return &chatv1.ListRoomsResponse{Rooms: f.rooms}, nil
}

func (f *fakeAdminService) MoveMember(
	ctx context.Context, req *chatv1.MoveMemberRequest,
) (*chatv1.MoveMemberResponse, error) {
	f.bearer, _ = auth.BearerFromContext(ctx)
	f.move = req
	return &chatv1.MoveMemberResponse{
		Member: member(req.GetToTeam(), req.GetName(), req.GetRoom()),
	}, nil
}

func (f *fakeAdminService) RegisterMember(
	ctx context.Context, req *chatv1.RegisterMemberRequest,
) (*chatv1.RegisterMemberResponse, error) {
	f.bearer, _ = auth.BearerFromContext(ctx)
	f.register = req
	return &chatv1.RegisterMemberResponse{
		Member: member(req.GetTeam(), req.GetName(), req.GetRoom()),
		Token:  "tok-issued",
	}, nil
}

func (f *fakeAdminService) Send(
	ctx context.Context, req *chatv1.AdminSendRequest,
) (*chatv1.AdminSendResponse, error) {
	f.bearer, _ = auth.BearerFromContext(ctx)
	f.send = req
	return &chatv1.AdminSendResponse{Delivered: f.delivered}, nil
}

func (f *fakeAdminService) History(
	ctx context.Context, req *chatv1.AdminHistoryRequest,
) (*chatv1.AdminHistoryResponse, error) {
	f.bearer, _ = auth.BearerFromContext(ctx)
	f.history = req
	return &chatv1.AdminHistoryResponse{Entries: f.entries}, nil
}

func TestClient_ListRoomsDecryptsTheChallenge(t *testing.T) {
	path, recipient := newIdentityFile(t)
	fake := &fakeAdminService{
		recipient: recipient,
		nonce:     "nonce-abc123",
		rooms: []*chatv1.Room{{
			Name:    "/work/proj",
			Members: []*chatv1.Member{member("backend", "alice", "/work/proj")},
		}},
	}
	d := serveTestDaemon(t, nil, fake)

	var out strings.Builder
	assert.NilError(t, d.client.ListRooms(t.Context(), &out, path))
	assert.Equal(t, fake.bearer, "nonce-abc123")
	assert.Equal(t, out.String(),
		"room: /work/proj\n  team: backend\n    alice\n")

	// Admin calls carry no identity token; the challenge is the credential.
	assert.DeepEqual(t, d.seenTokens(), []string{"", ""})
}

func TestClient_MoveMember(t *testing.T) {
	path, recipient := newIdentityFile(t)
	fake := &fakeAdminService{recipient: recipient, nonce: "nonce-move"}
	d := serveTestDaemon(t, nil, fake)

	var out strings.Builder
	assert.NilError(t, d.client.MoveMember(
		t.Context(), &out, path, "/work/proj", "backend/alice", "frontend"))
	assert.Equal(t, fake.bearer, "nonce-move")
	assert.Equal(t, fake.move.GetRoom(), "/work/proj")
	assert.Equal(t, fake.move.GetTeam(), "backend")
	assert.Equal(t, fake.move.GetName(), "alice")
	assert.Equal(t, fake.move.GetToTeam(), "frontend")
	assert.Equal(t, out.String(), "moved frontend/alice in room /work/proj\n")
}

// A malformed member address is caught before a challenge is spent on it.
func TestClient_MoveMemberRejectsUnqualifiedName(t *testing.T) {
	path, recipient := newIdentityFile(t)
	fake := &fakeAdminService{recipient: recipient, nonce: "nonce-move"}
	d := serveTestDaemon(t, nil, fake)

	err := d.client.MoveMember(
		t.Context(), &strings.Builder{}, path, "/work/proj", "alice", "frontend")
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "team/name"))
	assert.Equal(t, fake.nonceCalls, 0)
}

func TestClient_RegisterMemberPrintsTheToken(t *testing.T) {
	path, recipient := newIdentityFile(t)
	fake := &fakeAdminService{recipient: recipient, nonce: "nonce-reg"}
	d := serveTestDaemon(t, nil, fake)

	var out strings.Builder
	assert.NilError(t, d.client.RegisterMember(
		t.Context(), &out, path, "/work/proj", "humans", "yuki"))
	assert.Equal(t, fake.bearer, "nonce-reg")
	assert.Equal(t, fake.register.GetRoom(), "/work/proj")
	assert.Equal(t, fake.register.GetTeam(), "humans")
	assert.Equal(t, fake.register.GetName(), "yuki")
	assert.Equal(t, out.String(),
		"registered humans/yuki in room /work/proj\ntoken: tok-issued\n")
}

// The target picks the case of the request that carries it: everyone is the
// whole room, and the name half is left for the daemon to resolve with the same
// grammar member send uses.
func TestClient_AdminSendReportsTheDeliveredCount(t *testing.T) {
	path, recipient := newIdentityFile(t)
	fake := &fakeAdminService{recipient: recipient, nonce: "nonce-send", delivered: 3}
	d := serveTestDaemon(t, nil, fake)

	var out strings.Builder
	assert.NilError(t, d.client.AdminSend(
		t.Context(), &out, path, "/work/proj",
		AdminTarget{Everyone: true}, "standup in five"))
	assert.Equal(t, fake.bearer, "nonce-send")
	assert.Equal(t, fake.send.GetRoom(), "/work/proj")
	assert.Assert(t, fake.send.GetEveryone() != nil)
	assert.Equal(t, fake.send.GetText(), "standup in five")
	assert.Equal(t, out.String(),
		"sent to * in room /work/proj: delivered to 3 members\n")
}

// A team target carries the team alone; a qualified member target names the
// team it means, and a bare one leaves the team empty rather than guessing,
// which is what makes the daemon resolve it across the room the way it resolves
// a member's bare address.
func TestClient_AdminSendMapsTheTargetOntoItsCase(t *testing.T) {
	for _, tc := range []struct {
		target     string
		team, name string
	}{
		{"backend/*", "backend", ""},
		{"backend/alice", "backend", "alice"},
		{"alice", "", "alice"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			path, recipient := newIdentityFile(t)
			fake := &fakeAdminService{
				recipient: recipient, nonce: "nonce-send", delivered: 1,
			}
			d := serveTestDaemon(t, nil, fake)

			target, err := ParseAdminTarget(tc.target)
			assert.NilError(t, err)
			var out strings.Builder
			assert.NilError(t, d.client.AdminSend(
				t.Context(), &out, path, "/work/proj", target, "hi"))
			if tc.name == "" {
				assert.Equal(t, fake.send.GetTeam().GetTeam(), tc.team)
				assert.Assert(t, fake.send.GetMember() == nil)
			} else {
				assert.Equal(t, fake.send.GetMember().GetTeam(), tc.team)
				assert.Equal(t, fake.send.GetMember().GetName(), tc.name)
			}
			// The rendered line names the target the way it was written.
			assert.Assert(t, strings.HasPrefix(out.String(), "sent to "+tc.target+" "))
		})
	}
}

// An identity the daemon does not encrypt to stops the send at the challenge,
// and the failure has to name the file to look at rather than fail as the send.
func TestClient_AdminSendWithWrongIdentity(t *testing.T) {
	path, _ := newIdentityFile(t)
	_, other := newIdentityFile(t)
	fake := &fakeAdminService{recipient: other, nonce: "nonce-send"}
	d := serveTestDaemon(t, nil, fake)

	err := d.client.AdminSend(
		t.Context(), &strings.Builder{}, path, "/work/proj",
		AdminTarget{Team: "backend", Name: "alice"}, "hi")
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), path))
	assert.Assert(t, strings.Contains(err.Error(), "hint:"))
	assert.Assert(t, fake.send == nil)
}

// The verb reads the tail: the room and the window it was given reach the
// daemon, and the cursor stays at zero, since a command run once has nothing to
// carry forward from a previous run.
func TestClient_AdminLogPrintsTheTranscript(t *testing.T) {
	path, recipient := newIdentityFile(t)
	sent := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	fake := &fakeAdminService{
		recipient: recipient,
		nonce:     "nonce-log",
		entries: []*chatv1.AdminHistoryEntry{{
			Id:     7,
			From:   member("admin", "admin", "/work/proj"),
			Text:   "deploy is frozen",
			SentAt: timestamppb.New(sent),
		}},
	}
	d := serveTestDaemon(t, nil, fake)

	var out strings.Builder
	assert.NilError(t, d.client.AdminLog(t.Context(), &out, path, "/work/proj", 20))
	assert.Equal(t, fake.bearer, "nonce-log")
	assert.Equal(t, fake.history.GetRoom(), "/work/proj")
	assert.Equal(t, fake.history.GetLimit(), int32(20))
	assert.Equal(t, fake.history.GetSinceId(), int64(0))
	assert.Equal(t, out.String(),
		"[2026-08-27T09:30:00Z] admin/admin → *: deploy is frozen\n")
}

// An identity the daemon does not encrypt to stops the command at the
// challenge, before the operation is attempted.
func TestClient_AdminWithWrongIdentity(t *testing.T) {
	path, _ := newIdentityFile(t)
	_, other := newIdentityFile(t)
	fake := &fakeAdminService{recipient: other, nonce: "nonce-abc123"}
	d := serveTestDaemon(t, nil, fake)

	err := d.client.ListRooms(t.Context(), &strings.Builder{}, path)
	assert.Assert(t, err != nil)
	assert.Equal(t, fake.bearer, "")
}
