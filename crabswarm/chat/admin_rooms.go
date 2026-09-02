package chat

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// adminName is the name and team an admin message is attributed to. No member
// may take it — [validateName] refuses it — so within a room the attribution
// cannot be forged.
const adminName = "admin"

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
//
// A name already carried in the target team is AlreadyExists, unless whoever
// carries it is gone — an operator meets the same rule a joiner does, so a
// vanished member does not block a move that a fresh join would walk straight
// through.
func (a *AdminService) MoveMember(
	ctx context.Context,
	req *chatv1.MoveMemberRequest,
) (*chatv1.MoveMemberResponse, error) {
	if err := a.authenticate(ctx); err != nil {
		return nil, err
	}
	moved, err := a.moveMember(ctx, req)
	if err != nil {
		return nil, storeStatus(err)
	}
	a.logger.InfoContext(ctx, "chat: admin moved member",
		"room", moved.Room, "member", req.GetTeam()+"/"+req.GetName(),
		"to_team", moved.Team)
	// A move stays inside the room, so what a watcher sees is not a member
	// coming or going but one changing address: the member it knew as
	// "team/name" is gone and a member of that name is now in another team.
	// Moving into the team the member already occupies changes no address and
	// is announced as nothing.
	if moved.Team != req.GetTeam() {
		a.store.events.publish(moved.Room, memberLeftEvent(Member{
			Name: req.GetName(), Team: req.GetTeam(), Room: moved.Room,
		}))
		a.store.events.publish(moved.Room, memberJoinedEvent(moved))
	}
	return &chatv1.MoveMemberResponse{Member: memberProto(moved)}, nil
}

// moveMember runs the move itself, freeing a colliding name in the target team
// first when the member holding it has vanished. The error is the store's own,
// for [AdminService.MoveMember] to map onto a status.
//
// Retried once and no further: a name taken again in between is somebody else
// winning the race, which is a refusal to report rather than a reason to keep
// trying.
func (a *AdminService) moveMember(
	ctx context.Context,
	req *chatv1.MoveMemberRequest,
) (Member, error) {
	moved, err := a.store.MoveMemberByName(
		ctx, req.GetRoom(), req.GetTeam(), req.GetName(), req.GetToTeam())
	if !errors.Is(err, ErrNameTaken) || a.provider == nil {
		return moved, err
	}
	// Nothing to forget on a reap: no verdict is cached on this half, which
	// asks the provider afresh every time it asks at all.
	if !reclaimName(ctx, a.store, a.provider, a.logger, nil,
		req.GetRoom(), req.GetToTeam(), req.GetName()) {
		return moved, err
	}
	return a.store.MoveMemberByName(
		ctx, req.GetRoom(), req.GetTeam(), req.GetName(), req.GetToTeam())
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
	// Registering is where a human enters the room: they are listed from here
	// on, and their later Join finds the membership already made, so this is
	// the only moment their arrival can be announced.
	a.store.events.publish(registered.Room, memberJoinedEvent(registered))
	return &chatv1.RegisterMemberResponse{
		Member: memberProto(registered),
		Token:  token,
	}, nil
}

// Send delivers a message into a room the caller does not attend, addressed to
// everyone there, to one of its teams or to one member, and reports how many
// inboxes it reached.
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
	// A oneof carries at most one case, so the only way to miss "exactly one"
	// is to set none, which is a request that never said who it is for.
	case req.GetTarget() == nil:
		return nil, status.Error(codes.InvalidArgument,
			"no target: address everyone, a team or a member")
	case req.GetText() == "":
		return nil, status.Error(codes.InvalidArgument, "empty message text")
	}
	recipients, err := a.deliverAdminMessage(ctx, req)
	if err != nil {
		return nil, storeStatus(err)
	}
	a.logger.InfoContext(ctx, "chat: admin sent message",
		"room", req.GetRoom(), "target", adminTargetAddr(req),
		"delivered", len(recipients))
	return &chatv1.AdminSendResponse{Delivered: int32(len(recipients))}, nil
}

// deliverAdminMessage puts the message in the addressed inboxes and returns who
// received it. The error is the store's own, for [AdminService.Send] to map
// onto a status.
//
// A member is addressed through the same resolution a member's own send goes
// through, from the operator's perspective: the operator is in no team, so a
// [chatv1.MemberTarget] carrying none resolves its name across the room and is
// refused as ambiguous where two teams carry it.
func (a *AdminService) deliverAdminMessage(
	ctx context.Context,
	req *chatv1.AdminSendRequest,
) ([]Member, error) {
	from := adminSender(req.GetRoom())
	switch t := req.GetTarget().(type) {
	case *chatv1.AdminSendRequest_Everyone:
		return a.deliver.broadcastAs(ctx, from, req.GetText(), time.Now())
	case *chatv1.AdminSendRequest_Team:
		return a.deliver.broadcastTeamAs(
			ctx, from, t.Team.GetTeam(), req.GetText(), time.Now())
	case *chatv1.AdminSendRequest_Member:
		recipient, err := a.deliver.sendAs(
			ctx, from, memberAddr(t.Member), req.GetText(), time.Now())
		if err != nil {
			return nil, err
		}
		return []Member{recipient}, nil
	default:
		// [AdminService.Send] turns down a request carrying no case, so reaching
		// here takes a case added to the schema and not taught here — a defect of
		// the daemon, which is what Internal is for.
		return nil, fmt.Errorf("delivering an admin message: unhandled target %T", t)
	}
}

// memberAddr renders a member target as the send address [resolveFor] takes: a
// bare name where the target names no team, "team/name" where it does.
func memberAddr(t *chatv1.MemberTarget) string {
	if t.GetTeam() == "" {
		return t.GetName()
	}
	return t.GetTeam() + "/" + t.GetName()
}

// adminTargetAddr renders a send target for the log line, in the grammar an
// operator writes one in: "*" for the room, "team/*" for a team, and the send
// address for a member. A case nobody taught it renders as its own type, which
// names the omission.
func adminTargetAddr(req *chatv1.AdminSendRequest) string {
	switch t := req.GetTarget().(type) {
	case *chatv1.AdminSendRequest_Everyone:
		return "*"
	case *chatv1.AdminSendRequest_Team:
		return t.Team.GetTeam() + "/*"
	case *chatv1.AdminSendRequest_Member:
		return memberAddr(t.Member)
	default:
		return fmt.Sprintf("%T", t)
	}
}

// History hands back a named room's conversation without the caller attending
// it, and consumes nothing, so the same stretch can be read again.
//
// It answers in one of two ways. With no cursor it takes the tail, the newest
// entries counted back by limit, which is what a reader opening a room wants.
// With one it reads forward instead, handing back what was said after the entry
// of that id, which is what a reader that already has the tail wants: it
// advances its cursor by the id of the last entry it got and asks again.
//
// A room nobody has spoken in — including one that has never existed — answers
// with no entries rather than NotFound, the same way the members' own read
// does: a room is what was said in it.
func (a *AdminService) History(
	ctx context.Context,
	req *chatv1.AdminHistoryRequest,
) (*chatv1.AdminHistoryResponse, error) {
	if err := a.authenticate(ctx); err != nil {
		return nil, err
	}
	if req.GetRoom() == "" {
		return nil, status.Error(codes.InvalidArgument, "empty room")
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = defaultHistoryWindow
	}
	// Asking for more than the room keeps is not an error: the answer to it is
	// everything there is, which is what the retention cap left. A store that
	// records nothing has no cap to clamp against — it answers with whatever it
	// held before it was switched off, and that is already a short list.
	if kept := a.store.historyLimit; kept > 0 && limit > kept {
		limit = kept
	}
	entries, err := a.readHistory(ctx, req.GetRoom(), req.GetSinceId(), limit)
	if err != nil {
		return nil, storeStatus(err)
	}
	out := make([]*chatv1.AdminHistoryEntry, len(entries))
	for i, e := range entries {
		entry := &chatv1.AdminHistoryEntry{
			Id:     e.ID,
			From:   senderProto(e.From),
			Text:   e.Text,
			SentAt: timestamppb.New(e.SentAt),
		}
		if e.To != nil {
			entry.To = senderProto(*e.To)
		}
		out[i] = entry
	}
	return &chatv1.AdminHistoryResponse{Entries: out}, nil
}

// readHistory picks the store read the request asked for: the tail when there
// is no cursor to read forward from. The error is the store's own, for
// [AdminService.History] to map onto a status.
func (a *AdminService) readHistory(
	ctx context.Context,
	room string,
	sinceID int64,
	limit int,
) ([]HistoryEntry, error) {
	if sinceID <= 0 {
		return a.store.History(ctx, room, limit)
	}
	return a.store.HistorySince(ctx, room, sinceID, limit)
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
