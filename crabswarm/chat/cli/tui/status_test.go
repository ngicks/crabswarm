package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"gotest.tools/v3/assert"
)

// plainFields is the bar as words, which is what the drop order is read off.
// The rendered bar cannot be: the key and its label are two styles, so an
// escape sequence sits between them and no substring of "^g editor" is in it.
func plainFields(fields []statusField) []string {
	words := make([]string, len(fields))
	for i, f := range fields {
		words[i] = f.plain()
	}
	return words
}

// The bar as written is wider than an 80-column terminal, so segments drop as
// the terminal narrows — the key hints first, least useful first, then whether
// the view is following, then the room. The connection never drops: it is the
// one carrying `log unread: …`.
func TestTheStatusBarDropsSegmentsInOrder(t *testing.T) {
	for _, tc := range []struct {
		width int
		want  []string
	}{
		{
			width: 100,
			want: []string{
				"room " + fixtureRoom, "tailing", "connected",
				"^hjkl panes", "^enter/^x sends", "^g editor", "q quits",
			},
		},
		{
			// One cell short of the whole bar: the way out of the screen is
			// the one hint an operator can be expected to guess.
			width: 91,
			want: []string{
				"room " + fixtureRoom, "tailing", "connected",
				"^hjkl panes", "^enter/^x sends", "^g editor",
			},
		},
		{
			// The conventional terminal. The editor key goes; what remains is
			// how to move and how to send.
			width: 80,
			want: []string{
				"room " + fixtureRoom, "tailing", "connected",
				"^hjkl panes", "^enter/^x sends",
			},
		},
		{
			// Which frame is lit already suggests the pane keys; nothing on
			// screen says how a message is sent.
			width: 60,
			want: []string{
				"room " + fixtureRoom, "tailing", "connected", "^enter/^x sends",
			},
		},
		{
			width: 50,
			want:  []string{"room " + fixtureRoom, "tailing", "connected"},
		},
		{
			// The conversation standing still already says the view is not
			// following.
			width: 35,
			want:  []string{"room " + fixtureRoom, "connected"},
		},
		{
			// Narrower than the room fits beside anything, and the room is the
			// last thing the panes themselves still say.
			width: 20,
			want:  []string{"connected"},
		},
	} {
		t.Run(fmt.Sprintf("%d columns", tc.width), func(t *testing.T) {
			m := fixtureModel(t, Deps{})
			assert.DeepEqual(t, plainFields(m.statusFields(tc.width)), tc.want)
			// Whatever dropped, what is drawn fits the line it is drawn on.
			assert.Assert(t, lipgloss.Width(m.statusBar(tc.width)) <= tc.width)
		})
	}
}

// A daemon that stopped answering is what the bar exists to say, so the
// connection segment survives every width the others are dropped at — and its
// second line of hint is folded, since the bar is one line and the whole
// screen would shift down otherwise.
func TestTheConnectionSegmentOutlivesTheRest(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m.tailErr = errors.New("rpc error: code = Unavailable\nis the daemon running?")

	fields := m.statusFields(defaultWidth)
	assert.Equal(t, len(fields), 1)
	assert.Assert(t, strings.HasPrefix(fields[0].plain(), "log unread: rpc error"))
	assert.Assert(t, !strings.Contains(fields[0].plain(), "\n"))
	assert.Equal(t, lipgloss.Height(m.statusBar(defaultWidth)), 1)
}

// A screen opened on a daemon that knows no rooms still says which room it is
// on, because that is the fact the operator most needs and there is no pane
// showing it.
func TestTheRoomSegmentNamesTheAbsenceOfARoom(t *testing.T) {
	m := newModel(t.Context(), Deps{}, "", nil)
	assert.Equal(t, plainFields(m.statusFields(defaultWidth))[0], "room (none)")
}

// Scrolling back is the other half of the tailing segment.
func TestTheStatusBarSaysWhetherTheViewIsFollowing(t *testing.T) {
	m := fixtureModel(t, Deps{})
	assert.Equal(t, plainFields(m.statusFields(100))[1], "tailing")
	m.following = false
	assert.Equal(t, plainFields(m.statusFields(100))[1], "scrolled back")
}

// The system line is the screen's last word to the operator, and nothing at
// all until there is one.
func TestTheSystemLineShowsTheLastNotice(t *testing.T) {
	m := fixtureModel(t, Deps{})
	assert.Equal(t, m.systemLine(80), "")

	m.notice = "editor exited: exit status 1"
	assert.Assert(t, strings.Contains(m.systemLine(80), "editor exited: exit status 1"))
	// A notice longer than the line is cut rather than wrapped: a second row
	// would push the status bar off the screen.
	assert.Equal(t, lipgloss.Height(m.systemLine(20)), 1)
	assert.Assert(t, lipgloss.Width(m.systemLine(20)) <= 20)
}
