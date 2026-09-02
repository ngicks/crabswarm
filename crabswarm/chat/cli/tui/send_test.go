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

// typeLine moves the focus down into the message pane and types line into it,
// one key at a time, the way the operator does.
func typeLine(t *testing.T, m *model, line string) *model {
	t.Helper()
	m = update(t, m, ctrlPress('j'))
	for _, r := range line {
		m = update(t, m, press(r, string(r)))
	}
	return m
}

// ctrlEnter is the send key a terminal reports where it can tell it from a
// plain enter, and ctrlX the one it sends where it cannot.
func ctrlEnter() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl}
}

func ctrlX() tea.KeyPressMsg { return ctrlPress('x') }

// sendOn sends what is written and hands back what the model asked for.
func sendOn(t *testing.T, m *model) (*model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(ctrlX())
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

// The keys the screen reads are the keys a terminal spells: what the router
// matches on is what the two send keys report themselves as.
func TestTheSendKeysAreSpelledTheWayTheRouterReadsThem(t *testing.T) {
	assert.Equal(t, ctrlEnter().String(), sendKey)
	assert.Equal(t, ctrlX().String(), sendFallbackKey)
}

// The message says who it is for: the first bare `@token` in it, or the whole
// room where there is none. The target reaches the sender as the case it means,
// and the text goes whole — the token included, since it is also the mention.
func TestSendingAddressesTheFirstTokenAndSendsTheTextWhole(t *testing.T) {
	for _, tc := range []struct {
		line   string
		target string
		text   string
	}{
		{"@backend/alice rebase onto main", "backend/alice", "@backend/alice rebase onto main"},
		{"@alice rebase onto main", "alice", "@alice rebase onto main"},
		{"@backend/* rebase onto main", "backend/*", "@backend/* rebase onto main"},
		{"standup in five", "*", "standup in five"},
		{"ask `@here` who owns it", "*", "ask `@here` who owns it"},
		{"@backend/alice ping @backend/bob too", "backend/alice",
			"@backend/alice ping @backend/bob too"},
	} {
		t.Run(tc.line, func(t *testing.T) {
			target, err := cli.ParseAdminTarget(tc.target)
			assert.NilError(t, err)

			sender := &fakeSender{delivered: 1}
			m := fixtureModel(t, Deps{Sender: sender})
			m = typeLine(t, m, tc.line)

			m, cmd := sendOn(t, m)
			m = runCmd(t, m, cmd)

			assert.Equal(t, len(sender.calls), 1)
			assert.Equal(t, sender.calls[0],
				sendCall{room: fixtureRoom, target: target, text: tc.text})
			assert.Equal(t, m.text.Value(), "")
			assert.Assert(t, strings.Contains(m.systemLine(80),
				"sent to "+tc.target+" (1 delivered)"))
		})
	}
}

// enter is a newline and never a send: the operator asked for that in so many
// words. Both send keys send, since a terminal that cannot report ctrl+enter
// reports nothing at all rather than saying so.
func TestEnterWritesANewlineAndTheTwoSendKeysSend(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{name: "ctrl+enter", key: ctrlEnter()},
		{name: "ctrl+x", key: ctrlX()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sender := &fakeSender{delivered: 1}
			m := fixtureModel(t, Deps{Sender: sender})
			m = typeLine(t, m, "@backend/alice first")

			// enter takes the message onto a second line and sends nothing.
			m = update(t, m, press(tea.KeyEnter, ""))
			m = typeLine(t, m, "second")
			assert.Equal(t, m.text.Value(), "@backend/alice first\nsecond")
			assert.Equal(t, len(sender.calls), 0)
			// The pane grew the row the newline asked for, and the
			// conversation gave it up.
			assert.Equal(t, m.textRows(), 2)

			m, cmd := enterKey(t, m, tc.key)
			m = runCmd(t, m, cmd)
			assert.Equal(t, len(sender.calls), 1)
			assert.Equal(t, sender.calls[0].text, "@backend/alice first\nsecond")
			assert.Equal(t, m.text.Value(), "")
			assert.Equal(t, m.textRows(), messageMinRows)
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
	m = typeLine(t, m, "standup in five")

	m, cmd := sendOn(t, m)
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

// A message that addresses half of somebody — or that says nothing at all — is
// refused before anything is dialled, and is left on the screen to be fixed.
func TestAMessageThatAddressesNobodyIsNotSent(t *testing.T) {
	for _, tc := range []struct {
		name string
		line string
	}{
		{name: "a bare @", line: "@"},
		{name: "a bare @ before the message", line: "@ hold the deploy"},
		{name: "a team with no member", line: "@backend/ hi"},
		{name: "a member with no team", line: "@/alice hi"},
		{name: "nothing but spaces", line: "   "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sender := &fakeSender{}
			m := fixtureModel(t, Deps{Sender: sender})
			m = typeLine(t, m, tc.line)

			m, cmd := sendOn(t, m)
			assert.Assert(t, cmd == nil)
			assert.Equal(t, len(sender.calls), 0)
			assert.Equal(t, m.text.Value(), tc.line)
			assert.Assert(t, m.notice != "",
				"the system line says nothing about a refused message")
		})
	}
}

// A delivery the daemon refused is reported and the message handed back, since
// the alternative is retyping it from memory.
func TestAFailedSendHandsTheMessageBack(t *testing.T) {
	sender := &fakeSender{err: errors.New(`no member "ghost" in room /work/proj`)}
	m := fixtureModel(t, Deps{Sender: sender})
	m = typeLine(t, m, "@ghost are you there")

	m, cmd := sendOn(t, m)
	m = runCmd(t, m, cmd)

	assert.Assert(t, strings.Contains(m.systemLine(80), "not sent"))
	assert.Assert(t, strings.Contains(m.systemLine(80), "ghost"))
	assert.Equal(t, m.text.Value(), "@ghost are you there")
	// The cursor is at the end of what came back, where the next word goes.
	assert.Equal(t, m.text.Line(), 0)
	assert.Equal(t, m.text.Column(), len("@ghost are you there"))
}

// A refused message is handed back into an empty pane only: an operator already
// writing the next one keeps what they are writing, and is told on the system
// line that the last one did not go.
func TestAFailedSendLeavesAMessageAlreadyBeingWrittenAlone(t *testing.T) {
	sender := &fakeSender{err: errors.New(`no member "ghost" in room /work/proj`)}
	m := fixtureModel(t, Deps{Sender: sender})
	m = typeLine(t, m, "@ghost are you there")

	// The send is in flight — the pane cleared as it went — and the operator
	// starts the next message before it comes back.
	m, cmd := sendOn(t, m)
	assert.Equal(t, m.text.Value(), "")
	for _, r := range "@alice never mind" {
		m = update(t, m, press(r, string(r)))
	}

	m = runCmd(t, m, cmd)
	assert.Assert(t, strings.Contains(m.systemLine(80), "not sent"))
	assert.Equal(t, m.text.Value(), "@alice never mind")
}

// A refusal that comes back after the operator has moved to another room hands
// the message to the room it was written for rather than into the one being
// written here: the address in it is that room's.
func TestAFailedSendFollowsTheRoomItWasWrittenFor(t *testing.T) {
	sender := &fakeSender{err: errors.New(`no member "ghost" in room /work/proj`)}
	m := fixtureModel(t, Deps{Sender: sender})
	m.rooms = twoRooms()
	m = typeLine(t, m, "@ghost are you there")

	m, cmd := sendOn(t, m)
	m.selectRoom(otherRoom)
	m = runCmd(t, m, cmd)

	assert.Assert(t, strings.Contains(m.systemLine(80), "not sent"))
	assert.Equal(t, m.text.Value(), "")
	assert.Equal(t, m.drafts[fixtureRoom], "@ghost are you there")

	m.selectRoom(fixtureRoom)
	assert.Equal(t, m.text.Value(), "@ghost are you there")
}
