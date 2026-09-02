package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// membersKey is what the members pane does with a key. A cursor over the rows
// and the enter that addresses one come next; leaving the screen is all it
// answers for now, so the key that leaves every list leaves this one.
func (m *model) membersKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "q" {
		return m, tea.Quit
	}
	return m, nil
}

// membersPane renders the room's attendance grouped by team, each member with
// the harness state that says whether it can be interrupted. The count is on
// the frame's title rather than in the pane, since the frame is drawn anyway.
//
// A room with more members than the pane has lines is cut to fit rather than
// run past the bottom of it, and what was cut is counted on the last line, so
// the pane says it is showing part of the room rather than looking like all of
// it.
func (m *model) membersPane(width, height int) string {
	lines := make([]string, 0, len(m.roster))
	// memberRow[i] is the line member i sits on, which is what says how many
	// members a cut at a given line takes off the pane.
	memberRow := make([]int, 0, len(m.roster))
	team := ""
	for _, member := range m.roster {
		if member.GetTeam() != team {
			team = member.GetTeam()
			lines = append(lines, teamStyle.Render(clip(team, width)))
		}
		memberRow = append(memberRow, len(lines))
		lines = append(lines, clip(fmt.Sprintf(" %-*s %s",
			nameColumn, member.GetName(), cli.HarnessStateName(member.GetState())),
			width))
	}
	if height > 0 && len(lines) > height {
		// The last line the pane has goes to the count, so the cut is one line
		// above it.
		kept := height - 1
		shown := 0
		for _, row := range memberRow {
			if row < kept {
				shown++
			}
		}
		lines = append(lines[:kept],
			clip(fmt.Sprintf("… +%d more", len(m.roster)-shown), width))
	}
	return strings.Join(lines, "\n")
}
