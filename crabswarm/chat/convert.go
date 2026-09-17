package chat

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// This file is the whole of the translation between the wire schema and the
// store: both halves of the broker speak the same two vocabularies, and keeping
// the mapping in one place is what stops them drifting apart.

// memberState maps the reported harness state onto the stored one. The
// unspecified state is rejected rather than defaulted: a hook that failed to
// fill it in must not silently mark its agent done, which is the one state a
// keystroke nudge is sent to on sight rather than only once the report has
// gone stale.
func memberState(state chatv1.HarnessState) (MemberState, error) {
	switch state {
	case chatv1.HarnessState_HARNESS_STATE_DONE:
		return StateDone, nil
	case chatv1.HarnessState_HARNESS_STATE_WORKING:
		return StateWorking, nil
	case chatv1.HarnessState_HARNESS_STATE_WAITING:
		return StateWaiting, nil
	default:
		return "", status.Errorf(codes.InvalidArgument,
			"unknown harness state %q", state)
	}
}

// memberKind maps the declared kind onto the stored one. The unspecified kind
// is rejected rather than defaulted: a member taken for a human is never
// mirrored to the status display and never nudged, so a request that filled in
// nothing would settle the question wrongly for the life of the session.
//
// The refusal names the field of the request rather than a flag of any client.
// The CLI is one caller among several — the MCP bridge is another — and it says
// which flag to add itself, in words a person typing it can act on.
func memberKind(kind chatv1.MemberKind) (MemberKind, error) {
	switch kind {
	case chatv1.MemberKind_MEMBER_KIND_AGENT:
		return KindAgent, nil
	case chatv1.MemberKind_MEMBER_KIND_HUMAN:
		return KindHuman, nil
	default:
		return "", status.Error(codes.InvalidArgument,
			"the request declares no member kind")
	}
}

// harnessOf maps the declared harness onto the stored one. Unlike the kind, the
// unspecified value is taken as written: nothing the daemon does turns on which
// CLI a member runs, and a server whose handshake named no client it recognises
// has nothing truthful to put there. A value outside the enum is still refused,
// since that is a client speaking a schema this daemon does not have.
func harnessOf(h chatv1.Harness) (Harness, error) {
	switch h {
	case chatv1.Harness_HARNESS_UNSPECIFIED:
		return "", nil
	case chatv1.Harness_HARNESS_CLAUDE_CODE:
		return HarnessClaudeCode, nil
	case chatv1.Harness_HARNESS_CODEX:
		return HarnessCodex, nil
	case chatv1.Harness_HARNESS_OPENCODE:
		return HarnessOpenCode, nil
	case chatv1.Harness_HARNESS_OTHER:
		return HarnessOther, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "unknown harness %q", h)
	}
}

// nudgeOf maps the declared delivery onto the stored one. The unspecified value
// is left empty rather than filled in here: what an agent that named no
// delivery gets is the store's business, since the same default has to hold for
// a member the daemon builds itself.
func nudgeOf(n chatv1.NudgeDelivery) (NudgeDelivery, error) {
	switch n {
	case chatv1.NudgeDelivery_NUDGE_DELIVERY_UNSPECIFIED:
		return "", nil
	case chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL:
		return NudgeTerminal, nil
	case chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE:
		return NudgeNative, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "unknown nudge delivery %q", n)
	}
}

// harnessStateProto maps the stored state back onto the wire enum. Unlike
// [memberState] it refuses nothing: a [Member] carrying no state is one nobody
// recorded a state for, which is a member to describe rather than a request to
// turn down.
func harnessStateProto(state MemberState) chatv1.HarnessState {
	switch state {
	case StateWorking:
		return chatv1.HarnessState_HARNESS_STATE_WORKING
	case StateWaiting:
		return chatv1.HarnessState_HARNESS_STATE_WAITING
	case StateDone:
		return chatv1.HarnessState_HARNESS_STATE_DONE
	default:
		return chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED
	}
}

// memberKindProto maps the stored kind back onto the wire enum. Unlike
// [memberKind] it refuses nothing: every attending member carries a kind, and a
// caller reading one that does not — a role nobody attends under, a sender
// snapshot — is better served by the unspecified value than by an error about a
// member it only asked to see.
func memberKindProto(kind MemberKind) chatv1.MemberKind {
	switch kind {
	case KindAgent:
		return chatv1.MemberKind_MEMBER_KIND_AGENT
	case KindHuman:
		return chatv1.MemberKind_MEMBER_KIND_HUMAN
	default:
		return chatv1.MemberKind_MEMBER_KIND_UNSPECIFIED
	}
}

// harnessProto maps the stored harness back onto the wire enum. Like
// [memberKindProto] it refuses nothing: a member carrying no harness is one
// nobody named a harness for, which every human and every sender snapshot is.
func harnessProto(h Harness) chatv1.Harness {
	switch h {
	case HarnessClaudeCode:
		return chatv1.Harness_HARNESS_CLAUDE_CODE
	case HarnessCodex:
		return chatv1.Harness_HARNESS_CODEX
	case HarnessOpenCode:
		return chatv1.Harness_HARNESS_OPENCODE
	case HarnessOther:
		return chatv1.Harness_HARNESS_OTHER
	default:
		return chatv1.Harness_HARNESS_UNSPECIFIED
	}
}

// nudgeProto maps the stored delivery back onto the wire enum. The unspecified
// value is a member nothing is ever delivered to by itself, which is what a
// human and a sender snapshot both are.
func nudgeProto(n NudgeDelivery) chatv1.NudgeDelivery {
	switch n {
	case NudgeTerminal:
		return chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL
	case NudgeNative:
		return chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE
	default:
		return chatv1.NudgeDelivery_NUDGE_DELIVERY_UNSPECIFIED
	}
}

func memberProto(m Member) *chatv1.Member {
	return &chatv1.Member{
		Name:    m.Name,
		Team:    m.Team,
		Room:    m.Room,
		State:   harnessStateProto(m.State),
		Kind:    memberKindProto(m.Kind),
		Harness: harnessProto(m.Harness),
		Nudge:   nudgeProto(m.Nudge),
	}
}

func membersProto(members []Member) []*chatv1.Member {
	out := make([]*chatv1.Member, len(members))
	for i, m := range members {
		out[i] = memberProto(m)
	}
	return out
}

// senderProto renders a role rather than an attendance: a sender is who spoke
// as of send time, so it carries no state and no kind.
func senderProto(s Sender) *chatv1.Member {
	return &chatv1.Member{Name: s.Name, Team: s.Team, Room: s.Room}
}

// targetOf maps a written target onto the store's. An absent target is a board
// post: the schema answers "who is this for" by leaving the field out rather
// than by a case naming nobody.
func targetOf(t *chatv1.Target) Target {
	switch tt := t.GetTarget().(type) {
	case *chatv1.Target_Everyone:
		return Target{Kind: TargetEveryone}
	case *chatv1.Target_Roles:
		written := tt.Roles.GetRoles()
		roles := make([]Sender, len(written))
		for i, r := range written {
			// No room: a target is resolved within the sender's room, and one
			// that could name another room would address a room the sender is
			// not in.
			roles[i] = Sender{Name: r.GetName(), Team: r.GetTeam()}
		}
		return Target{Kind: TargetRoles, Roles: roles}
	default:
		return Target{Kind: TargetNone}
	}
}

// targetProto maps a written target back onto the wire. A board post carries no
// target at all, which is how the schema spells one.
func targetProto(t Target) *chatv1.Target {
	switch t.Kind {
	case TargetEveryone:
		return &chatv1.Target{
			Target: &chatv1.Target_Everyone{Everyone: &chatv1.Everyone{}},
		}
	case TargetRoles:
		roles := make([]*chatv1.MemberTarget, len(t.Roles))
		for i, r := range t.Roles {
			roles[i] = &chatv1.MemberTarget{Team: r.Team, Name: r.Name}
		}
		return &chatv1.Target{
			Target: &chatv1.Target_Roles{Roles: &chatv1.Roles{Roles: roles}},
		}
	default:
		return nil
	}
}

func messageProto(m Message) *chatv1.Message {
	return &chatv1.Message{
		Id:           m.Id,
		Seq:          m.Seq,
		From:         senderProto(m.From),
		Target:       targetProto(m.Target),
		Text:         m.Text,
		SentAt:       timestamppb.New(m.SentAt),
		MentionedYou: m.MentionedYou,
	}
}

func messagesProto(messages []Message) []*chatv1.Message {
	out := make([]*chatv1.Message, len(messages))
	for i, m := range messages {
		out[i] = messageProto(m)
	}
	return out
}

// readFilterOf maps the one read shape onto the store's.
//
// An unspecified cursor is left empty rather than filled in here, so the store
// applies the default belonging to the read being made: past the role's
// position for a member, the room's tail for an operator who has no position.
func readFilterOf(f *chatv1.ReadFilter) (ReadFilter, error) {
	out := ReadFilter{
		Range: int(f.GetRange()),
		Since: f.GetSince(),
		Until: f.GetUntil(),
	}
	switch f.GetCursor() {
	case chatv1.ReadCursor_READ_CURSOR_UNSPECIFIED:
	case chatv1.ReadCursor_READ_CURSOR_UNREAD:
		out.Cursor = CursorUnread
	case chatv1.ReadCursor_READ_CURSOR_HEAD:
		out.Cursor = CursorHead
	case chatv1.ReadCursor_READ_CURSOR_TAIL:
		out.Cursor = CursorTail
	default:
		return ReadFilter{}, status.Errorf(codes.InvalidArgument,
			"unknown read cursor %q", f.GetCursor())
	}
	if to := f.GetTo(); to != nil {
		target := targetOf(to)
		out.To = &target
	}
	return out, nil
}

// storeStatus maps a store error onto the status code its sentinel means. The
// store's message is kept: it names the role, the room and the colliding teams,
// which is exactly what the caller has to act on.
func storeStatus(err error) error {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrUnknownRole):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, ErrAmbiguousRole):
		// InvalidArgument rather than FailedPrecondition: nothing about the room
		// has to change for the call to go through. The caller writes the bare
		// name as "team/name" — the error says which teams carry it — and sends
		// the same request again.
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrInvalidArgument), errors.Is(err, ErrInvalidName):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrAlreadyAttending):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, ErrRoomAttended):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, ErrNotAttending):
		// Every call that can raise it asks about the caller's own token, so it
		// reports a session that is gone rather than something the caller
		// addressed being missing.
		return status.Error(codes.Unauthenticated, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
