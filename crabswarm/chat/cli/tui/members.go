package tui

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// rosterRow is one line of the members pane. The cursor stops on team headings
// and on members alike, because both are addressable — a heading is the whole
// team — so they are the same kind of row, and a heading is the one with no
// member.
type rosterRow struct {
	team   string
	member *chatv1.Member
}

func (r rosterRow) heading() bool { return r.member == nil }

// address is who enter on this row writes the message to.
func (r rosterRow) address() string {
	if r.heading() {
		return r.team + "/*"
	}
	return r.team + "/" + r.member.GetName()
}

// text is what the row says: a team's name, or a member's beside the harness
// state that says whether it can be interrupted.
func (r rosterRow) text() string {
	if r.heading() {
		return r.team
	}
	return fmt.Sprintf(" %-*s %s", nameColumn, r.member.GetName(),
		cli.HarnessStateName(r.member.GetState()))
}

// rosterRows lays the attendance out as the rows the pane draws and the cursor
// moves over. The listing arrives grouped by team, so a team is headed where
// its first member is met.
func rosterRows(roster []*chatv1.Member) []rosterRow {
	rows := make([]rosterRow, 0, len(roster))
	team := ""
	for i, member := range roster {
		if i == 0 || member.GetTeam() != team {
			team = member.GetTeam()
			rows = append(rows, rosterRow{team: team})
		}
		rows = append(rows, rosterRow{team: team, member: member})
	}
	return rows
}

// membersOf is one room's attendance out of a listing. The reply the rooms pane
// is drawn from carries every room's members, so the room switched to has its
// attendance without waiting for a poll of its own. A room the listing does not
// name has none: it is gone, or the daemon has not answered yet.
func membersOf(rooms []*chatv1.Room, room string) []*chatv1.Member {
	i := slices.IndexFunc(rooms, func(r *chatv1.Room) bool {
		return r.GetName() == room
	})
	if i < 0 {
		return nil
	}
	return rooms[i].GetMembers()
}

// membersKey moves the cursor over the room and addresses the row it is on;
// leaving the screen is the key that leaves every list.
func (m *model) membersKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	rows := rosterRows(m.roster)
	s := msg.String()
	if cursor, moved := m.listKey(
		s, m.membersCursor, len(rows), m.rects().members.Dy()-2,
	); moved {
		m.membersCursor = cursor
		return m, nil
	}
	switch s {
	case "q":
		return m, tea.Quit
	case "enter":
		if m.membersCursor < len(rows) {
			return m, m.mention(rows[m.membersCursor])
		}
	}
	return m, nil
}

// mention writes the row's address in front of the message and follows it into
// the message pane, which is where an operator who just picked an addressee was
// going anyway.
//
// In front of it rather than over it: what was already written is the message,
// and the cursor lands between the two, where the next word goes.
func (m *model) mention(row rosterRow) tea.Cmd {
	prefix := "@" + row.address() + " "
	m.input.SetValue(prefix + m.input.Value())
	m.input.SetCursor(utf8.RuneCountInString(prefix))
	cmd := m.setFocus(focusMessage)
	m.layout()
	return cmd
}

// membersPane renders the room's attendance grouped by team, each member with
// the harness state that says whether it can be interrupted and the row under
// the cursor picked out. The count is on the frame's title rather than in the
// pane, since the frame is drawn anyway.
//
// A room with more members than the pane has lines scrolls rather than being
// cut: the window follows the cursor, so every member can be reached, and the
// frame says how many there are.
func (m *model) membersPane(width, height int) string {
	rows := rosterRows(m.roster)
	lines := make([]string, 0, len(rows))
	for i, row := range rows {
		text := clip(row.text(), width)
		switch {
		case i == m.membersCursor && m.focus == focusMembers:
			lines = append(lines, pickedStyle.Render(text))
		case row.heading():
			lines = append(lines, teamStyle.Render(text))
		default:
			lines = append(lines, text)
		}
	}
	return strings.Join(window(lines, m.membersCursor, height), "\n")
}
