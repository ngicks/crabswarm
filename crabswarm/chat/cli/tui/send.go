package tui

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// sentMsg is how an attempted delivery came back. The message is carried along
// so a send that failed can be handed back to the operator instead of vanishing
// with the error.
type sentMsg struct {
	// room is where the line was addressed, which is not necessarily the room
	// on screen by the time the answer arrives.
	room   string
	target *chatv1.Target
	line   string
	// absent is the roles the target named that nobody is attending under. The
	// message was accepted either way and waits at their read position.
	absent []*chatv1.Member
	err    error
}

// submit sends what is written in the message pane.
//
// Who it is for is read out of the message itself — its bare `@token`s, or
// nobody where there are none — and the text goes whole, those tokens included,
// so the room reads who was asked.
//
// The pane is cleared as it goes and nothing is added to the conversation: what
// the room said is the log's to say, and the message appears in the pane when
// the next read of the log brings it back, exactly as everyone else's does. A
// send that fails puts the text back — in front of whatever has been typed
// since — because the operator's alternative is to type it again from memory.
func (m *model) submit() tea.Cmd {
	line := m.text.Value()
	if strings.TrimSpace(line) == "" {
		m.notice = "nothing to send"
		return nil
	}
	// The address is read here rather than left to the daemon: a half-written
	// one comes back as NotFound, which reads as "nobody by that name" and sends
	// the operator looking for a member instead of for the missing word.
	target, text, err := parseAddress(line)
	if err != nil {
		m.notice = err.Error()
		return nil
	}
	m.text.Reset()
	m.closeCompletion()
	m.notice = "sending " + addressed(target)
	// Sending is asking to be read: whatever the operator had scrolled back to,
	// their own message and the answer to it are at the bottom.
	m.following = true
	m.view.GotoBottom()
	// The pane has just shrunk back to one row, which the conversation takes.
	m.layout()

	ctx, sender, room := m.ctx, m.deps.Sender, m.room
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, callTimeout)
		defer cancel()
		resp, err := sender.Send(ctx, room, target, text)
		return sentMsg{
			room: room, target: target, line: line,
			absent: resp.GetAbsent(), err: err,
		}
	}
}

// addressed says who a message is for in the words the operator wrote it with,
// so the system line names what they typed. A post names nobody, and saying so
// is the point: it interrupts no one.
func addressed(target *chatv1.Target) string {
	if target == nil {
		return "a post"
	}
	return "to " + cli.TargetString(target)
}

// applySent reports how the delivery went on the system line.
//
// A role nobody is attending under is worth saying: the daemon took the message
// and it waits at that role's read position, so an operator expecting an answer
// is waiting for a session that is not there. The warnings share the one line
// the system line is, since a second would push the whole screen down.
func (m *model) applySent(msg sentMsg) {
	if msg.err != nil {
		m.notice = "not sent: " + msg.err.Error()
		if msg.room != m.room {
			// The operator has moved on. The line goes back to the room it was
			// written for, where it is waiting when they return, rather than
			// into a message being written at another room.
			m.drafts[msg.room] = rejoined(msg.line, m.drafts[msg.room])
			return
		}
		m.text.SetValue(rejoined(msg.line, m.text.Value()))
		m.text.MoveToEnd()
		m.layout()
		return
	}
	notice := []string{"sent " + addressed(msg.target)}
	for _, member := range msg.absent {
		notice = append(notice,
			"warning: "+cli.Address(member)+" is not attending; the mention waits")
	}
	m.notice = strings.Join(notice, "; ")
}

// rejoined puts a refused message back in front of whatever has been written
// since, on a line of its own. Neither half is the screen's to drop: the
// refused text exists only because the daemon would not take it, and the draft
// is what the operator is writing now — one of them going missing is the one
// outcome that cannot be undone by typing.
func rejoined(refused, draft string) string {
	if draft == "" {
		return refused
	}
	return refused + "\n" + draft
}
