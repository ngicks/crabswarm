// Package cmdman types lines into a member's terminal through the cmdman CLI.
//
// It is the terminal-injection machinery the chat notifiers are built on, kept
// here rather than beside them so ../../notify holds only implementors of the
// broker's notification hook. The chat package is consumed, never the other way
// round: a notifier composes a [Terminal], and nothing in chat knows this
// package exists.
package cmdman

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/ngicks/crabswarm/crabswarm/chat"
	"github.com/ngicks/crabswarm/crabswarm/chat/resolver"
)

// sendTimeout bounds one whole send — the snapshot and the injections
// together. A caller runs [Terminal.SendCommand] inside the request that
// occasioned it, so a cmdman that hangs would otherwise hang that request.
const sendTimeout = 3 * time.Second

// ErrDeclined reports that a guard stopped the send before cmdman typed
// anything: the member has no terminal to type into, or its terminal is in no
// state to be typed into. Callers that treat a declined send as an ordinary
// outcome match it with [errors.Is]; every other error means the send itself
// failed.
var ErrDeclined = errors.New("declined to type into the member's terminal")

// DialogMarkers are the strings that mean the recipient's terminal is showing a
// dialog rather than an idle prompt: injecting there would answer the dialog
// instead of typing the line. They are heuristics read off Claude Code's
// permission and question UI plus the classic yes/no prompt, matched
// case-insensitively as substrings — updating the set is a one-line edit here.
//
// Exported only so sibling packages' tests can build a snapshot that trips the
// guard without copying a marker; this is an internal package, so the set is
// not public API and stays free to change.
var DialogMarkers = []string{
	"Do you want",
	"❯ 1. Yes",
	"Esc to cancel",
	"esc to interrupt)",
	"(y/n)",
}

// Terminal types a line into a member's terminal through the cmdman CLI. Agents
// run in containers where nothing watches them, and keystrokes are the one
// channel every harness accepts.
type Terminal struct {
	bin    string
	logger *slog.Logger
}

// NewTerminal returns a sender that shells out to the cmdman binary named by
// bin. An empty bin means "cmdman", resolved on PATH, the same default the
// token resolver uses; a nil logger discards logs.
func NewTerminal(bin string, logger *slog.Logger) *Terminal {
	if bin == "" {
		bin = resolver.DefaultCmdmanBin
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Terminal{bin: bin, logger: logger}
}

// Logger returns the logger the terminal logs through, already defaulted. A
// wrapper that logs about its own decisions reads it rather than defaulting the
// caller's argument again, so a nil logger is turned into a discarding one in
// exactly one place.
func (t *Terminal) Logger() *slog.Logger { return t.logger }

// Bin returns the cmdman binary the terminal shells out to, already defaulted.
// Exported only so sibling packages' tests can pin what a wrapper's constructor
// resolved without exec'ing whatever cmdman happens to be on PATH; this is an
// internal package, so it is not public API.
func (t *Terminal) Bin() string { return t.bin }

// SendCommand types line into member's terminal and submits it.
//
// Typing into a terminal is only safe while that terminal is waiting for a
// command, so the send passes three guards — the member is an agent, its token
// is one cmdman can take, and a snapshot of its screen shows no dialog.
// That last one is a best-effort text scan: what a dialog looks like is
// whatever the harness happens to paint today, so the scan catches the obvious
// cases and is revised as those UIs change. It is not a guarantee that the
// terminal is idle.
//
// A guard that declines logs why and returns an error wrapping [ErrDeclined],
// leaving the terminal untouched. Any other error means cmdman itself failed.
func (t *Terminal) SendCommand(ctx context.Context, member chat.Member, line string) error {
	who := member.Team + "/" + member.Name

	// Not "== KindHuman": a member kind this package has never heard of has no
	// terminal it may type into either. Only a joiner that declared itself a
	// harness is typed at — anything else reads its inbox when it chooses to,
	// and a line typed into it would land wherever its shell happens to be.
	if member.Kind != chat.KindAgent {
		t.logger.Debug("chat: not typing into a member that runs no harness",
			"member", who, "kind", member.Kind)
		return fmt.Errorf("member runs no harness: %w", ErrDeclined)
	}
	if err := resolver.ValidateToken(member.Token); err != nil {
		t.logger.Warn("chat: not typing into a member whose token cmdman cannot take",
			"member", who, "err", err)
		return fmt.Errorf("token cmdman cannot take: %w", ErrDeclined)
	}

	// Detached from the request: a client that cancels it mid-flight would
	// otherwise kill cmdman halfway through the injection, leaving a half-typed
	// line in someone's terminal. The timeout replaces the cancellation the
	// request context would have provided.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), sendTimeout)
	defer cancel()

	snapshot, err := t.captureScreen(ctx, member.Token)
	if err != nil {
		// Fail safe: with no snapshot there is no evidence the terminal is at a
		// prompt, and a dropped line costs the caller a retry while a wrong one
		// answers a dialog.
		t.logger.Info("chat: not typing, terminal snapshot unavailable",
			"member", who, "err", err)
		// The exec failure stays out of the chain: it is why the guard could
		// not decide, not something a caller should match through a decline.
		return fmt.Errorf("terminal snapshot unavailable: %w", ErrDeclined)
	}
	if marker, found := dialogMarker(snapshot); found {
		t.logger.Info("chat: not typing, terminal is showing a dialog",
			"member", who, "marker", marker)
		return fmt.Errorf("terminal is showing a dialog: %w", ErrDeclined)
	}

	// Text and submit go in separate invocations. A terminal handed the line
	// and the Enter key in one send-keys treats the trailing key as part of the
	// pasted text rather than as a keypress, and the line is never submitted.
	if err := t.sendKeys(ctx, member.Token, line); err != nil {
		return err
	}
	// Not swallowed: a line typed but never submitted sits in the recipient's
	// prompt, where the next thing typed runs it. That is worth an error even
	// though the text did land.
	return t.sendKeys(ctx, member.Token, "Enter")
}

// captureScreen snapshots what the member's terminal is showing.
//
// The visible screen with no line range, not the scrollback: it is what a
// dialog is painting right now, so a marker from some long-finished dialog
// cannot linger in the scan the way it could in a log replay. Only a command
// running under a TTY has a screen; for one without, cmdman errors here and the
// caller reads that as any other unavailable snapshot and declines.
func (t *Terminal) captureScreen(ctx context.Context, token string) (string, error) {
	// Plain text, no --escapes: attribute sequences would sit inside the strings
	// the scan looks for and split a marker the terminal is showing whole.
	//
	// CombinedOutput: the screen comes back on stdout, and on failure cmdman's
	// own diagnostic on stderr is what makes the error say anything.
	out, err := exec.CommandContext(
		ctx, t.bin, "capture-screen", token,
	).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cmdman capture-screen %q: %w: %s",
			token, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// sendKeys hands cmdman one argument to deliver to the member's terminal.
func (t *Terminal) sendKeys(ctx context.Context, token, arg string) error {
	// "Enter" is a cmdman key name, translated to a carriage return; anything
	// that is not a key name — the line itself — is sent as literal bytes.
	out, err := exec.CommandContext(
		ctx, t.bin, "send-keys", token, arg,
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cmdman send-keys %q %q: %w: %s",
			token, arg, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// dialogMarker returns the first of [DialogMarkers] that snapshot contains.
func dialogMarker(snapshot string) (string, bool) {
	lower := strings.ToLower(snapshot)
	for _, m := range DialogMarkers {
		// Both sides are lowered here rather than keeping the list lowered, so
		// adding a marker stays a copy of what the harness actually prints.
		if strings.Contains(lower, strings.ToLower(m)) {
			return m, true
		}
	}
	return "", false
}
