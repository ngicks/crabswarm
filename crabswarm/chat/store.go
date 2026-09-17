package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	// The pure-Go driver keeps the daemon cgo-free.
	_ "modernc.org/sqlite"

	"github.com/ngicks/crabswarm/crabswarm/chat/internal/db"
	"github.com/ngicks/crabswarm/crabswarm/chat/internal/schema"
)

// Sentinel errors the transport layer maps to status codes. Every returned
// error wraps one of these with context, so match with [errors.Is].
var (
	// ErrNotFound reports that no room carries the given name.
	ErrNotFound = errors.New("room not found")
	// ErrInvalidName reports a name or team that is empty or contains "/",
	// which would make the "team/name" address grammar unparseable.
	ErrInvalidName = errors.New("invalid name")
	// ErrInvalidArgument reports a request that cannot be carried out as
	// written: a read range running against its cursor, an unread cursor on a
	// room read, a roles target naming nobody.
	ErrInvalidArgument = errors.New("invalid argument")
	// ErrAlreadyAttending reports a second attendance for a token that is
	// already attending. One token is one session, and one session attends once.
	ErrAlreadyAttending = errors.New("already attending")
	// ErrNotAttending reports that no attendance is held under the token.
	ErrNotAttending = errors.New("not attending")
	// ErrUnknownRole reports a target naming a role the room has never seen. A
	// role exists once it has attended once, and only then can it be addressed.
	ErrUnknownRole = errors.New("unknown role")
	// ErrAmbiguousRole reports a bare name carried by two or more teams of the
	// room, so the target needs the "team/name" form. The error names the teams.
	ErrAmbiguousRole = errors.New("ambiguous role")
	// ErrRoomAttended reports a room operation refused because somebody is
	// attending the room.
	ErrRoomAttended = errors.New("room is attended")
)

// MemberKind tells what the daemon may do to a member besides showing it the
// room. An agent said it runs a harness, so it is nudged when it is mentioned
// and its command carries the state display. Anything else is left alone:
// nothing is delivered to it and nothing is published.
//
// By what route an agent is nudged is [NudgeDelivery], declared separately: a
// keystroke into its terminal, or its own server pushing the mention through
// the harness.
//
// Which one a joiner is, it declares — the daemon cannot tell a harness from a
// shell that happens to run under the same command.
type MemberKind string

const (
	// KindAgent is an agent harness; nudgeable, by whichever route its
	// [NudgeDelivery] names.
	KindAgent MemberKind = "agent"
	// KindHuman is any other member (plain shell, admin-registered); read-only
	// as far as nudging goes.
	KindHuman MemberKind = "human"
)

// MemberState is the harness state a member last reported. Notifiers use it to
// decide whether the member can be interrupted.
type MemberState string

// State names mirror the vocabulary of `cmdman status set working|waiting|done`.
const (
	// StateWorking means the harness is working on a turn.
	StateWorking MemberState = "working"
	// StateWaiting means the harness is blocked on a prompt or a
	// permission dialog.
	StateWaiting MemberState = "waiting"
	// StateDone means the harness finished its turn and is waiting for work.
	StateDone MemberState = "done"
)

// Member is one participant attending a room right now.
//
// Attendance is daemon state, not stored state: it lasts as long as the stream
// that declared it, and a restart starts the room empty. What survives a
// restart is the role — the team and name — which the room remembers through
// its read position.
type Member struct {
	// Token identifies the attendance across the whole store. The store treats
	// it as opaque: whoever attends from a command the team-info provider knows
	// presents the session id it reports, and whoever an admin registered
	// carries a secret the daemon minted.
	Token string
	// Name is the member's display name, unique within Team.
	Name string
	// Team is the name namespace the member belongs to.
	Team string
	// Room is the space whose members can address each other.
	Room string
	// Kind is what the joiner declared it is, and so what may be done to the
	// member besides showing it the room — being nudged by keystroke injection
	// above all. See [MemberKind].
	Kind MemberKind
	// Harness is the CLI the member declared it runs. See [Harness].
	Harness Harness
	// Nudge is how a mention is delivered to the member. See [NudgeDelivery].
	Nudge NudgeDelivery
	// State is the last harness state reported for the member.
	State MemberState
	// StateReportedAt is when State was reported. A notifier reads it to tell a
	// member that is genuinely busy from one whose state-reporting hook was
	// missed — an interrupted session, or a harness that has no idle
	// notification at all — and would otherwise stay busy forever.
	StateReportedAt time.Time
}

// Sender is a role in a room: who speaks or is spoken to, as of the moment it
// was written down. A message keeps it as a snapshot rather than a reference so
// the message still reads correctly after the sender stops attending.
//
// The host operator sends with an empty Team, having none.
type Sender struct {
	Name string
	Team string
	Room string
}

// Room is a room and who is attending it.
type Room struct {
	Name string
	// Members is everyone attending, ordered by team then name. A room that
	// exists in the log with nobody in it has none.
	Members []Member
}

// defaultHistoryLimit is how many messages a room keeps when the configuration
// names no cap. A thousand utterances is far more than a room re-reads and
// still a bounded database.
const defaultHistoryLimit = 1000

// Store is the chat broker's state: the rooms and their conversation in SQLite,
// and the attendance of the running daemon in memory. It is safe for concurrent
// use.
type Store struct {
	db *sql.DB
	q  *db.Queries
	// historyLimit is the per-room message cap, already resolved: positive is
	// the cap, negative prunes nothing.
	historyLimit int
	// events carries what changed to the watchers of the room it changed in.
	// It hangs on the store rather than on either service because both of them
	// mutate the same rooms, and the store is where the two meet.
	//
	// The store itself never publishes: an event must announce a mutation that
	// has already persisted, which is only known one call up.
	events *roomBroadcaster
	// mu guards attending. It may be held across a transaction — attendance
	// changes are rare and have to settle atomically against the read position
	// they seed — so nothing may take it from inside one: the store holds a
	// single connection, and a transaction waiting for the lock while the lock
	// holder waits for the connection would never resolve.
	mu sync.Mutex
	// attending is who is in which room right now, keyed by the token that
	// declared it. Tokens never reach SQLite: a session is not a fact about the
	// room, it is a fact about this run of the daemon.
	attending map[string]Member
}

// NewStore opens the SQLite database at path, creating it and its schema when
// missing, and returns a store ready for use. path is used as given — "~" is
// not expanded, that belongs to the configuration layer — except for
// ":memory:", which opens a private in-memory database.
//
// historyLimit caps how many messages each room keeps: zero means the default,
// a negative value prunes nothing at all. Zero is resolved here rather than by
// the caller so that every caller of an unconfigured store keeps a bounded
// history instead of pruning every row it writes.
//
// The returned store attends nobody: attendance belongs to the running daemon,
// so every restart starts every room empty while the rooms themselves, their
// messages and their read positions come back as they were.
//
// The caller must [Store.Close] the returned store.
func NewStore(ctx context.Context, path string, historyLimit int) (*Store, error) {
	if historyLimit == 0 {
		historyLimit = defaultHistoryLimit
	}
	conn, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("opening chat store %q: %w", path, err)
	}
	// One connection: the daemon is a single low-traffic writer, so
	// serializing costs nothing, it keeps SQLITE_BUSY off the table, and it
	// makes ":memory:" behave as one database instead of one per pooled
	// connection.
	conn.SetMaxOpenConns(1)
	// The tables come from the same files sqlc typed the queries against, so
	// the two cannot drift.
	if _, err := conn.ExecContext(ctx, schema.DDL()); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("creating chat store schema in %q: %w", path, err)
	}
	return &Store{
		db:           conn,
		q:            db.New(conn),
		historyLimit: historyLimit,
		events:       newRoomBroadcaster(),
		attending:    make(map[string]Member),
	}, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error {
	return s.db.Close()
}

// dsn builds the driver DSN for path. WAL and a busy timeout are set even
// though a single daemon writes: a stray reader (sqlite3 CLI, a second daemon
// racing the flock) must not turn into a locked database. Foreign keys are on
// so deleting a room takes its messages, their mentions and its read positions
// with it.
func dsn(path string) string {
	pragmas := url.Values{"_pragma": []string{
		"busy_timeout(5000)",
		"journal_mode(WAL)",
		"foreign_keys(1)",
	}}
	if path == ":memory:" {
		return "file::memory:?" + pragmas.Encode()
	}
	return "file:" + path + "?" + pragmas.Encode()
}

// tx runs fn in a transaction, committing when it returns nil and rolling back
// otherwise. fn works through queries bound to that transaction.
func (s *Store) tx(ctx context.Context, fn func(q *db.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	if err := fn(s.q.WithTx(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

// formatTimestamp renders t the way every timestamp column stores one:
// RFC3339Nano in UTC, which parses back to the same instant. Rows are ordered
// by seq rather than by this text, which would sort wrong — RFC3339Nano drops
// trailing zeros from the fraction, so ".5Z" and "Z" do not compare as their
// instants do.
func formatTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// parseTimestamp reads a stored timestamp back.
func parseTimestamp(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing message timestamp %q: %w", s, err)
	}
	return t, nil
}

// validateName rejects the "/" that separates team from name in an address.
func validateName(team, name string) error {
	switch {
	case name == "":
		return fmt.Errorf("empty name: %w", ErrInvalidName)
	case team == "":
		return fmt.Errorf("empty team: %w", ErrInvalidName)
	case strings.Contains(name, "/"):
		return fmt.Errorf("name %q contains %q: %w", name, "/", ErrInvalidName)
	case strings.Contains(team, "/"):
		return fmt.Errorf("team %q contains %q: %w", team, "/", ErrInvalidName)
	}
	return nil
}

// senderOf is the role an attending member speaks under.
func senderOf(m Member) Sender {
	return Sender{Name: m.Name, Team: m.Team, Room: m.Room}
}
