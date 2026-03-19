package cmdman

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store provides access to the SQLite database for command management.
type Store struct {
	db *sql.DB
}

// OpenStore opens the SQLite database at the given path, configuring WAL mode,
// busy timeout, and foreign keys. It creates the schema if needed.
func OpenStore(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := configureDB(db); err != nil {
		db.Close()
		return nil, err
	}

	if err := createSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	if err := verifyJSONSupport(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// DB returns the underlying *sql.DB.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func configureDB(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("exec %q: %w", p, err)
		}
	}
	return nil
}

func createSchema(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS CommandConfig (
    ID              TEXT PRIMARY KEY,
    Name            TEXT UNIQUE,
    JSON            TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_command_config_name ON CommandConfig(Name);

CREATE TABLE IF NOT EXISTS CommandState (
    ID              TEXT PRIMARY KEY,
    State           TEXT NOT NULL,
    ExitCode        INTEGER CHECK (ExitCode BETWEEN -1 AND 255),
    JSON            TEXT NOT NULL,
    FOREIGN KEY (ID) REFERENCES CommandConfig(ID)
        ON DELETE CASCADE
        DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX IF NOT EXISTS idx_command_state_state ON CommandState(State);

CREATE TABLE IF NOT EXISTS CommandExitCode (
    ID              TEXT NOT NULL,
    Timestamp       TEXT NOT NULL,
    ExitCode        INTEGER NOT NULL CHECK (ExitCode BETWEEN -1 AND 255),
    FOREIGN KEY (ID) REFERENCES CommandConfig(ID)
        ON DELETE CASCADE
        DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX IF NOT EXISTS idx_command_exit_code_id_ts ON CommandExitCode(ID, Timestamp);
`
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	return nil
}

func verifyJSONSupport(db *sql.DB) error {
	var result string
	err := db.QueryRow(`SELECT json_extract('{"a":"b"}', '$.a')`).Scan(&result)
	if err != nil {
		return fmt.Errorf("SQLite JSON support unavailable: %w", err)
	}
	if result != "b" {
		return fmt.Errorf("SQLite JSON support broken: expected %q, got %q", "b", result)
	}
	return nil
}

// Command states.
const (
	StateCreated  = "created"
	StateStarting = "starting"
	StateRunning  = "running"
	StateExited   = "exited"
	StateErrored  = "errored"
)

// InsertCommandConfig inserts a new CommandConfig row.
func (s *Store) InsertCommandConfig(id, name string, cfg *CommandConfigJSON) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO CommandConfig (ID, Name, JSON) VALUES (?, ?, ?)`,
		id, nullableString(name), string(data),
	)
	return err
}

// InsertCommandState inserts a new CommandState row.
func (s *Store) InsertCommandState(id, state string, stateJSON *CommandStateJSON) error {
	data, err := json.Marshal(stateJSON)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO CommandState (ID, State, ExitCode, JSON) VALUES (?, ?, NULL, ?)`,
		id, state, string(data),
	)
	return err
}

// UpdateCommandState updates the state and JSON of a CommandState row.
func (s *Store) UpdateCommandState(id, state string, exitCode *int, stateJSON *CommandStateJSON) error {
	data, err := json.Marshal(stateJSON)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`UPDATE CommandState SET State = ?, ExitCode = ?, JSON = ? WHERE ID = ?`,
		state, exitCode, string(data), id,
	)
	return err
}

// InsertCommandExitCode records an exit code for a command.
func (s *Store) InsertCommandExitCode(id string, exitCode int) error {
	_, err := s.db.Exec(
		`INSERT INTO CommandExitCode (ID, Timestamp, ExitCode) VALUES (?, ?, ?)`,
		id, time.Now().UTC().Format(time.RFC3339), exitCode,
	)
	return err
}

// GetCommandConfig retrieves a CommandConfig by ID or name.
func (s *Store) GetCommandConfig(idOrName string) (id, name string, cfg *CommandConfigJSON, err error) {
	var nameSQL sql.NullString
	var jsonStr string
	err = s.db.QueryRow(
		`SELECT ID, Name, JSON FROM CommandConfig WHERE ID = ? OR Name = ?`,
		idOrName, idOrName,
	).Scan(&id, &nameSQL, &jsonStr)
	if err != nil {
		return "", "", nil, err
	}
	if nameSQL.Valid {
		name = nameSQL.String
	}
	cfg = &CommandConfigJSON{}
	if err := json.Unmarshal([]byte(jsonStr), cfg); err != nil {
		return "", "", nil, err
	}
	return id, name, cfg, nil
}

// GetCommandState retrieves the CommandState for a command by ID.
func (s *Store) GetCommandState(id string) (state string, exitCode *int, stateJSON *CommandStateJSON, err error) {
	var ecSQL sql.NullInt64
	var jsonStr string
	err = s.db.QueryRow(
		`SELECT State, ExitCode, JSON FROM CommandState WHERE ID = ?`,
		id,
	).Scan(&state, &ecSQL, &jsonStr)
	if err != nil {
		return "", nil, nil, err
	}
	if ecSQL.Valid {
		ec := int(ecSQL.Int64)
		exitCode = &ec
	}
	stateJSON = &CommandStateJSON{}
	if err := json.Unmarshal([]byte(jsonStr), stateJSON); err != nil {
		return "", nil, nil, err
	}
	return state, exitCode, stateJSON, nil
}

// CommandEntry represents a joined row from CommandConfig and CommandState.
type CommandEntry struct {
	ID        string
	Name      string
	State     string
	ExitCode  *int
	ConfigJSON *CommandConfigJSON
	StateJSON  *CommandStateJSON
}

// ListCommands lists commands, optionally filtering by state and labels.
func (s *Store) ListCommands(allStates bool, labels map[string]string) ([]CommandEntry, error) {
	query := `SELECT c.ID, c.Name, s.State, s.ExitCode, c.JSON, s.JSON
		FROM CommandConfig c
		JOIN CommandState s ON c.ID = s.ID`

	var args []any
	var conditions []string

	if !allStates {
		conditions = append(conditions, `s.State IN ('created', 'starting', 'running')`)
	}

	for k, v := range labels {
		conditions = append(conditions, `json_extract(c.JSON, '$.labels.' || ?) = ?`)
		args = append(args, k, v)
	}

	if len(conditions) > 0 {
		query += " WHERE "
		for i, c := range conditions {
			if i > 0 {
				query += " AND "
			}
			query += c
		}
	}

	query += " ORDER BY c.ID"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []CommandEntry
	for rows.Next() {
		var e CommandEntry
		var nameSQL sql.NullString
		var ecSQL sql.NullInt64
		var cfgStr, stateStr string
		if err := rows.Scan(&e.ID, &nameSQL, &e.State, &ecSQL, &cfgStr, &stateStr); err != nil {
			return nil, err
		}
		if nameSQL.Valid {
			e.Name = nameSQL.String
		}
		if ecSQL.Valid {
			ec := int(ecSQL.Int64)
			e.ExitCode = &ec
		}
		e.ConfigJSON = &CommandConfigJSON{}
		if err := json.Unmarshal([]byte(cfgStr), e.ConfigJSON); err != nil {
			return nil, err
		}
		e.StateJSON = &CommandStateJSON{}
		if err := json.Unmarshal([]byte(stateStr), e.StateJSON); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// GetExitHistory retrieves exit code history for a command.
func (s *Store) GetExitHistory(id string) ([]ExitRecord, error) {
	rows, err := s.db.Query(
		`SELECT Timestamp, ExitCode FROM CommandExitCode WHERE ID = ? ORDER BY Timestamp`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ExitRecord
	for rows.Next() {
		var r ExitRecord
		if err := rows.Scan(&r.Timestamp, &r.ExitCode); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// ExitRecord represents an entry in CommandExitCode.
type ExitRecord struct {
	Timestamp string `json:"timestamp"`
	ExitCode  int    `json:"exit_code"`
}

// DeleteCommand removes all rows and the command directory for a command.
func (s *Store) DeleteCommand(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range []string{"CommandExitCode", "CommandState", "CommandConfig"} {
		if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE ID = ?", table), id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ResolveID resolves an ID or name to a command ID.
func (s *Store) ResolveID(idOrName string) (string, error) {
	var id string
	err := s.db.QueryRow(
		`SELECT ID FROM CommandConfig WHERE ID = ? OR Name = ?`,
		idOrName, idOrName,
	).Scan(&id)
	return id, err
}

// FindByLabels returns command IDs matching all the given labels.
func (s *Store) FindByLabels(labels map[string]string) ([]string, error) {
	query := `SELECT ID FROM CommandConfig WHERE 1=1`
	var args []any
	for k, v := range labels {
		query += ` AND json_extract(JSON, '$.labels.' || ?) = ?`
		args = append(args, k, v)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func nullableString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
