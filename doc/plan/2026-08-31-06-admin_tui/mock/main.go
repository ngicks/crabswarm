//go:build tuimock

// Command mock draws the admin watch screen against fixture data, so the
// layout can be looked at before any of it is wired to a daemon.
//
// Run it with:
//
//	go run -tags tuimock ./doc/plan/2026-08-31-06-admin_tui/mock
//
// Keys: q quits, i focuses the input line, esc leaves it, enter "sends",
// up/down/pgup/pgdn scroll the conversation. Resize the terminal narrower
// than 60 columns to watch the roster collapse.
//
// # MOCK_LIMITS
//
// What this fakes, and therefore what it cannot validate:
//
//   - RPCs: nothing is dialled. The conversation, the roster and the send
//     path are in-process fixtures, so nothing here says whether the admin
//     auth plane, the room-log read or the admin send behave as assumed.
//   - States: harness states are hardcoded strings that never change. Whether
//     a member's state tracks its reports is untested.
//   - Timing: a canned message is appended on a timer instead of arriving from
//     a poll, and a "sent" line appears immediately instead of coming back
//     through the log. Poll latency, cursor paging and the no-local-echo rule
//     are all outside what this shows.
//   - Retention: the scrollback is a fixed fixture list; it says nothing about
//     how far back the daemon's retention actually reaches.
//
// The screen itself — region sizes, collapse order, where the status bar
// reads — is what this is for.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// entry is one line of the conversation, the way the room log hands it over.
type entry struct {
	at   string
	from string
	to   string
	text string
}

// member is one row of the roster sidebar.
type member struct {
	team  string
	name  string
	state string
}

// fixtureEntries is the scrollback the screen opens on.
var fixtureEntries = []entry{
	{"09:12:04", "backend/alice", "*", "picking up the schema migration"},
	{"09:12:31", "backend/bob", "backend/alice", "the column is still nullable on main"},
	{"09:13:02", "backend/alice", "backend/bob", "rebasing, give me a minute"},
	{"09:14:20", "frontend/cid", "*", "blocked on the new field until that lands"},
	{"09:15:44", "admin/admin", "*", "standup in five"},
	{"09:16:10", "backend/bob", "*", "rebase is green"},
	{"09:17:55", "frontend/cid", "backend/alice", "which field name won?"},
	{"09:18:12", "backend/alice", "frontend/cid", "author_id, unchanged"},
}

// fixtureRoster is the attendance the sidebar lists.
var fixtureRoster = []member{
	{"backend", "alice", "working"},
	{"backend", "bob", "waiting"},
	{"frontend", "cid", "done"},
	{"frontend", "dee", "unknown"},
}

// liveInterval is how often the fake tail appends a message.
const liveInterval = 3 * time.Second

// liveMsg is the timer standing in for a message arriving from the room.
type liveMsg time.Time

func live() tea.Cmd {
	return tea.Tick(liveInterval, func(t time.Time) tea.Msg { return liveMsg(t) })
}

// Layout constants: the roster is a fixed column that disappears entirely
// rather than shrinking, and the conversation takes whatever is left.
const (
	rosterWidth    = 22
	collapseBelow  = 60
	defaultWidth   = 80
	defaultHeight  = 24
	chromeHeight   = 2 // input line + status bar
	minViewHeight  = 1
	rosterGapWidth = 1
)

type model struct {
	width, height int
	entries       []entry
	roster        []member
	view          viewport.Model
	input         textinput.Model
	following     bool
	status        string
	liveIndex     int
}

func newModel() *model {
	in := textinput.New()
	in.Prompt = "> "
	in.Placeholder = "team/name: text   (or *: text)"

	view := viewport.New()
	// A conversation is read, not scrolled sideways: a long message wraps
	// rather than running off the edge of the pane.
	view.SoftWrap = true

	m := &model{
		width:     defaultWidth,
		height:    defaultHeight,
		entries:   fixtureEntries,
		roster:    fixtureRoster,
		view:      view,
		input:     in,
		following: true,
	}
	m.layout()
	return m
}

func (m *model) Init() tea.Cmd {
	return live()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil
	case liveMsg:
		m.appendLive()
		return m, live()
	case tea.KeyPressMsg:
		return m.key(msg)
	}
	return m, nil
}

// key routes a keypress by mode: an unfocused screen is being watched, a
// focused one is being typed into.
func (m *model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if !m.input.Focused() {
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit
		case "i", "enter":
			return m, m.input.Focus()
		}
		before := m.view.AtBottom()
		var cmd tea.Cmd
		m.view, cmd = m.view.Update(msg)
		if before != m.view.AtBottom() {
			m.following = m.view.AtBottom()
		}
		return m, cmd
	}
	switch msg.String() {
	case "esc":
		m.input.Blur()
		return m, nil
	case "enter":
		m.send()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// send fakes the admin send: the line is split at the first colon and the
// result is appended straight to the conversation.
func (m *model) send() {
	line := strings.TrimSpace(m.input.Value())
	target, text, ok := strings.Cut(line, ":")
	target, text = strings.TrimSpace(target), strings.TrimSpace(text)
	if !ok || target == "" || text == "" {
		m.status = `write it as "team/name: text"`
		return
	}
	m.entries = append(m.entries, entry{
		at: time.Now().Format("15:04:05"), from: "admin/admin", to: target, text: text,
	})
	m.input.Reset()
	m.status = "sent to " + target
	m.layout()
}

// appendLive adds the next canned message, standing in for the tail poll.
func (m *model) appendLive() {
	canned := []entry{
		{from: "backend/bob", to: "*", text: "tests are green on the branch"},
		{from: "frontend/cid", to: "*", text: "pulling the new field in now"},
		{from: "backend/alice", to: "frontend/cid", text: "it is deployed to staging"},
	}
	next := canned[m.liveIndex%len(canned)]
	m.liveIndex++
	next.at = time.Now().Format("15:04:05")
	m.entries = append(m.entries, next)
	m.layout()
}

// layout re-sizes the regions to the current terminal and refills the
// viewport, keeping it pinned to the tail while the view is following.
func (m *model) layout() {
	width, height := m.width, m.height
	if width <= 0 {
		width = defaultWidth
	}
	if height <= 0 {
		height = defaultHeight
	}
	conversation := width
	if m.rosterVisible() {
		conversation = width - rosterWidth - rosterGapWidth
	}
	body := max(height-chromeHeight, minViewHeight)
	m.view.SetWidth(conversation)
	m.view.SetHeight(body)
	m.input.SetWidth(width)
	m.view.SetContent(m.conversation())
	if m.following {
		m.view.GotoBottom()
	}
}

// rosterVisible reports whether the sidebar still fits. The conversation is
// the part that has to survive a narrow terminal, so the roster is what goes.
func (m *model) rosterVisible() bool {
	return m.width >= collapseBelow
}

func (m *model) conversation() string {
	var b strings.Builder
	for _, e := range m.entries {
		fmt.Fprintf(&b, "%s %s → %s: %s\n", e.at, e.from, e.to, e.text)
	}
	return b.String()
}

// rosterPane renders the sidebar: members grouped under their team, each with
// the harness state that says whether it can be interrupted.
func (m *model) rosterPane(height int) string {
	var b strings.Builder
	b.WriteString("roster\n")
	team := ""
	for _, mem := range m.roster {
		if mem.team != team {
			team = mem.team
			b.WriteString(team + "\n")
		}
		fmt.Fprintf(&b, " %-10s %s\n", mem.name, mem.state)
	}
	return lipgloss.NewStyle().Width(rosterWidth).Height(height).Render(b.String())
}

func (m *model) statusBar() string {
	mode := "scrolled back"
	if m.following {
		mode = "tailing"
	}
	line := fmt.Sprintf("room /work/proj · %s · connected (mock)", mode)
	if m.status != "" {
		line += " · " + m.status
	}
	return lipgloss.NewStyle().Width(m.width).Render(line)
}

func (m *model) View() tea.View {
	body := m.view.View()
	if m.rosterVisible() {
		body = lipgloss.JoinHorizontal(
			lipgloss.Top, body, strings.Repeat(" ", rosterGapWidth),
			m.rosterPane(m.view.Height()))
	}
	v := tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left, body, m.input.View(), m.statusBar()))
	v.AltScreen = true
	return v
}

func main() {
	if _, err := tea.NewProgram(newModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "mock:", err)
		os.Exit(1)
	}
}
