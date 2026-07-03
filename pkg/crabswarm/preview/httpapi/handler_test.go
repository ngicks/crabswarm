package httpapi_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"gotest.tools/v3/assert"

	crabpreviewv1 "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/crabpreview/v1"
	"github.com/ngicks/crabswarm/pkg/api/gen/proto/go/crabpreview/v1/crabpreviewv1connect"
	"github.com/ngicks/crabswarm/pkg/crabswarm/preview"
	"github.com/ngicks/crabswarm/pkg/crabswarm/preview/httpapi"
)

// startService runs a real preview.Service on an ephemeral loopback port and
// returns a connect client plus the http base URL. The composed handler is
// exercised through a real HTTP server (rather than httptest.NewServer) because
// the streaming test needs the service's live watcher supervision, which
// Serve drives; readiness uses the same /healthz contract EnsureDaemon polls.
func startService(t *testing.T) (crabpreviewv1connect.PreviewServiceClient, string) {
	t.Helper()
	addr := freeAddr(t)
	svc, err := preview.New(nil, preview.Config{Addr: addr, DaemonName: "test"})
	assert.NilError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- svc.Serve(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-errCh:
			assert.NilError(t, err)
		case <-time.After(5 * time.Second):
			t.Error("Serve did not return after context cancel")
		}
	})

	base := "http://" + addr
	waitHealthy(t, base)
	return preview.NewClient(addr), base
}

// freeAddr reserves an ephemeral loopback port and releases it for Serve to
// rebind. The brief reuse window is acceptable for a test.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NilError(t, err)
	defer func() { _ = l.Close() }()
	return l.Addr().String()
}

// waitHealthy polls /healthz until it answers 200 or a deadline elapses.
func waitHealthy(t *testing.T, base string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/healthz")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("server did not become healthy in time")
}

func httpGet(t *testing.T, url string) (status int, body string, header http.Header) {
	t.Helper()
	resp, err := http.Get(url)
	assert.NilError(t, err)
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	assert.NilError(t, err)
	return resp.StatusCode, string(b), resp.Header
}

func TestHealthz(t *testing.T) {
	_, base := startService(t)
	status, body, header := httpGet(t, base+"/healthz")
	assert.Equal(t, status, http.StatusOK)
	assert.Assert(t, strings.Contains(header.Get("Content-Type"), "application/json"))
	assert.Assert(t, strings.Contains(body, "ok"))
}

func TestAddRootTreeAndDocument(t *testing.T) {
	client, _ := startService(t)
	ctx := context.Background()

	dir := t.TempDir()
	assert.NilError(t, os.WriteFile(
		filepath.Join(dir, "readme.md"),
		[]byte("# Title\n\nhello world\n\n## Section\n"), 0o644))
	assert.NilError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))

	addResp, err := client.AddRoot(
		ctx,
		connect.NewRequest(&crabpreviewv1.AddRootRequest{Path: dir}),
	)
	assert.NilError(t, err)
	rootID := addResp.Msg.GetRoot().GetId()
	assert.Assert(t, rootID != "")

	// Re-adding the same path is idempotent: same id.
	addResp2, err := client.AddRoot(
		ctx,
		connect.NewRequest(&crabpreviewv1.AddRootRequest{Path: dir}),
	)
	assert.NilError(t, err)
	assert.Equal(t, addResp2.Msg.GetRoot().GetId(), rootID)

	// GetTree: dirs first, then files, with markdown flagged.
	treeResp, err := client.GetTree(
		ctx,
		connect.NewRequest(&crabpreviewv1.GetTreeRequest{RootId: rootID, Path: "."}),
	)
	assert.NilError(t, err)
	entries := treeResp.Msg.GetEntries()
	assert.Equal(t, len(entries), 2)
	assert.Equal(t, entries[0].GetName(), "sub")
	assert.Equal(t, entries[0].GetType(), crabpreviewv1.EntryType_ENTRY_TYPE_DIR)
	assert.Equal(t, entries[1].GetName(), "readme.md")
	assert.Equal(t, entries[1].GetType(), crabpreviewv1.EntryType_ENTRY_TYPE_FILE)
	assert.Equal(t, entries[1].GetIsMarkdown(), true)

	// GetDocument: rendered HTML, title from first h1, TOC, and mtime.
	docResp, err := client.GetDocument(
		ctx,
		connect.NewRequest(&crabpreviewv1.GetDocumentRequest{RootId: rootID, Path: "readme.md"}),
	)
	assert.NilError(t, err)
	assert.Equal(t, docResp.Msg.GetTitle(), "Title")
	assert.Assert(t, strings.Contains(docResp.Msg.GetHtml(), "hello world"))
	assert.Assert(t, len(docResp.Msg.GetToc()) >= 2)
	assert.Assert(t, docResp.Msg.GetMtime() != nil)
}

func TestGetDocumentTitleFallsBackToFileName(t *testing.T) {
	client, _ := startService(t)
	ctx := context.Background()

	dir := t.TempDir()
	// No h1 -> title falls back to the file name.
	assert.NilError(
		t,
		os.WriteFile(filepath.Join(dir, "no-title.md"), []byte("plain body\n"), 0o644),
	)
	addResp, err := client.AddRoot(
		ctx,
		connect.NewRequest(&crabpreviewv1.AddRootRequest{Path: dir}),
	)
	assert.NilError(t, err)

	docResp, err := client.GetDocument(ctx, connect.NewRequest(&crabpreviewv1.GetDocumentRequest{
		RootId: addResp.Msg.GetRoot().GetId(),
		Path:   "no-title.md",
	}))
	assert.NilError(t, err)
	assert.Equal(t, docResp.Msg.GetTitle(), "no-title.md")
}

func TestGetDocumentRejectsTraversalAndMissingRoot(t *testing.T) {
	client, _ := startService(t)
	ctx := context.Background()

	dir := t.TempDir()
	assert.NilError(t, os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# Hi\n"), 0o644))
	// A symlink inside the root pointing at a file outside it.
	outside := t.TempDir()
	assert.NilError(
		t,
		os.WriteFile(filepath.Join(outside, "secret.md"), []byte("# SECRET\n"), 0o644),
	)
	assert.NilError(t, os.Symlink(outside, filepath.Join(dir, "escape")))

	addResp, err := client.AddRoot(
		ctx,
		connect.NewRequest(&crabpreviewv1.AddRootRequest{Path: dir}),
	)
	assert.NilError(t, err)
	rootID := addResp.Msg.GetRoot().GetId()

	// Every traversal variant is rejected as an invalid argument.
	for name, rel := range map[string]string{
		"parent":   filepath.Join("..", "secret"),
		"absolute": string(filepath.Separator) + filepath.Join("etc", "passwd"),
		"symlink":  filepath.Join("escape", "secret.md"),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := client.GetDocument(ctx, connect.NewRequest(&crabpreviewv1.GetDocumentRequest{
				RootId: rootID,
				Path:   rel,
			}))
			assert.Assert(t, err != nil)
			assert.Equal(t, connect.CodeOf(err), connect.CodeInvalidArgument)
		})
	}

	// Unknown root is not-found.
	_, err = client.GetDocument(ctx, connect.NewRequest(&crabpreviewv1.GetDocumentRequest{
		RootId: "does-not-exist",
		Path:   "readme.md",
	}))
	assert.Assert(t, err != nil)
	assert.Equal(t, connect.CodeOf(err), connect.CodeNotFound)
}

func TestRawServesFileAndRejectsTraversal(t *testing.T) {
	client, base := startService(t)
	ctx := context.Background()

	dir := t.TempDir()
	assert.NilError(
		t,
		os.WriteFile(filepath.Join(dir, "note.txt"), []byte("raw-bytes-content"), 0o644),
	)

	// A symlink inside the root pointing outside it: reaching through it must be
	// rejected by the SafeJoin gate even though the URL contains no "..".
	outside := t.TempDir()
	assert.NilError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("SECRET"), 0o644))
	assert.NilError(t, os.Symlink(outside, filepath.Join(dir, "escape")))

	addResp, err := client.AddRoot(
		ctx,
		connect.NewRequest(&crabpreviewv1.AddRootRequest{Path: dir}),
	)
	assert.NilError(t, err)
	rootID := addResp.Msg.GetRoot().GetId()

	// Happy path: raw bytes with a detected content type.
	status, body, _ := httpGet(t, base+"/raw/"+rootID+"/note.txt")
	assert.Equal(t, status, http.StatusOK)
	assert.Equal(t, body, "raw-bytes-content")

	// Symlink escape -> rejected by the SafeJoin gate, secret not leaked.
	status, body, _ = httpGet(t, base+"/raw/"+rootID+"/escape/secret.txt")
	assert.Equal(t, status, http.StatusBadRequest)
	assert.Assert(t, !strings.Contains(body, "SECRET"))

	// "../" and absolute-path attempts: normalized away by the URL layer and/or
	// rejected by SafeJoin — either way the outside secret is never served. The
	// "../" target is a genuinely reachable-on-disk path (dir and outside are
	// sibling temp dirs). (The SafeJoin escape codes themselves are asserted
	// directly in the preview package's ResolveRaw test.)
	for _, url := range []string{
		base + "/raw/" + rootID + "/../" + filepath.Base(outside) + "/secret.txt",
		base + "/raw/" + rootID + "//etc/passwd",
	} {
		_, body, _ := httpGet(t, url)
		assert.Assert(t, !strings.Contains(body, "SECRET"))
	}

	// Missing file -> 404.
	status, _, _ = httpGet(t, base+"/raw/"+rootID+"/missing.txt")
	assert.Equal(t, status, http.StatusNotFound)

	// Unknown root -> 404.
	status, _, _ = httpGet(t, base+"/raw/unknown-root/note.txt")
	assert.Equal(t, status, http.StatusNotFound)
}

func TestStaticPlaceholderAndCSSAssets(t *testing.T) {
	_, base := startService(t)

	// No SPA FS injected -> built-in placeholder index.html, and SPA fallback
	// serves it for unknown routes too.
	for _, path := range []string{"/", "/r/some/deep/route"} {
		status, body, header := httpGet(t, base+path)
		assert.Equal(t, status, http.StatusOK)
		assert.Assert(t, strings.Contains(header.Get("Content-Type"), "text/html"))
		assert.Assert(t, strings.Contains(body, "crabswarm preview"))
	}

	// Render stylesheets are served at their documented, stable paths.
	for _, path := range []string{
		httpapi.ChromaLightCSSPath,
		httpapi.ChromaDarkCSSPath,
		httpapi.AlertCSSPath,
	} {
		status, body, header := httpGet(t, base+path)
		assert.Equal(t, status, http.StatusOK)
		assert.Assert(t, strings.Contains(header.Get("Content-Type"), "text/css"))
		assert.Assert(t, len(body) > 0)
	}
}

func TestWatchEventsStreamsFileChange(t *testing.T) {
	client, _ := startService(t)
	ctx := context.Background()

	dir := t.TempDir()
	addResp, err := client.AddRoot(
		ctx,
		connect.NewRequest(&crabpreviewv1.AddRootRequest{Path: dir}),
	)
	assert.NilError(t, err)
	rootID := addResp.Msg.GetRoot().GetId()

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// The connect client's WatchEvents call blocks until the server flushes its
	// first response, which happens on the handler's first Send. So keep writing
	// the file from a background goroutine: once the handler has subscribed, a
	// write produces a DocChanged the handler forwards, which both unblocks the
	// stream open and delivers the event. Repeated writes absorb the watcher's
	// initial-walk and subscribe races (and the ~100ms debounce).
	doc := filepath.Join(dir, "watch.md")
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		tick := time.NewTicker(150 * time.Millisecond)
		defer tick.Stop()
		for {
			_ = os.WriteFile(doc, []byte("# a "+time.Now().String()), 0o644)
			select {
			case <-streamCtx.Done():
				return
			case <-tick.C:
			}
		}
	}()

	stream, err := client.WatchEvents(
		streamCtx,
		connect.NewRequest(&crabpreviewv1.WatchEventsRequest{}),
	)
	assert.NilError(t, err)
	defer func() { _ = stream.Close() }()

	found := make(chan *crabpreviewv1.DocChanged, 1)
	go func() {
		for stream.Receive() {
			if dc := stream.Msg().
				GetDocChanged(); dc != nil &&
				strings.Contains(dc.GetPath(), "watch.md") {
				select {
				case found <- dc:
				default:
				}
				return
			}
		}
	}()

	select {
	case dc := <-found:
		assert.Equal(t, dc.GetRootId(), rootID)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for DocChanged event")
	}

	// Stop the writer before the temp dir is torn down.
	cancel()
	<-writerDone
}
