package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	// The pure-Go driver keeps the daemon cgo-free.
	_ "modernc.org/sqlite"

	"github.com/ngicks/crabswarm/crabswarm/chat/internal/db"
	"github.com/ngicks/crabswarm/crabswarm/chat/internal/schema"
)

// Sentinel errors the transport layer maps to status codes. Every returned
// error wraps one of these with context, so match with [errors.Is].
var (
	// ErrNotFound reports that no member matches the token or the address.
	ErrNotFound = errors.New("member not found")
	// ErrNameTaken reports that another member of the same team already uses
	// the name. Names are unique per team, not per room.
	ErrNameTaken = errors.New("name already taken in this team")
	// ErrAmbiguousName reports that a bare name matches members in more than
	// one other team of the room, so the address needs the "team/name" form.
	ErrAmbiguousName = errors.New("ambiguous name")
	// ErrInvalidName reports a name or team containing "/", which would make
	// the "team/name" address grammar unparseable.
	ErrInvalidName = errors.New("invalid name")
)

// MemberKind tells what the daemon may do to a member besides handing it its
// inbox. An agent said it runs a harness, so its terminal is typed into, its
// command carries the state display, and its attendance lasts only as long as
// the team-info provider still places its token. Anything else is left alone:
// nothing is injected, nothing is published, and no provider verdict takes its
// membership away.
//
// Which one a joiner is, it declares — the daemon cannot tell a harness from a
// shell that happens to run under the same command.
type MemberKind string

const (
	// KindAgent is an agent harness; nudgeable by keystroke injection.
	KindAgent MemberKind = "agent"
	// KindHuman is any other member (plain shell, admin-registered);
	// inbox-only.
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

// Member is one participant of a room.
type Member struct {
	// Token identifies the member across the whole store. The store treats it
	// as opaque: whoever joined from a command the team-info provider knows
	// presents the session id it reports, and whoever an admin registered
	// carries a secret the daemon minted. Which of the two it is follows from
	// how the member joined, not from [Member.Kind] — a person joining by hand
	// from a plain shell presents a provider-reported token too.
	Token string
	// Name is the member's display name, unique within Team.
	Name string
	// Team is the name namespace the member belongs to.
	Team string
	// Room is the space whose members can address each other.
	Room string
	// Kind is what the joiner declared it is, and so what may be done to the
	// member besides handing it its inbox — being nudged by keystroke
	// injection above all. See [MemberKind].
	Kind MemberKind
	// State is the last harness state reported for the member.
	State MemberState
	// StateReportedAt is when State was reported. A notifier reads it to tell
	// a member that is genuinely busy from one whose state-reporting hook was
	// missed — an interrupted session, or a harness that has no idle
	// notification at all — and would otherwise stay busy forever.
	StateReportedAt time.Time
}

// Sender is the identity a message carries: who sent it, as of send time. It
// is a snapshot rather than a token reference so a delivered message still
// reads correctly after the sender leaves or moves team.
type Sender struct {
	Name string
	Team string
	Room string
}

// Message is one delivered message waiting in a member's inbox.
type Message struct {
	From   Sender
	Text   string
	SentAt time.Time
}

// Team is a name namespace within a room, with the members that occupy it.
type Team struct {
	Name    string
	Members []Member
}

// Room is a whole room as an admin sees it.
type Room struct {
	Name  string
	Teams []Team
}

// defaultHistoryLimit is how many conversation rows a room keeps when the
// configuration names no cap. A thousand utterances is far more than a room
// re-reads and still a bounded database.
const defaultHistoryLimit = 1000

// Store is the persistent room state — its members, their inboxes and the
// conversation log — backed by SQLite. It is safe for concurrent use.
type Store struct {
	db *sql.DB
	q  *db.Queries
	// historyLimit is the per-room row cap of the conversation log, already
	// resolved: positive is the cap, negative means nothing is logged at all.
	historyLimit int
	// events carries what changed to the watchers of the room it changed in.
	// It hangs on the store rather than on either service because both of them
	// mutate the same rooms — a member leaving and an operator moving one are
	// the same news to a watcher — and the store is where the two meet.
	//
	// The store itself never publishes: an event must announce a mutation that
	// has already persisted, which is only known one call up.
	events *roomBroadcaster
}

// NewStore opens the SQLite database at path, creating it and its schema when
// missing, and returns a store ready for use. path is used as given — "~" is
// not expanded, that belongs to the configuration layer — except for
// ":memory:", which opens a private in-memory database.
//
// historyLimit caps how many conversation rows each room keeps: zero means the
// default, a negative value records no conversation at all. Zero is resolved
// here rather than by the caller so that every caller of an unconfigured store
// keeps its history instead of pruning every row it writes.
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
	}, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error {
	return s.db.Close()
}

// dsn builds the driver DSN for path. WAL and a busy timeout are set even
// though a single daemon writes: a stray reader (sqlite3 CLI, a second daemon
// racing the flock) must not turn into a locked database. Foreign keys are on
// so removing a member drops their inbox with them.
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
// by id rather than by this text, which would sort wrong — RFC3339Nano drops
// trailing zeros from the fraction, so ".5Z" and "Z" do not compare as their
// instants do.
func formatTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// validateName rejects the "/" that separates team from name in an address,
// and the name the host operator sends under.
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
	case name == adminName:
		return fmt.Errorf("name %q is reserved for the host operator: %w",
			name, ErrInvalidName)
	case team == adminName:
		// A team named admin would win bare-name resolution for admin sends
		// (the resolver tries the sender's own team first) and render members
		// as admin/<name>, next door to the reserved attribution.
		return fmt.Errorf("team %q is reserved for the host operator: %w",
			team, ErrInvalidName)
	}
	return nil
}
