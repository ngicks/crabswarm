package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// selectedMark and unselectedMark head a room row, so the room the screen is
// watching is still named while a cursor is somewhere else.
const (
	selectedMark   = "▸ "
	unselectedMark = "  "
)

// roomsKey moves the cursor over the room list and switches the screen to the
// room it is on; leaving the screen is the key that leaves every list.
func (m *model) roomsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	if cursor, moved := m.listKey(
		s, m.roomsCursor, len(m.rooms), m.rects().rooms.Dy()-2,
	); moved {
		m.roomsCursor = cursor
		return m, nil
	}
	switch s {
	case "q":
		return m, tea.Quit
	case "enter":
		if m.roomsCursor < len(m.rooms) {
			m.selectRoom(m.rooms[m.roomsCursor].GetName())
		}
	}
	return m, nil
}

// selectRoom moves the screen to another room.
//
// What was said in the room being left is not this room's conversation, so the
// entries go and the log cursor with them: the next tail poll reads the new
// room from its newest entries, and a poll already in flight for the old room
// is dropped rather than applied here. The members come out of the listing the
// rooms pane is drawn from, which carries every room's attendance, so the pane
// is right at once rather than after the next roster poll.
//
// The half-written message stays with the room it was addressed at — an
// `@team/name` belongs to the room that team is in — and the room's own draft
// takes its place, cursor at the end of it, where the operator left off.
func (m *model) selectRoom(name string) {
	if name == m.room {
		return
	}
	m.drafts[m.room] = m.input.Value()
	m.room = name
	m.input.SetValue(m.drafts[name])
	m.input.CursorEnd()
	m.roster = membersOf(m.rooms, name)
	m.membersCursor = 0
	m.entries = nil
	m.cursor = 0
	m.tailGen++
	// Nothing has been asked of the daemon about this room yet, so whatever it
	// last said about the other one is not this room's news.
	m.tailErr = nil
	// A room is entered at its newest message, which is where the operator who
	// just opened it is looking.
	m.following = true
	m.layout()
}

// roomsPane lists the rooms the daemon last said it knew, the one on screen
// marked and the one under the cursor picked out. The list is the roster
// poll's, so a room that appears while the screen is open appears here on the
// next poll.
//
// More rooms than the pane has lines scrolls rather than cuts: the window
// follows the cursor, so every room can be reached, and how many there are is
// on the pane's frame.
func (m *model) roomsPane(width, height int) string {
	lines := make([]string, 0, len(m.rooms))
	for i, room := range m.rooms {
		name := room.GetName()
		mark := unselectedMark
		if name == m.room {
			mark = selectedMark
		}
		// The path is cut at its head: what tells two rooms of one tree apart
		// is its tail.
		path := clipHead(name, max(width-lipgloss.Width(mark), 0))
		switch {
		case i == m.roomsCursor && m.focus == focusRooms:
			lines = append(lines, pickedStyle.Render(mark+path))
		case mark == selectedMark:
			// Off the cursor the mark alone is coloured: the path is the room's
			// name, not something the operator picked out.
			lines = append(lines, markStyle.Render(mark)+path)
		default:
			lines = append(lines, mark+path)
		}
	}
	return strings.Join(window(lines, m.roomsCursor, height), "\n")
}
