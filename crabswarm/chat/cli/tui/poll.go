package tui

import (
	"context"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// The screen keeps itself current by asking, on two clocks. The daemon has no
// feed to subscribe to yet, so the conversation is re-read by cursor often
// enough to feel live, and the roster — which changes when somebody joins,
// leaves or reports a state — is re-read rarely enough to cost nothing.
const (
	tailInterval   = time.Second
	rosterInterval = 5 * time.Second
	// callTimeout bounds one poll. A daemon that stops answering mid-session
	// would otherwise leave the loop waiting on a call that never returns,
	// which is a screen that has quietly stopped updating rather than one that
	// says the connection is gone.
	callTimeout = 3 * time.Second
	// backfill is how many entries the screen opens with: the scrollback the
	// operator arriving late reads back through. The daemon clamps it to what
	// its retention actually kept, and there is no reading further back than
	// that — the log is read forward from a cursor or from its tail, never
	// backwards past one.
	backfill = 500
	// tailLimit caps one poll of what was said since the last one. Generous
	// for a second of a busy room, and a room busier than that catches up over
	// the polls that follow.
	tailLimit = 200
	// scrollback is how many entries the screen keeps in hand. Old lines are
	// dropped rather than held forever, since a screen left running for days
	// is the point of it.
	scrollback = 2000
)

// tailTickMsg and rosterTickMsg are the two clocks; tailMsg and rosterMsg are
// what came back from the read each one asked for.
type (
	tailTickMsg   struct{}
	rosterTickMsg struct{}
)

type tailMsg struct {
	entries []*chatv1.AdminHistoryEntry
	err     error
}

type rosterMsg struct {
	members []*chatv1.Member
	// rooms is every room the reply named. The listing the roster comes from
	// already carries them, so one read fills both left panes rather than a
	// second loop asking for what the first one was told.
	rooms []string
	err   error
}

// tail reads what the room said after the cursor — or, with no cursor yet, the
// stretch of it the screen opens on.
func (m *model) tail() tea.Cmd {
	ctx, log, room, since := m.ctx, m.deps.Log, m.room, m.cursor
	limit := int32(tailLimit)
	if since == 0 {
		limit = backfill
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, callTimeout)
		defer cancel()
		entries, err := log.RoomLog(ctx, room, since, limit)
		return tailMsg{entries: entries, err: err}
	}
}

// pollRoster re-reads who is attending and in what state.
func (m *model) pollRoster() tea.Cmd {
	ctx, roster, room := m.ctx, m.deps.Roster, m.room
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, callTimeout)
		defer cancel()
		rooms, err := roster.Rooms(ctx)
		if err != nil {
			return rosterMsg{err: err}
		}
		msg := rosterMsg{rooms: make([]string, 0, len(rooms))}
		for _, r := range rooms {
			msg.rooms = append(msg.rooms, r.GetName())
			if r.GetName() == room {
				msg.members = r.GetMembers()
			}
		}
		// The room emptying out is not a failure: what was said in it is still
		// there to read, and somebody may yet join. Nor is the room being gone
		// from the listing: the rooms pane says so, and the conversation stays.
		return msg
	}
}

// tickTail and tickRoster start the wait before the next read of their kind.
// Each loop waits for its own read to come back before it starts its clock
// again, so a slow daemon is asked once at a time rather than piled on.
func tickTail() tea.Cmd {
	return tea.Tick(tailInterval, func(time.Time) tea.Msg { return tailTickMsg{} })
}

func tickRoster() tea.Cmd {
	return tea.Tick(rosterInterval, func(time.Time) tea.Msg { return rosterTickMsg{} })
}

// applyTail takes in what the room said since the last read.
func (m *model) applyTail(msg tailMsg) {
	m.tailErr = msg.err
	if msg.err != nil {
		return
	}
	if len(msg.entries) > 0 {
		m.entries = append(m.entries, msg.entries...)
		m.cursor = msg.entries[len(msg.entries)-1].GetId()
		// Dropped only while the view is following: taking the oldest lines
		// out from under a reader scrolled into them would move what they are
		// reading mid-sentence.
		if excess := len(m.entries) - scrollback; excess > 0 && m.following {
			m.entries = slices.Delete(m.entries, 0, excess)
		}
		m.layout()
	}
}

// applyRoster takes in the attendance as of the last read.
func (m *model) applyRoster(msg rosterMsg) {
	m.rosterErr = msg.err
	if msg.err != nil {
		return
	}
	m.roster = msg.members
	m.rooms = msg.rooms
	// The rooms pane asks for its own rows, so a longer or shorter list moves
	// every rectangle under it.
	m.layout()
}
