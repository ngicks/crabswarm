package chat

import (
	"testing"

	"gotest.tools/v3/assert"
)

// readPosition is a white-box read of where a role has got to.
func readPosition(t *testing.T, s *Store, room, team, name string) int {
	t.Helper()
	return countRows(t, s,
		`SELECT last_seq FROM read_positions WHERE room = ? AND team = ? AND name = ?`,
		room, team, name)
}

func TestStore_ReadUnreadMovesThePosition(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	bob := attend(t, s, "tok-b", testRoom, "beta", "bob")

	for _, text := range []string{"one", "two", "three"} {
		send(t, s, senderOf(alice), toRoles(role("beta", "bob")), text)
	}

	msgs, remaining, err := s.Read(t.Context(), senderOf(bob), ReadFilter{Range: 2})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(msgs), []string{"one", "two"})
	assert.Assert(t, msgs[0].MentionedYou)
	assert.Equal(t, remaining, 1)
	assert.Equal(t, readPosition(t, s, testRoom, "beta", "bob"), 2)

	msgs, remaining, err = s.Read(t.Context(), senderOf(bob), ReadFilter{})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(msgs), []string{"three"})
	assert.Equal(t, remaining, 0)

	// An empty read moves nothing and reports nothing left.
	msgs, remaining, err = s.Read(t.Context(), senderOf(bob), ReadFilter{})
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 0)
	assert.Equal(t, remaining, 0)
	assert.Equal(t, readPosition(t, s, testRoom, "beta", "bob"), 3)
}

// Counting is asking without taking: it answers what a read would leave behind
// and moves the position nowhere, so the same question asked twice answers
// twice the same and the messages are still there to be read.
func TestStore_CountUnreadMovesNothing(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	bob := attend(t, s, "tok-b", testRoom, "beta", "bob")

	unread, err := s.CountUnread(t.Context(), senderOf(bob))
	assert.NilError(t, err)
	assert.Equal(t, unread, 0)

	send(t, s, senderOf(alice), toRoles(role("beta", "bob")), "for you")
	send(t, s, senderOf(alice), Target{Kind: TargetEveryone}, "for the room")
	// Neither of these is bob's unread: a post names nobody, and a role does not
	// count itself.
	send(t, s, senderOf(alice), Target{}, "a board post")
	send(t, s, senderOf(bob), Target{Kind: TargetEveryone}, "my own announcement")

	for range 2 {
		unread, err = s.CountUnread(t.Context(), senderOf(bob))
		assert.NilError(t, err)
		assert.Equal(t, unread, 2)
	}
	assert.Equal(t, readPosition(t, s, testRoom, "beta", "bob"), 0)

	// The read that follows still has both to hand over, and counting after it
	// reports what it left.
	msgs, _, err := s.Read(t.Context(), senderOf(bob), ReadFilter{Range: 1})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(msgs), []string{"for you"})
	unread, err = s.CountUnread(t.Context(), senderOf(bob))
	assert.NilError(t, err)
	assert.Equal(t, unread, 1)

	// A role the room has never carried has no position to count from.
	_, err = s.CountUnread(t.Context(), Sender{Room: testRoom, Team: "beta", Name: "nobody"})
	assert.ErrorIs(t, err, ErrUnknownRole)
}

func TestStore_ReadUnreadSkipsPostsAndOwnMessages(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	bob := attend(t, s, "tok-b", testRoom, "beta", "bob")

	send(t, s, senderOf(alice), Target{}, "a board post")
	send(t, s, senderOf(bob), Target{Kind: TargetEveryone}, "my own announcement")
	send(t, s, senderOf(alice), toRoles(role("beta", "bob")), "for you")

	// A post names nobody, so it is nobody's unread, and a role does not hear
	// itself.
	msgs, remaining, err := s.Read(t.Context(), senderOf(bob), ReadFilter{})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(msgs), []string{"for you"})
	assert.Equal(t, remaining, 0)

	// Reading the room itself shows all three: a post is in the room, it is
	// just addressed to nobody.
	all, err := s.ReadRoom(t.Context(), testRoom, ReadFilter{})
	assert.NilError(t, err)
	assert.Equal(t, len(all), 3)
	for _, m := range all {
		assert.Assert(t, !m.MentionedYou)
	}
}

func TestStore_ReadTailThenUnread(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	bob := attend(t, s, "tok-b", testRoom, "beta", "bob")

	for _, text := range []string{"one", "two", "three"} {
		send(t, s, senderOf(alice), toRoles(role("beta", "bob")), text)
	}

	// The tail is read backward and handed back oldest first.
	msgs, remaining, err := s.Read(t.Context(), senderOf(bob),
		ReadFilter{Cursor: CursorTail, Range: -2})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(msgs), []string{"two", "three"})
	assert.Equal(t, remaining, 0)

	// The position is a cursor, not a receipt: reading the tail read past
	// everything before it, so the first mention is not handed out again.
	assert.Equal(t, readPosition(t, s, testRoom, "beta", "bob"), 3)
	msgs, remaining, err = s.Read(t.Context(), senderOf(bob), ReadFilter{})
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 0)
	assert.Equal(t, remaining, 0)
}

func TestStore_ReadHeadMovesNothingWhenItShowsNothingNewer(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	bob := attend(t, s, "tok-b", testRoom, "beta", "bob")

	for _, text := range []string{"one", "two", "three"} {
		send(t, s, senderOf(alice), toRoles(role("beta", "bob")), text)
	}
	_, _, err := s.Read(t.Context(), senderOf(bob), ReadFilter{Cursor: CursorTail})
	assert.NilError(t, err)
	assert.Equal(t, readPosition(t, s, testRoom, "beta", "bob"), 3)

	msgs, remaining, err := s.Read(t.Context(), senderOf(bob),
		ReadFilter{Cursor: CursorHead, Range: 1})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(msgs), []string{"one"})
	// Already seen, so it is not a mention of the reader and the position is
	// left where it was.
	assert.Assert(t, !msgs[0].MentionedYou)
	assert.Equal(t, remaining, 0)
	assert.Equal(t, readPosition(t, s, testRoom, "beta", "bob"), 3)
}

func TestStore_ReadFilterByTarget(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	attend(t, s, "tok-b", testRoom, "beta", "bob")
	attend(t, s, "tok-c", testRoom, "beta", "carl")

	send(t, s, senderOf(alice), toRoles(role("beta", "bob")), "for bob")
	send(t, s, senderOf(alice), toRoles(role("beta", "carl")), "for carl")
	send(t, s, senderOf(alice), Target{Kind: TargetEveryone}, "for all")
	send(t, s, senderOf(alice), Target{}, "for nobody")

	for _, tc := range []struct {
		name string
		to   Target
		want []string
	}{
		{"one role", toRoles(role("beta", "bob")), []string{"for bob"}},
		{
			"any of several roles",
			toRoles(role("beta", "bob"), role("beta", "carl")),
			[]string{"for bob", "for carl"},
		},
		{"everyone", Target{Kind: TargetEveryone}, []string{"for all"}},
		{"posts", Target{Kind: TargetNone}, []string{"for nobody"}},
		{"a role the room never had", toRoles(role("beta", "nobody")), []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			to := tc.to
			msgs, err := s.ReadRoom(t.Context(), testRoom,
				ReadFilter{Cursor: CursorHead, To: &to})
			assert.NilError(t, err)
			assert.DeepEqual(t, texts(msgs), tc.want)
		})
	}
}

// A role the room never had is an answer; a role the caller did not write
// unambiguously is not. A bare name two teams carry is turned down the way the
// same name is turned down as a send target, so a reader never gets one team's
// messages in place of the other's.
func TestStore_ReadFilterRefusesAnUnanswerableTarget(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	attend(t, s, "tok-b", testRoom, "beta", "bob")
	attend(t, s, "tok-g", testRoom, "gamma", "bob")

	// The host reads with no team of its own, so a bare name is only ever
	// resolved across the room.
	ambiguous := toRoles(role("", "bob"))
	_, err := s.ReadRoom(t.Context(), testRoom, ReadFilter{Cursor: CursorHead, To: &ambiguous})
	assert.ErrorIs(t, err, ErrAmbiguousRole)

	// A member whose own team carries neither is in the same position.
	_, _, err = s.Read(t.Context(), senderOf(alice), ReadFilter{Cursor: CursorHead, To: &ambiguous})
	assert.ErrorIs(t, err, ErrAmbiguousRole)

	// A target naming no role at all says nothing about which messages to keep.
	nameless := toRoles(role("beta", ""))
	_, err = s.ReadRoom(t.Context(), testRoom, ReadFilter{Cursor: CursorHead, To: &nameless})
	assert.ErrorIs(t, err, ErrInvalidArgument)
}

func TestStore_ReadBoundsBySeq(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")

	for _, text := range []string{"one", "two", "three", "four"} {
		send(t, s, senderOf(alice), Target{Kind: TargetEveryone}, text)
	}

	// Both bounds are exclusive.
	msgs, err := s.ReadRoom(t.Context(), testRoom,
		ReadFilter{Cursor: CursorHead, Since: 1, Until: 4})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(msgs), []string{"two", "three"})

	// The tail counts back from the upper bound.
	msgs, err = s.ReadRoom(t.Context(), testRoom, ReadFilter{Range: -2, Until: 4})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(msgs), []string{"two", "three"})
}

func TestStore_ReadRefusesARangeAgainstItsCursor(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")

	for _, f := range []ReadFilter{
		{Cursor: CursorUnread, Range: -1},
		{Cursor: CursorHead, Range: -1},
		{Cursor: CursorTail, Range: 1},
		{Cursor: ReadCursor("sideways")},
	} {
		_, _, err := s.Read(t.Context(), senderOf(alice), f)
		assert.ErrorIs(t, err, ErrInvalidArgument)
	}
}

func TestStore_ReadNeedsARoleThatHasAttended(t *testing.T) {
	s, _ := newTestStore(t)
	attend(t, s, "tok-a", testRoom, "alpha", "alice")

	// A role with no position has never been in the room, so there is no
	// "unread" to measure for it.
	_, _, err := s.Read(t.Context(),
		Sender{Room: testRoom, Team: "beta", Name: "ghost"}, ReadFilter{})
	assert.ErrorIs(t, err, ErrUnknownRole)
}

func TestStore_ReadRoomReadsOnNobodysBehalf(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	bob := attend(t, s, "tok-b", testRoom, "beta", "bob")
	for _, text := range []string{"one", "two", "three"} {
		send(t, s, senderOf(alice), toRoles(role("beta", "bob")), text)
	}

	// An unspecified cursor is the tail, and the default range is the last ten.
	msgs, err := s.ReadRoom(t.Context(), testRoom, ReadFilter{})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(msgs), []string{"one", "two", "three"})

	// It moves nobody's position and marks nothing as a mention.
	assert.Equal(t, readPosition(t, s, testRoom, "beta", "bob"), 0)
	for _, m := range msgs {
		assert.Assert(t, !m.MentionedYou)
	}
	msgs, remaining, err := s.Read(t.Context(), senderOf(bob), ReadFilter{})
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 3)
	assert.Equal(t, remaining, 0)

	// Unread is measured from a position, which a room read has none of.
	_, err = s.ReadRoom(t.Context(), testRoom, ReadFilter{Cursor: CursorUnread})
	assert.ErrorIs(t, err, ErrInvalidArgument)

	// A room nothing was ever said in reads empty rather than failing.
	msgs, err = s.ReadRoom(t.Context(), "/work/nowhere", ReadFilter{})
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 0)
}

func TestStore_ReadCarriesTheWrittenTarget(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	attend(t, s, "tok-b", testRoom, "beta", "bob")
	attend(t, s, "tok-c", testRoom, "beta", "carl")

	send(t, s, senderOf(alice), toRoles(role("beta", "bob"), role("beta", "carl")), "you two")

	msgs, err := s.ReadRoom(t.Context(), testRoom, ReadFilter{})
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 1)
	assert.Equal(t, msgs[0].Target.Kind, TargetRoles)
	assert.DeepEqual(t, msgs[0].Target.Roles, []Sender{
		{Team: "beta", Name: "bob", Room: testRoom},
		{Team: "beta", Name: "carl", Room: testRoom},
	})
	assert.DeepEqual(t, msgs[0].From, senderOf(alice))
}
