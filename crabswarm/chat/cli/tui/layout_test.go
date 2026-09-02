package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// fixtureRooms is a listing of n rooms, the first of them the room the screen
// is watching. The rooms pane asks for its own rows, so how many there are
// moves the boundary every horizontal focus move is measured against.
func fixtureRooms(n int) []*chatv1.Room {
	rooms := make([]*chatv1.Room, 0, n)
	for i := range n {
		if i == 0 {
			rooms = append(rooms, &chatv1.Room{
				Name: fixtureRoom, Members: fixtureRoster()})
			continue
		}
		rooms = append(rooms, &chatv1.Room{Name: fmt.Sprintf("/work/proj%d", i)})
	}
	return rooms
}

// paneModel is the screen at a size, watching a room list of a known length,
// which is what the focus measurements below depend on.
func paneModel(t *testing.T, width, height, roomCount int) *model {
	t.Helper()
	m := fixtureModel(t, Deps{})
	m.rooms = fixtureRooms(roomCount)
	return update(t, m, tea.WindowSizeMsg{Width: width, Height: height})
}

// The regions are solved rather than added up, so what has to hold is
// geometric: every cell of the terminal belongs to exactly one region — the
// panes on screen plus the system line and the status bar — at every size, and
// whichever column a narrow terminal is showing.
func TestPanesTileTheTerminalExactlyOnce(t *testing.T) {
	for _, size := range []struct{ width, height int }{
		{80, 24}, {60, 10}, {59, 24}, {120, 40}, {40, 8},
	} {
		for _, showLeft := range []bool{false, true} {
			t.Run(
				fmt.Sprintf("%dx%d/left=%t", size.width, size.height, showLeft),
				func(t *testing.T) {
					r := rects(size.width, size.height, 4, messageRows, showLeft)
					regions := []uv.Rectangle{r.system, r.status}
					if r.leftShown {
						regions = append(regions, r.rooms, r.members)
					}
					if r.rightShown {
						regions = append(regions, r.conversation, r.message)
					}
					assert.Assert(t, r.leftShown || r.rightShown, "no column is drawn")
					if size.width >= leftMinWidth {
						assert.Assert(t, r.leftShown && r.rightShown,
							"a terminal at or above the gate shows both columns")
					}

					covered := make([]int, size.width*size.height)
					for _, region := range regions {
						for y := region.Min.Y; y < region.Max.Y; y++ {
							for x := region.Min.X; x < region.Max.X; x++ {
								covered[y*size.width+x]++
							}
						}
					}
					for y := range size.height {
						for x := range size.width {
							assert.Equal(t, covered[y*size.width+x], 1,
								"cell %d,%d is in %d regions, want exactly one",
								x, y, covered[y*size.width+x])
						}
					}
				},
			)
		}
	}
}

// ctrl+h/j/k/l are read off the solved rectangles and nothing else: the pane
// they land on is whichever lies on that side and shares the most edge with the
// pane being left. The default split therefore sends ctrl+h from the tall
// conversation to the members pane under the short room list, and a terminal
// short enough to squeeze the room list back to its floor sends the same key to
// the rooms pane instead.
func TestFocusMovesAreReadOffTheLayout(t *testing.T) {
	for _, tc := range []struct {
		name          string
		width, height int
		rooms         int
		from          paneFocus
		dir           string
		want          paneFocus
	}{
		{"log to members at 80x24", 80, 24, 4, focusConversation, "ctrl+h", focusMembers},
		{"log to rooms at 60x10", 60, 10, 4, focusConversation, "ctrl+h", focusRooms},
		{"log to message", 80, 24, 4, focusConversation, "ctrl+j", focusMessage},
		{"message back to the log", 80, 24, 4, focusMessage, "ctrl+k", focusConversation},
		{"rooms to members", 80, 24, 4, focusRooms, "ctrl+j", focusMembers},
		{"members to the log", 80, 24, 4, focusMembers, "ctrl+l", focusConversation},
		{"message to members", 80, 24, 4, focusMessage, "ctrl+h", focusMembers},
		// Nothing on that side leaves the focus where it was: the screen does
		// not wrap around.
		{"nothing above the rooms pane", 80, 24, 4, focusRooms, "ctrl+k", focusRooms},
		{"nothing left of the rooms pane", 80, 24, 4, focusRooms, "ctrl+h", focusRooms},
		{"nothing right of the log", 80, 24, 4, focusConversation, "ctrl+l", focusConversation},
		{"nothing under the message pane", 80, 24, 4, focusMessage, "ctrl+j", focusMessage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := paneModel(t, tc.width, tc.height, tc.rooms)
			m.focus = tc.from
			m.moveFocus(tc.dir)
			assert.Equal(t, m.focus, tc.want)
		})
	}
}

// Below the width gate the body holds one column, and a move toward the other
// brings it on screen in place of this one — with the focus on the pane the
// same overlap rule picks. The column is hidden, not lost.
func TestANarrowTerminalSwapsColumnsOnAFocusMove(t *testing.T) {
	m := paneModel(t, leftMinWidth-1, 24, 4)
	assert.Assert(t, !m.showLeft)
	assert.Assert(t, !m.rects().leftShown)
	assert.Equal(t, m.focus, focusConversation)

	m.moveFocus("ctrl+h")
	assert.Assert(t, m.showLeft, "the left column did not come on screen")
	r := m.rects()
	assert.Assert(t, r.leftShown && !r.rightShown, "both columns are drawn below the gate")
	// The conversation's rows overlap the members pane's more than the short
	// room list's, so that is where the move lands.
	assert.Equal(t, m.focus, focusMembers)
	assert.Assert(t, strings.Contains(m.View().Content, "members ("))
	assert.Assert(t, !strings.Contains(m.View().Content, "conversation"))

	m.moveFocus("ctrl+l")
	assert.Assert(t, !m.showLeft, "the right column did not come back")
	assert.Equal(t, m.focus, focusConversation)
	assert.Assert(t, strings.Contains(m.View().Content, "conversation"))
}

// A terminal narrowed past the gate while the left column had focus keeps that
// column on screen: the focus would otherwise sit on a pane that is not drawn
// and that no directional move could leave.
func TestNarrowingKeepsTheFocusedColumnOnScreen(t *testing.T) {
	m := paneModel(t, 80, 24, 4)
	m.focus = focusMembers

	m = update(t, m, tea.WindowSizeMsg{Width: leftMinWidth - 1, Height: 24})
	assert.Assert(t, m.showLeft)
	assert.Equal(t, m.focus, focusMembers)
	assert.Assert(t, strings.Contains(m.View().Content, "members ("))

	// Wide again, both columns are back and the focus never moved.
	m = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	r := m.rects()
	assert.Assert(t, r.leftShown && r.rightShown)
	assert.Equal(t, m.focus, focusMembers)
}

// A frame is exactly the rectangle it was given, down to the two-row box a
// rooms pane with nothing to list asks for: lipgloss draws an empty body as a
// row of its own, which would put the pane over its neighbour's top edge.
func TestAFrameIsExactlyItsRectangle(t *testing.T) {
	for _, height := range []int{2, 3, 4, 8} {
		frame := box("rooms (0)", fit("", 20, height-2), 22, height, false)
		assert.Equal(t, lipgloss.Height(frame), height, "a box %d rows tall", height)
		assert.Equal(t, lipgloss.Width(frame), 22, "a box %d rows tall", height)
	}

	m := fixtureModel(t, Deps{})
	m.rooms = nil
	m = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 24})
	assert.Equal(t, m.rects().rooms.Dy(), 2)
	assert.Equal(t, lipgloss.Height(m.View().Content), 24)
	// The members pane's own top edge is the row under the empty rooms pane.
	lines := strings.Split(m.View().Content, "\n")
	assert.Assert(t, strings.Contains(lines[2], "members ("),
		"the rooms pane overflowed onto the members pane:\n%s", m.View().Content)
}
