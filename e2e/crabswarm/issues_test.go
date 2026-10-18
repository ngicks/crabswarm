package crabswarm_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// fakeBdListScript is a stand-in for bd that answers `bd where --json` as a
// governed directory and `bd list --json` with the recorded listing in
// FAKE_BD_LIST, and refuses everything else. Those two are the whole
// conversation `issues lint` has with bd as long as no issue carries
// comments, so a case that needs none can stop here.
const fakeBdListScript = `#!/bin/sh
case "$1" in
where) echo '` + fakeBdWhereEnvelope + `' ;;
list) cat "$FAKE_BD_LIST" ;;
*) echo "fake bd: unsupported subcommand $*" >&2; exit 2 ;;
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
// at all is fatal.
func runIssuesLint(
	ctx context.Context,
	t *testing.T,
	script string,
	listing string,
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
// what lets the command gate a turn as a Stop hook. The diagram that parses
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

// fakeBdProjectScript is a stand-in for bd that answers `bd where --json`
// and `bd list --json` (with FAKE_BD_LIST) only when run from
// FAKE_BD_PROJECT. Anywhere else it answers the way real bd does in a
// directory no beads database governs.
const fakeBdProjectScript = `#!/bin/sh
if [ "$(pwd -P)" != "$FAKE_BD_PROJECT" ]; then
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
*) echo "fake bd: unsupported subcommand $*" >&2; exit 2 ;;
esac
`

// stopHookCommand is the one Stop command a hooks file of the
// crabswarm-issues-lint package declares.
func stopHookCommand(t *testing.T, path string) string {
	t.Helper()
	var file struct {
		Hooks struct {
			Stop []struct {
				Hooks []struct {
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"Stop"`
		} `json:"hooks"`
	}
	readJSONFile(t, path, &file)
	if len(file.Hooks.Stop) != 1 || len(file.Hooks.Stop[0].Hooks) != 1 {
		t.Fatalf("%s: want exactly one Stop command, got %+v", path, file.Hooks.Stop)
	}
	return file.Hooks.Stop[0].Hooks[0].Command
}

// The Stop hook lints the project, not whatever directory the session's
// shell was left in. An agent that changed into a temporary directory leaves
// the hook process there, where bd finds no beads database. Claude Code names
// the project in $CLAUDE_PROJECT_DIR; Codex sets no such variable and its hook
// input's cwd is the project.
//
// The package is also installed user-wide, so the hook runs in projects that
// keep no issues at all. There bd finds no database from the project either,
// and the hook still allows the turn.
func TestIssuesLintStopHookLintsProjectDir(t *testing.T) {
	requireMermaidLint(t)

	claudeHooks := filepath.Join(
		apmSkillPluginDir("crabswarm-issues-lint"),
		"hooks",
		"hooks.json",
	)
	codexHooks := filepath.Join(
		apmPackageDir("crabswarm-issues-lint"),
		".apm",
		"hooks",
		"codex-hooks.json",
	)
	cases := []struct {
		name       string
		hooksFile  string
		projectEnv bool
		// noBeads makes bd answer as if no database governs the project.
		noBeads bool
	}{
		{name: "claude", hooksFile: claudeHooks, projectEnv: true},
		{name: "codex", hooksFile: codexHooks},
		{name: "claude no beads", hooksFile: claudeHooks, projectEnv: true, noBeads: true},
		{name: "codex no beads", hooksFile: codexHooks, noBeads: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			project, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			drifted := t.TempDir()
			// The fake bd answers only from governed, so a project without
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
			writeFile(t, listPath, bdListWithoutFences)

			inputCwd := drifted
			var env []string
			for _, kv := range os.Environ() {
				if !strings.HasPrefix(kv, "CLAUDE_PROJECT_DIR=") {
					env = append(env, kv)
				}
			}
			env = append(env,
				"PATH="+bdDir+string(os.PathListSeparator)+
					filepath.Dir(crabswarmBin)+string(os.PathListSeparator)+os.Getenv("PATH"),
				"FAKE_BD_LIST="+listPath,
				"FAKE_BD_PROJECT="+governed,
			)
			if tc.projectEnv {
				env = append(env, "CLAUDE_PROJECT_DIR="+project)
			} else {
				inputCwd = project
			}

			command := stopHookCommand(t, tc.hooksFile)
			cmd := exec.CommandContext(t.Context(), "sh", "-c", command)
			cmd.Dir = drifted
			cmd.Env = env
			cmd.Stdin = strings.NewReader(`{"hook_event_name":"Stop","session_id":"e2e",` +
				`"cwd":` + strconv.Quote(inputCwd) + `,` +
				`"transcript_path":"/dev/null","stop_hook_active":false}`)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf(
					"%s: %v\nstdout:\n%s\nstderr:\n%s",
					command,
					err,
					stdout.String(),
					stderr.String(),
				)
			}
			if stdout.String() != "" {
				t.Errorf("hook output = %q, want none (allow)", stdout.String())
			}
		})
	}
}
