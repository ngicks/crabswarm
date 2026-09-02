package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

type sendCall struct {
	room   string
	target cli.AdminTarget
	text   string
}

type fakeSender struct {
	calls     []sendCall
	delivered int32
	err       error
}

func (f *fakeSender) Send(
	_ context.Context,
	room string,
	target cli.AdminTarget,
	text string,
) (int32, error) {
	f.calls = append(f.calls, sendCall{room: room, target: target, text: text})
	return f.delivered, f.err
}

// typeLine focuses the input and types line into it, one key at a time, the way
// the operator does.
func typeLine(t *testing.T, m *model, line string) *model {
	t.Helper()
	m = update(t, m, press('i', "i"))
	for _, r := range line {
		m = update(t, m, press(r, string(r)))
	}
	return m
}

// The input line takes the addressing `chat admin send` takes — a whole team
// and the star included — and the message is whatever follows the first colon.
// The address reaches the sender as the case it means, not as the word it was
// written as.
func TestSendingTakesTheSameAddressingAsChatSend(t *testing.T) {
	for _, tc := range []struct {
		line   string
		target string
		text   string
	}{
		{"backend/alice: rebase onto main", "backend/alice", "rebase onto main"},
		{"alice: rebase onto main", "alice", "rebase onto main"},
		{"backend/*: rebase onto main", "backend/*", "rebase onto main"},
		{"*: standup in five", "*", "standup in five"},
		{"alice: see http://host/x: it fails", "alice", "see http://host/x: it fails"},
	} {
		t.Run(tc.line, func(t *testing.T) {
			target, err := cli.ParseAdminTarget(tc.target)
			assert.NilError(t, err)

			sender := &fakeSender{delivered: 1}
			m := fixtureModel(t, Deps{Sender: sender})
			m = typeLine(t, m, tc.line)

			m, cmd := enterOn(t, m)
			m = runCmd(t, m, cmd)

			assert.Equal(t, len(sender.calls), 1)
			assert.Equal(t, sender.calls[0],
				sendCall{room: fixtureRoom, target: target, text: tc.text})
			assert.Equal(t, m.input.Value(), "")
			assert.Assert(t, strings.Contains(m.statusBar(), "sent to "+tc.target))
		})
	}
}

// What the room said is the log's to say: a sent message reaches the pane when
// the next read brings it back, not because the screen put it there.
func TestASentMessageAppearsOnlyWhenTheLogSaysSo(t *testing.T) {
	sender := &fakeSender{delivered: 2}
	log := &fakeLog{
		reply: func(logCall) ([]*chatv1.AdminHistoryEntry, error) { return nil, nil },
	}
	m := fixtureModel(t, Deps{Log: log, Sender: sender})
	m = typeLine(t, m, "*: standup in five")

	m, cmd := enterOn(t, m)
	m = runCmd(t, m, cmd)
	assert.Assert(t, !strings.Contains(m.conversation(), "standup in five"))

	log.reply = func(logCall) ([]*chatv1.AdminHistoryEntry, error) {
		return []*chatv1.AdminHistoryEntry{{
			Id:   9,
			From: &chatv1.Member{Team: "admin", Name: "admin", Room: fixtureRoom},
			Text: "standup in five",
		}}, nil
	}
	m, _ = step(t, m, m.tail())
	assert.Assert(t, strings.Contains(m.conversation(), "standup in five"))
}

// A line that addresses nobody — or addresses half of somebody — is refused
// before anything is dialled, and is left on the screen to be fixed.
func TestALineThatAddressesNobodyIsNotSent(t *testing.T) {
	for _, line := range []string{
		"no colon at all",
		": text without an addressee",
		"alice:",
		"backend/: hi",
		"/alice: hi",
	} {
		t.Run(line, func(t *testing.T) {
			sender := &fakeSender{}
			m := fixtureModel(t, Deps{Sender: sender})
			m = typeLine(t, m, line)

			m, cmd := enterOn(t, m)
			assert.Assert(t, cmd == nil)
			assert.Equal(t, len(sender.calls), 0)
			assert.Equal(t, m.input.Value(), line)
			assert.Assert(t, m.notice != "", "the bar says nothing about a refused line")
		})
	}
}

// A delivery the daemon refused is reported and the message handed back, since
// the alternative is retyping it from memory.
func TestAFailedSendHandsTheLineBack(t *testing.T) {
	sender := &fakeSender{err: errors.New(`no member "ghost" in room /work/proj`)}
	m := fixtureModel(t, Deps{Sender: sender})
	m = typeLine(t, m, "ghost: are you there")

	m, cmd := enterOn(t, m)
	m = runCmd(t, m, cmd)

	assert.Assert(t, strings.Contains(m.statusBar(), "not sent"))
	assert.Assert(t, strings.Contains(m.statusBar(), "ghost"))
	assert.Equal(t, m.input.Value(), "ghost: are you there")
}

// A refused message is handed back into an empty line only: an operator already
// writing the next one keeps what they are writing, and is told on the bar that
// the last one did not go.
func TestAFailedSendLeavesALineAlreadyBeingWrittenAlone(t *testing.T) {
	sender := &fakeSender{err: errors.New(`no member "ghost" in room /work/proj`)}
	m := fixtureModel(t, Deps{Sender: sender})
	m = typeLine(t, m, "ghost: are you there")

	// The send is in flight — the line cleared as it went — and the operator
	// starts the next message before it comes back.
	m, cmd := enterOn(t, m)
	assert.Equal(t, m.input.Value(), "")
	for _, r := range "alice: never mind" {
		m = update(t, m, press(r, string(r)))
	}

	m = runCmd(t, m, cmd)
	assert.Assert(t, strings.Contains(m.statusBar(), "not sent"))
	assert.Equal(t, m.input.Value(), "alice: never mind")
}

// enterOn presses enter and hands back what the model asked for.
func enterOn(t *testing.T, m *model) (*model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(press(tea.KeyEnter, ""))
	updated, ok := next.(*model)
	assert.Assert(t, ok, "update returned %T", next)
	return updated, cmd
}

// runCmd runs a command and feeds back what it produced.
func runCmd(t *testing.T, m *model, cmd tea.Cmd) *model {
	t.Helper()
	m, _ = step(t, m, cmd)
	return m
}
