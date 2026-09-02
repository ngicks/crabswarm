package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// sentMsg is how an attempted delivery came back. The line is carried along so
// a send that failed can be handed back to the operator instead of vanishing
// with the error.
type sentMsg struct {
	target    cli.AdminTarget
	line      string
	delivered int32
	err       error
}

// submit sends what is on the input line.
//
// The line is cleared as it goes and nothing is added to the conversation:
// what the room said is the log's to say, and the message appears in the pane
// when the next read of the log brings it back, exactly as everyone else's
// does. A send that fails puts the line back, since the operator's alternative
// is to type it again from memory.
func (m *model) submit() tea.Cmd {
	line := m.input.Value()
	to, text, err := cli.ParseAddressedLine(line)
	if err != nil {
		m.notice = err.Error()
		return nil
	}
	// The address is checked here rather than left to the daemon: a half-written
	// one comes back as NotFound, which reads as "nobody by that name" and sends
	// the operator looking for a member instead of for the missing word.
	target, err := cli.ParseAdminTarget(to)
	if err != nil {
		m.notice = err.Error()
		return nil
	}
	m.input.Reset()
	m.notice = "sending to " + target.String()
	// Sending is asking to be read: whatever the operator had scrolled back to,
	// their own message and the answer to it are at the bottom.
	m.following = true
	m.view.GotoBottom()

	ctx, sender, room := m.ctx, m.deps.Sender, m.room
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, callTimeout)
		defer cancel()
		delivered, err := sender.Send(ctx, room, target, text)
		return sentMsg{target: target, line: line, delivered: delivered, err: err}
	}
}

// applySent reports how the delivery went on the system line.
func (m *model) applySent(msg sentMsg) {
	if msg.err != nil {
		m.notice = "not sent: " + msg.err.Error()
		// Only into an empty line: an operator who has already started the next
		// message keeps it.
		if m.input.Value() == "" {
			m.input.SetValue(msg.line)
		}
		return
	}
	m.notice = fmt.Sprintf("sent to %s (%d delivered)", msg.target, msg.delivered)
}
