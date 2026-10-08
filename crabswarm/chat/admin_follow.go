package chat

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// Follow streams a named room the way [Service.Follow] streams the caller's
// own, without the caller attending it.
//
// The credential is spent once, as the stream opens: the stream is one call,
// however long it runs.
//
// mentioned_you is never set. An operator follows as nobody, and a message
// cannot be addressed to nobody.
//
// A room nothing has been said in follows from seq zero rather than failing
// with NotFound, and so does one that does not exist yet: an operator may well
// open the stream before the agents arrive.
func (a *AdminService) Follow(
	req *chatv1.AdminFollowRequest,
	stream grpc.ServerStreamingServer[chatv1.FollowEvent],
) error {
	ctx := stream.Context()
	if err := a.authenticate(ctx); err != nil {
		return err
	}
	if req.GetRoom() == "" {
		return status.Error(codes.InvalidArgument, "empty room")
	}
	return follow(ctx, a.store, stream, req.GetRoom(), req.Since, nil)
}
