package chat

import (
	"errors"
	"testing"

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
	svc, id := newTestAdminService(t)
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
		_, err := svc.MoveMember(adminCtx(t, adminNonce(t, svc, id)), &chatv1.MoveMemberRequest{
			Room: "/work", Team: "gamma", Name: "bob", ToTeam: "beta",
		})
		assert.Equal(t, status.Code(err), codes.AlreadyExists)
	})

	t.Run("a team name that breaks addressing is InvalidArgument", func(t *testing.T) {
		_, err := svc.MoveMember(adminCtx(t, adminNonce(t, svc, id)), &chatv1.MoveMemberRequest{
			Room: "/work", Team: "beta", Name: "bob", ToTeam: "beta/gamma",
		})
		assert.Equal(t, status.Code(err), codes.InvalidArgument)
	})
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
