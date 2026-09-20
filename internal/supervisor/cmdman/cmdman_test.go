package cmdman

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/internal/supervisor"
)

// fakeCmdmanScript is a stand-in for the real cmdman binary, with the three
// paths it works against baked in. It appends every invocation (subcommand +
// args, space-joined, one per line) to the log, and:
//   - inspect prints the state file and exits 0 when that file exists,
//     otherwise exits 1 the way cmdman reports an unknown name;
//   - run fails with output on stderr when the fail marker exists;
//   - everything else succeeds silently.
const fakeCmdmanScript = `#!/bin/sh
printf '%%s\n' "$*" >> '%[1]s'

case "$1" in
inspect)
  if [ -f '%[2]s' ]; then
    cat '%[2]s'
    exit 0
  fi
  echo 'error: resolve command: no command found matching' >&2
  exit 1
  ;;
run)
  if [ -f '%[3]s' ]; then
    echo 'error: cmdman refused the run' >&2
    exit 1
  fi
  ;;
esac
exit 0
`

type fakeCmdman struct {
	// bin is what the tests pass to New.
	bin string
	log string
}

// newFakeCmdman writes the fake cmdman into a temp dir. state, when non-empty,
// makes inspect report a present command in that state; empty means absent.
// failRun makes run exit non-zero.
//
// The tests around it stay sequential: a fork issued by one test keeps the file
// descriptor another test still has open for writing its script, and exec'ing
// that script during the window fails with ETXTBSY.
func newFakeCmdman(t *testing.T, state string, failRun bool) fakeCmdman {
	t.Helper()

	dir := t.TempDir()
	fake := fakeCmdman{
		bin: filepath.Join(dir, "cmdman"),
		log: filepath.Join(dir, "invocations.log"),
	}
	statePath := filepath.Join(dir, "state")
	failPath := filepath.Join(dir, "fail")

	script := fmt.Sprintf(fakeCmdmanScript, fake.log, statePath, failPath)
	assert.NilError(t, os.WriteFile(fake.bin, []byte(script), 0o755))
	if state != "" {
		assert.NilError(t, os.WriteFile(statePath, []byte(state+"\n"), 0o600))
	}
	if failRun {
		assert.NilError(t, os.WriteFile(failPath, nil, 0o600))
	}
	return fake
}

// invocations returns the recorded cmdman invocations, one per line. A missing
// log (cmdman never invoked) reads as no invocations.
func (f fakeCmdman) invocations(t *testing.T) []string {
	t.Helper()

	b, err := os.ReadFile(f.log)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	assert.NilError(t, err)

	var lines []string
	for l := range strings.SplitSeq(strings.TrimSpace(string(b)), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func TestInspectAbsent(t *testing.T) {
	fake := newFakeCmdman(t, "", false)

	state, err := New(fake.bin).Inspect(t.Context(), "web")
	assert.Assert(t, errors.Is(err, supervisor.ErrNotFound), "got %v", err)
	assert.Equal(t, state, supervisor.State(""))

	got := fake.invocations(t)
	assert.Equal(t, len(got), 1, "invocations: %v", got)
	assert.Equal(t, got[0], "inspect web --format {{.State}}")
}

func TestInspectPresent(t *testing.T) {
	fake := newFakeCmdman(t, "exited", false)

	state, err := New(fake.bin).Inspect(t.Context(), "web")
	assert.NilError(t, err)
	assert.Equal(t, state, supervisor.State("exited"))
}

func TestRemove(t *testing.T) {
	fake := newFakeCmdman(t, "exited", false)

	assert.NilError(t, New(fake.bin).Remove(t.Context(), "web"))

	got := fake.invocations(t)
	assert.Equal(t, len(got), 1, "invocations: %v", got)
	assert.Equal(t, got[0], "rm web")
}

func TestRunNamed(t *testing.T) {
	fake := newFakeCmdman(t, "", false)

	assert.NilError(t, New(fake.bin).Run(t.Context(), "web", []string{"sleep", "1"}))

	got := fake.invocations(t)
	assert.Equal(t, len(got), 1, "invocations: %v", got)
	assert.Equal(t, got[0], "run --name web -- sleep 1")
}

func TestRunAnonymous(t *testing.T) {
	fake := newFakeCmdman(t, "", false)

	assert.NilError(t, New(fake.bin).Run(t.Context(), "", []string{"sleep", "1"}))

	got := fake.invocations(t)
	assert.Equal(t, len(got), 1, "invocations: %v", got)
	assert.Equal(t, got[0], "run -- sleep 1")
}

func TestRunFails(t *testing.T) {
	fake := newFakeCmdman(t, "", true)

	err := New(fake.bin).Run(t.Context(), "web", []string{"sleep", "1"})
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "web"), "got %v", err)
	assert.Assert(t, strings.Contains(err.Error(), "cmdman refused the run"), "got %v", err)
}

// An anonymous run owns no name, so its error names the command line instead.
func TestRunFailsAnonymous(t *testing.T) {
	fake := newFakeCmdman(t, "", true)

	err := New(fake.bin).Run(t.Context(), "", []string{"sleep", "1"})
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "sleep 1"), "got %v", err)
}
