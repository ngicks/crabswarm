package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNoGitRoot is returned by [Service.FindRoot] when neither the directory
// walk nor git can locate a repository for the start directory.
var ErrNoGitRoot = errors.New(
	"no git repository found between the current directory and the filesystem root",
)

// NotBareCloneError reports that the resolved git root is an ordinary clone
// rather than crabswarm's bare-plus-worktree layout: its shared git directory
// is a ".git" working tree rather than a ".bare" bare clone. [Service.FindRoot]
// returns it so callers can tell "not a --bare clone" apart from
// [ErrNoGitRoot].
type NotBareCloneError struct {
	// Dir is the ordinary clone's root directory.
	Dir string
}

func (e *NotBareCloneError) Error() string {
	return fmt.Sprintf(
		"git root %q is not a --bare clone (it has no %q directory)",
		e.Dir,
		bareDirName,
	)
}

// FindRoot resolves the crabswarm bare-plus-worktree repository root for
// startDir (the current working directory when empty) — the directory that
// holds the ".bare" bare clone, the same directory [Service.Clone] materializes.
//
// It first walks from startDir up to the filesystem root, which resolves the
// common case — the repository directory and its sibling worktrees — without
// shelling out to git. When the walk reaches the filesystem root without
// finding a repository, as happens for a worktree placed outside its repository
// directory (where climbing never reaches ".bare"), it falls back to asking git
// for the worktree's common directory, which follows the worktree link no
// matter where the worktree lives.
//
// It returns [ErrNoGitRoot] when neither method finds a repository, and a
// [*NotBareCloneError] when the resolved root is an ordinary (non-bare) clone.
func (s Service) FindRoot(ctx context.Context, startDir string) (string, error) {
	start := startDir
	if start == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolving current directory: %w", err)
		}
		start = wd
	}

	root, err := findBareRootByWalk(start)
	if !errors.Is(err, ErrNoGitRoot) {
		// The walk either resolved a root or classified the nearest one as an
		// ordinary clone — both verdicts are definitive, so return them as-is.
		return root, err
	}

	// Nothing on disk between start and the filesystem root. The directory may
	// still be a worktree whose repository lives elsewhere; ask git, which
	// follows the worktree link.
	root, gerr := s.findBareRootByGit(ctx, start)
	if gerr == nil {
		return root, nil
	}
	if _, ok := errors.AsType[*NotBareCloneError](gerr); ok {
		return "", gerr
	}
	// git could not resolve a repository either; report the walk's verdict.
	return "", err
}

// findBareRootByWalk climbs from startDir to the filesystem root, classifying
// each ancestor: a ".bare" directory marks the bare-plus-worktree root (checked
// first, so the repository directory — which carries both ".bare" and a ".git"
// pointer file — resolves to itself); a ".git" *directory* marks an ordinary
// clone; a ".git" *file* is a worktree pointer that is climbed past. It returns
// [ErrNoGitRoot] when it reaches the root having found neither layout.
func findBareRootByWalk(startDir string) (string, error) {
	abs, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving %q: %w", startDir, err)
	}
	for {
		if isDir(filepath.Join(abs, bareDirName)) {
			return abs, nil
		}
		if isDir(filepath.Join(abs, gitDirFile)) {
			// A ".git" directory (rather than a worktree pointer file) is an
			// ordinary clone, not crabswarm's bare layout.
			return "", &NotBareCloneError{Dir: abs}
		}
		// A ".git" file is a worktree pointer (or there is no ".git" at all):
		// keep climbing toward the directory that holds ".bare".
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", ErrNoGitRoot
		}
		abs = parent
	}
}

// findBareRootByGit asks git for the repository owning startDir and returns its
// root when the shared git directory is a ".bare" bare clone. It returns a
// [*NotBareCloneError] when git resolves an ordinary clone instead, and the
// underlying ResolveRepo error when startDir is not inside any git repository.
func (s Service) findBareRootByGit(ctx context.Context, startDir string) (string, error) {
	repo, err := s.ResolveRepo(ctx, startDir)
	if err != nil {
		return "", err
	}
	if filepath.Base(repo.CommonDir) != bareDirName {
		return "", &NotBareCloneError{Dir: repo.Dir}
	}
	return repo.Dir, nil
}

// isDir reports whether path exists and is a directory, following symlinks to
// match how git treats the ".bare"/".git" entries.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
