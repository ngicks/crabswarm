package crabswarm_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ngicks/crabswarm/crabswarm/issues"
)

// issuesLintRecord is the record `issues lint --changed` keeps under
// stateHome for the database the fake bd reports, "" when there is none.
// It fails the test when anything else sits in the record directory.
func issuesLintRecord(t *testing.T, stateHome string) string {
	t.Helper()
	dir := filepath.Join(stateHome, "crabswarm", "issues-lint")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	if err != nil {
		t.Fatalf("read the record directory: %v", err)
	}
	want := issues.SourceID("/fake/.beads")
	for _, e := range entries {
		if e.Name() != want {
			t.Fatalf("record directory holds %q, want only %q", e.Name(), want)
		}
	}
	if len(entries) == 0 {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(dir, want))
	if err != nil {
		t.Fatalf("read the record: %v", err)
	}
	return string(b)
}

// --changed lints only when the beads commit or the scope moved since the
// last clean lint, and prints nothing either way while the backlog is clean.
// Every run goes through a fresh -C directory that bd resolves to the same
// database, the way every worktree of a repository does, and they all share
// one record.
func TestIssuesLintChanged(t *testing.T) {
	stateHome := t.TempDir()
	calls := filepath.Join(t.TempDir(), "calls")

	steps := []struct {
		name   string
		commit string
		args   []string
		want   []string
	}{
		{name: "first run", commit: "c1", want: []string{"where", "vc", "list"}},
		{name: "same commit", commit: "c1", want: []string{"where", "vc"}},
		{
			name:   "another scope",
			commit: "c1",
			args:   []string{"--all"},
			want:   []string{"where", "vc", "list"},
		},
		{
			name:   "same scope again",
			commit: "c1",
			args:   []string{"--all"},
			want:   []string{"where", "vc"},
		},
		{
			name:   "moved commit",
			commit: "c2",
			args:   []string{"--all"},
			want:   []string{"where", "vc", "list"},
		},
	}
	for _, s := range steps {
		before := len(bdCalls(t, calls))
		res := runIssuesLintEnv(
			t.Context(), t, fakeBdListScript, bdListWithoutFences,
			[]string{
				"XDG_STATE_HOME=" + stateHome,
				"FAKE_BD_COMMIT=" + s.commit,
				"FAKE_BD_CALLS=" + calls,
			},
			append([]string{"--changed"}, s.args...)...,
		)
		if res.exitCode != 0 || res.stdout != "" || res.stderr != "" {
			t.Fatalf("%s: exit code = %d, want 0 and no output\nstdout:\n%s\nstderr:\n%s",
				s.name, res.exitCode, res.stdout, res.stderr)
		}
		got := bdCalls(t, calls)[before:]
		if strings.Join(got, " ") != strings.Join(s.want, " ") {
			t.Errorf("%s: bd ran %v, want %v", s.name, got, s.want)
		}
	}

	const want = `{"beads_path":"/fake/.beads","commit":"c2","all":true,"limit":0}` + "\n"
	if got := issuesLintRecord(t, stateHome); got != want {
		t.Errorf("record = %q, want %q", got, want)
	}
}

// A lint that finds something records nothing, so the next run at the same
// beads commit lints again and refuses the commit again.
func TestIssuesLintChangedReportsFindingsAgain(t *testing.T) {
	requireMermaidLint(t)

	stateHome := t.TempDir()
	calls := filepath.Join(t.TempDir(), "calls")
	env := []string{
		"XDG_STATE_HOME=" + stateHome,
		"FAKE_BD_COMMIT=c1",
		"FAKE_BD_CALLS=" + calls,
	}
	for run := 1; run <= 2; run++ {
		res := runIssuesLintEnv(
			t.Context(),
			t,
			fakeBdListScript,
			bdListWithFences,
			env,
			"--changed",
		)
		if res.exitCode != 1 {
			t.Fatalf("run %d: exit code = %d, want 1\nstdout:\n%s\nstderr:\n%s",
				run, res.exitCode, res.stdout, res.stderr)
		}
		if !strings.HasPrefix(res.stdout, "e2e-002 description:3:4: ") {
			t.Errorf("run %d: stdout = %q, want the finding", run, res.stdout)
		}
	}
	if got := countCalls(t, calls, "list"); got != 2 {
		t.Errorf("bd list ran %d times, want 2", got)
	}
	if got := issuesLintRecord(t, stateHome); got != "" {
		t.Errorf("record = %q, want none", got)
	}
}

// Without --changed the command lints every run and neither reads nor writes
// the record: a run at a commit the record does not hold leaves it as it was.
func TestIssuesLintWithoutChangedIgnoresRecord(t *testing.T) {
	stateHome := t.TempDir()
	calls := filepath.Join(t.TempDir(), "calls")
	env := func(commit string) []string {
		return []string{
			"XDG_STATE_HOME=" + stateHome,
			"FAKE_BD_COMMIT=" + commit,
			"FAKE_BD_CALLS=" + calls,
		}
	}

	res := runIssuesLintEnv(
		t.Context(), t, fakeBdListScript, bdListWithoutFences, env("c1"), "--changed")
	if res.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr:\n%s", res.exitCode, res.stderr)
	}
	recorded := issuesLintRecord(t, stateHome)

	for _, commit := range []string{"c1", "c2"} {
		before := len(bdCalls(t, calls))
		res := runIssuesLintEnv(
			t.Context(), t, fakeBdListScript, bdListWithoutFences, env(commit))
		if res.exitCode != 0 || res.stdout != "" {
			t.Fatalf("commit %s: exit code = %d, want 0 and no output\nstdout:\n%s\nstderr:\n%s",
				commit, res.exitCode, res.stdout, res.stderr)
		}
		got := bdCalls(t, calls)[before:]
		if strings.Join(got, " ") != "where list" {
			t.Errorf("commit %s: bd ran %v, want [where list]", commit, got)
		}
	}
	if got := issuesLintRecord(t, stateHome); got != recorded {
		t.Errorf("record = %q, want it left at %q", got, recorded)
	}
}

// A beads commit that cannot be read is a failure, not a reason to skip: the
// command cannot tell whether the backlog moved.
func TestIssuesLintChangedCommitError(t *testing.T) {
	calls := filepath.Join(t.TempDir(), "calls")
	res := runIssuesLintEnv(
		t.Context(), t, fakeBdListScript, bdListWithoutFences,
		[]string{
			"XDG_STATE_HOME=" + t.TempDir(),
			"FAKE_BD_VC_FAIL=1",
			"FAKE_BD_CALLS=" + calls,
		},
		"--changed",
	)
	if res.exitCode != 1 {
		t.Fatalf("exit code = %d, want 1\nstdout:\n%s\nstderr:\n%s",
			res.exitCode, res.stdout, res.stderr)
	}
	if res.stdout != "" {
		t.Errorf("stdout = %q, want empty", res.stdout)
	}
	if !strings.Contains(res.stderr, "database is locked") {
		t.Errorf("stderr = %q, want bd's own complaint", res.stderr)
	}
	if got := countCalls(t, calls, "list"); got != 0 {
		t.Errorf("bd list ran %d times, want 0", got)
	}
}

// bdCalls returns the subcommands the fake bd recorded in the file at path,
// one per invocation, in order.
func bdCalls(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("read bd calls: %v", err)
	}
	return strings.Fields(string(b))
}

// countCalls counts the fake bd invocations of subcommand sub.
func countCalls(t *testing.T, path, sub string) int {
	t.Helper()
	n := 0
	for _, c := range bdCalls(t, path) {
		if c == sub {
			n++
		}
	}
	return n
}
