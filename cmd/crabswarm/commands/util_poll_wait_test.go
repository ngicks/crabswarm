package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

// runUtilCmd executes "util <args...>" through the real root command,
// capturing stdout and stderr separately (see runPreviewCmd).
func runUtilCmd(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := rootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs(append([]string{"util"}, args...))
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

// An existing file satisfies a file:// target on the first probe, so the
// command returns at once without spending any of the retry budget.
func TestUtilPollWait_FileTargetReady(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ready")
	assert.NilError(t, os.WriteFile(path, []byte("ok"), 0o600))

	_, _, err := runUtilCmd(t, "poll", "wait", "file://"+path)
	assert.NilError(t, err)
}

// wait takes exactly one target, so calling it with none fails before any
// probe runs.
func TestUtilPollWait_MissingArg(t *testing.T) {
	_, _, err := runUtilCmd(t, "poll", "wait")
	assert.Assert(t, err != nil)
}
