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

// ListRooms prints every room the daemon knows and who attends it.
func (c *Client) ListRooms(ctx context.Context, w io.Writer, identityPath string) error {
	nonce, err := c.nonce(ctx, identityPath)
	if err != nil {
		return err
	}
	resp, err := c.admin.ListRooms(auth.ContextWithBearer(ctx, nonce),
		&chatv1.ListRoomsRequest{})
	if err != nil {
		return callError(err)
	}
	return RenderRooms(w, resp.GetRooms())
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
