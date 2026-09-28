package crabswarm_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// `crabswarm mcp codex-proxy` serves its socket in the directory the config
// names, whether the config file or the environment names it, and leaves the
// default under the runtime directory alone. `crabswarm config` prints the same
// directory the proxy serves in.
func TestMCPCodexProxy_ServesInTheConfiguredSockDir(t *testing.T) {
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
			runtimeDir := shortSocketDir(t)
			sockDir := filepath.Join(shortSocketDir(t), "configured")
			out := filepath.Join(t.TempDir(), "remote")
			env := append(chatEnviron(),
				"XDG_RUNTIME_DIR="+runtimeDir,
				"XDG_CONFIG_HOME="+t.TempDir(),
				"E2E_CODEX_PROXY_REMOTE="+out,
			)
			var global []string
			if tc.fromFile {
				conf := filepath.Join(t.TempDir(), "config.json")
				writeFile(t, conf, fmt.Sprintf(`{"mcp":{"codex_proxy_sock_dir":%q}}`, sockDir))
				global = []string{"--config", conf}
			} else {
				env = append(env, "CRABSWARM_MCP_CODEX_PROXY_SOCK_DIR="+sockDir)
			}

			config := exec.Command(crabswarmBin,
				append(global, "config", "--format", "{{.MCP.CodexProxySockDir}}")...)
			config.Env = env
			printed, err := config.Output()
			if err != nil {
				t.Fatalf("crabswarm config: %v", err)
			}
			if got := strings.TrimSpace(string(printed)); got != sockDir {
				t.Errorf("crabswarm config printed %q, want %q", got, sockDir)
			}

			proxy := exec.Command(crabswarmBin, append(global,
				"mcp", "codex-proxy", "--", sh, "-c",
				`test -S "${2#unix://}" || exit 99
printf '%s\n' "$2" > "$E2E_CODEX_PROXY_REMOTE"`,
				"sh", "--remote", "unix:///nowhere.sock")...)
			proxy.Env = env
			proxy.Stderr = os.Stderr
			if err := proxy.Run(); err != nil {
				t.Fatalf("crabswarm mcp codex-proxy: %v", err)
			}

			sock := filepath.Join(sockDir, strconv.Itoa(proxy.Process.Pid)+".sock")
			remote, err := os.ReadFile(out)
			if err != nil {
				t.Fatalf("the command never wrote the address it was handed: %v", err)
			}
			if got, want := string(remote), "unix://"+sock+"\n"; got != want {
				t.Errorf("the command was handed %q, want %q", got, want)
			}
			if _, err := os.Stat(sock); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("the socket outlived the proxy: %v", err)
			}
			info, err := os.Stat(sockDir)
			if err != nil {
				t.Fatalf("stat the socket directory: %v", err)
			}
			if perm := info.Mode().Perm(); perm != 0o700 {
				t.Errorf("the socket directory is %v, want 0700", perm)
			}
			fallback := filepath.Join(runtimeDir, "crabswarm-codex-proxy")
			if _, err := os.Stat(fallback); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("the default directory %s was made: %v", fallback, err)
			}
		})
	}
}
