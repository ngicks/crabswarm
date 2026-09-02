package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"gotest.tools/v3/assert"
)

// The cursor stops on team headings and on members alike, and enter writes what
// it is on in front of the message: a member by name, a whole team by its
// heading. Either way the operator lands in the message pane, which is where
// they were going.
func TestMembersPaneEnterAddressesTheRowUnderTheCursor(t *testing.T) {
	for _, tc := range []struct {
		name string
		down int
		want string
	}{
		{name: "a team heading is the whole team", down: 0, want: "@backend/* "},
		{name: "the member under it", down: 1, want: "@backend/alice "},
		{name: "the next member", down: 2, want: "@backend/bob "},
		{name: "the next team's heading", down: 3, want: "@frontend/* "},
		{name: "its member", down: 4, want: "@frontend/cid "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := fixtureModel(t, Deps{})
			m.setFocus(focusMembers)
			for range tc.down {
				m = update(t, m, press('j', "j"))
			}

			m = update(t, m, press(tea.KeyEnter, ""))
			assert.Equal(t, m.text.Value(), tc.want)
			assert.Equal(t, m.text.Line(), 0)
			assert.Equal(t, m.text.Column(), len([]rune(tc.want)))
			assert.Equal(t, m.focus, focusMessage)
			assert.Assert(t, m.text.Focused())
		})
	}
}

// The address goes in front of what was already written rather than over it,
// and the cursor lands between the two, where the next word goes.
func TestAddressingKeepsWhatWasAlreadyWritten(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m = typeLine(t, m, "hold the deploy")

	m.setFocus(focusMembers)
	m = update(t, m, press('j', "j"))
	m = update(t, m, press(tea.KeyEnter, ""))

	assert.Equal(t, m.text.Value(), "@backend/alice hold the deploy")
	assert.Equal(t, m.text.Line(), 0)
	assert.Equal(t, m.text.Column(), len("@backend/alice "))
}

// The members cursor moves the way every list on the screen moves, and stops at
// either end of the room rather than wrapping.
func TestTheMembersCursorMoves(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m.roster = fixtureCrowd(3, 5)
	m.setFocus(focusMembers)
	rows := len(rosterRows(m.roster))

	m = update(t, m, press('G', "G"))
	assert.Equal(t, m.membersCursor, rows-1)
	m = update(t, m, press('j', "j"))
	assert.Equal(t, m.membersCursor, rows-1)

	m = update(t, m, press('g', "g"))
	m = update(t, m, press('g', "g"))
	assert.Equal(t, m.membersCursor, 0)
	m = update(t, m, press('k', "k"))
	assert.Equal(t, m.membersCursor, 0)

	// Half a page is measured off the pane, which holds seventeen rows here.
	m = update(t, m, ctrlPress('d'))
	assert.Equal(t, m.membersCursor, 8)
	m = update(t, m, ctrlPress('u'))
	assert.Equal(t, m.membersCursor, 0)

	// A room nobody is in has no row to be on, and no enter to answer.
	m.roster = nil
	m.layout()
	assert.Equal(t, m.membersCursor, 0)
	m = update(t, m, press(tea.KeyEnter, ""))
	assert.Equal(t, m.text.Value(), "")
}

// The row the cursor is on is picked out, and a team heading keeps the team
// colour where the cursor is somewhere else.
func TestTheMembersCursorIsDrawn(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m.setFocus(focusMembers)
	m = update(t, m, press('j', "j"))

	r := m.rects()
	pane := m.membersPane(r.members.Dx()-2, r.members.Dy()-2)
	lines := strings.Split(pane, "\n")
	assert.Assert(t, strings.Contains(lines[1], "alice"))
	assert.Equal(t, lines[1], pickedStyle.Render(clip(rosterRows(m.roster)[1].text(),
		r.members.Dx()-2)))
	assert.Equal(t, lines[0], teamStyle.Render("backend"))
}
