package commands

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/crabswarm/mcp/codexproxy"
)

// proxyShortDir is a directory short enough to hold a unix socket, which a
// test's own temporary directory is not.
func proxyShortDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "cxp")
	assert.NilError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

// proxyRuntimeDir points $XDG_RUNTIME_DIR at a directory short enough for a
// unix socket, so the proxy never serves under the host's own.
func proxyRuntimeDir(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_RUNTIME_DIR", proxyShortDir(t))
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

// The socket directory the config names, in the file or in the environment, is
// where the command finds the proxy's socket.
func TestMCPCodexProxyCmd_ServesInTheConfiguredSockDir(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh to run a command with")
	}
	for _, tc := range []struct {
		name     string
		fromFile bool
	}{
		{name: "from the config file", fromFile: true},
		{name: "from the environment"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chatHermeticEnv(t)
			proxyRuntimeDir(t)
			sockDir := filepath.Join(proxyShortDir(t), "configured")
			args := []string{"codex-proxy"}
			if tc.fromFile {
				conf := filepath.Join(t.TempDir(), "config.json")
				raw, err := json.Marshal(map[string]any{
					"mcp": map[string]string{"codex_proxy_sock_dir": sockDir},
				})
				assert.NilError(t, err)
				assert.NilError(t, os.WriteFile(conf, raw, 0o600))
				args = append(args, "--config", conf)
			} else {
				t.Setenv("CRABSWARM_MCP_CODEX_PROXY_SOCK_DIR", sockDir)
			}
			out := filepath.Join(t.TempDir(), "remote")
			t.Setenv("CODEXPROXY_TEST_REMOTE", out)

			assert.NilError(t, runMCPCmd(t.Context(), append(args,
				"--", sh, "-c",
				`test -S "${2#unix://}" || exit 99
printf '%s\n' "$2" > "$CODEXPROXY_TEST_REMOTE"`,
				"sh", "--remote", "unix:///nowhere.sock")...))

			got, err := os.ReadFile(out)
			assert.NilError(t, err)
			want := filepath.Join(sockDir, strconv.Itoa(os.Getpid())+".sock")
			assert.Equal(t, string(got), "unix://"+want+"\n")
		})
	}
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
