package tui

import (
	"cmp"
	"slices"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/layout"
)

// The screen is two columns of framed panes — rooms over members on the left,
// the conversation over the message line on the right — with the system line
// and the status bar running under both. Sizes below are in terminal cells and
// none of them is added up by hand: the rectangles come from the constraint
// solver, so a resize can neither overlap two panes nor push one off the
// terminal.
const (
	// leftWidth is the left column's column count. Fixed rather than
	// proportional: it holds a room path and a name beside a state word, and
	// none of them gets more readable for being given half the screen.
	leftWidth = 22
	// leftMinWidth is the terminal width below which the two columns no longer
	// fit side by side. Below it the body holds one column at a time and a
	// focus move toward the other brings it on screen in place of this one.
	leftMinWidth = 60
	// roomsMinHeight is the floor the rooms pane keeps when a third of the left
	// column is less than a border and one row.
	roomsMinHeight = 3
	// messageRows is how many rows the message pane writes into, inside its
	// frame. One for now; the multi-line textarea that grows with its contents
	// comes later.
	messageRows = 1
	// nameColumn is how much of a member's name the members pane spells before
	// the state word.
	nameColumn = 10
)

// A terminal that never reported its size — a program driven through a pipe,
// which is how the tests drive it — is drawn at the size a terminal is
// conventionally assumed to have rather than at nothing. Below minWidth ×
// minHeight the panes are solved at the floor and the screen is cut to the
// terminal instead: a solution that small says nothing about the layout, and
// the solver is documented to refuse an area it cannot solve.
const (
	defaultWidth  = 80
	defaultHeight = 24
	minWidth      = 24
	minHeight     = 8
)

// paneFocus names the pane that keys reach. There is no watch mode and no
// insert mode: there is only which pane is focused.
type paneFocus int

const (
	focusRooms paneFocus = iota
	focusMembers
	focusConversation
	focusMessage
)

// paneRects is where one frame put every region. A column a narrow terminal is
// not showing has no rectangle and is not drawn; the flags say which.
type paneRects struct {
	rooms        uv.Rectangle
	members      uv.Rectangle
	conversation uv.Rectangle
	message      uv.Rectangle
	system       uv.Rectangle
	status       uv.Rectangle

	leftShown  bool
	rightShown bool
}

// rects solves the screen: the two bottom lines come off the terminal first,
// then the body is divided into the columns, and each column into its two
// panes. showLeft is read only below [leftMinWidth], where the body holds one
// column and that flag says which.
func rects(width, height, roomCount, textRows int, showLeft bool) paneRects {
	// The system line and the status bar run the whole width under both
	// columns, so they are split off first.
	rows := layout.Vertical(
		layout.Fill(1),
		layout.Len(1),
		layout.Len(1),
	).Split(uv.Rect(0, 0, width, height))

	r := paneRects{system: rows[1], status: rows[2]}
	left, right := rows[0], rows[0]
	switch {
	case width >= leftMinWidth:
		cols := layout.Horizontal(layout.Len(leftWidth), layout.Fill(1)).Split(rows[0])
		left, right = cols[0], cols[1]
		r.leftShown, r.rightShown = true, true
	case showLeft:
		r.leftShown = true
	default:
		r.rightShown = true
	}
	if r.leftShown {
		// The rooms list asks for exactly its own rows and a border, capped at a
		// third of the column so a long list cannot crowd out the members.
		roomsHeight := min(roomCount+2, max(left.Dy()/3, roomsMinHeight))
		col := layout.Vertical(layout.Len(roomsHeight), layout.Fill(1)).Split(left)
		r.rooms, r.members = col[0], col[1]
	}
	if r.rightShown {
		col := layout.Vertical(layout.Fill(1), layout.Len(textRows+2)).Split(right)
		r.conversation, r.message = col[0], col[1]
	}
	return r
}

// rects solves this frame's regions from what the screen currently holds.
func (m *model) rects() paneRects {
	width, height := m.size()
	return rects(width, height, len(m.rooms), messageRows, m.showLeft)
}

// termSize is the terminal's size, or the assumed one until it says.
func (m *model) termSize() (width, height int) {
	width, height = m.width, m.height
	if width <= 0 {
		width = defaultWidth
	}
	if height <= 0 {
		height = defaultHeight
	}
	return width, height
}

// size is what the panes are solved for: the terminal's size, floored at what
// the frames need. A terminal below the floor is drawn cut off rather than
// solved for.
func (m *model) size() (width, height int) {
	width, height = m.termSize()
	return max(width, minWidth), max(height, minHeight)
}

// layout re-sizes the regions to the terminal and refills the conversation,
// keeping the view on the newest entry while it is following the room and both
// list cursors on a row that is there.
func (m *model) layout() {
	m.showColumnWithFocus()
	r := m.rects()
	m.view.SetWidth(max(r.conversation.Dx()-2, 1))
	m.view.SetHeight(max(r.conversation.Dy()-2, 1))
	// One cell under the prompt is the caret's, which the input line draws past
	// its width: asking for the whole remainder makes the line a cell wider
	// than the pane and the caret the cell that falls off its right edge.
	m.input.SetWidth(max(r.message.Dx()-2-lipgloss.Width(m.input.Prompt)-1, 1))
	m.view.SetContent(m.conversation())
	if m.following {
		m.view.GotoBottom()
	}
	// Both lists change under their cursor: a member leaves, the daemon stops
	// listing a room. Clamping here catches every way that happens, since
	// nothing changes either list without the screen being laid out again.
	m.roomsCursor = clampCursor(m.roomsCursor, len(m.rooms))
	m.membersCursor = clampCursor(m.membersCursor, len(rosterRows(m.roster)))
}

// showColumnWithFocus keeps the focused pane on screen. Below the gate the body
// holds one column, and a terminal narrowed while the left column had focus
// would otherwise leave the focus on a pane that is not drawn and that no
// directional move can leave. The column is hidden, never lost.
func (m *model) showColumnWithFocus() {
	if width, _ := m.size(); width >= leftMinWidth {
		return
	}
	m.showLeft = m.focus == focusRooms || m.focus == focusMembers
}

// pane is one focusable region together with the rectangle this frame solved it
// to.
type pane struct {
	focus paneFocus
	rect  uv.Rectangle
}

// panes lists what focus can reach. A hidden column contributes nothing, so no
// direction leads into it — reaching it swaps the columns instead.
func (m *model) panes() []pane {
	r := m.rects()
	var panes []pane
	if r.leftShown {
		panes = append(panes, pane{focusRooms, r.rooms}, pane{focusMembers, r.members})
	}
	if r.rightShown {
		panes = append(panes,
			pane{focusConversation, r.conversation},
			pane{focusMessage, r.message},
		)
	}
	return panes
}

// moveFocus is a plain directional move, and the whole of what ctrl+h/j/k/l
// mean: the target is read off the layout so a changed split moves the keys
// with it. Of the panes lying wholly on that side, the one taken is whichever
// shares the most edge with the focused pane, ties going to the upper — then
// the leftmost — of them. Nothing on that side leaves focus where it was rather
// than wrapping around the screen. There is no pane-to-pane table anywhere.
func (m *model) moveFocus(dir string) tea.Cmd {
	panes := m.panes()
	i := slices.IndexFunc(panes, func(p pane) bool { return p.focus == m.focus })
	if i < 0 {
		return nil
	}
	from := panes[i].rect
	candidates := slices.DeleteFunc(panes, func(p pane) bool {
		return p.focus == m.focus || !beyond(from, p.rect, dir) || shares(from, p.rect, dir) <= 0
	})
	if len(candidates) == 0 {
		return m.swapColumn(from, dir)
	}
	return m.setFocus(pick(from, candidates, dir).focus)
}

// swapColumn answers a horizontal move toward the column a narrow terminal is
// hiding: that column takes the body's place, and focus lands on the pane the
// same overlap rule picks among the panes that arrived. Only the overlap is
// asked for here — the column that just moved no longer lies beyond the one it
// replaced.
func (m *model) swapColumn(from uv.Rectangle, dir string) tea.Cmd {
	if width, _ := m.size(); width >= leftMinWidth {
		return nil
	}
	switch {
	case dir == "ctrl+h" && !m.showLeft:
		m.showLeft = true
	case dir == "ctrl+l" && m.showLeft:
		m.showLeft = false
	default:
		return nil
	}
	panes := m.panes()
	if len(panes) == 0 {
		return nil
	}
	return m.setFocus(pick(from, panes, dir).focus)
}

// pick is the choice moveFocus makes among the panes it may move to.
func pick(from uv.Rectangle, panes []pane, dir string) pane {
	return slices.MaxFunc(panes, func(a, b pane) int {
		return cmp.Or(
			cmp.Compare(shares(from, a.rect, dir), shares(from, b.rect, dir)),
			// Reversed, so the larger of the two is the upper — then the
			// leftmost — pane and the tie breaks that way.
			cmp.Compare(b.rect.Min.Y, a.rect.Min.Y),
			cmp.Compare(b.rect.Min.X, a.rect.Min.X),
		)
	})
}

// beyond reports whether to lies wholly on the dir side of from.
func beyond(from, to uv.Rectangle, dir string) bool {
	switch dir {
	case "ctrl+h":
		return to.Max.X <= from.Min.X
	case "ctrl+l":
		return to.Min.X >= from.Max.X
	case "ctrl+k":
		return to.Max.Y <= from.Min.Y
	case "ctrl+j":
		return to.Min.Y >= from.Max.Y
	}
	return false
}

// shares is how many cells of edge the two rectangles have in common across the
// direction of travel: rows for a sideways move, columns for a vertical one.
func shares(from, to uv.Rectangle, dir string) int {
	if dir == "ctrl+h" || dir == "ctrl+l" {
		return min(from.Max.Y, to.Max.Y) - max(from.Min.Y, to.Min.Y)
	}
	return min(from.Max.X, to.Max.X) - max(from.Min.X, to.Min.X)
}

// setFocus moves the keys to another pane and re-solves the screen, since a
// swapped column changes every rectangle on it. The line is focused only while
// it is the pane being typed into, so the letters are text there and nowhere
// else.
func (m *model) setFocus(next paneFocus) tea.Cmd {
	if next == m.focus {
		return nil
	}
	m.pendingG = false
	m.focus = next
	m.layout()
	if next == focusMessage {
		return m.input.Focus()
	}
	m.input.Blur()
	return nil
}
