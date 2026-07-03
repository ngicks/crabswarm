package preview

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestRootStore_AddIdempotent(t *testing.T) {
	dir := t.TempDir()
	s := NewRootStore()

	first, err := s.Add(dir)
	assert.NilError(t, err)
	assert.Equal(t, first.Path, dir)
	assert.Assert(t, first.ID != "")
	assert.Equal(t, first.Name, filepath.Base(dir))

	// Re-adding the same directory (via a non-cleaned path) returns the same
	// entry and does not create a duplicate.
	second, err := s.Add(filepath.Join(dir, "."))
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
	_, err := s.Add(file)
	assert.Assert(t, err != nil, "adding a file should fail")

	// A missing path fails too.
	_, err = s.Add(filepath.Join(t.TempDir(), "does-not-exist"))
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

	ra, err := s.Add(a)
	assert.NilError(t, err)
	assert.Equal(t, ra.Name, "repo")

	rb, err := s.Add(b)
	assert.NilError(t, err)
	assert.Equal(t, rb.Name, "repo-2")

	rc, err := s.Add(c)
	assert.NilError(t, err)
	assert.Equal(t, rc.Name, "repo-3")

	// Distinct paths get distinct IDs.
	assert.Assert(t, ra.ID != rb.ID && rb.ID != rc.ID && ra.ID != rc.ID)
}

func TestRootStore_Remove(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	s := NewRootStore()

	ra, err := s.Add(dirA)
	assert.NilError(t, err)
	rb, err := s.Add(dirB)
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

	ra, err := s.Add(a)
	assert.NilError(t, err)
	assert.Equal(t, ra.Name, "repo")
	rb, err := s.Add(b)
	assert.NilError(t, err)
	assert.Equal(t, rb.Name, "repo-2")

	// Dropping "repo" frees the base name for the next collision.
	_, ok := s.Remove(ra.Name)
	assert.Assert(t, ok)

	c := filepath.Join(base, "z", "repo")
	assert.NilError(t, os.MkdirAll(c, 0o755))
	rc, err := s.Add(c)
	assert.NilError(t, err)
	assert.Equal(t, rc.Name, "repo")
}
