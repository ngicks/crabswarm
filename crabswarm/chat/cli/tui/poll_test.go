package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// The fakes below stand in for the daemon and remember what they were asked
// for. The commands under test are run by the test itself, one at a time, so
// nothing here has to be safe for concurrent use.

type logCall struct {
	room  string
	since int64
	limit int32
}

type fakeLog struct {
	calls []logCall
	reply func(call logCall) ([]*chatv1.AdminHistoryEntry, error)
}

func (f *fakeLog) RoomLog(
	_ context.Context,
	room string,
	sinceID int64,
	limit int32,
) ([]*chatv1.AdminHistoryEntry, error) {
	call := logCall{room: room, since: sinceID, limit: limit}
	f.calls = append(f.calls, call)
	if f.reply == nil {
		return nil, nil
	}
	return f.reply(call)
}

type fakeRoster struct {
	calls int
	rooms []*chatv1.Room
	err   error
}

func (f *fakeRoster) Rooms(_ context.Context) ([]*chatv1.Room, error) {
	f.calls++
	return f.rooms, f.err
}

// step runs one command the way the program would and feeds back what it
// produced, handing over whatever the model asked for next. A batch is run
// command by command. The command that comes back is not run: the loops end
// each round with a tick, and running one would sit out its whole interval.
func step(t *testing.T, m *model, cmd tea.Cmd) (*model, tea.Cmd) {
	t.Helper()
	assert.Assert(t, cmd != nil, "no command to run")
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var next tea.Cmd
		for _, sub := range batch {
			m, next = step(t, m, sub)
		}
		return m, next
	}
	next, out := m.Update(msg)
	updated, ok := next.(*model)
	assert.Assert(t, ok, "update returned %T", next)
	return updated, out
}

// The screen opens on the tail of the log and then reads forward from what it
// already has: an operator arriving an hour late reads the conversation that
// happened without them, and every read after that asks only for what is new.
func TestOpeningReadsTheTailThenFollowsTheCursor(t *testing.T) {
	log := &fakeLog{
		reply: func(call logCall) ([]*chatv1.AdminHistoryEntry, error) {
			if call.since == 0 {
				return fixtureEntries(3), nil
			}
			return fixtureEntries(4)[3:], nil
		},
	}
	roster := &fakeRoster{
		rooms: []*chatv1.Room{{Name: fixtureRoom, Members: fixtureRoster()}},
	}
	m := fixtureModel(t, Deps{Log: log, Roster: roster})

	m, _ = step(t, m, m.Init())
	assert.Equal(t, len(log.calls), 1)
	assert.Equal(t, log.calls[0], logCall{room: fixtureRoom, since: 0, limit: backfill})
	assert.Equal(t, m.cursor, int64(3))
	assert.Equal(t, roster.calls, 1)
	assert.Assert(t, strings.Contains(m.View().Content, "line 3"))

	m, next := step(t, m, tailTick(t, m))
	assert.Equal(t, len(log.calls), 2)
	assert.Equal(t, log.calls[1], logCall{room: fixtureRoom, since: 3, limit: tailLimit})
	assert.Equal(t, m.cursor, int64(4))
	assert.Assert(t, strings.Contains(m.View().Content, "line 4"))
	// The loop starts its clock again rather than stopping after one round.
	assert.Assert(t, next != nil)
}

// tailTick is the command the tail loop's clock hands back when it fires.
func tailTick(t *testing.T, m *model) tea.Cmd {
	t.Helper()
	_, cmd := m.Update(tailTickMsg{})
	assert.Assert(t, cmd != nil)
	return cmd
}

// The roster is re-read on its own, slower clock, so a member that started
// working shows as working without the operator asking.
func TestRosterPollTracksAttendanceAndState(t *testing.T) {
	roster := &fakeRoster{
		rooms: []*chatv1.Room{{
			Name: fixtureRoom,
			Members: []*chatv1.Member{{
				Team:  "backend",
				Name:  "alice",
				Room:  fixtureRoom,
				State: chatv1.HarnessState_HARNESS_STATE_DONE,
			}},
		}},
	}
	m := fixtureModel(t, Deps{Log: &fakeLog{}, Roster: roster})

	_, cmd := m.Update(rosterTickMsg{})
	m, _ = step(t, m, cmd)
	assert.Equal(t, len(m.roster), 1)
	r := m.rects()
	assert.Assert(t, strings.Contains(m.membersPane(r.members.Dx()-2, r.members.Dy()-2), "done"))

	// A room everyone has left is still a room being watched: what was said in
	// it is there to read, so an empty roster is not an error.
	roster.rooms = []*chatv1.Room{{Name: "/work/other"}}
	_, cmd = m.Update(rosterTickMsg{})
	m, _ = step(t, m, cmd)
	assert.Equal(t, len(m.roster), 0)
	assert.Assert(t, strings.Contains(m.statusBar(defaultWidth), "connected"))
}

// A daemon that stopped answering says so on the status bar — a quiet room and
// a dead socket look identical otherwise — and the loop keeps asking, so the
// screen recovers on its own when the daemon comes back.
func TestAFailedReadIsReportedAndRetried(t *testing.T) {
	failure := errors.New("chat daemon unreachable")
	log := &fakeLog{
		reply: func(logCall) ([]*chatv1.AdminHistoryEntry, error) { return nil, failure },
	}
	roster := &fakeRoster{err: failure}
	m := fixtureModel(t, Deps{Log: log, Roster: roster})

	m, next := step(t, m, m.tail())
	assert.Assert(t, strings.Contains(m.statusBar(defaultWidth), "log unread"))
	assert.Assert(t, strings.Contains(m.statusBar(defaultWidth), failure.Error()))
	assert.Assert(t, next != nil)

	m, next = step(t, m, m.pollRoster())
	// The conversation is the point of the screen, so its failure is the one
	// reported while both are failing.
	assert.Assert(t, strings.Contains(m.statusBar(defaultWidth), "log unread"))
	assert.Assert(t, next != nil)

	log.reply = func(logCall) ([]*chatv1.AdminHistoryEntry, error) {
		return fixtureEntries(1), nil
	}
	m, _ = step(t, m, m.tail())
	// The log is being read again, so what was behind it is what the bar has
	// left to report.
	assert.Assert(t, strings.Contains(m.statusBar(defaultWidth), "roster unread"))

	roster.err = nil
	m, _ = step(t, m, m.pollRoster())
	assert.Assert(t, strings.Contains(m.statusBar(defaultWidth), "connected"))
}

// The daemon's errors carry a second line of hint, and the status bar is one
// line: the whole screen would shift down otherwise.
func TestAMultiLineErrorStaysOnOneLine(t *testing.T) {
	log := &fakeLog{
		reply: func(logCall) ([]*chatv1.AdminHistoryEntry, error) {
			return nil, errors.New(
				"chat daemon unreachable\nhint: start it by running `crabswarm serve`")
		},
	}
	m := fixtureModel(t, Deps{Log: log, Roster: &fakeRoster{}})
	m = update(t, m, tea.WindowSizeMsg{Width: 100, Height: 12})
	m, _ = step(t, m, m.tail())

	assert.Assert(t, strings.Contains(m.statusBar(defaultWidth), "chat daemon unreachable"))
	bar := m.statusBar(defaultWidth)
	assert.Assert(t, !strings.Contains(bar, "\n"), "status bar = %q", bar)
	assert.Equal(t, len(strings.Split(m.View().Content, "\n")), 12)
}

// The screen drops the oldest lines rather than growing forever — but not out
// from under a reader who has scrolled into them.
func TestScrollbackIsTrimmedOnlyWhileFollowing(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m.entries = fixtureEntries(scrollback)
	m.cursor = int64(scrollback)

	m.applyTail(tailMsg{entries: fixtureEntries(scrollback + 5)[scrollback:]})
	assert.Equal(t, len(m.entries), scrollback)
	assert.Equal(t, m.entries[0].GetId(), int64(6))

	m.following = false
	m.applyTail(tailMsg{entries: fixtureEntries(scrollback + 10)[scrollback+5:]})
	assert.Equal(t, len(m.entries), scrollback+5)
	assert.Equal(t, m.entries[0].GetId(), int64(6))
}
