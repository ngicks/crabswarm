package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/ngicks/crabswarm/crabswarm/chat/internal/db"
)

// Send resolves addr against the caller's room — see [Store.Resolve] for the
// addressing rules — and appends text to the addressed member's inbox. It
// returns that member so the caller can notify their harness.
//
// Resolution and delivery share one transaction, so a member removed between
// the two cannot be handed back as a recipient of a message that no longer
// exists.
func (s *Store) Send(
	ctx context.Context,
	fromToken, addr, text string,
	sentAt time.Time,
) (Member, error) {
	var recipient Member
	err := s.tx(ctx, func(q *db.Queries) error {
		from, err := memberByToken(ctx, q, fromToken)
		if err != nil {
			return fmt.Errorf("sending message: %w", err)
		}
		recipient, err = s.sendFrom(ctx, q, senderOf(from), addr, text, sentAt)
		return err
	})
	if err != nil {
		return Member{}, err
	}
	return recipient, nil
}

// sendAs is [Store.Send] for a sender that holds no member row — the host
// operator, who addresses a room without attending it. from carries both the
// perspective the address is resolved from and the attribution the message
// keeps.
func (s *Store) sendAs(
	ctx context.Context,
	from Sender,
	addr, text string,
	sentAt time.Time,
) (Member, error) {
	var recipient Member
	err := s.tx(ctx, func(q *db.Queries) error {
		var err error
		recipient, err = s.sendFrom(ctx, q, from, addr, text, sentAt)
		return err
	})
	if err != nil {
		return Member{}, err
	}
	return recipient, nil
}

// sendFrom resolves addr, appends the message and records it in the room's
// conversation through the caller's queries handle, which is what keeps the
// halves of a delivery in one transaction.
func (s *Store) sendFrom(
	ctx context.Context,
	q *db.Queries,
	from Sender,
	addr, text string,
	sentAt time.Time,
) (Member, error) {
	to, err := resolveFor(ctx, q, from, addr)
	if err != nil {
		return Member{}, err
	}
	if err := appendMessage(ctx, q, to.Token, from, text, sentAt); err != nil {
		return Member{}, err
	}
	addressed := senderOf(to)
	if err := s.logMessage(ctx, q, from, &addressed, text, sentAt); err != nil {
		return Member{}, err
	}
	return to, nil
}

// Broadcast appends text to the inbox of every member of the caller's room and
// returns the recipients, ordered by team then name. excludeSender leaves the
// caller out; whether an announcement should echo back to its sender is the
// caller's call, not the store's.
func (s *Store) Broadcast(
	ctx context.Context,
	fromToken, text string,
	sentAt time.Time,
	excludeSender bool,
) ([]Member, error) {
	var recipients []Member
	err := s.tx(ctx, func(q *db.Queries) error {
		from, err := memberByToken(ctx, q, fromToken)
		if err != nil {
			return fmt.Errorf("broadcasting message: %w", err)
		}
		excluded := ""
		if excludeSender {
			excluded = from.Token
		}
		recipients, err = s.broadcastFrom(ctx, q, senderOf(from), excluded, text, sentAt)
		return err
	})
	if err != nil {
		return nil, err
	}
	return recipients, nil
}

// broadcastAs is [Store.Broadcast] for a sender that holds no member row — the
// host operator addressing a whole room they do not attend. There is nobody to
// leave out, and a room with no members at all is [ErrNotFound]: a room exists
// because members are in it, so an empty one is a room that was misspelled.
func (s *Store) broadcastAs(
	ctx context.Context,
	from Sender,
	text string,
	sentAt time.Time,
) ([]Member, error) {
	var recipients []Member
	err := s.tx(ctx, func(q *db.Queries) error {
		var err error
		recipients, err = s.broadcastFrom(ctx, q, from, "", text, sentAt)
		if err != nil {
			return err
		}
		if len(recipients) == 0 {
			return fmt.Errorf("broadcasting to room %q: %w", from.Room, ErrNotFound)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return recipients, nil
}

// broadcastFrom appends the message to every member of from's room and records
// it once in the room's conversation, inside the caller's transaction, skipping
// the member holding excludeToken. An empty excludeToken excludes nobody: no
// member holds one, [Store.Join] refuses it.
//
// The single record is the announcement itself, so it is written whoever heard
// it — a room that was empty at the time still said the words.
func (s *Store) broadcastFrom(
	ctx context.Context,
	q *db.Queries,
	from Sender,
	excludeToken, text string,
	sentAt time.Time,
) ([]Member, error) {
	rows, err := q.ListRoomMembers(ctx, from.Room)
	if err != nil {
		return nil, fmt.Errorf("broadcasting to room %q: %w", from.Room, err)
	}
	members, err := membersOf(rows)
	if err != nil {
		return nil, fmt.Errorf("broadcasting to room %q: %w", from.Room, err)
	}
	var recipients []Member
	for _, m := range members {
		if m.Token == excludeToken {
			continue
		}
		if err := appendMessage(ctx, q, m.Token, from, text, sentAt); err != nil {
			return nil, err
		}
		recipients = append(recipients, m)
	}
	if err := s.logMessage(ctx, q, from, nil, text, sentAt); err != nil {
		return nil, err
	}
	return recipients, nil
}

// Read returns the pending messages of the member holding token, oldest first,
// and drains the inbox in the same transaction: a message is delivered exactly
// once, and a second Read returns nothing. An unknown token is [ErrNotFound].
func (s *Store) Read(ctx context.Context, token string) ([]Message, error) {
	var messages []Message
	err := s.tx(ctx, func(q *db.Queries) error {
		if _, err := memberByToken(ctx, q, token); err != nil {
			return fmt.Errorf("reading inbox: %w", err)
		}
		msgs, err := pendingMessages(ctx, q, token)
		if err != nil {
			return err
		}
		if err := q.DeleteMessages(ctx, token); err != nil {
			return fmt.Errorf("draining inbox of %q: %w", token, err)
		}
		messages = msgs
		return nil
	})
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// pendingMessages reads the whole inbox before the caller deletes it. The
// generated query closes its rows before returning, which the delete depends
// on since the store holds a single connection.
func pendingMessages(ctx context.Context, q *db.Queries, token string) ([]Message, error) {
	rows, err := q.PendingMessages(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("reading inbox of %q: %w", token, err)
	}
	var messages []Message
	for _, row := range rows {
		sentAt, err := time.Parse(time.RFC3339Nano, row.SentAt)
		if err != nil {
			return nil, fmt.Errorf("parsing message timestamp %q: %w", row.SentAt, err)
		}
		messages = append(messages, Message{
			From: Sender{
				Name: row.FromName,
				Team: row.FromTeam,
				Room: row.FromRoom,
			},
			Text:   row.Text,
			SentAt: sentAt,
		})
	}
	return messages, nil
}

func appendMessage(
	ctx context.Context,
	q *db.Queries,
	recipient string,
	from Sender,
	text string,
	sentAt time.Time,
) error {
	err := q.InsertMessage(ctx, db.InsertMessageParams{
		Recipient: recipient,
		FromName:  from.Name,
		FromTeam:  from.Team,
		FromRoom:  from.Room,
		Text:      text,
		SentAt:    formatTimestamp(sentAt),
	})
	if err != nil {
		return fmt.Errorf("appending message for %q: %w", recipient, err)
	}
	return nil
}

func senderOf(m Member) Sender {
	return Sender{Name: m.Name, Team: m.Team, Room: m.Room}
}
