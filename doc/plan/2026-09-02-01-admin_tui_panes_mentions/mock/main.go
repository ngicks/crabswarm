//go:build tuimock

// Command mock draws the pane'd admin chat screen against fixture data, so the
// layout, the focus scheme and the `@` addressing can be looked at before any
// of it is wired to a daemon.
//
// Run it with:
//
//	go run -tags tuimock ./doc/plan/2026-09-02-01-admin_tui_panes_mentions/mock
//
// The screen is a left column — rooms above members — beside a right column of
// the conversation above the message, with the system line and the status bar
// across the bottom.
//
// Keys:
//
//	ctrl+c            quit, from anywhere
//	q                 quit, unless the message pane holds focus
//	ctrl+h/j/k/l      move focus one pane left / down / up / right. The target
//	                  is read off the solved layout — of the panes wholly on
//	                  that side, the one sharing the most edge with the focused
//	                  pane — so a changed split moves the keys with it, and a
//	                  direction with nothing on it stays put. At the default
//	                  split that comes out as:
//	                  rooms        ctrl+j members      · ctrl+l conversation
//	                  members      ctrl+k rooms        · ctrl+l conversation
//	                  conversation ctrl+h members      · ctrl+j message
//	                  message      ctrl+k conversation · ctrl+h members
//
//	rooms pane        j/k (↓/↑) move the cursor · gg top · G bottom ·
//	                  ctrl+d/ctrl+u half a page
//	                  enter selects the room, which swaps the conversation and
//	                  the members for that room's fixture and the message for
//	                  the draft left in that room
//	members pane      j/k/↓/↑ move the cursor over teams and members
//	                  enter puts @team/name (or @team/* on a heading) at the
//	                  start of the message and moves focus there
//	conversation pane j/k (↓/↑) scroll a line · ctrl+d/ctrl+u half a page ·
//	                  pgdn/pgup a page · gg top · G bottom · h/l do nothing
//	message pane      typing goes to the textarea · enter is a newline
//	                  ctrl+enter (or ctrl+x) sends · ctrl+g opens $EDITOR
//	                  tab after an @token opens the completion dropdown
//	                  (tab/↓/j next · shift+tab/↑/k previous · enter accepts
//	                  · esc closes)
//
// Resize the terminal narrower than 60 columns to watch the left column drop.
//
// # MOCK_LIMITS
//
// What this fakes, and therefore what it cannot validate:
//
//   - RPCs: nothing is dialled. The rooms, the conversation, the roster and the
//     send path are in-process fixtures, so nothing here says whether the admin
//     auth plane, the room-log read or the admin send behave as assumed.
//   - Rooms: the list is a fixture of four paths, not a discovery. Selecting one
//     swaps the conversation and the roster for that room's fixture instead of
//     reading another room's log, and the 8s tail appends to whichever room is
//     on screen, so nothing here says how many rooms there are, how the list is
//     kept current, or what happens to a room while it is not being looked at.
//   - Drafts: the message left in a room comes back on returning to it, but only
//     for the life of the process. Whether a draft should outlive the screen,
//     and where it would be kept if it did, is not answered here.
//   - Target resolution: the mock resolves `@token` against its own fixture
//     roster — exact `team/name`, a bare `name` (first match wins, so the
//     name collision the real daemon has to reject never happens here), or
//     `team/*`. The daemon's rules for ambiguity, for members that joined
//     since the roster was read, and for an empty team are untested.
//   - Delivered counts: N is counted off the fixture roster (1 for a member,
//     the team size for `team/*`, everyone for `*`), not reported by a
//     delivery. Whether a send that reaches nobody still counts is unanswered.
//   - Local echo: a sent message is appended to the conversation here after a
//     fake 300ms delay. The real screen adds nothing and waits for the log to
//     bring it back, so this shows the timing of neither.
//   - Timing: a canned message is appended on an 8s timer instead of arriving
//     from a poll. Poll latency, cursor paging and retention are outside it.
//   - States: harness states are hardcoded strings that never change.
//   - Retention: the scrollback is a fixed fixture list.
//   - Keyboard enhancements: ctrl+enter is only distinguishable from enter in
//     terminals that answer the kitty keyboard protocol. A terminal that
//     ignores the query answers nothing at all, so [tea.KeyboardEnhancementsMsg]
//     never arrives and the fallback hint on the status bar never appears even
//     though ctrl+enter will not work — which is why ctrl+x always sends.
//     Whether the operator's terminal is one of those is exactly what this
//     cannot tell you.
//   - Editor: $EDITOR/$VISUAL is run for real, so that part is not faked. What
//     is faked is everything it hands back to.
//   - Palette: bubbletea's own, hardcoded ANSI-256 — purple "62" on the focused
//     frame, its title and the keys the status bar names; pink "205" on a
//     cursor, the selected room's mark and a mention; violet "99" on a team
//     heading; greys "240"/"245" on a blurred frame and its title, "245" on the
//     system line and "241" on the status bar. What those numbers actually
//     look like is the terminal's palette, not this, so nothing here says
//     whether the operator's theme keeps them apart or readable.
//   - Mentions: the admin is named textually. They speak as `admin/admin` and
//     have no member row, so a line is a mention when its text holds a bare
//     `@admin` or `@admin/admin` — bare by [parseAddress]'s rules, so backticks
//     and `\@` exclude it — and is then drawn in the mention colour with the
//     token bold. Whether the daemon would instead say who a message reached,
//     and what a mention should do beyond colour it, is not answered here.
//
// Two things the screen itself is worth looking at for, since the mock is
// honest about them:
//
//   - Unbinding ctrl+h from the textarea's DeleteCharacterBackward (so it can
//     move focus) costs backspace in every terminal that still sends ^H for
//     it. bubbles binds backspace and ctrl+h together for exactly that reason.
//   - A multi-line message is folded into one conversation line with ` ⏎ `
//     marking the newlines, rather than drawn as continuation lines.
package main

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/layout"
	shellwords "github.com/mattn/go-shellwords"
)

// entry is one line of the conversation, the way the room log hands it over.
type entry struct {
	at   string
	from string
	to   string
	text string
}

// member is one member of the room, the way the roster lists it.
type member struct {
	team  string
	name  string
	state string
}

// rosterRow is one line of the roster pane. The cursor moves over team
// headings and members alike, so both are the same kind of row; a heading is
// the one with no name.
type rosterRow struct {
	team  string
	name  string
	state string
}

func (r rosterRow) heading() bool { return r.name == "" }

// address is what enter on this row puts in front of the message.
func (r rosterRow) address() string {
	if r.heading() {
		return r.team + "/*"
	}
	return r.team + "/" + r.name
}

// room is one of the rooms the left column lists, together with the fixture the
// rest of the screen shows while it is the selected one and whatever was left
// half-written in it.
type room struct {
	path    string
	entries []entry
	roster  []member
	// draft is the message the operator had typed when they left the room. It
	// is not part of the fixture and lives no longer than the process.
	draft string
}

const (
	admin = "admin/admin"
	// adminName is the admin's name on its own. The admin has no member row —
	// they are the one holding the screen — so `@admin` and `@admin/admin` both
	// name them and neither is ever offered by the completion.
	adminName = "admin"
	// stamp spells the time of day alone, as the real pane does.
	stamp = "15:04:05"
	// broadcast is the target of a message with no bare @ in it.
	broadcast = "*"
)

// fixtureRooms is the room list, each with a room of its own so selecting one
// visibly changes the conversation and the members.
var fixtureRooms = []room{
	{
		path: "/work",
		entries: []entry{
			{"11:58:20", "admin/admin", "*", "morning — alpha takes the schema, beta takes the importer"},
			{"11:58:44", "alpha/ana", "*", "picking up the migration now"},
			{"11:59:02", "beta/cy", "*", "importer branch is cut"},
			{"12:00:01", "alpha/ana", "*", "rebasing onto main"},
			{"12:00:07", "alpha/bob", "*", "the branch is green"},
			{"12:00:09", "admin/admin", "alpha/ana", "hold the deploy until the schema lands"},
			{"12:01:12", "alpha/ana", "admin/admin", "holding"},
			{"12:02:30", "beta/dee", "beta/*", "who owns the retry loop?"},
			{"12:02:51", "beta/cy", "beta/dee", "I do — leave it to me"},
			{"12:03:26", "beta/dee", "admin/admin", "@admin staging is out of disk, can you clear the old builds?"},
			{"12:03:58", "alpha/bob", "admin/admin", "@admin/admin the schema freeze — does it cover the importer too?"},
			{"12:04:18", "alpha/bob", "beta/cy", "the importer reads author_id, not author"},
			{"12:05:02", "beta/cy", "alpha/bob", "fixed on my branch"},
			{"12:05:33", "beta/dee", "*", "the docs still write `@admin` in backticks, which addresses nobody"},
			{"12:06:40", "alpha/ana", "*", "migration is on staging"},
		},
		roster: []member{
			{"alpha", "ana", "working"},
			{"alpha", "bob", "waiting"},
			{"beta", "cy", "done"},
			{"beta", "dee", "working"},
		},
	},
	{
		path: "/home/op/.dotfiles",
		entries: []entry{
			{"09:12:03", "admin/admin", "*", "split the zsh rc before touching the prompt"},
			{"09:13:40", "shell/eve", "*", "rc is split, prompt is next"},
		},
		roster: []member{
			{"shell", "eve", "working"},
		},
	},
	{
		path: "/src/cmdman",
		entries: []entry{
			{"14:41:11", "admin/admin", "docs/fay", "the flag table is out of date"},
			{"14:41:58", "docs/fay", "admin/admin", "regenerating it now"},
			{"14:43:02", "docs/gus", "*", "examples build clean"},
		},
		roster: []member{
			{"docs", "fay", "working"},
			{"docs", "gus", "done"},
		},
	},
	{
		path: "/src/crabswarm",
		entries: []entry{
			{"16:02:19", "core/hal", "*", "admin plane is stubbed"},
			{"16:04:30", "core/ivy", "core/hal", "I'll take the room log read"},
		},
		roster: []member{
			{"core", "hal", "waiting"},
			{"core", "ivy", "working"},
		},
	},
}

// Layout: the left column is a fixed width that disappears rather than
// shrinking, the message pane is as tall as what is written in it within
// bounds, and the conversation takes whatever the rest leave.
const (
	leftWidth     = 22
	leftMinWidth  = 60
	textMinHeight = 3
	textMaxHeight = 6
	defaultWidth  = 80
	defaultHeight = 24
	// minWidth and minHeight are what the constraint solver is handed when the
	// terminal is smaller than the chrome. The canvas is still the terminal's
	// size, so a screen this small is cut off rather than wrongly solved.
	minWidth  = 24
	minHeight = 8
	// dropdownMaxRows caps the completion list: it is an overlay, and one that
	// covers the conversation is worse than one that scrolls.
	dropdownMaxRows = 6
	// nameColumn is how much of a member's name the roster spells before the
	// state word.
	nameColumn = 10
	// roomsMinHeight is the floor the rooms pane keeps when a third of the left
	// column is less than a border and one row.
	roomsMinHeight = 3
	// selectedMark and unselectedMark head a room row, so the selected room is
	// still named while the cursor is somewhere else.
	selectedMark   = "▸ "
	unselectedMark = "  "
)

const (
	// sendDelay stands in for the round trip to the daemon.
	sendDelay = 300 * time.Millisecond
	// liveInterval is how often the fake tail appends a message.
	liveInterval = 8 * time.Second
)

// paneFocus names the pane that keys reach. There is no watch mode and no
// insert mode: there is only which pane is focused.
type paneFocus int

const (
	focusRooms paneFocus = iota
	focusMembers
	focusConversation
	focusMessage
)

// The screen wears bubbletea's own colours, ANSI-256 throughout so a terminal
// with a themed palette recolours it rather than fighting it: purple on the
// pane the keys reach, pink on whatever a cursor or a mention sits on, grey on
// everything the operator is not being asked to look at.
var (
	// purple marks what has focus — the frame, its title, the keys the status
	// bar names.
	purple = lipgloss.Color("62")
	// violet heads a team in the members pane.
	violet = lipgloss.Color("99")
	// pink is a selection: the cursor's row, the marked room, a mention.
	pink = lipgloss.Color("205")
	// dim, muted and faint are the three greys: a blurred frame, a blurred
	// title beside the system line, and the status bar.
	dim   = lipgloss.Color("240")
	muted = lipgloss.Color("245")
	faint = lipgloss.Color("241")
)

var (
	focusedEdge       = purple
	blurredEdge       = dim
	focusedTitleStyle = lipgloss.NewStyle().Foreground(purple).Bold(true)
	blurredTitleStyle = lipgloss.NewStyle().Foreground(muted)
	systemStyle       = lipgloss.NewStyle().Foreground(muted)
	statusStyle       = lipgloss.NewStyle().Foreground(faint)
	// keyStyle lifts the key out of the hint it heads, so the bar reads as
	// keys with words after them rather than one grey sentence.
	keyStyle = lipgloss.NewStyle().Foreground(purple)
	// pickedStyle is the row a cursor is on, in the rooms and members panes
	// and in the completion dropdown.
	pickedStyle = lipgloss.NewStyle().Foreground(pink).Bold(true)
	// markStyle colours the ▸ of the selected room while the cursor is
	// somewhere else, so the mark alone carries the colour and the path stays
	// the room's name.
	markStyle = lipgloss.NewStyle().Foreground(pink)
	teamStyle = lipgloss.NewStyle().Foreground(violet)
	// mentionStyle draws a line that names the admin and mentionTokenStyle the
	// `@admin` in it, so the mention is found at a glance and then read.
	mentionStyle      = lipgloss.NewStyle().Foreground(pink)
	mentionTokenStyle = lipgloss.NewStyle().Foreground(pink).Bold(true)
)

// liveMsg is the timer standing in for a message arriving from the room.
type liveMsg time.Time

func live() tea.Cmd {
	return tea.Tick(liveInterval, func(t time.Time) tea.Msg { return liveMsg(t) })
}

// sentMsg is how the fake delivery came back. The text is carried along so a
// send that failed can be handed back instead of vanishing with the error, and
// the room so a send that lands after the operator moved on goes to the room it
// was written in.
type sentMsg struct {
	room      int
	target    string
	text      string
	delivered int
	err       error
}

// editedMsg is the editor having exited, with whatever it left in the file.
type editedMsg struct {
	text string
	ok   bool
	err  error
}

// completionItem is one row of the `@` dropdown.
type completionItem struct {
	address string
	state   string
}

// completion is the dropdown's state. It is open only while the cursor sits at
// the end of an `@token`, and no key that moves the cursor reaches the
// textarea while it is, so the token it was opened on stays where it was.
type completion struct {
	open  bool
	items []completionItem
	index int
	token string
}

type model struct {
	width, height int

	rooms []room
	// roomIndex is the selected room — the one the rest of the screen shows —
	// and roomCursor is where the rooms pane's highlight sits, which is only the
	// same until the cursor moves off it.
	roomIndex  int
	roomCursor int
	rows       []rosterRow

	view viewport.Model
	text textarea.Model

	focus  paneFocus
	cursor int
	// pendingG is a `g` waiting for the second one that means "top". Any other
	// key abandons it, so a mistyped g is a keypress and not a mode.
	pendingG bool

	following bool
	notice    string

	comp completion

	// sendKey is ctrl+enter where the terminal can report it and ctrl+x
	// where it said it cannot.
	sendKey   string
	liveIndex int
}

func newModel() *model {
	text := textarea.New()
	// The pane draws the frame and the title, so the textarea draws neither a
	// prompt nor line numbers — and the dropdown's column arithmetic is the
	// cursor's offset plus the border, with nothing in between.
	text.Prompt = ""
	text.ShowLineNumbers = false
	text.Placeholder = "message — @team/name addresses one, no @ broadcasts"
	text.DynamicHeight = true
	text.MinHeight = textMinHeight
	text.MaxHeight = textMaxHeight
	// MaxHeight alone doubles as the content guard, which would stop the
	// operator typing at six lines. MaxContentHeight moves the guard onto
	// visual lines so MaxHeight only caps how much of the message is on screen.
	text.MaxContentHeight = 500
	text.SetHeight(textMinHeight)
	text.SetWidth(defaultWidth - 2)
	// ctrl+h and ctrl+k move focus and ctrl+g opens the editor, so the
	// textarea's defaults for them are unbound rather than shadowed: this
	// screen intercepts them first, and a binding still holding the key would
	// only fire when that interception is wrong.
	text.KeyMap.DeleteAfterCursor = key.NewBinding()
	text.KeyMap.SelectAll = key.NewBinding()
	text.KeyMap.DeleteCharacterBackward = key.NewBinding(key.WithKeys("backspace"))
	// The message pane takes the accent colour on the line being written. The
	// styles are built fresh rather than tinted, since the default focused
	// cursor line carries a background that would stay behind the colour.
	styles := text.Styles()
	styles.Focused.Prompt = lipgloss.NewStyle().Foreground(purple)
	styles.Focused.CursorLine = lipgloss.NewStyle().Foreground(purple)
	text.SetStyles(styles)

	view := viewport.New()
	// A conversation is read, not scrolled sideways: a long message wraps
	// rather than running off the edge of the pane.
	view.SoftWrap = true

	m := &model{
		width:     defaultWidth,
		height:    defaultHeight,
		rooms:     fixtureRooms,
		rows:      rosterRows(fixtureRooms[0].roster),
		view:      view,
		text:      text,
		focus:     focusConversation,
		following: true,
		sendKey:   "ctrl+enter",
	}
	m.layout()
	return m
}

// current is the selected room, by pointer: the fake tail and the fake delivery
// append to the room they were meant for.
func (m *model) current() *room { return &m.rooms[m.roomIndex] }

// rosterRows flattens the roster into the lines the pane draws, a heading
// before each team's members.
func rosterRows(roster []member) []rosterRow {
	var rows []rosterRow
	team := ""
	for _, mem := range roster {
		if mem.team != team {
			team = mem.team
			rows = append(rows, rosterRow{team: team})
		}
		rows = append(rows, rosterRow{team: mem.team, name: mem.name, state: mem.state})
	}
	return rows
}

func (m *model) Init() tea.Cmd {
	return live()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if (m.focus == focusRooms || m.focus == focusMembers) && !m.leftShown() {
			m.focus = focusConversation
		}
		m.layout()
		return m, nil
	case tea.KeyboardEnhancementsMsg:
		// Without disambiguation the terminal reports ctrl+enter as enter,
		// which is the newline key here, so the send moves to a chord the
		// terminal can still tell apart.
		if !msg.SupportsKeyDisambiguation() {
			m.sendKey = "ctrl+x"
		}
		return m, nil
	case liveMsg:
		m.appendLive()
		m.layout()
		return m, live()
	case sentMsg:
		m.applySent(msg)
		m.layout()
		return m, nil
	case editedMsg:
		m.applyEdited(msg)
		m.layout()
		return m, nil
	case tea.KeyPressMsg:
		return m.key(msg)
	}
	// Everything else — cursor blink, mostly — belongs to the textarea while
	// it has focus.
	if m.focus == focusMessage {
		var cmd tea.Cmd
		m.text, cmd = m.text.Update(msg)
		return m, cmd
	}
	return m, nil
}

// key routes a keypress: the dropdown first if it is open, then pane movement,
// then whichever pane holds focus. Pane movement is read before the textarea
// ever sees the key, which is what "no pane swallows those keys" means.
func (m *model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	if s == "ctrl+c" {
		return m, tea.Quit
	}
	if m.comp.open {
		if handled, cmd := m.completionKey(s); handled {
			return m, cmd
		}
		// Any other key closes the list and is handled normally.
	}
	switch s {
	case "ctrl+h", "ctrl+j", "ctrl+k", "ctrl+l":
		return m, m.moveFocus(s)
	}
	switch m.focus {
	case focusRooms:
		return m.roomsKey(msg)
	case focusMembers:
		return m.membersKey(msg)
	case focusConversation:
		return m.conversationKey(msg)
	default:
		return m.messageKey(msg)
	}
}

// pane is one focusable region together with the rectangle this frame solved
// it to.
type pane struct {
	focus paneFocus
	rect  uv.Rectangle
}

// panes lists what focus can reach. A hidden left column contributes nothing,
// so no direction leads into it.
func (m *model) panes() []pane {
	r := m.rects()
	var panes []pane
	if r.leftShown {
		panes = append(panes, pane{focusRooms, r.rooms}, pane{focusMembers, r.members})
	}
	return append(panes,
		pane{focusConversation, r.conversation},
		pane{focusMessage, r.message},
	)
}

// moveFocus is a plain directional move. The target is read off the layout so a
// changed split moves the keys with it: of the panes lying wholly on that side,
// the one taken is whichever shares the most edge with the focused pane, ties
// going to the upper — then the leftmost — of them. Nothing on that side leaves
// focus where it was rather than wrapping.
func (m *model) moveFocus(k string) tea.Cmd {
	panes := m.panes()
	i := slices.IndexFunc(panes, func(p pane) bool { return p.focus == m.focus })
	if i < 0 {
		return nil
	}
	from := panes[i].rect

	candidates := slices.DeleteFunc(panes, func(p pane) bool {
		return p.focus == m.focus || !beyond(from, p.rect, k) || shares(from, p.rect, k) <= 0
	})
	if len(candidates) == 0 {
		return nil
	}
	best := slices.MaxFunc(candidates, func(a, b pane) int {
		return cmp.Or(
			cmp.Compare(shares(from, a.rect, k), shares(from, b.rect, k)),
			// Reversed, so the larger of the two is the upper — then the
			// leftmost — pane and the tie breaks that way.
			cmp.Compare(b.rect.Min.Y, a.rect.Min.Y),
			cmp.Compare(b.rect.Min.X, a.rect.Min.X),
		)
	})
	return m.setFocus(best.focus)
}

// beyond reports whether to lies wholly on the k side of from.
func beyond(from, to uv.Rectangle, k string) bool {
	switch k {
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
func shares(from, to uv.Rectangle, k string) int {
	if k == "ctrl+h" || k == "ctrl+l" {
		return min(from.Max.Y, to.Max.Y) - max(from.Min.Y, to.Min.Y)
	}
	return min(from.Max.X, to.Max.X) - max(from.Min.X, to.Min.X)
}

func (m *model) setFocus(next paneFocus) tea.Cmd {
	if (next == focusRooms || next == focusMembers) && !m.leftShown() {
		next = focusConversation
	}
	if next == m.focus {
		return nil
	}
	m.closeCompletion()
	m.pendingG = false
	m.focus = next
	if next == focusMessage {
		return m.text.Focus()
	}
	m.text.Blur()
	return nil
}

// conversationKey scrolls the log. The movement is spelled out here rather
// than handed to the viewport's own keymap: this pane scrolls vertically only
// — a message wraps instead of running off the edge — so h and l, which the
// viewport binds to horizontal scrolling, do nothing.
func (m *model) conversationKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	pending := m.pendingG
	m.pendingG = false
	switch s {
	case "q":
		return m, tea.Quit
	case "g":
		if pending {
			m.view.GotoTop()
			break
		}
		m.pendingG = true
		return m, nil
	case "G":
		m.view.GotoBottom()
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
	// but where the scroll left it, so it is read back off the viewport.
	m.following = m.view.AtBottom()
	return m, nil
}

// roomsKey moves the cursor over the room list and selects what it lands on.
func (m *model) roomsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	pending := m.pendingG
	m.pendingG = false
	// Half a page is measured off the pane, so a taller terminal moves further.
	half := max((m.rects().rooms.Dy()-2)/2, 1)
	switch s {
	case "q":
		return m, tea.Quit
	case "g":
		if pending {
			m.roomCursor = 0
			break
		}
		m.pendingG = true
		return m, nil
	case "G":
		m.roomCursor = len(m.rooms) - 1
	case "j", "down":
		m.roomCursor++
	case "k", "up":
		m.roomCursor--
	case "ctrl+d":
		m.roomCursor += half
	case "ctrl+u":
		m.roomCursor -= half
	case "enter":
		m.selectRoom(m.roomCursor)
		return m, nil
	}
	m.roomCursor = min(max(m.roomCursor, 0), len(m.rooms)-1)
	return m, nil
}

// selectRoom swaps the conversation and the members for the room's own fixture.
// A room is entered at its newest message, which is where the operator who just
// opened it is looking, and with the message they had half written in it: a
// draft belongs to the room it was addressed at, not to the screen.
func (m *model) selectRoom(i int) {
	if i < 0 || i >= len(m.rooms) || i == m.roomIndex {
		return
	}
	m.current().draft = m.text.Value()
	m.roomIndex = i
	m.rows = rosterRows(m.current().roster)
	m.cursor = 0
	m.closeCompletion()
	m.text.SetValue(m.current().draft)
	m.text.MoveToEnd()
	m.following = true
	m.layout()
}

// membersKey moves the cursor over the room and addresses what it lands on.
func (m *model) membersKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "j", "down":
		m.cursor = min(m.cursor+1, len(m.rows)-1)
	case "k", "up":
		m.cursor = max(m.cursor-1, 0)
	case "enter":
		return m, m.mention()
	}
	return m, nil
}

// mention puts the row under the cursor in front of the message and follows it
// into the textarea, which is where the operator was going anyway.
func (m *model) mention() tea.Cmd {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return nil
	}
	m.text.MoveToBegin()
	m.text.InsertString("@" + m.rows[m.cursor].address() + " ")
	cmd := m.setFocus(focusMessage)
	m.layout()
	return cmd
}

// messageKey writes the message.
func (m *model) messageKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch {
	case s == "ctrl+g":
		return m, m.openEditor()
	case s == m.sendKey || s == "ctrl+x":
		return m, m.send()
	case s == "tab":
		m.openCompletion()
		m.layout()
		return m, nil
	}
	// Typing answers whatever the system line last said, so the report goes
	// with it.
	m.notice = ""
	var cmd tea.Cmd
	m.text, cmd = m.text.Update(msg)
	m.layout()
	return m, cmd
}

// token is the `@token` the cursor sits at the end of: everything back to the
// last whitespace on the cursor's line, if it starts with an `@`.
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
	items := completions(m.current().roster, strings.TrimPrefix(tok, "@"))
	switch len(items) {
	case 0:
		m.notice = "nothing matches " + tok
	case 1:
		m.replaceToken(tok, items[0])
	default:
		m.comp = completion{open: true, items: items, token: tok}
	}
}

func (m *model) closeCompletion() {
	m.comp = completion{}
}

// completionKey drives the dropdown, and reports whether it took the key.
func (m *model) completionKey(s string) (bool, tea.Cmd) {
	n := len(m.comp.items)
	switch s {
	case "tab", "down", "j":
		// Tab on the last row accepts: a list walked to the end has made the
		// choice, and asking for one more key to say so is a key too many.
		if s == "tab" && m.comp.index == n-1 {
			m.replaceToken(m.comp.token, m.comp.items[m.comp.index])
			m.layout()
			return true, nil
		}
		m.comp.index = (m.comp.index + 1) % n
		return true, nil
	case "shift+tab", "up", "k":
		m.comp.index = (m.comp.index - 1 + n) % n
		return true, nil
	case "enter":
		m.replaceToken(m.comp.token, m.comp.items[m.comp.index])
		m.layout()
		return true, nil
	case "esc":
		m.closeCompletion()
		return true, nil
	}
	m.closeCompletion()
	return false, nil
}

// replaceToken swaps the typed token for the unambiguous address.
//
// The token is deleted one backspace at a time through the textarea's own
// Update rather than by rewriting the value: a rewrite would have to put the
// cursor back, and the textarea only moves it by visual line, which a wrapped
// message does not agree with. Backspace only fires while the textarea has
// focus, which is where completion happens.
func (m *model) replaceToken(tok string, item completionItem) {
	for range utf8.RuneCountInString(tok) {
		m.text, _ = m.text.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	m.text.InsertString("@" + item.address + " ")
	m.closeCompletion()
}

// completions lists what `@prefix` could mean: each team as `team/*` above its
// members, matched on the full address or on the bare name, since the operator
// types whichever they remember.
func completions(roster []member, prefix string) []completionItem {
	var items []completionItem
	team := ""
	for _, mem := range roster {
		if mem.team != team {
			team = mem.team
			addr := team + "/" + broadcast
			if strings.HasPrefix(addr, prefix) || strings.HasPrefix(team, prefix) {
				items = append(items, completionItem{
					address: addr,
					state:   fmt.Sprintf("%d members", teamSize(roster, team)),
				})
			}
		}
		addr := mem.team + "/" + mem.name
		if strings.HasPrefix(addr, prefix) || strings.HasPrefix(mem.name, prefix) {
			items = append(items, completionItem{address: addr, state: mem.state})
		}
	}
	return items
}

func teamSize(roster []member, team string) int {
	var n int
	for _, mem := range roster {
		if mem.team == team {
			n++
		}
	}
	return n
}

// parseAddress reads the message the way the room will: left to right, a
// backtick opening a span whose content is text, `\@` a literal `@` whose
// backslash does not travel, and the first bare `@token` the target. No bare
// `@` means the whole room. The text goes whole, target token included: it
// doubles as the mention that names who was asked.
func parseAddress(text string) (target, out string) {
	runes := []rune(text)
	var b strings.Builder
	var span bool
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case r == '`':
			span = !span
			b.WriteRune(r)
		case r == '\\' && i+1 < len(runes) && runes[i+1] == '@':
			b.WriteRune('@')
			i++
		case r == '@' && !span && target == "":
			j := i + 1
			for j < len(runes) && !unicode.IsSpace(runes[j]) {
				j++
			}
			target = string(runes[i+1 : j])
			b.WriteString(string(runes[i:j]))
			i = j - 1
		default:
			b.WriteRune(r)
		}
	}
	if target == "" {
		target = broadcast
	}
	return target, b.String()
}

// send parses what is written, clears the line and asks for it to be
// delivered. Sending is asking to be read, so whatever the operator had
// scrolled back to, the answer will be at the bottom.
func (m *model) send() tea.Cmd {
	text := strings.TrimSpace(m.text.Value())
	if text == "" {
		m.notice = "nothing to send"
		return nil
	}
	target, out := parseAddress(text)
	m.text.Reset()
	m.closeCompletion()
	m.notice = "sending to " + target
	m.following = true
	m.view.GotoBottom()
	m.layout()

	roster := m.current().roster
	from := m.roomIndex
	return tea.Tick(sendDelay, func(time.Time) tea.Msg {
		delivered, err := resolve(roster, target)
		return sentMsg{room: from, target: target, text: out, delivered: delivered, err: err}
	})
}

// resolve stands in for the daemon deciding who a target names, and counts who
// it would have reached.
func resolve(roster []member, target string) (int, error) {
	if target == broadcast {
		return len(roster), nil
	}
	if team, ok := strings.CutSuffix(target, "/"+broadcast); ok {
		if n := teamSize(roster, team); n > 0 {
			return n, nil
		}
		return 0, fmt.Errorf("team %q not found", team)
	}
	for _, mem := range roster {
		if target == mem.team+"/"+mem.name || target == mem.name {
			return 1, nil
		}
	}
	return 0, fmt.Errorf("member %q not found", target)
}

// applySent reports the delivery, and puts a message nobody took back where it
// was written.
func (m *model) applySent(msg sentMsg) {
	if msg.err != nil {
		m.notice = "not sent: " + msg.err.Error()
		// Only into an empty textarea: an operator who has already started the
		// next message keeps it.
		if m.text.Value() == "" {
			m.text.SetValue(msg.text)
			m.text.MoveToEnd()
		}
		return
	}
	m.rooms[msg.room].entries = append(m.rooms[msg.room].entries, entry{
		at: time.Now().Format(stamp), from: admin, to: msg.target, text: msg.text,
	})
	m.notice = fmt.Sprintf("sent to %s (%d delivered)", msg.target, msg.delivered)
}

// openEditor hands the message to $EDITOR, or $VISUAL, on a markdown temp file.
func (m *model) openEditor() tea.Cmd {
	name := strings.TrimSpace(os.Getenv("EDITOR"))
	if name == "" {
		name = strings.TrimSpace(os.Getenv("VISUAL"))
	}
	if name == "" {
		m.notice = "no EDITOR or VISUAL set"
		return nil
	}
	// The variable may carry arguments — `code -w`, `emacsclient -nw` — so it
	// is split the way a shell would rather than run as one word.
	argv, err := shellwords.Parse(name)
	if err != nil || len(argv) == 0 {
		m.notice = "editor exited: cannot read $EDITOR/$VISUAL"
		return nil
	}
	f, err := os.CreateTemp("", "crabswarm-chat-*.md")
	if err != nil {
		m.notice = "editor exited: " + err.Error()
		return nil
	}
	path := f.Name()
	_, werr := f.WriteString(m.text.Value())
	cerr := f.Close()
	if werr != nil || cerr != nil {
		_ = os.Remove(path)
		m.notice = "editor exited: " + firstErr(werr, cerr).Error()
		return nil
	}
	return tea.ExecProcess(exec.Command(argv[0], append(argv[1:], path)...), func(err error) tea.Msg {
		defer func() { _ = os.Remove(path) }()
		if err != nil {
			return editedMsg{err: err}
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return editedMsg{err: rerr}
		}
		return editedMsg{text: string(b), ok: true}
	})
}

// firstErr is whichever of the two errors is one.
func firstErr(a, b error) error {
	if a != nil {
		return a
	}
	return b
}

// applyEdited takes the file back, or says why it did not. Nothing is sent:
// the operator reviews what they wrote and sends it themselves.
func (m *model) applyEdited(msg editedMsg) {
	if msg.err != nil {
		m.notice = "editor exited: " + msg.err.Error()
		return
	}
	if !msg.ok {
		return
	}
	// Editors end a file with a newline the operator did not type.
	m.text.SetValue(strings.TrimRight(msg.text, "\n"))
	m.text.MoveToEnd()
	m.notice = ""
}

// appendLive adds the next canned message, standing in for the tail poll.
func (m *model) appendLive() {
	canned := []entry{
		{from: "alpha/bob", to: "*", text: "tests are green on the branch"},
		{from: "beta/dee", to: "alpha/ana", text: "does the importer need a backfill?"},
		{from: "alpha/ana", to: "beta/dee", text: "yes, one pass over the old rows"},
		{from: "beta/cy", to: "*", text: "importer is merged"},
	}
	next := canned[m.liveIndex%len(canned)]
	m.liveIndex++
	next.at = time.Now().Format(stamp)
	cur := m.current()
	cur.entries = append(cur.entries, next)
}

// rects places the regions with the constraint solver, so a resize can neither
// overlap them nor push one off the terminal.
type rects struct {
	rooms        uv.Rectangle
	members      uv.Rectangle
	conversation uv.Rectangle
	message      uv.Rectangle
	system       uv.Rectangle
	status       uv.Rectangle
	leftShown    bool
}

func (m *model) rects() rects {
	width, height := m.size()
	// The system line and the status bar run the whole width under both
	// columns, so they are split off first.
	rows := layout.Vertical(
		layout.Fill(1),
		layout.Len(1),
		layout.Len(1),
	).Split(uv.Rect(0, 0, width, height))

	r := rects{system: rows[1], status: rows[2]}
	main := rows[0]
	if width >= leftMinWidth {
		cols := layout.Horizontal(layout.Len(leftWidth), layout.Fill(1)).Split(main)
		// The rooms list asks for exactly its own rows and a border, capped at a
		// third of the column so a long list cannot crowd out the members.
		roomsHeight := min(len(m.rooms)+2, max(cols[0].Dy()/3, roomsMinHeight))
		left := layout.Vertical(layout.Len(roomsHeight), layout.Fill(1)).Split(cols[0])
		r.rooms, r.members, r.leftShown = left[0], left[1], true
		main = cols[1]
	}
	right := layout.Vertical(
		layout.Fill(1),
		layout.Len(m.text.Height()+2),
	).Split(main)
	r.conversation, r.message = right[0], right[1]
	return r
}

// size is the terminal's, floored at what the chrome needs. A terminal smaller
// than the floor is drawn cut off rather than solved for, since a solution
// that small says nothing about the screen.
func (m *model) size() (width, height int) {
	width, height = m.width, m.height
	if width <= 0 {
		width = defaultWidth
	}
	if height <= 0 {
		height = defaultHeight
	}
	return max(width, minWidth), max(height, minHeight)
}

func (m *model) leftShown() bool {
	width, _ := m.size()
	return width >= leftMinWidth
}

// layout re-sizes the regions to the terminal and refills the conversation,
// keeping the view on the newest entry while it is following the room.
func (m *model) layout() {
	r := m.rects()
	m.text.SetWidth(max(r.message.Dx()-2, 1))
	// The message pane is as tall as what is written in it, and what is
	// written in it re-wraps at the new width, so the rows are solved again
	// with the height that produced.
	r = m.rects()
	m.view.SetWidth(max(r.conversation.Dx()-2, 1))
	m.view.SetHeight(max(r.conversation.Dy()-2, 1))
	m.view.SetContent(m.conversation())
	if m.following {
		m.view.GotoBottom()
	}
	m.cursor = min(max(m.cursor, 0), max(len(m.rows)-1, 0))
	m.roomCursor = min(max(m.roomCursor, 0), max(len(m.rooms)-1, 0))
}

// conversation renders the room's log: one entry per line, naming who said it
// and who to. A message written over several lines is folded into one with the
// newlines marked, so every entry is one row and the pane stays readable.
func (m *model) conversation() string {
	var b strings.Builder
	for _, e := range m.current().entries {
		fmt.Fprintf(&b, "%s %s → %s: %s\n", e.at, e.from, e.to, mentioned(fold(e.text)))
	}
	return b.String()
}

func fold(s string) string {
	return strings.NewReplacer("\r\n", " ⏎ ", "\n", " ⏎ ", "\r", " ⏎ ").Replace(s)
}

// adminTokens finds where a line names the admin: the rune ranges of its bare
// `@admin` and `@admin/admin` tokens, each token's `@` included.
//
// Bare is read the way [parseAddress] reads it, since a mention is the same `@`
// the room was addressed with: a backtick opens a span whose content is text,
// and `\@` is a literal `@` that addresses nobody. No ranges means the line
// does not name the admin.
func adminTokens(runes []rune) [][2]int {
	var spans [][2]int
	var span bool
	for i := 0; i < len(runes); i++ {
		switch r := runes[i]; {
		case r == '`':
			span = !span
		case r == '\\' && i+1 < len(runes) && runes[i+1] == '@':
			i++
		case r == '@' && !span:
			j := i + 1
			for j < len(runes) && !unicode.IsSpace(runes[j]) {
				j++
			}
			if tok := string(runes[i+1 : j]); tok == adminName || tok == admin {
				spans = append(spans, [2]int{i, j})
			}
			i = j - 1
		}
	}
	return spans
}

// mentioned colours a line that names the admin: the message in the mention
// colour, with the tokens that name them bold on top of it. The admin has no
// member row to point at, so being named is textual — which is also why the
// colour is on the message and not on the time and sender in front of it,
// where nothing was said.
//
// The runs are rendered one at a time rather than the bold token being drawn
// inside an already-coloured line: the reset that ends the bold would end the
// colour with it. A line that does not name the admin comes back untouched.
func mentioned(text string) string {
	runes := []rune(text)
	spans := adminTokens(runes)
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
	for _, s := range spans {
		plain(runes[prev:s[0]])
		b.WriteString(mentionTokenStyle.Render(string(runes[s[0]:s[1]])))
		prev = s[1]
	}
	plain(runes[prev:])
	return b.String()
}

// roomsPane renders the room list, the selected room marked so it is still
// named once the cursor has moved off it.
func (m *model) roomsPane(width, height int) string {
	lines := make([]string, 0, len(m.rooms))
	for i, r := range m.rooms {
		mark := unselectedMark
		if i == m.roomIndex {
			mark = selectedMark
		}
		text := clip(mark+r.path, width)
		switch rest, marked := strings.CutPrefix(text, selectedMark); {
		case i == m.roomCursor && m.focus == focusRooms:
			text = pickedStyle.Render(text)
		case marked:
			// Off the cursor the mark alone is coloured: the path is the room's
			// name, not something the operator picked out.
			text = markStyle.Render(selectedMark) + rest
		}
		lines = append(lines, text)
	}
	return fit(strings.Join(window(lines, m.roomCursor, height), "\n"), width, height)
}

// membersPane renders the room's attendance grouped by team, with the cursor on
// the row enter would address.
func (m *model) membersPane(width, height int) string {
	lines := make([]string, 0, len(m.rows))
	for i, row := range m.rows {
		text := clip(rowText(row), width)
		switch {
		case i == m.cursor && m.focus == focusMembers:
			lines = append(lines, pickedStyle.Render(text))
		case row.heading():
			lines = append(lines, teamStyle.Render(text))
		default:
			lines = append(lines, text)
		}
	}
	return fit(strings.Join(window(lines, m.cursor, height), "\n"), width, height)
}

// window is the slice of a list a pane that tall can show, scrolled to keep the
// cursor in it — the cursor centred where there is list on both sides of it.
func window(lines []string, cursor, height int) []string {
	if height <= 0 || len(lines) <= height {
		return lines
	}
	first := min(max(cursor-height/2, 0), len(lines)-height)
	return lines[first : first+height]
}

// rowText is what a roster row says: a team heading, or a member and the
// harness state that says whether it can be interrupted.
func rowText(row rosterRow) string {
	if row.heading() {
		return row.team
	}
	return fmt.Sprintf(" %-*s %s", nameColumn, row.name, row.state)
}

// systemLine is the screen's last word to the operator that is not the
// conversation: a send result, an editor problem, a rejected key.
func (m *model) systemLine(width int) string {
	if m.notice == "" {
		return ""
	}
	return systemStyle.Render(clip(" "+m.notice, width))
}

// statusField is one field of the status bar: a word about where the operator
// is, or a key hint, which is the key and then what it does. The two are the
// same kind of field so the fitting below can drop either, and different enough
// that only the key takes the accent colour.
type statusField struct {
	key   string
	label string
}

// plain is the field with no colour on it, which is what the bar is measured
// and fitted as.
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

// statusBar says where the operator is and names the keys the screen cannot
// show: pane movement, the editor, the send chord, and how to leave.
func (m *model) statusBar(width int) string {
	// MaxWidth below reads a zero as no limit, where a bar with no room is
	// nothing at all.
	if width <= 0 {
		return ""
	}
	mode := "scrolled back"
	if m.following {
		mode = "tailing"
	}
	send := statusField{"^enter", "sends"}
	if m.sendKey == "ctrl+x" {
		send = statusField{"ctrl+x", "sends (terminal cannot report ctrl+enter)"}
	}
	// The bar as written is wider than an 80-column terminal, which would lose
	// the end of it — and the end is the half that cannot be guessed from the
	// screen. What goes first is therefore the state the panes already show:
	// the connection, then the room, then whether the view is following.
	where := statusField{label: "room " + m.current().path}
	parts := []statusField{
		where,
		{label: mode},
		{label: "connected"},
		{"^hjkl", "panes"},
		{"^g", "editor"},
		send,
		{"q", "quits"},
	}
	plain := func() string {
		words := make([]string, len(parts))
		for i, p := range parts {
			words[i] = p.plain()
		}
		return strings.Join(words, " · ")
	}
	for _, drop := range []statusField{{label: "connected"}, where, {label: mode}} {
		if lipgloss.Width(plain())+1 <= width {
			break
		}
		parts = slices.DeleteFunc(parts, func(p statusField) bool { return p == drop })
	}
	words := make([]string, len(parts))
	for i, p := range parts {
		words[i] = p.render()
	}
	// The bar is coloured before it is cut, so the cut is the one that counts
	// escape sequences as the nothing they are wide.
	return lipgloss.NewStyle().MaxWidth(width).
		Render(" " + strings.Join(words, statusStyle.Render(" · ")))
}

func (m *model) View() tea.View {
	width, height := m.size()
	r := m.rects()

	layers := []*lipgloss.Layer{
		paneLayer("conversation", m.view.View(), r.conversation, m.focus == focusConversation),
		paneLayer("message", m.text.View(), r.message, m.focus == focusMessage),
		lipgloss.NewLayer(m.systemLine(r.system.Dx())).X(r.system.Min.X).Y(r.system.Min.Y),
		lipgloss.NewLayer(m.statusBar(r.status.Dx())).X(r.status.Min.X).Y(r.status.Min.Y),
	}
	if r.leftShown {
		rooms := fmt.Sprintf("rooms (%d)", len(m.rooms))
		layers = append(layers, paneLayer(
			rooms,
			m.roomsPane(r.rooms.Dx()-2, r.rooms.Dy()-2),
			r.rooms,
			m.focus == focusRooms,
		))
		members := fmt.Sprintf("members (%d)", len(m.current().roster))
		layers = append(layers, paneLayer(
			members,
			m.membersPane(r.members.Dx()-2, r.members.Dy()-2),
			r.members,
			m.focus == focusMembers,
		))
	}
	if m.comp.open {
		layers = append(layers, m.dropdownLayer(r, width))
	}

	canvas := lipgloss.NewCanvas(width, height)
	canvas.Compose(lipgloss.NewCompositor(layers...))

	v := tea.NewView(canvas.Render())
	// The screen is a place the operator stays, not output scrolling past.
	v.AltScreen = true
	// Asking for every key as an escape code is what lets ctrl+enter be told
	// apart from enter in the terminals that can do it. Terminals that cannot
	// simply do not answer, which is what the ctrl+x fallback is for.
	v.KeyboardEnhancements.ReportAllKeysAsEscapeCodes = true
	return v
}

// dropdownLayer draws the completion list over whatever is above the message
// pane, left-aligned to the token it is completing.
func (m *model) dropdownLayer(r rects, screenWidth int) *lipgloss.Layer {
	items := m.comp.items
	index := m.comp.index
	// A list longer than the cap scrolls with the selection rather than
	// covering the conversation.
	first := 0
	if len(items) > dropdownMaxRows {
		first = min(max(index-dropdownMaxRows/2, 0), len(items)-dropdownMaxRows)
		items = items[first : first+dropdownMaxRows]
	}

	var addrWidth, stateWidth int
	for _, it := range items {
		addrWidth = max(addrWidth, lipgloss.Width("@"+it.address))
		stateWidth = max(stateWidth, lipgloss.Width(it.state))
	}
	inner := addrWidth + 2 + stateWidth
	lines := make([]string, 0, len(items))
	for i, it := range items {
		line := fmt.Sprintf("%-*s  %-*s", addrWidth, "@"+it.address, stateWidth, it.state)
		if first+i == index {
			line = pickedStyle.Render(line)
		}
		lines = append(lines, line)
	}

	content := box("", fit(strings.Join(lines, "\n"), inner, len(items)), inner+2, len(items)+2, true)
	boxWidth, boxHeight := inner+2, len(items)+2

	// The list points at the token: its left edge is the token's column,
	// inside the message pane's border, and it sits on top of that border.
	li := m.text.LineInfo()
	x := r.message.Min.X + 1 + max(li.CharOffset-lipgloss.Width(m.comp.token), 0)
	x = min(max(x, 0), max(screenWidth-boxWidth, 0))
	y := max(r.message.Min.Y-boxHeight, 0)
	return lipgloss.NewLayer(content).X(x).Y(y).Z(1)
}

// paneLayer positions a titled, bordered pane at the rectangle the solver gave
// it.
func paneLayer(title, content string, r uv.Rectangle, focused bool) *lipgloss.Layer {
	body := fit(content, r.Dx()-2, r.Dy()-2)
	return lipgloss.NewLayer(box(title, body, r.Dx(), r.Dy(), focused)).
		X(r.Min.X).Y(r.Min.Y)
}

// box draws content inside a rounded border of exactly width×height cells with
// the title written into the top edge — purple and bold while the pane has
// focus, and a shade lighter than its own frame while it does not, so a blurred
// pane is still named without being read first.
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

	// No Width/Height on the style: lipgloss measures a block including its
	// border, so a width asked for here would come off the content and the
	// pane would draw two cells narrower than the rectangle it was given.
	// [fit] has already made the content exactly the inside of the box.
	body := lipgloss.NewStyle().
		Border(border).BorderTop(false).BorderForeground(edge).
		Render(content)

	label := ""
	if title != "" {
		label = " " + clip(title, max(width-5, 0)) + " "
	}
	fill := max(width-3-lipgloss.Width(label), 0)
	top := edgeStyle.Render(border.TopLeft+border.Top) +
		titleStyle.Render(label) +
		edgeStyle.Render(strings.Repeat(border.Top, fill)+border.TopRight)
	return top + "\n" + body
}

// fit makes a block exactly width×height cells, since a pane that outgrows its
// rectangle is drawn over its neighbour.
//
// Lines are cut and padded one at a time rather than by a style's Width, which
// re-wraps what the viewport already wrapped and returns a taller block than
// it was given. MaxWidth does the cutting because the blocks passed through
// here — the viewport, the textarea, the roster — carry styling that a
// rune-counting cut would break in the middle of an escape sequence.
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
// under moves.
func clip(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(s)
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

func main() {
	if _, err := tea.NewProgram(newModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "mock:", err)
		os.Exit(1)
	}
}
