package chat

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"
)

// sentAt is a fixed send time; RFC3339Nano round-trips it exactly.
var sentAt = time.Date(2026, 8, 27, 10, 30, 0, 0, time.UTC)

// newTestStore opens a store on a temp file. A file, not ":memory:", so a test
// can close and reopen the same database.
func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "chat.db")
	s, err := NewStore(t.Context(), path)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s, path
}

// join adds a provider-originated member, failing the test on error.
func join(t *testing.T, s *Store, token, room, team, name string) Member {
	t.Helper()
	m, err := s.Join(t.Context(), Member{
		Token: token,
		Name:  name,
		Team:  team,
		Room:  room,
		Kind:  KindAgent,
	})
	assert.NilError(t, err)
	return m
}

// countRows is a white-box count of a table, for asserting that nothing was
// duplicated or left behind.
func countRows(t *testing.T, s *Store, query string, args ...any) int {
	t.Helper()
	var n int
	assert.NilError(t, s.db.QueryRowContext(t.Context(), query, args...).Scan(&n))
	return n
}

func TestStore_OpenCreatesSchema(t *testing.T) {
	s, path := newTestStore(t)

	// A fresh database has no members and no rooms.
	rooms, err := s.ListRooms(t.Context())
	assert.NilError(t, err)
	assert.Equal(t, len(rooms), 0)

	// Opening an existing database again is not an error either.
	again, err := NewStore(t.Context(), path)
	assert.NilError(t, err)
	assert.NilError(t, again.Close())
}

func TestStore_OpenAppliesPragmas(t *testing.T) {
	s, _ := newTestStore(t)

	var journal string
	assert.NilError(t, s.db.QueryRowContext(t.Context(), `PRAGMA journal_mode`).Scan(&journal))
	assert.Equal(t, journal, "wal")

	var busyTimeout int
	assert.NilError(t, s.db.QueryRowContext(t.Context(), `PRAGMA busy_timeout`).Scan(&busyTimeout))
	assert.Equal(t, busyTimeout, 5000)

	// Foreign keys are what drop an inbox with its member.
	var foreignKeys int
	assert.NilError(t, s.db.QueryRowContext(t.Context(), `PRAGMA foreign_keys`).Scan(&foreignKeys))
	assert.Equal(t, foreignKeys, 1)
}

func TestStore_PersistsAcrossReopen(t *testing.T) {
	s, path := newTestStore(t)
	alice := join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	join(t, s, "tok-b", "/work/repo", "beta", "bob")
	_, err := s.Send(t.Context(), "tok-b", "alpha/alice", "still here?", sentAt)
	assert.NilError(t, err)
	assert.NilError(t, s.SetState(t.Context(), alice.Token, StateWorking))
	assert.NilError(t, s.Close())

	reopened, err := NewStore(t.Context(), path)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = reopened.Close() })

	// Members, their state, and the room layout survive.
	got, err := reopened.Member(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.DeepEqual(t, got, Member{
		Token: "tok-a", Name: "alice", Team: "alpha", Room: "/work/repo",
		Kind: KindAgent, State: StateWorking,
	})
	rooms, err := reopened.ListRooms(t.Context())
	assert.NilError(t, err)
	assert.Equal(t, len(rooms), 1)
	assert.Equal(t, len(rooms[0].Teams), 2)

	// So do pending messages, timestamp included.
	msgs, err := reopened.Read(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 1)
	assert.Equal(t, msgs[0].Text, "still here?")
	assert.Assert(t, msgs[0].SentAt.Equal(sentAt))
}

func TestStore_InMemory(t *testing.T) {
	s, err := NewStore(t.Context(), ":memory:")
	assert.NilError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	// The single connection makes this one database rather than one per
	// pooled connection, so a write is visible to the next read.
	join(t, s, "tok-a", "/work/repo", "alpha", "alice")
	join(t, s, "tok-b", "/work/repo", "alpha", "bob")
	_, err = s.Send(t.Context(), "tok-a", "bob", "hi", sentAt)
	assert.NilError(t, err)
	msgs, err := s.Read(t.Context(), "tok-b")
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 1)
}

func TestStore_ConcurrentUse(t *testing.T) {
	s, _ := newTestStore(t)
	join(t, s, "tok-reader", "/work/repo", "alpha", "reader")

	const senders = 8
	var eg errgroup.Group
	for i := range senders {
		eg.Go(func() error {
			name := fmt.Sprintf("sender-%d", i)
			if _, err := s.Join(t.Context(), Member{
				Token: "tok-" + name, Name: name, Team: "alpha",
				Room: "/work/repo", Kind: KindAgent,
			}); err != nil {
				return err
			}
			_, err := s.Send(t.Context(), "tok-"+name, "reader", name, sentAt)
			return err
		})
	}
	assert.NilError(t, eg.Wait())

	members, err := s.ListMembers(t.Context(), "/work/repo")
	assert.NilError(t, err)
	assert.Equal(t, len(members), senders+1)
	msgs, err := s.Read(t.Context(), "tok-reader")
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), senders)
}

func TestStore_ClosedStoreFails(t *testing.T) {
	s, _ := newTestStore(t)
	assert.NilError(t, s.Close())

	_, err := s.ListRooms(context.Background())
	assert.Assert(t, err != nil, "using a closed store should fail")
}
