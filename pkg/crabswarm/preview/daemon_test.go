package preview

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// fakeCmdmanScript is a stand-in for the real cmdman binary. It appends every
// invocation (subcommand + args, space-joined, one per line) to the file named
// by FAKE_CMDMAN_LOG, and lets the test steer `inspect` via env:
//   - FAKE_CMDMAN_INSPECT_STATE unset/empty -> inspect exits 1 (daemon absent),
//     matching cmdman's "no command found" behavior.
//   - FAKE_CMDMAN_INSPECT_STATE set         -> inspect prints it and exits 0.
//
// `run` and `rm` just succeed; the fake never actually starts anything, so the
// tests stand up their own httptest /healthz server to satisfy the poll.
const fakeCmdmanScript = `#!/bin/sh
{
  first=1
  for a in "$@"; do
    if [ "$first" = 1 ]; then printf '%s' "$a"; first=0; else printf ' %s' "$a"; fi
  done
  printf '\n'
} >> "$FAKE_CMDMAN_LOG"

case "$1" in
inspect)
  if [ -n "$FAKE_CMDMAN_INSPECT_STATE" ]; then
    printf '%s\n' "$FAKE_CMDMAN_INSPECT_STATE"
    exit 0
  fi
  echo 'error: resolve command: no command found matching' >&2
  exit 1
  ;;
esac
exit 0
`

// installFakeCmdman writes the fake cmdman onto a fresh dir prepended to PATH
// and returns the path to its invocation log. inspectState, when non-empty,
// makes the fake's `inspect` report a present daemon in that state; empty means
// the daemon is absent. It uses t.Setenv, so the test cannot be parallel.
func installFakeCmdman(t *testing.T, inspectState string) (logPath string) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "cmdman")
	assert.NilError(t, os.WriteFile(bin, []byte(fakeCmdmanScript), 0o755))

	logPath = filepath.Join(dir, "invocations.log")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_CMDMAN_LOG", logPath)
	t.Setenv("FAKE_CMDMAN_INSPECT_STATE", inspectState)
	return logPath
}

// readInvocations returns the recorded cmdman invocations, one per line. A
// missing log (cmdman never invoked) reads as no invocations.
func readInvocations(t *testing.T, logPath string) []string {
	t.Helper()
	b, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
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

// healthzServer starts an httptest server answering 200 on /healthz and returns
// a wildcard listen address ("0.0.0.0:<port>") pointing at it. The wildcard host
// exercises EnsureDaemon's loopback rewrite: the daemon is reached at 127.0.0.1.
func healthzServer(t *testing.T) (addr string) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	assert.NilError(t, err)
	return "0.0.0.0:" + u.Port()
}

func TestEnsureDaemon_AlreadyRunning_DoesNotRun(t *testing.T) {
	logPath := installFakeCmdman(t, "running")
	addr := healthzServer(t)

	cfg := Config{Addr: addr, DaemonName: "crabswarm-preview"}
	err := EnsureDaemon(context.Background(), nil, cfg, "")
	assert.NilError(t, err)

	got := readInvocations(t, logPath)
	// Only the inspect happened; no run, no rm.
	assert.Equal(t, len(got), 1, "invocations: %v", got)
	assert.Assert(t, strings.HasPrefix(got[0], "inspect crabswarm-preview"), "got %q", got[0])
	for _, line := range got {
		assert.Assert(t, !strings.HasPrefix(line, "run "), "unexpected run: %q", line)
	}
}

func TestEnsureDaemon_NotPresent_RunsWithExpectedArgs(t *testing.T) {
	logPath := installFakeCmdman(t, "") // inspect -> absent
	addr := healthzServer(t)

	cfg := Config{Addr: addr, DaemonName: "crabswarm-preview"}
	err := EnsureDaemon(context.Background(), nil, cfg, "/etc/crabswarm/config.yml")
	assert.NilError(t, err)

	self, err := os.Executable()
	assert.NilError(t, err)

	got := readInvocations(t, logPath)
	assert.Assert(t, len(got) >= 2, "want inspect then run, got %v", got)
	assert.Assert(t, strings.HasPrefix(got[0], "inspect crabswarm-preview"), "got %q", got[0])

	runLine := findPrefixed(got, "run ")
	assert.Assert(t, runLine != "", "no run invocation in %v", got)
	// An absent daemon must not be rm'd before running.
	assert.Assert(t, findPrefixed(got, "rm ") == "", "unexpected rm in %v", got)

	wantRun := "run --name crabswarm-preview -- " + self +
		" preview __serve --addr " + addr + " --config /etc/crabswarm/config.yml"
	assert.Equal(t, runLine, wantRun)
}

func TestEnsureDaemon_NotPresent_OmitsConfigWhenEmpty(t *testing.T) {
	logPath := installFakeCmdman(t, "")
	addr := healthzServer(t)

	cfg := Config{Addr: addr, DaemonName: "crabswarm-preview"}
	err := EnsureDaemon(context.Background(), nil, cfg, "")
	assert.NilError(t, err)

	self, err := os.Executable()
	assert.NilError(t, err)

	runLine := findPrefixed(readInvocations(t, logPath), "run ")
	wantRun := "run --name crabswarm-preview -- " + self +
		" preview __serve --addr " + addr
	assert.Equal(t, runLine, wantRun)
}

func TestEnsureDaemon_StaleExited_RemovedThenRun(t *testing.T) {
	logPath := installFakeCmdman(t, "exited") // present but not running
	addr := healthzServer(t)

	cfg := Config{Addr: addr, DaemonName: "crabswarm-preview"}
	err := EnsureDaemon(context.Background(), nil, cfg, "")
	assert.NilError(t, err)

	got := readInvocations(t, logPath)
	// inspect -> rm (stale) -> run, in that order.
	assert.Assert(t, len(got) == 3, "want inspect, rm, run; got %v", got)
	assert.Assert(t, strings.HasPrefix(got[0], "inspect "), "got %q", got[0])
	assert.Equal(t, got[1], "rm crabswarm-preview")
	assert.Assert(
		t,
		strings.HasPrefix(got[2], "run --name crabswarm-preview -- "),
		"got %q",
		got[2],
	)
}

func TestEnsureDaemon_HealthzTimeout(t *testing.T) {
	installFakeCmdman(t, "running")

	// A closed port: nothing answers /healthz. Use a bounded ctx so the ~10s
	// poll budget does not stall the test.
	ctx, cancel := context.WithTimeout(context.Background(), 500*healthzInterval/100)
	defer cancel()

	cfg := Config{Addr: "0.0.0.0:6", DaemonName: "crabswarm-preview"}
	err := EnsureDaemon(ctx, nil, cfg, "")
	assert.Assert(t, err != nil, "want a timeout error when /healthz never answers")
	assert.Assert(t, strings.Contains(err.Error(), "did not become healthy"), "got %v", err)
}

func findPrefixed(lines []string, prefix string) string {
	for _, l := range lines {
		if strings.HasPrefix(l, prefix) {
			return l
		}
	}
	return ""
}
