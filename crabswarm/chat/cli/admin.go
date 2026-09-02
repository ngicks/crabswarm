package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/auth"
)

// The admin RPCs are not gated by a token but by possession of an age identity
// file the host keeps outside the mounts participants see. age encrypts without
// signing, so possession is proven by challenge-response: GetNonce hands back a
// nonce encrypted to the daemon's configured recipient, and each following call
// sends the decrypted nonce as its "authorization: Bearer" credential. One
// challenge is taken per command — a nonce spans a single admin operation, so
// there is nothing to cache between runs.

// ResolveIdentityPath picks the age identity file the admin verbs authenticate
// with: the --identity flag first, then the path configured for the chat
// broker. A leading "~" is expanded here rather than where the path is
// configured, so the config prints back exactly as it was written.
func ResolveIdentityPath(flagIdentity, configured string) (string, error) {
	path := flagIdentity
	if path == "" {
		path = configured
	}
	if path == "" {
		return "", errors.New(
			"no admin age identity file: pass --identity FILE naming the identity " +
				"whose recipient the daemon encrypts admin challenges to")
	}
	return expandHome(path)
}

// expandHome resolves a leading "~" against the user's home directory.
func expandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("expanding %q: %w", path, err)
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~")), nil
}

// DecryptNonce decrypts an admin challenge with the age identity file at path
// and returns the nonce to send back as the bearer credential.
//
// The ciphertext is read as a binary age file, which is what the daemon puts in
// the response's bytes field; nothing here armors or dearmors it.
func DecryptNonce(path string, encrypted []byte) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening the admin identity file: %w", err)
	}
	defer f.Close()

	identities, err := age.ParseIdentities(f)
	if err != nil {
		return "", fmt.Errorf("parsing the admin identity file %q: %w", path, err)
	}
	nonce, err := auth.DecryptNonce(encrypted, identities...)
	if err != nil {
		return "", fmt.Errorf(
			"decrypting the admin challenge with %q: %w\n"+
				"hint: the daemon encrypts it to one configured recipient; "+
				"this file must hold that recipient's identity",
			path, err)
	}
	return nonce, nil
}

// nonce runs one challenge-response round: fetch a challenge and decrypt it
// with the identity file at identityPath.
func (c *Client) nonce(ctx context.Context, identityPath string) (string, error) {
	resp, err := c.admin.GetNonce(ctx, &chatv1.GetNonceRequest{})
	if err != nil {
		return "", callError(err)
	}
	return DecryptNonce(identityPath, resp.GetEncryptedNonce())
}

// AdminClient is the admin half of a [Client] bound to one identity file. It
// hands back what the daemon said instead of rendering it, for the callers that
// keep asking — a screen following a room — rather than making one call and
// printing it.
//
// Each of its calls takes its own challenge, exactly as a one-shot verb does: a
// nonce is spent by the RPC it accompanies, so there is nothing to hold onto
// between two of them.
type AdminClient struct {
	client   *Client
	identity string
}

// Admin returns the admin half bound to the age identity file at identityPath.
func (c *Client) Admin(identityPath string) *AdminClient {
	return &AdminClient{client: c, identity: identityPath}
}

// Rooms reports every room the daemon knows, with everyone attending it and the
// state each member last reported.
func (a *AdminClient) Rooms(ctx context.Context) ([]*chatv1.Room, error) {
	nonce, err := a.client.nonce(ctx, a.identity)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.admin.ListRooms(auth.ContextWithBearer(ctx, nonce),
		&chatv1.ListRoomsRequest{})
	if err != nil {
		return nil, callError(err)
	}
	return resp.GetRooms(), nil
}

// RoomLog reads a room's conversation, oldest first. sinceID is the id of the
// newest entry the caller already holds, which asks for what was said after it;
// zero asks for the tail instead. limit caps how many entries come back, zero
// leaving the window to the daemon, which also clamps a limit larger than the
// room keeps.
func (a *AdminClient) RoomLog(
	ctx context.Context,
	room string,
	sinceID int64,
	limit int32,
) ([]*chatv1.AdminHistoryEntry, error) {
	nonce, err := a.client.nonce(ctx, a.identity)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.admin.History(auth.ContextWithBearer(ctx, nonce),
		&chatv1.AdminHistoryRequest{
			Room:    room,
			Limit:   limit,
			SinceId: sinceID,
		})
	if err != nil {
		return nil, callError(err)
	}
	return resp.GetEntries(), nil
}

// AdminTarget is who an admin send is for: exactly one of the three cases the
// request carries. It is a struct rather than an interface because the CLI
// builds one from a written word and hands it straight on — [ParseAdminTarget]
// is the only thing that decides which case a spelling means.
//
// Everyone is the whole room. Otherwise a Name names a member, with Team
// narrowing it — an empty Team leaves the name for the daemon to resolve across
// the room, exactly as a member's own bare address is resolved. A Team with no
// Name is the team itself, everyone attending it when the send is counted.
type AdminTarget struct {
	Everyone bool
	Team     string // TeamTarget when set and Name is empty
	Name     string // MemberTarget with Team (may be empty) when set
}

// ParseAdminTarget maps the grammar `chat admin send` takes onto the case it
// means: "*" is everyone in the room, "team/*" that whole team, "team/name"
// that member of that team, and a bare name the member the daemon resolves it
// to across the room.
//
// A half-written address is refused here rather than sent: an empty team or
// name reaches the daemon as an address nothing answers to, and NotFound is a
// poor way to be told that a word was left out.
func ParseAdminTarget(s string) (AdminTarget, error) {
	switch s {
	case "":
		return AdminTarget{}, errors.New(
			`no target: write "*", "team/*", "team/name" or "name"`)
	case broadcastTarget:
		return AdminTarget{Everyone: true}, nil
	}
	team, name, qualified := strings.Cut(s, "/")
	if !qualified {
		return AdminTarget{Name: s}, nil
	}
	if team == "" || name == "" || strings.Contains(name, "/") {
		return AdminTarget{}, fmt.Errorf(
			`target %q is not in "team/name" or "team/*" form`, s)
	}
	if name == broadcastTarget {
		return AdminTarget{Team: team}, nil
	}
	return AdminTarget{Team: team, Name: name}, nil
}

// String spells the target the way [ParseAdminTarget] takes it, so a rendered
// line and a log entry name a target the reader can type back.
func (t AdminTarget) String() string {
	switch {
	case t.Everyone:
		return broadcastTarget
	case t.Name == "":
		return t.Team + "/" + broadcastTarget
	case t.Team == "":
		return t.Name
	default:
		return t.Team + "/" + t.Name
	}
}

// Send delivers text into room addressed to target and reports how many
// inboxes it reached.
func (a *AdminClient) Send(
	ctx context.Context,
	room string,
	target AdminTarget,
	text string,
) (delivered int32, err error) {
	nonce, err := a.client.nonce(ctx, a.identity)
	if err != nil {
		return 0, err
	}
	resp, err := a.client.admin.Send(auth.ContextWithBearer(ctx, nonce),
		adminSendRequest(room, target, text))
	if err != nil {
		return 0, callError(err)
	}
	return resp.GetDelivered(), nil
}

// adminSendRequest puts the target in the case of the request that carries it.
// The zero target has no case of its own: it arrives as a team target naming no
// team, which the daemon refuses rather than guessing at.
func adminSendRequest(room string, target AdminTarget, text string) *chatv1.AdminSendRequest {
	req := &chatv1.AdminSendRequest{Room: room, Text: text}
	switch {
	case target.Everyone:
		req.Target = &chatv1.AdminSendRequest_Everyone{Everyone: &chatv1.Everyone{}}
	case target.Name == "":
		req.Target = &chatv1.AdminSendRequest_Team{
			Team: &chatv1.TeamTarget{Team: target.Team},
		}
	default:
		req.Target = &chatv1.AdminSendRequest_Member{
			Member: &chatv1.MemberTarget{Team: target.Team, Name: target.Name},
		}
	}
	return req
}

// ListRooms prints every room the daemon knows and who attends it.
func (c *Client) ListRooms(ctx context.Context, w io.Writer, identityPath string) error {
	rooms, err := c.Admin(identityPath).Rooms(ctx)
	if err != nil {
		return err
	}
	return RenderRooms(w, rooms)
}

// MoveMember moves the member addressed as "team/name" in room into toTeam.
func (c *Client) MoveMember(
	ctx context.Context,
	w io.Writer,
	identityPath, room, member, toTeam string,
) error {
	team, name, err := ParseQualifiedName(member)
	if err != nil {
		return err
	}
	nonce, err := c.nonce(ctx, identityPath)
	if err != nil {
		return err
	}
	resp, err := c.admin.MoveMember(auth.ContextWithBearer(ctx, nonce),
		&chatv1.MoveMemberRequest{
			Room:   room,
			Team:   team,
			Name:   name,
			ToTeam: toTeam,
		})
	if err != nil {
		return callError(err)
	}
	return RenderMoved(w, resp.GetMember())
}

// RegisterMember registers a member no provider can vouch for — a human on the
// host — and prints the token it presents to the member-facing RPCs.
func (c *Client) RegisterMember(
	ctx context.Context,
	w io.Writer,
	identityPath, room, team, name string,
) error {
	nonce, err := c.nonce(ctx, identityPath)
	if err != nil {
		return err
	}
	resp, err := c.admin.RegisterMember(auth.ContextWithBearer(ctx, nonce),
		&chatv1.RegisterMemberRequest{
			Room: room,
			Team: team,
			Name: name,
		})
	if err != nil {
		return callError(err)
	}
	return RenderRegistered(w, resp.GetMember(), resp.GetToken())
}

// AdminSend delivers text into a room the operator does not attend, addressed
// to whoever target names — [ParseAdminTarget] turns the written form into one.
//
// Unlike the member address [Client.MoveMember] takes, the name half is left
// for the daemon to resolve: the target only says which case of the request the
// operator meant, which is what keeps the grammar the same as the one `chat
// send` takes.
func (c *Client) AdminSend(
	ctx context.Context,
	w io.Writer,
	identityPath, room string,
	target AdminTarget,
	text string,
) error {
	delivered, err := c.Admin(identityPath).Send(ctx, room, target, text)
	if err != nil {
		return err
	}
	return RenderAdminSent(w, room, target, delivered)
}

// AdminLog prints the conversation of a room the operator does not attend, the
// tail of it that limit asks for — zero meaning the daemon's own window.
//
// It reads the tail rather than paging: the cursor the RPC takes is there for a
// caller that keeps following the room, and a command run once has nothing to
// carry between runs.
func (c *Client) AdminLog(
	ctx context.Context,
	w io.Writer,
	identityPath, room string,
	limit int32,
) error {
	entries, err := c.Admin(identityPath).RoomLog(ctx, room, 0, limit)
	if err != nil {
		return err
	}
	return RenderAdminHistory(w, entries)
}
