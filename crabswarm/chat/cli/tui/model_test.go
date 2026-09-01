package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// The fixtures below stand in for the daemon: the screen is told what a room
// said and who attends it, which is all it ever knows.

const fixtureRoom = "/work/proj"

func fixtureRoster() []*chatv1.Member {
	return []*chatv1.Member{
		{
			Team:  "backend",
			Name:  "alice",
			Room:  fixtureRoom,
			State: chatv1.HarnessState_HARNESS_STATE_WORKING,
		},
		{
			Team:  "backend",
			Name:  "bob",
			Room:  fixtureRoom,
			State: chatv1.HarnessState_HARNESS_STATE_WAITING,
		},
		{
			Team:  "frontend",
			Name:  "cid",
			Room:  fixtureRoom,
			State: chatv1.HarnessState_HARNESS_STATE_DONE,
		},
	}
}

// fixtureEntries builds n conversation entries, each carrying its own number so
// a test can tell which stretch of the room is on screen.
func fixtureEntries(n int) []*chatv1.AdminHistoryEntry {
	sent := time.Date(2026, 9, 2, 9, 0, 0, 0, time.UTC)
	entries := make([]*chatv1.AdminHistoryEntry, 0, n)
	for i := range n {
		entries = append(entries, &chatv1.AdminHistoryEntry{
			Id:     int64(i + 1),
			From:   &chatv1.Member{Team: "backend", Name: "alice", Room: fixtureRoom},
			Text:   fmt.Sprintf("line %d", i+1),
			SentAt: timestamppb.New(sent.Add(time.Duration(i) * time.Second)),
		})
	}
	return entries
}

// press builds the keypress a terminal would have delivered.
func press(code rune, text string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Text: text}
}

// update runs one message through the model, which mutates in place and hands
// itself back.
func update(t *testing.T, m *model, msg tea.Msg) *model {
	t.Helper()
	next, _ := m.Update(msg)
	updated, ok := next.(*model)
	assert.Assert(t, ok, "update returned %T", next)
	return updated
}

// A terminal too narrow for both panes loses the roster, not the conversation:
// the conversation is what the operator is watching, and the roster is a
// second, slower question.
func TestResizeDropsTheRosterBeforeTheConversation(t *testing.T) {
	m := newModel(fixtureRoom, fixtureRoster())
	m.entries = fixtureEntries(3)
	m.layout()

	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	assert.Assert(t, m.rosterShown())
	assert.Equal(t, m.view.Width(), 100-rosterWidth-rosterGap)
	assert.Equal(t, m.view.Height(), 30-chromeHeight)
	assert.Assert(t, strings.Contains(m.View().Content, "alice"))

	m = update(t, m, tea.WindowSizeMsg{Width: rosterMinWidth - 1, Height: 30})
	assert.Assert(t, !m.rosterShown())
	assert.Equal(t, m.view.Width(), rosterMinWidth-1)
	assert.Assert(t, !strings.Contains(m.View().Content, "roster ("))
	assert.Assert(t, strings.Contains(m.View().Content, "line 3"))

	// The chrome is two lines whatever happens, so a terminal shorter than the
	// chrome still leaves the conversation a line to show.
	m = update(t, m, tea.WindowSizeMsg{Width: 40, Height: 1})
	assert.Equal(t, m.view.Height(), 1)
}

// A terminal that has not said how big it is — a program driven through a pipe
// — is drawn at the conventional size rather than at nothing.
func TestUnreportedTerminalSizeFallsBackToATerminalShape(t *testing.T) {
	m := newModel(fixtureRoom, fixtureRoster())

	m = update(t, m, tea.WindowSizeMsg{Width: 0, Height: 0})
	assert.Assert(t, m.rosterShown())
	assert.Equal(t, m.view.Width(), defaultWidth-rosterWidth-rosterGap)
	assert.Equal(t, m.view.Height(), defaultHeight-chromeHeight)
}

// Scrolling up leaves the tail behind and the view stays where it was put while
// the room talks on; scrolling back to the bottom picks the tail up again.
func TestScrollingAwayAndBackReattachesTheTail(t *testing.T) {
	m := newModel(fixtureRoom, fixtureRoster())
	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 12})
	m.entries = fixtureEntries(60)
	m.layout()

	assert.Assert(t, m.following)
	assert.Assert(t, strings.Contains(m.View().Content, "line 60"))

	m = update(t, m, press(tea.KeyUp, ""))
	assert.Assert(t, !m.following)
	assert.Assert(t, strings.Contains(m.statusBar(), "scrolled back"))
	parked := m.view.YOffset()

	// The room says more while the operator is reading older lines. What is on
	// screen must not move under them.
	m.entries = append(m.entries, fixtureEntries(61)[60])
	m.layout()
	assert.Equal(t, m.view.YOffset(), parked)
	assert.Assert(t, !m.following)

	m = update(t, m, press(tea.KeyEnd, ""))
	assert.Assert(t, m.following)
	assert.Assert(t, strings.Contains(m.statusBar(), "tailing"))
	assert.Assert(t, strings.Contains(m.View().Content, "line 61"))

	// Following, the newest entry is on screen the moment it arrives.
	m.entries = append(m.entries, fixtureEntries(62)[61])
	m.layout()
	assert.Assert(t, strings.Contains(m.View().Content, "line 62"))
}

// The input line is out of the way until it is asked for: the letters navigate
// while the screen is being watched and are text once it is focused.
func TestFocusSwitchesLettersBetweenNavigationAndText(t *testing.T) {
	m := newModel(fixtureRoom, fixtureRoster())
	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 12})
	m.entries = fixtureEntries(60)
	m.layout()

	assert.Assert(t, !m.input.Focused())

	m = update(t, m, press('i', "i"))
	assert.Assert(t, m.input.Focused())
	assert.Equal(t, m.input.Value(), "")

	for _, r := range "q: hi" {
		m = update(t, m, press(r, string(r)))
	}
	assert.Equal(t, m.input.Value(), "q: hi")

	m = update(t, m, press(tea.KeyEscape, ""))
	assert.Assert(t, !m.input.Focused())
	// The line survives leaving it: a half-written message is not thrown away
	// by a glance at the scrollback.
	assert.Equal(t, m.input.Value(), "q: hi")
}

// Every member is on the sidebar with the state that says whether it can be
// interrupted, grouped under the team it belongs to.
func TestRosterListsEveryMemberWithItsState(t *testing.T) {
	m := newModel(fixtureRoom, fixtureRoster())
	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 20})

	pane := m.rosterPane(m.view.Height())
	for _, want := range []string{
		"roster (3)", "backend", "alice", "working", "bob", "waiting",
		"frontend", "cid", "done",
	} {
		assert.Assert(t, strings.Contains(pane, want), "roster pane missing %q:\n%s", want, pane)
	}
	for line := range strings.SplitSeq(pane, "\n") {
		assert.Equal(t, len(line), rosterWidth, "roster line %q is not the pane's width", line)
	}
}

// The transcript reads the way the CLI's does — who said it, who to, and what —
// so an operator comparing the screen with `chat admin log` reads one text.
func TestConversationSpellsSenderAddressee(t *testing.T) {
	m := newModel(fixtureRoom, fixtureRoster())
	m.entries = []*chatv1.AdminHistoryEntry{
		{
			Id:     1,
			From:   &chatv1.Member{Team: "admin", Name: "admin", Room: fixtureRoom},
			To:     &chatv1.Member{Team: "backend", Name: "alice", Room: fixtureRoom},
			Text:   "ship it",
			SentAt: timestamppb.New(time.Date(2026, 9, 2, 9, 30, 15, 0, time.UTC)),
		},
		{
			Id:     2,
			From:   &chatv1.Member{Team: "backend", Name: "alice", Room: fixtureRoom},
			Text:   "on it",
			SentAt: timestamppb.New(time.Date(2026, 9, 2, 9, 30, 20, 0, time.UTC)),
		},
	}

	assert.Equal(t, m.conversation(),
		"09:30:15 admin/admin → backend/alice: ship it\n"+
			"09:30:20 backend/alice → *: on it\n")
}
