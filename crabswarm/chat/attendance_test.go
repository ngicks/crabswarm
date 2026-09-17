package chat

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestStore_AttendRecordsAndRefusesASecondStream(t *testing.T) {
	s, _ := newTestStore(t)

	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	assert.Equal(t, alice.State, StateDone)
	assert.Assert(t, !alice.StateReportedAt.IsZero())

	got, err := s.Member(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.DeepEqual(t, got, alice)

	// One token is one session, so a second declaration for it is refused
	// rather than taken for a second participant.
	_, err = s.Attend(t.Context(), Member{
		Token: "tok-a", Name: "alice", Team: "alpha", Room: testRoom, Kind: KindAgent,
	})
	assert.ErrorIs(t, err, ErrAlreadyAttending)

	// So is a second session under the same role: the role has one read
	// position, and two sessions sharing it would each mark what the other has
	// not seen as shown.
	_, err = s.Attend(t.Context(), Member{
		Token: "tok-a2", Name: "alice", Team: "alpha", Room: testRoom, Kind: KindAgent,
	})
	assert.ErrorIs(t, err, ErrAlreadyAttending)

	// The same name in another team, and the same role in another room, are
	// other roles: neither shares a read position with alice.
	_, err = s.Attend(t.Context(), Member{
		Token: "tok-b", Name: "alice", Team: "beta", Room: testRoom, Kind: KindAgent,
	})
	assert.NilError(t, err)
	_, err = s.Attend(t.Context(), Member{
		Token: "tok-c", Name: "alice", Team: "alpha", Room: "/work/elsewhere", Kind: KindAgent,
	})
	assert.NilError(t, err)
}

// The harness and the delivery are kept as declared, and an agent that declared
// no delivery is typed at through its terminal — which is how every agent was
// reached before a harness could deliver a mention itself, so a client still
// speaking the older schema keeps working. A human declares neither and is
// given neither: nothing is ever pushed at one.
func TestStore_AttendKeepsTheHarnessAndDefaultsTheDelivery(t *testing.T) {
	s, _ := newTestStore(t)

	native, err := s.Attend(t.Context(), Member{
		Token: "tok-n", Name: "nina", Team: "alpha", Room: testRoom,
		Kind: KindAgent, Harness: HarnessCodex, Nudge: NudgeNative,
	})
	assert.NilError(t, err)
	assert.Equal(t, native.Harness, HarnessCodex)
	assert.Equal(t, native.Nudge, NudgeNative)

	silent, err := s.Attend(t.Context(), Member{
		Token: "tok-s", Name: "sam", Team: "alpha", Room: testRoom, Kind: KindAgent,
	})
	assert.NilError(t, err)
	assert.Equal(t, silent.Harness, Harness(""))
	assert.Equal(t, silent.Nudge, NudgeTerminal)

	person, err := s.Attend(t.Context(), Member{
		Token: "tok-h", Name: "hana", Team: "alpha", Room: testRoom, Kind: KindHuman,
	})
	assert.NilError(t, err)
	assert.Equal(t, person.Harness, Harness(""))
	assert.Equal(t, person.Nudge, NudgeDelivery(""))

	// And what was recorded is what comes back, since the attendance is read
	// from the same map the roster is drawn from.
	got, err := s.Member(t.Context(), "tok-n")
	assert.NilError(t, err)
	assert.DeepEqual(t, got, native)
}

func TestStore_AttendValidates(t *testing.T) {
	s, _ := newTestStore(t)

	for _, tc := range []struct {
		name string
		m    Member
	}{
		{"no token", Member{Name: "alice", Team: "alpha", Room: testRoom, Kind: KindAgent}},
		{"no room", Member{Token: "t", Name: "alice", Team: "alpha", Kind: KindAgent}},
		{"no kind", Member{Token: "t", Name: "alice", Team: "alpha", Room: testRoom}},
		{"no name", Member{Token: "t", Team: "alpha", Room: testRoom, Kind: KindAgent}},
		{"no team", Member{Token: "t", Name: "alice", Room: testRoom, Kind: KindAgent}},
		{
			"slash in name",
			Member{Token: "t", Name: "a/b", Team: "alpha", Room: testRoom, Kind: KindAgent},
		},
		{
			"slash in team",
			Member{Token: "t", Name: "alice", Team: "a/b", Room: testRoom, Kind: KindAgent},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.Attend(t.Context(), tc.m)
			assert.Assert(t, err != nil)
		})
	}
	// Nothing half-recorded: a refused attendance created no room.
	rooms, err := s.ListRooms(t.Context())
	assert.NilError(t, err)
	assert.Equal(t, len(rooms), 0)
}

func TestStore_DetachLeavesTheRoleBehind(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")

	gone, err := s.Detach(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.DeepEqual(t, gone, alice)

	// The session is gone; the role, its read position and the room are not.
	_, err = s.Member(t.Context(), "tok-a")
	assert.ErrorIs(t, err, ErrNotAttending)
	_, err = s.Detach(t.Context(), "tok-a")
	assert.ErrorIs(t, err, ErrNotAttending)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM read_positions`), 1)

	rooms, err := s.ListRooms(t.Context())
	assert.NilError(t, err)
	assert.Equal(t, len(rooms), 1)
	assert.Equal(t, len(rooms[0].Members), 0)
}

func TestStore_SetState(t *testing.T) {
	s, _ := newTestStore(t)
	attend(t, s, "tok-a", testRoom, "alpha", "alice")

	assert.NilError(t, s.SetState(t.Context(), "tok-a", StateWorking, reportedAt))
	got, err := s.Member(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, got.State, StateWorking)
	assert.Assert(t, got.StateReportedAt.Equal(reportedAt))

	assert.Assert(t, s.SetState(t.Context(), "tok-a", MemberState("napping"), reportedAt) != nil)
	assert.ErrorIs(t,
		s.SetState(t.Context(), "tok-unknown", StateDone, reportedAt), ErrNotAttending)
}

func TestStore_ListMembersIsPerRoomAndOrdered(t *testing.T) {
	s, _ := newTestStore(t)
	attend(t, s, "tok-c", testRoom, "beta", "carl")
	attend(t, s, "tok-a", testRoom, "alpha", "zoe")
	attend(t, s, "tok-b", testRoom, "beta", "bob")
	attend(t, s, "tok-far", "/work/elsewhere", "alpha", "stranger")

	members, err := s.ListMembers(t.Context(), testRoom)
	assert.NilError(t, err)
	assert.Equal(t, len(members), 3)
	assert.Equal(t, members[0].Name, "zoe")  // alpha
	assert.Equal(t, members[1].Name, "bob")  // beta, by name
	assert.Equal(t, members[2].Name, "carl") // beta, by name

	// A room nobody is in has nobody in it, and it is not an error to ask.
	members, err = s.ListMembers(t.Context(), "/work/nowhere")
	assert.NilError(t, err)
	assert.Equal(t, len(members), 0)
}

// Attending crosses the rooms ListMembers stays inside, for a watcher that
// belongs to none of them.
func TestStore_AttendingSpansEveryRoom(t *testing.T) {
	s, _ := newTestStore(t)

	members, err := s.Attending(t.Context())
	assert.NilError(t, err)
	assert.Equal(t, len(members), 0)

	attend(t, s, "tok-c", testRoom, "beta", "carl")
	attend(t, s, "tok-a", testRoom, "alpha", "zoe")
	attend(t, s, "tok-far", "/work/elsewhere", "alpha", "stranger")

	members, err = s.Attending(t.Context())
	assert.NilError(t, err)
	assert.Equal(t, len(members), 3)
	assert.Equal(t, members[0].Name, "stranger") // alpha, by name
	assert.Equal(t, members[1].Name, "zoe")      // alpha, by name
	assert.Equal(t, members[2].Name, "carl")     // beta

	// A member that stopped attending is gone from it, the way it is gone from
	// the room: attendance is the stream, not a record that outlives one.
	_, err = s.Detach(t.Context(), "tok-far")
	assert.NilError(t, err)
	members, err = s.Attending(t.Context())
	assert.NilError(t, err)
	assert.Equal(t, len(members), 2)
}

func TestStore_AttendSeedsPositionOnceAtTheRoomsEnd(t *testing.T) {
	s, path := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	bob := attend(t, s, "tok-b", testRoom, "beta", "bob")

	send(t, s, senderOf(alice), toRoles(role("beta", "bob")), "one")
	send(t, s, senderOf(alice), toRoles(role("beta", "bob")), "two")

	// Bob reads the first of the two, which leaves its position at seq 1.
	msgs, remaining, err := s.Read(t.Context(), senderOf(bob), ReadFilter{Range: 1})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(msgs), []string{"one"})
	assert.Equal(t, remaining, 1)

	// Attending again, in this run or after a restart, must not push the
	// position forward: what arrived while the role was away is still waiting.
	_, err = s.Detach(t.Context(), bob.Token)
	assert.NilError(t, err)
	reopened := reopen(t, s, path)
	back := attend(t, reopened, "tok-b2", testRoom, "beta", "bob")

	msgs, remaining, err = reopened.Read(t.Context(), senderOf(back), ReadFilter{})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(msgs), []string{"two"})
	assert.Equal(t, remaining, 0)

	// Having been shown, it is not shown again.
	msgs, remaining, err = reopened.Read(t.Context(), senderOf(back), ReadFilter{})
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 0)
	assert.Equal(t, remaining, 0)
}

func TestStore_FirstAttendanceHasNothingUnread(t *testing.T) {
	s, _ := newTestStore(t)
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	send(t, s, senderOf(alice), Target{Kind: TargetEveryone}, "before you got here")

	// The position starts at the room's end, so a first attendance is not
	// handed the backlog it was never part of.
	carl := attend(t, s, "tok-c", testRoom, "beta", "carl")
	msgs, remaining, err := s.Read(t.Context(), senderOf(carl), ReadFilter{})
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 0)
	assert.Equal(t, remaining, 0)

	// The room's history is still there for it to ask for.
	msgs, err = s.ReadRoom(t.Context(), testRoom, ReadFilter{})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(msgs), []string{"before you got here"})
}
