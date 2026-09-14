package chat

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestStore_SendRecordsOneRowPerMessage(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	attend(t, s, "tok-b", testRoom, "beta", "bob")

	sent := send(t, s, senderOf(alice), toRoles(role("beta", "bob")), "just for you")
	assert.Equal(t, sent.Message.Seq, int64(1))
	assert.Assert(t, sent.Message.Id != "")
	assert.DeepEqual(t, sent.Message.From, senderOf(alice))
	assert.Equal(t, sent.Message.Target.Kind, TargetRoles)
	assert.DeepEqual(t, sent.Message.Target.Roles,
		[]Sender{{Team: "beta", Name: "bob", Room: testRoom}})
	assert.Equal(t, len(sent.Mentioned), 1)
	assert.Equal(t, sent.Mentioned[0].Token, "tok-b")
	assert.Equal(t, len(sent.Absent), 0)

	// One row for the message and one per named role, whoever heard it.
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM messages`), 1)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM message_mention`), 1)

	// Ids are unique across the whole database, seqs dense within the room.
	second := send(t, s, senderOf(alice), Target{}, "a board post")
	assert.Equal(t, second.Message.Seq, int64(2))
	assert.Assert(t, second.Message.Id != sent.Message.Id)
	assert.Equal(t, second.Message.Target.Kind, TargetNone)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM message_mention`), 1)
}

func TestStore_SendNumbersRoomsApart(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	far := attend(t, s, "tok-far", "/work/elsewhere", "alpha", "stranger")

	first := send(t, s, senderOf(alice), Target{Kind: TargetEveryone}, "here")
	elsewhere := send(t, s, senderOf(far), Target{Kind: TargetEveryone}, "there")

	// Two rooms share no numbering: both start at one.
	assert.Equal(t, first.Message.Seq, int64(1))
	assert.Equal(t, elsewhere.Message.Seq, int64(1))
	assert.Equal(t, first.Message.Room, testRoom)
	assert.Equal(t, elsewhere.Message.Room, "/work/elsewhere")
}

func TestStore_SendStoresTimestampAsUTC(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	bob := attend(t, s, "tok-b", testRoom, "alpha", "bob")

	zone := time.FixedZone("JST", 9*60*60)
	local := time.Date(2026, 8, 27, 19, 30, 0, 0, zone)
	_, err := s.Send(t.Context(), senderOf(alice), toRoles(role("", "bob")), "hi", local)
	assert.NilError(t, err)

	msgs, _, err := s.Read(t.Context(), senderOf(bob), ReadFilter{})
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 1)
	assert.Assert(t, msgs[0].SentAt.Equal(local))
	assert.Equal(t, msgs[0].SentAt.Location(), time.UTC)
}

func TestStore_SendEveryoneNamesNobody(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	bob := attend(t, s, "tok-b", testRoom, "beta", "bob")

	sent := send(t, s, senderOf(alice), Target{Kind: TargetEveryone}, "standup")
	assert.Equal(t, len(sent.Mentioned), 0)
	assert.Equal(t, len(sent.Absent), 0)
	assert.Equal(t, len(sent.Message.Target.Roles), 0)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM message_mention`), 0)

	// It is unread for every role of the room but the sender.
	msgs, _, err := s.Read(t.Context(), senderOf(alice), ReadFilter{})
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 0)

	msgs, remaining, err := s.Read(t.Context(), senderOf(bob), ReadFilter{})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(msgs), []string{"standup"})
	assert.Assert(t, msgs[0].MentionedYou)
	assert.Equal(t, remaining, 0)
}

func TestStore_SendReportsAbsentRoles(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	bob := attend(t, s, "tok-b", testRoom, "beta", "bob")
	_, err := s.Detach(t.Context(), bob.Token)
	assert.NilError(t, err)

	sent := send(t, s, senderOf(alice), toRoles(role("beta", "bob")), "when you are back")
	assert.Equal(t, len(sent.Mentioned), 1)
	assert.DeepEqual(t, sent.Mentioned[0],
		Member{Room: testRoom, Team: "beta", Name: "bob"})
	assert.DeepEqual(t, sent.Absent, sent.Mentioned)

	// The mention was still written: it waits at the role's read position.
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM message_mention`), 1)
}

func TestStore_SendResolvesBareNames(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	attend(t, s, "tok-a2", testRoom, "alpha", "shared")
	attend(t, s, "tok-b", testRoom, "beta", "shared")
	attend(t, s, "tok-c", testRoom, "beta", "unique")

	// The sender's own team wins a name two teams carry.
	sent := send(t, s, senderOf(alice), toRoles(role("", "shared")), "mine")
	assert.DeepEqual(t, sent.Message.Target.Roles,
		[]Sender{{Team: "alpha", Name: "shared", Room: testRoom}})

	// A name only one team carries resolves across the room.
	sent = send(t, s, senderOf(alice), toRoles(role("", "unique")), "yours")
	assert.DeepEqual(t, sent.Message.Target.Roles,
		[]Sender{{Team: "beta", Name: "unique", Room: testRoom}})

	// A sender in neither of the two teams carrying the name has nothing to
	// break the tie with.
	gamma := attend(t, s, "tok-g", testRoom, "gamma", "gwen")
	_, err := s.Send(t.Context(), senderOf(gamma), toRoles(role("", "shared")), "whose?", sentAt)
	assert.ErrorIs(t, err, ErrAmbiguousRole)

	// The host operator has no team, so only the second half applies to it.
	admin := Sender{Name: "admin", Room: testRoom}
	_, err = s.Send(t.Context(), admin, toRoles(role("", "shared")), "whose?", sentAt)
	assert.ErrorIs(t, err, ErrAmbiguousRole)
	sent = send(t, s, admin, toRoles(role("", "unique")), "yours too")
	assert.DeepEqual(t, sent.Message.Target.Roles,
		[]Sender{{Team: "beta", Name: "unique", Room: testRoom}})
}

func TestStore_SendRefusesUnknownRoles(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	attend(t, s, "tok-far", "/work/elsewhere", "beta", "stranger")

	// A role the room has never had, qualified or bare.
	_, err := s.Send(t.Context(), senderOf(alice),
		toRoles(role("beta", "nobody")), "hi", sentAt)
	assert.ErrorIs(t, err, ErrUnknownRole)
	_, err = s.Send(t.Context(), senderOf(alice), toRoles(role("", "nobody")), "hi", sentAt)
	assert.ErrorIs(t, err, ErrUnknownRole)

	// A role of another room stays invisible.
	_, err = s.Send(t.Context(), senderOf(alice),
		toRoles(role("beta", "stranger")), "hi", sentAt)
	assert.ErrorIs(t, err, ErrUnknownRole)

	// A roles target that names nobody is a request that cannot be carried out.
	_, err = s.Send(t.Context(), senderOf(alice), Target{Kind: TargetRoles}, "hi", sentAt)
	assert.ErrorIs(t, err, ErrInvalidArgument)

	// A refused send left the room as it was, seq counter included.
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM messages`), 0)
	assert.Equal(t, countRows(t, s,
		`SELECT last_seq FROM rooms WHERE name = ?`, testRoom), 0)
}

func TestStore_SendCollapsesDuplicateRoles(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	attend(t, s, "tok-b", testRoom, "beta", "bob")

	sent := send(t, s, senderOf(alice),
		toRoles(role("beta", "bob"), role("", "bob"), role("beta", "bob")), "once")
	assert.Equal(t, len(sent.Message.Target.Roles), 1)
	assert.Equal(t, len(sent.Mentioned), 1)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM message_mention`), 1)
}

func TestStore_SendPrunesPastTheCap(t *testing.T) {
	s, _ := newTestStoreWithHistory(t, 2)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	attend(t, s, "tok-b", testRoom, "beta", "bob")

	for _, text := range []string{"one", "two", "three", "four"} {
		send(t, s, senderOf(alice), toRoles(role("beta", "bob")), text)
	}

	// Only the cap's worth is left, and the mentions of what was pruned went
	// with it.
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM messages`), 2)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM message_mention`), 2)
	assert.Equal(t, countRows(t, s, `SELECT MIN(seq) FROM messages`), 3)
	assert.Equal(t, countRows(t, s, `SELECT MAX(seq) FROM messages`), 4)

	// The counter is not reset by pruning, so the next seq carries on.
	sent := send(t, s, senderOf(alice), toRoles(role("beta", "bob")), "five")
	assert.Equal(t, sent.Message.Seq, int64(5))
}

func TestStore_NegativeHistoryLimitPrunesNothing(t *testing.T) {
	s, _ := newTestStoreWithHistory(t, -1)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	attend(t, s, "tok-b", testRoom, "beta", "bob")

	for _, text := range []string{"one", "two", "three"} {
		send(t, s, senderOf(alice), toRoles(role("beta", "bob")), text)
	}
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM messages`), 3)
}
