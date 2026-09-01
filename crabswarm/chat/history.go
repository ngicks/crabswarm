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
	// ID orders the entry within its room and is the cursor [Store.HistorySince]
	// reads forward from. It grows, but not by one per entry of the room: every
	// room shares one log, so the ids in between went to what was said elsewhere.
	ID int64
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
	rows, err := s.q.RoomLogTail(ctx, db.RoomLogTailParams{
		Room:  room,
		Limit: int64(s.readLimit(limit)),
	})
	if err != nil {
		return nil, fmt.Errorf("reading history of room %q: %w", room, err)
	}
	entries := make([]HistoryEntry, len(rows))
	for i, row := range rows {
		entry, err := historyEntry(room, logRow(row))
		if err != nil {
			return nil, err
		}
		// The query hands back the newest first, since that is the end the tail
		// is cheap to take; the reader wants the conversation in the order it
		// happened.
		entries[len(rows)-1-i] = entry
	}
	return entries, nil
}

// HistorySince returns up to limit entries of room's conversation newer than
// sinceID, oldest first, and consumes nothing the way [Store.History] does not.
//
// It is how a reader follows a room it has already read part of: it remembers
// the [HistoryEntry.ID] it last saw and asks for what came after, so the answer
// is what is new rather than a window it has to diff against the last one. A
// sinceID of zero precedes every entry, which makes it the whole retained
// conversation from the beginning; a sinceID at or past the newest entry yields
// nothing, which is what a reader that is up to date is told.
//
// Entries the room has pruned away are simply gone, so a reader that was away
// longer than the cap resumes at what is left rather than being told it missed
// something. limit is clamped like [Store.History]'s.
func (s *Store) HistorySince(
	ctx context.Context,
	room string,
	sinceID int64,
	limit int,
) ([]HistoryEntry, error) {
	rows, err := s.q.RoomLogSince(ctx, db.RoomLogSinceParams{
		Room:  room,
		ID:    sinceID,
		Limit: int64(s.readLimit(limit)),
	})
	if err != nil {
		return nil, fmt.Errorf("reading history of room %q: %w", room, err)
	}
	entries := make([]HistoryEntry, len(rows))
	for i, row := range rows {
		entry, err := historyEntry(room, logRow(row))
		if err != nil {
			return nil, err
		}
		entries[i] = entry
	}
	return entries, nil
}

// readLimit settles how many rows a transcript read asks the log for. A
// non-positive limit means the whole retained tail, which the per-room cap
// already bounds — clamped at zero because the cap of a store that records
// nothing is negative, and SQLite reads a negative LIMIT as no limit at all.
func (s *Store) readLimit(limit int) int {
	if limit > 0 {
		return limit
	}
	return max(s.historyLimit, 0)
}

// logRow is one room_log row as both reads select it. The generated row types
// differ only in the order their query returns them, so they convert to this
// one shape and one reader turns it into an entry.
type logRow struct {
	ID       int64
	FromName string
	FromTeam string
	ToName   string
	ToTeam   string
	Text     string
	SentAt   string
}

// historyEntry reads one logged row back as it was said. The room comes from
// the caller rather than the row: it is what the read asked for, and the log
// stores it once per row instead of once per identity.
func historyEntry(room string, row logRow) (HistoryEntry, error) {
	sentAt, err := time.Parse(time.RFC3339Nano, row.SentAt)
	if err != nil {
		return HistoryEntry{}, fmt.Errorf("parsing message timestamp %q: %w", row.SentAt, err)
	}
	entry := HistoryEntry{
		ID:     row.ID,
		From:   Sender{Name: row.FromName, Team: row.FromTeam, Room: room},
		Text:   row.Text,
		SentAt: sentAt,
	}
	if row.ToName != "" {
		entry.To = &Sender{Name: row.ToName, Team: row.ToTeam, Room: room}
	}
	return entry, nil
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
