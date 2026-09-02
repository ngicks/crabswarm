package tui

import (
	"cmp"
	"slices"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

// Which pane the keys reach is the screen's only mode, and this file is the
// whole of it: the four panes, the move between them, and what a move means
// where the terminal is too narrow to show both columns at once.

// paneFocus names the pane that keys reach. There is no watch mode and no
// insert mode: there is only which pane is focused.
type paneFocus int

const (
	focusRooms paneFocus = iota
	focusMembers
	focusConversation
	focusMessage
)

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
// swapped column changes every rectangle on it. The message is focused only
// while it is the pane being typed into, so the letters are text there and
// nowhere else.
func (m *model) setFocus(next paneFocus) tea.Cmd {
	if next == m.focus {
		return nil
	}
	m.pendingG = false
	m.focus = next
	// The dropdown points at a token under a cursor in a pane the keys no
	// longer reach, so it goes with the focus.
	m.closeCompletion()
	m.layout()
	if next == focusMessage {
		return m.text.Focus()
	}
	m.text.Blur()
	return nil
}
