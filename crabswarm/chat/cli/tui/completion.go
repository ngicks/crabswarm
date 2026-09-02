package tui

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// dropdownMaxRows caps the completion list: it is an overlay over the
// conversation, and one that scrolls is better than one that covers it.
const dropdownMaxRows = 6

// completionItem is one row of the `@` dropdown: an address the operator can
// send to, beside what the roster says about it.
type completionItem struct {
	address string
	state   string
}

// completionState is the dropdown. It is open only while the cursor sits at the
// end of the `@token` it was opened on, since every key that moves the cursor
// closes it — so the token it replaces is still where it was found.
type completionState struct {
	open  bool
	items []completionItem
	index int
	// token is the `@token` as it was typed, which is what accepting replaces.
	token string
}

// token is the `@token` the cursor sits at the end of: everything back to the
// last space on the cursor's line, if it starts with an `@`. The prefix may be
// empty — a bare `@` asks for the whole room.
func (m *model) token() (string, bool) {
	lines := strings.Split(m.text.Value(), "\n")
	row := m.text.Line()
	if row < 0 || row >= len(lines) {
		return "", false
	}
	runes := []rune(lines[row])
	col := min(max(m.text.Column(), 0), len(runes))
	start := col
	for start > 0 && !unicode.IsSpace(runes[start-1]) {
		start--
	}
	tok := string(runes[start:col])
	if !strings.HasPrefix(tok, "@") {
		return "", false
	}
	return tok, true
}

// openCompletion answers tab. One match is not a list — it is the answer — so
// it is applied instead of offered.
func (m *model) openCompletion() {
	tok, ok := m.token()
	if !ok {
		m.notice = "tab completes an @token"
		return
	}
	items := completions(m.roster, strings.TrimPrefix(tok, "@"))
	switch len(items) {
	case 0:
		m.notice = "nothing matches " + tok
	case 1:
		m.replaceToken(tok, items[0])
	default:
		m.completion = completionState{open: true, items: items, token: tok}
	}
}

// closeCompletion puts the dropdown away, leaving the text as it was typed.
// Every way out of the message pane goes through here: a list pointing at a
// token that is no longer under the cursor points at nothing.
func (m *model) closeCompletion() {
	m.completion = completionState{}
}

// completionKey drives the dropdown and reports whether it took the key. j and
// k move the highlight here and nowhere else in this pane: while the list is
// open they are navigation, and the moment it closes they are letters again.
//
// Tab walks the list and accepts on the last row, since a list walked to the
// end has made the choice and one more key to say so is a key too many. enter
// accepts wherever the highlight is, and esc leaves the token as typed.
func (m *model) completionKey(s string) bool {
	n := len(m.completion.items)
	if n == 0 {
		m.closeCompletion()
		return false
	}
	switch s {
	case "tab", "down", "j":
		if s == "tab" && m.completion.index == n-1 {
			m.accept()
			return true
		}
		m.completion.index = (m.completion.index + 1) % n
		return true
	case "shift+tab", "up", "k":
		m.completion.index = (m.completion.index - 1 + n) % n
		return true
	case "enter":
		m.accept()
		return true
	case "esc":
		m.closeCompletion()
		return true
	}
	m.closeCompletion()
	return false
}

// accept swaps the typed token for the address the highlight is on.
func (m *model) accept() {
	m.replaceToken(m.completion.token, m.completion.items[m.completion.index])
}

// replaceToken swaps the typed token for the unambiguous address, with the
// space that ends it, so the next word is the message.
//
// The token is deleted one backspace at a time through the textarea's own
// Update rather than by rewriting the value: a rewrite would have to put the
// cursor back, and the textarea moves it by visual row, which a wrapped message
// does not agree with. Backspace only fires while the textarea has focus, which
// is where completion happens.
func (m *model) replaceToken(tok string, item completionItem) {
	for range utf8.RuneCountInString(tok) {
		m.text, _ = m.text.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	m.text.InsertString("@" + item.address + " ")
	m.closeCompletion()
}

// completions lists what `@prefix` could mean: each team as `team/*` above its
// own members, matched on the whole address or on the bare name, since the
// operator types whichever of the two they remember.
//
// The roster arrives grouped by team, so a team is offered where its first
// member is met. The admin is never offered: they are the one at the screen.
func completions(roster []*chatv1.Member, prefix string) []completionItem {
	var items []completionItem
	team := ""
	for i, member := range roster {
		if i == 0 || member.GetTeam() != team {
			team = member.GetTeam()
			addr := cli.AdminTarget{Team: team}.String()
			if strings.HasPrefix(addr, prefix) || strings.HasPrefix(team, prefix) {
				items = append(items, completionItem{
					address: addr,
					state:   teamSizeLabel(teamSize(roster, team)),
				})
			}
		}
		addr := member.GetTeam() + "/" + member.GetName()
		if strings.HasPrefix(addr, prefix) || strings.HasPrefix(member.GetName(), prefix) {
			items = append(items, completionItem{
				address: addr,
				state:   cli.HarnessStateName(member.GetState()),
			})
		}
	}
	return items
}

// teamSize is how many of the room's members a `team/*` would reach, which is
// what its row says instead of a harness state.
func teamSize(roster []*chatv1.Member, team string) int {
	var n int
	for _, member := range roster {
		if member.GetTeam() == team {
			n++
		}
	}
	return n
}

// teamSizeLabel says how many members a `team/*` row would reach, counted so a
// team of one reads as one: a row saying "1 members" is a row the operator has
// to look past.
func teamSizeLabel(n int) string {
	if n == 1 {
		return "1 member"
	}
	return fmt.Sprintf("%d members", n)
}

// dropdownLayer draws the completion list over whatever is above the message
// pane, left-aligned to the token it is completing.
func (m *model) dropdownLayer(r paneRects, screenWidth int) *lipgloss.Layer {
	items := m.completion.items
	index := m.completion.index
	// A list longer than the cap scrolls with the highlight rather than
	// covering the conversation.
	first := 0
	if len(items) > dropdownMaxRows {
		first = min(max(index-dropdownMaxRows/2, 0), len(items)-dropdownMaxRows)
		items = items[first : first+dropdownMaxRows]
	}

	var addressWidth, stateWidth int
	for _, item := range items {
		addressWidth = max(addressWidth, lipgloss.Width("@"+item.address))
		stateWidth = max(stateWidth, lipgloss.Width(item.state))
	}
	inner := addressWidth + 2 + stateWidth
	lines := make([]string, 0, len(items))
	for i, item := range items {
		line := fmt.Sprintf("%-*s  %-*s",
			addressWidth, "@"+item.address, stateWidth, item.state)
		if first+i == index {
			line = pickedStyle.Render(line)
		}
		lines = append(lines, line)
	}

	width, height := inner+2, len(items)+2
	content := box("", fit(strings.Join(lines, "\n"), inner, len(items)), width, height, true)
	// The list points at the token: its left edge is the token's column —
	// inside the pane's border and past the prompt — and it sits on the border
	// above, which is the row the message pane can spare.
	offset := m.text.LineInfo().CharOffset + lipgloss.Width(m.text.Prompt)
	x := r.message.Min.X + 1 + max(offset-lipgloss.Width(m.completion.token), 0)
	x = min(max(x, 0), max(screenWidth-width, 0))
	y := max(r.message.Min.Y-height, 0)
	return lipgloss.NewLayer(content).X(x).Y(y).Z(1)
}
