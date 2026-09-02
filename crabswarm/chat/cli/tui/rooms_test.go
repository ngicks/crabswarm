package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// twoRooms is the listing both rooms are in, which is what a switch moves
// between.
func twoRooms() []*chatv1.Room {
	return append(fixtureListing(), fixtureOtherRoom())
}

// Selecting a room in the rooms pane moves the whole screen to it: the
// conversation is the new room's, read from its newest entries by the next poll
// rather than continued from the old room's cursor; the members are the new
// room's; and the status bar says where the operator now is.
func TestSelectingARoomMovesTheScreenToIt(t *testing.T) {
	log := &fakeLog{
		reply: func(call logCall) ([]*chatv1.AdminHistoryEntry, error) {
			if call.room == otherRoom {
				return fixtureEntries(2), nil
			}
			return fixtureEntries(5), nil
		},
	}
	m := fixtureModel(t, Deps{Log: log, Roster: &fakeRoster{rooms: twoRooms()}})

	// One round of both loops: the screen is watching the room it opened on and
	// the rooms pane has the listing.
	m, _ = step(t, m, m.Init())
	assert.Equal(t, m.cursor, int64(5))
	assert.Equal(t, len(m.rooms), 2)

	m.setFocus(focusRooms)
	m = update(t, m, press('j', "j"))
	m = update(t, m, press(tea.KeyEnter, ""))

	assert.Equal(t, m.room, otherRoom)
	// What the room just left said is not this room's conversation, and its
	// cursor is not this room's either.
	assert.Equal(t, len(m.entries), 0)
	assert.Equal(t, m.cursor, int64(0))
	assert.Assert(t, m.following)
	// The members come out of the listing the rooms pane is drawn from, so the
	// pane is right before a roster poll of its own comes back.
	assert.Equal(t, len(m.roster), 1)
	assert.Equal(t, m.roster[0].GetName(), "dee")
	assert.Assert(t, strings.Contains(m.statusBar(defaultWidth), "room "+otherRoom))

	// The next tail poll reads the new room, from its tail rather than from the
	// old room's cursor.
	m, _ = step(t, m, tailTick(t, m))
	assert.Equal(t, log.calls[len(log.calls)-1],
		logCall{room: otherRoom, since: 0, limit: backfill})
	assert.Assert(t, strings.Contains(m.View().Content, "line 2"))

	// The rooms pane marks the room now on screen, and the cursor is on it.
	r := m.rects()
	rooms := m.roomsPane(r.rooms.Dx()-2, r.rooms.Dy()-2)
	assert.Assert(t, strings.Contains(rooms, selectedMark+otherRoom),
		"the room switched to is not marked:\n%s", rooms)
}

// A read of the log that was in flight when the operator left the room brings
// back the other room's conversation. It is dropped rather than applied — and
// dropped even when the operator has switched back, since the cursor it was
// read with is not this room's cursor any more.
func TestAReadInFlightForTheRoomLeftIsDropped(t *testing.T) {
	log := &fakeLog{
		reply: func(logCall) ([]*chatv1.AdminHistoryEntry, error) {
			return fixtureEntries(3), nil
		},
	}
	m := fixtureModel(t, Deps{Log: log, Roster: &fakeRoster{}})
	m.rooms = twoRooms()
	m.cursor = 9

	inFlight := m.tail()
	m.selectRoom(otherRoom)
	m = update(t, m, inFlight())
	assert.Equal(t, len(m.entries), 0)
	assert.Equal(t, m.cursor, int64(0))

	// Away and back inside one interval: the reply is still the read that was
	// started before either switch.
	inFlight = m.tail()
	m.selectRoom(fixtureRoom)
	m.selectRoom(otherRoom)
	m = update(t, m, inFlight())
	assert.Equal(t, len(m.entries), 0)
	assert.Equal(t, m.cursor, int64(0))
}

// The message half written in a room stays with it: an `@team/name` is
// addressed at the team in that room, not at the screen. Coming back finds the
// text where it was left, with the cursor at the end of it.
func TestEachRoomKeepsItsOwnDraft(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m.rooms = twoRooms()

	m = typeLine(t, m, "hold the deploy")
	assert.Equal(t, m.text.Value(), "hold the deploy")

	m.selectRoom(otherRoom)
	assert.Equal(t, m.text.Value(), "", "the other room's line is not empty")
	m = typeLine(t, m, "ship it")

	m.selectRoom(fixtureRoom)
	assert.Equal(t, m.text.Value(), "hold the deploy")
	assert.Equal(t, m.text.Line(), 0)
	assert.Equal(t, m.text.Column(), len("hold the deploy"))

	m.selectRoom(otherRoom)
	assert.Equal(t, m.text.Value(), "ship it")
	assert.Equal(t, m.text.Line(), 0)
	assert.Equal(t, m.text.Column(), len("ship it"))
}

// The rooms cursor moves the way every list on the screen moves, and enter on
// the room already on screen changes nothing.
func TestTheRoomsCursorMovesAndSelects(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m.rooms = fixtureRooms(6)
	m.setFocus(focusRooms)

	// The cursor opens on the room being watched, which is the first of these.
	assert.Equal(t, m.roomsCursor, 0)
	m = update(t, m, press(tea.KeyEnter, ""))
	assert.Equal(t, m.room, fixtureRoom)

	m = update(t, m, press('G', "G"))
	assert.Equal(t, m.roomsCursor, 5)
	m = update(t, m, press('g', "g"))
	m = update(t, m, press('g', "g"))
	assert.Equal(t, m.roomsCursor, 0)
	m = update(t, m, press('j', "j"))
	m = update(t, m, press('j', "j"))
	assert.Equal(t, m.roomsCursor, 2)
	m = update(t, m, press('k', "k"))
	assert.Equal(t, m.roomsCursor, 1)
	// Half a page is measured off the pane, which holds five rows at this size.
	m = update(t, m, ctrlPress('d'))
	assert.Equal(t, m.roomsCursor, 3)
	m = update(t, m, ctrlPress('u'))
	assert.Equal(t, m.roomsCursor, 1)

	m = update(t, m, press(tea.KeyEnter, ""))
	assert.Equal(t, m.room, "/work/proj1")
}

// A list that shrank under the cursor — the daemon stopped listing a room,
// members left — leaves it on a row that is no longer there.
func TestTheCursorsStayOnRowsThatExist(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m.rooms = fixtureRooms(6)
	m.roster = fixtureCrowd(3, 5)
	m.setFocus(focusRooms)
	m = update(t, m, press('G', "G"))
	m.setFocus(focusMembers)
	m = update(t, m, press('G', "G"))
	assert.Equal(t, m.roomsCursor, 5)
	assert.Equal(t, m.membersCursor, len(rosterRows(m.roster))-1)

	m.applyRoster(rosterMsg{rooms: fixtureListing()})
	assert.Equal(t, m.roomsCursor, 0)
	assert.Equal(t, m.membersCursor, len(rosterRows(m.roster))-1)

	// And an empty listing holds them at zero rather than at nothing.
	m.applyRoster(rosterMsg{})
	assert.Equal(t, m.roomsCursor, 0)
	assert.Equal(t, m.membersCursor, 0)
	assert.Equal(t, len(m.roster), 0)
}

// A room path too long for the column is cut at its head: rooms under one tree
// differ at the end of the path, which is what a cut from the other side takes.
func TestALongRoomPathKeepsItsTail(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m.rooms = []*chatv1.Room{{Name: "/home/operator/gitrepo/crabswarm/worktree/main"}}
	m.room = m.rooms[0].GetName()

	pane := m.roomsPane(leftWidth-2, 3)
	assert.Assert(t, strings.Contains(pane, "worktree/main"),
		"the end of the path was cut off:\n%s", pane)
	assert.Assert(t, strings.Contains(pane, "…"),
		"the cut is not marked:\n%s", pane)
}

// The room the screen opens on is the one named, or the first the daemon lists
// when none is named. A named room the daemon does not know is refused before
// the terminal is taken over; a daemon that knows no rooms at all refuses
// nothing, since the rooms pane fills on a later poll.
func TestOpeningChoosesTheRoomOrTheFirstListed(t *testing.T) {
	roster := &fakeRoster{rooms: twoRooms()}

	room, rooms, err := openRoom(t.Context(), Deps{Roster: roster})
	assert.NilError(t, err)
	assert.Equal(t, room, fixtureRoom)
	assert.Equal(t, len(rooms), 2)

	room, _, err = openRoom(t.Context(), Deps{Room: otherRoom, Roster: roster})
	assert.NilError(t, err)
	assert.Equal(t, room, otherRoom)

	_, _, err = openRoom(t.Context(), Deps{Room: "/work/nowhere", Roster: roster})
	assert.ErrorContains(t, err,
		`no room "/work/nowhere": the daemon knows `+fixtureRoom)

	roster.rooms = nil
	room, rooms, err = openRoom(t.Context(), Deps{Roster: roster})
	assert.NilError(t, err)
	assert.Equal(t, room, "")
	assert.Equal(t, len(rooms), 0)

	_, _, err = openRoom(t.Context(), Deps{Room: fixtureRoom, Roster: roster})
	assert.ErrorContains(t, err, "the daemon knows no rooms yet")
}

// A screen opened on a daemon that knew no rooms draws, says so, and starts
// reading as soon as there is a room to read: the rooms pane fills on a poll
// and the operator picks one there.
func TestAScreenWithNoRoomWaitsForOne(t *testing.T) {
	log := &fakeLog{
		reply: func(logCall) ([]*chatv1.AdminHistoryEntry, error) {
			return fixtureEntries(1), nil
		},
	}
	m := newModel(t.Context(), Deps{Log: log, Roster: &fakeRoster{}}, "", nil)

	assert.Assert(t, strings.Contains(m.statusBar(defaultWidth), "room (none)"))
	assert.Assert(t, strings.Contains(m.View().Content, "rooms (0)"))
	// There is nothing to read, so nothing is asked of the log; the command the
	// loop hands back is its clock, which is not run here — it would wait out
	// the whole interval.
	assert.Assert(t, m.tail() != nil)
	assert.Equal(t, len(log.calls), 0)

	m.applyRoster(rosterMsg{rooms: twoRooms()})
	m.setFocus(focusRooms)
	m = update(t, m, press('j', "j"))
	m = update(t, m, press(tea.KeyEnter, ""))
	assert.Equal(t, m.room, otherRoom)

	m, _ = step(t, m, m.tail())
	assert.Equal(t, log.calls[0], logCall{room: otherRoom, since: 0, limit: backfill})
	assert.Assert(t, strings.Contains(m.statusBar(defaultWidth), "room "+otherRoom))
}
