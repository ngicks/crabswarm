package git

import (
	"context"
	"fmt"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"strings"
)

// WalkEntry is one discovered repository root or worktree directory yielded by
// [Service.WalkRepos] or [Service.WalkWorktrees], paired in the iterator with
// a per-entry error.
type WalkEntry struct {
	// Path is the absolute path of the repository root or worktree directory.
	Path string
	// Repo is the repository the entry belongs to, with absolute Dir (and,
	// for worktree entries, CommonDir). For a repo-root entry Repo.Dir
	// equals Path.
	Repo Repo
}

// WalkRepos discovers git repository roots under root and yields them as an
// [iter.Seq2] of (entry, error) pairs, one per repository, emitted as it is
// found so a consumer streaming to stdout (e.g. piping to fzf) sees results
// immediately. The paired error is per-entry and non-fatal: a non-nil error
// reports a directory that could not be read, and iteration continues — a
// single failure never aborts the whole listing. Stop early by breaking out
// of the range loop.
//
// A repository root is any directory holding a ".git" entry — a directory for
// an ordinary clone or the pointer file of the bare-plus-worktree layout — so
// repositories created by crabswarm, ghq, or plain git are all discovered. A
// discovered root is emitted and not descended into, which keeps its
// per-branch worktree subdirectories (each with their own ".git") from being
// reported as separate repositories. Dot-directories (".bare", ".git",
// ".cache", …) are skipped during descent, and symlinked directories are not
// followed (lstat-based, [filepath.WalkDir] semantics).
//
// When root does not exist the iterator yields a single (zero-entry, error)
// pair wrapping the stat failure; it does not panic or silently complete.
func (s Service) WalkRepos(ctx context.Context, root string) iter.Seq2[WalkEntry, error] {
	return func(yield func(WalkEntry, error) bool) {
		abs, err := filepath.Abs(root)
		if err != nil {
			yield(WalkEntry{Path: root}, fmt.Errorf("resolving %q: %w", root, err))
			return
		}
		if _, err := os.Lstat(abs); err != nil {
			yield(WalkEntry{Path: abs}, fmt.Errorf("walking %q: %w", abs, err))
			return
		}

		stopped := false
		walkErr := filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
			if cerr := ctx.Err(); cerr != nil {
				return cerr
			}
			if err != nil {
				// Report the unreadable directory and keep walking the rest.
				if !yield(WalkEntry{Path: p}, err) {
					stopped = true
					return fs.SkipAll
				}
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			// Skip dot-directories (.bare, .git, .cache, …) but never the
			// root itself, even when the base dir happens to be a dotdir.
			if p != abs && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			if !isRepoRoot(p) {
				return nil
			}
			if !yield(WalkEntry{Path: p, Repo: Repo{Dir: p}}, nil) {
				stopped = true
				return fs.SkipAll
			}
			// A repo root owns the directory subtree; do not descend so its
			// worktree siblings are not reported as separate repositories.
			return fs.SkipDir
		})
		if walkErr != nil && !stopped && ctx.Err() != nil {
			yield(WalkEntry{Path: abs}, ctx.Err())
		}
	}
}

// WalkWorktrees discovers git repository roots under root like
// [Service.WalkRepos] and, for each, yields one entry per worktree (the bare
// entry excluded) as reported by [Service.ListWorktrees]. Results stream as
// repositories are found. The paired error is per-entry and non-fatal: a
// repository whose worktrees cannot be listed (e.g. it is corrupt) yields a
// single (entry, error) pair and the walk continues. Stop early by breaking
// out of the range loop.
func (s Service) WalkWorktrees(ctx context.Context, root string) iter.Seq2[WalkEntry, error] {
	return func(yield func(WalkEntry, error) bool) {
		for entry, err := range s.WalkRepos(ctx, root) {
			if err != nil {
				if !yield(entry, err) {
					return
				}
				continue
			}
			repo, err := s.ResolveRepo(ctx, entry.Path)
			if err != nil {
				if !yield(WalkEntry{Path: entry.Path, Repo: entry.Repo}, err) {
					return
				}
				continue
			}
			wts, err := s.ListWorktrees(ctx, repo)
			if err != nil {
				if !yield(WalkEntry{Path: entry.Path, Repo: repo}, err) {
					return
				}
				continue
			}
			for _, wt := range wts {
				if wt.Bare {
					continue
				}
				if !yield(WalkEntry{Path: wt.Path, Repo: repo}, nil) {
					return
				}
			}
		}
	}
}

// isRepoRoot reports whether dir holds a ".git" entry — a directory (ordinary
// clone) or the pointer file of the bare-plus-worktree layout.
func isRepoRoot(dir string) bool {
	_, err := os.Lstat(filepath.Join(dir, gitDirFile))
	return err == nil
}
