package chat

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"
)

// testRoom is the room nearly every store case plays out in.
const testRoom = "/work/repo"

// sentAt is a fixed send time; RFC3339Nano round-trips it exactly.
var sentAt = time.Date(2026, 8, 27, 10, 30, 0, 0, time.UTC)

// reportedAt is a fixed state-report time, a different instant from [sentAt] so
// a test that swapped the two would not still pass.
var reportedAt = time.Date(2026, 8, 27, 11, 45, 0, 0, time.UTC)

// newTestStore opens a store on a temp file, keeping history at the default
// cap. A file, not ":memory:", so a test can close and reopen the same
// database and see what a daemon restart sees.
func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	return newTestStoreWithHistory(t, 0)
}

// newTestStoreWithHistory is [newTestStore] with the message cap chosen, for
// the cases that assert on pruning or on pruning being switched off.
func newTestStoreWithHistory(t *testing.T, historyLimit int) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "chat.db")
	s, err := NewStore(t.Context(), path, historyLimit)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s, path
}

// reopen closes s and opens the same database again, which is what a daemon
// restart looks like to the store.
func reopen(t *testing.T, s *Store, path string) *Store {
	t.Helper()
	assert.NilError(t, s.Close())
	again, err := NewStore(t.Context(), path, 0)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = again.Close() })
	return again
}

// attend puts an agent in a room, failing the test on error.
func attend(t *testing.T, s *Store, token, room, team, name string) Member {
	t.Helper()
	m, err := s.Attend(t.Context(), Member{
		Token: token,
		Name:  name,
		Team:  team,
		Room:  room,
		Kind:  KindAgent,
	})
	assert.NilError(t, err)
	return m
}

// role names a target without a room; resolution takes the room from the
// sender.
func role(team, name string) Sender {
	return Sender{Team: team, Name: name}
}

// toRoles is the explicit-list target.
func toRoles(rs ...Sender) Target {
	return Target{Kind: TargetRoles, Roles: rs}
}

// send is [Store.Send] with the test's fixed timestamp, failing on error.
func send(t *testing.T, s *Store, from Sender, target Target, text string) Sent {
	t.Helper()
	sent, err := s.Send(t.Context(), from, target, text, sentAt)
	assert.NilError(t, err)
	return sent
}

// texts is what a read returned, for comparing a whole window at once.
func texts(messages []Message) []string {
	out := make([]string, len(messages))
	for i, m := range messages {
		out[i] = m.Text
	}
	return out
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

	// A fresh database has no rooms and nobody attending.
	rooms, err := s.ListRooms(t.Context())
	assert.NilError(t, err)
	assert.Equal(t, len(rooms), 0)

	// Opening an existing database again is not an error either.
	again, err := NewStore(t.Context(), path, 0)
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

	// Foreign keys are what take a room's messages and read positions with it.
	var foreignKeys int
	assert.NilError(t, s.db.QueryRowContext(t.Context(), `PRAGMA foreign_keys`).Scan(&foreignKeys))
	assert.Equal(t, foreignKeys, 1)
}

func TestStore_KeepsNoTokens(t *testing.T) {
	s, _ := newTestStore(t)
	attend(t, s, "tok-secret", testRoom, "alpha", "alice")

	// A session is a fact about this run of the daemon, not about the room, so
	// no column anywhere may hold one.
	var found int
	rows, err := s.db.QueryContext(t.Context(),
		`SELECT name, sql FROM sqlite_master WHERE type = 'table'`)
	assert.NilError(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name, ddl string
		assert.NilError(t, rows.Scan(&name, &ddl))
		assert.Assert(t, !strings.Contains(strings.ToLower(ddl), "token"),
			"table %q has a token column: %s", name, ddl)
		found++
	}
	assert.NilError(t, rows.Err())
	assert.Assert(t, found > 0)
}

func TestStore_InMemory(t *testing.T) {
	s, err := NewStore(t.Context(), ":memory:", 0)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	// The single connection makes this one database rather than one per
	// pooled connection, so a write is visible to the next read.
	alice := attend(t, s, "tok-a", testRoom, "alpha", "alice")
	bob := attend(t, s, "tok-b", testRoom, "alpha", "bob")
	send(t, s, senderOf(alice), toRoles(role("", "bob")), "hi")

	msgs, _, err := s.Read(t.Context(), senderOf(bob), ReadFilter{})
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 1)
}

func TestStore_ConcurrentUse(t *testing.T) {
	s, _ := newTestStore(t)
	reader := attend(t, s, "tok-reader", testRoom, "alpha", "reader")

	const senders = 8
	var eg errgroup.Group
	for i := range senders {
		eg.Go(func() error {
			name := fmt.Sprintf("sender-%d", i)
			from, err := s.Attend(t.Context(), Member{
				Token: "tok-" + name, Name: name, Team: "alpha",
				Room: testRoom, Kind: KindAgent,
			})
			if err != nil {
				return err
			}
			_, err = s.Send(t.Context(), senderOf(from),
				toRoles(role("alpha", "reader")), name, sentAt)
			return err
		})
	}
	assert.NilError(t, eg.Wait())

	members, err := s.ListMembers(t.Context(), testRoom)
	assert.NilError(t, err)
	assert.Equal(t, len(members), senders+1)

	// Every send took its own seq: the room numbers one to eight with no gap
	// and no repeat.
	assert.Equal(t, countRows(t, s, `SELECT COUNT(*) FROM messages`), senders)
	assert.Equal(t, countRows(t, s, `SELECT COUNT(DISTINCT seq) FROM messages`), senders)
	assert.Equal(t, countRows(t, s, `SELECT MAX(seq) FROM messages`), senders)

	msgs, remaining, err := s.Read(t.Context(), senderOf(reader),
		ReadFilter{Range: senders})
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), senders)
	assert.Equal(t, remaining, 0)
}

func TestStore_ClosedStoreFails(t *testing.T) {
	s, _ := newTestStore(t)
	assert.NilError(t, s.Close())

	_, err := s.ListRooms(context.Background())
	assert.Assert(t, err != nil, "using a closed store should fail")
}
