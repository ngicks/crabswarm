package chat

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/ngicks/crabswarm/crabswarm/chat/internal/db"
)

// ListRooms returns every room with its attendance, ordered by name, the
// members of each by team then name. It is the whole broker, for an operator to
// inspect.
//
// A room is listed when the log knows it or when somebody attends it. The two
// sets are almost always the same; they part for the moment between a daemon
// restart and the first attend, when the log holds rooms nobody is in.
func (s *Store) ListRooms(ctx context.Context) ([]Room, error) {
	logged, err := s.q.ListRoomNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing rooms: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	names := make(map[string]struct{}, len(logged))
	for _, name := range logged {
		names[name] = struct{}{}
	}
	for _, m := range s.attending {
		names[m.Room] = struct{}{}
	}

	rooms := make([]Room, 0, len(names))
	for name := range names {
		rooms = append(rooms, Room{Name: name, Members: s.membersOf(name)})
	}
	slices.SortFunc(rooms, func(a, b Room) int { return cmp.Compare(a.Name, b.Name) })
	return rooms, nil
}

// DeleteRoom removes a room and everything it holds — its messages, their
// mentions and its read positions — and returns how many messages went with it.
// An unknown room is [ErrNotFound].
//
// It is refused with [ErrRoomAttended] while anybody attends the room: deleting
// the room under a live session would leave that session addressing a room the
// broker no longer has, and the operator asking for it means the room is done,
// which somebody still being in it contradicts.
func (s *Store) DeleteRoom(ctx context.Context, room string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if attending := s.membersOf(room); len(attending) > 0 {
		return 0, fmt.Errorf("deleting room %q: %q is attending it: %w",
			room, attending[0].Team+"/"+attending[0].Name, ErrRoomAttended)
	}

	var deleted int64
	err := s.tx(ctx, func(q *db.Queries) error {
		messages, err := q.CountRoomMessages(ctx, room)
		if err != nil {
			return fmt.Errorf("counting messages of room %q: %w", room, err)
		}
		rows, err := q.DeleteRoom(ctx, room)
		if err != nil {
			return fmt.Errorf("deleting room %q: %w", room, err)
		}
		if rows == 0 {
			return fmt.Errorf("deleting room %q: %w", room, ErrNotFound)
		}
		deleted = messages
		return nil
	})
	if err != nil {
		return 0, err
	}
	return deleted, nil
}
