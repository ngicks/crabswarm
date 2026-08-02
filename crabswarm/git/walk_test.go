package git

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"gotest.tools/v3/assert"
)

// collectWalk drains an iter.Seq2 walk into the discovered paths and the
// per-entry errors, so a test can assert on both independently.
func collectWalk(
	seq func(yield func(WalkEntry, error) bool),
) (paths []string, errs []error) {
	for entry, err := range seq {
		if err != nil {
			errs = append(errs, err)
			continue
		}
		paths = append(paths, entry.Path)
	}
	return paths, errs
}

// buildWalkFixture lays out a base directory exercising every walker case and
// returns (base, sorted absolute repo roots). It contains:
//
//   - host/owner/bare      a bare-plus-worktree repo via cloneInto
//   - host/owner/plain     a plain `git init` repo
//   - host/owner/nest/deep a repo nested under a non-repo directory
//   - .hidden/repo         a repo under a dot-dir that must be skipped
func buildWalkFixture(t *testing.T) (string, []string) {
	t.Helper()
	requireGit(t)
	base := t.TempDir()
	origin := newOrigin(t, "main", "feature")

	// (a) bare-plus-worktree layout — its <branch> worktree subdirs each hold
	// their own .git file and must NOT be reported as separate repos.
	bareRepo := filepath.Join(base, "host", "owner", "bare")
	svc := Service{}
	_, err := svc.cloneInto(context.Background(), origin, bareRepo)
	assert.NilError(t, err)

	// (b) a plain clone (.git is a directory).
	plainRepo := filepath.Join(base, "host", "owner", "plain")
	assert.NilError(t, os.MkdirAll(plainRepo, 0o755))
	gitExec(t, plainRepo, "init", "-q")

	// (c) a repo nested under an ordinary non-repo directory.
	deepRepo := filepath.Join(base, "host", "owner", "nest", "deep")
	assert.NilError(t, os.MkdirAll(deepRepo, 0o755))
	gitExec(t, deepRepo, "init", "-q")

	// (d) a repo under a dot-dir, which descent must skip entirely.
	hiddenRepo := filepath.Join(base, ".hidden", "repo")
	assert.NilError(t, os.MkdirAll(hiddenRepo, 0o755))
	gitExec(t, hiddenRepo, "init", "-q")

	want := []string{bareRepo, plainRepo, deepRepo}
	slices.Sort(want)
	return base, want
}

func TestService_WalkRepos(t *testing.T) {
	base, want := buildWalkFixture(t)

	paths, errs := collectWalk(Service{}.WalkRepos(context.Background(), base))
	assert.Assert(t, len(errs) == 0, "unexpected walk errors: %v", errs)

	slices.Sort(paths)
	assert.DeepEqual(t, paths, want)

	// The bare repo's per-branch worktree subdirectory (host/owner/bare/main)
	// must not appear as its own repo root.
	wtSub := filepath.Join(base, "host", "owner", "bare", "main")
	assert.Assert(t, !slices.Contains(paths, wtSub),
		"worktree subdir reported as a repo: %s", wtSub)

	// The dot-dir repo must have been skipped.
	hidden := filepath.Join(base, ".hidden", "repo")
	assert.Assert(t, !slices.Contains(paths, hidden),
		"repo under dot-dir was not skipped: %s", hidden)
}

func TestService_WalkWorktrees(t *testing.T) {
	base, _ := buildWalkFixture(t)

	paths, errs := collectWalk(Service{}.WalkWorktrees(context.Background(), base))
	assert.Assert(t, len(errs) == 0, "unexpected walk errors: %v", errs)

	// The bare repo has a checked-out "main" worktree; it must be listed,
	// while the bare entry (the repo dir itself) must be excluded.
	bareRepo := filepath.Join(base, "host", "owner", "bare")
	mainWt := filepath.Join(bareRepo, "main")
	assert.Assert(t, slices.Contains(paths, mainWt),
		"expected worktree %s in %v", mainWt, paths)
	assert.Assert(t, !slices.Contains(paths, bareRepo),
		"bare entry must be excluded from worktree listing: %s", bareRepo)

	// The plain repo is itself a worktree, so its own directory is listed.
	plainRepo := filepath.Join(base, "host", "owner", "plain")
	assert.Assert(t, slices.Contains(paths, plainRepo),
		"expected plain repo %s as its own worktree in %v", plainRepo, paths)
}

func TestService_WalkWorktrees_BrokenRepoContinues(t *testing.T) {
	base, _ := buildWalkFixture(t)

	// A directory whose ".git" pointer leads nowhere looks like a repo root
	// to the walker but cannot be resolved; it must surface one error and
	// not abort the rest of the listing.
	broken := filepath.Join(base, "host", "owner", "broken")
	assert.NilError(t, os.MkdirAll(broken, 0o755))
	assert.NilError(t, os.WriteFile(
		filepath.Join(broken, ".git"), []byte("gitdir: /nonexistent\n"), 0o644,
	))

	paths, errs := collectWalk(Service{}.WalkWorktrees(context.Background(), base))
	assert.Assert(t, len(errs) == 1, "expected one error for the broken repo, got %v", errs)

	// Healthy repos before and after the broken one are still listed.
	mainWt := filepath.Join(base, "host", "owner", "bare", "main")
	plainRepo := filepath.Join(base, "host", "owner", "plain")
	assert.Assert(t, slices.Contains(paths, mainWt))
	assert.Assert(t, slices.Contains(paths, plainRepo))
}

func TestService_WalkRepos_IgnorePattern(t *testing.T) {
	base, _ := buildWalkFixture(t)

	// An ignore pattern matching the "owner" directory base name must skip the
	// whole subtree beneath it without descending, so none of its repos appear.
	svc := Service{IgnorePatterns: []string{"owner"}}
	paths, errs := collectWalk(svc.WalkRepos(context.Background(), base))
	assert.Assert(t, len(errs) == 0, "unexpected walk errors: %v", errs)
	assert.Assert(
		t,
		len(paths) == 0,
		"expected the ignored subtree to yield no repos, got %v",
		paths,
	)

	// A glob pattern matches by base name too; "pl*" hits only host/owner/plain.
	svc = Service{IgnorePatterns: []string{"pl*"}}
	paths, errs = collectWalk(svc.WalkRepos(context.Background(), base))
	assert.Assert(t, len(errs) == 0, "unexpected walk errors: %v", errs)
	plainRepo := filepath.Join(base, "host", "owner", "plain")
	assert.Assert(t, !slices.Contains(paths, plainRepo),
		"glob-ignored repo was still listed: %s", plainRepo)
	// The unmatched repos survive.
	bareRepo := filepath.Join(base, "host", "owner", "bare")
	deepRepo := filepath.Join(base, "host", "owner", "nest", "deep")
	assert.Assert(t, slices.Contains(paths, bareRepo))
	assert.Assert(t, slices.Contains(paths, deepRepo))
}

// A repo root whose own base name matches a pattern is itself skipped, not just
// its children — the match is applied before the repo-root check.
func TestService_WalkRepos_IgnoreMatchesRepoRoot(t *testing.T) {
	base, _ := buildWalkFixture(t)

	svc := Service{IgnorePatterns: []string{"plain"}}
	paths, errs := collectWalk(svc.WalkRepos(context.Background(), base))
	assert.Assert(t, len(errs) == 0, "unexpected walk errors: %v", errs)
	plainRepo := filepath.Join(base, "host", "owner", "plain")
	assert.Assert(t, !slices.Contains(paths, plainRepo),
		"ignored repo root was still listed: %s", plainRepo)
	// A sibling repo is unaffected.
	bareRepo := filepath.Join(base, "host", "owner", "bare")
	assert.Assert(t, slices.Contains(paths, bareRepo))
}

// The base directory itself is never ignored, even when its own name matches a
// pattern — otherwise a misconfigured pattern would silently list nothing.
func TestService_WalkRepos_IgnoreNeverSkipsRoot(t *testing.T) {
	base, want := buildWalkFixture(t)

	svc := Service{IgnorePatterns: []string{filepath.Base(base)}}
	paths, errs := collectWalk(svc.WalkRepos(context.Background(), base))
	assert.Assert(t, len(errs) == 0, "unexpected walk errors: %v", errs)
	slices.Sort(paths)
	assert.DeepEqual(t, paths, want)
}

// The worktree walker inherits IgnorePatterns through its WalkRepos pass: an
// ignored repo contributes no worktrees.
func TestService_WalkWorktrees_IgnorePattern(t *testing.T) {
	base, _ := buildWalkFixture(t)

	svc := Service{IgnorePatterns: []string{"bare"}}
	paths, errs := collectWalk(svc.WalkWorktrees(context.Background(), base))
	assert.Assert(t, len(errs) == 0, "unexpected walk errors: %v", errs)
	mainWt := filepath.Join(base, "host", "owner", "bare", "main")
	assert.Assert(t, !slices.Contains(paths, mainWt),
		"worktree of an ignored repo was still listed: %s", mainWt)
	// The plain repo (its own worktree) survives.
	plainRepo := filepath.Join(base, "host", "owner", "plain")
	assert.Assert(t, slices.Contains(paths, plainRepo))
}

// A malformed glob fails fast with a single error before any directory is
// walked, rather than surfacing per-directory.
func TestService_WalkRepos_BadIgnorePattern(t *testing.T) {
	base, _ := buildWalkFixture(t)

	svc := Service{IgnorePatterns: []string{"["}}
	paths, errs := collectWalk(svc.WalkRepos(context.Background(), base))
	assert.Assert(t, len(paths) == 0, "expected no paths on bad pattern, got %v", paths)
	assert.Assert(t, len(errs) == 1, "expected a single error, got %v", errs)
	assert.ErrorContains(t, errs[0], "invalid ignore pattern")
}

func TestService_WalkRepos_StopsEarly(t *testing.T) {
	base, want := buildWalkFixture(t)
	assert.Assert(t, len(want) > 1)

	// Breaking out of the range loop must stop the walk.
	var seen int
	svc := Service{}
	for range svc.WalkRepos(context.Background(), base) {
		seen++
		break
	}
	assert.Equal(t, seen, 1)
}

func TestService_WalkRepos_MissingBaseDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	paths, errs := collectWalk(Service{}.WalkRepos(context.Background(), missing))
	assert.Assert(t, len(paths) == 0, "expected no paths, got %v", paths)
	assert.Assert(t, len(errs) == 1, "expected a single error, got %v", errs)
	assert.ErrorContains(t, errs[0], "walking")
}
