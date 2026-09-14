package chat

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/ngicks/crabswarm/crabswarm/chat/internal/db"
)

// readWindow is one read resolved to what the queries take.
type readWindow struct {
	room string
	// after and before bound the seqs, both exclusive; a zero before is
	// unbounded, which no real seq is, since seqs start at one.
	after, before int64
	// kind keeps one written target kind, empty keeps every kind.
	kind string
	// roles keeps messages naming any of them, empty keeps every message. They
	// are read one query per role and merged, since sqlite parameters cannot
	// carry a list.
	roles []roleKey
	// role switches on the unread predicate for that role; its zero value
	// leaves it off, which no real role is, names being non-empty.
	role  roleKey
	limit int64
	// desc takes the window from its upper bound backward, which is how the
	// tail cursor reads. The rows still come back oldest first.
	desc bool
}

// selectMessages reads the window f describes, oldest first. viewer is the role
// the unread cursor is measured for and whose team resolves a bare name in the
// To filter; position is where that role has read up to.
func selectMessages(
	ctx context.Context,
	q *db.Queries,
	room string,
	f ReadFilter,
	viewer Sender,
	position int64,
) ([]Message, error) {
	w := readWindow{
		room:   room,
		after:  f.Since,
		before: f.Until,
		limit:  int64(f.Range),
	}
	switch f.Cursor {
	case CursorUnread:
		w.role = roleKey{Team: viewer.Team, Name: viewer.Name}
		w.after = max(f.Since, position)
	case CursorTail:
		w.desc = true
		w.limit = -w.limit
	}

	if f.To != nil {
		kind := f.To.Kind
		if kind == "" {
			kind = TargetNone
		}
		switch kind {
		case TargetNone, TargetEveryone:
			w.kind = string(kind)
		case TargetRoles:
			// A name the room never had matches nothing rather than failing:
			// filtering by a role that was never there is an answer, not a
			// mistake. An ambiguous one is still refused — see
			// [resolveTargets].
			roles, err := resolveTargets(ctx, q, viewer, f.To.Roles, false)
			if err != nil {
				return nil, err
			}
			if len(roles) == 0 {
				return nil, nil
			}
			w.roles = roles
		default:
			return nil, fmt.Errorf("unknown target kind %q: %w", kind, ErrInvalidArgument)
		}
	}

	return readWindowRows(ctx, q, w)
}

// readWindowRows runs the window and turns the rows into messages, oldest
// first, with the target each message was written with.
func readWindowRows(ctx context.Context, q *db.Queries, w readWindow) ([]Message, error) {
	if w.limit <= 0 {
		return nil, nil
	}
	passes := w.roles
	if len(passes) == 0 {
		// One pass with the role narrowing switched off.
		passes = []roleKey{{}}
	}

	rows := make(map[int64]Message)
	for _, to := range passes {
		got, err := readPass(ctx, q, w, to)
		if err != nil {
			return nil, err
		}
		for _, m := range got {
			rows[m.Seq] = m
		}
	}

	// Merging per-role passes can overshoot the limit, so the window is cut
	// here: the n the reader asked for are the n nearest its cursor.
	seqs := slices.Sorted(maps.Keys(rows))
	if int64(len(seqs)) > w.limit {
		if w.desc {
			seqs = seqs[int64(len(seqs))-w.limit:]
		} else {
			seqs = seqs[:w.limit]
		}
	}
	if len(seqs) == 0 {
		return nil, nil
	}

	messages := make([]Message, len(seqs))
	for i, seq := range seqs {
		messages[i] = rows[seq]
	}
	if err := attachTargets(ctx, q, w.room, messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// readPass runs the window once, narrowed to messages naming to when to names a
// role.
func readPass(ctx context.Context, q *db.Queries, w readWindow, to roleKey) ([]Message, error) {
	if w.desc {
		rows, err := q.MessagesDesc(ctx, db.MessagesDescParams{
			Room:     w.room,
			After:    w.after,
			Before:   w.before,
			Kind:     w.kind,
			ToTeam:   to.Team,
			ToName:   to.Name,
			RoleTeam: w.role.Team,
			RoleName: w.role.Name,
			Lim:      w.limit,
		})
		if err != nil {
			return nil, fmt.Errorf("reading room %q: %w", w.room, err)
		}
		messages := make([]Message, len(rows))
		for i, row := range rows {
			m, err := messageOf(w.room, row.ID, row.Seq, row.FromName, row.FromTeam,
				row.TargetKind, row.Text, row.SentAt)
			if err != nil {
				return nil, err
			}
			messages[i] = m
		}
		return messages, nil
	}
	rows, err := q.MessagesAsc(ctx, db.MessagesAscParams{
		Room:     w.room,
		After:    w.after,
		Before:   w.before,
		Kind:     w.kind,
		ToTeam:   to.Team,
		ToName:   to.Name,
		RoleTeam: w.role.Team,
		RoleName: w.role.Name,
		Lim:      w.limit,
	})
	if err != nil {
		return nil, fmt.Errorf("reading room %q: %w", w.room, err)
	}
	messages := make([]Message, len(rows))
	for i, row := range rows {
		m, err := messageOf(w.room, row.ID, row.Seq, row.FromName, row.FromTeam,
			row.TargetKind, row.Text, row.SentAt)
		if err != nil {
			return nil, err
		}
		messages[i] = m
	}
	return messages, nil
}

// messageOf turns one stored row into a message, less the roles a 'roles'
// message names, which live in their own table.
func messageOf(
	room, id string,
	seq int64,
	fromName, fromTeam, targetKind, text, sentAt string,
) (Message, error) {
	at, err := parseTimestamp(sentAt)
	if err != nil {
		return Message{}, err
	}
	return Message{
		Id:     id,
		Room:   room,
		Seq:    seq,
		From:   Sender{Name: fromName, Team: fromTeam, Room: room},
		Target: Target{Kind: TargetKind(targetKind)},
		Text:   text,
		SentAt: at,
	}, nil
}

// attachTargets fills in the roles each 'roles' message names. The mentions are
// fetched for the whole seq span of the window at once rather than per message.
func attachTargets(ctx context.Context, q *db.Queries, room string, messages []Message) error {
	mentions, err := q.MentionsInSeqRange(ctx, db.MentionsInSeqRangeParams{
		Room:    room,
		FromSeq: messages[0].Seq,
		ToSeq:   messages[len(messages)-1].Seq,
	})
	if err != nil {
		return fmt.Errorf("reading mentions of room %q: %w", room, err)
	}
	roles := make(map[string][]Sender, len(mentions))
	for _, m := range mentions {
		roles[m.MessageID] = append(roles[m.MessageID],
			Sender{Name: m.Name, Team: m.Team, Room: room})
	}
	for i := range messages {
		if messages[i].Target.Kind == TargetRoles {
			messages[i].Target.Roles = roles[messages[i].Id]
		}
	}
	return nil
}
