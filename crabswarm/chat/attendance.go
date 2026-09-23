package chat

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/ngicks/crabswarm/crabswarm/chat/internal/db"
)

// Attend records m as attending its room and returns it as recorded.
//
// A token already attending is [ErrAlreadyAttending]: one token is one session,
// and a second declaration for it is a client that lost track of the first
// rather than a second participant. So is a second session under a role
// somebody is already attending under, for the reason spelled out where the
// check is made.
//
// Attending creates the room when it is new and gives the role a read position
// at the room's current last_seq when it has none, which is what leaves a role
// attending for the first time with nothing unread instead of the whole
// retained backlog. A role that has attended before keeps the position it left
// with, so what arrived while it was away is waiting for it.
//
// An empty State defaults to [StateDone]: attendance is declared as the
// member's MCP server opens its stream to the room, before the session has work
// to do. A zero StateReportedAt defaults to now, the moment that state was
// declared.
//
// An agent with an empty Nudge defaults to [NudgeTerminal], which is how every
// agent was reached before a harness could deliver a mention itself. A member
// of any other kind keeps the empty value: nothing is ever typed at it, so
// there is no delivery to name.
func (s *Store) Attend(ctx context.Context, m Member) (Member, error) {
	if m.Token == "" {
		return Member{}, fmt.Errorf("attending chat: empty token")
	}
	if m.Room == "" {
		return Member{}, fmt.Errorf("attending chat: empty room")
	}
	if m.Kind != KindAgent && m.Kind != KindHuman {
		return Member{}, fmt.Errorf("attending chat: unknown member kind %q", m.Kind)
	}
	if err := validateName(m.Team, m.Name); err != nil {
		return Member{}, fmt.Errorf("attending chat: %w", err)
	}
	if m.Kind == KindAgent && m.Nudge == "" {
		m.Nudge = NudgeTerminal
	}
	if m.State == "" {
		m.State = StateDone
	}
	if m.StateReportedAt.IsZero() {
		m.StateReportedAt = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.attending[m.Token]; ok {
		return Member{}, fmt.Errorf("attending room %q as %q: %w",
			m.Room, m.Team+"/"+m.Name, ErrAlreadyAttending)
	}
	// One read position per role cannot serve two live sessions: the position
	// is a single cursor, so whichever session read last would mark what the
	// other has not seen as shown, and each would lose messages meant for it.
	if _, ok := s.attendingAs(senderOf(m)); ok {
		return Member{}, fmt.Errorf("attending room %q as %q: %w",
			m.Room, m.Team+"/"+m.Name, ErrAlreadyAttending)
	}
	err := s.tx(ctx, func(q *db.Queries) error {
		if err := q.EnsureRoom(ctx, m.Room); err != nil {
			return fmt.Errorf("recording room %q: %w", m.Room, err)
		}
		err := q.SeedReadPosition(ctx, db.SeedReadPositionParams{
			Room: m.Room,
			Team: m.Team,
			Name: m.Name,
		})
		if err != nil {
			return fmt.Errorf("seeding read position of %q in room %q: %w",
				m.Team+"/"+m.Name, m.Room, err)
		}
		return nil
	})
	if err != nil {
		return Member{}, err
	}
	s.attending[m.Token] = m
	return m, nil
}

// Detach withdraws the attendance held under token and returns the member that
// held it. An unknown token is [ErrNotAttending].
//
// Nothing persistent changes: the role, its read position and the room's
// messages are all still there. Detaching says the session is gone, not that
// the role is.
func (s *Store) Detach(_ context.Context, token string) (Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.attending[token]
	if !ok {
		return Member{}, fmt.Errorf("detaching: %w", ErrNotAttending)
	}
	delete(s.attending, token)
	return m, nil
}

// Member returns the member attending under token, or [ErrNotAttending].
func (s *Store) Member(_ context.Context, token string) (Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.attending[token]
	if !ok {
		return Member{}, fmt.Errorf("looking up attendance: %w", ErrNotAttending)
	}
	return m, nil
}

// SetState records the harness state token last reported, as of reportedAt.
// Reading both back is [Store.Member]. An unknown token is [ErrNotAttending].
func (s *Store) SetState(
	_ context.Context,
	token string,
	state MemberState,
	reportedAt time.Time,
) error {
	switch state {
	case StateDone, StateWorking, StateWaiting:
	default:
		return fmt.Errorf("setting state of %q: unknown state %q", token, state)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.attending[token]
	if !ok {
		return fmt.Errorf("setting state of %q: %w", token, ErrNotAttending)
	}
	m.State = state
	m.StateReportedAt = reportedAt
	s.attending[token] = m
	return nil
}

// Attending returns everyone attending the daemon right now, every room
// together, ordered by team then name.
//
// It is what a watcher with no room of its own walks. The screen poller is that
// watcher: it serves no caller who sits in one room, and it reads the terminal
// of every attending member whose harness reports on no feed of its own. A
// watcher that listed the rooms first would have to ask the log for names
// nobody attends.
func (s *Store) Attending(_ context.Context) ([]Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	members := make([]Member, 0, len(s.attending))
	for _, m := range s.attending {
		members = append(members, m)
	}
	sortMembers(members)
	return members, nil
}

// ListMembers returns everyone attending room, ordered by team then name.
func (s *Store) ListMembers(_ context.Context, room string) ([]Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.membersOf(room), nil
}

// membersOf collects the attendance of one room in display order. The caller
// holds mu.
func (s *Store) membersOf(room string) []Member {
	var members []Member
	for _, m := range s.attending {
		if m.Room == room {
			members = append(members, m)
		}
	}
	sortMembers(members)
	return members
}

// attendingAs returns the member attending under the role, and whether anyone
// is. At most one ever does, so the answer does not depend on map iteration
// order: [Store.Attend] turns a second session for an attended role away. The
// caller holds mu.
func (s *Store) attendingAs(role Sender) (Member, bool) {
	for _, m := range s.attending {
		if m.Room == role.Room && m.Team == role.Team && m.Name == role.Name {
			return m, true
		}
	}
	return Member{}, false
}

// sortMembers puts members in the order a room is displayed in: by team, then
// by name, then by token so equal roles keep a stable order.
func sortMembers(members []Member) {
	slices.SortFunc(members, func(a, b Member) int {
		return cmp.Or(
			cmp.Compare(a.Team, b.Team),
			cmp.Compare(a.Name, b.Name),
			cmp.Compare(a.Token, b.Token),
		)
	})
}
