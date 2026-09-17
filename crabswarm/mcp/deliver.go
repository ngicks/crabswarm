package mcp

import (
	"context"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/nudge"
	"github.com/ngicks/crabswarm/crabswarm/mcp/harness"
)

// Delivering a mention is the other half of attending, for the members whose
// harness has a channel of its own. The daemon types at a terminal member and
// types nothing at a native one, so for a native one the waking happens here,
// off the feed the attendance already carries: nothing is polled, and the
// server hears about a mention at the same moment the room does.
//
// What is delivered is a notice, never the message. The text is
// sender-controlled and the agent reads it through the chat tools, where the
// read moves its position; a notice pushed into its context would hand it the
// words and leave the room believing they were still unread.

// observe reacts to one event of the room on this member's behalf: it follows
// what this member's harness reports about itself, and delivers the mentions
// nothing else will.
func (s *Server) observe(ctx context.Context, ev *chatv1.RoomEvent) {
	switch e := ev.GetEvent().(type) {
	case *chatv1.RoomEvent_MemberStateChanged:
		changed := e.MemberStateChanged
		if !s.isSelf(changed.GetMember()) {
			return
		}
		if s.stateReported(changed.GetState()) {
			s.deliverWaiting(ctx)
		}
	case *chatv1.RoomEvent_MessageAppended:
		msg := e.MessageAppended.GetMessage()
		if s.mentionsSelf(msg) {
			s.deliverArrival(ctx, msg.GetFrom())
		}
	}
}

// deliverArrival hands over the notice for a mention that has just arrived, or
// leaves it outstanding for when the agent can be interrupted.
func (s *Server) deliverArrival(ctx context.Context, from *chatv1.Member) {
	if !s.deliversNatively() || !s.claimArrival() {
		return
	}
	addr := nudge.Sanitize(nudge.Address(from.GetTeam(), from.GetName()))
	if !s.deliver(ctx, harness.Notice{From: addr, Text: nudge.NewMessage(addr)}) {
		s.remember()
	}
}

// deliverWaiting says how much is waiting, which is what the mentions that
// arrived while the agent was mid-turn are delivered as. It counts them rather
// than replaying them one by one: several notices in a row would each name one
// sender and the last would be as much of a surprise as the first, where one
// line says the same thing once.
//
// Counting is not reading — the read position stays where it is, so the agent
// still finds every mention waiting for it in the room.
func (s *Server) deliverWaiting(ctx context.Context) {
	if !s.deliversNatively() || !s.claimWaiting() {
		return
	}
	token, err := s.ResolveToken()
	if err != nil {
		s.remember()
		return
	}
	waiting, err := s.client.CountUnread(ctx, token)
	if err != nil {
		s.logger.Warn("counting the unread chat mentions failed", "error", err)
		s.remember()
		return
	}
	if waiting == 0 {
		return
	}
	if !s.deliver(ctx, harness.Notice{Text: nudge.Waiting(waiting)}) {
		s.remember()
	}
}

// deliver hands one notice to the harness, reporting whether it took it.
//
// A channel that refused is reported and left alone. The message is in the room
// either way, so a refusal costs the agent a late read rather than the message,
// and retrying on the spot would hold up the feed for a channel that is
// unlikely to take the same notice a moment later. The next report ending a
// turn and the next attendance both try again, which is retry enough.
func (s *Server) deliver(ctx context.Context, n harness.Notice) bool {
	// The room is filled in here rather than by the callers: every notice
	// belongs to the one room this member attends, and it is only known once the
	// attendance has landed — the harness was chosen before that, off the
	// handshake. Sanitized like the sender for the same reason, since a name is
	// whatever the attending agent asked to be called.
	n.Room = nudge.Sanitize(s.selfMember().GetRoom())
	if err := s.harness.Deliver(ctx, n); err != nil {
		s.logger.Warn("delivering a chat notice through the harness failed",
			"from", n.From, "error", err)
		return false
	}
	s.logger.Info("delivered a chat notice through the harness", "from", n.From)
	return true
}

// deliversNatively reports whether this member's mentions are the server's to
// deliver. They are not before the handshake has landed: nothing has said what
// the harness is, and there is no attendance yet either.
func (s *Server) deliversNatively() bool {
	return s.harness != nil &&
		s.harness.Nudge() == chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE
}

// claimArrival reports whether a mention that just arrived may be delivered
// now.
//
// It may not while the agent is working or waiting on a dialog, however long
// ago it said so: the report is the only thing that speaks for the harness, and
// a notice pushed into a turn in progress is an interruption the agent did not
// ask for. It waits for the report that ends the turn.
func (s *Server) claimArrival() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != chatv1.HarnessState_HARNESS_STATE_DONE {
		s.pending = true
		return false
	}
	return true
}

// claimWaiting takes what is outstanding for delivery, reporting whether this
// is the moment to deliver it. It is not while the agent is mid-turn, and what
// was outstanding stays so for the report that ends the turn.
func (s *Server) claimWaiting() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != chatv1.HarnessState_HARNESS_STATE_DONE {
		s.pending = true
		return false
	}
	s.pending = false
	return true
}

// remember leaves the mentions outstanding, so the next report that ends a turn
// and the next attendance both come back to them.
func (s *Server) remember() {
	s.mu.Lock()
	s.pending = true
	s.mu.Unlock()
}

// stateReported records what this member's harness now says about itself and
// reports whether the mentions it could not be interrupted for are due.
func (s *Server) stateReported(state chatv1.HarnessState) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
	return s.pending && state == chatv1.HarnessState_HARNESS_STATE_DONE
}

// isSelf reports whether m is the member this server attends as. A member is
// named by its room, team and name; the token it holds is its own business and
// no event carries one.
func (s *Server) isSelf(m *chatv1.Member) bool {
	self := s.selfMember()
	return self != nil &&
		m.GetRoom() == self.GetRoom() &&
		m.GetTeam() == self.GetTeam() &&
		m.GetName() == self.GetName()
}

// mentionsSelf reports whether msg is one this member owes an answer to: it
// names this member's role or the whole room, and somebody else wrote it.
//
// The room is not compared here. The feed is this member's own room and carries
// nothing else, and a target names a role within it rather than across rooms.
func (s *Server) mentionsSelf(msg *chatv1.Message) bool {
	self := s.selfMember()
	if self == nil {
		return false
	}
	from := msg.GetFrom()
	if from.GetTeam() == self.GetTeam() && from.GetName() == self.GetName() {
		// Its own message is never unread for it, so the notice would buy it an
		// empty read.
		return false
	}
	switch target := msg.GetTarget().GetTarget().(type) {
	case *chatv1.Target_Everyone:
		return true
	case *chatv1.Target_Roles:
		for _, role := range target.Roles.GetRoles() {
			if role.GetTeam() == self.GetTeam() && role.GetName() == self.GetName() {
				return true
			}
		}
		return false
	default:
		// A board post is addressed to nobody: it sits in the room for whoever
		// comes looking.
		return false
	}
}

// selfMember is the member the attendance named, or nil before one landed.
func (s *Server) selfMember() *chatv1.Member {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.self
}
