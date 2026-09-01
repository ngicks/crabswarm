package chat

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/resolver"
)

func TestAdminService_ListRooms(t *testing.T) {
	svc, id := newTestAdminService(t)
	join(t, svc.store, "tok-a", "/work", "alpha", "ana")
	join(t, svc.store, "tok-b", "/work", "beta", "bob")
	join(t, svc.store, "tok-c", "/other", "alpha", "cho")

	res, err := svc.ListRooms(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.ListRoomsRequest{})
	assert.NilError(t, err)

	rooms := res.GetRooms()
	assert.Equal(t, len(rooms), 2)
	assert.Equal(t, rooms[0].GetName(), "/other")
	assert.Equal(t, len(rooms[0].GetMembers()), 1)
	assert.Equal(t, rooms[1].GetName(), "/work")

	// Teams are flattened into the member list, each member carrying its own.
	members := rooms[1].GetMembers()
	assert.Equal(t, len(members), 2)
	assert.Equal(t, members[0].GetTeam(), "alpha")
	assert.Equal(t, members[0].GetName(), "ana")
	assert.Equal(t, members[0].GetRoom(), "/work")
	assert.Equal(t, members[1].GetTeam(), "beta")
	assert.Equal(t, members[1].GetName(), "bob")

	// An empty store lists nothing rather than failing.
	empty, emptyID := newTestAdminService(t)
	res, err = empty.ListRooms(adminCtx(t, adminNonce(t, empty, emptyID)),
		&chatv1.ListRoomsRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(res.GetRooms()), 0)
}

func TestAdminService_MoveMember(t *testing.T) {
	svc, id, _, provider := newTestAdminServiceWith(t)
	join(t, svc.store, "tok-a", "/work", "alpha", "ana")
	join(t, svc.store, "tok-b", "/work", "beta", "bob")

	t.Run("moves within the room", func(t *testing.T) {
		res, err := svc.MoveMember(adminCtx(t, adminNonce(t, svc, id)), &chatv1.MoveMemberRequest{
			Room: "/work", Team: "alpha", Name: "ana", ToTeam: "beta",
		})
		assert.NilError(t, err)
		assert.Equal(t, res.GetMember().GetTeam(), "beta")
		assert.Equal(t, res.GetMember().GetName(), "ana")
		assert.Equal(t, res.GetMember().GetRoom(), "/work")

		// The move is what the store holds afterwards, token untouched.
		m, err := svc.store.Member(t.Context(), "tok-a")
		assert.NilError(t, err)
		assert.Equal(t, m.Team, "beta")
	})

	t.Run("an unknown member is NotFound", func(t *testing.T) {
		_, err := svc.MoveMember(adminCtx(t, adminNonce(t, svc, id)), &chatv1.MoveMemberRequest{
			Room: "/work", Team: "alpha", Name: "ana", ToTeam: "gamma",
		})
		assert.Equal(t, status.Code(err), codes.NotFound)
	})

	t.Run("a colliding name is AlreadyExists", func(t *testing.T) {
		join(t, svc.store, "tok-c", "/work", "gamma", "bob")
		// The provider still places the bob already in the target team, so that
		// bob keeps the name and the move is refused.
		provider.vouch("tok-b", "/work", "beta")
		_, err := svc.MoveMember(adminCtx(t, adminNonce(t, svc, id)), &chatv1.MoveMemberRequest{
			Room: "/work", Team: "gamma", Name: "bob", ToTeam: "beta",
		})
		assert.Equal(t, status.Code(err), codes.AlreadyExists)

		stored, err := svc.store.Member(t.Context(), "tok-b")
		assert.NilError(t, err)
		assert.Equal(t, stored.Team+"/"+stored.Name, "beta/bob")
	})

	t.Run("a team name that breaks addressing is InvalidArgument", func(t *testing.T) {
		_, err := svc.MoveMember(adminCtx(t, adminNonce(t, svc, id)), &chatv1.MoveMemberRequest{
			Room: "/work", Team: "beta", Name: "bob", ToTeam: "beta/gamma",
		})
		assert.Equal(t, status.Code(err), codes.InvalidArgument)
	})
}

// An operator meets the same rule a joiner does: a name carried by a member the
// provider no longer knows is not in the way, and the ghost goes with the move.
func TestAdminService_MoveMemberReclaimsTheNameOfAGoneMember(t *testing.T) {
	svc, id, _, provider := newTestAdminServiceWith(t)
	join(t, svc.store, "tok-old", "/work", "beta", "worker-1")
	join(t, svc.store, "tok-a", "/work", "alpha", "worker-1")
	provider.vouch("tok-a", "/work", "alpha")

	res, err := svc.MoveMember(adminCtx(t, adminNonce(t, svc, id)), &chatv1.MoveMemberRequest{
		Room: "/work", Team: "alpha", Name: "worker-1", ToTeam: "beta",
	})
	assert.NilError(t, err)
	assert.Equal(t, res.GetMember().GetTeam(), "beta")

	_, err = svc.store.Member(t.Context(), "tok-old")
	assert.ErrorIs(t, err, ErrNotFound)
	moved, err := svc.store.Member(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, moved.Team+"/"+moved.Name, "beta/worker-1")
}

// A human's token was minted by the daemon and no provider can vouch for it, so
// the name an operator's move collides with stays the human's.
func TestAdminService_MoveMemberNeverReclaimsAHumanName(t *testing.T) {
	svc, id, _, provider := newTestAdminServiceWith(t)
	_, err := svc.store.Join(t.Context(), Member{
		Token: "human-tok", Name: "hana", Team: "beta", Room: "/work", Kind: KindHuman,
	})
	assert.NilError(t, err)
	join(t, svc.store, "tok-a", "/work", "alpha", "hana")

	_, err = svc.MoveMember(adminCtx(t, adminNonce(t, svc, id)), &chatv1.MoveMemberRequest{
		Room: "/work", Team: "alpha", Name: "hana", ToTeam: "beta",
	})
	assert.Equal(t, status.Code(err), codes.AlreadyExists)

	stored, err := svc.store.Member(t.Context(), "human-tok")
	assert.NilError(t, err)
	assert.Equal(t, stored.Name, "hana")
	assert.Equal(t, provider.callCount(), 0)
}

// A cmdman that could not be asked says nothing about the member holding the
// name, and nothing is not enough to move it aside.
func TestAdminService_MoveMemberKeepsAHolderTheProviderCouldNotJudge(t *testing.T) {
	svc, id, _, provider := newTestAdminServiceWith(t)
	join(t, svc.store, "tok-old", "/work", "beta", "worker-1")
	join(t, svc.store, "tok-a", "/work", "alpha", "worker-1")
	provider.failLookup("tok-old", errors.New("cmdman: connection refused"))

	_, err := svc.MoveMember(adminCtx(t, adminNonce(t, svc, id)), &chatv1.MoveMemberRequest{
		Room: "/work", Team: "alpha", Name: "worker-1", ToTeam: "beta",
	})
	assert.Equal(t, status.Code(err), codes.AlreadyExists)

	stored, err := svc.store.Member(t.Context(), "tok-old")
	assert.NilError(t, err)
	assert.Equal(t, stored.Team+"/"+stored.Name, "beta/worker-1")
	stayed, err := svc.store.Member(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, stayed.Team, "alpha")
}

// A daemon with no team-info provider has nothing that could show a holder
// gone, so every collision an operator's move runs into stays a refusal — the
// holder is left alone rather than asked about by nobody.
func TestAdminService_MoveMemberWithoutAProviderRefusesTheCollision(t *testing.T) {
	svc, id, _ := newTestAdminServiceOver(t, nil)
	// Both are agents: a human holder is kept without the provider being
	// consulted at all, which would leave the collision refused either way.
	join(t, svc.store, "tok-old", "/work", "beta", "worker-1")
	join(t, svc.store, "tok-a", "/work", "alpha", "worker-1")

	_, err := svc.MoveMember(adminCtx(t, adminNonce(t, svc, id)), &chatv1.MoveMemberRequest{
		Room: "/work", Team: "alpha", Name: "worker-1", ToTeam: "beta",
	})
	assert.Equal(t, status.Code(err), codes.AlreadyExists)

	held, err := svc.store.Member(t.Context(), "tok-old")
	assert.NilError(t, err)
	assert.Equal(t, held.Team+"/"+held.Name, "beta/worker-1")
	stayed, err := svc.store.Member(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, stayed.Team+"/"+stayed.Name, "alpha/worker-1")
}

// A move stays inside the room, so what the room hears is an address change:
// the member it knew is gone and one of that name is now in another team.
func TestAdminService_MoveMemberPublishesTheAddressChange(t *testing.T) {
	svc, id := newTestAdminService(t)
	join(t, svc.store, "tok-a", "/work", "alpha", "ana")

	sub := svc.store.events.subscribe("/work")
	defer svc.store.events.unsubscribe(sub)

	_, err := svc.MoveMember(adminCtx(t, adminNonce(t, svc, id)), &chatv1.MoveMemberRequest{
		Room: "/work", Team: "alpha", Name: "ana", ToTeam: "beta",
	})
	assert.NilError(t, err)
	assert.Equal(t, describeEvent(nextEvent(t, sub.events)), "left:alpha/ana")
	assert.Equal(t, describeEvent(nextEvent(t, sub.events)), "joined:beta/ana")

	// Moving into the team the member already occupies changes no address.
	_, err = svc.MoveMember(adminCtx(t, adminNonce(t, svc, id)), &chatv1.MoveMemberRequest{
		Room: "/work", Team: "beta", Name: "ana", ToTeam: "beta",
	})
	assert.NilError(t, err)
	noMoreEvents(t, sub.events)
}

// Registering is where a human enters the room: their later Join finds the
// membership already made, so no other moment can announce their arrival.
func TestAdminService_RegisterMemberPublishesTheArrival(t *testing.T) {
	svc, id := newTestAdminService(t)

	sub := svc.store.events.subscribe("/work")
	defer svc.store.events.unsubscribe(sub)

	_, err := svc.RegisterMember(
		adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.RegisterMemberRequest{Room: "/work", Team: "hosts", Name: "hana"},
	)
	assert.NilError(t, err)
	assert.Equal(t, describeEvent(nextEvent(t, sub.events)), "joined:hosts/hana")
}

func TestAdminService_RegisterMember(t *testing.T) {
	svc, id := newTestAdminService(t)

	res, err := svc.RegisterMember(
		adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.RegisterMemberRequest{
			Room: "/work", Team: "hosts", Name: "hana",
		},
	)
	assert.NilError(t, err)
	assert.Equal(t, res.GetMember().GetName(), "hana")
	assert.Equal(t, res.GetMember().GetTeam(), "hosts")
	assert.Equal(t, res.GetMember().GetRoom(), "/work")

	// The minted token is a secret, and it is what the store holds the member
	// under, as a human the provider is never asked about.
	token := res.GetToken()
	assert.Assert(t, len(token) >= 26)
	stored, err := svc.store.Member(t.Context(), token)
	assert.NilError(t, err)
	assert.Equal(t, stored.Kind, KindHuman)
	assert.Equal(t, stored.Name, "hana")

	t.Run("a second registration mints a different token", func(t *testing.T) {
		res2, err := svc.RegisterMember(
			adminCtx(t, adminNonce(t, svc, id)),
			&chatv1.RegisterMemberRequest{
				Room: "/work", Team: "hosts", Name: "hugo",
			},
		)
		assert.NilError(t, err)
		assert.Assert(t, res2.GetToken() != token)
	})

	t.Run("a taken name is AlreadyExists", func(t *testing.T) {
		_, err := svc.RegisterMember(
			adminCtx(t, adminNonce(t, svc, id)),
			&chatv1.RegisterMemberRequest{
				Room: "/work", Team: "hosts", Name: "hana",
			},
		)
		assert.Equal(t, status.Code(err), codes.AlreadyExists)
	})

	t.Run("the same name in another team is fine", func(t *testing.T) {
		_, err := svc.RegisterMember(
			adminCtx(t, adminNonce(t, svc, id)),
			&chatv1.RegisterMemberRequest{
				Room: "/work", Team: "guests", Name: "hana",
			},
		)
		assert.NilError(t, err)
	})

	t.Run("an empty room is InvalidArgument", func(t *testing.T) {
		_, err := svc.RegisterMember(
			adminCtx(t, adminNonce(t, svc, id)),
			&chatv1.RegisterMemberRequest{
				Team: "hosts", Name: "hana",
			},
		)
		assert.Equal(t, status.Code(err), codes.InvalidArgument)
	})

	t.Run("an empty name is InvalidArgument", func(t *testing.T) {
		_, err := svc.RegisterMember(
			adminCtx(t, adminNonce(t, svc, id)),
			&chatv1.RegisterMemberRequest{
				Room: "/work", Team: "hosts",
			},
		)
		assert.Equal(t, status.Code(err), codes.InvalidArgument)
	})

	// The name an admin message is attributed to belongs to nobody, so a
	// recipient reading it cannot be reading a peer that named itself well.
	t.Run("the reserved name is InvalidArgument", func(t *testing.T) {
		_, err := svc.RegisterMember(
			adminCtx(t, adminNonce(t, svc, id)),
			&chatv1.RegisterMemberRequest{
				Room: "/work", Team: "hosts", Name: adminName,
			},
		)
		assert.Equal(t, status.Code(err), codes.InvalidArgument)

		// Not the admin RPC's rule but the store's: an agent joining under the
		// name is refused the same way.
		_, err = svc.store.Join(t.Context(), Member{
			Token: "tok-admin", Name: adminName, Team: "alpha",
			Room: "/work", Kind: KindAgent,
		})
		assert.ErrorIs(t, err, ErrInvalidName)

		// The team is reserved too: members of a team named admin would win
		// bare-name resolution for admin sends and render as admin/<name>.
		_, err = svc.store.Join(t.Context(), Member{
			Token: "tok-admteam", Name: "carl", Team: adminName,
			Room: "/work", Kind: KindAgent,
		})
		assert.ErrorIs(t, err, ErrInvalidName)
	})
}

func TestAdminService_Send(t *testing.T) {
	svc, id, notifier := newTestAdminServiceWithNotifier(t)
	join(t, svc.store, "tok-a", "/work", "alpha", "ana")
	join(t, svc.store, "tok-b", "/work", "beta", "bob")
	join(t, svc.store, "tok-c", "/other", "alpha", "cho")

	t.Run("a call carrying no credential is refused", func(t *testing.T) {
		_, err := svc.Send(t.Context(), &chatv1.AdminSendRequest{
			Room: "/work", Target: "alpha/ana", Text: "hi",
		})
		assert.Equal(t, status.Code(err), codes.PermissionDenied)
		assert.Equal(t, countRows(t, svc.store, `SELECT COUNT(*) FROM messages`), 0)
	})

	t.Run("delivers to the addressed member as the operator", func(t *testing.T) {
		before := len(notifier.notified())
		res, err := svc.Send(adminCtx(t, adminNonce(t, svc, id)), &chatv1.AdminSendRequest{
			Room: "/work", Target: "alpha/ana", Text: "stand down",
		})
		assert.NilError(t, err)
		assert.Equal(t, res.GetDelivered(), int32(1))

		// The recipient reads the message as coming from the operator, not from
		// anyone in the room.
		msgs, err := svc.store.Read(t.Context(), "tok-a")
		assert.NilError(t, err)
		assert.Equal(t, len(msgs), 1)
		assert.Equal(t, msgs[0].Text, "stand down")
		assert.Equal(t, msgs[0].From, Sender{Name: "admin", Team: "admin", Room: "/work"})

		// Nobody else was written to, and the notifier heard about the one
		// delivery the way it hears about a member's.
		assert.Equal(t, countRows(t, svc.store, `SELECT COUNT(*) FROM messages`), 0)
		got := notifier.notified()[before:]
		assert.Equal(t, len(got), 1)
		assert.Equal(t, got[0].recipient.Name, "ana")
		assert.Equal(t, got[0].from, Sender{Name: "admin", Team: "admin", Room: "/work"})
	})

	t.Run("a bare name resolves as it does for a member", func(t *testing.T) {
		res, err := svc.Send(adminCtx(t, adminNonce(t, svc, id)), &chatv1.AdminSendRequest{
			Room: "/work", Target: "bob", Text: "you too",
		})
		assert.NilError(t, err)
		assert.Equal(t, res.GetDelivered(), int32(1))

		msgs, err := svc.store.Read(t.Context(), "tok-b")
		assert.NilError(t, err)
		assert.Equal(t, len(msgs), 1)
	})

	t.Run("* reaches everyone in the room and nobody outside it", func(t *testing.T) {
		before := len(notifier.notified())
		res, err := svc.Send(adminCtx(t, adminNonce(t, svc, id)), &chatv1.AdminSendRequest{
			Room: "/work", Target: "*", Text: "all hands",
		})
		assert.NilError(t, err)
		assert.Equal(t, res.GetDelivered(), int32(2))
		assert.Equal(t, len(notifier.notified()[before:]), 2)

		for _, token := range []string{"tok-a", "tok-b"} {
			msgs, err := svc.store.Read(t.Context(), token)
			assert.NilError(t, err)
			assert.Equal(t, len(msgs), 1)
			assert.Equal(t, msgs[0].Text, "all hands")
			assert.Equal(t, msgs[0].From.Name, "admin")
		}
		// The member of the other room shares a team name with one of them and
		// still hears nothing: the room is the boundary.
		msgs, err := svc.store.Read(t.Context(), "tok-c")
		assert.NilError(t, err)
		assert.Equal(t, len(msgs), 0)
	})

	t.Run("an unknown room is NotFound", func(t *testing.T) {
		_, err := svc.Send(adminCtx(t, adminNonce(t, svc, id)), &chatv1.AdminSendRequest{
			Room: "/nowhere", Target: "alpha/ana", Text: "hi",
		})
		assert.Equal(t, status.Code(err), codes.NotFound)

		// Addressing the whole of a room nobody attends is the same mistake.
		_, err = svc.Send(adminCtx(t, adminNonce(t, svc, id)), &chatv1.AdminSendRequest{
			Room: "/nowhere", Target: "*", Text: "hi",
		})
		assert.Equal(t, status.Code(err), codes.NotFound)
	})

	t.Run("an unknown target is NotFound", func(t *testing.T) {
		_, err := svc.Send(adminCtx(t, adminNonce(t, svc, id)), &chatv1.AdminSendRequest{
			Room: "/work", Target: "alpha/nobody", Text: "hi",
		})
		assert.Equal(t, status.Code(err), codes.NotFound)

		// A member of another room is unknown here too.
		_, err = svc.Send(adminCtx(t, adminNonce(t, svc, id)), &chatv1.AdminSendRequest{
			Room: "/work", Target: "alpha/cho", Text: "hi",
		})
		assert.Equal(t, status.Code(err), codes.NotFound)
	})

	t.Run("an empty field is InvalidArgument", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			req  *chatv1.AdminSendRequest
		}{
			{"room", &chatv1.AdminSendRequest{Target: "alpha/ana", Text: "hi"}},
			{"target", &chatv1.AdminSendRequest{Room: "/work", Text: "hi"}},
			{"text", &chatv1.AdminSendRequest{Room: "/work", Target: "alpha/ana"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, err := svc.Send(adminCtx(t, adminNonce(t, svc, id)), tc.req)
				assert.Equal(t, status.Code(err), codes.InvalidArgument)
			})
		}
	})

	// Speaking into a room never puts the operator in it: no member row, so
	// nothing to address back, to nudge, or to reap.
	assert.Equal(t, countRows(t, svc.store, `SELECT COUNT(*) FROM members`), 3)
	assert.Equal(t, countRows(t, svc.store,
		`SELECT COUNT(*) FROM members WHERE name = ?`, "admin"), 0)
}

func TestAdminService_History(t *testing.T) {
	svc, id := newTestAdminService(t)
	join(t, svc.store, "tok-a", "/work", "alpha", "ana")
	join(t, svc.store, "tok-b", "/work", "beta", "bob")
	join(t, svc.store, "tok-c", "/other", "alpha", "cho")

	_, err := svc.store.Send(t.Context(), "tok-a", "beta/bob", "just for you", sentAt)
	assert.NilError(t, err)
	_, err = svc.store.Broadcast(
		t.Context(), "tok-a", "for everyone", sentAt.Add(time.Minute), true)
	assert.NilError(t, err)

	t.Run("a call carrying no credential is refused", func(t *testing.T) {
		_, err := svc.History(t.Context(), &chatv1.AdminHistoryRequest{Room: "/work"})
		assert.Equal(t, status.Code(err), codes.PermissionDenied)
	})

	t.Run("an empty room is InvalidArgument", func(t *testing.T) {
		_, err := svc.History(adminCtx(t, adminNonce(t, svc, id)),
			&chatv1.AdminHistoryRequest{})
		assert.Equal(t, status.Code(err), codes.InvalidArgument)
	})

	t.Run("reads the whole room, oldest first", func(t *testing.T) {
		res, err := svc.History(adminCtx(t, adminNonce(t, svc, id)),
			&chatv1.AdminHistoryRequest{Room: "/work"})
		assert.NilError(t, err)
		entries := res.GetEntries()
		assert.Equal(t, len(entries), 2)

		// A directed send is in the transcript the operator never received,
		// with the member it was addressed to spelled out.
		assert.Equal(t, entries[0].GetText(), "just for you")
		assert.Equal(t, entries[0].GetFrom().GetName(), "ana")
		assert.Equal(t, entries[0].GetFrom().GetTeam(), "alpha")
		assert.Equal(t, entries[0].GetFrom().GetRoom(), "/work")
		assert.Equal(t, entries[0].GetTo().GetName(), "bob")
		assert.Assert(t, entries[0].GetSentAt().AsTime().Equal(sentAt))

		// A broadcast addressed the room rather than anyone in it.
		assert.Equal(t, entries[1].GetText(), "for everyone")
		assert.Assert(t, entries[1].GetTo() == nil)

		// Every entry carries the cursor to ask for the next ones by, and they
		// grow with the conversation.
		assert.Assert(t, entries[0].GetId() > 0)
		assert.Assert(t, entries[1].GetId() > entries[0].GetId())

		// Reading consumed nothing: the addressee still has its message.
		msgs, err := svc.store.Read(t.Context(), "tok-b")
		assert.NilError(t, err)
		assert.Equal(t, len(msgs), 2)
	})

	t.Run("a room nothing was said in is empty rather than NotFound", func(t *testing.T) {
		// One room exists and has been silent, the other was never heard of at
		// all; there is nothing to tell apart between them.
		for _, room := range []string{"/other", "/nowhere"} {
			res, err := svc.History(adminCtx(t, adminNonce(t, svc, id)),
				&chatv1.AdminHistoryRequest{Room: room})
			assert.NilError(t, err)
			assert.Equal(t, len(res.GetEntries()), 0)
		}
	})
}

// The window is counted back from the newest entry, and a request that names no
// window gets a screenful rather than the whole retained room.
func TestAdminService_HistoryWindow(t *testing.T) {
	svc, id := newTestAdminService(t)
	join(t, svc.store, "tok-a", "/work", "alpha", "ana")
	for i := range defaultHistoryWindow + 5 {
		_, err := svc.store.Broadcast(t.Context(), "tok-a",
			fmt.Sprintf("line %d", i), sentAt.Add(time.Duration(i)*time.Minute), true)
		assert.NilError(t, err)
	}

	res, err := svc.History(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.AdminHistoryRequest{Room: "/work"})
	assert.NilError(t, err)
	entries := res.GetEntries()
	assert.Equal(t, len(entries), defaultHistoryWindow)
	assert.Equal(t, entries[0].GetText(), "line 5")
	assert.Equal(t, entries[len(entries)-1].GetText(),
		fmt.Sprintf("line %d", defaultHistoryWindow+4))

	res, err = svc.History(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.AdminHistoryRequest{Room: "/work", Limit: 2})
	assert.NilError(t, err)
	entries = res.GetEntries()
	assert.Equal(t, len(entries), 2)
	assert.Equal(t, entries[0].GetText(), fmt.Sprintf("line %d", defaultHistoryWindow+3))
	assert.Equal(t, entries[1].GetText(), fmt.Sprintf("line %d", defaultHistoryWindow+4))
}

// Following a room is asking for what came after the last entry seen, which is
// what keeps a reader that polls from re-reading the window it already has.
func TestAdminService_HistoryPagesFromACursor(t *testing.T) {
	svc, id := newTestAdminService(t)
	join(t, svc.store, "tok-a", "/work", "alpha", "ana")
	say := func(text string) {
		t.Helper()
		_, err := svc.store.Broadcast(t.Context(), "tok-a", text, sentAt, true)
		assert.NilError(t, err)
	}
	page := func(sinceID int64, limit int32) []*chatv1.AdminHistoryEntry {
		t.Helper()
		res, err := svc.History(adminCtx(t, adminNonce(t, svc, id)),
			&chatv1.AdminHistoryRequest{Room: "/work", SinceId: sinceID, Limit: limit})
		assert.NilError(t, err)
		return res.GetEntries()
	}
	for i := range 3 {
		say(fmt.Sprintf("line %d", i))
	}

	said := page(0, 0)
	assert.Equal(t, len(said), 3)

	// One step forward from the oldest is the next entry alone, carrying the id
	// the step after it is asked for by.
	next := page(said[0].GetId(), 1)
	assert.Equal(t, len(next), 1)
	assert.Equal(t, next[0].GetText(), "line 1")
	assert.Equal(t, next[0].GetId(), said[1].GetId())

	// The rest of the room follows from there, oldest first.
	rest := page(next[0].GetId(), 0)
	assert.Equal(t, len(rest), 1)
	assert.Equal(t, rest[0].GetText(), "line 2")

	// A reader that is up to date is handed nothing rather than the tail again,
	// and hears the next utterance on its next ask.
	assert.Equal(t, len(page(said[2].GetId(), 0)), 0)
	say("line 3")
	fresh := page(said[2].GetId(), 0)
	assert.Equal(t, len(fresh), 1)
	assert.Equal(t, fresh[0].GetText(), "line 3")
}

// A window wider than the room is allowed to keep is answered with what the cap
// allows, not refused: the operator asking for a screenful of a room that keeps
// one line gets the one line.
func TestAdminService_HistoryClampsToTheRetentionCap(t *testing.T) {
	store, path := newTestStoreWithHistory(t, 0)
	join(t, store, "tok-a", "/work", "alpha", "ana")
	for i := range 3 {
		_, err := store.Broadcast(t.Context(), "tok-a",
			fmt.Sprintf("line %d", i), sentAt.Add(time.Duration(i)*time.Minute), true)
		assert.NilError(t, err)
	}
	assert.NilError(t, store.Close())

	// Pruning happens on insert, so a run under a tighter cap starts out holding
	// more than the cap: three lines recorded, one of them keepable.
	tight, err := NewStore(t.Context(), path, 1)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = tight.Close() })
	assert.Equal(t, countRows(t, tight, `SELECT COUNT(*) FROM room_log`), 3)

	svc, id, _ := newTestAdminServiceOn(t, tight,
		&fakeProvider{infos: map[string]resolver.TeamInfo{}})
	res, err := svc.History(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.AdminHistoryRequest{Room: "/work"})
	assert.NilError(t, err)
	entries := res.GetEntries()
	assert.Equal(t, len(entries), 1)
	assert.Equal(t, entries[0].GetText(), "line 2")
}

// TestAdminService_RegisteredMemberChatsAsHuman is the point of RegisterMember:
// the printed token is a working ChatService identity even though no team-info
// provider has ever heard of it.
func TestAdminService_RegisteredMemberChatsAsHuman(t *testing.T) {
	admin, id := newTestAdminService(t)
	// The provider fails every lookup: a human's token must never depend on it.
	provider := &fakeProvider{
		infos: map[string]resolver.TeamInfo{},
		err:   errors.New("cmdman: connection refused"),
	}
	member := NewService(admin.store, provider, nil, nil, nil)

	registered, err := admin.RegisterMember(
		adminCtx(t, adminNonce(t, admin, id)),
		&chatv1.RegisterMemberRequest{
			Room: "/work", Team: "hosts", Name: "hana",
		},
	)
	assert.NilError(t, err)
	token := registered.GetToken()

	joined, err := member.Join(callCtx(t, token), &chatv1.JoinRequest{Name: "ignored"})
	assert.NilError(t, err)
	assert.Equal(t, joined.GetSelf().GetName(), "hana")
	assert.Equal(t, joined.GetSelf().GetRoom(), "/work")

	seedAgent(t, member, provider, "tok-a", "/work", "alpha", "ana")
	list, err := member.ListMembers(callCtx(t, token), &chatv1.ListMembersRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(list.GetMembers()), 2)

	_, err = member.Send(callCtx(t, token), &chatv1.SendRequest{To: "ana", Text: "hi"})
	assert.NilError(t, err)
	assert.Equal(t, provider.callCount(), 0)
}
