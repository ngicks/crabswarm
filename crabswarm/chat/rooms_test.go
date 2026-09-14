package chat

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestStore_ListRoomsUnionsTheLogAndAttendance(t *testing.T) {
	s, path := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	attend(t, s, "tok-b", testRoom, "beta", "bob")
	attend(t, s, "tok-far", "/work/elsewhere", "alpha", "stranger")
	send(t, s, senderOf(alice), Target{Kind: TargetEveryone}, "hello")

	rooms, err := s.ListRooms(t.Context())
	assert.NilError(t, err)
	assert.Equal(t, len(rooms), 2)
	assert.Equal(t, rooms[0].Name, "/work/elsewhere")
	assert.Equal(t, rooms[1].Name, testRoom)
	assert.Equal(t, len(rooms[1].Members), 2)
	assert.Equal(t, rooms[1].Members[0].Name, "alice")
	assert.Equal(t, rooms[1].Members[1].Name, "bob")

	// A restart empties the rooms without removing them: the log remembers
	// where the conversation happened, attendance does not survive the daemon.
	reopened := reopen(t, s, path)
	rooms, err = reopened.ListRooms(t.Context())
	assert.NilError(t, err)
	assert.Equal(t, len(rooms), 2)
	for _, room := range rooms {
		assert.Equal(t, len(room.Members), 0)
	}
	msgs, err := reopened.ReadRoom(t.Context(), testRoom, ReadFilter{})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(msgs), []string{"hello"})
}

func TestStore_DeleteRoomRefusedWhileAttended(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	attend(t, s, "tok-b", testRoom, "beta", "bob")
	send(t, s, senderOf(alice), toRoles(role("beta", "bob")), "one")
	send(t, s, senderOf(alice), Target{Kind: TargetEveryone}, "two")

	_, err := s.DeleteRoom(t.Context(), testRoom)
	assert.ErrorIs(t, err, ErrRoomAttended)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM messages`), 2)

	for _, token := range []string{"tok-a", "tok-b"} {
		_, err := s.Detach(t.Context(), token)
		assert.NilError(t, err)
	}

	deleted, err := s.DeleteRoom(t.Context(), testRoom)
	assert.NilError(t, err)
	assert.Equal(t, deleted, int64(2))

	// The room takes its messages, their mentions and its read positions with
	// it, and nothing else.
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM rooms`), 0)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM messages`), 0)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM message_mention`), 0)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM read_positions`), 0)

	rooms, err := s.ListRooms(t.Context())
	assert.NilError(t, err)
	assert.Equal(t, len(rooms), 0)
}

func TestStore_DeleteRoomUnknown(t *testing.T) {
	s, _ := newTestStore(t)
	attend(t, s, "tok-a", testRoom, "alpha", "alice")

	_, err := s.DeleteRoom(t.Context(), "/work/nowhere")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestStore_DeleteRoomLeavesOtherRoomsAlone(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	far := attend(t, s, "tok-far", "/work/elsewhere", "alpha", "stranger")
	send(t, s, senderOf(alice), Target{Kind: TargetEveryone}, "here")
	send(t, s, senderOf(far), Target{Kind: TargetEveryone}, "there")
	_, err := s.Detach(t.Context(), "tok-a")
	assert.NilError(t, err)

	deleted, err := s.DeleteRoom(t.Context(), testRoom)
	assert.NilError(t, err)
	assert.Equal(t, deleted, int64(1))

	msgs, err := s.ReadRoom(t.Context(), "/work/elsewhere", ReadFilter{})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(msgs), []string{"there"})
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM read_positions`), 1)
}
