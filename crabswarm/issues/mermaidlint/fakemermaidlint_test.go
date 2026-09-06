package mermaidlint

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// fakeMermaidLintScript is a stand-in for the real mermaid-lint binary.
// Every invocation is appended to the file named by FAKE_MERMAID_LINT_LOG
// as one tab-separated line — working directory, the space-joined argv,
// then the names present in that directory — so a test can assert both the
// command line and the set of files that were written for it. Output is
// the report recorded in FAKE_MERMAID_LINT_REPORT, or, when that is empty,
// the line in FAKE_MERMAID_LINT_MESSAGE, standing for the message the real
// binary prints on stdout instead of a report when a run fails outright.
// The exit status is FAKE_MERMAID_LINT_EXIT, defaulting to 1, which is
// what the real binary returns whenever it refused a diagram.
const fakeMermaidLintScript = `#!/bin/sh
{
  printf '%s\t' "$(pwd)"
  sep=
  for a in "$@"; do
    printf '%s%s' "$sep" "$a"
    sep=' '
  done
  printf '\t'
  sep=
  for f in *; do
    printf '%s%s' "$sep" "$f"
    sep=' '
  done
  printf '\n'
} >> "$FAKE_MERMAID_LINT_LOG"

if [ -n "$FAKE_MERMAID_LINT_REPORT" ]; then
  cat "$FAKE_MERMAID_LINT_REPORT"
else
  echo "$FAKE_MERMAID_LINT_MESSAGE"
fi
exit "${FAKE_MERMAID_LINT_EXIT:-1}"
`

// invocation is one recorded fake mermaid-lint call.
type invocation struct {
	dir   string // the directory the binary ran in
	args  string // argv, space-joined
	files string // the names in that directory, sorted, space-joined
}

// installFakeMermaidLint writes the fake mermaid-lint onto a fresh dir
// prepended to PATH, replaying the named fixture from testdata, and
// returns a function reading back the invocations recorded so far. Pass an
// empty fixture to make the fake print FAKE_MERMAID_LINT_MESSAGE instead
// of a report. It uses t.Setenv, so the test cannot be parallel.
func installFakeMermaidLint(t *testing.T, fixture string) (invocations func() []invocation) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "mermaid-lint")
	assert.NilError(t, os.WriteFile(bin, []byte(fakeMermaidLintScript), 0o755))

	report := ""
	if fixture != "" {
		abs, err := filepath.Abs(filepath.Join("testdata", fixture))
		assert.NilError(t, err)
		report = abs
	}

	logPath := filepath.Join(dir, "invocations.log")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_MERMAID_LINT_LOG", logPath)
	t.Setenv("FAKE_MERMAID_LINT_REPORT", report)
	// Inherited values must not leak into a run that does not set them.
	t.Setenv("FAKE_MERMAID_LINT_MESSAGE", "")
	t.Setenv("FAKE_MERMAID_LINT_EXIT", "")

	return func() []invocation {
		t.Helper()
		b, err := os.ReadFile(logPath)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		assert.NilError(t, err)
		var out []invocation
		for line := range strings.SplitSeq(strings.TrimSuffix(string(b), "\n"), "\n") {
			if line == "" {
				continue
			}
			f := strings.Split(line, "\t")
			assert.Equal(t, len(f), 3, "malformed invocation line %q", line)
			out = append(out, invocation{dir: f[0], args: f[1], files: f[2]})
		}
		return out
	}
}
