package preview

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/crabswarm/preview/httpapi"
)

// runningService starts a Service on an ephemeral loopback port with watcher
// supervision active and returns it; Serve is stopped on cleanup. The service's
// internal add/remove/resolve helpers are exercised directly (white-box) so the
// watcher lifecycle can be observed without going through the HTTP layer.
func runningService(t *testing.T) *Service {
	t.Helper()
	svc, err := New(nil, Config{Addr: "127.0.0.1:0", DaemonName: "test"})
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

	// Supervision (s.runCtx) is set at the start of Serve, before it listens.
	deadline := time.Now().Add(5 * time.Second)
	for {
		svc.mu.Lock()
		ready := svc.runCtx != nil
		svc.mu.Unlock()
		if ready {
			return svc
		}
		if time.Now().After(deadline) {
			t.Fatal("service supervision did not start")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// hasWatcher reports whether a live watcher is registered for id.
func hasWatcher(s *Service, id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.watchers[id]
	return ok
}

func TestService_RemoveRootTearsDownWatcher(t *testing.T) {
	svc := runningService(t)
	dir := t.TempDir()

	root, err := svc.addRoot(t.Context(), dir)
	assert.NilError(t, err)
	assert.Assert(t, hasWatcher(svc, root.ID))

	removed, ok := svc.removeRoot(root.ID)
	assert.Assert(t, ok)
	assert.Equal(t, removed.ID, root.ID)
	// The watcher is torn down and the store no longer knows the root.
	assert.Assert(t, !hasWatcher(svc, root.ID))
	_, present := svc.store.Get(root.ID)
	assert.Assert(t, !present)
	// Removing an already-removed root reports "not found".
	_, ok = svc.removeRoot(root.ID)
	assert.Assert(t, !ok)
}

func TestService_RemoveThenReAddKeepsLiveWatcher(t *testing.T) {
	svc := runningService(t)
	dir := t.TempDir()

	root, err := svc.addRoot(t.Context(), dir)
	assert.NilError(t, err)
	_, ok := svc.removeRoot(root.ID)
	assert.Assert(t, ok)

	// Re-adding the same path yields the same deterministic ID and a fresh, live
	// watcher.
	root2, err := svc.addRoot(t.Context(), dir)
	assert.NilError(t, err)
	assert.Equal(t, root2.ID, root.ID)
	assert.Assert(t, hasWatcher(svc, root2.ID))

	// Prove the watcher is live: a write is observed as a DocChanged on the hub.
	ch, unsub := svc.hub.Subscribe()
	defer unsub()
	doc := filepath.Join(dir, "live.md")
	got := triggerUntil(
		t,
		ch,
		func() { _ = os.WriteFile(doc, []byte("x"), 0o644) },
		func(ev Event) bool { return ev.Kind == DocChanged && strings.Contains(ev.Path, "live.md") },
	)
	assert.Equal(t, got.RootID, root2.ID)
}

func TestService_ConcurrentAddRemoveSamePathKeepsInvariant(t *testing.T) {
	svc := runningService(t)
	dir := t.TempDir()

	// Learn the deterministic ID for this path.
	root, err := svc.addRoot(t.Context(), dir)
	assert.NilError(t, err)
	id := root.ID

	// Storm the same path with concurrent adds and removes. Because the store
	// mutation and the watcher lifecycle share one critical section, no
	// interleaving can register the root without a watcher. Run under -race to
	// also validate the locking.
	var g errgroup.Group
	for range 30 {
		g.Go(func() error { _, _ = svc.addRoot(t.Context(), dir); return nil })
		g.Go(func() error { svc.removeRoot(id); return nil })
	}
	_ = g.Wait()

	// A final add with no concurrent remove must leave the root registered with a
	// live watcher — the invariant the TOCTOU race would break.
	_, err = svc.addRoot(t.Context(), dir)
	assert.NilError(t, err)
	_, present := svc.store.Get(id)
	assert.Assert(t, present)
	assert.Assert(t, hasWatcher(svc, id))
}

func TestService_ResolveRawRejectsTraversal(t *testing.T) {
	svc := runningService(t)
	dir := t.TempDir()
	assert.NilError(t, os.WriteFile(filepath.Join(dir, "in.txt"), []byte("inside"), 0o644))

	// A symlink inside the root pointing at a directory outside it.
	outside := t.TempDir()
	assert.NilError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("SECRET"), 0o644))
	assert.NilError(t, os.Symlink(outside, filepath.Join(dir, "escape")))

	root, err := svc.addRoot(t.Context(), dir)
	assert.NilError(t, err)

	// A valid in-root path resolves to an absolute path.
	got, err := svc.ResolveRaw(root.ID, "in.txt")
	assert.NilError(t, err)
	assert.Assert(t, strings.HasSuffix(got, "in.txt"))

	// Every traversal variant is rejected as a path escape.
	for name, rel := range map[string]string{
		"parent":   filepath.Join("..", "secret"),
		"absolute": string(filepath.Separator) + filepath.Join("etc", "passwd"),
		"symlink":  filepath.Join("escape", "secret.txt"),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := svc.ResolveRaw(root.ID, rel)
			assert.Assert(
				t,
				errors.Is(err, httpapi.ErrPathEscape),
				"want ErrPathEscape, got %v",
				err,
			)
		})
	}

	// An unknown root is a not-found.
	_, err = svc.ResolveRaw("does-not-exist", "in.txt")
	assert.Assert(t, errors.Is(err, httpapi.ErrRootNotFound))
}
