package chat

import (
	"fmt"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestStore_HistoryRecordsWhatWasSaid(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	join(t, s, "tok-b", "/work/repo", "beta", "bob")
	join(t, s, "tok-c", "/work/repo", "beta", "cid")

	_, err := s.Send(t.Context(), "tok-a", "beta/bob", "just for you", sentAt)
	assert.NilError(t, err)
	_, err = s.Broadcast(t.Context(), "tok-a", "for everyone", sentAt.Add(time.Minute), true)
	assert.NilError(t, err)

	entries, err := s.History(t.Context(), "/work/repo", 0)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 2)

	// Oldest first, and a directed send names who it was addressed to.
	assert.Equal(t, entries[0].Text, "just for you")
	assert.DeepEqual(t, entries[0].From, Sender{Name: "alice", Team: "alpha", Room: "/work/repo"})
	assert.Assert(t, entries[0].To != nil)
	assert.DeepEqual(t, *entries[0].To, Sender{Name: "bob", Team: "beta", Room: "/work/repo"})
	assert.Assert(t, entries[0].SentAt.Equal(sentAt))

	// The broadcast reached two inboxes but is one utterance, addressed to
	// nobody in particular.
	assert.Equal(t, entries[1].Text, "for everyone")
	assert.Assert(t, entries[1].To == nil)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM room_log`), 2)
}

func TestStore_HistoryIsPerRoomAndNonDestructive(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	join(t, s, "tok-b", "/work/repo", "alpha", "bob")
	join(t, s, "tok-far", "/work/elsewhere", "alpha", "stranger")

	_, err := s.Send(t.Context(), "tok-a", "bob", "hi", sentAt)
	assert.NilError(t, err)

	// Draining the inbox leaves the transcript alone, and reading it twice
	// hands back the same thing.
	msgs, err := s.Read(t.Context(), "tok-b")
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 1)
	for range 2 {
		entries, err := s.History(t.Context(), "/work/repo", 0)
		assert.NilError(t, err)
		assert.Equal(t, len(entries), 1)
		assert.Equal(t, entries[0].Text, "hi")
	}

	// Another room heard nothing, and a room nobody spoke in is empty rather
	// than an error.
	entries, err := s.History(t.Context(), "/work/elsewhere", 0)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 0)
}

// History outlives the members who wrote it: that is why the log has no foreign
// key to them, unlike the inbox rows that leave with their owner.
func TestStore_HistoryOutlivesItsAuthor(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	join(t, s, "tok-b", "/work/repo", "alpha", "bob")
	_, err := s.Send(t.Context(), "tok-a", "bob", "handing over", sentAt)
	assert.NilError(t, err)

	_, err = s.RemoveMember(t.Context(), "tok-a")
	assert.NilError(t, err)
	_, err = s.RemoveMember(t.Context(), "tok-b")
	assert.NilError(t, err)

	// The inbox went with its owner; what was said stayed.
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM messages`), 0)
	entries, err := s.History(t.Context(), "/work/repo", 0)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 1)
	assert.Equal(t, entries[0].Text, "handing over")
	assert.Equal(t, entries[0].From.Name, "alice")
}

// What the host operator says into a room reaches the same inboxes a member's
// message does, so the transcript records it too — otherwise it would claim
// members received something nobody said.
func TestStore_HistoryRecordsWhatTheHostSaid(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	admin := adminSender("/work/repo")

	_, err := s.sendAs(t.Context(), admin, "alice", "restarting the box", sentAt)
	assert.NilError(t, err)
	_, err = s.broadcastAs(t.Context(), admin, "all hands", sentAt.Add(time.Minute))
	assert.NilError(t, err)

	entries, err := s.History(t.Context(), "/work/repo", 0)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 2)
	assert.DeepEqual(t, entries[0].From, admin)
	assert.Equal(t, entries[0].Text, "restarting the box")
	assert.Equal(t, entries[0].To.Name, "alice")
	assert.Equal(t, entries[1].Text, "all hands")
	assert.Assert(t, entries[1].To == nil)

	// A broadcast into a room with nobody in it fails, and the failure takes
	// the record with it: the transaction that would have delivered it rolled
	// back.
	_, err = s.broadcastAs(t.Context(), adminSender("/work/empty"), "anyone?", sentAt)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM room_log`), 2)
}

func TestStore_HistoryWindowTakesTheNewest(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")

	for i := range 5 {
		_, err := s.Broadcast(t.Context(), "tok-a",
			fmt.Sprintf("line %d", i), sentAt.Add(time.Duration(i)*time.Minute), true)
		assert.NilError(t, err)
	}

	entries, err := s.History(t.Context(), "/work/repo", 2)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 2)
	assert.Equal(t, entries[0].Text, "line 3")
	assert.Equal(t, entries[1].Text, "line 4")
}

func TestStore_HistoryPrunesToTheCap(t *testing.T) {
	s, _ := newTestStoreWithHistory(t, 3)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	join(t, s, "tok-far", "/work/elsewhere", "alpha", "stranger")

	for i := range 5 {
		_, err := s.Broadcast(t.Context(), "tok-a",
			fmt.Sprintf("line %d", i), sentAt.Add(time.Duration(i)*time.Minute), true)
		assert.NilError(t, err)
	}
	_, err := s.Broadcast(t.Context(), "tok-far", "elsewhere", sentAt, true)
	assert.NilError(t, err)

	// The cap is per room: the oldest two lines are gone, the other room's one
	// line is not.
	entries, err := s.History(t.Context(), "/work/repo", 0)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 3)
	assert.Equal(t, entries[0].Text, "line 2")
	assert.Equal(t, entries[2].Text, "line 4")
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM room_log`), 4)
}

func TestStore_HistoryDisabledRecordsNothing(t *testing.T) {
	s, _ := newTestStoreWithHistory(t, -1)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	join(t, s, "tok-b", "/work/repo", "alpha", "bob")

	_, err := s.Send(t.Context(), "tok-a", "bob", "hi", sentAt)
	assert.NilError(t, err)
	_, err = s.Broadcast(t.Context(), "tok-a", "hello", sentAt, true)
	assert.NilError(t, err)

	// Delivery is unaffected; only the transcript is.
	msgs, err := s.Read(t.Context(), "tok-b")
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 2)
	entries, err := s.History(t.Context(), "/work/repo", 0)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 0)
}

// The cap defaults rather than pruning everything away: an unconfigured store
// is one nobody has an opinion about, not one that must forget.
func TestStore_HistoryCapDefaults(t *testing.T) {
	s, _ := newTestStoreWithHistory(t, 0)
	assert.Equal(t, s.historyLimit, defaultHistoryLimit)
}
