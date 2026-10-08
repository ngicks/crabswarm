package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ngicks/crabswarm/crabswarm/chat/internal/db"
)

// Read returns messages of role's room from the filter's cursor, oldest first,
// and reports how many unread mentions role has left afterwards.
//
// It moves role's read position to the newest seq it returns, or leaves it
// where it is when that is not newer — a read of old history shows what the
// role has already seen and must not un-read what came after. The position is a
// cursor rather than a receipt: one number per role and room, so a role that
// reads the tail has read past everything before it, whether or not those
// messages were ever shown.
//
// An empty Cursor means [CursorUnread]. A role that has never attended the room
// has no position to read from and is [ErrUnknownRole].
func (s *Store) Read(ctx context.Context, role Sender, f ReadFilter) ([]Message, int, error) {
	if role.Room == "" {
		return nil, 0, fmt.Errorf("reading chat: empty room")
	}
	if err := validateName(role.Team, role.Name); err != nil {
		return nil, 0, fmt.Errorf("reading chat: %w", err)
	}
	f, err := normalizeFilter(f, CursorUnread)
	if err != nil {
		return nil, 0, fmt.Errorf("reading room %q: %w", role.Room, err)
	}

	var (
		messages  []Message
		remaining int64
	)
	err = s.tx(ctx, func(q *db.Queries) error {
		position, err := q.ReadPosition(ctx, db.ReadPositionParams{
			Room: role.Room,
			Team: role.Team,
			Name: role.Name,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("reading room %q as %q: %w",
				role.Room, role.Team+"/"+role.Name, ErrUnknownRole)
		}
		if err != nil {
			return fmt.Errorf("reading position of %q in room %q: %w",
				role.Team+"/"+role.Name, role.Room, err)
		}

		messages, err = selectMessages(ctx, q, role.Room, f, role, position)
		if err != nil {
			return err
		}

		moved := position
		if n := len(messages); n > 0 && messages[n-1].Seq > moved {
			moved = messages[n-1].Seq
			err := q.AdvanceReadPosition(ctx, db.AdvanceReadPositionParams{
				LastSeq: moved,
				Room:    role.Room,
				Team:    role.Team,
				Name:    role.Name,
			})
			if err != nil {
				return fmt.Errorf("moving read position of %q in room %q: %w",
					role.Team+"/"+role.Name, role.Room, err)
			}
		}
		// Against the position as it was before this read: a message shown by
		// this very read is one the role had not seen.
		for i := range messages {
			messages[i].MentionedYou = mentions(messages[i], role, position)
		}

		remaining, err = q.CountUnreadMentions(ctx, db.CountUnreadMentionsParams{
			Room:     role.Room,
			After:    moved,
			RoleTeam: role.Team,
			RoleName: role.Name,
		})
		if err != nil {
			return fmt.Errorf("counting unread of %q in room %q: %w",
				role.Team+"/"+role.Name, role.Room, err)
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return messages, int(remaining), nil
}

// Skip moves role's read position through seq, inclusive, without handing any
// message over, and reports how many unread mentions role has left afterwards.
//
// It is for a caller that already showed those messages some other way and
// would otherwise have to read them a second time only to mark them read. The
// position is one number per role and room, so skipping through seq also marks
// every earlier unread mention read, whether or not it was ever shown.
//
// The position never moves backward, so skipping through a seq at or before it
// changes nothing and a repeated skip answers the same. A seq past the room's
// newest reads through the newest: the position never runs ahead of the room,
// where it would swallow messages nobody has sent yet.
//
// A seq below one names no message and is [ErrInvalidArgument]. A role that has
// never attended the room has no position to move and is [ErrUnknownRole].
func (s *Store) Skip(ctx context.Context, role Sender, seq int64) (int, error) {
	if role.Room == "" {
		return 0, fmt.Errorf("skipping chat: empty room")
	}
	if err := validateName(role.Team, role.Name); err != nil {
		return 0, fmt.Errorf("skipping chat: %w", err)
	}
	if seq < 1 {
		return 0, fmt.Errorf("skipping room %q through seq %d: seqs start at 1: %w",
			role.Room, seq, ErrInvalidArgument)
	}

	var remaining int64
	err := s.tx(ctx, func(q *db.Queries) error {
		position, err := q.ReadPosition(ctx, db.ReadPositionParams{
			Room: role.Room,
			Team: role.Team,
			Name: role.Name,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("skipping room %q as %q: %w",
				role.Room, role.Team+"/"+role.Name, ErrUnknownRole)
		}
		if err != nil {
			return fmt.Errorf("reading position of %q in room %q: %w",
				role.Team+"/"+role.Name, role.Room, err)
		}

		last, err := q.RoomLastSeq(ctx, role.Room)
		if err != nil {
			return fmt.Errorf("reading the newest seq of room %q: %w", role.Room, err)
		}

		moved := position
		if through := min(seq, last); through > moved {
			moved = through
			err := q.AdvanceReadPosition(ctx, db.AdvanceReadPositionParams{
				LastSeq: moved,
				Room:    role.Room,
				Team:    role.Team,
				Name:    role.Name,
			})
			if err != nil {
				return fmt.Errorf("moving read position of %q in room %q: %w",
					role.Team+"/"+role.Name, role.Room, err)
			}
		}

		remaining, err = q.CountUnreadMentions(ctx, db.CountUnreadMentionsParams{
			Room:     role.Room,
			After:    moved,
			RoleTeam: role.Team,
			RoleName: role.Name,
		})
		if err != nil {
			return fmt.Errorf("counting unread of %q in room %q: %w",
				role.Team+"/"+role.Name, role.Room, err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return int(remaining), nil
}

// CountUnread reports how many messages past role's read position mention it:
// the same count [Store.Read] leaves behind, asked for on its own.
//
// It moves nothing. A caller that only wants to know whether there is anything
// waiting — before waking an agent that has just reattended, or one that
// reported itself done — would otherwise have to read, which consumes the very
// mentions it was asking about.
//
// A role that has never attended the room has no position to count from and is
// [ErrUnknownRole].
func (s *Store) CountUnread(ctx context.Context, role Sender) (int, error) {
	if role.Room == "" {
		return 0, fmt.Errorf("counting unread: empty room")
	}
	if err := validateName(role.Team, role.Name); err != nil {
		return 0, fmt.Errorf("counting unread: %w", err)
	}

	var unread int64
	err := s.tx(ctx, func(q *db.Queries) error {
		position, err := q.ReadPosition(ctx, db.ReadPositionParams{
			Room: role.Room,
			Team: role.Team,
			Name: role.Name,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("counting unread of %q in room %q: %w",
				role.Team+"/"+role.Name, role.Room, ErrUnknownRole)
		}
		if err != nil {
			return fmt.Errorf("reading position of %q in room %q: %w",
				role.Team+"/"+role.Name, role.Room, err)
		}
		unread, err = q.CountUnreadMentions(ctx, db.CountUnreadMentionsParams{
			Room:     role.Room,
			After:    position,
			RoleTeam: role.Team,
			RoleName: role.Name,
		})
		if err != nil {
			return fmt.Errorf("counting unread of %q in room %q: %w",
				role.Team+"/"+role.Name, role.Room, err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return int(unread), nil
}

// ReadRoom is [Store.Read] on nobody's behalf: the same filter over a named
// room, for the host operator who reads a room without attending it. It moves
// no read position and marks nothing as mentioning the reader.
//
// An empty Cursor means [CursorTail]. [CursorUnread] is [ErrInvalidArgument]:
// unread is measured from a role's position, and this read has no role.
func (s *Store) ReadRoom(ctx context.Context, room string, f ReadFilter) ([]Message, error) {
	if room == "" {
		return nil, fmt.Errorf("reading chat: empty room")
	}
	if f.Cursor == CursorUnread {
		return nil, fmt.Errorf(
			"reading room %q: unread is measured from a role's position and this read has none: %w",
			room, ErrInvalidArgument)
	}
	f, err := normalizeFilter(f, CursorTail)
	if err != nil {
		return nil, fmt.Errorf("reading room %q: %w", room, err)
	}

	var messages []Message
	err = s.tx(ctx, func(q *db.Queries) error {
		var err error
		messages, err = selectMessages(ctx, q, room, f, Sender{Room: room}, 0)
		return err
	})
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// normalizeFilter fills in what the caller left blank and refuses a range that
// runs against its cursor: nothing lies before the head or after the tail, and
// unread only ever counts forward.
func normalizeFilter(f ReadFilter, fallback ReadCursor) (ReadFilter, error) {
	if f.Cursor == "" {
		f.Cursor = fallback
	}
	switch f.Cursor {
	case CursorUnread, CursorHead:
		if f.Range == 0 {
			f.Range = defaultReadRange
		}
		if f.Range < 0 {
			return f, fmt.Errorf("a %s read counts forward, not %d back: %w",
				f.Cursor, -f.Range, ErrInvalidArgument)
		}
	case CursorTail:
		if f.Range == 0 {
			f.Range = -defaultReadRange
		}
		if f.Range > 0 {
			return f, fmt.Errorf("a %s read counts backward, not %d forward: %w",
				f.Cursor, f.Range, ErrInvalidArgument)
		}
	default:
		return f, fmt.Errorf("unknown read cursor %q: %w", f.Cursor, ErrInvalidArgument)
	}
	return f, nil
}

// mentions reports whether m is one role had not been shown as of position:
// addressed to it or to everyone, not sent by it, and newer than where it had
// read up to.
func mentions(m Message, role Sender, position int64) bool {
	if m.Seq <= position {
		return false
	}
	return addressedTo(m.From, m.Target, role)
}

// addressedTo reports whether a message from from, written to target, is for
// role: addressed to it or to everyone, and not sent by it. It is [mentions]
// with no read position, for a follower, which moves none.
func addressedTo(from Sender, target Target, role Sender) bool {
	if from.Team == role.Team && from.Name == role.Name {
		return false
	}
	switch target.Kind {
	case TargetEveryone:
		return true
	case TargetRoles:
		for _, r := range target.Roles {
			if r.Team == role.Team && r.Name == role.Name {
				return true
			}
		}
	}
	return false
}

// newestSeq reports the newest seq room has handed out. A room nothing has been
// said in has none, and neither does one that does not exist yet, which is zero
// for both: a follower may well open its stream before anybody arrives.
func (s *Store) newestSeq(ctx context.Context, room string) (int64, error) {
	last, err := s.q.RoomLastSeq(ctx, room)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("reading the newest seq of room %q: %w", room, err)
	}
	return last, nil
}
