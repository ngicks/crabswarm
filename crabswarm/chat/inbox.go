package chat

import (
	"context"
	"database/sql"
	"fmt"
	"time"
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
	err := s.tx(ctx, func(tx *sql.Tx) error {
		from, err := memberByToken(ctx, tx, fromToken)
		if err != nil {
			return fmt.Errorf("sending message: %w", err)
		}
		to, err := resolveFor(ctx, tx, from, addr)
		if err != nil {
			return err
		}
		if err := appendMessage(ctx, tx, to.Token, senderOf(from), text, sentAt); err != nil {
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
	err := s.tx(ctx, func(tx *sql.Tx) error {
		from, err := memberByToken(ctx, tx, fromToken)
		if err != nil {
			return fmt.Errorf("broadcasting message: %w", err)
		}
		members, err := queryMembers(ctx, tx,
			`SELECT `+memberColumns+` FROM members WHERE room = ? ORDER BY team, name`,
			from.Room)
		if err != nil {
			return fmt.Errorf("broadcasting to room %q: %w", from.Room, err)
		}
		sender := senderOf(from)
		for _, m := range members {
			if excludeSender && m.Token == from.Token {
				continue
			}
			if err := appendMessage(ctx, tx, m.Token, sender, text, sentAt); err != nil {
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
	err := s.tx(ctx, func(tx *sql.Tx) error {
		if _, err := memberByToken(ctx, tx, token); err != nil {
			return fmt.Errorf("reading inbox: %w", err)
		}
		msgs, err := pendingMessages(ctx, tx, token)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM messages WHERE recipient = ?`, token); err != nil {
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

// pendingMessages reads the whole inbox before the caller deletes it: the rows
// must be consumed and closed before the delete runs, since the store holds a
// single connection.
func pendingMessages(ctx context.Context, tx *sql.Tx, token string) ([]Message, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT from_name, from_team, from_room, text, sent_at
		 FROM messages WHERE recipient = ? ORDER BY id`, token)
	if err != nil {
		return nil, fmt.Errorf("reading inbox of %q: %w", token, err)
	}
	defer func() { _ = rows.Close() }()
	var messages []Message
	for rows.Next() {
		var (
			m      Message
			sentAt string
		)
		if err := rows.Scan(
			&m.From.Name, &m.From.Team, &m.From.Room, &m.Text, &sentAt); err != nil {
			return nil, fmt.Errorf("reading message row: %w", err)
		}
		parsed, err := time.Parse(time.RFC3339Nano, sentAt)
		if err != nil {
			return nil, fmt.Errorf("parsing message timestamp %q: %w", sentAt, err)
		}
		m.SentAt = parsed
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading inbox of %q: %w", token, err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("reading inbox of %q: %w", token, err)
	}
	return messages, nil
}

func appendMessage(
	ctx context.Context,
	tx *sql.Tx,
	recipient string,
	from Sender,
	text string,
	sentAt time.Time,
) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO messages (recipient, from_name, from_team, from_room, text, sent_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		recipient, from.Name, from.Team, from.Room, text,
		sentAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("appending message for %q: %w", recipient, err)
	}
	return nil
}

func senderOf(m Member) Sender {
	return Sender{Name: m.Name, Team: m.Team, Room: m.Room}
}
