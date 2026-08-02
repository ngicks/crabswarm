package preview

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestRootStore_AddIdempotent(t *testing.T) {
	dir := t.TempDir()
	s := NewRootStore()

	first, err := s.Add(t.Context(), dir)
	assert.NilError(t, err)
	assert.Equal(t, first.Path, dir)
	assert.Assert(t, first.ID != "")
	assert.Equal(t, first.Name, filepath.Base(dir))

	// Re-adding the same directory (via a non-cleaned path) returns the same
	// entry and does not create a duplicate.
	second, err := s.Add(t.Context(), filepath.Join(dir, "."))
	assert.NilError(t, err)
	assert.Equal(t, second.ID, first.ID)
	assert.Equal(t, second.Name, first.Name)
	assert.Equal(t, len(s.List()), 1)
}

func TestRootStore_AddRequiresExistingDir(t *testing.T) {
	s := NewRootStore()

	// A regular file is not a directory.
	file := filepath.Join(t.TempDir(), "file.md")
	assert.NilError(t, os.WriteFile(file, []byte("x"), 0o644))
	_, err := s.Add(t.Context(), file)
	assert.Assert(t, err != nil, "adding a file should fail")

	// A missing path fails too.
	_, err = s.Add(t.Context(), filepath.Join(t.TempDir(), "does-not-exist"))
	assert.Assert(t, err != nil, "adding a missing path should fail")

	assert.Equal(t, len(s.List()), 0)
}

func TestRootStore_NameDedup(t *testing.T) {
	base := t.TempDir()
	a := filepath.Join(base, "x", "repo")
	b := filepath.Join(base, "y", "repo")
	c := filepath.Join(base, "z", "repo")
	for _, d := range []string{a, b, c} {
		assert.NilError(t, os.MkdirAll(d, 0o755))
	}
	s := NewRootStore()

	ra, err := s.Add(t.Context(), a)
	assert.NilError(t, err)
	assert.Equal(t, ra.Name, "repo")

	rb, err := s.Add(t.Context(), b)
	assert.NilError(t, err)
	assert.Equal(t, rb.Name, "repo-2")

	rc, err := s.Add(t.Context(), c)
	assert.NilError(t, err)
	assert.Equal(t, rc.Name, "repo-3")

	// Distinct paths get distinct IDs.
	assert.Assert(t, ra.ID != rb.ID && rb.ID != rc.ID && ra.ID != rc.ID)
}

// gitRun runs git with args in dir, failing the test on any error. Tests
// assume a git binary is installed, matching displayBase itself.
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	assert.NilError(t, err, "git %v: %s", args, out)
}

// gitInitRepo creates a git repository with one commit at dir (worktree add
// needs a commit to point at).
func gitInitRepo(t *testing.T, dir string) {
	t.Helper()
	assert.NilError(t, os.MkdirAll(dir, 0o755))
	gitRun(t, dir, "init")
	gitRun(t, dir, "-c", "user.name=t", "-c", "user.email=t@example.com",
		"commit", "--allow-empty", "-m", "init")
}

func TestRootStore_WorktreeName(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "origin")
	gitInitRepo(t, repo)
	wt := filepath.Join(base, "crabswarm", "main")
	gitRun(t, repo, "worktree", "add", "--detach", wt)
	sub := filepath.Join(wt, "docs")
	assert.NilError(t, os.MkdirAll(sub, 0o755))

	s := NewRootStore()

	// A linked worktree's display name gains the parent directory prefix.
	rw, err := s.Add(t.Context(), wt)
	assert.NilError(t, err)
	assert.Equal(t, rw.Name, "crabswarm/main")

	// The main working tree keeps the plain base name.
	rr, err := s.Add(t.Context(), repo)
	assert.NilError(t, err)
	assert.Equal(t, rr.Name, "origin")

	// A subdirectory inside a worktree is not itself the worktree: no prefix.
	rs, err := s.Add(t.Context(), sub)
	assert.NilError(t, err)
	assert.Equal(t, rs.Name, "docs")

	// The prefixed name participates in Remove-by-name as usual.
	_, ok := s.Remove("crabswarm/main")
	assert.Assert(t, ok)
}

func TestRootStore_WorktreeNameDedup(t *testing.T) {
	base := t.TempDir()
	// Two same-named worktrees of different repositories under same-named
	// parents still collide; the "-N" suffix applies to the full prefixed name.
	var wts []string
	for _, side := range []string{"x", "y"} {
		repo := filepath.Join(base, side, "origin")
		gitInitRepo(t, repo)
		wt := filepath.Join(base, side, "crabswarm", "main")
		gitRun(t, repo, "worktree", "add", "--detach", wt)
		wts = append(wts, wt)
	}
	s := NewRootStore()

	ra, err := s.Add(t.Context(), wts[0])
	assert.NilError(t, err)
	assert.Equal(t, ra.Name, "crabswarm/main")

	rb, err := s.Add(t.Context(), wts[1])
	assert.NilError(t, err)
	assert.Equal(t, rb.Name, "crabswarm/main-2")
}

func TestRootStore_Remove(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	s := NewRootStore()

	ra, err := s.Add(t.Context(), dirA)
	assert.NilError(t, err)
	rb, err := s.Add(t.Context(), dirB)
	assert.NilError(t, err)

	// Remove by ID.
	removed, ok := s.Remove(ra.ID)
	assert.Assert(t, ok)
	assert.Equal(t, removed.ID, ra.ID)
	_, ok = s.Get(ra.ID)
	assert.Assert(t, !ok)

	// Remove by Name.
	removed, ok = s.Remove(rb.Name)
	assert.Assert(t, ok)
	assert.Equal(t, removed.ID, rb.ID)

	assert.Equal(t, len(s.List()), 0)

	// Removing something unknown reports false.
	_, ok = s.Remove("nope")
	assert.Assert(t, !ok)
}

func TestRootStore_RemoveFreesName(t *testing.T) {
	base := t.TempDir()
	a := filepath.Join(base, "x", "repo")
	b := filepath.Join(base, "y", "repo")
	for _, d := range []string{a, b} {
		assert.NilError(t, os.MkdirAll(d, 0o755))
	}
	s := NewRootStore()

	ra, err := s.Add(t.Context(), a)
	assert.NilError(t, err)
	assert.Equal(t, ra.Name, "repo")
	rb, err := s.Add(t.Context(), b)
	assert.NilError(t, err)
	assert.Equal(t, rb.Name, "repo-2")

	// Dropping "repo" frees the base name for the next collision.
	_, ok := s.Remove(ra.Name)
	assert.Assert(t, ok)

	c := filepath.Join(base, "z", "repo")
	assert.NilError(t, os.MkdirAll(c, 0o755))
	rc, err := s.Add(t.Context(), c)
	assert.NilError(t, err)
	assert.Equal(t, rc.Name, "repo")
}
