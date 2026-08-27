package chat

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	// The pure-Go driver keeps the daemon cgo-free.
	_ "modernc.org/sqlite"

	"github.com/ngicks/crabswarm/crabswarm/chat/internal/chatdb"
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

// MemberKind tells how a member entered the store, which decides whether the
// daemon may reap it: an agent's token comes from the team-info provider and
// stops resolving once the agent is gone, while a human's token is issued by
// the daemon itself and no provider ever knows it.
type MemberKind string

const (
	// KindAgent is a provider-originated agent session.
	KindAgent MemberKind = "agent"
	// KindHuman is a member registered through an admin RPC.
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
	// as opaque: it is a provider-reported session id for [KindAgent] and a
	// daemon-issued secret for [KindHuman].
	Token string
	// Name is the member's display name, unique within Team.
	Name string
	// Team is the name namespace the member belongs to.
	Team string
	// Room is the space whose members can address each other.
	Room string
	// Kind records how the member entered the store.
	Kind MemberKind
	// State is the last harness state reported for the member.
	State MemberState
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

// Store is the persistent room/member/inbox state, backed by SQLite. It is
// safe for concurrent use.
type Store struct {
	db *sql.DB
	q  *chatdb.Queries
}

//go:generate sqlc generate

// schema is the DDL the store is built from. Embedding the same file sqlc
// reads keeps the runtime tables and the generated queries from drifting: a
// column added here is a compile error in the generated code until the
// queries follow.
//
//go:embed schema.sql
var schema string

// NewStore opens the SQLite database at path, creating it and its schema when
// missing, and returns a store ready for use. path is used as given — "~" is
// not expanded, that belongs to the configuration layer — except for
// ":memory:", which opens a private in-memory database.
//
// The caller must [Store.Close] the returned store.
func NewStore(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("opening chat store %q: %w", path, err)
	}
	// One connection: the daemon is a single low-traffic writer, so
	// serializing costs nothing, it keeps SQLITE_BUSY off the table, and it
	// makes ":memory:" behave as one database instead of one per pooled
	// connection.
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(ctx, schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("creating chat store schema in %q: %w", path, err)
	}
	return &Store{db: db, q: chatdb.New(db)}, nil
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
func (s *Store) tx(ctx context.Context, fn func(q *chatdb.Queries) error) error {
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
