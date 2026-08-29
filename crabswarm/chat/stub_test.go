package chat

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// logArgs is a stub prologue that appends the invocation's arguments to
// "args.log" next to the stub. The stub locates the file relative to $0 rather
// than through the environment so tests need no t.Setenv and stay independent.
const logArgs = "printf '%s\\n' \"$*\" >> \"$(dirname \"$0\")/args.log\"\n"

// stubCmdman writes a stand-in cmdman whose body is the given shell script and
// returns its absolute path. [NewCmdmanStatusMirror] takes the binary path
// directly, so nothing here touches PATH.
//
// The tests using it do not call t.Parallel: writing an executable in one test
// while another forks makes the child inherit the still-open write descriptor,
// and the exec then fails with ETXTBSY.
//
// The notify and resolver packages keep their own copies rather than sharing
// this one: a test helper exported for another package's tests would have to
// live in non-test code, which is a worse trade than a few short functions.
func stubCmdman(t *testing.T, body string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "cmdman")
	assert.NilError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"+body), 0o755))
	return bin
}

// stubArgs returns the argument lines the stub at bin recorded, one invocation
// per line. A missing log means the stub was never run.
func stubArgs(t *testing.T, bin string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(filepath.Dir(bin), "args.log"))
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
