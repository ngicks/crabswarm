package chat

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestStore_JoinIsIdempotentPerToken(t *testing.T) {
	s, _ := newTestStore(t)
	first := join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	assert.Equal(t, first.State, StateIdle) // default for a fresh member
	join(t, s, "tok-b", "/work/repo", "alpha", "bob")
	_, err := s.Send(t.Context(), "tok-b", "alice", "hi", sentAt)
	assert.NilError(t, err)

	// A hook firing twice re-joins with the same token: same member back, no
	// second row, and the pending message is untouched.
	second, err := s.Join(t.Context(), Member{
		Token: "tok-a", Name: "alice", Team: "alpha", Room: "/work/repo", Kind: KindAgent,
	})
	assert.NilError(t, err)
	assert.DeepEqual(t, second, first)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM members`), 2)

	// Even a re-join carrying different details is a no-op: the stored member
	// wins, and the inbox is still there.
	third, err := s.Join(t.Context(), Member{
		Token: "tok-a", Name: "renamed", Team: "beta", Room: "/work/other", Kind: KindHuman,
	})
	assert.NilError(t, err)
	assert.DeepEqual(t, third, first)

	msgs, err := s.Read(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 1)
}

func TestStore_JoinRejectsTakenName(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")

	// Same name, same team, different token: rejected.
	_, err := s.Join(t.Context(), Member{
		Token: "tok-other", Name: "alice", Team: "alpha", Room: "/work/repo", Kind: KindAgent,
	})
	assert.ErrorIs(t, err, ErrNameTaken)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM members`), 1)

	// The same name in another team of the room is what teams are for.
	_, err = s.Join(t.Context(), Member{
		Token: "tok-b", Name: "alice", Team: "beta", Room: "/work/repo", Kind: KindAgent,
	})
	assert.NilError(t, err)

	// So is the same name in another room.
	_, err = s.Join(t.Context(), Member{
		Token: "tok-c", Name: "alice", Team: "alpha", Room: "/work/other", Kind: KindAgent,
	})
	assert.NilError(t, err)
}

func TestStore_JoinValidatesMember(t *testing.T) {
	s, _ := newTestStore(t)

	for _, tc := range []struct {
		name   string
		member Member
	}{
		{"slash in name", Member{
			Token: "t", Name: "a/b", Team: "alpha", Room: "r", Kind: KindAgent}},
		{"slash in team", Member{
			Token: "t", Name: "alice", Team: "al/pha", Room: "r", Kind: KindAgent}},
		{"empty name", Member{
			Token: "t", Name: "", Team: "alpha", Room: "r", Kind: KindAgent}},
		{"empty team", Member{
			Token: "t", Name: "alice", Team: "", Room: "r", Kind: KindAgent}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.Join(t.Context(), tc.member)
			assert.ErrorIs(t, err, ErrInvalidName)
		})
	}

	// A token, a room and a known kind are all required.
	_, err := s.Join(t.Context(), Member{Name: "alice", Team: "alpha", Room: "r", Kind: KindAgent})
	assert.Assert(t, err != nil, "empty token should fail")
	_, err = s.Join(t.Context(), Member{Token: "t", Name: "alice", Team: "alpha", Kind: KindAgent})
	assert.Assert(t, err != nil, "empty room should fail")
	_, err = s.Join(t.Context(), Member{Token: "t", Name: "alice", Team: "alpha", Room: "r"})
	assert.Assert(t, err != nil, "unknown kind should fail")
}

func TestStore_JoinRegistersHuman(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")

	// An admin-registered human is stored the same way; only Kind differs, and
	// the daemon-issued token is opaque to the store.
	human, err := s.Join(t.Context(), Member{
		Token: "daemon-issued-secret", Name: "watage", Team: "humans",
		Room: "/work/repo", Kind: KindHuman,
	})
	assert.NilError(t, err)
	assert.Equal(t, human.Kind, KindHuman)

	// They are an ordinary room member: addressable and able to read.
	to, err := s.Send(t.Context(), "tok-a", "humans/watage", "ping", sentAt)
	assert.NilError(t, err)
	assert.Equal(t, to.Token, "daemon-issued-secret")
	msgs, err := s.Read(t.Context(), "daemon-issued-secret")
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 1)
}

func TestStore_ResolvePrefersOwnTeam(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-caller", "/work/repo", "alpha", "caller")
	mine := join(t, s, "tok-mine", "/work/repo", "alpha", "worker")
	theirs := join(t, s, "tok-theirs", "/work/repo", "beta", "worker")

	// A bare name that exists in the caller's own team resolves there, even
	// though another team has the same name.
	got, err := s.Resolve(t.Context(), "tok-caller", "worker")
	assert.NilError(t, err)
	assert.Equal(t, got.Token, mine.Token)

	// The other team's member is reachable with the qualified form.
	got, err = s.Resolve(t.Context(), "tok-caller", "beta/worker")
	assert.NilError(t, err)
	assert.Equal(t, got.Token, theirs.Token)
}

func TestStore_ResolveUniqueAcrossRoom(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-caller", "/work/repo", "alpha", "caller")
	other := join(t, s, "tok-other", "/work/repo", "beta", "reviewer")

	// Not in the caller's team, but unique in the room: no qualification needed.
	got, err := s.Resolve(t.Context(), "tok-caller", "reviewer")
	assert.NilError(t, err)
	assert.Equal(t, got.Token, other.Token)
}

func TestStore_ResolveAmbiguousAcrossTeams(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-caller", "/work/repo", "alpha", "caller")
	join(t, s, "tok-b", "/work/repo", "beta", "worker")
	join(t, s, "tok-c", "/work/repo", "gamma", "worker")

	// Two other teams carry the name and the caller's own team does not, so
	// the bare form cannot pick one.
	_, err := s.Resolve(t.Context(), "tok-caller", "worker")
	assert.ErrorIs(t, err, ErrAmbiguousName)
	assert.ErrorContains(t, err, "beta")
	assert.ErrorContains(t, err, "gamma")
	assert.ErrorContains(t, err, "<team>/worker")

	// Qualifying the address resolves it.
	got, err := s.Resolve(t.Context(), "tok-caller", "gamma/worker")
	assert.NilError(t, err)
	assert.Equal(t, got.Token, "tok-c")
}

func TestStore_ResolveNotFound(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-caller", "/work/repo", "alpha", "caller")
	join(t, s, "tok-far", "/work/elsewhere", "alpha", "stranger")

	for _, addr := range []string{
		"nobody",            // no such name anywhere in the room
		"alpha/nobody",      // right team, no such name
		"nosuchteam/caller", // no such team
		"stranger",          // another room's member is invisible
		"alpha/stranger",    // ... qualified too
	} {
		t.Run(addr, func(t *testing.T) {
			_, err := s.Resolve(t.Context(), "tok-caller", addr)
			assert.ErrorIs(t, err, ErrNotFound)
		})
	}

	// An unknown caller resolves nothing at all.
	_, err := s.Resolve(t.Context(), "tok-unknown", "caller")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestStore_SetState(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")

	for _, state := range []MemberState{StateRunning, StateWaitingInput, StateIdle} {
		assert.NilError(t, s.SetState(t.Context(), "tok-a", state))
		got, err := s.Member(t.Context(), "tok-a")
		assert.NilError(t, err)
		assert.Equal(t, got.State, state)
	}

	assert.Assert(t, s.SetState(t.Context(), "tok-a", "dozing") != nil,
		"an unknown state should be rejected")
	assert.ErrorIs(t, s.SetState(t.Context(), "tok-unknown", StateIdle), ErrNotFound)
}

func TestStore_ListMembersIsOrderedAndRoomScoped(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-1", "/work/repo", "beta", "zoe")
	join(t, s, "tok-2", "/work/repo", "alpha", "bob")
	join(t, s, "tok-3", "/work/repo", "beta", "amy")
	join(t, s, "tok-4", "/work/repo", "alpha", "alice")
	join(t, s, "tok-5", "/work/elsewhere", "alpha", "stranger")

	members, err := s.ListMembers(t.Context(), "/work/repo")
	assert.NilError(t, err)
	got := make([]string, len(members))
	for i, m := range members {
		got[i] = m.Team + "/" + m.Name
	}
	assert.DeepEqual(t, got, []string{
		"alpha/alice", "alpha/bob", "beta/amy", "beta/zoe",
	})

	// An empty room simply lists nothing.
	members, err = s.ListMembers(t.Context(), "/work/nowhere")
	assert.NilError(t, err)
	assert.Equal(t, len(members), 0)
}

func TestStore_ListRooms(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-1", "/work/repo", "beta", "zoe")
	join(t, s, "tok-2", "/work/repo", "alpha", "alice")
	join(t, s, "tok-3", "/work/repo", "alpha", "bob")
	join(t, s, "tok-4", "/work/elsewhere", "gamma", "stranger")

	rooms, err := s.ListRooms(t.Context())
	assert.NilError(t, err)
	assert.Equal(t, len(rooms), 2)

	// Rooms, teams and members all come out sorted by name.
	assert.Equal(t, rooms[0].Name, "/work/elsewhere")
	assert.Equal(t, len(rooms[0].Teams), 1)
	assert.Equal(t, rooms[0].Teams[0].Name, "gamma")

	assert.Equal(t, rooms[1].Name, "/work/repo")
	assert.Equal(t, len(rooms[1].Teams), 2)
	assert.Equal(t, rooms[1].Teams[0].Name, "alpha")
	assert.Equal(t, len(rooms[1].Teams[0].Members), 2)
	assert.Equal(t, rooms[1].Teams[0].Members[0].Name, "alice")
	assert.Equal(t, rooms[1].Teams[0].Members[1].Name, "bob")
	assert.Equal(t, rooms[1].Teams[1].Name, "beta")
	assert.Equal(t, rooms[1].Teams[1].Members[0].Name, "zoe")
}

func TestStore_RemoveMemberDropsInbox(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	join(t, s, "tok-b", "/work/repo", "alpha", "bob")
	_, err := s.Send(t.Context(), "tok-b", "alice", "unread", sentAt)
	assert.NilError(t, err)

	removed, err := s.RemoveMember(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, removed.Name, "alice")

	_, err = s.Member(t.Context(), "tok-a")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Equal(t, countRows(t, s,
		`SELECT COUNT(*) FROM messages WHERE recipient = ?`, "tok-a"), 0)

	// Removing again reports the member is gone.
	_, err = s.RemoveMember(t.Context(), "tok-a")
	assert.ErrorIs(t, err, ErrNotFound)

	// The freed name can be taken by someone else.
	_, err = s.Join(t.Context(), Member{
		Token: "tok-new", Name: "alice", Team: "alpha", Room: "/work/repo", Kind: KindAgent,
	})
	assert.NilError(t, err)
}

func TestStore_MoveMember(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	join(t, s, "tok-b", "/work/repo", "beta", "bob")
	join(t, s, "tok-c", "/work/repo", "gamma", "alice")

	moved, err := s.MoveMember(t.Context(), "tok-a", "beta")
	assert.NilError(t, err)
	assert.Equal(t, moved.Team, "beta")
	assert.Equal(t, moved.Room, "/work/repo")

	// The move is what addressing now sees.
	got, err := s.Resolve(t.Context(), "tok-b", "alice")
	assert.NilError(t, err)
	assert.Equal(t, got.Token, "tok-a")

	// A name collision in the target team blocks the move.
	_, err = s.MoveMember(t.Context(), "tok-a", "gamma")
	assert.ErrorIs(t, err, ErrNameTaken)
	unchanged, err := s.Member(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, unchanged.Team, "beta")

	// Moving into the team the member is already in changes nothing.
	same, err := s.MoveMember(t.Context(), "tok-a", "beta")
	assert.NilError(t, err)
	assert.DeepEqual(t, same, unchanged)

	_, err = s.MoveMember(t.Context(), "tok-unknown", "beta")
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = s.MoveMember(t.Context(), "tok-a", "in/valid")
	assert.ErrorIs(t, err, ErrInvalidName)
}
