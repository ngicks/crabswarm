package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"gotest.tools/v3/assert"
)

// tab is the key that opens and walks the dropdown.
func tab() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyTab} }

// addresses is the list a dropdown offers, in the order it offers them.
func addresses(items []completionItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.address)
	}
	return out
}

// `@prefix` is matched against the whole address and against the bare name,
// since the operator types whichever of the two they remember, and each team is
// offered whole above its own members.
func TestCompletionsOfferTeamsAboveTheirMembers(t *testing.T) {
	for _, tc := range []struct {
		name   string
		prefix string
		want   []string
	}{
		{
			name:   "an empty prefix is the whole room",
			prefix: "",
			want: []string{
				"backend/*", "backend/alice", "backend/bob", "frontend/*", "frontend/cid",
			},
		},
		{
			name:   "a team narrows to it",
			prefix: "backend",
			want:   []string{"backend/*", "backend/alice", "backend/bob"},
		},
		{
			name:   "a qualified prefix narrows further",
			prefix: "frontend/",
			want:   []string{"frontend/*", "frontend/cid"},
		},
		{
			name:   "a bare name matches without its team",
			prefix: "al",
			want:   []string{"backend/alice"},
		},
		{
			name:   "nobody",
			prefix: "zz",
			want:   []string{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.DeepEqual(t, addresses(completions(fixtureRoster(), tc.prefix)), tc.want)
		})
	}
}

// Each row says who it is and what the roster says about them: a member's
// harness state, and how many a whole team would reach — counted, so a team of
// one is not offered as "1 members".
func TestCompletionRowsCarryTheRosterState(t *testing.T) {
	items := completions(fixtureRoster(), "backend")
	assert.Equal(t, items[0], completionItem{address: "backend/*", state: "2 members"})
	assert.Equal(t, items[1], completionItem{address: "backend/alice", state: "working"})
	assert.Equal(t, items[2], completionItem{address: "backend/bob", state: "waiting"})

	items = completions(fixtureRoster(), "frontend")
	assert.Equal(t, items[0], completionItem{address: "frontend/*", state: "1 member"})
}

// One match is not a list — it is the answer — so the first tab applies it,
// with the space that ends the token, and offers nothing.
func TestTabCompletesAUniqueTokenInPlace(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m = typeLine(t, m, "hold it @al")

	m = update(t, m, tab())
	assert.Equal(t, m.text.Value(), "hold it @backend/alice ")
	assert.Assert(t, !m.completion.open)
}

// Several matches are a list. tab and j walk down it, k walks back up, and
// enter takes the row the highlight is on — replacing the token that was typed
// and nothing else of the message.
func TestTabOpensAListAndEnterTakesTheHighlightedRow(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m = typeLine(t, m, "hold it @b")

	m = update(t, m, tab())
	assert.Assert(t, m.completion.open)
	assert.DeepEqual(t, addresses(m.completion.items),
		[]string{"backend/*", "backend/alice", "backend/bob"})
	assert.Equal(t, m.completion.index, 0)

	m = update(t, m, press('j', "j"))
	assert.Equal(t, m.completion.index, 1)
	m = update(t, m, press('k', "k"))
	assert.Equal(t, m.completion.index, 0)
	// j and k are navigation only while the list is open: neither reached the
	// message.
	assert.Equal(t, m.text.Value(), "hold it @b")

	m = update(t, m, press('j', "j"))
	m = update(t, m, press(tea.KeyEnter, ""))
	assert.Equal(t, m.text.Value(), "hold it @backend/alice ")
	assert.Assert(t, !m.completion.open)
}

// tab walks the list and accepts on the last row: a list walked to the end has
// made the choice, and one more key to say so is a key too many.
func TestTabOnTheLastRowAccepts(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m = typeLine(t, m, "@b")

	m = update(t, m, tab())
	m = update(t, m, tab())
	m = update(t, m, tab())
	assert.Equal(t, m.completion.index, 2)
	m = update(t, m, tab())

	assert.Equal(t, m.text.Value(), "@backend/bob ")
	assert.Assert(t, !m.completion.open)
}

// esc closes the list and leaves the token exactly as it was typed, and the
// letters are letters again the moment it is gone.
func TestEscLeavesTheTokenAsTyped(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m = typeLine(t, m, "@b")

	m = update(t, m, tab())
	m = update(t, m, press(tea.KeyEscape, ""))
	assert.Assert(t, !m.completion.open)
	assert.Equal(t, m.text.Value(), "@b")

	m = update(t, m, press('j', "j"))
	assert.Equal(t, m.text.Value(), "@bj")
}

// tab is for tokens: pressed anywhere else it says so rather than offering the
// whole room, and a token nobody answers to says that instead.
func TestTabOutsideATokenReportsInsteadOfOffering(t *testing.T) {
	for _, tc := range []struct {
		name string
		line string
	}{
		{name: "not a token at all", line: "hold the deploy"},
		{name: "a token nobody answers to", line: "@zz"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := fixtureModel(t, Deps{})
			m = typeLine(t, m, tc.line)

			m = update(t, m, tab())
			assert.Assert(t, !m.completion.open)
			assert.Equal(t, m.text.Value(), tc.line)
			assert.Assert(t, m.notice != "", "the system line says nothing")
		})
	}
}

// The list is drawn over what is above the message pane, and it goes with the
// focus: a list pointing at a token under a cursor in another pane points at
// nothing.
func TestTheDropdownIsDrawnAndClosesWithTheFocus(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 24})
	m = typeLine(t, m, "@b")
	m = update(t, m, tab())

	view := m.View().Content
	assert.Assert(t, strings.Contains(view, "@backend/alice"),
		"the dropdown is not on the screen:\n%s", view)
	assert.Assert(t, strings.Contains(view, "2 members"),
		"the team row is not on the screen:\n%s", view)

	m = update(t, m, ctrlPress('k'))
	assert.Equal(t, m.focus, focusConversation)
	assert.Assert(t, !m.completion.open)
	assert.Assert(t, !strings.Contains(m.View().Content, "@backend/alice"))
}

// The list offers the room being watched: switching rooms takes it away with
// the roster it was built from.
func TestSwitchingRoomsClosesTheDropdown(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m.rooms = twoRooms()
	m = typeLine(t, m, "@b")
	m = update(t, m, tab())
	assert.Assert(t, m.completion.open)

	m.selectRoom(otherRoom)
	assert.Assert(t, !m.completion.open)
}
