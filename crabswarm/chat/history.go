package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/ngicks/crabswarm/crabswarm/chat/internal/db"
)

// HistoryEntry is one utterance of a room's conversation, kept as it was said
// rather than as it was delivered: the identities are snapshots of send time,
// so an entry still reads correctly after its author leaves or moves team.
type HistoryEntry struct {
	// From is who said it.
	From Sender
	// To is the member a directed send was addressed to, nil for a broadcast,
	// which was addressed to the room rather than to anyone in it.
	To *Sender
	// Text is the message body.
	Text string
	// SentAt is when the daemon accepted the message.
	SentAt time.Time
}

// History returns the last limit entries of room's conversation, oldest first.
// It reads and consumes nothing, so two callers — or the same one twice — see
// the same transcript.
//
// It is keyed by the room rather than by a member's token because the
// transcript belongs to the room: the host operator reads one without
// attending it, and a member's own token only ever names one room anyway.
//
// A non-positive limit returns the whole retained tail, which the per-room cap
// already bounds, and nothing at all from a store that records no conversation:
// a host who switched history off is not asking to be handed what an earlier
// run left behind. A room nobody has spoken in yields no entries and no error:
// there is no such thing as a room that does not exist yet, only one nothing
// was said in.
func (s *Store) History(ctx context.Context, room string, limit int) ([]HistoryEntry, error) {
	if limit <= 0 {
		// Clamped because the cap of a store that records nothing is negative,
		// and SQLite reads a negative LIMIT as no limit at all.
		limit = max(s.historyLimit, 0)
	}
	rows, err := s.q.RoomLogTail(ctx, db.RoomLogTailParams{
		Room:  room,
		Limit: int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("reading history of room %q: %w", room, err)
	}
	entries := make([]HistoryEntry, len(rows))
	for i, row := range rows {
		sentAt, err := time.Parse(time.RFC3339Nano, row.SentAt)
		if err != nil {
			return nil, fmt.Errorf("parsing message timestamp %q: %w", row.SentAt, err)
		}
		entry := HistoryEntry{
			From:   Sender{Name: row.FromName, Team: row.FromTeam, Room: room},
			Text:   row.Text,
			SentAt: sentAt,
		}
		if row.ToName != "" {
			entry.To = &Sender{Name: row.ToName, Team: row.ToTeam, Room: room}
		}
		// The query hands back the newest first, since that is the end the tail
		// is cheap to take; the reader wants the conversation in the order it
		// happened.
		entries[len(rows)-1-i] = entry
	}
	return entries, nil
}

// logMessage records one utterance in the room's conversation log and prunes
// the room back to its cap, both through the caller's queries handle so the
// record shares the transaction that delivered the message: a message nobody
// received is a message the transcript must not claim was said.
//
// to is the member a directed send was addressed to, nil for a broadcast, which
// addresses the room rather than anyone in it. One row per utterance either
// way: history records what was said, the inbox rows record who received it.
func (s *Store) logMessage(
	ctx context.Context,
	q *db.Queries,
	from Sender,
	to *Sender,
	text string,
	sentAt time.Time,
) error {
	if s.historyLimit < 0 {
		return nil
	}
	var toName, toTeam string
	if to != nil {
		toName, toTeam = to.Name, to.Team
	}
	err := q.InsertRoomLog(ctx, db.InsertRoomLogParams{
		Room:     from.Room,
		FromName: from.Name,
		FromTeam: from.Team,
		ToName:   toName,
		ToTeam:   toTeam,
		Text:     text,
		SentAt:   formatTimestamp(sentAt),
	})
	if err != nil {
		return fmt.Errorf("recording message of room %q: %w", from.Room, err)
	}
	// Pruning on insert keeps the table bounded without a background job, and
	// the room is indexed, so the delete costs one lookup per message.
	err = q.PruneRoomLog(ctx, db.PruneRoomLogParams{
		Room: from.Room,
		Keep: int64(s.historyLimit),
	})
	if err != nil {
		return fmt.Errorf("pruning history of room %q: %w", from.Room, err)
	}
	return nil
}
