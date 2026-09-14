package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ngicks/crabswarm/crabswarm/chat/internal/db"
)

// roleKey is a role within one room: the key a mention, a read position and a
// target are all stored under. The room is the context it is used in, so it is
// not part of the key.
type roleKey struct {
	Team string
	Name string
}

func (r roleKey) String() string { return r.Team + "/" + r.Name }

// sender turns the key back into the role it names, within room.
func (r roleKey) sender(room string) Sender {
	return Sender{Name: r.Name, Team: r.Team, Room: room}
}

// Send appends text to from's room and returns what it recorded, together with
// the roles the target resolved to and which of them nobody is attending under.
//
// The message takes the room's next seq, so the room's conversation stays dense
// and ordered however the clocks of its members disagree. The seq, the message,
// its mentions and the pruning of what fell past the history cap are one
// transaction: a message is either wholly in the room or not in it.
//
// An empty target Kind is a board post. A roles target names roles that exist,
// meaning roles that have attended the room at least once, and is otherwise
// [ErrUnknownRole]; a bare name that two teams of the room carry is
// [ErrAmbiguousRole]. Nothing is recorded when the target does not resolve.
func (s *Store) Send(
	ctx context.Context,
	from Sender,
	target Target,
	text string,
	sentAt time.Time,
) (Sent, error) {
	if from.Room == "" {
		return Sent{}, fmt.Errorf("sending message: empty room")
	}
	if from.Name == "" {
		return Sent{}, fmt.Errorf("sending message: empty sender name")
	}
	if target.Kind == "" {
		target.Kind = TargetNone
	}
	switch target.Kind {
	case TargetNone, TargetEveryone:
		// Neither names a role, so neither writes a mention row.
		target.Roles = nil
	case TargetRoles:
		if len(target.Roles) == 0 {
			return Sent{}, fmt.Errorf("sending to room %q: a roles target names nobody: %w",
				from.Room, ErrInvalidArgument)
		}
	default:
		return Sent{}, fmt.Errorf("sending to room %q: unknown target kind %q: %w",
			from.Room, target.Kind, ErrInvalidArgument)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return Sent{}, fmt.Errorf("minting message id: %w", err)
	}

	var (
		seq      int64
		resolved []roleKey
	)
	err = s.tx(ctx, func(q *db.Queries) error {
		if err := q.EnsureRoom(ctx, from.Room); err != nil {
			return fmt.Errorf("recording room %q: %w", from.Room, err)
		}
		if target.Kind == TargetRoles {
			roles, err := resolveTargets(ctx, q, from, target.Roles, true)
			if err != nil {
				return err
			}
			resolved = roles
		}
		next, err := q.NextRoomSeq(ctx, from.Room)
		if err != nil {
			return fmt.Errorf("claiming next seq of room %q: %w", from.Room, err)
		}
		seq = next
		err = q.InsertMessage(ctx, db.InsertMessageParams{
			ID:         id.String(),
			Room:       from.Room,
			Seq:        seq,
			FromName:   from.Name,
			FromTeam:   from.Team,
			TargetKind: string(target.Kind),
			Text:       text,
			SentAt:     formatTimestamp(sentAt),
		})
		if err != nil {
			return fmt.Errorf("appending message to room %q: %w", from.Room, err)
		}
		for _, r := range resolved {
			err := q.InsertMessageMention(ctx, db.InsertMessageMentionParams{
				MessageID: id.String(),
				Team:      r.Team,
				Name:      r.Name,
			})
			if err != nil {
				return fmt.Errorf("recording mention of %q: %w", r, err)
			}
		}
		return s.prune(ctx, q, from.Room, seq)
	})
	if err != nil {
		return Sent{}, err
	}

	sent := Sent{Message: Message{
		Id:     id.String(),
		Room:   from.Room,
		Seq:    seq,
		From:   from,
		Target: Target{Kind: target.Kind},
		Text:   text,
		SentAt: sentAt.UTC(),
	}}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range resolved {
		role := r.sender(from.Room)
		sent.Message.Target.Roles = append(sent.Message.Target.Roles, role)
		if m, ok := s.attendingAs(role); ok {
			sent.Mentioned = append(sent.Mentioned, m)
			continue
		}
		// A role nobody attends under is still a role: the mention waits at its
		// read position until it attends again.
		absent := Member{Room: role.Room, Team: role.Team, Name: role.Name}
		sent.Mentioned = append(sent.Mentioned, absent)
		sent.Absent = append(sent.Absent, absent)
	}
	return sent, nil
}

// prune drops what fell past the room's history cap, which the caller applies
// in the same transaction that appended the message pushing it over.
//
// A negative cap prunes nothing. A host that sets one is saying the room's
// conversation is theirs to keep and to clear by hand, not that it is worth
// nothing: the read positions are seqs into that conversation, and throwing it
// away would leave every role pointing at messages that are gone.
func (s *Store) prune(ctx context.Context, q *db.Queries, room string, lastSeq int64) error {
	if s.historyLimit <= 0 {
		return nil
	}
	err := q.PruneRoomMessages(ctx, db.PruneRoomMessagesParams{
		Room: room,
		Seq:  lastSeq - int64(s.historyLimit),
	})
	if err != nil {
		return fmt.Errorf("pruning room %q: %w", room, err)
	}
	return nil
}

// resolveTargets turns the written targets into the roles of from's room they
// name, dropping duplicates and keeping the order they were written in.
//
// A target carrying a team is matched exactly. A bare name resolves in the
// sender's own team first, then uniquely across the room; the host operator has
// no team, so for it only the second half applies.
//
// strict decides what an unresolvable target is. A send refuses one, since a
// message addressed to nobody is a message the sender meant to go somewhere. A
// read filter takes it as matching nothing: filtering by a role that the room
// has never had is an answer, not a mistake.
func resolveTargets(
	ctx context.Context,
	q *db.Queries,
	from Sender,
	targets []Sender,
	strict bool,
) ([]roleKey, error) {
	rows, err := q.ListRoomRoles(ctx, from.Room)
	if err != nil {
		return nil, fmt.Errorf("listing roles of room %q: %w", from.Room, err)
	}
	known := make([]roleKey, len(rows))
	for i, row := range rows {
		known[i] = roleKey{Team: row.Team, Name: row.Name}
	}

	var (
		resolved []roleKey
		seen     = make(map[roleKey]struct{}, len(targets))
	)
	for _, t := range targets {
		r, err := resolveTarget(known, from, t)
		if err != nil {
			if strict {
				return nil, err
			}
			continue
		}
		if _, dup := seen[r]; dup {
			continue
		}
		seen[r] = struct{}{}
		resolved = append(resolved, r)
	}
	return resolved, nil
}

// resolveTarget resolves one written target against the roles the room knows.
func resolveTarget(known []roleKey, from Sender, t Sender) (roleKey, error) {
	if t.Name == "" {
		return roleKey{}, fmt.Errorf("target of room %q names no role: %w",
			from.Room, ErrInvalidArgument)
	}
	if t.Team != "" {
		want := roleKey{Team: t.Team, Name: t.Name}
		for _, k := range known {
			if k == want {
				return k, nil
			}
		}
		return roleKey{}, fmt.Errorf("addressing %q in room %q: %w",
			want, from.Room, ErrUnknownRole)
	}

	var candidates []roleKey
	for _, k := range known {
		if k.Name != t.Name {
			continue
		}
		if from.Team != "" && k.Team == from.Team {
			// The sender's own team wins outright, so a name it shares with
			// another team still addresses the colleague next to it.
			return k, nil
		}
		candidates = append(candidates, k)
	}
	switch len(candidates) {
	case 0:
		return roleKey{}, fmt.Errorf("addressing %q in room %q: %w",
			t.Name, from.Room, ErrUnknownRole)
	case 1:
		return candidates[0], nil
	default:
		teams := make([]string, len(candidates))
		for i, c := range candidates {
			teams[i] = c.Team
		}
		return roleKey{}, fmt.Errorf(
			"addressing %q in room %q: teams %s all have a role named %q, address it as %q: %w",
			t.Name, from.Room, strings.Join(teams, ", "), t.Name, "<team>/"+t.Name,
			ErrAmbiguousRole)
	}
}
