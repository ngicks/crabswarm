package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"gotest.tools/v3/assert"
)

// requireGit skips the test when no git binary is available.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

// gitExec runs git in dir with a fixed committer identity, failing the test
// on error. Used only to set up fixtures; the service under test shells out
// to plain git itself.
func gitExec(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
	)
	out, err := cmd.CombinedOutput()
	assert.NilError(t, err, "git %v: %s", args, out)
	return string(out)
}

// newOrigin creates a local source repository with a single commit on
// defaultBranch plus any extraBranches, and returns its path. It is used as
// the clone source so tests need no network.
func newOrigin(t *testing.T, defaultBranch string, extraBranches ...string) string {
	t.Helper()
	dir := t.TempDir()
	gitExec(t, dir, "init", "-q", "-b", defaultBranch)
	gitExec(t, dir, "commit", "-q", "--allow-empty", "-m", "init")
	for _, b := range extraBranches {
		gitExec(t, dir, "branch", b)
	}
	return dir
}

func TestService_cloneInto(t *testing.T) {
	requireGit(t)
	origin := newOrigin(t, "trunk", "feature")

	repoDir := filepath.Join(t.TempDir(), "host", "owner", "repo")
	svc := Service{}
	res, err := svc.cloneInto(context.Background(), origin, repoDir)
	assert.NilError(t, err)

	assert.Equal(t, res.RepoDir, repoDir)
	assert.Equal(t, res.BareDir, filepath.Join(repoDir, ".bare"))
	assert.Equal(t, res.DefaultBranch, "trunk")
	assert.Equal(t, res.WorktreePath, filepath.Join(repoDir, "trunk"))

	// .bare exists and the .git pointer file is present.
	assertDir(t, res.BareDir)
	gitFile, err := os.ReadFile(filepath.Join(repoDir, ".git"))
	assert.NilError(t, err)
	assert.Equal(t, string(gitFile), "gitdir: ./.bare\n")

	// The worktree is checked out on the default branch.
	assertDir(t, res.WorktreePath)
	repo, err := svc.ResolveRepo(context.Background(), res.WorktreePath)
	assert.NilError(t, err)
	branches, err := svc.Branches(context.Background(), repo)
	assert.NilError(t, err)
	assert.Assert(t, slices.Contains(branches, "trunk"))
	assert.Assert(t, slices.Contains(branches, "feature"))
	// Remote-tracking refs were populated by the post-clone fetch.
	assert.Assert(t, slices.Contains(branches, "origin/trunk"))
	// The origin/HEAD symbolic pointer is filtered out — git shortens it
	// to "origin", which must not appear as a branch.
	assert.Assert(t, !slices.Contains(branches, "origin"))
	assert.Assert(t, !slices.Contains(branches, "origin/HEAD"))
}

func TestService_cloneInto_RefusesNonEmpty(t *testing.T) {
	requireGit(t)
	origin := newOrigin(t, "main")

	repoDir := filepath.Join(t.TempDir(), "repo")
	assert.NilError(t, os.MkdirAll(repoDir, 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(repoDir, "stray"), []byte("x"), 0o644))

	_, err := Service{}.cloneInto(context.Background(), origin, repoDir)
	assert.Assert(t, err != nil)
}

func TestService_Worktree_AddListRemove(t *testing.T) {
	requireGit(t)
	origin := newOrigin(t, "main", "existing")

	repoDir := filepath.Join(t.TempDir(), "repo")
	svc := Service{}
	_, err := svc.cloneInto(context.Background(), origin, repoDir)
	assert.NilError(t, err)

	ctx := context.Background()
	repo, err := svc.ResolveRepo(ctx, repoDir)
	assert.NilError(t, err)
	assert.Equal(t, repo.Dir, repoDir)

	// Existing local branch → worktree at <repo>/existing.
	p, err := svc.AddWorktree(ctx, repo, "existing", "")
	assert.NilError(t, err)
	assert.Equal(t, p, filepath.Join(repoDir, "existing"))
	assertDir(t, p)

	// Brand-new branch created from HEAD.
	p2, err := svc.AddWorktree(ctx, repo, "fresh", "")
	assert.NilError(t, err)
	assertDir(t, p2)

	wts, err := svc.ListWorktrees(ctx, repo)
	assert.NilError(t, err)
	assert.Assert(t, hasWorktree(wts, "existing"))
	assert.Assert(t, hasWorktree(wts, "fresh"))

	// Remove by handle (directory name).
	assert.NilError(t, svc.RemoveWorktree(ctx, repo, "fresh", false))
	wts, err = svc.ListWorktrees(ctx, repo)
	assert.NilError(t, err)
	assert.Assert(t, !hasWorktree(wts, "fresh"))

	// Unknown handle errors.
	assert.Assert(t, svc.RemoveWorktree(ctx, repo, "nope", false) != nil)
}

func TestService_AddWorktree_TracksRemoteBranch(t *testing.T) {
	requireGit(t)
	origin := newOrigin(t, "main")

	repoDir := filepath.Join(t.TempDir(), "repo")
	svc := Service{}
	_, err := svc.cloneInto(context.Background(), origin, repoDir)
	assert.NilError(t, err)

	// Create the branch on origin AFTER cloning so it exists only under
	// refs/remotes/origin/* after a fetch (a --bare clone mirrors heads
	// once, but later fetches only update remote-tracking refs).
	gitExec(t, origin, "branch", "release")
	gitExec(t, repoDir, "fetch", "origin")

	ctx := context.Background()
	repo, err := svc.ResolveRepo(ctx, repoDir)
	assert.NilError(t, err)
	assert.Assert(t, !svc.refExists(ctx, repo, "refs/heads/release"))
	assert.Assert(t, svc.refExists(ctx, repo, "refs/remotes/origin/release"))

	// "origin/release" → new local tracking branch "release".
	p, err := svc.AddWorktree(ctx, repo, "origin/release", "")
	assert.NilError(t, err)
	assert.Equal(t, p, filepath.Join(repoDir, "origin", "release"))

	wts, err := svc.ListWorktrees(ctx, repo)
	assert.NilError(t, err)
	found := slices.ContainsFunc(wts, func(w Worktree) bool { return w.Branch == "release" })
	assert.Assert(t, found, "expected a worktree on local branch release")
}

func assertDir(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	assert.NilError(t, err)
	assert.Assert(t, info.IsDir(), "%q is not a directory", path)
}

func hasWorktree(wts []Worktree, name string) bool {
	return slices.ContainsFunc(wts, func(w Worktree) bool { return w.Name() == name })
}
