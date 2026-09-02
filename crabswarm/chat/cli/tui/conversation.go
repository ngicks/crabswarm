package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// entryTimeFormat stamps a line with the time of day alone. The conversation
// pane is narrow and the reader is comparing messages minutes apart, so the
// date the CLI transcript carries would cost width and say nothing. UTC for the
// reason the CLI renders UTC: a room spans containers that need not agree on a
// time zone.
const entryTimeFormat = "15:04:05"

// conversationKey scrolls the log. The movement is spelled out here rather than
// handed to the viewport's own keymap: this pane scrolls vertically only — a
// message wraps instead of running off the edge — so h and l, which the
// viewport binds to horizontal scrolling, do nothing.
func (m *model) conversationKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	pending := m.pendingG
	m.pendingG = false
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "g":
		if pending {
			m.view.GotoTop()
			break
		}
		m.pendingG = true
		return m, nil
	case "G", "end":
		m.view.GotoBottom()
	case "home":
		m.view.GotoTop()
	case "j", "down":
		m.view.ScrollDown(1)
	case "k", "up":
		m.view.ScrollUp(1)
	case "ctrl+d":
		m.view.HalfPageDown()
	case "ctrl+u":
		m.view.HalfPageUp()
	case "pgdown":
		m.view.PageDown()
	case "pgup":
		m.view.PageUp()
	case "h", "l":
		// Nothing to scroll sideways to.
	}
	// Whether the view still follows the room is not a mode the operator sets
	// but where the scroll left it, so it is read back off the viewport after
	// every movement rather than tracked alongside it.
	m.following = m.view.AtBottom()
	return m, nil
}

// conversation renders the room's log as the pane's content: one entry per
// line, naming who said it and who to the way the CLI transcript names them, so
// an operator reading both reads one text. The stamp is where the two part, and
// deliberately — the pane spells the time of day where the transcript spells
// the whole date, for the reason [entryTimeFormat] gives.
//
// An entry with no recipient is addressed to [cli.BroadcastTarget]. That "*"
// spells a team-wide send as well as a room-wide one: the daemon records a team
// send with no recipient for now, so the pane cannot tell the two apart.
//
// A message that names the admin is coloured, which the transcript does not do:
// the screen is where the operator is watching for exactly that.
func (m *model) conversation() string {
	var b strings.Builder
	for _, e := range m.entries {
		at := "--:--:--"
		if ts := e.GetSentAt(); ts != nil {
			at = ts.AsTime().UTC().Format(entryTimeFormat)
		}
		to := cli.BroadcastTarget
		if e.GetTo() != nil {
			to = cli.Address(e.GetTo())
		}
		fmt.Fprintf(&b, "%s %s → %s: %s\n",
			at, cli.Address(e.GetFrom()), to, mentioned(e.GetText()))
	}
	return b.String()
}

// mentioned colours a message that names the admin: the text in the mention
// colour with the tokens that name them bold on top of it. The admin holds no
// member row — they are the one at the screen — so being named is textual,
// which is also why the colour is on the message and not on the time and the
// sender in front of it, where nothing was said.
//
// The runs are rendered one at a time rather than the bold token being drawn
// inside an already-coloured line: the reset that ends the bold would end the
// colour with it. A message that names nobody comes back untouched.
func mentioned(text string) string {
	runes := []rune(text)
	spans := adminMentions(runes)
	if len(spans) == 0 {
		return text
	}
	var b strings.Builder
	plain := func(rs []rune) {
		if len(rs) > 0 {
			b.WriteString(mentionStyle.Render(string(rs)))
		}
	}
	prev := 0
	for _, span := range spans {
		plain(runes[prev:span[0]])
		b.WriteString(mentionTokenStyle.Render(string(runes[span[0]:span[1]])))
		prev = span[1]
	}
	plain(runes[prev:])
	return b.String()
}
