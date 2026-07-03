package commands

import (
	"bytes"
	"net"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// runPreviewCmd executes "preview <args...>" through the real root command,
// capturing stdout and stderr separately (see runConfigCmd).
func runPreviewCmd(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := rootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs(append([]string{"preview"}, args...))
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

// closedAddr returns a 127.0.0.1 address that is guaranteed to have nothing
// listening: it binds an ephemeral port to reserve it, then closes the listener
// so a dial to it is refused.
func closedAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NilError(t, err)
	addr := l.Addr().String()
	assert.NilError(t, l.Close())
	return addr
}

// The preview command tree is wired: `list` and `remove` are visible
// subcommands and the daemon entry point `__serve` is present but hidden.
func TestPreviewCmd_Subcommands(t *testing.T) {
	root := rootCmd()
	preview, _, err := root.Find([]string{"preview"})
	assert.NilError(t, err)
	assert.Equal(t, preview.Name(), "preview")

	names := map[string]bool{}
	hidden := map[string]bool{}
	for _, c := range preview.Commands() {
		names[c.Name()] = true
		hidden[c.Name()] = c.Hidden
	}
	assert.Assert(t, names["list"])
	assert.Assert(t, names["remove"])
	assert.Assert(t, names["__serve"])
	assert.Assert(t, hidden["__serve"])
}

// `preview list` against an address with no daemon fails with the unreachable
// hint (not a rendered table), and writes nothing to stdout.
func TestPreviewList_UnreachableDaemonHint(t *testing.T) {
	hermeticEnv(t)

	stdout, _, err := runPreviewCmd(t, "list", "--addr", closedAddr(t))
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "unreachable"))
	assert.Equal(t, stdout, "")
}

// `preview remove` against an address with no daemon gives the same hint.
func TestPreviewRemove_UnreachableDaemonHint(t *testing.T) {
	hermeticEnv(t)

	_, _, err := runPreviewCmd(t, "remove", "some-root", "--addr", closedAddr(t))
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "unreachable"))
}
