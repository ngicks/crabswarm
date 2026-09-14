package chat

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// History hands back a named room's conversation without the caller attending
// it. It moves no read position, so the same stretch reads the same way twice,
// and it marks nothing as mentioning the reader — an operator is not in the
// room to be mentioned.
//
// The cursor defaults to the room's tail, the newest ten counted backward,
// which is what a reader opening a room wants. The unread cursor is
// InvalidArgument: unread is measured from a role's read position, and this
// read has no role.
//
// A room nobody has spoken in — including one that has never existed — answers
// with no messages rather than NotFound: a room is what was said in it.
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
	filter, err := readFilterOf(req.GetFilter())
	if err != nil {
		return nil, err
	}
	messages, err := a.store.ReadRoom(ctx, req.GetRoom(), filter)
	if err != nil {
		return nil, storeStatus(err)
	}
	return &chatv1.AdminHistoryResponse{Messages: messagesProto(messages)}, nil
}
