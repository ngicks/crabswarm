// Package issues reads a beads issue database through the `bd` command line
// tool. Every call runs `bd ... --json` as a subprocess in a working
// directory and decodes its output; nothing here opens the database
// directly, so bd stays the only component that knows where the database
// lives and how it is stored.
//
// [Where] resolves the beads directory that governs a directory. [Client]
// reads issues out of it: [Client.List] and [Client.Children] return
// [Summary] records, each carrying its own outgoing [Edge] records, and
// [Client.Get] returns a full [Issue] with its comments and dependencies.
// [Poller] turns a client into one listing per source that every reader of
// that source shares, so bd — which admits a single process per database —
// runs once for a screenful of questions.
package issues

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// defaultBinary is the bd executable looked up on PATH when the caller
// passes no [WithBinary].
const defaultBinary = "bd"

// jsonEnvelopeEnv makes bd wrap its JSON output — success payload under
// "data", failures as a machine-readable error code beside a message —
// instead of printing a bare payload. Only `bd where` is run with it: the
// envelope is what turns "this directory has no beads database" into
// [ErrNoBeads] rather than an opaque exit status.
const jsonEnvelopeEnv = "BD_JSON_ENVELOPE=1"

// noBeadsCode is the error code bd reports when no beads workspace governs
// the directory it ran in.
const noBeadsCode = "no_beads_directory"

// ErrNoBeads reports that no beads database governs the directory. Callers
// match it with [errors.Is] to tell "not a beads workspace" apart from a bd
// invocation that failed for any other reason.
var ErrNoBeads = errors.New("no beads directory")

// Location is the beads workspace governing a directory.
type Location struct {
	// BeadsPath is the .beads directory itself.
	BeadsPath string
	// DatabasePath is the database inside it.
	DatabasePath string
	// Prefix is the string every issue ID in this database starts with.
	Prefix string
}

// whereEnvelope is the enveloped output of `bd where --json`. bd reports a
// failure inside the same "data" object it uses for the payload, so both
// sets of fields are decoded together.
type whereEnvelope struct {
	Data struct {
		Path         string `json:"path"`
		Prefix       string `json:"prefix"`
		DatabasePath string `json:"database_path"`

		Error   string `json:"error"`
		Hint    string `json:"hint"`
		Message string `json:"message"`
	} `json:"data"`
}

// Where reports the beads workspace that governs dir by running
// `bd where --json` there, with bd taken from PATH. It returns an error
// wrapping [ErrNoBeads] when dir belongs to no beads workspace.
func Where(ctx context.Context, dir string) (Location, error) {
	c := NewClient(dir, WithEnv(jsonEnvelopeEnv))
	// bd prints the error envelope on stdout and exits non-zero, so the
	// output is decoded before the exit status is judged.
	out, runErr := c.run(ctx, "where", "--json")
	var env whereEnvelope
	if err := json.Unmarshal(out, &env); err != nil {
		if runErr != nil {
			return Location{}, runErr
		}
		return Location{}, fmt.Errorf("decoding bd where in %s: %w", dir, err)
	}
	if env.Data.Error != "" {
		msg := env.Data.Message
		if msg == "" {
			msg = env.Data.Error
		}
		if env.Data.Error == noBeadsCode {
			return Location{}, fmt.Errorf("bd where in %s: %s: %w", dir, msg, ErrNoBeads)
		}
		if runErr != nil {
			return Location{}, fmt.Errorf("bd where in %s: %s: %w", dir, msg, runErr)
		}
		return Location{}, fmt.Errorf("bd where in %s: %s: %s", dir, env.Data.Error, msg)
	}
	if runErr != nil {
		return Location{}, runErr
	}
	return Location{
		BeadsPath:    env.Data.Path,
		DatabasePath: env.Data.DatabasePath,
		Prefix:       env.Data.Prefix,
	}, nil
}
