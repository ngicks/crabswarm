package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

// paneLayer positions a titled, bordered pane at the rectangle the solver gave
// it, so the compositor can put it there without the pane knowing where "there"
// is.
func paneLayer(title, content string, r uv.Rectangle, focused bool) *lipgloss.Layer {
	body := fit(content, r.Dx()-2, r.Dy()-2)
	return lipgloss.NewLayer(box(title, body, r.Dx(), r.Dy(), focused)).
		X(r.Min.X).Y(r.Min.Y)
}

// box draws content inside a rounded border of exactly width×height cells with
// the title written into the top edge — purple and bold while the pane has
// focus, and a shade lighter than its own frame while it does not.
//
// The top edge is drawn by hand because lipgloss has no titled border: the
// style renders every side but the top, and the line the title sits on is
// built to the same width and put back on top.
func box(title, content string, width, height int, focused bool) string {
	if width < 2 || height < 2 {
		return ""
	}
	edge := blurredEdge
	titleStyle := blurredTitleStyle
	if focused {
		edge = focusedEdge
		titleStyle = focusedTitleStyle
	}
	border := lipgloss.RoundedBorder()
	edgeStyle := lipgloss.NewStyle().Foreground(edge)

	label := ""
	if title != "" {
		label = " " + clip(title, max(width-5, 0)) + " "
	}
	fill := max(width-3-lipgloss.Width(label), 0)
	top := edgeStyle.Render(border.TopLeft+border.Top) +
		titleStyle.Render(label) +
		edgeStyle.Render(strings.Repeat(border.Top, fill)+border.TopRight)
	if height == 2 {
		// A box two rows tall is its two edges and nothing between them, which
		// is what a rooms pane with no rooms in it asks for. lipgloss draws an
		// empty body as a row of its own, so the pane would be drawn one row
		// past the rectangle it was given and over its neighbour's top edge.
		return top + "\n" + edgeStyle.Render(
			border.BottomLeft+strings.Repeat(border.Bottom, width-2)+border.BottomRight)
	}
	// No Width/Height on the style: lipgloss measures a block including its
	// border, so a width asked for here would come off the content and the
	// pane would draw two cells narrower than the rectangle it was given.
	// [fit] has already made the content exactly the inside of the box.
	body := lipgloss.NewStyle().
		Border(border).BorderTop(false).BorderForeground(edge).
		Render(content)
	return top + "\n" + body
}

// fit makes a block exactly width×height cells, since a pane that outgrows its
// rectangle is drawn over its neighbour.
//
// Lines are cut and padded one at a time rather than by a style's Width, which
// re-wraps what the viewport already wrapped and returns a taller block than it
// was given. MaxWidth does the cutting because the blocks passed through here —
// the viewport, the input line, the roster — carry styling that a rune-counting
// cut would break in the middle of an escape sequence. The padding is what
// keeps a pane whose last row is blank the size it was solved to: the canvas
// trims trailing spaces on render, and a short block would otherwise let the
// frame under it show through.
func fit(s string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	cut := lipgloss.NewStyle().MaxWidth(width)
	out := make([]string, 0, height)
	for _, line := range lines {
		line = cut.Render(line)
		if w := lipgloss.Width(line); w < width {
			line += strings.Repeat(" ", width-w)
		}
		out = append(out, line)
	}
	for len(out) < height {
		out = append(out, strings.Repeat(" ", width))
	}
	return strings.Join(out, "\n")
}

// clip makes s one line of at most width cells, so nothing it is put beside or
// under moves. Lines are folded into spaces rather than kept: what goes through
// here is a region of a fixed size, and the daemon's errors — which the status
// bar shows — carry a second line of hint.
func clip(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = oneLine(s)
	if lipgloss.Width(s) <= width {
		return s
	}
	var b strings.Builder
	var w int
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if w+rw > width {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String()
}

// clipHead is [clip] from the other end: what does not fit is taken off the
// front and marked with an ellipsis, so the end of the line is what survives.
// The rooms pane cuts this way — rooms under one tree differ at the end of the
// path, which is exactly what a cut from the tail would take.
func clipHead(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = oneLine(s)
	if lipgloss.Width(s) <= width {
		return s
	}
	const ellipsis = "…"
	runes := []rune(s)
	kept := width - lipgloss.Width(ellipsis)
	var w int
	i := len(runes)
	for i > 0 {
		rw := lipgloss.Width(string(runes[i-1]))
		if w+rw > kept {
			break
		}
		w += rw
		i--
	}
	return ellipsis + string(runes[i:])
}

// oneLine folds every line break into a space. What goes through the two clips
// is a region of a fixed size, and a second line would move whatever is drawn
// under it.
func oneLine(s string) string {
	return strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(s)
}
