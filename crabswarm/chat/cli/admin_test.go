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
	register   *chatv1.RegisterMemberRequest
	send       *chatv1.AdminSendRequest
	history    *chatv1.AdminHistoryRequest
	deleted    *chatv1.DeleteRoomRequest

	rooms     []*chatv1.Room
	mentioned []*chatv1.Member
	absent    []*chatv1.Member
	messages  []*chatv1.Message
	removed   int64
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
	return &chatv1.AdminSendResponse{Mentioned: f.mentioned, Absent: f.absent}, nil
}

func (f *fakeAdminService) History(
	ctx context.Context, req *chatv1.AdminHistoryRequest,
) (*chatv1.AdminHistoryResponse, error) {
	f.bearer, _ = auth.BearerFromContext(ctx)
	f.history = req
	return &chatv1.AdminHistoryResponse{Messages: f.messages}, nil
}

func (f *fakeAdminService) DeleteRoom(
	ctx context.Context, req *chatv1.DeleteRoomRequest,
) (*chatv1.DeleteRoomResponse, error) {
	f.bearer, _ = auth.BearerFromContext(ctx)
	f.deleted = req
	return &chatv1.DeleteRoomResponse{DeletedMessages: f.removed}, nil
}

func TestClient_ListRoomsDecryptsTheChallenge(t *testing.T) {
	path, recipient := newIdentityFile(t)
	fake := &fakeAdminService{
		recipient: recipient,
		nonce:     "nonce-abc123",
		rooms: []*chatv1.Room{
			{
				Name: "/work/proj",
				Members: []*chatv1.Member{
					memberWith("backend", "alice", "/work/proj",
						chatv1.MemberKind_MEMBER_KIND_AGENT,
						chatv1.HarnessState_HARNESS_STATE_DONE),
				},
			},
			// A room nobody is in is still a room to read or to delete, so the
			// listing names it rather than leaving it out.
			{Name: "/work/done"},
		},
	}
	d := serveTestDaemon(t, nil, fake)

	var out strings.Builder
	assert.NilError(t, d.client.ListRooms(t.Context(), &out, path))
	assert.Equal(t, fake.bearer, "nonce-abc123")
	assert.Equal(t, out.String(),
		"room: /work/proj\n  team: backend\n    alice  agent\n"+
			"room: /work/done\n  (nobody attending)\n")

	// Admin calls carry no identity token; the challenge is the credential.
	assert.DeepEqual(t, d.seenTokens(), []string{"", ""})
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

// The admin addresses a room in the grammar `chat send` takes, and reads back
// the same two lists a member send does: who the target named, and who of them
// nobody is attending under.
func TestClient_AdminSend(t *testing.T) {
	path, recipient := newIdentityFile(t)
	fake := &fakeAdminService{
		recipient: recipient,
		nonce:     "nonce-send",
		mentioned: []*chatv1.Member{member("backend", "alice", "/work/proj")},
		absent:    []*chatv1.Member{member("backend", "alice", "/work/proj")},
	}
	d := serveTestDaemon(t, nil, fake)

	target, err := ParseTarget("backend/alice,bob")
	assert.NilError(t, err)
	var out, warn strings.Builder
	assert.NilError(t, d.client.AdminSend(
		t.Context(), &out, &warn, path, "/work/proj", target, "standup in five"))
	assert.Equal(t, fake.bearer, "nonce-send")
	assert.Equal(t, fake.send.GetRoom(), "/work/proj")
	assert.Equal(t, fake.send.GetText(), "standup in five")
	// A bare name reaches the daemon with no team, which is what makes it
	// resolve the name across the room rather than in a team the CLI guessed.
	assert.Equal(t, TargetString(fake.send.GetTarget()), "backend/alice,bob")
	assert.Equal(t, out.String(), "mentioned backend/alice\n")
	assert.Equal(t, warn.String(),
		"warning: backend/alice is not attending; the mention waits\n")
}

// A board post reaches a room with no target at all, so it mentions nobody and
// interrupts no one.
func TestClient_AdminSendPostCarriesNoTarget(t *testing.T) {
	path, recipient := newIdentityFile(t)
	fake := &fakeAdminService{recipient: recipient, nonce: "nonce-send"}
	d := serveTestDaemon(t, nil, fake)

	target, err := ParseTarget("")
	assert.NilError(t, err)
	var out, warn strings.Builder
	assert.NilError(t, d.client.AdminSend(
		t.Context(), &out, &warn, path, "/work/proj", target, "deploy is frozen"))
	assert.Assert(t, fake.send.GetTarget() == nil)
	assert.Equal(t, out.String(), "")
	assert.Equal(t, warn.String(), "")
}

// An identity the daemon does not encrypt to stops the send at the challenge,
// and the failure has to name the file to look at rather than fail as the send.
func TestClient_AdminSendWithWrongIdentity(t *testing.T) {
	path, _ := newIdentityFile(t)
	_, other := newIdentityFile(t)
	fake := &fakeAdminService{recipient: other, nonce: "nonce-send"}
	d := serveTestDaemon(t, nil, fake)

	target, err := ParseTarget("backend/alice")
	assert.NilError(t, err)
	err = d.client.AdminSend(
		t.Context(), &strings.Builder{}, &strings.Builder{}, path, "/work/proj",
		target, "hi")
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), path))
	assert.Assert(t, strings.Contains(err.Error(), "hint:"))
	assert.Assert(t, fake.send == nil)
}

// The filter reaches the daemon as the operator wrote it, and the transcript
// comes back in the shape every read prints. The admin sends with no team of
// its own, which is why its line reads "admin" rather than "/admin".
func TestClient_AdminLogPrintsTheTranscript(t *testing.T) {
	path, recipient := newIdentityFile(t)
	sent := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	fake := &fakeAdminService{
		recipient: recipient,
		nonce:     "nonce-log",
		messages: []*chatv1.Message{{
			Seq:    7,
			From:   member("", "admin", "/work/proj"),
			Text:   "deploy is frozen",
			SentAt: timestamppb.New(sent),
		}},
	}
	d := serveTestDaemon(t, nil, fake)

	filter, err := ReadFlags{Cursor: "tail", Range: -20}.HistoryFilter()
	assert.NilError(t, err)

	var out strings.Builder
	assert.NilError(t, d.client.AdminLog(t.Context(), &out, path, "/work/proj", filter))
	assert.Equal(t, fake.bearer, "nonce-log")
	assert.Equal(t, fake.history.GetRoom(), "/work/proj")
	assert.Equal(t, fake.history.GetFilter().GetCursor(),
		chatv1.ReadCursor_READ_CURSOR_TAIL)
	assert.Equal(t, fake.history.GetFilter().GetRange(), int32(-20))
	assert.Equal(t, out.String(),
		"7 2026-08-27T09:30:00Z admin -> -: deploy is frozen\n")
}

// Deleting a room is the one cleanup there is, so the verb reports what went
// with it rather than saying nothing.
func TestClient_DeleteRoom(t *testing.T) {
	path, recipient := newIdentityFile(t)
	fake := &fakeAdminService{recipient: recipient, nonce: "nonce-del", removed: 12}
	d := serveTestDaemon(t, nil, fake)

	var out strings.Builder
	assert.NilError(t, d.client.DeleteRoom(t.Context(), &out, path, "/work/proj"))
	assert.Equal(t, fake.bearer, "nonce-del")
	assert.Equal(t, fake.deleted.GetRoom(), "/work/proj")
	assert.Equal(t, out.String(), "deleted room /work/proj and 12 messages\n")
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
