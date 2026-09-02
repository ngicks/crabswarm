package tui

// The two list panes move their cursor with the keys the conversation scrolls
// with, so the movement is written here once and each pane is left with what is
// its own: what enter does with the row the cursor is on.

// listKey moves a cursor over count rows, reporting whether the key was a
// movement at all — a pane is handed back a key it has to answer itself
// otherwise. height is how many rows the pane draws, which is what ctrl+d and
// ctrl+u move half of: a taller pane moves further.
func (m *model) listKey(key string, cursor, count, height int) (int, bool) {
	// The first g of a gg is the one key on the screen that means nothing until
	// the next one arrives; every other key ends the wait.
	pending := m.pendingG
	m.pendingG = false
	switch key {
	case "g":
		if !pending {
			m.pendingG = true
			return cursor, true
		}
		cursor = 0
	case "home":
		cursor = 0
	case "G", "end":
		cursor = count - 1
	case "j", "down":
		cursor++
	case "k", "up":
		cursor--
	case "ctrl+d":
		cursor += max(height/2, 1)
	case "ctrl+u":
		cursor -= max(height/2, 1)
	default:
		return cursor, false
	}
	return clampCursor(cursor, count), true
}

// clampCursor keeps a cursor on a row that is there. A list shrinks under it
// without warning — a member leaves, the daemon stops listing a room — and an
// empty list holds it at zero rather than at nothing.
func clampCursor(cursor, count int) int {
	return min(max(cursor, 0), max(count-1, 0))
}

// window is the stretch of a list a pane that tall can show, scrolled to keep
// the cursor in it: centred where there is list on both sides of it, and at
// either end where there is not. The count of what is not shown is on the
// pane's frame, which is drawn anyway.
func window(lines []string, cursor, height int) []string {
	if height <= 0 || len(lines) <= height {
		return lines
	}
	first := min(max(cursor-height/2, 0), len(lines)-height)
	return lines[first : first+height]
}
