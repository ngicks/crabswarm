package preview

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// Root is a directory registered with the previewer. [RootStore] derives its
// ID and Name; the API references a root by either.
type Root struct {
	// ID is a stable, short, URL-safe hash of the absolute path. The same
	// path always yields the same ID, so re-adding a root is idempotent and
	// the ID is safe to embed in URLs (e.g. "/r/{id}/...").
	ID string
	// Path is the absolute, cleaned filesystem path of the root directory.
	Path string
	// Name is the display name — the path's base name, prefixed with the
	// parent directory name when the path is a linked git worktree, and
	// suffixed on collision so every registered root carries a unique Name.
	Name string
}

// RootStore is the in-memory registry of preview roots. It is safe for
// concurrent use. The registry is process-local state only: restarting the
// daemon starts empty (decided — no persistence).
type RootStore struct {
	mu    sync.RWMutex
	roots map[string]Root   // keyed by Root.ID
	names map[string]string // Name -> ID; the set of taken display names
}

// NewRootStore returns an empty registry ready for use.
func NewRootStore() *RootStore {
	return &RootStore{
		roots: make(map[string]Root),
		names: make(map[string]string),
	}
}

// Add registers path as a preview root and returns it. path is resolved to an
// absolute, cleaned path that must name an existing directory. Add is
// idempotent: re-adding the same directory returns the existing [Root]
// unchanged (same ID, same Name) without creating a duplicate entry.
func (s *RootStore) Add(ctx context.Context, path string) (Root, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Root{}, fmt.Errorf("resolving %q: %w", path, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return Root{}, fmt.Errorf("adding root %q: %w", path, err)
	}
	if !info.IsDir() {
		return Root{}, fmt.Errorf("adding root %q: not a directory", path)
	}

	id := rootID(abs)
	// Resolved before taking the lock: displayBase shells out to git and must
	// not stall other store readers behind it.
	base := displayBase(ctx, abs)

	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.roots[id]; ok {
		// Same absolute path -> same ID: idempotent, no duplicate entry.
		return existing, nil
	}
	root := Root{
		ID:   id,
		Path: abs,
		Name: s.uniqueName(base),
	}
	s.roots[id] = root
	s.names[root.Name] = id
	return root, nil
}

// Get returns the root with the given ID. The bool reports whether it exists.
func (s *RootStore) Get(id string) (Root, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	root, ok := s.roots[id]
	return root, ok
}

// List returns every registered root sorted by Name. The returned slice is a
// fresh copy; mutating it does not affect the store.
func (s *RootStore) List() []Root {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Root, 0, len(s.roots))
	for _, r := range s.roots {
		out = append(out, r)
	}
	slices.SortFunc(out, func(a, b Root) int {
		return strings.Compare(a.Name, b.Name)
	})
	return out
}

// Remove drops the root identified by idOrName — matched against Root.ID
// first, then Root.Name — releasing its Name for reuse. It returns the removed
// root and whether one was found.
func (s *RootStore) Remove(idOrName string) (Root, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := idOrName
	if _, ok := s.roots[id]; !ok {
		// Not an ID; try to resolve it as a Name.
		nid, ok := s.names[idOrName]
		if !ok {
			return Root{}, false
		}
		id = nid
	}
	root := s.roots[id]
	delete(s.roots, id)
	delete(s.names, root.Name)
	return root, true
}

// uniqueName returns base when it is free, otherwise base with the smallest
// "-N" (N>=2) suffix that is not yet taken. Callers hold s.mu.
func (s *RootStore) uniqueName(base string) string {
	if _, taken := s.names[base]; !taken {
		return base
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d", base, n)
		if _, taken := s.names[candidate]; !taken {
			return candidate
		}
	}
}

// displayBase returns the display-name base for abs. Normally that is the
// path's base name; when abs is the top level of a linked git worktree the
// parent directory name is prefixed. Worktrees are commonly named after
// branches ("main"), so the base name alone collides across repositories and
// degrades into "main-2", "main-3", …; "repo/main" stays telling.
//
// git itself is the detector (decided: shell out rather than parse the .git
// gitdir pointer file; git is assumed installed). A linked worktree is one
// whose --git-dir differs from --git-common-dir; the main working tree and
// regular clones have the two equal. Any git failure — not a repository,
// binary missing — just means no prefix: the name is cosmetic, so Add never
// fails on it.
func displayBase(ctx context.Context, abs string) string {
	base := filepath.Base(abs)
	out, err := exec.CommandContext(ctx, "git", "-C", abs, "rev-parse",
		"--path-format=absolute", "--show-toplevel", "--git-dir", "--git-common-dir").Output()
	if err != nil {
		return base
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 3 {
		return base
	}
	toplevel, gitDir, commonDir := lines[0], lines[1], lines[2]
	// Only the worktree's own top level gets the prefix: for a subdirectory
	// registered as a root, the parent name is arbitrary, not the repository.
	if toplevel != abs || gitDir == commonDir {
		return base
	}
	parent := filepath.Base(filepath.Dir(abs))
	if parent == "." || parent == string(filepath.Separator) {
		return base
	}
	return parent + "/" + base
}

// rootID derives a stable, short, URL-safe identifier from an absolute path:
// the first 8 bytes of its SHA-256 digest, base64url-encoded (11 chars, no
// padding). Deterministic, so the same path always maps to the same ID.
func rootID(absPath string) string {
	sum := sha256.Sum256([]byte(absPath))
	return base64.RawURLEncoding.EncodeToString(sum[:8])
}
