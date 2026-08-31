package chat

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"

	"github.com/ngicks/crabswarm/crabswarm/chat/resolver"
)

// ProviderUnavailableMessage opens the message of the only Unavailable status a
// running daemon returns: the team-info provider could not be asked. gRPC
// itself reports Unavailable when nothing answers on the socket, and the two
// mean opposite things to whoever reads them, so the CLI tells them apart by
// this wording rather than by the code alone.
const ProviderUnavailableMessage = "looking up team information"

// Join declares attendance under the requested name, deriving room and team
// from the caller's token.
//
// A token the provider does not know is NotFound: it carries no team
// coordination information, so there is nowhere to put its holder. A provider
// lookup that merely fails is Unavailable instead — refusing a joiner because
// cmdman was busy would read as "you do not belong here", which is not what
// happened.
//
// Joining again with the same token returns the existing membership unchanged,
// name included, since the store keeps the first join. An admin-registered
// human may call Join too: they are already a member, and their token is theirs
// to present, so it is answered from the store without consulting the provider.
func (s *Service) Join(
	ctx context.Context,
	req *chatv1.JoinRequest,
) (*chatv1.JoinResponse, error) {
	token, err := tokenFromContext(ctx)
	if err != nil {
		return nil, err
	}

	switch existing, err := s.store.Member(ctx, token); {
	case err == nil:
		if !s.stillKnown(ctx, existing) {
			return nil, status.Errorf(codes.NotFound,
				"token is no longer known to the team-info provider")
		}
		// Re-declared attendance re-publishes the stored state: a session that
		// starts again is often one whose display was reset under it.
		s.mirrorState(ctx, existing, existing.State)
		return &chatv1.JoinResponse{Self: memberProto(existing)}, nil
	case !errors.Is(err, ErrNotFound):
		return nil, storeStatus(err)
	}

	info, err := s.provider.Resolve(ctx, token)
	switch {
	case errors.Is(err, resolver.ErrUnknownToken):
		return nil, status.Errorf(codes.NotFound,
			"no team information for this token: %s", err)
	case err != nil:
		return nil, status.Errorf(codes.Unavailable,
			"%s: %s", ProviderUnavailableMessage, err)
	}
	s.recordVerified(token)

	name := req.GetName()
	if name == "" {
		name = defaultName(token)
	}
	joined, err := s.store.Join(ctx, Member{
		Token: token,
		Name:  name,
		Team:  info.Team,
		Room:  info.Room,
		Kind:  KindAgent,
	})
	if err != nil {
		return nil, storeStatus(err)
	}
	s.mirrorState(ctx, joined, joined.State)
	// Only a first join is news: re-declared attendance returns above, having
	// changed nothing a watcher of the room can see.
	s.store.events.publish(joined.Room, memberJoinedEvent(joined))
	return &chatv1.JoinResponse{Self: memberProto(joined)}, nil
}

// ListMembers lists everyone attending the caller's room, teams included.
func (s *Service) ListMembers(
	ctx context.Context,
	_ *chatv1.ListMembersRequest,
) (*chatv1.ListMembersResponse, error) {
	caller, err := s.caller(ctx)
	if err != nil {
		return nil, err
	}
	members, err := s.store.ListMembers(ctx, caller.Room)
	if err != nil {
		return nil, storeStatus(err)
	}
	out := make([]*chatv1.Member, len(members))
	for i, m := range members {
		out[i] = memberProto(m)
	}
	return &chatv1.ListMembersResponse{Members: out}, nil
}

// Leave withdraws the caller's attendance, discarding whatever is still in
// their inbox.
func (s *Service) Leave(
	ctx context.Context,
	_ *chatv1.LeaveRequest,
) (*chatv1.LeaveResponse, error) {
	caller, err := s.caller(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.store.RemoveMember(ctx, caller.Token); err != nil {
		return nil, storeStatus(err)
	}
	s.forgetVerified(caller.Token)
	s.mirrorGone(ctx, caller)
	s.store.events.publish(caller.Room, memberLeftEvent(caller))
	return &chatv1.LeaveResponse{}, nil
}

// ReportState records the harness state the caller's hooks report.
func (s *Service) ReportState(
	ctx context.Context,
	req *chatv1.ReportStateRequest,
) (*chatv1.ReportStateResponse, error) {
	caller, err := s.caller(ctx)
	if err != nil {
		return nil, err
	}
	state, err := memberState(req.GetState())
	if err != nil {
		return nil, err
	}
	if err := s.store.SetState(ctx, caller.Token, state); err != nil {
		return nil, storeStatus(err)
	}
	s.mirrorState(ctx, caller, state)
	s.store.events.publish(caller.Room, memberStateChangedEvent(caller, req.GetState()))
	return &chatv1.ReportStateResponse{}, nil
}
