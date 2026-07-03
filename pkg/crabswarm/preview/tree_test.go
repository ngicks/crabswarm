package preview

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestTree_ListsDirsFirstThenFiles(t *testing.T) {
	root := t.TempDir()
	// Files and directories in intentionally unsorted creation order.
	mkdir(t, filepath.Join(root, "zeta"))
	mkdir(t, filepath.Join(root, "alpha"))
	writeFile(t, filepath.Join(root, "b.txt"))
	writeFile(t, filepath.Join(root, "a.md"))
	writeFile(t, filepath.Join(root, "c.markdown"))
	writeFile(t, filepath.Join(root, "d.MD"))
	// A file inside a subdir must not leak into the one-level listing.
	writeFile(t, filepath.Join(root, "alpha", "nested.md"))

	entries, err := Tree(root, ".")
	assert.NilError(t, err)

	got := make([]string, len(entries))
	for i, e := range entries {
		got[i] = e.Name
	}
	// Directories first (sorted), then files (sorted) — no descent.
	want := []string{"alpha", "zeta", "a.md", "b.txt", "c.markdown", "d.MD"}
	assert.DeepEqual(t, got, want)

	byName := map[string]Entry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	assert.Equal(t, byName["alpha"].Type, EntryDir)
	assert.Equal(t, byName["alpha"].IsMarkdown, false)
	assert.Equal(t, byName["a.md"].Type, EntryFile)
	assert.Equal(t, byName["a.md"].IsMarkdown, true)
	assert.Equal(t, byName["c.markdown"].IsMarkdown, true)
	assert.Equal(t, byName["d.MD"].IsMarkdown, true) // extension match is case-insensitive
	assert.Equal(t, byName["b.txt"].IsMarkdown, false)
}

func TestTree_ListsSubdir(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "docs")
	mkdir(t, sub)
	writeFile(t, filepath.Join(sub, "readme.md"))

	entries, err := Tree(root, "docs")
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 1)
	assert.Equal(t, entries[0].Name, "readme.md")
	assert.Equal(t, entries[0].IsMarkdown, true)
}

func TestSafeJoin_AllowsInside(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	mkdir(t, sub)

	got, err := SafeJoin(root, filepath.Join("a", "b"))
	assert.NilError(t, err)
	// EvalSymlinks canonicalizes both sides, so compare against the resolved
	// expectation rather than the raw join.
	wantRoot, err := filepath.EvalSymlinks(root)
	assert.NilError(t, err)
	assert.Equal(t, got, filepath.Join(wantRoot, "a", "b"))

	// The root itself ("." or "") is in-bounds.
	gotRoot, err := SafeJoin(root, ".")
	assert.NilError(t, err)
	assert.Equal(t, gotRoot, wantRoot)
}

func TestSafeJoin_RejectsEscape(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "sub"))

	cases := map[string]string{
		"parent":           "..",
		"parent traversal": filepath.Join("..", "secret"),
		"nested traversal": filepath.Join("sub", "..", "..", "secret"),
		"absolute":         string(filepath.Separator) + filepath.Join("etc", "passwd"),
	}
	for name, rel := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := SafeJoin(root, rel)
			assert.Assert(t, errors.Is(err, ErrPathEscape), "want ErrPathEscape, got %v", err)
		})
	}
}

func TestSafeJoin_RejectsSymlinkToOutsideDir(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "secret.txt"))

	// A symlink inside the root pointing at a directory outside it.
	assert.NilError(t, os.Symlink(outside, filepath.Join(root, "escape")))

	_, err := SafeJoin(root, "escape")
	assert.Assert(t, errors.Is(err, ErrPathEscape), "want ErrPathEscape, got %v", err)

	// Reaching a file through the escaping symlink is rejected too.
	_, err = SafeJoin(root, filepath.Join("escape", "secret.txt"))
	assert.Assert(t, errors.Is(err, ErrPathEscape), "want ErrPathEscape, got %v", err)
}

func TestSafeJoin_RejectsSymlinkToOutsideFile(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	writeFile(t, secret)

	// A symlink inside the root pointing directly at a file outside it.
	assert.NilError(t, os.Symlink(secret, filepath.Join(root, "leak.txt")))

	_, err := SafeJoin(root, "leak.txt")
	assert.Assert(t, errors.Is(err, ErrPathEscape), "want ErrPathEscape, got %v", err)
}

func TestSafeJoin_AllowsSymlinkInsideRoot(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "real")
	mkdir(t, target)
	writeFile(t, filepath.Join(target, "doc.md"))

	// A symlink that stays within the root is fine.
	assert.NilError(t, os.Symlink(target, filepath.Join(root, "link")))

	got, err := SafeJoin(root, filepath.Join("link", "doc.md"))
	assert.NilError(t, err)
	wantRoot, err := filepath.EvalSymlinks(root)
	assert.NilError(t, err)
	assert.Equal(t, got, filepath.Join(wantRoot, "real", "doc.md"))
}

func TestTree_RejectsEscape(t *testing.T) {
	root := t.TempDir()
	_, err := Tree(root, "..")
	assert.Assert(t, errors.Is(err, ErrPathEscape), "want ErrPathEscape, got %v", err)
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	assert.NilError(t, os.MkdirAll(path, 0o755))
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	assert.NilError(t, os.WriteFile(path, []byte("x"), 0o644))
}
