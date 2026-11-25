package mermaidlint

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/crabswarm/issues"
)

// fakeCommitSource is a [fakeSource] standing at a beads commit.
type fakeCommitSource struct {
	*fakeSource
	commit    string
	commitErr error
}

func (f *fakeCommitSource) Commit(context.Context) (string, error) {
	return f.commit, f.commitErr
}

var _ CommitSource = (*issues.Client)(nil)

const ledgerBeads = "/repo/.beads"

// cleanBacklog holds no diagram, so a sweep of it lists and finds nothing
// without running mermaid-lint.
func cleanBacklog(commit string) *fakeCommitSource {
	return &fakeCommitSource{
		fakeSource: &fakeSource{backlog: []issues.Issue{{
			Summary: issues.Summary{ID: "plan-ccc", Description: "prose only"},
		}}},
		commit: commit,
	}
}

// debugLogger keeps every record the ledger logs, debug included.
func debugLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(h), &buf
}

func TestLedgerSweepRecordsCleanAndSkips(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state", "issues-lint")
	ledger := NewLedger(dir, nil)
	src := cleanBacklog("c1")

	got, skipped, err := ledger.Sweep(t.Context(), src, ledgerBeads, SweepOptions{})
	assert.NilError(t, err)
	assert.Assert(t, !skipped)
	assert.Equal(t, len(got), 0)
	assert.Equal(t, len(src.filters), 1)

	b, err := os.ReadFile(filepath.Join(dir, issues.SourceID(ledgerBeads)))
	assert.NilError(t, err)
	assert.Equal(t, string(b),
		`{"beads_path":"/repo/.beads","commit":"c1","all":false,"limit":0}`+"\n")

	got, skipped, err = ledger.Sweep(t.Context(), src, ledgerBeads, SweepOptions{})
	assert.NilError(t, err)
	assert.Assert(t, skipped)
	assert.Equal(t, len(got), 0)
	assert.Equal(t, len(src.filters), 1, "a skipped sweep must not list the backlog")

	// Another worktree of the repository spells the same .beads directory
	// differently and still finds the record.
	_, skipped, err = ledger.Sweep(t.Context(), src, ledgerBeads+"/", SweepOptions{})
	assert.NilError(t, err)
	assert.Assert(t, skipped)
}

func TestLedgerSweepAgain(t *testing.T) {
	for _, tc := range []struct {
		name   string
		commit string
		opts   SweepOptions
	}{
		{name: "commit moved", commit: "c2"},
		{name: "all", commit: "c1", opts: SweepOptions{All: true}},
		{name: "limit", commit: "c1", opts: SweepOptions{Limit: 20}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ledger := NewLedger(t.TempDir(), nil)
			_, _, err := ledger.Sweep(t.Context(), cleanBacklog("c1"), ledgerBeads, SweepOptions{})
			assert.NilError(t, err)

			src := cleanBacklog(tc.commit)
			_, skipped, err := ledger.Sweep(t.Context(), src, ledgerBeads, tc.opts)
			assert.NilError(t, err)
			assert.Assert(t, !skipped)
			assert.Equal(t, len(src.filters), 1)

			// The new commit and scope replace the record.
			_, skipped, err = ledger.Sweep(t.Context(), src, ledgerBeads, tc.opts)
			assert.NilError(t, err)
			assert.Assert(t, skipped)
		})
	}
}

func TestLedgerSweepFindingsAreNotRecorded(t *testing.T) {
	installFakeMermaidLint(t, "report.json")
	dir := t.TempDir()
	ledger := NewLedger(dir, nil)

	for range 2 {
		src := &fakeCommitSource{fakeSource: &fakeSource{backlog: linted()}, commit: "c1"}
		got, skipped, err := ledger.Sweep(t.Context(), src, ledgerBeads, SweepOptions{})
		assert.NilError(t, err)
		assert.Assert(t, !skipped)
		assert.DeepEqual(t, got, lintedFindings)
	}

	entries, err := os.ReadDir(dir)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 0)
}

func TestLedgerSweepCommitError(t *testing.T) {
	boom := errors.New("bd exploded")
	src := cleanBacklog("")
	src.commitErr = boom

	_, _, err := NewLedger(t.TempDir(), nil).Sweep(t.Context(), src, ledgerBeads, SweepOptions{})
	assert.ErrorIs(t, err, boom)
	assert.Equal(t, len(src.filters), 0)
}

func TestLedgerSweepUndecodableRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, issues.SourceID(ledgerBeads))
	assert.NilError(t, os.WriteFile(path, []byte("{half a rec"), 0o600))
	ledger := NewLedger(dir, nil)
	src := cleanBacklog("c1")

	_, skipped, err := ledger.Sweep(t.Context(), src, ledgerBeads, SweepOptions{})
	assert.NilError(t, err)
	assert.Assert(t, !skipped)
	assert.Equal(t, len(src.filters), 1)

	_, skipped, err = ledger.Sweep(t.Context(), src, ledgerBeads, SweepOptions{})
	assert.NilError(t, err)
	assert.Assert(t, skipped)
}

func TestLedgerSweepUnwritableRecord(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	assert.NilError(t, os.WriteFile(blocker, nil, 0o600))

	for _, tc := range []struct {
		name string
		dir  string
	}{
		{name: "no directory", dir: ""},
		{name: "relative directory", dir: "issues-lint"},
		{name: "directory under a file", dir: filepath.Join(blocker, "issues-lint")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			logger, logs := debugLogger()
			ledger := NewLedger(tc.dir, logger)
			src := cleanBacklog("c1")

			for range 2 {
				got, skipped, err := ledger.Sweep(t.Context(), src, ledgerBeads, SweepOptions{})
				assert.NilError(t, err)
				assert.Assert(t, !skipped)
				assert.Equal(t, len(got), 0)
			}
			assert.Equal(t, len(src.filters), 2)
			assert.Assert(t, bytes.Contains(logs.Bytes(), []byte("level=WARN")), logs.String())

			entries, err := os.ReadDir(".")
			assert.NilError(t, err)
			assert.Equal(t, len(entries), 0, "nothing may land in the working directory")
		})
	}
}
