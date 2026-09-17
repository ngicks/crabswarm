package chat

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"
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

// Attend declares attendance and holds it open for as long as the stream is,
// deriving room and team from the caller's token. Closing the stream is
// leaving: the attendance is the stream, so there is no second call to make and
// nothing to clean up after a client that vanished.
//
// What attends, the request says itself: one declaring an agent attends as
// [KindAgent] and has its terminal typed into when it is mentioned, one
// declaring a human attends read-only, and one declaring nothing is refused
// with InvalidArgument. The daemon does not guess — a harness and the shell a
// person types in are the same kind of command to the team-info provider, and
// guessing wrong means keystrokes in somebody's shell.
//
// The request also says which harness the attendee runs and how a mention
// should reach it, and both stand for the life of the attendance. Neither is
// required: a harness nothing named stays unspecified, and an agent that named
// no delivery is typed at through its terminal, which is what every agent was
// before a harness could deliver a mention itself. An agent declaring native
// delivery is never typed at — its own server is watching the same feed.
//
// A token the provider does not know is Unauthenticated: nothing places its
// holder, so there is nowhere to put them and no reason to believe the token.
// A provider lookup that merely fails is Unavailable instead — turning a caller
// away because cmdman was busy would read as "you do not belong here", which is
// not what happened.
//
// A token already attending is AlreadyExists, and so is a second session for a
// role somebody is already attending under: the role has one read position, and
// two streams sharing it would each hide messages from the other.
//
// The first event is Attended, carrying the member the token resolved to. The
// rest is the room's feed from the moment the stream opened; nothing said
// before it is replayed, so a client that wants the room as it stands reads it
// once the stream is up.
func (s *Service) Attend(
	req *chatv1.AttendRequest,
	stream grpc.ServerStreamingServer[chatv1.RoomEvent],
) error {
	ctx := stream.Context()
	token, err := tokenFromContext(ctx)
	if err != nil {
		return err
	}
	kind, err := memberKind(req.GetKind())
	if err != nil {
		return err
	}
	harness, err := harnessOf(req.GetHarness())
	if err != nil {
		return err
	}
	nudge, err := nudgeOf(req.GetNudge())
	if err != nil {
		return err
	}
	info, err := s.provider.Resolve(ctx, token)
	switch {
	case errors.Is(err, resolver.ErrUnknownToken):
		return status.Errorf(codes.Unauthenticated,
			"no team information for this token: %s", err)
	case err != nil:
		return status.Errorf(codes.Unavailable,
			"%s: %s", ProviderUnavailableMessage, err)
	}

	// Subscribed before the attendance lands rather than after: an event
	// published in the gap between the two would reach this stream's client
	// never, and the stream is the only thing telling it what happens while it
	// is here. Its own arrival comes back down the feed, which is right — every
	// attendee sees the same room.
	sub := s.store.events.subscribe(info.Room)
	defer s.store.events.unsubscribe(sub)

	self, err := s.store.Attend(ctx, Member{
		Token:   token,
		Name:    s.attendName(req.GetName(), info, token, kind),
		Team:    info.Team,
		Room:    info.Room,
		Kind:    kind,
		Harness: harness,
		Nudge:   nudge,
	})
	if err != nil {
		return storeStatus(err)
	}
	defer s.detach(ctx, self)

	s.mirrorState(ctx, self, self.State)
	s.store.events.publish(self.Room, memberJoinedEvent(self))
	if err := stream.Send(attendedEvent(self)); err != nil {
		return err
	}
	return forwardRoomEvents(ctx, stream, sub)
}

// attendName picks what to call the attendee: what it asked to be called, else
// what the provider derives from its command, else its kind and token.
func (s *Service) attendName(
	requested string,
	info resolver.TeamInfo,
	token string,
	kind MemberKind,
) string {
	switch {
	case requested != "":
		return requested
	case info.Name != "":
		return info.Name
	default:
		return defaultName(token, kind)
	}
}

// detach withdraws the attendance a stream declared, which is what the stream
// ending means however it ended. The room hears about it: the other attendees
// are sessions that are still running, and they would otherwise keep a member
// that is gone on their list forever.
func (s *Service) detach(ctx context.Context, m Member) {
	if _, err := s.store.Detach(ctx, m.Token); err != nil {
		s.logger.Warn("chat: withdrawing attendance failed",
			"member", m.Team+"/"+m.Name, "err", err)
		return
	}
	s.mirrorGone(ctx, m)
	s.store.events.publish(m.Room, memberLeftEvent(m))
}

// forwardRoomEvents writes room's feed to the stream until the client stops
// listening.
//
// An attendee that stops reading for [roomEventBuffer] events is dropped with
// ResourceExhausted rather than served the rest of the feed with holes in it,
// and since the feed is the attendance it stops attending with it. The answer
// is to attend again: a feed missing a member-left leaves the client wrong with
// nothing to notice.
func forwardRoomEvents(
	ctx context.Context,
	stream grpc.ServerStreamingServer[chatv1.RoomEvent],
	sub *roomSubscription,
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-sub.events:
			if !ok {
				return status.Errorf(codes.ResourceExhausted,
					"attendee fell more than %d events behind; attend again",
					roomEventBuffer)
			}
			if err := stream.Send(ev); err != nil {
				return err
			}
		}
	}
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
	return &chatv1.ListMembersResponse{Members: membersProto(members)}, nil
}

// ReportState records the harness state the caller's hooks report.
//
// The report is always stored, even when it repeats the state already held: it
// carries the moment the harness was last seen in that state, which is what
// tells a member still working from one that stopped saying so.
//
// The room only hears about it when the state actually changed. Hooks report
// working after every tool call, so a room of busy agents would otherwise spend
// its event feed telling every attendee to re-read a roster that says exactly
// what it said before.
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
	if err := s.recordState(ctx, caller, state); err != nil {
		return nil, storeStatus(err)
	}
	return &chatv1.ReportStateResponse{}, nil
}

// RecordState records the state a watcher observed for the member attending
// under token, through the very path [Service.ReportState] records a hook's
// report through — so a reading of the terminal overrides a report that never
// came, and an operator's status display and the room's feed say the same thing
// either way.
//
// It is the seam for a watcher outside the RPC surface: the screen poller,
// which reads the state off the terminal because an interrupted Claude Code
// turn fires no hook. An unknown token is [ErrNotAttending] — the session ended
// between the listing and the reading, which is ordinary.
func (s *Service) RecordState(ctx context.Context, token string, state MemberState) error {
	m, err := s.store.Member(ctx, token)
	if err != nil {
		return err
	}
	return s.recordState(ctx, m, state)
}

// recordState is the one path a member's state is written through: the store
// first, then the status display, then the room — and the room only when the
// state actually changed. Hooks report working after every tool call, so a room
// of busy agents would otherwise spend its event feed telling every attendee to
// re-read a roster that says exactly what it said before.
//
// m is the member as it stood before the write, which is what the change is
// measured against and what the event carries.
func (s *Service) recordState(ctx context.Context, m Member, state MemberState) error {
	if err := s.store.SetState(ctx, m.Token, state, time.Now()); err != nil {
		return err
	}
	s.mirrorState(ctx, m, state)
	if state != m.State {
		s.store.events.publish(m.Room, memberStateChangedEvent(m, harnessStateProto(state)))
	}
	return nil
}
