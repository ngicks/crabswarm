// Package notify wakes a member whose harness has finished its turn, so a
// message that just mentioned them is read now rather than whenever they next
// happen to look.
//
// It holds the daemon's two halves of that: [SendKeys], the implementor of the
// chat broker's notification hook, which types a notice into an agent's
// terminal; and [ScreenPoller], which reads the terminal of an attending
// session whose harness nobody recognised and records the state it shows. The
// poller is what keeps the guard [SendKeys] nudges behind honest for such a
// member, since nothing else says whether its turn has ended. Every harness
// this daemon names reports on a feed of its own, so nothing here polls it.
//
// The notification interface itself is declared at its consumer, in the chat
// package, and the terminal machinery both halves are built on lives in
// ../internal/cmdman.
package notify

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ngicks/crabswarm/crabswarm/chat"
	"github.com/ngicks/crabswarm/crabswarm/chat/internal/cmdman"
	"github.com/ngicks/crabswarm/crabswarm/chat/nudge"
)

// staleStateAfter is how long a reported working or waiting state is believed.
// A state only changes when a harness hook reports the change, and a hook can
// go missing — the user interrupts the session, or the harness has no idle
// notification to hook in the first place — which would leave the member busy
// forever and never nudged again. Past this, the report is treated as no
// longer describing the terminal, and the screen snapshot in
// [cmdman.Terminal.SendCommand] is what still stands between the nudge and a
// terminal that is busy after all.
const staleStateAfter = 10 * time.Minute

// SendKeys wakes an agent by typing a line into its terminal with a
// [cmdman.Terminal].
//
// Typing into a terminal is only safe while that terminal is waiting for a
// command, so a nudge passes three guards — the member is an agent, its last
// reported harness state invites one (see [nudgeable]), and a snapshot of its
// screen shows no dialog. A guard that declines drops the nudge and reports
// success: the message is already in the room and unread for the recipient, so
// it reads it at the end of its current turn instead of a moment from now.
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
	// and the reason not to is that the message already waits past its read
	// position.
	if !nudgeable(recipient) {
		n.logger.Debug("chat: not nudging a busy member",
			"recipient", recipient.Team+"/"+recipient.Name,
			"state", recipient.State,
			"reportedAt", recipient.StateReportedAt)
		return nil
	}

	err := n.terminal.SendCommand(ctx, recipient, nudgeLine(from))
	if errors.Is(err, cmdman.ErrDeclined) {
		// A declined nudge is reported as success: the message is already in
		// the room and unread for the recipient, so it reads it at the end of
		// its current turn instead of a moment from now.
		return nil
	}
	return err
}

// nudgeable reports whether the member's last state report invites a nudge.
//
// Done does. Working and waiting do not, until the report has gone stale: past
// [staleStateAfter] a member is likelier wedged by a state change nobody
// reported than still on the same turn, and a member that is never nudged
// again is the worse outcome — the screen snapshot the injection takes still
// stops a nudge landing in a terminal that is busy after all. Any other state,
// none reported included, is one this notifier cannot read, so it declines
// rather than guess: an unset state has no report time to have gone stale.
func nudgeable(m chat.Member) bool {
	switch m.State {
	case chat.StateDone:
		return true
	case chat.StateWorking, chat.StateWaiting:
		return time.Since(m.StateReportedAt) >= staleStateAfter
	default:
		return false
	}
}

// nudgeLine is the line typed into the recipient's terminal: who has written,
// and what hands the message over. The words are the room's own, shared with
// the servers that deliver a mention through a harness's notification channel
// instead of through a terminal.
func nudgeLine(from chat.Sender) string {
	return nudge.NewMessage(nudge.Sanitize(nudge.Address(from.Team, from.Name)))
}
