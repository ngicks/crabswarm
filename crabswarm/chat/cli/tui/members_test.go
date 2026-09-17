package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// The cursor stops on team headings and on members alike, and enter on a member
// writes their address in front of the message, landing the operator in the
// message pane, which is where they were going.
func TestMembersPaneEnterAddressesTheRowUnderTheCursor(t *testing.T) {
	for _, tc := range []struct {
		name string
		down int
		want string
	}{
		{name: "the member under the first heading", down: 1, want: "@backend/alice "},
		{name: "the next member", down: 2, want: "@backend/bob "},
		{name: "the next team's member", down: 4, want: "@frontend/cid "},
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

// A team is not something a message can be addressed to, so enter on a heading
// writes nothing and says why: the operator names the members under it, or the
// whole room.
func TestMembersPaneEnterOnATeamHeadingSaysATeamIsNotATarget(t *testing.T) {
	for _, down := range []int{0, 3} {
		m := fixtureModel(t, Deps{})
		m.setFocus(focusMembers)
		for range down {
			m = update(t, m, press('j', "j"))
		}

		m = update(t, m, press(tea.KeyEnter, ""))
		assert.Equal(t, m.text.Value(), "")
		assert.Equal(t, m.focus, focusMembers)
		assert.Assert(t, strings.Contains(m.notice, "a team is not a target"),
			"the system line = %q", m.notice)
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

// A member's row says what it runs and by what route a mention gets to it,
// beside what it is and what it is doing. An operator watching a room reads off
// one line whether the member is even typed at, which is what tells a nudge
// that never arrived from one that was never sent.
func TestARosterRowSpellsTheHarnessAndTheDelivery(t *testing.T) {
	for _, tc := range []struct {
		name   string
		member *chatv1.Member
		want   string
	}{
		{
			name: "an agent its server delivers to",
			member: &chatv1.Member{
				Team: "backend", Name: "nina", Room: fixtureRoom,
				Kind:    chatv1.MemberKind_MEMBER_KIND_AGENT,
				State:   chatv1.HarnessState_HARNESS_STATE_DONE,
				Harness: chatv1.Harness_HARNESS_CLAUDE_CODE,
				Nudge:   chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE,
			},
			want: " nina       agent done claude-code native",
		},
		{
			// A person runs no harness and is never pushed to, so both columns
			// say nothing was declared rather than naming something they are not.
			name: "a person",
			member: &chatv1.Member{
				Team: "humans", Name: "yuki", Room: fixtureRoom,
				Kind:  chatv1.MemberKind_MEMBER_KIND_HUMAN,
				State: chatv1.HarnessState_HARNESS_STATE_DONE,
			},
			want: " yuki       human done - -",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := rosterRow{team: tc.member.GetTeam(), member: tc.member}
			assert.Equal(t, row.text(), tc.want)
		})
	}
}

// A name wider than its column is cut, and what the pane is read for still
// makes the row: whether the member can be interrupted right now. A name
// allowed to push the row wider would take exactly that off the screen when the
// pane clips, since what follows the state is what a narrow pane drops first.
func TestALongNameDoesNotPushTheStateOffTheRow(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m.roster = fixtureDerivedName()
	m.layout()

	r := m.rects()
	pane := m.membersPane(r.members.Dx()-2, r.members.Dy()-2)
	lines := strings.Split(pane, "\n")

	// The head of the name goes rather than the tail: two agents of one team are
	// named after the same kind and differ only in the token behind it.
	assert.Equal(t, lines[1], " …-4f2c8a1b agent working")
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
