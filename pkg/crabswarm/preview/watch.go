package preview

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// defaultIgnorePatterns are the directory base names never watched: version
// control metadata and dependency caches that would otherwise blow past the
// inotify watch limit for no benefit. They are prepended to the caller's
// configured patterns, matched the same way (see [matchesAny]).
var defaultIgnorePatterns = []string{".git", "node_modules"}

// debounceInterval is the quiet period a path must observe before its
// coalesced [Event] is published. Rapid successive writes to one file collapse
// into a single DocChanged; a burst spanning several paths flushes together
// once the burst settles.
const debounceInterval = 100 * time.Millisecond

// EventKind classifies a live-reload [Event] into what the API layer needs to
// invalidate.
type EventKind int

const (
	// DocChanged means a file's contents changed (write or create). The
	// Event's Path is the file, relative to its root.
	DocChanged EventKind = iota
	// TreeChanged means a directory's one-level listing changed (an entry was
	// created, removed, or renamed). The Event's Path is that directory,
	// relative to its root ("." for the root itself).
	TreeChanged
	// RootsChanged means the set of registered roots changed. It carries no
	// Path; the watcher never emits it — the Service publishes it on the
	// shared [Hub] when a root is added or removed.
	RootsChanged
)

// String renders an EventKind for logs and tests.
func (k EventKind) String() string {
	switch k {
	case DocChanged:
		return "DocChanged"
	case TreeChanged:
		return "TreeChanged"
	case RootsChanged:
		return "RootsChanged"
	default:
		return fmt.Sprintf("EventKind(%d)", int(k))
	}
}

// Event is one debounced, classified live-reload notification fanned out to
// [Hub] subscribers. Path is root-relative and its meaning depends on Kind
// (see [EventKind]); RootID identifies the emitting root and is empty for a
// RootsChanged event.
type Event struct {
	Kind   EventKind
	RootID string
	Path   string
}

// Hub is a broadcast hub: it fans one [Event] out to every current subscriber.
// It is safe for concurrent use and shared across all watchers — a subscriber
// registers once and receives events from every root (each tagged with its
// RootID), which is what a Service that adds and removes roots at runtime
// wants: watchers come and go without subscribers re-subscribing, and the
// Service publishes RootsChanged through the same hub.
//
// A slow subscriber never blocks the publisher: each subscriber's channel is
// buffered and [Hub.Publish] delivers non-blocking, dropping when the buffer
// is full. Dropped events are harmless — they are invalidation hints, and a
// subscriber reconnecting after a drop revalidates everything.
type Hub struct {
	mu   sync.Mutex
	next int
	subs map[int]chan Event
}

// subBuffer is the per-subscriber channel depth. Deep enough to absorb a
// normal debounced burst; a subscriber that still can't keep up drops the
// overflow.
const subBuffer = 32

// NewHub returns an empty broadcast hub ready for use.
func NewHub() *Hub {
	return &Hub{subs: make(map[int]chan Event)}
}

// Subscribe registers a new subscriber and returns its receive channel plus an
// unsubscribe function. Calling unsubscribe removes the subscriber and closes
// the channel; it is idempotent. The returned channel is buffered and lossy
// under backpressure (see [Hub]).
func (h *Hub) Subscribe() (<-chan Event, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	id := h.next
	h.next++
	ch := make(chan Event, subBuffer)
	h.subs[id] = ch

	var once sync.Once
	unsub := func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			if c, ok := h.subs[id]; ok {
				delete(h.subs, id)
				close(c)
			}
		})
	}
	return ch, unsub
}

// Publish fans ev out to every current subscriber without blocking: a
// subscriber whose buffer is full drops ev. Both Publish and the unsubscribe
// closure hold h.mu, so a send never races a channel close.
func (h *Hub) Publish(ev Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ch := range h.subs {
		select {
		case ch <- ev:
		default:
			// Slow subscriber: drop rather than stall the watcher.
		}
	}
}

// Watcher recursively watches one root directory for changes and publishes
// debounced, classified [Event]s to a shared [Hub]. fsnotify is not recursive,
// so the watcher walks the root on start, watches every directory, and adds a
// watch for each directory created at runtime (rescanning it, since events
// during its creation may be missed); directories that vanish are dropped.
// One Watcher instance serves one root; the Service runs one per registered
// root under an errgroup.
type Watcher struct {
	logger   *slog.Logger
	rootID   string
	rootPath string
	patterns []string // ignore globs (default + configured), matched by base name
	hub      *Hub
	debounce time.Duration
}

// NewWatcher constructs a watcher for the directory rootPath, tagging its
// events with rootID and publishing them to hub. ignorePatterns are extra
// directory-name globs ([filepath.Match] syntax) to skip, on top of the
// built-in [defaultIgnorePatterns]; the matching approach mirrors
// `crabswarm git list`'s directory ignore. rootPath must name an existing
// directory and every pattern must be a valid glob, else an error is returned.
// A nil logger falls back to [slog.Default].
func NewWatcher(
	logger *slog.Logger,
	rootID, rootPath string,
	ignorePatterns []string,
	hub *Hub,
) (*Watcher, error) {
	abs, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("preview watch: resolving %q: %w", rootPath, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("preview watch: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("preview watch: %q is not a directory", rootPath)
	}
	patterns := append(append([]string{}, defaultIgnorePatterns...), ignorePatterns...)
	if err := validatePatterns(patterns); err != nil {
		return nil, fmt.Errorf("preview watch: %w", err)
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Watcher{
		logger:   logger,
		rootID:   rootID,
		rootPath: abs,
		patterns: patterns,
		hub:      hub,
		debounce: debounceInterval,
	}, nil
}

// Run watches the root until ctx is cancelled, blocking for its whole
// lifetime. It returns nil on a clean shutdown (ctx cancelled or the fsnotify
// channels closed) and a non-nil error only when the watcher cannot be created
// or its initial walk fails, so it composes as an errgroup.Group.Go body.
func (w *Watcher) Run(ctx context.Context) error {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("preview watch: new fsnotify watcher: %w", err)
	}
	defer func() { _ = fw.Close() }()

	if err := w.watchTree(fw, w.rootPath); err != nil {
		return fmt.Errorf("preview watch: initial walk of %q: %w", w.rootPath, err)
	}

	// pending holds the latest coalesced event per (kind, path); timer fires
	// once the burst that filled it has been quiet for w.debounce.
	pending := make(map[eventKey]Event)
	timer := time.NewTimer(w.debounce)
	if !timer.Stop() {
		<-timer.C
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-fw.Events:
			if !ok {
				return nil
			}
			w.classify(fw, ev, pending)
			if len(pending) > 0 {
				resetTimer(timer, w.debounce)
			}
		case err, ok := <-fw.Errors:
			if !ok {
				return nil
			}
			// A watch-limit or transient error must not kill the watcher;
			// surface it and keep going.
			w.logger.Warn("preview watch: fsnotify error", "root", w.rootPath, "err", err)
		case <-timer.C:
			for key, e := range pending {
				w.hub.Publish(e)
				delete(pending, key)
			}
		}
	}
}

// eventKey coalesces events by kind and root-relative path so a burst of
// writes to one file, or repeated tree churn in one directory, publishes once.
type eventKey struct {
	kind EventKind
	path string
}

// classify handles dynamic watch registration for ev and records the API-facing
// events it implies into pending. It runs on the single Run goroutine, so
// pending needs no locking.
func (w *Watcher) classify(fw *fsnotify.Watcher, ev fsnotify.Event, pending map[eventKey]Event) {
	name := ev.Name

	if ev.Op.Has(fsnotify.Create) {
		// A newly created directory must be watched immediately, together with
		// any children that already appeared, because inotify does not watch
		// subtrees and events during creation may already be lost.
		if info, err := os.Stat(
			name,
		); err == nil && info.IsDir() &&
			!w.ignored(filepath.Base(name)) {
			w.watchTree(fw, name)
		}
	}
	if ev.Op.Has(fsnotify.Remove) || ev.Op.Has(fsnotify.Rename) {
		// fsnotify auto-drops the watch for a vanished directory; remove it
		// defensively and ignore the "not watched" error for plain files.
		_ = fw.Remove(name)
	}

	// A write to a file is a document change.
	if ev.Op.Has(fsnotify.Write) {
		w.record(pending, DocChanged, w.rel(name))
	}
	// Creating, removing, or renaming an entry changes its directory's listing.
	if ev.Op.Has(fsnotify.Create) || ev.Op.Has(fsnotify.Remove) || ev.Op.Has(fsnotify.Rename) {
		w.record(pending, TreeChanged, w.rel(filepath.Dir(name)))
		// A freshly created file is also new content worth rendering.
		if ev.Op.Has(fsnotify.Create) {
			if info, err := os.Stat(name); err == nil && !info.IsDir() {
				w.record(pending, DocChanged, w.rel(name))
			}
		}
	}
}

// record stores (or overwrites) the coalesced event for (kind, path).
func (w *Watcher) record(pending map[eventKey]Event, kind EventKind, relPath string) {
	pending[eventKey{kind, relPath}] = Event{Kind: kind, RootID: w.rootID, Path: relPath}
}

// watchTree walks dir and adds a watch for it and every non-ignored
// subdirectory. It is used for the initial root walk and for each directory
// created at runtime. A single unreadable directory or failed Add is logged
// and skipped rather than aborting the walk, so a hit watch limit degrades the
// service instead of stopping it. dir itself is never ignore-filtered (it was
// vetted by the caller); its descendants are.
func (w *Watcher) watchTree(fw *fsnotify.Watcher, dir string) error {
	return filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			w.logger.Debug("preview watch: walk error", "path", p, "err", err)
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if p != dir && w.ignored(d.Name()) {
			return fs.SkipDir
		}
		if err := fw.Add(p); err != nil {
			w.logger.Warn("preview watch: add watch failed", "path", p, "err", err)
		}
		return nil
	})
}

// rel returns abs relative to the root; it falls back to abs when the two
// share no base (which should not happen for watched paths).
func (w *Watcher) rel(abs string) string {
	r, err := filepath.Rel(w.rootPath, abs)
	if err != nil {
		return abs
	}
	return r
}

// ignored reports whether a directory base name matches any ignore pattern.
func (w *Watcher) ignored(name string) bool {
	return matchesAny(w.patterns, name)
}

// validatePatterns reports the first malformed glob in patterns
// ([filepath.Match] syntax), mirroring `crabswarm git list`'s up-front
// validation so a bad ignore pattern fails at construction, not mid-walk.
func validatePatterns(patterns []string) error {
	for _, pat := range patterns {
		if _, err := filepath.Match(pat, ""); err != nil {
			return fmt.Errorf("invalid ignore pattern %q: %w", pat, err)
		}
	}
	return nil
}

// matchesAny reports whether name matches any pattern via [filepath.Match],
// the same base-name approach `crabswarm git list` uses to skip directories.
// Patterns are validated at construction, so a match error is treated as no
// match.
func matchesAny(patterns []string, name string) bool {
	for _, pat := range patterns {
		if ok, err := filepath.Match(pat, name); err == nil && ok {
			return true
		}
	}
	return false
}

// resetTimer safely rearms t to fire after d, draining a pending fire first so
// the debounce window restarts cleanly regardless of prior timer state.
func resetTimer(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}
