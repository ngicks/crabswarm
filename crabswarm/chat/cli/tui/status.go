package tui

import (
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
)

// The two lines under the columns. The system line answers what the operator
// just did — it is empty until there is something to say and goes quiet again
// as soon as they type — and the status bar says where they are, which is
// there whether they asked or not.

// systemLine is the screen's last word to the operator that is not the
// conversation: a send result, an editor that would not run, a message
// addressed to nobody.
func (m *model) systemLine(width int) string {
	if m.notice == "" {
		return ""
	}
	return systemStyle.Render(clip(" "+m.notice, width))
}

// statusField is one field of the status bar: a word about where the operator
// is, or a key hint, which is the key and then what it does. The two are the
// same kind of field so the fitting below can drop either, and different
// enough that only the key takes the accent colour.
type statusField struct {
	key   string
	label string
}

// plain is the field with no colour on it, which is what the bar is measured
// and fitted as: a rune count over rendered text would count escape sequences
// as cells they do not occupy.
func (f statusField) plain() string {
	if f.key == "" {
		return f.label
	}
	return f.key + " " + f.label
}

func (f statusField) render() string {
	if f.key == "" {
		return statusStyle.Render(f.label)
	}
	return keyStyle.Render(f.key) + statusStyle.Render(" "+f.label)
}

// statusSep is what the fields are read apart by, and the width of it is what
// makes dropping one field worth more than the field itself.
const statusSep = " · "

// statusBar says where the operator is — which room, whether the view still
// follows it, how the daemon is answering — and names the keys the screen
// cannot show.
func (m *model) statusBar(width int) string {
	// MaxWidth below reads a zero as no limit, where a bar with no room is
	// nothing at all.
	if width <= 0 {
		return ""
	}
	fields := m.statusFields(width)
	words := make([]string, len(fields))
	for i, f := range fields {
		words[i] = f.render()
	}
	// The bar is coloured before it is cut, so the cut is the one that counts
	// escape sequences as the nothing they are wide.
	return lipgloss.NewStyle().MaxWidth(width).
		Render(" " + strings.Join(words, statusStyle.Render(statusSep)))
}

// statusFields is the bar as fields, in the order they are written and with
// the ones that did not fit already gone. Kept apart from the rendering so
// what dropped can be read without counting escape sequences.
func (m *model) statusFields(width int) []statusField {
	mode := "scrolled back"
	if m.following {
		mode = "tailing"
	}
	room := m.room
	if room == "" {
		// The screen opened on a daemon that knew no rooms at all; the rooms
		// pane fills on a later poll and the operator picks one there.
		room = "(none)"
	}
	// The daemon's errors carry a second line of hint, and the bar is one
	// line: the whole screen would shift down otherwise, so what goes on it is
	// folded before it is measured.
	roomField := statusField{label: clip("room "+room, width)}
	modeField := statusField{label: mode}
	fields := []statusField{
		roomField,
		modeField,
		{label: clip(m.connection(), width)},
		{key: "^hjkl", label: "panes"},
		{key: "^enter/^x", label: "sends"},
		{key: "^g", label: "editor"},
		{key: "q", label: "quits"},
	}
	// The bar as written is wider than an 80-column terminal, and what is cut
	// off the end is the half that cannot be guessed from the screen. So it is
	// dropped by the field rather than by the cell, in the order below.
	//
	// The key hints go first, least useful first: q is the conventional way
	// out of a screen, ctrl+g does nothing that cannot be done in the pane,
	// and pane movement is at least suggested by which frame is lit — while
	// nothing on screen says how a message is sent. Then whether the view is
	// following, which the conversation shows by standing still. Then the
	// room, which is the last thing the panes themselves still say.
	//
	// The connection field is never dropped: it is the one carrying `log
	// unread: …`, and a screen that has stopped being fed must say so even
	// when there is room for nothing else.
	for _, drop := range []statusField{
		{key: "q", label: "quits"},
		{key: "^g", label: "editor"},
		{key: "^hjkl", label: "panes"},
		{key: "^enter/^x", label: "sends"},
		modeField,
		roomField,
	} {
		if statusWidth(fields) <= width {
			break
		}
		fields = slices.DeleteFunc(fields, func(f statusField) bool { return f == drop })
	}
	return fields
}

// statusWidth is how many cells the bar takes, counting the space it is
// written in from.
func statusWidth(fields []statusField) int {
	words := make([]string, len(fields))
	for i, f := range fields {
		words[i] = f.plain()
	}
	return lipgloss.Width(" " + strings.Join(words, statusSep))
}

// connection says how the daemon is answering. The conversation is what the
// operator is here for, so a log that stopped coming is reported ahead of a
// roster that did — and a screen still being fed says so, since silence in a
// quiet room otherwise looks exactly like silence from a dead socket.
func (m *model) connection() string {
	switch {
	case m.tailErr != nil:
		return "log unread: " + m.tailErr.Error()
	case m.rosterErr != nil:
		return "roster unread: " + m.rosterErr.Error()
	default:
		return "connected"
	}
}
