package chat

import (
	"context"
	"crypto/rand"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// adminName is the name and team an admin message is attributed to. No member
// may take it — [validateName] refuses it — so within a room the attribution
// cannot be forged.
const adminName = "admin"

// adminEveryone is the [AdminService.Send] target that addresses every member
// of the room instead of one. It shadows the bare name of a member that took
// "*" as its own, which is reachable as "team/*" — a name nobody is expected to
// pick, and not worth reserving for.
const adminEveryone = "*"

// ListRooms reports every room the daemon knows, with everyone attending it.
func (a *AdminService) ListRooms(
	ctx context.Context,
	req *chatv1.ListRoomsRequest,
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

// MoveMember moves the named member into another team of the same room. The
// member is addressed by room/team/name rather than by token: an operator reads
// names out of ListRooms and never sees a participant's token.
func (a *AdminService) MoveMember(
	ctx context.Context,
	req *chatv1.MoveMemberRequest,
) (*chatv1.MoveMemberResponse, error) {
	if err := a.authenticate(ctx); err != nil {
		return nil, err
	}
	moved, err := a.store.MoveMemberByName(
		ctx, req.GetRoom(), req.GetTeam(), req.GetName(), req.GetToTeam())
	if err != nil {
		return nil, storeStatus(err)
	}
	a.logger.InfoContext(ctx, "chat: admin moved member",
		"room", moved.Room, "member", req.GetTeam()+"/"+req.GetName(),
		"to_team", moved.Team)
	return &chatv1.MoveMemberResponse{Member: memberProto(moved)}, nil
}

// RegisterMember registers a human and returns the token they present to
// ChatService from then on, which is the only time the daemon reveals it.
//
// The token is minted here rather than derived from anything, so no team-info
// provider can ever resolve it — which is the point: the member exists because
// the operator said so, and the lazy reaper leaves a human alone for the same
// reason. Like the nonce it is [rand.Text], since the human retypes it into an
// env var.
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
	registered, err := a.store.Join(ctx, Member{
		Token: token,
		Name:  req.GetName(),
		Team:  req.GetTeam(),
		Room:  req.GetRoom(),
		Kind:  KindHuman,
	})
	if err != nil {
		return nil, storeStatus(err)
	}
	a.logger.InfoContext(ctx, "chat: admin registered human member",
		"room", registered.Room, "member", registered.Team+"/"+registered.Name)
	return &chatv1.RegisterMemberResponse{
		Member: memberProto(registered),
		Token:  token,
	}, nil
}

// Send delivers a message into a room the caller does not attend, addressed to
// one member or — as "*" — to everyone there, and reports how many inboxes it
// reached.
//
// The message is attributed to the reserved [adminName] identity, and no member
// is created for it: the operator speaks into the room without joining it, so
// there is nothing there to address back, to nudge, or to reap.
func (a *AdminService) Send(
	ctx context.Context,
	req *chatv1.AdminSendRequest,
) (*chatv1.AdminSendResponse, error) {
	if err := a.authenticate(ctx); err != nil {
		return nil, err
	}
	switch {
	case req.GetRoom() == "":
		return nil, status.Error(codes.InvalidArgument, "empty room")
	case req.GetTarget() == "":
		return nil, status.Error(codes.InvalidArgument, "empty target")
	case req.GetText() == "":
		return nil, status.Error(codes.InvalidArgument, "empty message text")
	}
	recipients, err := a.deliverAdminMessage(ctx, req)
	if err != nil {
		return nil, storeStatus(err)
	}
	a.logger.InfoContext(ctx, "chat: admin sent message",
		"room", req.GetRoom(), "target", req.GetTarget(),
		"delivered", len(recipients))
	return &chatv1.AdminSendResponse{Delivered: int32(len(recipients))}, nil
}

// deliverAdminMessage puts the message in the addressed inboxes and returns who
// received it. The error is the store's own, for [AdminService.Send] to map
// onto a status.
func (a *AdminService) deliverAdminMessage(
	ctx context.Context,
	req *chatv1.AdminSendRequest,
) ([]Member, error) {
	from := adminSender(req.GetRoom())
	if req.GetTarget() == adminEveryone {
		return a.deliver.broadcastAs(ctx, from, req.GetText(), time.Now())
	}
	recipient, err := a.deliver.sendAs(ctx, from, req.GetTarget(), req.GetText(), time.Now())
	if err != nil {
		return nil, err
	}
	return []Member{recipient}, nil
}

// adminSender is the identity an admin message carries into room. The name is
// reserved — [validateName] refuses it to a member — so a recipient reading it
// knows the message came from the host and not from a peer that named itself
// well. The team repeats the name rather than being left empty, so an address
// renders as "admin/admin" wherever a message shows "team/name".
func adminSender(room string) Sender {
	return Sender{Name: adminName, Team: adminName, Room: room}
}

// roomProto flattens a room's teams into the flat member list the schema
// carries: every member states its own team, so the grouping is already there.
func roomProto(r Room) *chatv1.Room {
	var members []*chatv1.Member
	for _, t := range r.Teams {
		for _, m := range t.Members {
			members = append(members, memberProto(m))
		}
	}
	return &chatv1.Room{Name: r.Name, Members: members}
}
