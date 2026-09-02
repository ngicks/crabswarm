package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

// fixtureCrowd is a room with more members than a terminal has lines: teams of
// perTeam members each, which is what the members pane has to fit into its
// share of the screen.
func fixtureCrowd(teams, perTeam int) []*chatv1.Member {
	members := make([]*chatv1.Member, 0, teams*perTeam)
	for team := range teams {
		for member := range perTeam {
			members = append(members, &chatv1.Member{
				Team:  fmt.Sprintf("team%d", team),
				Name:  fmt.Sprintf("member%02d", member),
				Room:  fixtureRoom,
				State: chatv1.HarnessState_HARNESS_STATE_WORKING,
			})
		}
	}
	return members
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

// fixtureModel is the screen as it stands the moment the room lookup answered:
// told who attends and nothing yet of what was said.
func fixtureModel(t *testing.T, deps Deps) *model {
	t.Helper()
	deps.Room = fixtureRoom
	return newModel(t.Context(), deps, fixtureRoster())
}

// press builds the keypress a terminal would have delivered.
func press(code rune, text string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Text: text}
}

// ctrlPress builds one of the four movement keys, which a terminal delivers as
// a control code rather than as text.
func ctrlPress(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Mod: tea.ModCtrl}
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

// A terminal too narrow for both columns shows one of them, and the one it
// opens on is the right: watching the conversation is what the screen is for.
// Whatever its shape, the screen drawn is exactly the terminal's size — a
// screen larger than the terminal is one whose last lines the alternate buffer
// never shows.
func TestANarrowTerminalShowsOneColumnAtATime(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m.entries = fixtureEntries(3)
	m.layout()

	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 30})
	r := m.rects()
	assert.Assert(t, r.leftShown && r.rightShown)
	assert.Equal(t, m.view.Width(), 100-leftWidth-2)
	assert.Equal(t, m.view.Height(), 30-2-(messageRows+2)-2)
	assert.Assert(t, strings.Contains(m.View().Content, "alice"))

	m = update(t, m, tea.WindowSizeMsg{Width: leftMinWidth - 1, Height: 30})
	r = m.rects()
	assert.Assert(t, !r.leftShown && r.rightShown)
	assert.Equal(t, m.view.Width(), leftMinWidth-1-2)
	assert.Assert(t, !strings.Contains(m.View().Content, "members ("))
	assert.Assert(t, strings.Contains(m.View().Content, "line 3"))

	for _, size := range []tea.WindowSizeMsg{
		{Width: 100, Height: 3},
		{Width: 100, Height: 5},
		{Width: 100, Height: 10},
		{Width: 100, Height: 24},
		{Width: 100, Height: 30},
		{Width: leftMinWidth - 1, Height: 24},
		{Width: 20, Height: 24},
	} {
		m = update(t, m, size)
		assert.Equal(t, lipgloss.Height(m.View().Content), size.Height,
			"the screen at %dx%d", size.Width, size.Height)
		assert.Assert(t, lipgloss.Width(m.View().Content) <= size.Width,
			"the screen at %dx%d is %d cells wide",
			size.Width, size.Height, lipgloss.Width(m.View().Content))
	}
}

// A terminal that has not said how big it is — a program driven through a pipe
// — is drawn at the conventional size rather than at nothing.
func TestUnreportedTerminalSizeFallsBackToATerminalShape(t *testing.T) {
	m := fixtureModel(t, Deps{})

	m = update(t, m, tea.WindowSizeMsg{Width: 0, Height: 0})
	r := m.rects()
	assert.Assert(t, r.leftShown && r.rightShown)
	assert.Equal(t, m.view.Width(), defaultWidth-leftWidth-2)
	assert.Equal(t, m.view.Height(), defaultHeight-2-(messageRows+2)-2)
}

// Scrolling up leaves the tail behind and the view stays where it was put while
// the room talks on; scrolling back to the bottom picks the tail up again.
func TestScrollingAwayAndBackReattachesTheTail(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 20})
	m.entries = fixtureEntries(60)
	m.layout()

	assert.Assert(t, m.following)
	assert.Assert(t, strings.Contains(m.View().Content, "line 60"))

	m = update(t, m, press('k', "k"))
	assert.Assert(t, !m.following)
	assert.Assert(t, strings.Contains(m.statusBar(100), "scrolled back"))
	parked := m.view.YOffset()

	// The room says more while the operator is reading older lines. What is on
	// screen must not move under them.
	m.entries = append(m.entries, fixtureEntries(61)[60])
	m.layout()
	assert.Equal(t, m.view.YOffset(), parked)
	assert.Assert(t, !m.following)

	m = update(t, m, press('G', "G"))
	assert.Assert(t, m.following)
	assert.Assert(t, strings.Contains(m.statusBar(100), "tailing"))
	assert.Assert(t, strings.Contains(m.View().Content, "line 61"))

	// Following, the newest entry is on screen the moment it arrives.
	m.entries = append(m.entries, fixtureEntries(62)[61])
	m.layout()
	assert.Assert(t, strings.Contains(m.View().Content, "line 62"))

	// gg is the top of the log and G the bottom of it, as they are in a pager.
	m = update(t, m, press('g', "g"))
	m = update(t, m, press('g', "g"))
	assert.Equal(t, m.view.YOffset(), 0)
	assert.Assert(t, !m.following)
}

// There is no watch mode and no insert mode: there is only the pane the keys
// reach. The letters navigate the conversation because that is what the
// conversation does with them, and are text in the message pane because that is
// what it does with them.
func TestLettersAreTextOnlyInTheFocusedMessagePane(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 20})
	m.entries = fixtureEntries(60)
	m.layout()

	assert.Equal(t, m.focus, focusConversation)
	assert.Assert(t, !m.input.Focused())

	m = update(t, m, ctrlPress('j'))
	assert.Equal(t, m.focus, focusMessage)
	assert.Assert(t, m.input.Focused())
	assert.Equal(t, m.input.Value(), "")

	for _, r := range "q: hi" {
		m = update(t, m, press(r, string(r)))
	}
	assert.Equal(t, m.input.Value(), "q: hi")

	m = update(t, m, ctrlPress('k'))
	assert.Equal(t, m.focus, focusConversation)
	assert.Assert(t, !m.input.Focused())
	// The line survives leaving it: a half-written message is not thrown away
	// by a glance at the scrollback.
	assert.Equal(t, m.input.Value(), "q: hi")

	// And the same letters navigate again rather than being typed into it.
	m = update(t, m, press('k', "k"))
	assert.Equal(t, m.input.Value(), "q: hi")
	assert.Assert(t, !m.following)
}

// Every member is in the members pane with the state that says whether it can
// be interrupted, grouped under the team it belongs to; the count is on the
// frame the pane is drawn in.
func TestMembersPaneListsEveryMemberWithItsState(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 20})

	r := m.rects()
	pane := m.membersPane(r.members.Dx()-2, r.members.Dy()-2)
	for _, want := range []string{
		"backend", "alice", "working", "bob", "waiting", "frontend", "cid", "done",
	} {
		assert.Assert(t, strings.Contains(pane, want),
			"members pane missing %q:\n%s", want, pane)
	}
	assert.Assert(t, strings.Contains(m.View().Content, "members (3)"))
	assert.Assert(t, strings.Contains(m.View().Content, "rooms (1)"))
	// The room being watched is marked in the rooms pane rather than only named
	// on the status bar. The mark carries the colour, so it and the path are
	// two runs of the line rather than one.
	rooms := m.roomsPane(r.rooms.Dx()-2, r.rooms.Dy()-2)
	assert.Assert(t, strings.Contains(rooms, selectedMark),
		"the watched room is not marked:\n%s", rooms)
	assert.Assert(t, strings.Contains(rooms, fixtureRoom),
		"the watched room is not listed:\n%s", rooms)
}

// A room with more members than the pane has lines keeps the screen the
// terminal's size, and says it is showing part of the room rather than looking
// like all of it.
func TestACrowdedMembersPaneDoesNotPushTheChromeOffTheScreen(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m.roster = fixtureCrowd(4, 10)
	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 24})

	view := m.View().Content
	assert.Equal(t, lipgloss.Height(view), 24)

	// The status bar is the last line, the system line the one above it, and
	// the message pane's line sits inside its frame above both.
	lines := strings.Split(view, "\n")
	assert.Assert(t, strings.Contains(lines[len(lines)-1], "room "+fixtureRoom),
		"the status bar is not the last line:\n%s", view)
	assert.Assert(t, strings.Contains(view, m.input.Prompt),
		"the line to type into is not on the screen:\n%s", view)

	// 14 of the 40 members fit the pane at this height; the rest are counted.
	assert.Assert(t, strings.Contains(view, "members (40)"))
	assert.Assert(t, strings.Contains(view, "… +26 more"),
		"the cut members are not counted:\n%s", view)

	for _, size := range []tea.WindowSizeMsg{
		{Width: 100, Height: 3},
		{Width: 100, Height: 5},
		{Width: 100, Height: 10},
		{Width: 100, Height: 30},
		{Width: 100, Height: 60},
		{Width: leftMinWidth - 1, Height: 24},
	} {
		m = update(t, m, size)
		assert.Equal(t, lipgloss.Height(m.View().Content), size.Height,
			"the screen at %dx%d", size.Width, size.Height)
	}
}

// ctrl+c leaves from anywhere, including from the line being written; q leaves
// from the three panes that are lists, where it is not text.
func TestTheQuitKeysQuit(t *testing.T) {
	for _, tc := range []struct {
		name     string
		focus    paneFocus
		key      tea.KeyPressMsg
		wantQuit bool
	}{
		{name: "q in the conversation", focus: focusConversation,
			key: press('q', "q"), wantQuit: true},
		{name: "q in the rooms pane", focus: focusRooms,
			key: press('q', "q"), wantQuit: true},
		{name: "q in the members pane", focus: focusMembers,
			key: press('q', "q"), wantQuit: true},
		{name: "ctrl+c in the conversation", focus: focusConversation,
			key: ctrlPress('c'), wantQuit: true},
		{name: "ctrl+c while writing", focus: focusMessage,
			key: ctrlPress('c'), wantQuit: true},
		// q is a letter where letters are text, and esc is no longer a mode to
		// leave: there is no mode.
		{name: "q while writing", focus: focusMessage, key: press('q', "q")},
		{name: "esc in the conversation", focus: focusConversation,
			key: press(tea.KeyEscape, "")},
		{name: "esc while writing", focus: focusMessage, key: press(tea.KeyEscape, "")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := fixtureModel(t, Deps{})
			m.setFocus(tc.focus)

			m, cmd := enterKey(t, m, tc.key)
			if !tc.wantQuit {
				assert.Equal(t, m.focus, tc.focus, "%s moved the focus", tc.name)
				if cmd == nil {
					return
				}
				_, quits := cmd().(tea.QuitMsg)
				assert.Assert(t, !quits, "%s asked to quit", tc.name)
				return
			}
			assert.Assert(t, cmd != nil, "%s asked for nothing", tc.name)
			_, quits := cmd().(tea.QuitMsg)
			assert.Assert(t, quits, "%s asked for %T, want a quit", tc.name, cmd())
		})
	}
}

// enterKey delivers one keypress and hands back what the model asked for.
func enterKey(t *testing.T, m *model, key tea.KeyPressMsg) (*model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(key)
	updated, ok := next.(*model)
	assert.Assert(t, ok, "update returned %T", next)
	return updated, cmd
}

// The transcript reads the way the CLI's does — who said it, who to, and what —
// so an operator comparing the screen with `chat admin log` reads one text.
func TestConversationSpellsSenderAddressee(t *testing.T) {
	m := fixtureModel(t, Deps{})
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
