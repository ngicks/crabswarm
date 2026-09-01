package chat

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// WatchRoom streams what happens in the caller's room until the client stops
// listening.
//
// The stream carries what changes from the moment it opens; nothing before it
// is replayed. A watcher that wants the room as it stands lists the members
// after the stream is up, so no change can slip between the two.
//
// The caller's token is checked once, when the stream opens, and not again for
// its lifetime: an agent whose session ends takes the connection with it, and a
// stream still being read is being read by someone.
//
// A watcher that stops reading for [roomEventBuffer] events is dropped with
// ResourceExhausted rather than served the rest of the feed with holes in it.
// The answer to that is to list the room again and resubscribe: a feed missing
// a member-left leaves the client wrong with nothing to notice.
func (s *Service) WatchRoom(
	_ *chatv1.WatchRoomRequest,
	stream grpc.ServerStreamingServer[chatv1.RoomEvent],
) error {
	ctx := stream.Context()
	caller, err := s.caller(ctx)
	if err != nil {
		return err
	}

	sub := s.store.events.subscribe(caller.Room)
	defer s.store.events.unsubscribe(sub)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-sub.events:
			if !ok {
				return status.Errorf(codes.ResourceExhausted,
					"watcher fell more than %d events behind; "+
						"list the room and watch again", roomEventBuffer)
			}
			if err := stream.Send(ev); err != nil {
				return err
			}
		}
	}
}
