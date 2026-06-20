package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

// cloneBareFixture clones the bare-plus-worktree layout into a fresh temp dir
// and returns the repository directory (which holds ".bare") and the
// default-branch worktree path.
func cloneBareFixture(t *testing.T) (repoDir, worktree string) {
	t.Helper()
	requireGit(t)
	origin := newOrigin(t, "main")

	repoDir = filepath.Join(t.TempDir(), "host", "owner", "repo")
	res, err := Service{}.cloneInto(context.Background(), origin, repoDir)
	assert.NilError(t, err)
	return res.RepoDir, res.WorktreePath
}

func TestService_FindRoot_ResolvesBareRoot(t *testing.T) {
	repoDir, worktree := cloneBareFixture(t)

	// A nested subdirectory inside the worktree exercises a multi-step climb.
	nested := filepath.Join(worktree, "pkg", "deep")
	assert.NilError(t, os.MkdirAll(nested, 0o755))

	cases := map[string]string{
		"repo dir itself":     repoDir,
		"worktree dir":        worktree,
		"nested in worktree":  nested,
		"the .bare directory": filepath.Join(repoDir, bareDirName),
	}
	for name, start := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := Service{}.FindRoot(context.Background(), start)
			assert.NilError(t, err)
			assert.Equal(t, got, repoDir)
		})
	}
}

// TestService_FindRoot_ExternalWorktree covers a worktree checked out outside
// its repository directory: climbing the filesystem never reaches ".bare", so
// resolution must fall back to git, which follows the worktree link.
func TestService_FindRoot_ExternalWorktree(t *testing.T) {
	repoDir, _ := cloneBareFixture(t)

	external := filepath.Join(t.TempDir(), "external-wt")
	gitExec(t, repoDir, "worktree", "add", "--relative-paths", "-b", "feature", external)

	sub := filepath.Join(external, "nested")
	assert.NilError(t, os.MkdirAll(sub, 0o755))

	for _, start := range []string{external, sub} {
		got, err := Service{}.FindRoot(context.Background(), start)
		assert.NilError(t, err)
		assert.Equal(t, got, repoDir)
	}
}

func TestService_FindRoot_OrdinaryCloneFails(t *testing.T) {
	requireGit(t)

	plain := filepath.Join(t.TempDir(), "plain")
	assert.NilError(t, os.MkdirAll(plain, 0o755))
	gitExec(t, plain, "init", "-q")

	sub := filepath.Join(plain, "sub")
	assert.NilError(t, os.MkdirAll(sub, 0o755))

	for _, start := range []string{plain, sub} {
		got, err := Service{}.FindRoot(context.Background(), start)
		assert.Equal(t, got, "")

		notBare, ok := errors.AsType[*NotBareCloneError](err)
		assert.Assert(t, ok, "want *NotBareCloneError, got %v", err)
		// The reported directory is the ordinary clone, not the subdirectory.
		assert.Equal(t, notBare.Dir, plain)
	}
}

// TestService_FindRoot_ExternalWorktreeOfOrdinaryClone covers the git-fallback
// path landing on an ordinary clone: git resolves the common dir to ".git"
// (not ".bare"), so it is reported as not a --bare clone.
func TestService_FindRoot_ExternalWorktreeOfOrdinaryClone(t *testing.T) {
	requireGit(t)

	plain := filepath.Join(t.TempDir(), "plain")
	assert.NilError(t, os.MkdirAll(plain, 0o755))
	gitExec(t, plain, "init", "-q", "-b", "main")
	gitExec(t, plain, "commit", "-q", "--allow-empty", "-m", "init")

	external := filepath.Join(t.TempDir(), "external-wt")
	gitExec(t, plain, "worktree", "add", "-b", "feature", external)

	got, err := Service{}.FindRoot(context.Background(), external)
	assert.Equal(t, got, "")

	notBare, ok := errors.AsType[*NotBareCloneError](err)
	assert.Assert(t, ok, "want *NotBareCloneError, got %v", err)
	assert.Equal(t, notBare.Dir, plain)
}

func TestService_FindRoot_NoRepo(t *testing.T) {
	// A bare temp directory with no git repository anywhere up to the root,
	// where neither the walk nor the git fallback can resolve anything.
	dir := t.TempDir()

	got, err := Service{}.FindRoot(context.Background(), dir)
	assert.Equal(t, got, "")
	assert.Assert(t, errors.Is(err, ErrNoGitRoot), "want ErrNoGitRoot, got %v", err)
}

func TestService_FindRoot_DefaultsToCwd(t *testing.T) {
	repoDir, worktree := cloneBareFixture(t)

	t.Chdir(worktree)

	got, err := Service{}.FindRoot(context.Background(), "")
	assert.NilError(t, err)
	assert.Equal(t, got, repoDir)
}
