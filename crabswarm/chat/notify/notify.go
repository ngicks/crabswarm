// Package notify wakes a member whose harness has finished its turn, so a
// message that just reached their inbox is read now rather than whenever they
// next happen to look.
//
// It holds the implementors of the chat broker's notification hook, and nothing
// else: the interface itself is declared at its consumer, in the chat package,
// and the terminal-injection machinery they are built on lives in
// ../internal/cmdman. Today the only implementor is [SendKeys].
package notify

import (
	"context"
	"errors"
	"log/slog"
	"unicode"

	"github.com/ngicks/crabswarm/crabswarm/chat"
	"github.com/ngicks/crabswarm/crabswarm/chat/internal/cmdman"
)

// maxNudgeAddrLen caps the sender address the injected line carries. A member
// name comes from the agent that joined and nothing upstream bounds it, so the
// cap is what keeps one line one line; a longer address is cut short.
const maxNudgeAddrLen = 64

// SendKeys wakes an agent by typing a line into its terminal with a
// [cmdman.Terminal].
//
// Typing into a terminal is only safe while that terminal is waiting for a
// command, so a nudge passes three guards — the member is an agent, its last
// reported harness state is done, and a snapshot of its screen shows no
// dialog. A guard that declines drops the nudge and reports success: the
// message is already in the inbox, so the recipient reads it at the end of its
// current turn instead of a moment from now.
type SendKeys struct {
	terminal *cmdman.Terminal
	logger   *slog.Logger
}

var _ chat.Notifier = (*SendKeys)(nil)

// NewSendKeys returns a notifier that shells out to the cmdman binary named by
// bin. An empty bin means "cmdman", resolved on PATH, the same default the
// token resolver uses; a nil logger discards logs.
func NewSendKeys(bin string, logger *slog.Logger) *SendKeys {
	term := cmdman.NewTerminal(bin, logger)
	// The Terminal's logger, not the argument: it has already been defaulted,
	// so a nil logger is turned into a discarding one in exactly one place.
	return &SendKeys{terminal: term, logger: term.Logger()}
}

// Notify types a one-line arrival notice into the recipient's terminal, unless
// one of the guards declines. The message text is not part of that line: it is
// sender-controlled content, and the terminal is the last place to repeat it —
// the notice only says who wrote and how to read.
//
// An error means the injection itself failed. A declined nudge is not an error.
func (n *SendKeys) Notify(
	ctx context.Context,
	recipient chat.Member,
	from chat.Sender,
	_ string,
) error {
	// The state guard is nudge policy rather than a property of typing, so it
	// stays here: a member mid-turn has a terminal that could be typed into,
	// and the reason not to is that its inbox already holds the message.
	if recipient.State != chat.StateDone {
		n.logger.Debug("chat: not nudging a busy member",
			"recipient", recipient.Team+"/"+recipient.Name, "state", recipient.State)
		return nil
	}

	err := n.terminal.SendCommand(ctx, recipient, nudgeLine(from))
	if errors.Is(err, cmdman.ErrDeclined) {
		// A declined nudge is reported as success: the message is already in
		// the inbox, so the recipient reads it at the end of its current turn
		// instead of a moment from now.
		return nil
	}
	return err
}

// nudgeLine is the line typed into the recipient's terminal: who has written,
// and the command that hands the message over.
func nudgeLine(from chat.Sender) string {
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
