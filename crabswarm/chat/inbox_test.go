package chat

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestStore_SendAndReadDrains(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	join(t, s, "tok-b", "/work/repo", "beta", "bob")

	to, err := s.Send(t.Context(), "tok-a", "beta/bob", "first", sentAt)
	assert.NilError(t, err)
	assert.Equal(t, to.Token, "tok-b")
	_, err = s.Send(t.Context(), "tok-a", "bob", "second", sentAt.Add(time.Minute))
	assert.NilError(t, err)

	// Both messages come back oldest first, carrying the sender snapshot.
	msgs, err := s.Read(t.Context(), "tok-b")
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 2)
	assert.Equal(t, msgs[0].Text, "first")
	assert.Equal(t, msgs[1].Text, "second")
	assert.DeepEqual(t, msgs[0].From, Sender{Name: "alice", Team: "alpha", Room: "/work/repo"})
	assert.Assert(t, msgs[1].SentAt.Equal(sentAt.Add(time.Minute)))

	// Reading drained the inbox: nothing is redelivered.
	msgs, err = s.Read(t.Context(), "tok-b")
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 0)

	// The sender's own inbox was never touched.
	msgs, err = s.Read(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 0)
}

func TestStore_SendStoresTimestampAsUTC(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	join(t, s, "tok-b", "/work/repo", "alpha", "bob")

	zone := time.FixedZone("JST", 9*60*60)
	local := time.Date(2026, 8, 27, 19, 30, 0, 0, zone)
	_, err := s.Send(t.Context(), "tok-a", "bob", "hi", local)
	assert.NilError(t, err)

	msgs, err := s.Read(t.Context(), "tok-b")
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 1)
	assert.Assert(t, msgs[0].SentAt.Equal(local))
	assert.Equal(t, msgs[0].SentAt.Location(), time.UTC)
}

func TestStore_SendRejectsUnknownParties(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	join(t, s, "tok-far", "/work/elsewhere", "alpha", "stranger")

	// An unknown sender.
	_, err := s.Send(t.Context(), "tok-unknown", "alice", "hi", sentAt)
	assert.ErrorIs(t, err, ErrNotFound)

	// An unknown recipient.
	_, err = s.Send(t.Context(), "tok-a", "nobody", "hi", sentAt)
	assert.ErrorIs(t, err, ErrNotFound)

	// A member of another room stays invisible, and nothing is stored.
	_, err = s.Send(t.Context(), "tok-a", "alpha/stranger", "hi", sentAt)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM messages`), 0)
}

func TestStore_SendAmbiguousDeliversNothing(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-caller", "/work/repo", "alpha", "caller")
	join(t, s, "tok-b", "/work/repo", "beta", "worker")
	join(t, s, "tok-c", "/work/repo", "gamma", "worker")

	_, err := s.Send(t.Context(), "tok-caller", "worker", "which one?", sentAt)
	assert.ErrorIs(t, err, ErrAmbiguousName)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM messages`), 0)
}

func TestStore_BroadcastReachesWholeRoom(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	join(t, s, "tok-b", "/work/repo", "alpha", "bob")
	join(t, s, "tok-z", "/work/repo", "beta", "zoe")
	join(t, s, "tok-far", "/work/elsewhere", "alpha", "stranger")

	// Including the sender: every member of the room, ordered team then name.
	recipients, err := s.Broadcast(t.Context(), "tok-a", "standup", sentAt, false)
	assert.NilError(t, err)
	got := make([]string, len(recipients))
	for i, m := range recipients {
		got[i] = m.Team + "/" + m.Name
	}
	assert.DeepEqual(t, got, []string{"alpha/alice", "alpha/bob", "beta/zoe"})

	for _, token := range []string{"tok-a", "tok-b", "tok-z"} {
		msgs, err := s.Read(t.Context(), token)
		assert.NilError(t, err)
		assert.Equal(t, len(msgs), 1, "token %s", token)
		assert.Equal(t, msgs[0].Text, "standup")
		assert.DeepEqual(t, msgs[0].From,
			Sender{Name: "alice", Team: "alpha", Room: "/work/repo"})
	}

	// Another room hears nothing.
	msgs, err := s.Read(t.Context(), "tok-far")
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 0)
}

func TestStore_BroadcastExcludingSender(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	join(t, s, "tok-b", "/work/repo", "alpha", "bob")

	recipients, err := s.Broadcast(t.Context(), "tok-a", "heads up", sentAt, true)
	assert.NilError(t, err)
	assert.Equal(t, len(recipients), 1)
	assert.Equal(t, recipients[0].Token, "tok-b")

	msgs, err := s.Read(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 0)
	msgs, err = s.Read(t.Context(), "tok-b")
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 1)
}

func TestStore_BroadcastFromUnknownSender(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")

	_, err := s.Broadcast(t.Context(), "tok-unknown", "hello?", sentAt, false)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM messages`), 0)
}

func TestStore_ReadUnknownToken(t *testing.T) {
	s, _ := newTestStore(t)

	_, err := s.Read(t.Context(), "tok-unknown")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestStore_MessagesSurviveSenderLeaving(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	join(t, s, "tok-b", "/work/repo", "alpha", "bob")
	_, err := s.Send(t.Context(), "tok-a", "bob", "bye", sentAt)
	assert.NilError(t, err)

	// The sender is a snapshot, not a reference: a message outlives its
	// sender's departure.
	_, err = s.RemoveMember(t.Context(), "tok-a")
	assert.NilError(t, err)

	msgs, err := s.Read(t.Context(), "tok-b")
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 1)
	assert.DeepEqual(t, msgs[0].From, Sender{Name: "alice", Team: "alpha", Room: "/work/repo"})
}
