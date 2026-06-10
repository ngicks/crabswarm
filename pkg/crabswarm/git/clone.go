package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CloneResult describes the layout produced by [Service.Clone].
type CloneResult struct {
	// RepoDir is the repository directory (<base>/<host>/<owner>/<repo>).
	RepoDir string
	// BareDir is the bare clone (<RepoDir>/.bare).
	BareDir string
	// DefaultBranch is the branch the remote HEAD points at (e.g. "main").
	DefaultBranch string
	// WorktreePath is the checked-out worktree of DefaultBranch
	// (<RepoDir>/<DefaultBranch>).
	WorktreePath string
}

// Clone materializes rawUrl under s.BaseDir using the bare-plus-worktree
// layout documented on the package. It clones in --bare mode, wires up a
// normal fetch refspec so later fetches populate refs/remotes/origin/*,
// detects the remote's default branch, and checks that branch out as a
// worktree.
//
// It refuses to proceed when the target repository directory already exists
// and is non-empty, so an existing clone is never clobbered.
func (s Service) Clone(ctx context.Context, rawUrl string) (CloneResult, error) {
	if s.BaseDir == "" {
		return CloneResult{}, fmt.Errorf("clone: base directory is empty")
	}

	rel, err := ParseRepoPath(rawUrl)
	if err != nil {
		return CloneResult{}, err
	}

	repoDir := filepath.Join(s.BaseDir, filepath.FromSlash(rel))
	return s.cloneInto(ctx, rawUrl, repoDir)
}

// cloneInto performs the bare clone of rawUrl into repoDir and the
// default-branch worktree checkout. It is the URL-agnostic core of Clone,
// split out so the layout logic can be exercised against a local origin.
func (s Service) cloneInto(ctx context.Context, rawUrl, repoDir string) (CloneResult, error) {
	bareDir := filepath.Join(repoDir, bareDirName)

	if err := ensureEmptyDir(repoDir); err != nil {
		return CloneResult{}, err
	}
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		return CloneResult{}, fmt.Errorf("creating %q: %w", repoDir, err)
	}

	if err := s.runProgress(ctx, "", "clone", "--bare", rawUrl, bareDir); err != nil {
		return CloneResult{}, err
	}

	// Make <repoDir> itself a git directory pointing at the bare clone.
	gitFile := filepath.Join(repoDir, gitDirFile)
	if err := os.WriteFile(gitFile, []byte("gitdir: ./"+bareDirName+"\n"), 0o644); err != nil {
		return CloneResult{}, fmt.Errorf("writing %q: %w", gitFile, err)
	}

	// A bare clone leaves no normal fetch refspec; add one so future
	// fetches mirror remote heads into refs/remotes/origin/*.
	if _, err := s.run(ctx, repoDir,
		"config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*",
	); err != nil {
		return CloneResult{}, err
	}
	if err := s.runProgress(ctx, repoDir, "fetch", "origin"); err != nil {
		return CloneResult{}, err
	}

	branch, err := s.defaultBranch(ctx, repoDir)
	if err != nil {
		return CloneResult{}, err
	}

	worktree := filepath.Join(repoDir, filepath.FromSlash(branch))
	if _, err := s.run(ctx, repoDir, "worktree", "add", worktree, branch); err != nil {
		return CloneResult{}, err
	}

	return CloneResult{
		RepoDir:       repoDir,
		BareDir:       bareDir,
		DefaultBranch: branch,
		WorktreePath:  worktree,
	}, nil
}

// defaultBranch returns the branch the repository's HEAD points at — for a
// fresh bare clone this mirrors the remote's default branch.
func (s Service) defaultBranch(ctx context.Context, repoDir string) (string, error) {
	out, err := s.run(ctx, repoDir, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("detecting default branch: %w", err)
	}
	branch := strings.TrimSpace(out)
	if branch == "" {
		return "", fmt.Errorf("detecting default branch: HEAD is not a symbolic ref")
	}
	return branch, nil
}

// ensureEmptyDir returns nil when path does not exist or exists but is an
// empty directory, and an error otherwise.
func ensureEmptyDir(path string) error {
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspecting %q: %w", path, err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("destination %q already exists and is not empty", path)
	}
	return nil
}
