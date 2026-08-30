package notify

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/ngicks/crabswarm/crabswarm/chat"
	"github.com/ngicks/crabswarm/crabswarm/chat/resolver"
)

// sendTimeout bounds one whole send — the snapshot and the injections
// together. A caller runs [Cmdman.SendCommand] inside the request that
// occasioned it, so a cmdman that hangs would otherwise hang that request.
const sendTimeout = 3 * time.Second

// logsTailLines is how much recent terminal output the snapshot guard reads
// back. Enough to hold a permission dialog and the prompt around it, few enough
// that a marker from some long-finished dialog has already scrolled out.
const logsTailLines = 40

// ErrDeclined reports that a guard stopped the send before cmdman typed
// anything: the member has no terminal to type into, or its terminal is in no
// state to be typed into. Callers that treat a declined send as an ordinary
// outcome match it with [errors.Is]; every other error means the send itself
// failed.
var ErrDeclined = errors.New("declined to type into the member's terminal")

// dialogMarkers are the strings that mean the recipient's terminal is showing a
// dialog rather than an idle prompt: injecting there would answer the dialog
// instead of typing the line. They are heuristics read off Claude Code's
// permission and question UI plus the classic yes/no prompt, matched
// case-insensitively as substrings — updating the set is a one-line edit here.
var dialogMarkers = []string{
	"Do you want",
	"❯ 1. Yes",
	"Esc to cancel",
	"esc to interrupt)",
	"(y/n)",
}

// Cmdman types a line into a member's terminal through the cmdman CLI. Agents
// run in containers where nothing watches them, and keystrokes are the one
// channel every harness accepts.
type Cmdman struct {
	bin    string
	logger *slog.Logger
}

// NewCmdman returns a sender that shells out to the cmdman binary named by bin.
// An empty bin means "cmdman", resolved on PATH, the same default the token
// resolver uses; a nil logger discards logs.
func NewCmdman(bin string, logger *slog.Logger) *Cmdman {
	if bin == "" {
		bin = resolver.DefaultCmdmanBin
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Cmdman{bin: bin, logger: logger}
}

// SendCommand types line into member's terminal and submits it.
//
// Typing into a terminal is only safe while that terminal is waiting for a
// command, so the send passes three guards — the member is an agent, its token
// is one cmdman can take, and a snapshot of its recent output shows no dialog.
// That last one is a best-effort text scan: what a dialog looks like is
// whatever the harness happens to paint today, so the scan catches the obvious
// cases and is revised as those UIs change. It is not a guarantee that the
// terminal is idle.
//
// A guard that declines logs why and returns an error wrapping [ErrDeclined],
// leaving the terminal untouched. Any other error means cmdman itself failed.
func (c *Cmdman) SendCommand(ctx context.Context, member chat.Member, line string) error {
	who := member.Team + "/" + member.Name

	// Not "== KindHuman": a member kind this package has never heard of has no
	// terminal it may type into either. A human's token is minted by the daemon
	// and names no cmdman command, so send-keys would fail to resolve it.
	if member.Kind != chat.KindAgent {
		c.logger.Debug("chat: not typing into a member that runs no harness",
			"member", who, "kind", member.Kind)
		return fmt.Errorf("member runs no harness: %w", ErrDeclined)
	}
	if err := resolver.ValidateToken(member.Token); err != nil {
		c.logger.Warn("chat: not typing into a member whose token cmdman cannot take",
			"member", who, "err", err)
		return fmt.Errorf("token cmdman cannot take: %w", ErrDeclined)
	}

	// Detached from the request: a client that cancels it mid-flight would
	// otherwise kill cmdman halfway through the injection, leaving a half-typed
	// line in someone's terminal. The timeout replaces the cancellation the
	// request context would have provided.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), sendTimeout)
	defer cancel()

	snapshot, err := c.tailLogs(ctx, member.Token)
	if err != nil {
		// Fail safe: with no snapshot there is no evidence the terminal is at a
		// prompt, and a dropped line costs the caller a retry while a wrong one
		// answers a dialog.
		c.logger.Info("chat: not typing, terminal snapshot unavailable",
			"member", who, "err", err)
		// The exec failure stays out of the chain: it is why the guard could
		// not decide, not something a caller should match through a decline.
		return fmt.Errorf("terminal snapshot unavailable: %w", ErrDeclined)
	}
	if marker, found := dialogMarker(snapshot); found {
		c.logger.Info("chat: not typing, terminal is showing a dialog",
			"member", who, "marker", marker)
		return fmt.Errorf("terminal is showing a dialog: %w", ErrDeclined)
	}

	// Text and submit go in separate invocations. A terminal handed the line
	// and the Enter key in one send-keys treats the trailing key as part of the
	// pasted text rather than as a keypress, and the line is never submitted.
	if err := c.sendKeys(ctx, member.Token, line); err != nil {
		return err
	}
	// Not swallowed: a line typed but never submitted sits in the recipient's
	// prompt, where the next thing typed runs it. That is worth an error even
	// though the text did land.
	return c.sendKeys(ctx, member.Token, "Enter")
}

// tailLogs snapshots the member terminal's recent output.
//
// cmdman replays its on-disk PTY log; it has no one-shot screen capture, and
// its only live view of the screen is the streaming `attach` protocol, which is
// far too heavy to open for a pre-send check. The replay is therefore the
// closest thing to a screenshot the CLI offers — good enough for a text scan,
// since a dialog paints its markers into the output like everything else.
func (c *Cmdman) tailLogs(ctx context.Context, token string) (string, error) {
	// CombinedOutput: a harness paints its dialogs on whichever stream it
	// likes, and a scan that reads only one of them would miss half of them.
	out, err := exec.CommandContext(
		ctx, c.bin, "logs", "--tail", strconv.Itoa(logsTailLines), token,
	).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cmdman logs %q: %w: %s",
			token, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// sendKeys hands cmdman one argument to deliver to the member's terminal.
func (c *Cmdman) sendKeys(ctx context.Context, token, arg string) error {
	// "Enter" is a cmdman key name, translated to a carriage return; anything
	// that is not a key name — the line itself — is sent as literal bytes.
	out, err := exec.CommandContext(
		ctx, c.bin, "send-keys", token, arg,
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cmdman send-keys %q %q: %w: %s",
			token, arg, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// dialogMarker returns the first of [dialogMarkers] that snapshot contains.
func dialogMarker(snapshot string) (string, bool) {
	lower := strings.ToLower(snapshot)
	for _, m := range dialogMarkers {
		// Both sides are lowered here rather than keeping the list lowered, so
		// adding a marker stays a copy of what the harness actually prints.
		if strings.Contains(lower, strings.ToLower(m)) {
			return m, true
		}
	}
	return "", false
}
