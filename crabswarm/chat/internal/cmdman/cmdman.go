// Package cmdman types lines into a member's terminal through the cmdman CLI.
//
// It is the terminal machinery ../../notify is built on — the injection a
// nudge needs, and the snapshots the screen poller reads a member's state off —
// kept here rather than beside them so that package holds only what acts on a
// member. The chat package is consumed, never the other way round: the notifier
// and the poller each compose a [Terminal], and nothing in chat knows this
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

// submitDelay is how long [Terminal.SendCommand] waits between typing the line
// and pressing Enter.
//
// Codex's TUI classifies characters that arrive within 8ms of each other as a
// paste once three of them have, and for 120ms after such a burst it inserts a
// newline for Enter instead of submitting (codex-rs/tui/src/bottom_pane/
// paste_burst.rs: PASTE_BURST_CHAR_INTERVAL, PASTE_ENTER_SUPPRESS_WINDOW). A
// line handed to cmdman lands in one write, so it always reads as a paste, and
// an Enter sent straight after it left the nudge sitting in the composer with a
// trailing newline. The delay outlasts that window with room for scheduling
// jitter; Claude Code has no such heuristic and is unaffected by the wait.
const submitDelay = 200 * time.Millisecond

// ErrDeclined reports that a guard stopped the send before cmdman typed
// anything: the member has no terminal to type into, or its terminal is in no
// state to be typed into. Callers that treat a declined send as an ordinary
// outcome match it with [errors.Is]; every other error means the send itself
// failed.
var ErrDeclined = errors.New("declined to type into the member's terminal")

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
// is one cmdman can take, and a snapshot of its screen shows neither a dialog
// nor a running turn. That last one is a best-effort text scan against
// [ScreenMarkers]: what a busy screen looks like is whatever the harness
// happens to paint today, so the scan catches the obvious cases and is revised
// as those UIs change. It is not a guarantee that the terminal is idle.
//
// A guard that declines logs why and returns an error wrapping [ErrDeclined],
// leaving the terminal untouched. Any other error means cmdman itself failed.
func (t *Terminal) SendCommand(ctx context.Context, member chat.Member, line string) error {
	who := member.Team + "/" + member.Name

	// Not "== KindHuman": a member kind this package has never heard of has no
	// terminal it may type into either. Only an attendee that declared itself a
	// harness is typed at — anything else reads the room when it chooses to,
	// and a line typed into it would land wherever its shell happens to be.
	if member.Kind != chat.KindAgent {
		// Warn, not Debug: this member will never be nudged, and without the
		// line the operator sees only a message waiting unread.
		t.logger.Warn("chat: not typing into a member that runs no harness",
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

	snapshot, err := t.CaptureScreen(ctx, member.Token)
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
		t.logger.Info("chat: not typing, terminal is busy",
			"member", who, "marker", marker)
		return fmt.Errorf("terminal is busy: %w", ErrDeclined)
	}

	// Text and submit go in separate invocations, with [submitDelay] between
	// them. A terminal handed the line and the Enter key in one send-keys
	// treats the trailing key as part of the pasted text rather than as a
	// keypress, and one handed the Enter too soon after the line may do the
	// same; either way the line is never submitted.
	if err := t.sendKeys(ctx, member.Token, line); err != nil {
		return err
	}
	// Not swallowed: a line typed but never submitted sits in the recipient's
	// prompt, where the next thing typed runs it. That is worth an error even
	// though the text did land.
	if err := t.waitBeforeSubmit(ctx); err != nil {
		return err
	}
	return t.sendKeys(ctx, member.Token, "Enter")
}

// CaptureScreen snapshots what the terminal of the command named by token is
// showing.
//
// The visible screen with no line range, not the scrollback: it is what a
// dialog is painting right now, so a marker from some long-finished dialog
// cannot linger in the scan the way it could in a log replay. Only a command
// running under a TTY has a screen; for one without, cmdman errors here and the
// caller reads that as any other unavailable snapshot and declines.
//
// It bounds nothing itself: [Terminal.SendCommand] has already put its send
// under one deadline by the time it gets here, and a watcher polling screens of
// its own accord bounds each capture the way its interval asks for.
func (t *Terminal) CaptureScreen(ctx context.Context, token string) (string, error) {
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

// waitBeforeSubmit sleeps [submitDelay], or less when ctx ends first. Running
// out of time here leaves the line typed and unsubmitted, which the error says.
func (t *Terminal) waitBeforeSubmit(ctx context.Context) error {
	select {
	case <-time.After(submitDelay):
		return nil
	case <-ctx.Done():
		return fmt.Errorf("line typed but not submitted: %w", ctx.Err())
	}
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
