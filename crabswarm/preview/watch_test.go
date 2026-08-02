package preview

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"
)

// startWatcher runs w under an errgroup and stops it on test cleanup, mirroring
// how the Service drives watchers. A clean shutdown must return nil.
func startWatcher(t *testing.T, w *Watcher) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return w.Run(ctx) })
	t.Cleanup(func() {
		cancel()
		if err := g.Wait(); err != nil {
			t.Errorf("watcher Run returned error: %v", err)
		}
	})
}

// triggerUntil runs action, then waits up to a debounce window for an event
// matching pred; if none arrives it re-runs action, until pred matches or an
// overall deadline elapses. It absorbs the startup race where the first action
// happens before the root watch is live: the action is simply retried.
func triggerUntil(t *testing.T, ch <-chan Event, action func(), pred func(Event) bool) Event {
	t.Helper()
	overall := time.After(3 * time.Second)
retryLoop:
	for {
		action()
		window := time.After(250 * time.Millisecond)
		for {
			select {
			case ev, ok := <-ch:
				if !ok {
					t.Fatal("watch event channel closed")
				}
				if pred(ev) {
					return ev
				}
			case <-window:
				continue retryLoop
			case <-overall:
				t.Fatal("timed out waiting for watch event")
			}
		}
	}
}

// waitEvent waits for an event matching pred without re-triggering; use it once
// the watch is known to be live so a single filesystem change is reliably seen.
func waitEvent(t *testing.T, ch <-chan Event, timeout time.Duration, pred func(Event) bool) Event {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				t.Fatal("watch event channel closed")
			}
			if pred(ev) {
				return ev
			}
		case <-deadline:
			t.Fatal("timed out waiting for watch event")
		}
	}
}

// awaitReady blocks until the root watch is confirmed live by probing on a
// throwaway subscription, so the test's own subscription stays free of probe
// noise. It leaves the probe file in place (removing it would emit a late tree
// event); t.TempDir cleans it up.
func awaitReady(t *testing.T, hub *Hub, root string) {
	t.Helper()
	ch, unsub := hub.Subscribe()
	defer unsub()
	probe := filepath.Join(root, ".watch-probe")
	triggerUntil(t, ch,
		func() { _ = os.WriteFile(probe, []byte("probe"), 0o644) },
		func(ev Event) bool { return strings.Contains(ev.Path, ".watch-probe") })
}

func TestWatch_DocEventOnWrite(t *testing.T) {
	root := t.TempDir()
	doc := filepath.Join(root, "doc.md")
	writeFile(t, doc) // pre-exists before watching -> only a later write fires

	hub := NewHub()
	w, err := NewWatcher(nil, "r1", root, nil, hub)
	assert.NilError(t, err)
	startWatcher(t, w)
	awaitReady(t, hub, root)

	ch, unsub := hub.Subscribe()
	defer unsub()

	// Rapid successive writes must debounce into a single document event.
	assert.NilError(t, os.WriteFile(doc, []byte("a"), 0o644))
	assert.NilError(t, os.WriteFile(doc, []byte("bb"), 0o644))
	assert.NilError(t, os.WriteFile(doc, []byte("ccc"), 0o644))

	got := waitEvent(t, ch, 3*time.Second, func(ev Event) bool {
		return ev.Kind == DocChanged && ev.Path == "doc.md"
	})
	assert.Equal(t, got.RootID, "r1")
}

func TestWatch_TreeEventInNewSubdir(t *testing.T) {
	root := t.TempDir()

	hub := NewHub()
	w, err := NewWatcher(nil, "r", root, nil, hub)
	assert.NilError(t, err)
	startWatcher(t, w)
	awaitReady(t, hub, root)

	ch, unsub := hub.Subscribe()
	defer unsub()

	// Creating a directory must register a watch for it dynamically; the root
	// tree event only fires once that watch has been added inside classify.
	sub := filepath.Join(root, "newsub")
	assert.NilError(t, os.Mkdir(sub, 0o755))
	waitEvent(t, ch, 3*time.Second, func(ev Event) bool {
		return ev.Kind == TreeChanged && ev.Path == "."
	})

	// A file created inside the just-created subdir proves the dynamic watch
	// is live: without it this event would never be delivered.
	assert.NilError(t, os.WriteFile(filepath.Join(sub, "note.md"), []byte("x"), 0o644))
	got := waitEvent(t, ch, 3*time.Second, func(ev Event) bool {
		return strings.Contains(ev.Path, "newsub")
	})
	assert.Assert(t, got.Kind == TreeChanged || got.Kind == DocChanged)
	assert.Equal(t, got.RootID, "r")
}

func TestWatch_TreeEventOnRemove(t *testing.T) {
	root := t.TempDir()
	doc := filepath.Join(root, "gone.md")
	writeFile(t, doc) // pre-exists before watching -> no create event to confuse

	hub := NewHub()
	w, err := NewWatcher(nil, "r", root, nil, hub)
	assert.NilError(t, err)
	startWatcher(t, w)
	awaitReady(t, hub, root)

	ch, unsub := hub.Subscribe()
	defer unsub()

	// The watch is live, so a single removal is reliably observed as a tree
	// change on the containing directory (root -> ".").
	assert.NilError(t, os.Remove(doc))
	got := waitEvent(t, ch, 3*time.Second, func(ev Event) bool {
		return ev.Kind == TreeChanged && ev.Path == "."
	})
	assert.Equal(t, got.RootID, "r")
}

func TestWatch_IgnoresDefaultAndConfiguredDirs(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, ".git"))
	mkdir(t, filepath.Join(root, "node_modules"))
	mkdir(t, filepath.Join(root, "vendor")) // custom pattern

	hub := NewHub()
	w, err := NewWatcher(nil, "r", root, []string{"vendor"}, hub)
	assert.NilError(t, err)
	startWatcher(t, w)
	awaitReady(t, hub, root)

	ch, unsub := hub.Subscribe()
	defer unsub()

	// Writes inside ignored directories (never watched) must produce nothing.
	for _, d := range []string{".git", "node_modules", "vendor"} {
		assert.NilError(t, os.WriteFile(filepath.Join(root, d, "x.md"), []byte("x"), 0o644))
	}
	// A normal file at the root gives us a marker to synchronize on: by the
	// time its event arrives, any (buggy) ignored-dir event would already be
	// batched alongside it.
	marker := filepath.Join(root, "marker.md")
	markerEv := triggerUntil(t, ch,
		func() { _ = os.WriteFile(marker, []byte("x"), 0o644) },
		func(ev Event) bool { return strings.Contains(ev.Path, "marker.md") })
	assert.Equal(t, markerEv.RootID, "r")

	// Drain everything queued; none of it may reference an ignored directory.
	for {
		select {
		case ev := <-ch:
			for _, bad := range []string{".git", "node_modules", "vendor"} {
				assert.Assert(t, !strings.Contains(ev.Path, bad),
					"unexpected event from ignored dir %q: %+v", bad, ev)
			}
		default:
			return
		}
	}
}

func TestWatch_RejectsBadPatternAndMissingRoot(t *testing.T) {
	hub := NewHub()

	// A malformed glob is rejected at construction.
	_, err := NewWatcher(nil, "r", t.TempDir(), []string{"["}, hub)
	assert.Assert(t, err != nil, "malformed ignore pattern should fail")

	// A non-directory root is rejected too.
	file := filepath.Join(t.TempDir(), "file.md")
	assert.NilError(t, os.WriteFile(file, []byte("x"), 0o644))
	_, err = NewWatcher(nil, "r", file, nil, hub)
	assert.Assert(t, err != nil, "non-directory root should fail")
}

func TestWatchHub_BroadcastAndUnsubscribe(t *testing.T) {
	hub := NewHub()
	a, unsubA := hub.Subscribe()
	b, unsubB := hub.Subscribe()
	defer unsubB()

	ev := Event{Kind: DocChanged, RootID: "r", Path: "x.md"}
	hub.Publish(ev)
	assert.Equal(t, <-a, ev)
	assert.Equal(t, <-b, ev)

	// After unsubscribe, the channel closes and no longer receives.
	unsubA()
	_, ok := <-a
	assert.Assert(t, !ok, "unsubscribed channel should be closed")

	hub.Publish(Event{Kind: TreeChanged, RootID: "r", Path: "."})
	got := <-b
	assert.Equal(t, got.Kind, TreeChanged)

	// Unsubscribe is idempotent.
	unsubA()
}

func TestWatchHub_DropsForSlowSubscriber(t *testing.T) {
	hub := NewHub()
	ch, unsub := hub.Subscribe()
	defer unsub()

	// Publishing far past the buffer must not block a slow (unread) subscriber.
	for range subBuffer * 4 {
		hub.Publish(Event{Kind: DocChanged, RootID: "r", Path: "x.md"})
	}
	assert.Equal(t, len(ch), subBuffer)
}
