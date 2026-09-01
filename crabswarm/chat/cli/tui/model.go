package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// The screen is three regions: the conversation, the roster beside it, and two
// lines of chrome under both — the line the operator types into and the status
// bar. Sizes below are in terminal cells.
const (
	// rosterWidth is the sidebar's column count. Fixed rather than
	// proportional: it holds a name and a state word, and neither gets more
	// readable for being given half the screen.
	rosterWidth = 22
	// rosterGap separates the two panes, since neither is boxed.
	rosterGap = 1
	// rosterMinWidth is the terminal width below which the sidebar is dropped
	// entirely. Watching the conversation is what the screen is for, so the
	// roster is what a narrow terminal loses.
	rosterMinWidth = 60
	// chromeHeight is the input line plus the status bar.
	chromeHeight = 2
	// nameColumn is how much of a member's name the sidebar spells before the
	// state word.
	nameColumn = 12
)

// A terminal that never reported its size — a program driven through a pipe,
// which is how the tests drive it — is drawn at the size a terminal is
// conventionally assumed to have rather than at nothing.
const (
	defaultWidth  = 80
	defaultHeight = 24
)

// broadcastTarget spells the addressee of an entry that went to the whole room.
// It is the "*" the admin send verb takes, so the transcript names a target the
// operator can type back into the input line.
const broadcastTarget = "*"

// entryTimeFormat stamps a line with the time of day alone. The conversation
// pane is narrow and the reader is comparing messages minutes apart, so the
// date the CLI transcript carries would cost width and say nothing. UTC for the
// reason the CLI renders UTC: a room spans containers that need not agree on a
// time zone.
const entryTimeFormat = "15:04:05"

// model is the whole screen. It owns no connection: everything it shows was
// handed to it, and everything it does to the room goes out through [Deps].
type model struct {
	// ctx is the one [Run] was given, held here rather than passed because a
	// bubbletea command is a func() Msg with nowhere to take one: the poll
	// loops are built in Update and have no other way to reach it.
	ctx  context.Context
	deps Deps
	room string

	width  int
	height int

	view  viewport.Model
	input textinput.Model

	// entries is the conversation, oldest first, exactly as the log handed it
	// over — the log is the only source of what the room said.
	entries []*chatv1.AdminHistoryEntry
	roster  []*chatv1.Member

	// cursor is the id of the newest entry the screen holds, which is what the
	// next read of the log asks to be told about.
	cursor int64

	// tailErr and rosterErr are how the last read of each kind went, so the
	// status bar can say the daemon stopped answering instead of the screen
	// simply going quiet.
	tailErr   error
	rosterErr error

	// following says the view is pinned to the newest entry. Scrolling away
	// clears it and scrolling back to the bottom sets it again, which is what
	// makes a scrolled-back reader stay where it is while the room talks on.
	following bool

	// notice is the last thing the screen has to say to the operator that is
	// not the conversation — a rejected input line, mostly.
	notice string
}

func newModel(ctx context.Context, deps Deps, roster []*chatv1.Member) *model {
	input := textinput.New()
	input.Prompt = "> "
	input.Placeholder = `team/name: text`

	view := viewport.New()
	// A conversation is read, not scrolled sideways: a long message wraps
	// rather than running off the edge of the pane.
	view.SoftWrap = true

	m := &model{
		ctx:       ctx,
		deps:      deps,
		room:      deps.Room,
		width:     defaultWidth,
		height:    defaultHeight,
		view:      view,
		input:     input,
		roster:    roster,
		following: true,
	}
	m.layout()
	return m
}

// Init opens both loops at once: the conversation the operator arrived to read
// is fetched immediately rather than a tick later, and so is the roster, which
// the room lookup already read but may since have moved.
func (m *model) Init() tea.Cmd {
	return tea.Batch(m.tail(), m.pollRoster())
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil
	case tailTickMsg:
		return m, m.tail()
	case rosterTickMsg:
		return m, m.pollRoster()
	case tailMsg:
		m.applyTail(msg)
		return m, tickTail()
	case rosterMsg:
		m.applyRoster(msg)
		return m, tickRoster()
	case sentMsg:
		m.applySent(msg)
		return m, nil
	case tea.KeyPressMsg:
		return m.key(msg)
	}
	return m, nil
}

// key routes a keypress by mode. An unfocused screen is being watched, which is
// what the operator is here for, so the letters navigate; a focused one is
// being typed into, where the same letters are text.
func (m *model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.input.Focused() {
		return m.composeKey(msg)
	}
	return m.watchKey(msg)
}

// watchKey navigates the conversation.
func (m *model) watchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		return m, tea.Quit
	case "i", "enter":
		return m, m.input.Focus()
	case "end", "G":
		m.view.GotoBottom()
	case "home", "g":
		m.view.GotoTop()
	default:
		var cmd tea.Cmd
		m.view, cmd = m.view.Update(msg)
		m.following = m.view.AtBottom()
		return m, cmd
	}
	// Whether the view still follows the room is not a mode the operator sets
	// but where the scroll left it, so it is read back off the viewport after
	// every movement rather than tracked alongside it.
	m.following = m.view.AtBottom()
	return m, nil
}

// composeKey edits the line being written, and sends it.
func (m *model) composeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.input.Blur()
		return m, nil
	case "enter":
		return m, m.submit()
	}
	// Typing answers whatever the bar last said, so the report goes with it.
	m.notice = ""
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// layout re-sizes the regions to the terminal and refills the conversation,
// keeping the view on the newest entry while it is following the room.
func (m *model) layout() {
	width, height := m.size()
	conversation := width
	if m.rosterShown() {
		conversation = width - rosterWidth - rosterGap
	}
	m.view.SetWidth(conversation)
	m.view.SetHeight(max(height-chromeHeight, 1))
	m.input.SetWidth(max(width-lipgloss.Width(m.input.Prompt), 1))
	m.view.SetContent(m.conversation())
	if m.following {
		m.view.GotoBottom()
	}
}

// size is the terminal's, or the assumed one until it says.
func (m *model) size() (width, height int) {
	width, height = m.width, m.height
	if width <= 0 {
		width = defaultWidth
	}
	if height <= 0 {
		height = defaultHeight
	}
	return width, height
}

// rosterShown reports whether the sidebar fits beside the conversation.
func (m *model) rosterShown() bool {
	width, _ := m.size()
	return width >= rosterMinWidth
}

// conversation renders the room's log as the pane's content: one entry per
// line, spelled the way the CLI transcript spells one so an operator reading
// both reads one text.
func (m *model) conversation() string {
	var b strings.Builder
	for _, e := range m.entries {
		at := "--:--:--"
		if ts := e.GetSentAt(); ts != nil {
			at = ts.AsTime().UTC().Format(entryTimeFormat)
		}
		to := broadcastTarget
		if e.GetTo() != nil {
			to = cli.Address(e.GetTo())
		}
		fmt.Fprintf(&b, "%s %s → %s: %s\n", at, cli.Address(e.GetFrom()), to, e.GetText())
	}
	return b.String()
}

// rosterPane renders the sidebar: the room's attendance grouped by team, each
// member with the harness state that says whether it can be interrupted.
func (m *model) rosterPane(height int) string {
	lines := []string{fmt.Sprintf("roster (%d)", len(m.roster))}
	team := ""
	for _, member := range m.roster {
		if member.GetTeam() != team {
			team = member.GetTeam()
			lines = append(lines, clip(team, rosterWidth))
		}
		lines = append(lines, clip(fmt.Sprintf(" %-*s %s",
			nameColumn, member.GetName(), cli.HarnessStateName(member.GetState())),
			rosterWidth))
	}
	return lipgloss.NewStyle().
		Width(rosterWidth).
		Height(height).
		Render(strings.Join(lines, "\n"))
}

// statusBar says where the operator is: which room, whether the view still
// follows it, and whatever the screen last had to report.
func (m *model) statusBar() string {
	width, _ := m.size()
	parts := []string{"room " + m.room}
	if m.following {
		parts = append(parts, "tailing")
	} else {
		parts = append(parts, "scrolled back")
	}
	parts = append(parts, m.connection())
	if m.notice != "" {
		parts = append(parts, m.notice)
	}
	parts = append(parts, m.keyHint())
	return clip(strings.Join(parts, " · "), width)
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

// keyHint names the two keys that are not guessable from the screen: how to
// leave, and how to reach the input line.
func (m *model) keyHint() string {
	if m.input.Focused() {
		return "enter sends · esc leaves the line"
	}
	return "q quits · i writes"
}

func (m *model) View() tea.View {
	body := m.view.View()
	if m.rosterShown() {
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			body, strings.Repeat(" ", rosterGap), m.rosterPane(m.view.Height()))
	}
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
		body, m.input.View(), m.statusBar()))
	// The screen is a place the operator stays, not output scrolling past, so
	// it takes the alternate buffer and gives the shell back untouched on exit.
	v.AltScreen = true
	return v
}

// clip shortens s to width cells, so an over-long line cannot push the region
// beside it off the screen.
func clip(s string, width int) string {
	if width <= 0 {
		return ""
	}
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
