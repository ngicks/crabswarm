package tui

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/layout"
)

// The screen is two columns of framed panes — rooms over members on the left,
// the conversation over the message textarea on the right — with the system
// line and the status bar running under both. Sizes below are in terminal cells and
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
	// messageMinRows and messageMaxRows bound the message pane, inside its
	// frame: it is as tall as what is written in it, from one row up to six,
	// and scrolls inside itself past that. The conversation pane is what gives
	// the rows up.
	messageMinRows = 1
	messageMaxRows = 6
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
	return rects(width, height, len(m.rooms), m.textRows(), m.showLeft)
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
	if r.rightShown {
		// The textarea takes the whole inside of its frame: it reserves the
		// prompt out of the width it is given. What is written in it then
		// re-wraps, which is a row the message pane may have just taken from
		// the conversation or given back — so the regions are solved again with
		// the height that produced.
		m.text.SetWidth(max(r.message.Dx()-2, 1))
		r = m.rects()
	}
	m.view.SetWidth(max(r.conversation.Dx()-2, 1))
	m.view.SetHeight(max(r.conversation.Dy()-2, 1))
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
