package resolver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// DefaultCmdmanBin is the cmdman binary to shell out to when a caller names
// none. It is expected on PATH (installed via mise, the same assumption the
// preview daemon makes).
const DefaultCmdmanBin = "cmdman"

// composeProjectLabel is the label cmdman-compose stamps on every command it
// brings up, holding the compose project name. A command started outside a
// compose project does not carry it.
const composeProjectLabel = "cmdman.compose.project"

// composeCommandLabel holds the name the compose file declares the command
// under, shared by every replica of it.
const composeCommandLabel = "cmdman.compose.command"

// composeScaleIndexLabel holds the 1-based index that tells one replica of a
// scaled command from another; an unscaled command's sole instance is "1".
// Verified against cmdman v0.0.24, where every compose-created command carries
// it — the name is still derived without it rather than treating its absence
// as an error, since that is a cmdman detail crabswarm does not control.
const composeScaleIndexLabel = "cmdman.compose.scale-index"

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

// commandStates maps what cmdman's `{{.State}}` prints to whether the command's
// process is alive. Recorded from cmdman v0.0.24, which reports "created" before
// a command's first run, "running" while its process lives and "exited" once
// that process ended or was stopped.
//
// A word outside this set draws no verdict at all: it fails the lookup, which
// the caller reports as something to try again. [ErrUnknownToken] refuses an
// attendance outright, so reading anything that is not "running" as gone would
// let a cmdman that renamed or re-cased its states — "Running" — turn away every
// agent trying to attend, while an unrecognised state reports loudly and costs
// nothing but a retry.
var commandStates = map[string]bool{
	"created": false,
	"running": true,
	"exited":  false,
}

// notFoundMessage is the fragment cmdman prints on stderr when the ID|NAME it
// was asked about resolves to nothing.
//
// Verified against cmdman v0.0.24: `cmdman inspect ID --format
// '{{.State}} {{json .Config}}'` prints the state and the command config as
// JSON and exits 0 for a known command, and exits 1 with
// `error: resolve command: no command found matching "ID"` for an unknown one.
// cmdman keeps answering for an exited command until `cmdman rm`, so the
// stderr message alone reports only a removed command, never a dead one.
//
// The match is deliberately narrow rather than "any non-zero exit". Reading
// every failure as unknown would turn a missing cmdman binary or a locked
// cmdman store into a room nobody can attend; if cmdman ever rephrases this
// message the lookup fails loudly instead, which is the recoverable
// direction.
const notFoundMessage = "no command found"

// CmdmanCompose resolves tokens by asking the cmdman CLI about the command a
// token identifies: the command's working directory becomes the room and its
// compose project becomes the team.
//
// It shells out instead of linking cmdman in. cmdman is an external tool that
// owns its own store, and its CLI is the only surface it keeps stable; this
// package is therefore where crabswarm keeps what it knows about that surface.
type CmdmanCompose struct {
	bin string
}

// NewCmdmanCompose returns a resolver that shells out to the cmdman binary
// named by bin. An empty bin means "cmdman", resolved on PATH; tests and
// non-standard installs pass an absolute path.
func NewCmdmanCompose(bin string) *CmdmanCompose {
	if bin == "" {
		bin = DefaultCmdmanBin
	}
	return &CmdmanCompose{bin: bin}
}

// Resolve maps the $CMDMAN_CMD_ID a client reported to the placement of the
// command that runs under it.
//
// It returns an error wrapping [ErrUnknownToken] when the token is malformed,
// when cmdman knows no such command, when the command it names is not running,
// or when that command has no working directory or no compose project — a
// command outside a compose project has no team coordination information, so
// there is nothing to place it against. Every other error means the cmdman
// lookup itself failed.
func (p *CmdmanCompose) Resolve(ctx context.Context, token string) (TeamInfo, error) {
	if err := ValidateToken(token); err != nil {
		return TeamInfo{}, err
	}

	out, err := p.inspectConfig(ctx, token)
	if err != nil {
		return TeamInfo{}, err
	}

	state, configJSON, ok := splitStatePrefix(out)
	if !ok {
		return TeamInfo{}, fmt.Errorf(
			"cmdman inspect %q: output carries no state followed by a config", token)
	}

	// Only the two fields that matter are decoded. cmdman's command config
	// carries a lot more, and mirroring its full shape here would make an
	// unrelated cmdman field addition a crabswarm change.
	var cfg struct {
		Dir    string            `json:"dir"`
		Labels map[string]string `json:"labels"`
	}
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return TeamInfo{}, fmt.Errorf("cmdman inspect %q: decoding config: %w", token, err)
	}

	// The verdict needs both halves of the output to be the shape this code
	// knows: the config has parsed by now, and only a state [commandStates]
	// recognises decides anything. Output cmdman prints differently than it did
	// therefore fails the lookup, which keeps every member, instead of reading
	// as a room full of vanished agents.
	switch running, known := commandStates[state]; {
	case !known:
		return TeamInfo{}, fmt.Errorf(
			"cmdman inspect %q: unrecognised command state %q", token, state)
	case !running:
		return TeamInfo{}, fmt.Errorf(
			"%w: cmdman command %q is %s, not running", ErrUnknownToken, token, state)
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
	// The label values are used verbatim. Compose authors choose them, and a
	// name that cannot be addressed — one carrying the "/" that separates team
	// from name — is rejected when the member attends; sanitizing here would
	// instead hand out a name nobody wrote. A command label is not required:
	// without it there is simply no derived name, and the caller falls back to
	// its own default.
	name := cfg.Labels[composeCommandLabel]
	if name != "" {
		if scaleIndex := cfg.Labels[composeScaleIndexLabel]; scaleIndex != "" {
			name += "-" + scaleIndex
		}
	}
	return TeamInfo{Room: cfg.Dir, Team: project, Name: name}, nil
}

// splitStatePrefix cuts `cmdman inspect` output into the leading state word and
// the config JSON behind it, reporting whether the output had both.
//
// Only the first run of whitespace separates the two. The config is a single
// JSON document that may carry spaces of its own — a working directory or an
// argv entry containing one — so splitting on every space would tear it apart.
func splitStatePrefix(out []byte) (state string, config []byte, ok bool) {
	rest := bytes.TrimLeft(out, " \t\r\n")
	i := bytes.IndexAny(rest, " \t\r\n")
	if i < 0 {
		return "", nil, false
	}
	config = bytes.TrimSpace(rest[i:])
	if len(config) == 0 {
		return "", nil, false
	}
	return string(rest[:i]), config, true
}

// inspectConfig runs `cmdman inspect <token> --format '{{.State}} {{json
// .Config}}'` and returns its stdout, classifying a failure as either
// [ErrUnknownToken] or a genuine lookup error.
//
// The state travels with the config because cmdman answers about a command
// until it is removed: without the state, an exited command looks exactly like
// a running one and the caller keeps vouching for a session that has ended.
func (p *CmdmanCompose) inspectConfig(ctx context.Context, token string) ([]byte, error) {
	// Output, not CombinedOutput: stdout has to stay the state and the JSON, and
	// only Output records stderr on the [exec.ExitError] the classification
	// reads.
	out, err := exec.CommandContext(
		ctx, p.bin, "inspect", token, "--format", "{{.State}} {{json .Config}}",
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

// ValidateToken rejects anything that cannot be a cmdman ID|NAME before it
// becomes an argv entry. Every caller that puts a token on a cmdman command
// line goes through it, not just [CmdmanCompose.Resolve].
//
// A malformed token is wrapped as [ErrUnknownToken] rather than given an error
// kind of its own: it is permanently unresolvable, so the caller should treat
// it exactly like a token cmdman does not know. A separate error kind would
// read as transient and keep such a member around forever.
func ValidateToken(token string) error {
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
