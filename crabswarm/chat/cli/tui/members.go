package tui

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// rosterRow is one line of the members pane. The cursor stops on team headings
// and on members alike, because a heading is where the operator reads which
// team a name below it belongs to; a heading is the row with no member, and the
// one enter has nothing to write, since a team is not a target.
type rosterRow struct {
	team   string
	member *chatv1.Member
}

func (r rosterRow) heading() bool { return r.member == nil }

// address is who enter on this row writes the message to, spelled by the one
// authority on the spelling so a row and a completion offer the same word. A
// heading has none: its callers ask [rosterRow.heading] first.
func (r rosterRow) address() string {
	return cli.Address(r.member)
}

// text is what the row says: a team's name, or a member's beside the kind that
// says whether a message is typed into it at all, the harness state that says
// whether it can be interrupted right now, the harness it runs and the route a
// mention takes to it.
//
// A name longer than [nameColumn] is cut rather than allowed to push the row
// wider, because the pane clips what does not fit and the state would be the
// half that went. It is cut from the front for the reason a room path is: the
// name the daemon derives for an unnamed member is its kind followed by the
// head of its token, so two agents of one team differ only at the tail.
func (r rosterRow) text() string {
	if r.heading() {
		return r.team
	}
	return fmt.Sprintf(" %-*s %s %s %s %s", nameColumn, clipHead(r.member.GetName(), nameColumn),
		cli.MemberKindName(r.member.GetKind()),
		cli.HarnessStateName(r.member.GetState()),
		cli.HarnessName(r.member.GetHarness()),
		cli.NudgeDeliveryName(r.member.GetNudge()))
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
//
// A heading names a team, and a team is not something a message can be
// addressed to: the operator is told so rather than left pressing a key that
// does nothing.
func (m *model) mention(row rosterRow) tea.Cmd {
	if row.heading() {
		m.notice = "a team is not a target: name the members under it, or everyone"
		return nil
	}
	m.text.MoveToBegin()
	m.text.InsertString("@" + row.address() + " ")
	cmd := m.setFocus(focusMessage)
	// setFocus lays the screen out, but only where the focus moved: an operator
	// who was already writing gets the same pre-fill and the same re-solve.
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
