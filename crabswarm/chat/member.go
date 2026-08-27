package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/ngicks/crabswarm/crabswarm/chat/internal/chatdb"
)

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
	err := s.tx(ctx, func(q *chatdb.Queries) error {
		existing, err := memberByToken(ctx, q, m.Token)
		switch {
		case err == nil:
			joined = existing
			return nil
		case !errors.Is(err, ErrNotFound):
			return err
		}
		if _, err := memberByName(ctx, q, m.Room, m.Team, m.Name); err == nil {
			return fmt.Errorf("joining room %q as %q: %w",
				m.Room, m.Team+"/"+m.Name, ErrNameTaken)
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		err = q.InsertMember(ctx, chatdb.InsertMemberParams{
			Token: m.Token,
			Name:  m.Name,
			Team:  m.Team,
			Room:  m.Room,
			Kind:  string(m.Kind),
			State: string(m.State),
		})
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
	return memberByToken(ctx, s.q, token)
}

// SetState records the harness state token last reported. Reading it back is
// [Store.Member].
func (s *Store) SetState(ctx context.Context, token string, state MemberState) error {
	switch state {
	case StateDone, StateWorking, StateWaiting:
	default:
		return fmt.Errorf("setting state of %q: unknown state %q", token, state)
	}
	n, err := s.q.SetMemberState(ctx, chatdb.SetMemberStateParams{
		State: string(state),
		Token: token,
	})
	if err != nil {
		return fmt.Errorf("setting state of %q: %w", token, err)
	}
	if n == 0 {
		return fmt.Errorf("setting state of %q: %w", token, ErrNotFound)
	}
	return nil
}

// ListMembers returns every member of room, ordered by team then name.
func (s *Store) ListMembers(ctx context.Context, room string) ([]Member, error) {
	rows, err := s.q.ListRoomMembers(ctx, room)
	if err != nil {
		return nil, fmt.Errorf("listing members of room %q: %w", room, err)
	}
	return membersOf(rows), nil
}

// ListRooms returns every room with its teams and members, all ordered by
// name. It is the whole store, for an operator to inspect.
func (s *Store) ListRooms(ctx context.Context) ([]Room, error) {
	rows, err := s.q.ListAllMembers(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing rooms: %w", err)
	}
	var rooms []Room
	for _, m := range membersOf(rows) {
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
	err := s.tx(ctx, func(q *chatdb.Queries) error {
		m, err := memberByToken(ctx, q, token)
		if err != nil {
			return fmt.Errorf("removing member: %w", err)
		}
		// Pending messages go through the ON DELETE CASCADE on messages.
		if err := q.DeleteMember(ctx, token); err != nil {
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
	err := s.tx(ctx, func(q *chatdb.Queries) error {
		m, err := memberByToken(ctx, q, token)
		if err != nil {
			return fmt.Errorf("moving member: %w", err)
		}
		moved, err = moveMember(ctx, q, m, team)
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
	err := s.tx(ctx, func(q *chatdb.Queries) error {
		m, err := memberByName(ctx, q, room, team, name)
		if err != nil {
			return fmt.Errorf("moving %q in room %q: %w", team+"/"+name, room, err)
		}
		moved, err = moveMember(ctx, q, m, toTeam)
		return err
	})
	if err != nil {
		return Member{}, err
	}
	return moved, nil
}

// moveMember implements the move itself for members already read inside the
// caller's transaction.
func moveMember(
	ctx context.Context,
	q *chatdb.Queries,
	m Member,
	team string,
) (Member, error) {
	if err := validateName(team, m.Name); err != nil {
		return Member{}, fmt.Errorf("moving member %q: %w", m.Name, err)
	}
	if m.Team == team {
		return m, nil
	}
	if _, err := memberByName(ctx, q, m.Room, team, m.Name); err == nil {
		return Member{}, fmt.Errorf("moving %q into team %q: %w", m.Name, team, ErrNameTaken)
	} else if !errors.Is(err, ErrNotFound) {
		return Member{}, err
	}
	err := q.SetMemberTeam(ctx, chatdb.SetMemberTeamParams{Team: team, Token: m.Token})
	if err != nil {
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
	return resolve(ctx, s.q, callerToken, addr)
}

// resolve implements [Store.Resolve] against any queries handle so a send can
// resolve and deliver inside one transaction.
func resolve(ctx context.Context, q *chatdb.Queries, callerToken, addr string) (Member, error) {
	caller, err := memberByToken(ctx, q, callerToken)
	if err != nil {
		return Member{}, fmt.Errorf("resolving %q: %w", addr, err)
	}
	return resolveFor(ctx, q, caller, addr)
}

// resolveFor resolves addr as caller sees it, for callers that already hold
// the member.
func resolveFor(
	ctx context.Context,
	q *chatdb.Queries,
	caller Member,
	addr string,
) (Member, error) {
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

	rows, err := q.MembersByRoomAndName(ctx, chatdb.MembersByRoomAndNameParams{
		Room: caller.Room,
		Name: addr,
	})
	if err != nil {
		return Member{}, fmt.Errorf("resolving %q: %w", addr, err)
	}
	candidates := membersOf(rows)
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

func memberByToken(ctx context.Context, q *chatdb.Queries, token string) (Member, error) {
	return memberFrom(q.MemberByToken(ctx, token))
}

func memberByName(ctx context.Context, q *chatdb.Queries, room, team, name string) (Member, error) {
	return memberFrom(q.MemberByName(ctx, chatdb.MemberByNameParams{
		Room: room,
		Team: team,
		Name: name,
	}))
}

// memberFrom adapts a single-row lookup, mapping the driver's "no rows" onto
// [ErrNotFound] so every lookup reports a missing member the same way.
func memberFrom(row chatdb.Member, err error) (Member, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return Member{}, ErrNotFound
	}
	if err != nil {
		return Member{}, fmt.Errorf("reading member row: %w", err)
	}
	return memberOf(row), nil
}

// memberOf converts a stored row into a [Member]. Kind and State are stored as
// the string values of their named types.
func memberOf(row chatdb.Member) Member {
	return Member{
		Token: row.Token,
		Name:  row.Name,
		Team:  row.Team,
		Room:  row.Room,
		Kind:  MemberKind(row.Kind),
		State: MemberState(row.State),
	}
}

// membersOf converts queried rows into members, staying nil for an empty
// result the way the queries themselves do.
func membersOf(rows []chatdb.Member) []Member {
	var members []Member
	for _, row := range rows {
		members = append(members, memberOf(row))
	}
	return members
}
