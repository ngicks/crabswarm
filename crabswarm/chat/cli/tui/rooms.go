package tui

import (
	"fmt"
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

// roomsKey is what the rooms pane does with a key. A cursor over the list and
// the enter that switches the room come next; leaving the screen is all it
// answers for now, so the key that leaves every list leaves this one.
func (m *model) roomsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "q" {
		return m, tea.Quit
	}
	return m, nil
}

// roomsPane lists the rooms the daemon last said it knew, the one on screen
// marked. The list is the roster poll's, so a room that appears while the
// screen is open appears here on the next poll.
//
// The pane asks for its own rows, so it is only cut when a long list has been
// capped at its share of the column; what was cut is counted on the last line
// rather than left looking like the whole list.
func (m *model) roomsPane(width, height int) string {
	lines := make([]string, 0, len(m.rooms))
	for _, room := range m.rooms {
		if room != m.room {
			lines = append(lines, clip(unselectedMark+room, width))
			continue
		}
		// Off the cursor the mark alone is coloured: the path is the room's
		// name, not something the operator picked out.
		lines = append(lines, markStyle.Render(selectedMark)+
			clip(room, max(width-lipgloss.Width(selectedMark), 0)))
	}
	if height > 0 && len(lines) > height {
		kept := max(height-1, 0)
		lines = append(lines[:kept],
			clip(fmt.Sprintf("… +%d more", len(m.rooms)-kept), width))
	}
	return strings.Join(lines, "\n")
}
