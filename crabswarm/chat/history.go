package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/ngicks/crabswarm/crabswarm/chat/internal/db"
)

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
