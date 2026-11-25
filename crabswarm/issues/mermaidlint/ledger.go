package mermaidlint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ngicks/crabswarm/crabswarm/issues"
)

// CommitSource is what [Ledger.Sweep] reads a backlog through: everything
// [Sweep] reads, plus the beads commit the backlog stands at.
// [*issues.Client] satisfies it.
type CommitSource interface {
	Source
	Commit(ctx context.Context) (string, error)
}

// Ledger remembers, for each beads database, the beads commit and the sweep
// scope of the last sweep that found nothing. [Ledger.Sweep] skips a sweep
// whose answer the ledger already holds, which spares a hook running on every
// commit the bd listing and the mermaid-lint run while the backlog stands
// still.
//
// Each database has one record file in the ledger's directory, named by the
// [issues.SourceID] of its .beads path, so every worktree of a repository
// reads and writes the same record.
type Ledger struct {
	dir    string
	logger *slog.Logger
}

// NewLedger returns a Ledger keeping its records in dir, which is created on
// the first write. dir must be absolute: with any other dir, "" included, the
// ledger never skips and records nothing. A nil logger logs to
// [slog.Default].
func NewLedger(dir string, logger *slog.Logger) *Ledger {
	if logger == nil {
		logger = slog.Default()
	}
	return &Ledger{dir: dir, logger: logger}
}

// Sweep runs [Sweep] over src unless the ledger holds a clean sweep of the
// database at beadsPath, taken at the commit src reports and in the scope o
// names. skipped reports that case, in which src lists nothing and findings
// is nil.
//
// A sweep that finds nothing is recorded. One that finds something is not, so
// the next call sweeps again and reports the findings again. A record that
// cannot be read counts as no record, and one that cannot be written is
// logged and dropped: either costs only a sweep that could have been skipped.
// A commit that cannot be read is an error, because without it the ledger
// cannot tell whether the backlog moved.
func (l *Ledger) Sweep(
	ctx context.Context,
	src CommitSource,
	beadsPath string,
	o SweepOptions,
	opts ...Option,
) (findings []Finding, skipped bool, err error) {
	commit, err := src.Commit(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("reading the beads commit: %w", err)
	}
	want := record{
		BeadsPath: filepath.Clean(beadsPath),
		Commit:    commit,
		All:       o.All,
		Limit:     o.Limit,
	}

	got, err := l.load(want.BeadsPath)
	switch {
	case err != nil:
		l.logger.Debug("issues lint: no usable record, linting",
			"beads", want.BeadsPath, "err", err)
	case got == want:
		l.logger.Debug("issues lint: backlog unchanged since the last clean lint, skipping",
			"beads", want.BeadsPath, "commit", commit)
		return nil, true, nil
	default:
		l.logger.Debug("issues lint: backlog or scope changed since the last clean lint, linting",
			"beads", want.BeadsPath, "commit", commit, "recorded", got.Commit)
	}

	findings, err = Sweep(ctx, src, o, opts...)
	if err != nil {
		return nil, false, err
	}
	if len(findings) > 0 {
		return findings, false, nil
	}
	// The commit recorded is the one read before the sweep. A write landing
	// while the sweep runs moves the database past it, so the next call
	// sweeps again instead of trusting a sweep that may have missed it.
	if err := l.save(want); err != nil {
		l.logger.Warn("issues lint: could not record the clean lint",
			"beads", want.BeadsPath, "err", err)
	}
	return nil, false, nil
}

// record is what a record file holds. BeadsPath is there for a person
// reading the file, whose name is only a hash of it.
type record struct {
	BeadsPath string `json:"beads_path"`
	Commit    string `json:"commit"`
	All       bool   `json:"all"`
	Limit     int    `json:"limit"`
}

// path is the record file of the database at beadsPath.
func (l *Ledger) path(beadsPath string) (string, error) {
	// A relative dir would put the record wherever the command happened to
	// run, which is usually the worktree being committed.
	if !filepath.IsAbs(l.dir) {
		return "", fmt.Errorf("record directory %q is not an absolute path", l.dir)
	}
	return filepath.Join(l.dir, issues.SourceID(beadsPath)), nil
}

func (l *Ledger) load(beadsPath string) (record, error) {
	path, err := l.path(beadsPath)
	if err != nil {
		return record{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return record{}, err
	}
	var r record
	if err := json.Unmarshal(b, &r); err != nil {
		return record{}, fmt.Errorf("decoding %s: %w", path, err)
	}
	return r, nil
}

func (l *Ledger) save(r record) error {
	path, err := l.path(r.BeadsPath)
	if err != nil {
		return err
	}
	b, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("encoding the record: %w", err)
	}
	if err := os.MkdirAll(l.dir, 0o700); err != nil {
		return fmt.Errorf("creating the record directory: %w", err)
	}
	// Written aside and renamed into place, so that the hook of another
	// worktree reading the record meanwhile sees the old one or the new one,
	// never half of one.
	f, err := os.CreateTemp(l.dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating a temp file for %s: %w", path, err)
	}
	tmp := f.Name()
	_, writeErr := f.Write(append(b, '\n'))
	if err := errors.Join(writeErr, f.Close()); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("moving the record into place at %s: %w", path, err)
	}
	return nil
}
