package chat

import (
	"context"
	"crypto/rand"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

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
