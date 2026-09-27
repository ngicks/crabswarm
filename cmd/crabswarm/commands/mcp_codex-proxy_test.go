package commands

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/crabswarm/mcp/codexproxy"
)

// proxyRuntimeDir points $XDG_RUNTIME_DIR at a directory short enough for a
// unix socket, so the proxy never serves under the host's own.
func proxyRuntimeDir(t *testing.T) {
	t.Helper()
	dir, err := os.MkdirTemp("", "cxp")
	assert.NilError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	t.Setenv("XDG_RUNTIME_DIR", dir)
}

// A command line the proxy cannot put itself in front of is refused, and the
// refusal says what to write instead.
func TestMCPCodexProxyCmd_RefusesACommandLineItCannotRelay(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{
			name: "without a dash",
			args: []string{"codex"},
			want: `must follow "--"`,
		},
		{
			name: "a dash after the command",
			args: []string{"codex", "--", "prompt"},
			want: `must follow "--"`,
		},
		{
			name: "no remote",
			args: []string{"--", "codex", "-m", "gpt"},
			want: "names no app server",
		},
		{
			name: "a websocket remote",
			args: []string{"--", "codex", "--remote", "ws://127.0.0.1:1"},
			want: "unix://",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chatHermeticEnv(t)
			proxyRuntimeDir(t)

			err := runMCPCmd(t.Context(), append([]string{"codex-proxy"}, tc.args...)...)
			assert.ErrorContains(t, err, tc.want)
		})
	}
}

// The command runs what follows "--", handed the proxy's socket, and returns
// cleanly when it exits with status zero.
func TestMCPCodexProxyCmd_RunsTheCommand(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh to run a command with")
	}
	chatHermeticEnv(t)
	proxyRuntimeDir(t)
	out := filepath.Join(t.TempDir(), "socket")
	t.Setenv("CODEXPROXY_TEST_SOCKET", out)

	assert.NilError(t, runMCPCmd(t.Context(),
		"codex-proxy", "--token", "tok", "--server-name", "cs",
		"--", sh, "-c",
		`if test -S "${2#unix://}"; then echo served; else echo missing; fi `+
			`> "$CODEXPROXY_TEST_SOCKET"; exit 0`,
		"sh", "--remote", "unix:///nowhere.sock"))

	got, err := os.ReadFile(out)
	assert.NilError(t, err)
	assert.Equal(t, string(got), "served\n")
}

// A command that exits unsuccessfully makes the command return an error
// carrying that status, which the process then exits with.
func TestMCPCodexProxyCmd_ReturnsTheCommandsStatus(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh to run a command with")
	}
	chatHermeticEnv(t)
	proxyRuntimeDir(t)

	err = runMCPCmd(t.Context(),
		"codex-proxy", "--token", "tok",
		"--", sh, "-c", "exit 7", "sh", "--remote", "unix:///nowhere.sock")

	exitErr, ok := errors.AsType[*codexproxy.ExitError](err)
	assert.Assert(t, ok, "err = %v", err)
	assert.Equal(t, exitErr.Code, 7)
	assert.Equal(t, exitErr.ExitStatus(), 7)
}
