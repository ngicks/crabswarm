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
	// gen stamps the read with the room it was started for, counted rather than
	// named: see [model.tailGen].
	gen     int
	entries []*chatv1.AdminHistoryEntry
	err     error
}

type rosterMsg struct {
	// rooms is the listing whole: every room the daemon knows and who attends
	// it. One read fills both left panes rather than a second loop asking for
	// what the first was told, and which room the members come from is decided
	// where the reply lands rather than where it was asked for — a reply that
	// arrives after a room switch is still the truth about both rooms.
	rooms []*chatv1.Room
	err   error
}

// tail reads what the room said after the cursor — or, with no cursor yet, the
// stretch of it the screen opens on.
func (m *model) tail() tea.Cmd {
	// A screen opened on a daemon that knew no rooms has nothing to read. The
	// clock is started anyway, so the first room picked in the rooms pane is
	// read an interval later rather than never.
	if m.room == "" {
		return tickTail()
	}
	ctx, log, room, since, gen := m.ctx, m.deps.Log, m.room, m.cursor, m.tailGen
	limit := int32(tailLimit)
	if since == 0 {
		limit = backfill
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, callTimeout)
		defer cancel()
		entries, err := log.RoomLog(ctx, room, since, limit)
		return tailMsg{gen: gen, entries: entries, err: err}
	}
}

// pollRoster re-reads what rooms there are, who is attending them and in what
// state — one listing, which both left panes are drawn from.
func (m *model) pollRoster() tea.Cmd {
	ctx, roster := m.ctx, m.deps.Roster
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, callTimeout)
		defer cancel()
		rooms, err := roster.Rooms(ctx)
		return rosterMsg{rooms: rooms, err: err}
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

// applyTail takes in what the room said since the last read. A read that was in
// flight when the operator switched rooms is dropped: what it brought back is
// the other room's conversation, and its cursor is not this room's.
func (m *model) applyTail(msg tailMsg) {
	if msg.gen != m.tailGen {
		return
	}
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

// applyRoster takes in the listing as of the last read: what rooms there are,
// and who is in the one on screen.
func (m *model) applyRoster(msg rosterMsg) {
	m.rosterErr = msg.err
	if msg.err != nil {
		return
	}
	m.rooms = msg.rooms
	// The room emptying out is not a failure: what was said in it is still
	// there to read, and somebody may yet join. Nor is the room being gone from
	// the listing: the rooms pane says so, and the conversation stays.
	m.roster = membersOf(msg.rooms, m.room)
	// The rooms pane asks for its own rows, so a longer or shorter list moves
	// every rectangle under it — and may have moved out from under a cursor.
	m.layout()
}
