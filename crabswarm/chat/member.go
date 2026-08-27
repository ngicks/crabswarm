package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// queryer is the part of [sql.DB] and [sql.Tx] the member queries need, so a
// query runs the same way inside and outside a transaction.
type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

const memberColumns = `token, name, team, room, kind, state`

// Join records m as a member and returns it.
//
// Join is idempotent per token: a token already in the store yields the stored
// member unchanged — including its inbox — and no error, so a harness hook
// firing twice needs no client-side guard. That also means Join never edits an
// existing member; use [Store.MoveMember] or [Store.RemoveMember] for that.
//
// A name already used by another member of the same team is [ErrNameTaken];
// the same name in another team of the room is fine, that is what teams are
// for. An empty State defaults to [StateDone]: attendance is declared from a
// session-start hook, before the session has work to do.
func (s *Store) Join(ctx context.Context, m Member) (Member, error) {
	if m.Token == "" {
		return Member{}, errors.New("joining chat: empty token")
	}
	if m.Room == "" {
		return Member{}, errors.New("joining chat: empty room")
	}
	if m.Kind != KindAgent && m.Kind != KindHuman {
		return Member{}, fmt.Errorf("joining chat: unknown member kind %q", m.Kind)
	}
	if err := validateName(m.Team, m.Name); err != nil {
		return Member{}, fmt.Errorf("joining chat: %w", err)
	}
	if m.State == "" {
		m.State = StateDone
	}

	joined := m
	err := s.tx(ctx, func(tx *sql.Tx) error {
		existing, err := memberByToken(ctx, tx, m.Token)
		switch {
		case err == nil:
			joined = existing
			return nil
		case !errors.Is(err, ErrNotFound):
			return err
		}
		if _, err := memberByName(ctx, tx, m.Room, m.Team, m.Name); err == nil {
			return fmt.Errorf("joining room %q as %q: %w",
				m.Room, m.Team+"/"+m.Name, ErrNameTaken)
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO members (`+memberColumns+`) VALUES (?, ?, ?, ?, ?, ?)`,
			m.Token, m.Name, m.Team, m.Room, string(m.Kind), string(m.State))
		if err != nil {
			return fmt.Errorf("inserting member %q: %w", m.Name, err)
		}
		return nil
	})
	if err != nil {
		return Member{}, err
	}
	return joined, nil
}

// Member returns the member holding token, or [ErrNotFound].
func (s *Store) Member(ctx context.Context, token string) (Member, error) {
	return memberByToken(ctx, s.db, token)
}

// SetState records the harness state token last reported. Reading it back is
// [Store.Member].
func (s *Store) SetState(ctx context.Context, token string, state MemberState) error {
	switch state {
	case StateDone, StateWorking, StateWaiting:
	default:
		return fmt.Errorf("setting state of %q: unknown state %q", token, state)
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE members SET state = ? WHERE token = ?`, string(state), token)
	if err != nil {
		return fmt.Errorf("setting state of %q: %w", token, err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("setting state of %q: %w", token, err)
	} else if n == 0 {
		return fmt.Errorf("setting state of %q: %w", token, ErrNotFound)
	}
	return nil
}

// ListMembers returns every member of room, ordered by team then name.
func (s *Store) ListMembers(ctx context.Context, room string) ([]Member, error) {
	members, err := queryMembers(ctx, s.db,
		`SELECT `+memberColumns+` FROM members WHERE room = ? ORDER BY team, name`, room)
	if err != nil {
		return nil, fmt.Errorf("listing members of room %q: %w", room, err)
	}
	return members, nil
}

// ListRooms returns every room with its teams and members, all ordered by
// name. It is the whole store, for an operator to inspect.
func (s *Store) ListRooms(ctx context.Context) ([]Room, error) {
	members, err := queryMembers(ctx, s.db,
		`SELECT `+memberColumns+` FROM members ORDER BY room, team, name`)
	if err != nil {
		return nil, fmt.Errorf("listing rooms: %w", err)
	}
	var rooms []Room
	for _, m := range members {
		if len(rooms) == 0 || rooms[len(rooms)-1].Name != m.Room {
			rooms = append(rooms, Room{Name: m.Room})
		}
		room := &rooms[len(rooms)-1]
		if len(room.Teams) == 0 || room.Teams[len(room.Teams)-1].Name != m.Team {
			room.Teams = append(room.Teams, Team{Name: m.Team})
		}
		team := &room.Teams[len(room.Teams)-1]
		team.Members = append(team.Members, m)
	}
	return rooms, nil
}

// RemoveMember drops the member holding token, discarding their pending
// messages with them, and returns the removed member. An unknown token is
// [ErrNotFound].
func (s *Store) RemoveMember(ctx context.Context, token string) (Member, error) {
	var removed Member
	err := s.tx(ctx, func(tx *sql.Tx) error {
		m, err := memberByToken(ctx, tx, token)
		if err != nil {
			return fmt.Errorf("removing member: %w", err)
		}
		// Pending messages go through the ON DELETE CASCADE on messages.
		if _, err := tx.ExecContext(ctx, `DELETE FROM members WHERE token = ?`, token); err != nil {
			return fmt.Errorf("removing member %q: %w", m.Name, err)
		}
		removed = m
		return nil
	})
	if err != nil {
		return Member{}, err
	}
	return removed, nil
}

// MoveMember moves the member holding token into team, within their current
// room, and returns the updated member. Moving into the team they already
// belong to changes nothing; a name already used in the target team is
// [ErrNameTaken].
func (s *Store) MoveMember(ctx context.Context, token, team string) (Member, error) {
	var moved Member
	err := s.tx(ctx, func(tx *sql.Tx) error {
		m, err := memberByToken(ctx, tx, token)
		if err != nil {
			return fmt.Errorf("moving member: %w", err)
		}
		moved, err = moveMember(ctx, tx, m, team)
		return err
	})
	if err != nil {
		return Member{}, err
	}
	return moved, nil
}

// MoveMemberByName is [Store.MoveMember] for a caller that addresses the member
// the way an operator sees one — room, current team and name — instead of by
// the token only its holder and the daemon know. No member there is
// [ErrNotFound]. The lookup and the move share one transaction, so the member
// cannot leave in between.
func (s *Store) MoveMemberByName(
	ctx context.Context,
	room, team, name, toTeam string,
) (Member, error) {
	var moved Member
	err := s.tx(ctx, func(tx *sql.Tx) error {
		m, err := memberByName(ctx, tx, room, team, name)
		if err != nil {
			return fmt.Errorf("moving %q in room %q: %w", team+"/"+name, room, err)
		}
		moved, err = moveMember(ctx, tx, m, toTeam)
		return err
	})
	if err != nil {
		return Member{}, err
	}
	return moved, nil
}

// moveMember implements the move itself for members already read inside tx.
func moveMember(ctx context.Context, tx *sql.Tx, m Member, team string) (Member, error) {
	if err := validateName(team, m.Name); err != nil {
		return Member{}, fmt.Errorf("moving member %q: %w", m.Name, err)
	}
	if m.Team == team {
		return m, nil
	}
	if _, err := memberByName(ctx, tx, m.Room, team, m.Name); err == nil {
		return Member{}, fmt.Errorf("moving %q into team %q: %w", m.Name, team, ErrNameTaken)
	} else if !errors.Is(err, ErrNotFound) {
		return Member{}, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE members SET team = ? WHERE token = ?`, team, m.Token); err != nil {
		return Member{}, fmt.Errorf("moving %q into team %q: %w", m.Name, team, err)
	}
	m.Team = team
	return m, nil
}

// Resolve turns a send address into the member it names, as seen by the caller
// holding callerToken. Members of other rooms are never visible.
//
// addr is either "team/name", which is matched exactly, or a bare "name",
// which resolves in the caller's own team first and otherwise has to be unique
// across the room. A bare name carried by members of two or more other teams
// is [ErrAmbiguousName], and the error names the teams so the caller can retry
// with "team/name".
func (s *Store) Resolve(ctx context.Context, callerToken, addr string) (Member, error) {
	return resolve(ctx, s.db, callerToken, addr)
}

// resolve implements [Store.Resolve] against any queryer so a send can resolve
// and deliver inside one transaction.
func resolve(ctx context.Context, q queryer, callerToken, addr string) (Member, error) {
	caller, err := memberByToken(ctx, q, callerToken)
	if err != nil {
		return Member{}, fmt.Errorf("resolving %q: %w", addr, err)
	}
	return resolveFor(ctx, q, caller, addr)
}

// resolveFor resolves addr as caller sees it, for callers that already hold
// the member.
func resolveFor(ctx context.Context, q queryer, caller Member, addr string) (Member, error) {
	if team, name, qualified := strings.Cut(addr, "/"); qualified {
		m, err := memberByName(ctx, q, caller.Room, team, name)
		if err != nil {
			return Member{}, fmt.Errorf("resolving %q in room %q: %w", addr, caller.Room, err)
		}
		return m, nil
	}

	m, err := memberByName(ctx, q, caller.Room, caller.Team, addr)
	if err == nil {
		return m, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Member{}, fmt.Errorf("resolving %q: %w", addr, err)
	}

	candidates, err := queryMembers(ctx, q,
		`SELECT `+memberColumns+` FROM members WHERE room = ? AND name = ? ORDER BY team`,
		caller.Room, addr)
	if err != nil {
		return Member{}, fmt.Errorf("resolving %q: %w", addr, err)
	}
	switch len(candidates) {
	case 0:
		return Member{}, fmt.Errorf("resolving %q in room %q: %w", addr, caller.Room, ErrNotFound)
	case 1:
		return candidates[0], nil
	default:
		teams := make([]string, len(candidates))
		for i, c := range candidates {
			teams[i] = c.Team
		}
		return Member{}, fmt.Errorf(
			"resolving %q: teams %s all have a member named %q, address it as %q: %w",
			addr, strings.Join(teams, ", "), addr, "<team>/"+addr, ErrAmbiguousName)
	}
}

func memberByToken(ctx context.Context, q queryer, token string) (Member, error) {
	row := q.QueryRowContext(ctx,
		`SELECT `+memberColumns+` FROM members WHERE token = ?`, token)
	m, err := scanMember(row)
	if err != nil {
		return Member{}, err
	}
	return m, nil
}

func memberByName(ctx context.Context, q queryer, room, team, name string) (Member, error) {
	row := q.QueryRowContext(ctx,
		`SELECT `+memberColumns+` FROM members WHERE room = ? AND team = ? AND name = ?`,
		room, team, name)
	return scanMember(row)
}

func scanMember(row *sql.Row) (Member, error) {
	var m Member
	err := row.Scan(&m.Token, &m.Name, &m.Team, &m.Room, &m.Kind, &m.State)
	if errors.Is(err, sql.ErrNoRows) {
		return Member{}, ErrNotFound
	}
	if err != nil {
		return Member{}, fmt.Errorf("reading member row: %w", err)
	}
	return m, nil
}

func queryMembers(ctx context.Context, q queryer, query string, args ...any) ([]Member, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var members []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.Token, &m.Name, &m.Team, &m.Room, &m.Kind, &m.State); err != nil {
			return nil, fmt.Errorf("reading member row: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return members, nil
}
