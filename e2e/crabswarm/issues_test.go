package crabswarm_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// fakeBdWhereEnvelope is what `bd where --json` prints under
// BD_JSON_ENVELOPE=1 in a directory a beads database governs.
const fakeBdWhereEnvelope = `{"data":{"path":"/fake/.beads","prefix":"e2e",` +
	`"database_path":"/fake/.beads/embeddeddolt"},"schema_version":1}`

// fakeBdNoBeadsEnvelope is what `bd where --json` prints under
// BD_JSON_ENVELOPE=1, with exit 1, where no beads database governs.
const fakeBdNoBeadsEnvelope = `{"data":{"error":"no_beads_directory",` +
	`"message":"No active beads workspace found."},"schema_version":1}`

// fakeBdVcStatus answers `bd vc status --json` with the commit in
// FAKE_BD_COMMIT, or fails the way bd fails to open its database when
// FAKE_BD_VC_FAIL is set. It is a case arm of the fake bd scripts.
const fakeBdVcStatus = `vc)
  if [ -n "$FAKE_BD_VC_FAIL" ]; then echo "Error: database is locked" >&2; exit 1; fi
  printf '{"branch":"main","commit":"%s","schema_version":1}\n' "${FAKE_BD_COMMIT:-e2e-commit}"
  ;;
`

// fakeBdRecordCall appends the subcommand of every fake bd invocation to the
// file FAKE_BD_CALLS names, when it names one.
const fakeBdRecordCall = `if [ -n "$FAKE_BD_CALLS" ]; then echo "$1" >> "$FAKE_BD_CALLS"; fi
`

// fakeBdListScript is a stand-in for bd that answers `bd where --json` as a
// governed directory, `bd list --json` with the recorded listing in
// FAKE_BD_LIST and `bd vc status --json` with the commit in FAKE_BD_COMMIT,
// and refuses everything else. Those three are the whole conversation
// `issues lint` has with bd as long as no issue carries comments, so a case
// that needs none can stop here.
const fakeBdListScript = `#!/bin/sh
` + fakeBdRecordCall + `case "$1" in
where) echo '` + fakeBdWhereEnvelope + `' ;;
list) cat "$FAKE_BD_LIST" ;;
` + fakeBdVcStatus + `*) echo "fake bd: unsupported subcommand $*" >&2; exit 2 ;;
esac
`

// fakeBdNoBeadsScript is a stand-in for bd run where no beads database
// governs: `bd where --json` reports so, and nothing else is expected to
// run.
const fakeBdNoBeadsScript = `#!/bin/sh
case "$1" in
where) echo '` + fakeBdNoBeadsEnvelope + `'; exit 1 ;;
*) echo "fake bd: unexpected subcommand $*" >&2; exit 2 ;;
esac
`

// bdListWithFences is a backlog of two issues whose descriptions each hold a
// mermaid diagram: one the parser accepts, one it refuses. Neither carries a
// comment, so the listing is all `issues lint` reads.
const bdListWithFences = `[
  {
    "id": "e2e-001",
    "title": "a diagram that parses",
    "status": "open",
    "description": "` + "```mermaid\\nflowchart TD\\n  A --> B\\n```" + `\n",
    "comment_count": 0
  },
  {
    "id": "e2e-002",
    "title": "a diagram that does not",
    "status": "open",
    "description": "` + "```mermaid\\nflowchart TD\\n  A -->\\n```" + `\n",
    "comment_count": 0
  }
]
`

// bdListWithoutFences is a backlog holding no diagram at all, which is the
// case mermaid-lint is never run for.
const bdListWithoutFences = `[
  {
    "id": "e2e-003",
    "title": "prose only",
    "status": "open",
    "description": "no diagram here",
    "comment_count": 0
  }
]
`

// runIssuesLint runs the built binary's `issues lint` with the fake bd
// script on PATH answering with listing, from a temp directory so no
// mermaid-lint configuration around the checkout reaches the run. A
// non-zero exit is the command reporting findings, so only a failure to run
// at all is fatal. The run keeps any record under a state directory of its
// own, never under the real ~/.local/state.
func runIssuesLint(
	ctx context.Context,
	t *testing.T,
	script string,
	listing string,
	args ...string,
) hookResult {
	t.Helper()
	stateHome := "XDG_STATE_HOME=" + t.TempDir()
	return runIssuesLintEnv(ctx, t, script, listing, []string{stateHome}, args...)
}

// runIssuesLintEnv is [runIssuesLint] with env added to the run's
// environment. A case running the command several times passes its own
// XDG_STATE_HOME, so that the runs share one record.
func runIssuesLintEnv(
	ctx context.Context,
	t *testing.T,
	script string,
	listing string,
	env []string,
	args ...string,
) hookResult {
	t.Helper()

	bdDir := t.TempDir()
	writeFile(t, filepath.Join(bdDir, "bd"), script)
	if err := os.Chmod(filepath.Join(bdDir, "bd"), 0o755); err != nil {
		t.Fatalf("make fake bd executable: %v", err)
	}
	listPath := filepath.Join(bdDir, "list.json")
	writeFile(t, listPath, listing)

	full := append([]string{"issues", "lint", "-C", t.TempDir()}, args...)
	cmd := exec.CommandContext(ctx, crabswarmBin, full...)
	cmd.Env = append(os.Environ(),
		"PATH="+bdDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_BD_LIST="+listPath,
	)
	cmd.Env = append(cmd.Env, env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if _, ok := errors.AsType[*exec.ExitError](err); !ok {
			t.Fatalf("crabswarm %s: %v\nstdout:\n%s\nstderr:\n%s",
				strings.Join(full, " "), err, stdout.String(), stderr.String())
		}
	}
	return hookResult{
		stdout:   stdout.String(),
		stderr:   stderr.String(),
		exitCode: cmd.ProcessState.ExitCode(),
	}
}

// requireMermaidLint skips a case that needs the real parser.
func requireMermaidLint(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("mermaid-lint"); err != nil {
		t.Skipf("mermaid-lint not on PATH: %v", err)
	}
}

// A broken diagram in an issue is one line on stdout and exit 1 — which is
// what lets the command gate a commit as a pre-commit step. The diagram that parses
// is not reported.
func TestIssuesLintReportsBrokenFence(t *testing.T) {
	requireMermaidLint(t)

	res := runIssuesLint(t.Context(), t, fakeBdListScript, bdListWithFences)

	if res.exitCode != 1 {
		t.Fatalf("exit code = %d, want 1\nstdout:\n%s\nstderr:\n%s",
			res.exitCode, res.stdout, res.stderr)
	}

	line, ok := strings.CutSuffix(res.stdout, "\n")
	if !ok || strings.Contains(line, "\n") {
		t.Fatalf("stdout should be exactly one finding; got:\n%s", res.stdout)
	}
	// The issue, the field, and the position inside that field's text — the
	// diagram opens on the first line and mermaid rejects the third. The
	// parser's own wording is left unpinned; it belongs to mermaid-lint.
	const want = "e2e-002 description:3:4: "
	if !strings.HasPrefix(line, want) {
		t.Errorf("finding = %q, want it to start with %q", line, want)
	}
	if strings.TrimPrefix(line, want) == "" {
		t.Errorf("finding = %q, want a parser message after the position", line)
	}
}

// A backlog without diagrams is a clean sweep: exit 0, and --json prints the
// empty array rather than null, so a consumer decodes it the same way it
// decodes a run that found something.
func TestIssuesLintCleanJSON(t *testing.T) {
	requireMermaidLint(t)

	res := runIssuesLint(t.Context(), t, fakeBdListScript, bdListWithoutFences, "--json")

	if res.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s",
			res.exitCode, res.stdout, res.stderr)
	}
	if res.stderr != "" {
		t.Errorf("stderr = %q, want empty", res.stderr)
	}
	if got := strings.TrimSpace(res.stdout); got != "[]" {
		t.Errorf("stdout = %q, want %q", got, "[]")
	}
}

// A directory no beads database governs holds nothing to lint: exit 0 and
// silence, so the hook installed user-wide passes in a repository that keeps
// no issues instead of blocking every turn there. The fake bd refuses any
// subcommand but `where`, so the test also proves no listing is attempted.
func TestIssuesLintSkipsWithoutBeads(t *testing.T) {
	res := runIssuesLint(t.Context(), t, fakeBdNoBeadsScript, "")

	if res.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s",
			res.exitCode, res.stdout, res.stderr)
	}
	if res.stdout != "" {
		t.Errorf("stdout = %q, want empty", res.stdout)
	}
	if res.stderr != "" {
		t.Errorf("stderr = %q, want empty", res.stderr)
	}
}

// fakeBdProjectScript is a stand-in for bd that answers `bd where --json`,
// `bd list --json` (with FAKE_BD_LIST) and `bd vc status --json` (with
// FAKE_BD_COMMIT) only when run from FAKE_BD_PROJECT. Anywhere else it
// answers the way real bd does in a directory no beads database governs.
const fakeBdProjectScript = `#!/bin/sh
` + fakeBdRecordCall + `if [ "$(pwd -P)" != "$FAKE_BD_PROJECT" ]; then
  if [ "$1" = where ]; then
    echo '` + fakeBdNoBeadsEnvelope + `'
    exit 1
  fi
  echo "Error: no beads database found" >&2
  echo "Hint: run 'bd where' to inspect the resolved workspace" >&2
  exit 1
fi
case "$1" in
where) echo '` + fakeBdWhereEnvelope + `' ;;
list) cat "$FAKE_BD_LIST" ;;
` + fakeBdVcStatus + `*) echo "fake bd: unsupported subcommand $*" >&2; exit 2 ;;
esac
`

// hkStepCheck matches the `check` of the one step the fragment declares.
var hkStepCheck = regexp.MustCompile(`(?m)^\s*check = "([^"]+)"$`)

// preCommitCheck is the command the pre-commit step of the
// crabswarm-issues-lint package runs, read out of the fragment hk-init.sh
// copies into a repository.
func preCommitCheck(t *testing.T) string {
	t.Helper()
	path := filepath.Join(
		apmSkillPluginDir("crabswarm-issues-lint"),
		"reference",
		"crabswarm-issues-lint.hk.pkl",
	)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the fragment: %v", err)
	}
	matches := hkStepCheck.FindAllSubmatch(b, -1)
	if len(matches) != 1 {
		t.Fatalf("%s: want exactly one check, got %d", path, len(matches))
	}
	return string(matches[0][1])
}

// The pre-commit step runs its check from the worktree root, the way hk does,
// and git reads nothing but its exit code: 0 lets the commit through, anything
// else refuses it and shows what the check printed.
//
// A repository that keeps no issues still gets the step when its hk.pkl
// spreads the fragment. There bd finds no database, and the commit goes
// through.
//
// Each case commits twice with the backlog standing still. The second commit
// of a clean backlog skips the listing; a broken diagram is listed and
// reported again.
func TestIssuesLintPreCommitCheck(t *testing.T) {
	requireMermaidLint(t)

	cases := []struct {
		name    string
		listing string
		// noBeads makes bd answer as if no database governs the repository.
		noBeads  bool
		wantExit int
		wantOut  string
		// wantLists is how many of the two runs listed the backlog.
		wantLists int
	}{
		{name: "clean backlog", listing: bdListWithoutFences, wantLists: 1},
		{
			name:      "broken diagram",
			listing:   bdListWithFences,
			wantExit:  1,
			wantOut:   "e2e-002 description:3:4: ",
			wantLists: 2,
		},
		{name: "no beads", listing: bdListWithFences, noBeads: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			project, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			// The fake bd answers only from governed, so a repository without
			// beads is one bd does not recognise.
			governed := project
			if tc.noBeads {
				governed = t.TempDir()
			}

			bdDir := t.TempDir()
			writeFile(t, filepath.Join(bdDir, "bd"), fakeBdProjectScript)
			if err := os.Chmod(filepath.Join(bdDir, "bd"), 0o755); err != nil {
				t.Fatalf("make fake bd executable: %v", err)
			}
			listPath := filepath.Join(bdDir, "list.json")
			writeFile(t, listPath, tc.listing)
			calls := filepath.Join(bdDir, "calls")

			command := preCommitCheck(t)
			env := append(os.Environ(),
				"PATH="+bdDir+string(os.PathListSeparator)+
					filepath.Dir(crabswarmBin)+string(os.PathListSeparator)+os.Getenv("PATH"),
				"FAKE_BD_LIST="+listPath,
				"FAKE_BD_PROJECT="+governed,
				"FAKE_BD_CALLS="+calls,
				"XDG_STATE_HOME="+t.TempDir(),
			)
			for run := 1; run <= 2; run++ {
				cmd := exec.CommandContext(t.Context(), "sh", "-c", command)
				cmd.Dir = project
				cmd.Env = env
				var stdout, stderr bytes.Buffer
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				if err := cmd.Run(); err != nil {
					if _, ok := errors.AsType[*exec.ExitError](err); !ok {
						t.Fatalf("run %d: %s: %v\nstdout:\n%s\nstderr:\n%s",
							run, command, err, stdout.String(), stderr.String())
					}
				}
				if got := cmd.ProcessState.ExitCode(); got != tc.wantExit {
					t.Fatalf("run %d: exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s",
						run, got, tc.wantExit, stdout.String(), stderr.String())
				}
				switch out := stdout.String(); {
				case tc.wantOut == "" && out != "":
					t.Errorf("run %d: stdout = %q, want empty", run, out)
				case !strings.HasPrefix(out, tc.wantOut):
					t.Errorf("run %d: stdout = %q, want it to start with %q", run, out, tc.wantOut)
				}
			}
			if got := countCalls(t, calls, "list"); got != tc.wantLists {
				t.Errorf("bd list ran %d times, want %d", got, tc.wantLists)
			}
		})
	}
}
