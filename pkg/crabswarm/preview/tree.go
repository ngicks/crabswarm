package preview

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// EntryType classifies a directory entry as a subdirectory or a regular file.
type EntryType string

const (
	// EntryDir is a subdirectory entry.
	EntryDir EntryType = "DIR"
	// EntryFile is a non-directory entry (all files are listed; symlinks and
	// other non-directory entries are reported as files).
	EntryFile EntryType = "FILE"
)

// Entry is one item in a one-level directory listing under a root.
type Entry struct {
	// Name is the entry's base name (no path components).
	Name string
	// Type is DIR for subdirectories, FILE for everything else.
	Type EntryType
	// IsMarkdown reports whether a FILE entry has a markdown extension
	// (".md" or ".markdown", case-insensitive). Always false for a DIR.
	IsMarkdown bool
}

// ErrPathEscape reports that a requested path is absolute, climbs above its
// root, or resolves through a symlink to a target outside the root. It is
// shared by the tree listing and the raw asset handler so both reject
// traversal attempts identically.
var ErrPathEscape = errors.New("path escapes the root")

// Tree lists the single directory dir (relative to rootPath) one level deep.
// Every entry is returned — no filtering — with subdirectories first, then
// files, each group sorted by name. dir is validated with [SafeJoin], so an
// absolute path, a "../" escape, or a symlink pointing outside rootPath is
// rejected with [ErrPathEscape].
func Tree(rootPath, dir string) ([]Entry, error) {
	target, err := SafeJoin(rootPath, dir)
	if err != nil {
		return nil, err
	}
	dirEntries, err := os.ReadDir(target)
	if err != nil {
		return nil, fmt.Errorf("reading directory %q: %w", dir, err)
	}
	entries := make([]Entry, 0, len(dirEntries))
	for _, de := range dirEntries {
		typ := EntryFile
		if de.IsDir() {
			typ = EntryDir
		}
		entries = append(entries, Entry{
			Name:       de.Name(),
			Type:       typ,
			IsMarkdown: typ == EntryFile && isMarkdown(de.Name()),
		})
	}
	slices.SortFunc(entries, func(a, b Entry) int {
		if a.Type != b.Type {
			// Directories sort before files.
			if a.Type == EntryDir {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Name, b.Name)
	})
	return entries, nil
}

// SafeJoin resolves rel against rootPath and returns the absolute,
// symlink-resolved path of the target, guaranteed to stay inside rootPath. It
// is the shared path-safety gate for every request-supplied path (tree
// listing, document rendering, raw asset serving).
//
// rel must be a root-relative path. It is cleaned, then rejected with
// [ErrPathEscape] when it is absolute or, per [filepath.IsLocal] plus a
// [filepath.Rel] check, would climb out of the subtree. The cleaned path is
// joined onto rootPath and both root and target are passed through
// [filepath.EvalSymlinks]; the resolved target must remain under the resolved
// root, so a symlink escaping the root is rejected too. EvalSymlinks requires
// the target to exist, so a missing path surfaces as an error.
func SafeJoin(rootPath, rel string) (string, error) {
	clean := filepath.Clean(rel)
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("%w: %q is absolute", ErrPathEscape, rel)
	}
	// IsLocal rejects "..", absolute paths, and empty; it guarantees the
	// cleaned path stays within the subtree using lexical analysis only.
	if !filepath.IsLocal(clean) {
		return "", fmt.Errorf("%w: %q", ErrPathEscape, rel)
	}
	joined := filepath.Join(rootPath, clean)
	// Belt-and-suspenders lexical check: the joined path must be reachable
	// from rootPath without stepping upward.
	back, err := filepath.Rel(rootPath, joined)
	if err != nil || escapes(back) {
		return "", fmt.Errorf("%w: %q", ErrPathEscape, rel)
	}
	// Resolve symlinks and confirm the real target stays under the real root,
	// catching a symlink inside the root that points outside it.
	evalRoot, err := filepath.EvalSymlinks(rootPath)
	if err != nil {
		return "", fmt.Errorf("resolving root %q: %w", rootPath, err)
	}
	evalTarget, err := filepath.EvalSymlinks(joined)
	if err != nil {
		return "", fmt.Errorf("resolving %q: %w", rel, err)
	}
	relEval, err := filepath.Rel(evalRoot, evalTarget)
	if err != nil || escapes(relEval) {
		return "", fmt.Errorf("%w: %q resolves outside the root", ErrPathEscape, rel)
	}
	return evalTarget, nil
}

// escapes reports whether a [filepath.Rel] result steps above its base — it is
// exactly ".." or begins with a "../" segment.
func escapes(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// isMarkdown reports whether name has a markdown file extension (".md" or
// ".markdown"), matched case-insensitively.
func isMarkdown(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".markdown":
		return true
	default:
		return false
	}
}
