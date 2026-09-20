package crabswarm_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// runUtil runs `crabswarm util <args...>` under env and returns its trimmed
// stdout, trimmed stderr and exit code. A nil env inherits the test's own.
//
// Unlike mustRunCrabswarm it tolerates a non-zero exit, because half of the
// cases below assert on a rejected invocation; only a failure to run the binary
// at all, or a run that outlives timeout, stops the test.
func runUtil(
	t *testing.T,
	env []string,
	timeout time.Duration,
	args ...string,
) (stdout, stderr string, exitCode int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	defer cancel()

	stdout, stderr, err := runCrabswarmEnv(ctx, t, env, append([]string{"util"}, args...)...)
	if ctx.Err() != nil {
		t.Fatalf("crabswarm util %s did not finish within %s\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), timeout, stdout, stderr)
	}
	if err != nil {
		exitErr, ok := errors.AsType[*exec.ExitError](err)
		if !ok {
			t.Fatalf("crabswarm util %s: %v\nstdout:\n%s\nstderr:\n%s",
				strings.Join(args, " "), err, stdout, stderr)
		}
		exitCode = exitErr.ExitCode()
	}
	return stdout, stderr, exitCode
}

// shortSocketDir returns a fresh directory short enough to hold a unix socket
// path. A unix address is capped near 108 bytes and t.TempDir() spends much of
// that budget on the test's own name, so the socket cases place theirs directly
// under the system temp directory instead.
func shortSocketDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "cs")
	if err != nil {
		t.Fatalf("create socket dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

// A live unix socket satisfies both spellings of a socket target: the unix://
// URI and the bare path. Nothing ever accepts on the listener, because a dial
// to a listening unix socket completes from the backlog alone — which is
// exactly the readiness signal the probe reads.
func TestUtilPollWaitUnixSocket(t *testing.T) {
	sock := filepath.Join(shortSocketDir(t), "s.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen on %s: %v", sock, err)
	}
	defer func() { _ = ln.Close() }()

	for name, target := range map[string]string{
		"unix scheme": "unix://" + sock,
		"bare path":   sock,
	} {
		t.Run(name, func(t *testing.T) {
			stdout, stderr, code := runUtil(t, nil, 15*time.Second, "poll", "wait", target)
			if code != 0 {
				t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s",
					code, stdout, stderr)
			}
		})
	}
}

// A file:// target that only appears after the command started shows wait
// really waiting rather than probing once: the command cannot return before the
// writer created the file, so the delay is a lower bound on its runtime.
func TestUtilPollWaitFileAppearsLate(t *testing.T) {
	const delay = 150 * time.Millisecond

	path := filepath.Join(t.TempDir(), "ready")
	// The writer reports through a channel instead of calling t.Fatalf, which
	// only the test goroutine may do; receiving from it also joins the goroutine.
	// The clock starts before the writer, so the elapsed time below is a lower
	// bound the writer's own delay can only exceed.
	start := time.Now()
	written := make(chan error, 1)
	go func() {
		time.Sleep(delay)
		written <- os.WriteFile(path, []byte("ready\n"), 0o644)
	}()

	stdout, stderr, code := runUtil(t, nil, 15*time.Second,
		"poll", "wait", "file://"+path, "--interval", "50ms")
	elapsed := time.Since(start)

	if err := <-written; err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if elapsed < delay {
		t.Errorf("wait returned after %s, before the file existed at %s", elapsed, delay)
	}
}

// --start-period holds the first probe back, so an endpoint that is ready all
// along is still left alone until it elapses. The clock starts before the
// command is spawned, so process start-up can only widen the margin the
// handler measures.
func TestUtilPollWaitHTTPStartPeriod(t *testing.T) {
	const startPeriod = 300 * time.Millisecond

	var (
		mu       sync.Mutex
		requests int
		firstAt  time.Time
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		requests++
		if requests == 1 {
			firstAt = time.Now()
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	start := time.Now()
	stdout, stderr, code := runUtil(t, nil, 15*time.Second,
		"poll", "wait", srv.URL+"/ready",
		"--interval", "50ms", "--start-period", startPeriod.String())
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}

	mu.Lock()
	defer mu.Unlock()
	if requests != 1 {
		t.Fatalf("handler saw %d requests, want exactly one", requests)
	}
	if delay := firstAt.Sub(start); delay < startPeriod {
		t.Errorf("first probe arrived %s after the command started, want at least %s",
			delay, startPeriod)
	}
}

// A target that never appears exhausts the retry budget and fails, naming how
// many probes were spent so a script's log says why the wait gave up.
func TestUtilPollWaitRetriesExhausted(t *testing.T) {
	stdout, stderr, code := runUtil(t, nil, 15*time.Second,
		"poll", "wait", "--interval", "20ms", "--retries", "3",
		"file:///nonexistent/crabswarm-e2e/never")
	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "3 failed probes") {
		t.Errorf("stderr does not report the failed-probe count; got:\n%s", stderr)
	}
}

// fakeCmdmanRecorder returns a cmdman stand-in that appends every invocation's
// argv to logPath, one line per call, so a case can assert which subcommands
// supervised issued and with which arguments. inspectBody is the shell the
// `inspect` subcommand runs: exiting non-zero makes the command look absent,
// printing a state makes it look known. `run` creates readyPath, which is what
// a --poll file target waits for; nothing is actually supervised.
//
// The paths are substituted rather than formatted in, so the script's own
// printf format needs no %% escaping. fakeBinEnv also replaces PATH with the
// fake bin directory alone, so the script may only use shell builtins — touch
// and tee are not reachable from it.
func fakeCmdmanRecorder(logPath, readyPath, inspectBody string) string {
	return strings.NewReplacer(
		"@LOG@", logPath,
		"@READY@", readyPath,
		"@INSPECT@", inspectBody,
	).Replace(`#!/bin/sh
printf '%s\n' "$*" >> '@LOG@'
case "$1" in
inspect) @INSPECT@ ;;
run) : > '@READY@' ;;
esac
exit 0
`)
}

// inertCmdmanScript answers every cmdman invocation with success and does
// nothing else. The two rejection cases below must fail before cmdman is
// reached at all; it stands on PATH so a regression that reaches it shows up as
// a failed assertion here instead of a command started on the real machine.
const inertCmdmanScript = `#!/bin/sh
exit 0
`

// cmdmanCalls reads back the argv lines the fake cmdman recorded. A missing log
// means cmdman was never invoked.
func cmdmanCalls(t *testing.T, logPath string) []string {
	t.Helper()
	b, err := os.ReadFile(logPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("read cmdman log %s: %v", logPath, err)
	}
	trimmed := strings.TrimSpace(string(b))
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

// A name cmdman does not know is started straight away: inspect reports it
// absent, no rm is needed to free the name, and the argv reaches cmdman behind
// "--" so the supervised command's own words stay its own. The --poll target is
// the file the fake run creates, so exit 0 also means the readiness wait ran
// against something the start actually produced.
func TestUtilSupervisedStartsCommand(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "cmdman.log")
	readyPath := filepath.Join(dir, "ready")

	env := fakeBinEnv(t, map[string]string{
		"cmdman": fakeCmdmanRecorder(logPath, readyPath, "exit 1"),
	})

	stdout, stderr, code := runUtil(t, env, 15*time.Second,
		"supervised", "--name", "web", "--poll", "file://"+readyPath,
		"--poll-interval", "50ms",
		"--", "some", "command", "args")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}

	calls := cmdmanCalls(t, logPath)
	var runs []string
	for _, call := range calls {
		if strings.HasPrefix(call, "rm ") {
			t.Errorf("supervised removed a command cmdman reported as absent: %q", call)
		}
		if strings.HasPrefix(call, "run ") {
			runs = append(runs, call)
		}
	}
	if len(runs) != 1 {
		t.Fatalf("cmdman run was called %d times, want once; calls:\n%s",
			len(runs), strings.Join(calls, "\n"))
	}
	if want := "run --name web -- some command args"; runs[0] != want {
		t.Errorf("cmdman argv = %q, want %q", runs[0], want)
	}
}

// A name cmdman already reports as running is left alone, so rerunning the same
// supervised command is safe. The ready file is created up front because the
// readiness wait still runs for an adopted command and nothing else would
// create it — the fake run never fires.
func TestUtilSupervisedAlreadyRunning(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "cmdman.log")
	readyPath := filepath.Join(dir, "ready")
	writeFile(t, readyPath, "ready\n")

	env := fakeBinEnv(t, map[string]string{
		"cmdman": fakeCmdmanRecorder(logPath, readyPath, "echo running"),
	})

	stdout, stderr, code := runUtil(t, env, 15*time.Second,
		"supervised", "--name", "web", "--poll", "file://"+readyPath,
		"--poll-interval", "50ms",
		"--", "some", "command", "args")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}

	calls := cmdmanCalls(t, logPath)
	for _, call := range calls {
		if strings.HasPrefix(call, "run ") {
			t.Errorf("supervised restarted a command cmdman reported as running: %q", call)
		}
	}
	if len(calls) != 1 || !strings.HasPrefix(calls[0], "inspect ") {
		t.Errorf("cmdman calls = %q, want the inspect alone", calls)
	}
}

// Without "--" pflag may already have eaten a flag meant for the supervised
// command, so the invocation is refused rather than guessed at, and the message
// names the separator the caller has to add.
func TestUtilSupervisedRequiresDashDash(t *testing.T) {
	env := fakeBinEnv(t, map[string]string{"cmdman": inertCmdmanScript})

	stdout, stderr, code := runUtil(t, env, 15*time.Second, "supervised", "some", "command")
	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, `must follow "--"`) {
		t.Errorf("stderr does not point at the missing separator; got:\n%s", stderr)
	}
}

// cmdman is the only supervisor implemented, and an unknown one is rejected
// before anything is spawned rather than after a command already started under
// the wrong owner.
func TestUtilSupervisedUnsupportedSupervisor(t *testing.T) {
	env := fakeBinEnv(t, map[string]string{"cmdman": inertCmdmanScript})

	stdout, stderr, code := runUtil(t, env, 15*time.Second,
		"supervised", "--supervisor", "systemd", "--", "true")
	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "unsupported supervisor") {
		t.Errorf("stderr does not name the rejected supervisor; got:\n%s", stderr)
	}
}
