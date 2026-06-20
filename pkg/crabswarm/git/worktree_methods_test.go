package git

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"gotest.tools/v3/assert"
)

// setupRepo clones the local origin into a fresh repo directory and returns
// the service and resolved Repo, ready for worktree operations.
func setupRepo(t *testing.T, defaultBranch string, extraBranches ...string) (Service, Repo) {
	t.Helper()
	origin := newOrigin(t, defaultBranch, extraBranches...)
	repoDir := filepath.Join(t.TempDir(), "repo")
	svc := Service{}
	_, err := svc.cloneInto(context.Background(), origin, repoDir)
	assert.NilError(t, err)
	repo, err := svc.ResolveRepo(context.Background(), repoDir)
	assert.NilError(t, err)
	return svc, repo
}

func TestService_MoveWorktree(t *testing.T) {
	requireGit(t)
	svc, repo := setupRepo(t, "main", "existing")
	ctx := context.Background()

	src, err := svc.AddWorktree(ctx, repo, "existing", "")
	assert.NilError(t, err)
	assertDir(t, src)

	// A relative dest resolves against repo.Dir, keeping the worktree a sibling.
	moved, err := svc.MoveWorktree(ctx, repo, "existing", "renamed")
	assert.NilError(t, err)
	assert.Equal(t, moved, filepath.Join(repo.Dir, "renamed"))

	// The new path is a real directory and git knows it as a worktree; the old
	// one is gone.
	assertDir(t, moved)
	assert.Assert(t, !dirExists(filepath.Join(repo.Dir, "existing")))
	// A plain `git worktree move` rewrites the links as absolute paths; the
	// --relative-paths flag keeps them relative across the move.
	assertRelativeWorktreeLinks(t, moved)

	wts, err := svc.ListWorktrees(ctx, repo)
	assert.NilError(t, err)
	assert.Assert(t, hasWorktree(wts, "renamed"))
	assert.Assert(t, !hasWorktree(wts, "existing"))

	// The moved worktree resolves by its new handle and operations work on it
	// (resolveWorktree found it via the refreshed list).
	resolved, err := svc.resolveWorktree(ctx, repo, "renamed")
	assert.NilError(t, err)
	assert.Equal(t, resolved, moved)

	// An empty dest is rejected, and an unknown source errors.
	_, err = svc.MoveWorktree(ctx, repo, "renamed", "")
	assert.Assert(t, err != nil)
	_, err = svc.MoveWorktree(ctx, repo, "nope", "wherever")
	assert.Assert(t, err != nil)
}

func TestService_LockUnlockWorktree(t *testing.T) {
	requireGit(t)
	svc, repo := setupRepo(t, "main", "existing")
	ctx := context.Background()

	_, err := svc.AddWorktree(ctx, repo, "existing", "")
	assert.NilError(t, err)

	// Lock with a reason; the porcelain list reflects it.
	assert.NilError(t, svc.LockWorktree(ctx, repo, "existing", "on a USB drive"))
	wts, err := svc.ListWorktrees(ctx, repo)
	assert.NilError(t, err)
	wt := findWorktree(wts, "existing")
	assert.Assert(t, wt != nil)
	assert.Assert(t, wt.Locked)
	assert.Equal(t, wt.LockReason, "on a USB drive")

	// A non-forced remove refuses a locked worktree.
	assert.Assert(t, svc.RemoveWorktree(ctx, repo, "existing", false) != nil)

	// After unlock the worktree is no longer locked and removes cleanly.
	assert.NilError(t, svc.UnlockWorktree(ctx, repo, "existing"))
	wts, err = svc.ListWorktrees(ctx, repo)
	assert.NilError(t, err)
	wt = findWorktree(wts, "existing")
	assert.Assert(t, wt != nil)
	assert.Assert(t, !wt.Locked)

	assert.NilError(t, svc.RemoveWorktree(ctx, repo, "existing", false))

	// Unknown handle errors for both lock and unlock.
	assert.Assert(t, svc.LockWorktree(ctx, repo, "nope", "") != nil)
	assert.Assert(t, svc.UnlockWorktree(ctx, repo, "nope") != nil)
}

func TestService_LockWorktree_NoReason(t *testing.T) {
	requireGit(t)
	svc, repo := setupRepo(t, "main", "existing")
	ctx := context.Background()

	_, err := svc.AddWorktree(ctx, repo, "existing", "")
	assert.NilError(t, err)

	// An empty reason locks without recording a reason.
	assert.NilError(t, svc.LockWorktree(ctx, repo, "existing", ""))
	wts, err := svc.ListWorktrees(ctx, repo)
	assert.NilError(t, err)
	wt := findWorktree(wts, "existing")
	assert.Assert(t, wt != nil)
	assert.Assert(t, wt.Locked)
	assert.Equal(t, wt.LockReason, "")

	assert.NilError(t, svc.UnlockWorktree(ctx, repo, "existing"))
}

func TestService_PruneWorktrees(t *testing.T) {
	requireGit(t)
	svc, repo := setupRepo(t, "main", "existing")
	ctx := context.Background()

	p, err := svc.AddWorktree(ctx, repo, "existing", "")
	assert.NilError(t, err)

	// Delete the worktree directory out from under git: its record goes stale.
	assert.NilError(t, os.RemoveAll(p))

	assert.NilError(t, svc.PruneWorktrees(ctx, repo))

	// Pruning dropped the stale record.
	wts, err := svc.ListWorktrees(ctx, repo)
	assert.NilError(t, err)
	assert.Assert(t, !hasWorktree(wts, "existing"))
}

func TestService_RepairWorktrees(t *testing.T) {
	requireGit(t)
	svc, repo := setupRepo(t, "main", "existing")
	ctx := context.Background()

	p, err := svc.AddWorktree(ctx, repo, "existing", "")
	assert.NilError(t, err)

	// Move the worktree directory by hand (not via git), breaking the link.
	moved := filepath.Join(repo.Dir, "existing-moved")
	assert.NilError(t, os.Rename(p, moved))

	// repair with the new path restores the administrative link so git tracks
	// the worktree at its new location.
	assert.NilError(t, svc.RepairWorktrees(ctx, repo, moved))

	wts, err := svc.ListWorktrees(ctx, repo)
	assert.NilError(t, err)
	found := slices.ContainsFunc(wts, func(w Worktree) bool { return w.Path == moved })
	assert.Assert(t, found, "expected a worktree at the repaired path %q", moved)

	// A no-argument repair is a harmless no-op on a healthy repository.
	assert.NilError(t, svc.RepairWorktrees(ctx, repo))
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func findWorktree(wts []Worktree, name string) *Worktree {
	for i := range wts {
		if wts[i].Name() == name {
			return &wts[i]
		}
	}
	return nil
}
