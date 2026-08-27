// Package chat brokers the rooms crabswarm participants — agents running in
// cmdman-managed containers and humans on the host — join to talk to each
// other.
package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// ErrUnknownToken reports that a [TeamInfoProvider] cannot place the token it
// was given: either it names nothing the provider knows about, or what it
// names carries no team coordination information.
//
// It is deliberately distinct from a lookup that merely failed. A caller may
// reject the join and reap the member behind an unknown token, but must keep
// the member across a failed lookup — a missing cmdman, a locked store or a
// cancelled context says nothing about whether the token is still valid.
var ErrUnknownToken = errors.New("unknown token")

// TeamInfo is where the holder of an identity token belongs in the chat
// topology.
type TeamInfo struct {
	// Room is the working directory of the command that reported the token.
	// Everything running in the same directory shares a room.
	Room string
	// Team is the name of the compose project the command belongs to. A team
	// is a name namespace inside a room, so a member whose bare name collides
	// with another team's is addressed as "<Team>/<name>".
	Team string
}

// TeamInfoProvider resolves an identity token to the placement of its holder.
//
// Resolve returns an error wrapping [ErrUnknownToken] when the token cannot be
// placed at all; any other error means the lookup itself failed and carries no
// verdict about the token.
type TeamInfoProvider interface {
	Resolve(ctx context.Context, token string) (TeamInfo, error)
}

// defaultCmdmanBin is the cmdman binary [CmdmanComposeProvider] shells out to
// when the caller names none. It is expected on PATH (installed via mise, the
// same assumption the preview daemon makes).
const defaultCmdmanBin = "cmdman"

// composeProjectLabel is the label cmdman-compose stamps on every command it
// brings up, holding the compose project name. A command started outside a
// compose project does not carry it.
const composeProjectLabel = "cmdman.compose.project"

// maxTokenLen bounds a token before it becomes an argv entry. A cmdman ID is
// 32 hex characters and a command name is short; the cap is generous for both
// while keeping an unbounded blob away from the CLI.
const maxTokenLen = 128

// tokenPattern is the conservative character set a token may be drawn from.
// cmdman resolves an "ID|NAME", so the set covers a hex ID and a plain command
// name and nothing else: no whitespace, no path separators, no leading dash.
// The argv is handed to exec directly, so this is not about shell quoting — it
// keeps a hostile token from arriving at cmdman as a flag or a path.
var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// notFoundMessage is the fragment cmdman prints on stderr when the ID|NAME it
// was asked about resolves to nothing.
//
// Verified against cmdman v0.0.23: `cmdman inspect ID --format
// '{{json .Config}}'` prints the command config as JSON and exits 0 for a
// known command, and exits 1 with
// `error: resolve command: no command found matching "ID"` for an unknown one.
//
// The match is deliberately narrow rather than "any non-zero exit". Reading
// every failure as unknown would turn a missing cmdman binary or a locked
// cmdman store into a mass member reap upstream; if cmdman ever rephrases
// this message the join fails loudly instead, which is the recoverable
// direction.
const notFoundMessage = "no command found"

// CmdmanComposeProvider resolves tokens by asking the cmdman CLI about the
// command a token identifies: the command's working directory becomes the room
// and its compose project becomes the team.
//
// It shells out instead of linking cmdman in. cmdman is an external tool that
// owns its own store, and its CLI is the only surface it keeps stable; this
// type is therefore the single place in crabswarm that knows what that surface
// looks like.
type CmdmanComposeProvider struct {
	bin string
}

var _ TeamInfoProvider = (*CmdmanComposeProvider)(nil)

// NewCmdmanComposeProvider returns a provider that shells out to the cmdman
// binary named by bin. An empty bin means "cmdman", resolved on PATH; tests
// and non-standard installs pass an absolute path.
func NewCmdmanComposeProvider(bin string) *CmdmanComposeProvider {
	if bin == "" {
		bin = defaultCmdmanBin
	}
	return &CmdmanComposeProvider{bin: bin}
}

// Resolve maps the $CMDMAN_CMD_ID a client reported to the placement of the
// command that runs under it.
//
// It returns an error wrapping [ErrUnknownToken] when the token is malformed,
// when cmdman knows no such command, or when the command it names has no
// working directory or no compose project — a command outside a compose
// project has no team coordination information, so there is nothing to place
// it against. Every other error means the cmdman lookup itself failed.
func (p *CmdmanComposeProvider) Resolve(
	ctx context.Context,
	token string,
) (TeamInfo, error) {
	if err := validateToken(token); err != nil {
		return TeamInfo{}, err
	}

	out, err := p.inspectConfig(ctx, token)
	if err != nil {
		return TeamInfo{}, err
	}

	// Only the two fields that matter are decoded. cmdman's command config
	// carries a lot more, and mirroring its full shape here would make an
	// unrelated cmdman field addition a crabswarm change.
	var cfg struct {
		Dir    string            `json:"dir"`
		Labels map[string]string `json:"labels"`
	}
	if err := json.Unmarshal(out, &cfg); err != nil {
		return TeamInfo{}, fmt.Errorf("cmdman inspect %q: decoding config: %w", token, err)
	}

	if cfg.Dir == "" {
		return TeamInfo{}, fmt.Errorf(
			"%w: cmdman command %q has no working directory", ErrUnknownToken, token)
	}
	project := cfg.Labels[composeProjectLabel]
	if project == "" {
		return TeamInfo{}, fmt.Errorf(
			"%w: cmdman command %q is not part of a compose project", ErrUnknownToken, token)
	}
	return TeamInfo{Room: cfg.Dir, Team: project}, nil
}

// inspectConfig runs `cmdman inspect <token> --format '{{json .Config}}'` and
// returns its stdout, classifying a failure as either [ErrUnknownToken] or a
// genuine lookup error.
func (p *CmdmanComposeProvider) inspectConfig(
	ctx context.Context,
	token string,
) ([]byte, error) {
	// Output, not CombinedOutput: stdout has to stay pure JSON, and only
	// Output records stderr on the [exec.ExitError] the classification reads.
	out, err := exec.CommandContext(
		ctx, p.bin, "inspect", token, "--format", "{{json .Config}}",
	).Output()
	if err == nil {
		return out, nil
	}

	// Cancellation kills the child, which surfaces as an ExitError carrying
	// whatever stderr the kill left behind. Check the context first so a
	// cancelled lookup is never read as an unknown token.
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, fmt.Errorf("cmdman inspect %q: %w", token, ctxErr)
	}

	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		stderr := strings.TrimSpace(string(exitErr.Stderr))
		if strings.Contains(stderr, notFoundMessage) {
			return nil, fmt.Errorf("%w: cmdman knows no command %q", ErrUnknownToken, token)
		}
		return nil, fmt.Errorf("cmdman inspect %q: %w: %s", token, err, stderr)
	}
	return nil, fmt.Errorf("cmdman inspect %q: %w", token, err)
}

// validateToken rejects anything that cannot be a cmdman ID|NAME before it
// becomes an argv entry.
//
// A malformed token is wrapped as [ErrUnknownToken] rather than given an error
// kind of its own: it is permanently unresolvable, so the caller should treat
// it exactly like a token cmdman does not know. A separate error kind would
// read as transient and keep such a member around forever.
func validateToken(token string) error {
	switch {
	case token == "":
		return fmt.Errorf("%w: empty token", ErrUnknownToken)
	case len(token) > maxTokenLen:
		return fmt.Errorf("%w: token is longer than %d bytes", ErrUnknownToken, maxTokenLen)
	case !tokenPattern.MatchString(token):
		return fmt.Errorf("%w: token %q is not a cmdman ID or name", ErrUnknownToken, token)
	}
	return nil
}
