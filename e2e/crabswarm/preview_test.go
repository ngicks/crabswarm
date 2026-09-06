package crabswarm_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"connectrpc.com/connect"

	previewv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/preview/v1"
	"github.com/ngicks/crabswarm/crabswarm/preview"
)

// TestPreviewServe exercises the previewer end-to-end without cmdman: it runs
// the hidden `preview __serve` command directly (the same process cmdman would
// otherwise daemonize), then drives it over the connect API — AddRoot,
// GetDocument, and a WatchEvents live-reload event triggered by writing to a
// watched markdown file.
func TestPreviewServe(t *testing.T) {
	// rpcCtx bounds the client-side RPCs and the WatchEvents stream; the serve
	// process has its own lifecycle terminated in Cleanup, so it is not tied to
	// this context.
	rpcCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	addr := freeAddr(t)

	root := t.TempDir()
	docPath := filepath.Join(root, "doc.md")
	writeFile(t, docPath, "# Hello World\n\nsome text\n")

	// Start `preview __serve --addr <addr>` directly — no cmdman dependency.
	serve := exec.Command(crabswarmBin, "preview", "__serve", "--addr", addr)
	serve.Stdout = os.Stderr
	serve.Stderr = os.Stderr
	if err := serve.Start(); err != nil {
		t.Fatalf("start preview __serve: %v", err)
	}
	t.Cleanup(func() { stopProcess(t, serve) })

	waitHealthz(rpcCtx, t, addr, 30*time.Second)

	client := preview.NewClient(addr)

	// AddRoot: register the temp directory.
	addResp, err := client.AddRoot(rpcCtx,
		connect.NewRequest(&previewv1.AddRootRequest{Path: root}))
	if err != nil {
		t.Fatalf("AddRoot: %v", err)
	}
	rootID := addResp.Msg.GetRoot().GetId()
	if rootID == "" {
		t.Fatal("AddRoot returned an empty root id")
	}

	// GetDocument: the first h1 becomes the title and appears in the rendered
	// HTML.
	docResp, err := client.GetDocument(rpcCtx,
		connect.NewRequest(&previewv1.GetDocumentRequest{RootId: rootID, Path: "doc.md"}))
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if got := docResp.Msg.GetTitle(); got != "Hello World" {
		t.Errorf("GetDocument title = %q, want %q", got, "Hello World")
	}
	if html := docResp.Msg.GetHtml(); !strings.Contains(html, "Hello World") {
		t.Errorf("GetDocument HTML missing heading text; got:\n%s", html)
	}

	// WatchEvents: a connect server-stream doesn't hand control back to the
	// client until the server flushes its first message, and this server's
	// WatchEvents handler stays silent until a file event arrives. So the whole
	// stream (establish + receive) runs in a goroutine while the test keeps
	// rewriting the watched file on a ticker until a DocChanged event flows.
	// Driving writes concurrently also covers the subscription-registration race
	// (an early write landing before the server subscribes is simply followed by
	// another).
	watchCtx, watchCancel := context.WithTimeout(rpcCtx, 30*time.Second)
	defer watchCancel()

	events := make(chan *previewv1.WatchEventsResponse, 8)
	errs := make(chan error, 1)
	go func() {
		stream, err := client.WatchEvents(watchCtx,
			connect.NewRequest(&previewv1.WatchEventsRequest{}))
		if err != nil {
			errs <- err
			return
		}
		defer func() { _ = stream.Close() }()
		for stream.Receive() {
			select {
			case events <- stream.Msg():
			case <-watchCtx.Done():
				return
			}
		}
		if err := stream.Err(); err != nil {
			select {
			case errs <- err:
			default:
			}
		}
	}()

	writeDoc := func() {
		writeFile(t, docPath,
			fmt.Sprintf("# Hello World\n\nupdated %d\n", time.Now().UnixNano()))
	}
	writeDoc()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case ev := <-events:
			if dc := ev.GetDocChanged(); dc != nil && dc.GetRootId() == rootID {
				if dc.GetPath() != "doc.md" {
					t.Errorf("DocChanged path = %q, want %q", dc.GetPath(), "doc.md")
				}
				return // success
			}
			// Ignore any other event kind (e.g. RootsChanged/TreeChanged).
		case err := <-errs:
			t.Fatalf("WatchEvents stream: %v", err)
		case <-ticker.C:
			writeDoc()
		case <-watchCtx.Done():
			t.Fatalf("timed out waiting for a DocChanged WatchEvents event: %v", watchCtx.Err())
		}
	}
}

// TestPreviewCmdmanPath exercises the full `crabswarm preview` flow through
// preview.EnsureDaemon, which shells out to cmdman to daemonize `preview
// __serve`. It is opt-in: it skips unless cmdman is on PATH. The daemon runs
// under a non-default name and an ephemeral port (both supplied through a temp
// --config file) so it can never collide with a real crabswarm-preview daemon,
// and Cleanup always stops and removes it.
func TestPreviewCmdmanPath(t *testing.T) {
	if _, err := exec.LookPath("cmdman"); err != nil {
		t.Skip("cmdman not on PATH; skipping cmdman-path preview e2e")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	addr := freeAddr(t)
	daemonName := fmt.Sprintf("crabswarm-preview-e2e-%d", os.Getpid())

	// Non-default daemon name + ephemeral port via a temp config file so the
	// test daemon is isolated from any real one.
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.json")
	writeFile(t, cfgPath,
		fmt.Sprintf(`{"preview":{"addr":%q,"daemon_name":%q}}`, addr, daemonName))

	// Always tear the daemon down, even if the test fails partway. Registered
	// before the daemon is started so it runs last.
	t.Cleanup(func() { cmdmanStopRemove(t, daemonName) })

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "doc.md"), "# Cmdman Path\n\nvia EnsureDaemon\n")

	// `crabswarm preview ROOT --config <cfg>`: EnsureDaemon (cmdman run) +
	// healthz poll + AddRoot, then prints the URL.
	out := runCrabswarm(ctx, t, "preview", root, "--config", cfgPath)
	if !strings.Contains(out, "/roots/") {
		t.Errorf("preview did not print a root URL; got:\n%s", out)
	}

	// `crabswarm preview list --config <cfg>` reaches the same daemon and lists
	// the root added above, confirming the daemon is up through EnsureDaemon.
	list := runCrabswarm(ctx, t, "preview", "list", "--config", cfgPath)
	if !strings.Contains(list, filepath.Base(root)) {
		t.Errorf("preview list missing the added root %q; got:\n%s", filepath.Base(root), list)
	}
}

// fakeBdScript is a stand-in for the real bd binary. `bd where` reports the
// .beads directory of the directory it runs in, so a temp directory holding one
// registers as an issue source and a directory without one does not; `bd list`
// answers with an empty backlog, which is what the daemon's poller reads once a
// source is registered. Both outputs follow the shapes the issues client
// decodes: the JSON envelope for where (error envelope on stdout with a
// non-zero exit), a bare array for list.
const fakeBdScript = `#!/bin/sh
case "$1" in
where)
  if [ -d "$PWD/.beads" ]; then
    printf '{"data":{"path":"%s/.beads","prefix":"e2e","database_path":"%s/.beads/db"}}\n' \
      "$PWD" "$PWD"
    exit 0
  fi
  printf '{"data":{"error":"no_beads_directory","message":"No active beads workspace found."}}\n'
  exit 1
  ;;
list)
  echo '[]'
  ;;
*)
  echo "fake bd: unknown subcommand $1" >&2
  exit 2
  ;;
esac
exit 0
`

// fakeCmdmanScript is a stand-in for cmdman that always reports the daemon as
// running, so `crabswarm preview` adopts the `preview __serve` process the test
// started itself instead of daemonizing one.
const fakeCmdmanScript = `#!/bin/sh
case "$1" in
inspect) echo running ;;
esac
exit 0
`

// fakeBinEnv writes the named scripts into a fresh directory and returns an
// environment whose PATH is that directory alone, for the daemon process and
// the CLI invocations alike. PATH is replaced rather than extended so a real bd
// or cmdman installed on the developer's machine cannot answer instead — which
// is the whole point of the case that runs without bd.
func fakeBinEnv(t *testing.T, scripts map[string]string) []string {
	t.Helper()
	dir := t.TempDir()
	for name, script := range scripts {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatalf("write fake %s: %v", name, err)
		}
	}
	return append(os.Environ(), "PATH="+dir)
}

// startPreviewServe runs the hidden daemon entry point directly under env — the
// same process cmdman would otherwise daemonize — and returns once it answers
// /healthz.
func startPreviewServe(ctx context.Context, t *testing.T, env []string, addr, cfgPath string) {
	t.Helper()
	serve := exec.Command(crabswarmBin, "preview", "__serve", "--addr", addr, "--config", cfgPath)
	serve.Env = env
	serve.Stdout = os.Stderr
	serve.Stderr = os.Stderr
	if err := serve.Start(); err != nil {
		t.Fatalf("start preview __serve: %v", err)
	}
	t.Cleanup(func() { stopProcess(t, serve) })

	waitHealthz(ctx, t, addr, 30*time.Second)
}

// TestPreviewIssueSources covers the registration side of `crabswarm preview`:
// a directory governed by a beads database becomes both a file root and an
// issue source, one without becomes a root alone, --issue insists on the
// database, and `preview remove` reaches either kind. The daemon is the same
// directly started `preview __serve` process TestPreviewServe uses; the fake
// cmdman only satisfies EnsureDaemon's liveness check.
func TestPreviewIssueSources(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	env := fakeBinEnv(t, map[string]string{
		"bd":     fakeBdScript,
		"cmdman": fakeCmdmanScript,
	})
	addr := freeAddr(t)

	cfgPath := filepath.Join(t.TempDir(), "config.json")
	writeFile(t, cfgPath, "{}")

	withBeads := t.TempDir()
	if err := os.Mkdir(filepath.Join(withBeads, ".beads"), 0o755); err != nil {
		t.Fatalf("create .beads: %v", err)
	}
	plain := t.TempDir()

	startPreviewServe(ctx, t, env, addr, cfgPath)

	preview := func(args ...string) string {
		t.Helper()
		return mustRunCrabswarm(ctx, t, env,
			append([]string{"preview"}, append(args, "--addr", addr, "--config", cfgPath)...)...)
	}

	// A directory with a beads database registers on both sides.
	out := preview(withBeads)
	if !strings.Contains(out, "/roots/") {
		t.Errorf("preview did not print a root URL; got:\n%s", out)
	}
	if !strings.Contains(out, "issue source e2e") {
		t.Errorf("preview did not report the issue source; got:\n%s", out)
	}

	counts := countKinds(t, preview("list"))
	if counts["root"] != 1 || counts["source"] != 1 {
		t.Errorf("after registering %s: got %v, want 1 root and 1 source", withBeads, counts)
	}

	// A directory outside every beads workspace registers as a root only. A
	// repository that keeps no issues is the ordinary case, so it is neither an
	// error nor worth a warning.
	stdout, stderr, err := runCrabswarmEnv(ctx, t, env,
		"preview", plain, "--addr", addr, "--config", cfgPath)
	if err != nil {
		t.Fatalf("preview on a directory without a beads database failed: %v\n"+
			"stdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "/roots/") {
		t.Errorf("preview did not print a root URL; got:\n%s", stdout)
	}
	if strings.Contains(stdout, "issue source") {
		t.Errorf("preview registered an issue source for a directory without a beads database;"+
			" got:\n%s", stdout)
	}
	if strings.Contains(stderr, "no issues source registered") {
		t.Errorf("preview warned about a directory that simply keeps no issues; got:\n%s", stderr)
	}

	counts = countKinds(t, preview("list"))
	if counts["root"] != 2 || counts["source"] != 1 {
		t.Errorf("after registering %s: got %v, want 2 roots and 1 source", plain, counts)
	}

	// --issue asked for the source explicitly, so the missing database is an
	// error rather than a silently skipped registration.
	stdout, stderr, err = runCrabswarmEnv(ctx, t, env,
		"preview", "--issue", plain, "--addr", addr, "--config", cfgPath)
	if err == nil {
		t.Errorf("preview --issue on a directory without a beads database succeeded; got:\n%s",
			stdout)
	}
	if !strings.Contains(stderr, "beads") {
		t.Errorf("preview --issue error does not mention beads; got:\n%s", stderr)
	}

	// remove takes the source's prefix, the name `preview list` prints for it.
	if got := preview("remove", "e2e"); !strings.Contains(got, "source") {
		t.Errorf("remove did not report removing a source; got:\n%s", got)
	}
	counts = countKinds(t, preview("list"))
	if counts["root"] != 2 || counts["source"] != 0 {
		t.Errorf("after removing the source: got %v, want 2 roots and no source", counts)
	}
}

// TestPreviewWithoutBd covers the daemon failing to resolve a beads database
// for a reason other than the directory having none: bd is not installed at
// all. Unlike a directory that simply keeps no issues, this is unexpected, so
// the default registration keeps working and says so — the root is registered,
// its URL printed, and the failure reported as one warning on stderr — while
// --issue, whose whole request is the source, fails.
func TestPreviewWithoutBd(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	env := fakeBinEnv(t, map[string]string{"cmdman": fakeCmdmanScript})
	addr := freeAddr(t)

	cfgPath := filepath.Join(t.TempDir(), "config.json")
	writeFile(t, cfgPath, "{}")
	dir := t.TempDir()

	startPreviewServe(ctx, t, env, addr, cfgPath)

	stdout, stderr, err := runCrabswarmEnv(ctx, t, env,
		"preview", dir, "--addr", addr, "--config", cfgPath)
	if err != nil {
		t.Fatalf("preview failed although only the issues source could not be registered: %v\n"+
			"stdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "/roots/") {
		t.Errorf("preview did not print a root URL; got:\n%s", stdout)
	}
	if !strings.Contains(stderr, "no issues source registered") {
		t.Errorf("preview did not warn about the unregistered issues source; got:\n%s", stderr)
	}

	_, stderr, err = runCrabswarmEnv(ctx, t, env,
		"preview", "--issue", dir, "--addr", addr, "--config", cfgPath)
	if err == nil {
		t.Errorf("preview --issue succeeded without bd; stderr:\n%s", stderr)
	}
}

// countKinds tallies the KIND column of `preview list` output by row. The
// always-printed header is cut off first, so an empty registry counts nothing.
func countKinds(t *testing.T, out string) map[string]int {
	t.Helper()
	_, rows, _ := strings.Cut(strings.TrimSpace(out), "\n")
	counts := map[string]int{}
	for line := range strings.SplitSeq(rows, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		counts[fields[0]]++
	}
	return counts
}

// runCrabswarm runs the built crabswarm binary with args, returning trimmed
// stdout and failing the test if it exits non-zero.
func runCrabswarm(ctx context.Context, t *testing.T, args ...string) string {
	t.Helper()
	return mustRunCrabswarm(ctx, t, nil, args...)
}

// mustRunCrabswarm runs the binary under env and fails the test when it exits
// non-zero, returning its trimmed stdout. A nil env inherits the test's own.
func mustRunCrabswarm(ctx context.Context, t *testing.T, env []string, args ...string) string {
	t.Helper()
	stdout, stderr, err := runCrabswarmEnv(ctx, t, env, args...)
	if err != nil {
		t.Fatalf("crabswarm %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout, stderr)
	}
	return stdout
}

// runCrabswarmEnv runs the binary under env and returns its trimmed stdout,
// trimmed stderr and run error. Unlike mustRunCrabswarm it tolerates a non-zero
// exit, which the cases asserting on a rejected invocation need.
func runCrabswarmEnv(
	ctx context.Context,
	t *testing.T,
	env []string,
	args ...string,
) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.CommandContext(ctx, crabswarmBin, args...)
	cmd.Env = env
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()), err
}

// cmdmanStopRemove stops and removes the named cmdman daemon, best-effort. It
// uses a fresh short-lived context so cleanup still runs after the test's own
// context is cancelled.
func cmdmanStopRemove(t *testing.T, name string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "cmdman", "stop", name).CombinedOutput(); err != nil {
		t.Logf("cmdman stop %s: %v: %s", name, err, strings.TrimSpace(string(out)))
	}
	if out, err := exec.CommandContext(ctx, "cmdman", "rm", name).CombinedOutput(); err != nil {
		t.Logf("cmdman rm %s: %v: %s", name, err, strings.TrimSpace(string(out)))
	}
}

// stopProcess terminates a serve subprocess: it sends SIGTERM (the signal the
// process's cmdsignals context handles for a graceful HTTP shutdown) and waits,
// escalating to Kill if it does not exit promptly.
func stopProcess(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		<-done
	}
}
