package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// hermeticEnv isolates LoadConfig from the host environment so the command
// wiring tests are deterministic: no CRABSWARM_* / CLAUDE_PROJECT_DIR overrides
// and an empty config dir (so no config.json is discovered). XDG_RUNTIME_DIR
// and HOME are fixed so the resolved defaults are predictable.
func hermeticEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"CRABSWARM_SOCK",
		"CRABSWARM_CONF",
		"CRABSWARM_GIT_REPO_BASE_DIR",
		"CRABSWARM_GIT_LIST_IGNORE_PATTERNS",
		"CRABSWARM_CHAT_DB",
		"CRABSWARM_CHAT_CMDMAN_BIN",
		"CRABSWARM_CHAT_ADMIN_RECIPIENTS",
		"CRABSWARM_CHAT_ADMIN_IDENTITY_FILE",
		"CRABSWARM_PREVIEW_ADDR",
		"CRABSWARM_PREVIEW_DAEMON_NAME",
		"CRABSWARM_MCP_CODEX_PROXY_SOCK_DIR",
		"CLAUDE_PROJECT_DIR",
		"XDG_RUNTIME_DIR",
	} {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	t.Setenv("HOME", "/home/tester")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // empty dir -> no config.json
}

// runConfigCmd executes "config" with args through the real root command,
// capturing stdout and stderr separately. SetOut/SetErr on the root propagate
// to the subcommand via cobra's OutOrStdout/ErrOrStderr parent walk.
func runConfigCmd(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := rootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs(append([]string{"config"}, args...))
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

// The resolved config is written to stdout, not stderr — the regression this
// fixes (cobra's cmd.Println routes to stderr). Rendering correctness is covered
// by crabswarm/cli; here we assert only the command wiring.
func TestConfigCmd_WritesToStdout(t *testing.T) {
	hermeticEnv(t)

	stdout, stderr, err := runConfigCmd(t)
	assert.NilError(t, err)
	assert.Equal(t, stderr, "")
	assert.Assert(t, strings.Contains(stdout, `"sock"`))
	assert.Assert(t, strings.Contains(stdout,
		`"codex_proxy_sock_dir": "/run/user/1000/crabswarm-codex-proxy"`), stdout)
}

// The --format flag (and its -f shorthand) is wired through to the renderer, and
// its output also goes to stdout.
func TestConfigCmd_FormatFlagToStdout(t *testing.T) {
	hermeticEnv(t)

	stdout, stderr, err := runConfigCmd(t, "-f", "{{.GitRepoBaseDir}}")
	assert.NilError(t, err)
	assert.Equal(t, stderr, "")
	assert.Equal(t, strings.TrimSpace(stdout), "/home/tester/gitrepo")
}

// The codex-proxy socket directory is printed under the mcp section, and the
// environment reaches it as CRABSWARM_MCP_CODEX_PROXY_SOCK_DIR.
func TestConfigCmd_PrintsTheCodexProxySockDir(t *testing.T) {
	hermeticEnv(t)
	t.Setenv("CRABSWARM_MCP_CODEX_PROXY_SOCK_DIR", "/env/proxy")

	stdout, _, err := runConfigCmd(t, "-f", "{{.MCP.CodexProxySockDir}}")
	assert.NilError(t, err)
	assert.Equal(t, strings.TrimSpace(stdout), "/env/proxy")
}
