package chat

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// notifyTimeout bounds one whole notification — the snapshot and the injection
// together. [Service] calls a notifier inside the RPC that delivered the
// message, so a cmdman that hangs would otherwise hang the sender's Send.
const notifyTimeout = 3 * time.Second

// logsTailLines is how much recent terminal output the snapshot guard reads
// back. Enough to hold a permission dialog and the prompt around it, few enough
// that a marker from some long-finished dialog has already scrolled out.
const logsTailLines = 40

// maxNudgeAddrLen caps the sender address the injected line carries. A member
// name comes from the agent that joined and nothing upstream bounds it, so the
// cap is what keeps one line one line; a longer address is cut short.
const maxNudgeAddrLen = 64

// dialogMarkers are the strings that mean the recipient's terminal is showing a
// dialog rather than an idle prompt: injecting there would answer the dialog
// instead of typing a nudge. They are heuristics read off Claude Code's
// permission and question UI plus the classic yes/no prompt, matched
// case-insensitively as substrings — updating the set is a one-line edit here.
var dialogMarkers = []string{
	"Do you want",
	"❯ 1. Yes",
	"Esc to cancel",
	"esc to interrupt)",
	"(y/n)",
}

// SendKeysNotifier wakes an agent by typing a line into its terminal through
// the cmdman CLI: `cmdman send-keys <token> '<line>' Enter`. Agents run in
// containers where nothing watches the inbox for them, and keystrokes are the
// one channel every harness accepts.
//
// Typing into a terminal is only safe while that terminal is waiting for a
// command, so a nudge passes three guards — the member is an agent, its last
// reported harness state is idle, and a snapshot of its recent output shows no
// dialog. A guard that declines drops the nudge and reports success: the
// message is already in the inbox, so the recipient reads it at the end of its
// current turn instead of a moment from now.
type SendKeysNotifier struct {
	bin    string
	logger *slog.Logger
}

var _ Notifier = (*SendKeysNotifier)(nil)

// NewSendKeysNotifier returns a notifier that shells out to the cmdman binary
// named by bin. An empty bin means "cmdman", resolved on PATH, as with
// [NewCmdmanComposeProvider]; a nil logger discards logs.
func NewSendKeysNotifier(bin string, logger *slog.Logger) *SendKeysNotifier {
	if bin == "" {
		bin = defaultCmdmanBin
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &SendKeysNotifier{bin: bin, logger: logger}
}

// Notify types a one-line arrival notice into the recipient's terminal, unless
// one of the guards declines. The message text is not part of that line: it is
// sender-controlled content, and the terminal is the last place to repeat it —
// the notice only says who wrote and how to read.
//
// An error means the injection itself failed. A declined nudge is not an error.
func (n *SendKeysNotifier) Notify(
	ctx context.Context,
	recipient Member,
	from Sender,
	_ string,
) error {
	who := recipient.Team + "/" + recipient.Name

	// Not "== KindHuman": a member kind this notifier has never heard of has no
	// terminal it may type into either. A human's token is minted by the daemon
	// and names no cmdman command, so send-keys would fail to resolve it.
	if recipient.Kind != KindAgent {
		n.logger.Debug("chat: not nudging a member that runs no harness",
			"recipient", who, "kind", recipient.Kind)
		return nil
	}
	if recipient.State != StateIdle {
		n.logger.Debug("chat: not nudging a busy member",
			"recipient", who, "state", recipient.State)
		return nil
	}
	if err := validateToken(recipient.Token); err != nil {
		n.logger.Warn("chat: not nudging a member whose token cmdman cannot take",
			"recipient", who, "err", err)
		return nil
	}

	// Detached from the request: a client that cancels its Send mid-flight
	// would otherwise kill cmdman halfway through the injection, leaving a
	// half-typed line in someone's terminal. The timeout replaces the
	// cancellation the RPC context would have provided.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), notifyTimeout)
	defer cancel()

	snapshot, err := n.tailLogs(ctx, recipient.Token)
	if err != nil {
		// Fail safe: with no snapshot there is no evidence the terminal is at a
		// prompt, and a dropped nudge costs a late read while a wrong one
		// answers a dialog.
		n.logger.Info("chat: not nudging, terminal snapshot unavailable",
			"recipient", who, "err", err)
		return nil
	}
	if marker, found := dialogMarker(snapshot); found {
		n.logger.Info("chat: not nudging, terminal is showing a dialog",
			"recipient", who, "marker", marker)
		return nil
	}

	return n.sendKeys(ctx, recipient.Token, nudgeLine(from))
}

// tailLogs snapshots the recipient terminal's recent output.
//
// cmdman replays its on-disk PTY log; it has no one-shot screen capture, and
// its only live view of the screen is the streaming `attach` protocol, which is
// far too heavy to open for a pre-send check. The replay is therefore the
// closest thing to a screenshot the CLI offers — good enough for a text scan,
// since a dialog paints its markers into the output like everything else.
func (n *SendKeysNotifier) tailLogs(ctx context.Context, token string) (string, error) {
	// CombinedOutput: a harness paints its dialogs on whichever stream it
	// likes, and a scan that reads only one of them would miss half of them.
	out, err := exec.CommandContext(
		ctx, n.bin, "logs", "--tail", strconv.Itoa(logsTailLines), token,
	).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cmdman logs %q: %w: %s",
			token, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// sendKeys types line into the recipient's terminal and submits it.
func (n *SendKeysNotifier) sendKeys(ctx context.Context, token, line string) error {
	// "Enter" is a cmdman key name, translated to a carriage return; anything
	// that is not a key name — the line itself — is sent as literal bytes.
	out, err := exec.CommandContext(
		ctx, n.bin, "send-keys", token, line, "Enter",
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cmdman send-keys %q: %w: %s",
			token, err, strings.TrimSpace(string(out)))
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

// nudgeLine is the line typed into the recipient's terminal: who has written,
// and the command that hands the message over.
func nudgeLine(from Sender) string {
	return "[crabswarm chat] new message from " +
		sanitizeLine(from.Team+"/"+from.Name) +
		" — run: crabswarm chat read"
}

// sanitizeLine makes s safe to type into a terminal as part of one line. It
// drops control characters — a carriage return would submit the line early and
// leave the rest of it running as a command of its own — and truncates at
// [maxNudgeAddrLen]. Neither bound is guaranteed upstream: a name is whatever
// the joining agent asked to be called.
func sanitizeLine(s string) string {
	cleaned := make([]rune, 0, len(s))
	for _, r := range s {
		if unicode.IsControl(r) {
			continue
		}
		cleaned = append(cleaned, r)
		if len(cleaned) == maxNudgeAddrLen {
			break
		}
	}
	return string(cleaned)
}
