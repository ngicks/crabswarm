package tui

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

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

	// focus is the pane the keys reach. It is the screen's only mode: there is
	// no watching and no writing, only which of the four panes is being talked
	// to.
	focus paneFocus
	// showLeft says which column the body holds where it can hold only one,
	// which is every terminal narrower than [leftMinWidth]. Above that width it
	// is not read: both columns are on screen.
	showLeft bool
	// pendingG is the first g of a gg, which is the one key on the screen that
	// means nothing until the next one arrives.
	pendingG bool
	// roomsCursor and membersCursor are the rows the two left panes are pointed
	// at, which is what enter acts on there. They are kept on a row that exists
	// by [model.layout], since both lists change under them.
	roomsCursor   int
	membersCursor int

	// entries is the conversation, oldest first, exactly as the log handed it
	// over — the log is the only source of what the room said.
	entries []*chatv1.AdminHistoryEntry
	roster  []*chatv1.Member
	// rooms is the listing as the daemon last gave it: every room it knows and
	// who attends it. It rides the roster poll — one reply fills both left
	// panes — and it is held whole rather than as names, so the room switched
	// to has its attendance before a poll of its own comes back.
	rooms []*chatv1.Room
	// drafts is the message left half-written in each room other than the one
	// on screen, whose draft is in the input line. A draft belongs to the room
	// it was addressed at and lives no longer than the screen.
	drafts map[string]string

	// cursor is the id of the newest entry the screen holds, which is what the
	// next read of the log asks to be told about.
	cursor int64
	// tailGen counts the rooms this screen has watched, and every read of the
	// log is stamped with it. A read is in flight when the operator switches
	// rooms; what comes back is the other room's conversation, and the stamp is
	// what says so — a name would not, since switching away and back inside one
	// interval would let a stale reply pass for this room's.
	tailGen int

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
	// not the conversation — a rejected message, mostly. It is the system
	// line's text.
	notice string
}

// newModel is the screen as it stands the moment the room lookup answered:
// holding the listing it will draw its left column from, opened on the room
// that lookup chose — which is no room at all when the daemon knew none.
func newModel(ctx context.Context, deps Deps, room string, rooms []*chatv1.Room) *model {
	input := textinput.New()
	input.Prompt = "> "
	input.Placeholder = `team/name: text`
	styles := input.Styles()
	styles.Focused.Prompt = promptStyle
	styles.Blurred.Prompt = promptStyle
	input.SetStyles(styles)

	view := viewport.New()
	// A conversation is read, not scrolled sideways: a long message wraps
	// rather than running off the edge of the pane.
	view.SoftWrap = true

	m := &model{
		ctx:    ctx,
		deps:   deps,
		room:   room,
		width:  defaultWidth,
		height: defaultHeight,
		view:   view,
		input:  input,
		// The listing the room was chosen out of is the one the rooms pane
		// draws, so both left panes are filled from the first frame rather than
		// from the first poll.
		rooms:     rooms,
		roster:    membersOf(rooms, room),
		drafts:    map[string]string{},
		following: true,
		// The conversation is what the operator opened the screen to read.
		focus: focusConversation,
	}
	// The rooms cursor opens on the room being watched, so an enter that moved
	// nothing selects the room already on screen rather than the first listed.
	m.roomsCursor = max(slices.IndexFunc(rooms, func(r *chatv1.Room) bool {
		return r.GetName() == room
	}), 0)
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

// key is the whole router: leaving the screen and moving between panes are the
// screen's own, and everything else is the focused pane's. A pane never sees a
// movement key, and no key means one thing in one pane and another in the next
// unless that pane says so.
func (m *model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch s := msg.String(); s {
	case "ctrl+c":
		return m, tea.Quit
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

// messageKey edits the line being written, and sends it.
func (m *model) messageKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" {
		return m, m.submit()
	}
	// Typing answers whatever the system line last said, so the report goes
	// with it.
	m.notice = ""
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// systemLine is the screen's last word to the operator that is not the
// conversation: a send result, a rejected message.
func (m *model) systemLine(width int) string {
	if m.notice == "" {
		return ""
	}
	return systemStyle.Render(clip(" "+m.notice, width))
}

// statusBar says where the operator is — which room, whether the view still
// follows it, how the daemon is answering — and names the keys the screen
// cannot show.
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
	room := m.room
	if room == "" {
		// The screen opened on a daemon that knew no rooms at all; the rooms
		// pane fills on a later poll and the operator picks one there.
		room = "(none)"
	}
	// The daemon's errors carry a second line of hint, and the bar is one line:
	// the whole screen would shift down otherwise, so what goes on it is folded
	// before it is coloured.
	parts := []string{
		statusStyle.Render(clip("room "+room, width)),
		statusStyle.Render(mode),
		statusStyle.Render(clip(m.connection(), width)),
		keyStyle.Render("^hjkl") + statusStyle.Render(" panes"),
		keyStyle.Render("enter") + statusStyle.Render(" sends"),
		keyStyle.Render("q") + statusStyle.Render(" quits"),
	}
	// The bar is coloured before it is cut, so the cut is the one that counts
	// escape sequences as the nothing they are wide.
	return lipgloss.NewStyle().MaxWidth(width).
		Render(" " + strings.Join(parts, statusStyle.Render(" · ")))
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

func (m *model) View() tea.View {
	width, height := m.size()
	r := m.rects()

	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(m.systemLine(r.system.Dx())).
			X(r.system.Min.X).Y(r.system.Min.Y),
		lipgloss.NewLayer(m.statusBar(r.status.Dx())).
			X(r.status.Min.X).Y(r.status.Min.Y),
	}
	if r.leftShown {
		layers = append(layers,
			paneLayer(
				fmt.Sprintf("rooms (%d)", len(m.rooms)),
				m.roomsPane(r.rooms.Dx()-2, r.rooms.Dy()-2),
				r.rooms,
				m.focus == focusRooms,
			),
			paneLayer(
				fmt.Sprintf("members (%d)", len(m.roster)),
				m.membersPane(r.members.Dx()-2, r.members.Dy()-2),
				r.members,
				m.focus == focusMembers,
			),
		)
	}
	if r.rightShown {
		layers = append(layers,
			paneLayer("conversation", m.view.View(), r.conversation,
				m.focus == focusConversation),
			paneLayer("message", m.input.View(), r.message,
				m.focus == focusMessage),
		)
	}

	canvas := lipgloss.NewCanvas(width, height)
	canvas.Compose(lipgloss.NewCompositor(layers...))
	content := canvas.Render()
	// A terminal below the floor the panes are solved at is shown the part of
	// the screen it has room for rather than a screen larger than itself.
	if tw, th := m.termSize(); tw < width || th < height {
		content = fit(content, tw, th)
	}

	v := tea.NewView(content)
	// The screen is a place the operator stays, not output scrolling past, so
	// it takes the alternate buffer and gives the shell back untouched on exit.
	v.AltScreen = true
	return v
}
