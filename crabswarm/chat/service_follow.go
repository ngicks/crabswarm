package chat

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"

	"github.com/ngicks/crabswarm/crabswarm/chat/resolver"
)

// Follow streams the caller's room as a conversation: Followed first, then the
// messages past since when since is set, then each message as it is appended,
// every one of them once and in seq order. A client that lost its stream
// follows again from the last seq it saw and misses nothing.
//
// Following is not attending. It puts nobody in the room, announces nothing and
// moves no read position, so a program watching the room beside an agent takes
// nothing the agent has yet to read. It needs no attendance either: a client
// following again after a daemon restart may well be back before the agent's
// own server has attended again.
//
// mentioned_you is measured for the role the token stands for, regardless of
// that role's read position.
func (s *Service) Follow(
	req *chatv1.FollowRequest,
	stream grpc.ServerStreamingServer[chatv1.FollowEvent],
) error {
	ctx := stream.Context()
	viewer, err := s.followerRole(ctx)
	if err != nil {
		return err
	}
	return follow(ctx, s.store, stream, viewer.Room, req.GetSince(), &viewer)
}

// followerRole is the role the request's token stands for.
//
// A token attending right now stands for the role it attends under, which is
// the one certain answer: the attendance may have asked for its own name, and a
// person the operator registered holds a token no provider can place.
//
// Otherwise the team-info provider places the token the way [Service.Attend]
// does and with the same refusals, and the name is the one an attendance under
// the token would get. The agent's own MCP server is what attends under a token
// the provider places, and it asks for no name and attends as an agent.
func (s *Service) followerRole(ctx context.Context) (Sender, error) {
	token, err := tokenFromContext(ctx)
	if err != nil {
		return Sender{}, err
	}
	if m, err := s.store.Member(ctx, token); err == nil {
		return senderOf(m), nil
	}
	info, err := s.provider.Resolve(ctx, token)
	switch {
	case errors.Is(err, resolver.ErrUnknownToken):
		return Sender{}, status.Errorf(codes.Unauthenticated,
			"no team information for this token: %s", err)
	case err != nil:
		return Sender{}, status.Errorf(codes.Unavailable,
			"%s: %s", ProviderUnavailableMessage, err)
	}
	return Sender{
		Name: s.attendName("", info, token, KindAgent),
		Team: info.Team,
		Room: info.Room,
	}, nil
}
