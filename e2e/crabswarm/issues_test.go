package crabswarm_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeBdListScript is a stand-in for bd that answers `bd list --json` with
// the recorded listing in FAKE_BD_LIST and refuses everything else. The
// listing is the whole conversation `issues lint` has with bd as long as no
// issue carries comments, so a case that needs none can stop here.
const fakeBdListScript = `#!/bin/sh
case "$1" in
list) cat "$FAKE_BD_LIST" ;;
*) echo "fake bd: unsupported subcommand $*" >&2; exit 2 ;;
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

// runIssuesLint runs the built binary's `issues lint` with a fake bd on PATH
// answering with listing, from a temp directory so no mermaid-lint
// configuration around the checkout reaches the run. A non-zero exit is the
// command reporting findings, so only a failure to run at all is fatal.
func runIssuesLint(
	ctx context.Context,
	t *testing.T,
	listing string,
	args ...string,
) hookResult {
	t.Helper()

	bdDir := t.TempDir()
	writeFile(t, filepath.Join(bdDir, "bd"), fakeBdListScript)
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

	res := runIssuesLint(t.Context(), t, bdListWithFences)

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

	res := runIssuesLint(t.Context(), t, bdListWithoutFences, "--json")

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
