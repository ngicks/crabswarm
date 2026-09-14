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

// RoomLog reads a room's conversation, oldest first, through the same filter a
// member read takes — minus its unread cursor, which is measured from a read
// position an operator has none of. It moves no position, so the same stretch
// reads the same way twice.
func (a *AdminClient) RoomLog(
	ctx context.Context,
	room string,
	filter *chatv1.ReadFilter,
) ([]*chatv1.Message, error) {
	nonce, err := a.client.nonce(ctx, a.identity)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.admin.History(auth.ContextWithBearer(ctx, nonce),
		&chatv1.AdminHistoryRequest{Room: room, Filter: filter})
	if err != nil {
		return nil, callError(err)
	}
	return resp.GetMessages(), nil
}

// Send delivers text into room addressed to target — nil for a board post — and
// hands back who it mentioned and which of them is absent.
func (a *AdminClient) Send(
	ctx context.Context,
	room string,
	target *chatv1.Target,
	text string,
) (*chatv1.AdminSendResponse, error) {
	nonce, err := a.client.nonce(ctx, a.identity)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.admin.Send(auth.ContextWithBearer(ctx, nonce),
		&chatv1.AdminSendRequest{Room: room, Target: target, Text: text})
	if err != nil {
		return nil, callError(err)
	}
	return resp, nil
}

// DeleteRoom removes a room's messages and read positions and reports how many
// messages went with it. The daemon refuses while anybody attends the room.
func (a *AdminClient) DeleteRoom(ctx context.Context, room string) (int64, error) {
	nonce, err := a.client.nonce(ctx, a.identity)
	if err != nil {
		return 0, err
	}
	resp, err := a.client.admin.DeleteRoom(auth.ContextWithBearer(ctx, nonce),
		&chatv1.DeleteRoomRequest{Room: room})
	if err != nil {
		return 0, callError(err)
	}
	return resp.GetDeletedMessages(), nil
}

// ListRooms prints every room the daemon knows and who attends it.
func (c *Client) ListRooms(ctx context.Context, w io.Writer, identityPath string) error {
	rooms, err := c.Admin(identityPath).Rooms(ctx)
	if err != nil {
		return err
	}
	return RenderRooms(w, rooms)
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
// to whoever target names — [ParseTarget] turns the written form into one, in
// the same grammar `chat send` takes.
//
// The roles are left for the daemon to resolve. The operator is in no team, so
// a bare name is looked for across the whole room and a name two teams carry is
// refused as ambiguous rather than guessed at.
func (c *Client) AdminSend(
	ctx context.Context,
	out, warn io.Writer,
	identityPath, room string,
	target *chatv1.Target,
	text string,
) error {
	resp, err := c.Admin(identityPath).Send(ctx, room, target, text)
	if err != nil {
		return err
	}
	return RenderSent(out, warn, resp)
}

// AdminLog prints the conversation of a room the operator does not attend, the
// stretch of it the filter asks for.
func (c *Client) AdminLog(
	ctx context.Context,
	w io.Writer,
	identityPath, room string,
	filter *chatv1.ReadFilter,
) error {
	messages, err := c.Admin(identityPath).RoomLog(ctx, room, filter)
	if err != nil {
		return err
	}
	return RenderHistory(w, messages)
}

// DeleteRoom removes a room and everything it holds, and reports how much that
// was. The daemon refuses while anybody attends it: ending those sessions is
// the operator's move, not the daemon's.
func (c *Client) DeleteRoom(
	ctx context.Context,
	w io.Writer,
	identityPath, room string,
) error {
	deleted, err := c.Admin(identityPath).DeleteRoom(ctx, room)
	if err != nil {
		return err
	}
	return RenderDeletedRoom(w, room, deleted)
}
