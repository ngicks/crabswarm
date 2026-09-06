package issues

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// Source is a registered beads database. [SourceStore] derives its ID from
// the .beads directory bd reports for the registered directory, so every
// worktree of one repository registers as the same source.
type Source struct {
	// ID is a stable, short, URL-safe hash of BeadsPath. The same database
	// always yields the same ID, so re-adding it is idempotent and the ID is
	// safe to embed in URLs.
	ID string
	// BeadsPath is the absolute .beads directory.
	BeadsPath string
	// Prefix is the issue-ID prefix every issue of this database carries; it
	// is the source's display name.
	Prefix string
	// Dir is the directory the source was registered from. bd runs there, so
	// it must keep resolving to BeadsPath.
	Dir string
}

// SourceStore is the in-memory registry of issue sources. It is safe for
// concurrent use. Like the previewer's root registry it is process-local
// state only: restarting the daemon starts empty.
type SourceStore struct {
	mu      sync.RWMutex
	sources map[string]Source // keyed by Source.ID
}

// NewSourceStore returns an empty registry ready for use.
func NewSourceStore() *SourceStore {
	return &SourceStore{sources: make(map[string]Source)}
}

// Add resolves the beads database governing dir with [Where] and registers
// it. dir may be relative; it is resolved against the process working
// directory before bd runs there. The bool reports whether this call created
// the entry: a directory resolving to an already registered database returns
// that source with false, which is how two worktrees of one repository stay
// one source. It returns an error wrapping [ErrNoBeads] when dir belongs to
// no beads workspace.
func (s *SourceStore) Add(ctx context.Context, dir string) (Source, bool, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Source{}, false, fmt.Errorf("resolving %q: %w", dir, err)
	}
	loc, err := Where(ctx, abs)
	if err != nil {
		return Source{}, false, err
	}
	beads := filepath.Clean(loc.BeadsPath)

	id := sourceID(beads)
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.sources[id]; ok {
		return existing, false, nil
	}
	src := Source{ID: id, BeadsPath: beads, Prefix: loc.Prefix, Dir: abs}
	s.sources[id] = src
	return src, true, nil
}

// Get returns the source with the given ID. The bool reports whether it
// exists.
func (s *SourceStore) Get(id string) (Source, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	src, ok := s.sources[id]
	return src, ok
}

// List returns every registered source sorted by Prefix, then by ID so
// databases sharing a prefix still order deterministically. The returned
// slice is a fresh copy.
func (s *SourceStore) List() []Source {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Source, 0, len(s.sources))
	for _, src := range s.sources {
		out = append(out, src)
	}
	slices.SortFunc(out, func(a, b Source) int {
		if c := strings.Compare(a.Prefix, b.Prefix); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	return out
}

// Remove drops the source with the given ID. It returns the removed source
// and whether one was found.
func (s *SourceStore) Remove(id string) (Source, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src, ok := s.sources[id]
	if !ok {
		return Source{}, false
	}
	delete(s.sources, id)
	return src, true
}

// sourceID derives a stable, short, URL-safe identifier from the absolute
// .beads path: the first 8 bytes of its SHA-256 digest, base64url-encoded
// (11 chars, no padding), the same scheme the previewer uses for roots.
func sourceID(beadsPath string) string {
	sum := sha256.Sum256([]byte(beadsPath))
	return base64.RawURLEncoding.EncodeToString(sum[:8])
}
