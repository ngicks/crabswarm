package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/ngicks/crabswarm/crabswarm/chat/internal/chatdb"
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
	err := s.tx(ctx, func(q *chatdb.Queries) error {
		from, err := memberByToken(ctx, q, fromToken)
		if err != nil {
			return fmt.Errorf("sending message: %w", err)
		}
		to, err := resolveFor(ctx, q, from, addr)
		if err != nil {
			return err
		}
		if err := appendMessage(ctx, q, to.Token, senderOf(from), text, sentAt); err != nil {
			return err
		}
		recipient = to
		return nil
	})
	if err != nil {
		return Member{}, err
	}
	return recipient, nil
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
	err := s.tx(ctx, func(q *chatdb.Queries) error {
		from, err := memberByToken(ctx, q, fromToken)
		if err != nil {
			return fmt.Errorf("broadcasting message: %w", err)
		}
		rows, err := q.ListRoomMembers(ctx, from.Room)
		if err != nil {
			return fmt.Errorf("broadcasting to room %q: %w", from.Room, err)
		}
		sender := senderOf(from)
		for _, m := range membersOf(rows) {
			if excludeSender && m.Token == from.Token {
				continue
			}
			if err := appendMessage(ctx, q, m.Token, sender, text, sentAt); err != nil {
				return err
			}
			recipients = append(recipients, m)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return recipients, nil
}

// Read returns the pending messages of the member holding token, oldest first,
// and drains the inbox in the same transaction: a message is delivered exactly
// once, and a second Read returns nothing. An unknown token is [ErrNotFound].
func (s *Store) Read(ctx context.Context, token string) ([]Message, error) {
	var messages []Message
	err := s.tx(ctx, func(q *chatdb.Queries) error {
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
func pendingMessages(ctx context.Context, q *chatdb.Queries, token string) ([]Message, error) {
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
	q *chatdb.Queries,
	recipient string,
	from Sender,
	text string,
	sentAt time.Time,
) error {
	err := q.InsertMessage(ctx, chatdb.InsertMessageParams{
		Recipient: recipient,
		FromName:  from.Name,
		FromTeam:  from.Team,
		FromRoom:  from.Room,
		Text:      text,
		SentAt:    sentAt.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return fmt.Errorf("appending message for %q: %w", recipient, err)
	}
	return nil
}

func senderOf(m Member) Sender {
	return Sender{Name: m.Name, Team: m.Team, Room: m.Room}
}
