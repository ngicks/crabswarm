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

	crabpreviewv1 "github.com/ngicks/crabswarm/api/gen/proto/go/crabpreview/v1"
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
		connect.NewRequest(&crabpreviewv1.AddRootRequest{Path: root}))
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
		connect.NewRequest(&crabpreviewv1.GetDocumentRequest{RootId: rootID, Path: "doc.md"}))
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

	events := make(chan *crabpreviewv1.WatchEventsResponse, 8)
	errs := make(chan error, 1)
	go func() {
		stream, err := client.WatchEvents(watchCtx,
			connect.NewRequest(&crabpreviewv1.WatchEventsRequest{}))
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
	if !strings.Contains(out, "/r/") {
		t.Errorf("preview did not print a root URL; got:\n%s", out)
	}

	// `crabswarm preview list --config <cfg>` reaches the same daemon and lists
	// the root added above, confirming the daemon is up through EnsureDaemon.
	list := runCrabswarm(ctx, t, "preview", "list", "--config", cfgPath)
	if !strings.Contains(list, filepath.Base(root)) {
		t.Errorf("preview list missing the added root %q; got:\n%s", filepath.Base(root), list)
	}
}

// runCrabswarm runs the built crabswarm binary with args, returning trimmed
// stdout and failing the test if it exits non-zero.
func runCrabswarm(ctx context.Context, t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, crabswarmBin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("crabswarm %s failed: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return strings.TrimSpace(stdout.String())
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
