package chat

import (
	"context"
	"crypto/rand"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// adminName is the name an admin message is attributed to. It carries no team,
// and nobody can attend a room without one, so an unteamed sender in a room is
// the host operator and nothing else can present itself as one.
const adminName = "admin"

// ListRooms reports every room the daemon knows, with everyone attending it.
//
// A room nobody is in is listed all the same: its conversation and its read
// positions outlive the sessions that made them, and an operator looking for a
// room to read or to delete is looking for exactly those.
func (a *AdminService) ListRooms(
	ctx context.Context,
	_ *chatv1.ListRoomsRequest,
) (*chatv1.ListRoomsResponse, error) {
	if err := a.authenticate(ctx); err != nil {
		return nil, err
	}
	rooms, err := a.store.ListRooms(ctx)
	if err != nil {
		return nil, storeStatus(err)
	}
	out := make([]*chatv1.Room, len(rooms))
	for i, r := range rooms {
		out[i] = roomProto(r)
	}
	return &chatv1.ListRoomsResponse{Rooms: out}, nil
}

// RegisterMember puts a person in a room and returns the token they present to
// ChatService from then on, which is the only time the daemon reveals it.
//
// The token is minted here rather than derived from anything, so no team-info
// provider can ever resolve it — which is the point: the member exists because
// the operator said so. Like the nonce it is [rand.Text], since the person
// retypes it into an env var.
//
// The attendance it opens is the one attendance no stream holds, and nothing
// closes it but a restart: a person reads their room from a terminal they open
// and close as they please, and the role would otherwise stop existing between
// two of their commands. A role somebody is already attending under is
// AlreadyExists, the same refusal a second stream meets.
func (a *AdminService) RegisterMember(
	ctx context.Context,
	req *chatv1.RegisterMemberRequest,
) (*chatv1.RegisterMemberResponse, error) {
	if err := a.authenticate(ctx); err != nil {
		return nil, err
	}
	// The store rejects an empty room too, but as a plain error — it has no
	// sentinel for it — which would reach the caller as Internal.
	if req.GetRoom() == "" {
		return nil, status.Error(codes.InvalidArgument, "empty room")
	}
	token := rand.Text()
	registered, err := a.store.Attend(ctx, Member{
		Token: token,
		Name:  req.GetName(),
		Team:  req.GetTeam(),
		Room:  req.GetRoom(),
		Kind:  KindHuman,
	})
	if err != nil {
		return nil, storeStatus(err)
	}
	a.logger.InfoContext(ctx, "chat: admin registered a person",
		"room", registered.Room, "member", registered.Team+"/"+registered.Name)
	a.store.events.publish(registered.Room, memberJoinedEvent(registered))
	return &chatv1.RegisterMemberResponse{
		Member: memberProto(registered),
		Token:  token,
	}, nil
}

// DeleteRoom removes a room and everything it holds — its messages, their
// mentions and its read positions — and reports how many messages went with it.
// A room the daemon does not know is NotFound.
//
// It is refused with FailedPrecondition while anybody attends the room: an
// operator asking for it means the room is done, which somebody still being in
// it contradicts, and the deletion would leave that session addressing a room
// the broker no longer has. Ending the attendance is the caller's move, not the
// daemon's, so the refusal stands until the sessions close.
func (a *AdminService) DeleteRoom(
	ctx context.Context,
	req *chatv1.DeleteRoomRequest,
) (*chatv1.DeleteRoomResponse, error) {
	if err := a.authenticate(ctx); err != nil {
		return nil, err
	}
	if req.GetRoom() == "" {
		return nil, status.Error(codes.InvalidArgument, "empty room")
	}
	deleted, err := a.store.DeleteRoom(ctx, req.GetRoom())
	if err != nil {
		return nil, storeStatus(err)
	}
	a.logger.InfoContext(ctx, "chat: admin deleted room",
		"room", req.GetRoom(), "messages", deleted)
	return &chatv1.DeleteRoomResponse{DeletedMessages: deleted}, nil
}

// adminSender is the identity an admin message carries into room. The team is
// left empty because the operator is in none, which is also what makes the
// attribution unforgeable: every attendee has a team, so no member can write
// under this identity.
func adminSender(room string) Sender {
	return Sender{Name: adminName, Room: room}
}

// roomProto renders a room and its attendance. Every member states its own
// team, so the flat list the schema carries loses no grouping.
func roomProto(r Room) *chatv1.Room {
	return &chatv1.Room{Name: r.Name, Members: membersProto(r.Members)}
}
