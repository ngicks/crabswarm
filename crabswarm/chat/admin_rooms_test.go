package chat

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

func TestAdminService_ListRooms(t *testing.T) {
	svc, id := newTestAdminService(t)
	attend(t, svc.store, "tok-a", testRoom, "alpha", "ana")
	attend(t, svc.store, "tok-c", testRoom, "beta", "carl")
	gone := attend(t, svc.store, "tok-g", "/work/quiet", "alpha", "ghost")
	_, err := svc.store.Detach(t.Context(), gone.Token)
	assert.NilError(t, err)

	res, err := svc.ListRooms(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.ListRoomsRequest{})
	assert.NilError(t, err)

	// Both rooms, by name. The empty one is listed all the same: its
	// conversation and its read positions are still there, which is what an
	// operator deleting a room is deciding about.
	assert.Equal(t, len(res.GetRooms()), 2)
	assert.Equal(t, res.GetRooms()[0].GetName(), "/work/quiet")
	assert.Equal(t, len(res.GetRooms()[0].GetMembers()), 0)
	assert.Equal(t, res.GetRooms()[1].GetName(), testRoom)
	assert.DeepEqual(t, addresses(res.GetRooms()[1].GetMembers()),
		[]string{"alpha/ana", "beta/carl"})
}

func TestAdminService_RegisterMember(t *testing.T) {
	svc, id := newTestAdminService(t)

	res, err := svc.RegisterMember(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.RegisterMemberRequest{Room: testRoom, Team: "hosts", Name: "hana"})
	assert.NilError(t, err)
	assert.Equal(t, address(res.GetMember()), "hosts/hana")
	assert.Equal(t, res.GetMember().GetKind(), chatv1.MemberKind_MEMBER_KIND_HUMAN)
	assert.Assert(t, res.GetToken() != "")

	// Attending from the moment they are registered, with no stream holding it:
	// a person's terminal comes and goes, and the role must not go with it.
	stored, err := svc.store.Member(t.Context(), res.GetToken())
	assert.NilError(t, err)
	assert.Equal(t, addressOf(stored), "hosts/hana")
	assert.Equal(t, stored.Room, testRoom)
}

// The token the registration minted is what the person presents to the member
// plane, and it works there without a stream of its own.
func TestAdminService_RegisteredMemberChatsAsHuman(t *testing.T) {
	svc, id := newTestAdminService(t)
	member := NewService(svc.store, &fakeProvider{}, &fakeNotifier{}, nil, nil)
	attend(t, svc.store, "tok-a", testRoom, "alpha", "ana")

	res, err := svc.RegisterMember(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.RegisterMemberRequest{Room: testRoom, Team: "hosts", Name: "hana"})
	assert.NilError(t, err)

	sendAs(t, member, res.GetToken(), to("ana"), "standup in five")
	listed, err := member.ListMembers(callCtx(t, res.GetToken()),
		&chatv1.ListMembersRequest{})
	assert.NilError(t, err)
	assert.DeepEqual(t, addresses(listed.GetMembers()),
		[]string{"alpha/ana", "hosts/hana"})

	read, err := member.Read(callCtx(t, "tok-a"), &chatv1.ReadRequest{})
	assert.NilError(t, err)
	assert.DeepEqual(t, messageTexts(read.GetMessages()), []string{"standup in five"})
	assert.Equal(t, address(read.GetMessages()[0].GetFrom()), "hosts/hana")
}

func TestAdminService_RegisterMemberRefuses(t *testing.T) {
	svc, id := newTestAdminService(t)
	attend(t, svc.store, "tok-a", testRoom, "alpha", "ana")

	for _, tc := range []struct {
		name string
		req  *chatv1.RegisterMemberRequest
		want codes.Code
	}{
		{
			"no room",
			&chatv1.RegisterMemberRequest{Team: "hosts", Name: "hana"},
			codes.InvalidArgument,
		},
		{
			"no team",
			&chatv1.RegisterMemberRequest{Room: testRoom, Name: "hana"},
			codes.InvalidArgument,
		},
		{
			"a name with the team separator in it",
			&chatv1.RegisterMemberRequest{Room: testRoom, Team: "hosts", Name: "a/b"},
			codes.InvalidArgument,
		},
		{
			// The role has one read position, so the person would be reading
			// the agent's place in the room.
			"a role somebody is already attending under",
			&chatv1.RegisterMemberRequest{Room: testRoom, Team: "alpha", Name: "ana"},
			codes.AlreadyExists,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.RegisterMember(adminCtx(t, adminNonce(t, svc, id)), tc.req)
			assert.Equal(t, status.Code(err), tc.want)
		})
	}
}

// Registering is where a person enters the room, so it is the moment their
// arrival can be announced.
func TestAdminService_RegisterMemberPublishesTheArrival(t *testing.T) {
	svc, id := newTestAdminService(t)
	watcher := svc.store.events.subscribe(testRoom)
	defer svc.store.events.unsubscribe(watcher)

	_, err := svc.RegisterMember(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.RegisterMemberRequest{Room: testRoom, Team: "hosts", Name: "hana"})
	assert.NilError(t, err)

	assert.Equal(t, describeEvent(nextEvent(t, watcher.events)), "joined:hosts/hana")
}

func TestAdminService_DeleteRoom(t *testing.T) {
	svc, id := newTestAdminService(t)
	ana := attend(t, svc.store, "tok-a", testRoom, "alpha", "ana")
	send(t, svc.store, senderOf(ana), Target{Kind: TargetEveryone}, "one")
	send(t, svc.store, senderOf(ana), Target{Kind: TargetEveryone}, "two")

	// Somebody is still in the room, which contradicts the operator saying it
	// is done, and the deletion would leave that session addressing a room the
	// broker no longer has.
	_, err := svc.DeleteRoom(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.DeleteRoomRequest{Room: testRoom})
	assert.Equal(t, status.Code(err), codes.FailedPrecondition)

	_, err = svc.store.Detach(t.Context(), "tok-a")
	assert.NilError(t, err)

	res, err := svc.DeleteRoom(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.DeleteRoomRequest{Room: testRoom})
	assert.NilError(t, err)
	assert.Equal(t, res.GetDeletedMessages(), int64(2))

	// The room is gone with everything it held, so asking again finds nothing.
	assert.Equal(t, countRows(t, svc.store, `SELECT COUNT(*) FROM messages`), 0)
	assert.Equal(t, countRows(t, svc.store, `SELECT COUNT(*) FROM read_positions`), 0)
	_, err = svc.DeleteRoom(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.DeleteRoomRequest{Room: testRoom})
	assert.Equal(t, status.Code(err), codes.NotFound)

	_, err = svc.DeleteRoom(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.DeleteRoomRequest{})
	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}
