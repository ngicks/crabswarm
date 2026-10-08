package chat

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// Send appends a message to the caller's room, addressed to the roles it names,
// to everyone there, or to nobody.
//
// An unset target is a board post: it is in the room and is nobody's unread, so
// it interrupts no one. A roles target must name roles the room has — meaning
// roles that have attended it at least once — and one it has never had is
// NotFound, since there is nobody of that name to address. A bare name two
// teams of the room carry is InvalidArgument, which the caller answers by
// writing it as "team/name"; the error says which teams. Nothing is recorded
// when the target does not resolve.
func (s *Service) Send(
	ctx context.Context,
	req *chatv1.SendRequest,
) (*chatv1.SendResponse, error) {
	caller, err := s.caller(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetText() == "" {
		return nil, status.Error(codes.InvalidArgument, "empty message text")
	}
	sent, err := s.deliver.send(
		ctx, senderOf(caller), targetOf(req.GetTarget()), req.GetText(), time.Now())
	if err != nil {
		return nil, storeStatus(err)
	}
	return &chatv1.SendResponse{
		Mentioned: membersProto(sent.Mentioned),
		Absent:    membersProto(sent.Absent),
	}, nil
}

// Read hands back messages of the caller's room from the filter's cursor,
// oldest first, and reports how many unread mentions are left afterwards.
//
// It moves the caller's read position to the newest message it shows, whichever
// cursor was used: the position is one number per role, so a caller that read
// the tail has read past everything before it. A read of old history therefore
// never un-reads what came after.
//
// The cursor defaults to unread, which keeps only what the caller has not been
// shown. A range running against its cursor is InvalidArgument: nothing lies
// before the head or after the tail, and unread counts forward only.
//
// A non-zero skip shows nothing: it moves the read position through that seq
// and reports what is left, for a caller that already showed those messages
// some other way. A negative skip names no message, and a filter beside a skip
// asks for messages the skip does not show; both are InvalidArgument.
func (s *Service) Read(
	ctx context.Context,
	req *chatv1.ReadRequest,
) (*chatv1.ReadResponse, error) {
	caller, err := s.caller(ctx)
	if err != nil {
		return nil, err
	}
	if skip := req.GetSkip(); skip != 0 {
		// An empty filter narrows nothing, so it is accepted beside a skip: a
		// client that always allocates one is not asking for anything.
		if f := req.GetFilter(); f != nil && proto.Size(f) > 0 {
			return nil, status.Error(codes.InvalidArgument,
				"a skip shows no messages, so it takes no filter")
		}
		remaining, err := s.store.Skip(ctx, senderOf(caller), skip)
		if err != nil {
			return nil, storeStatus(err)
		}
		return &chatv1.ReadResponse{RemainingUnread: int32(remaining)}, nil
	}
	filter, err := readFilterOf(req.GetFilter())
	if err != nil {
		return nil, err
	}
	messages, remaining, err := s.store.Read(ctx, senderOf(caller), filter)
	if err != nil {
		return nil, storeStatus(err)
	}
	return &chatv1.ReadResponse{
		Messages:        messagesProto(messages),
		RemainingUnread: int32(remaining),
	}, nil
}

// CountUnread reports how many unread messages mention the caller, without
// showing any of them and without moving the caller's read position.
//
// It is the question a caller asks before it interrupts somebody: an agent's
// own MCP server, deciding whether a session that just reattended or just
// reported itself done has anything waiting. Reading to find out would hand the
// messages over, and the wake-up that followed would have nothing left to
// deliver.
func (s *Service) CountUnread(
	ctx context.Context,
	_ *chatv1.CountUnreadRequest,
) (*chatv1.CountUnreadResponse, error) {
	caller, err := s.caller(ctx)
	if err != nil {
		return nil, err
	}
	unread, err := s.store.CountUnread(ctx, senderOf(caller))
	if err != nil {
		return nil, storeStatus(err)
	}
	return &chatv1.CountUnreadResponse{UnreadMentions: int32(unread)}, nil
}
