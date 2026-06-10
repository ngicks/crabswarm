package git

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

// Repo identifies a resolved repository for worktree operations.
type Repo struct {
	// CommonDir is the shared git directory (the bare ".bare" dir or a
	// plain ".git" dir), as an absolute path.
	CommonDir string
	// Dir is the directory under which worktrees are placed as siblings —
	// the parent of CommonDir. For the bare layout this is the repository
	// directory created by Clone.
	Dir string
}

// Worktree is one entry of `git worktree list`.
type Worktree struct {
	// Path is the absolute worktree directory.
	Path string
	// Branch is the short branch name checked out there, empty when the
	// worktree is detached or bare.
	Branch string
	// Bare reports whether this entry is the bare/main repository itself.
	Bare bool
}

// Name is the worktree's directory name, used as its handle in the CLI.
func (w Worktree) Name() string {
	return filepath.Base(w.Path)
}

// ResolveRepo locates the repository that owns startDir (cwd when empty) by
// asking git for the common git directory. It works from inside any worktree
// or from the repository directory itself.
func (s Service) ResolveRepo(ctx context.Context, startDir string) (Repo, error) {
	out, err := s.run(ctx, startDir,
		"rev-parse", "--path-format=absolute", "--git-common-dir",
	)
	if err != nil {
		return Repo{}, fmt.Errorf("not inside a git repository: %w", err)
	}
	common := strings.TrimSpace(out)
	if common == "" {
		return Repo{}, fmt.Errorf("could not resolve git common directory")
	}
	return Repo{CommonDir: common, Dir: filepath.Dir(common)}, nil
}

// Branches returns the short names of local and remote-tracking branches,
// sorted and de-duplicated, suitable for shell completion. Symbolic refs
// such as the "origin/HEAD" pointer are omitted (their %(symref) is set).
func (s Service) Branches(ctx context.Context, repo Repo) ([]string, error) {
	// %(symref) is non-empty only for symbolic refs (e.g. origin/HEAD),
	// which we skip; a NUL separator keeps it unambiguous from the name.
	out, err := s.run(ctx, repo.Dir,
		"for-each-ref", "--format=%(refname:short)%00%(symref)", "refs/heads", "refs/remotes",
	)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var names []string
	for line := range strings.SplitSeq(out, "\n") {
		name, symref, _ := strings.Cut(line, "\x00")
		name = strings.TrimSpace(name)
		if name == "" || symref != "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	slices.Sort(names)
	return names, nil
}

// AddWorktree creates a worktree for branch under repo.Dir. When path is
// empty it defaults to <repo.Dir>/<branch>. Branch resolution:
//
//   - an existing local branch is checked out as-is;
//   - a remote-only "<remote>/<branch>" is checked out as a new local
//     tracking branch;
//   - otherwise a new branch is created from the current HEAD.
//
// It returns the absolute worktree path.
func (s Service) AddWorktree(ctx context.Context, repo Repo, branch, path string) (string, error) {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return "", fmt.Errorf("branch name is required")
	}
	if path == "" {
		path = filepath.Join(repo.Dir, filepath.FromSlash(branch))
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(repo.Dir, path)
	}

	var args []string
	switch {
	case s.refExists(ctx, repo, "refs/heads/"+branch):
		args = []string{"worktree", "add", path, branch}
	case s.refExists(ctx, repo, "refs/remotes/"+branch):
		local := branch[strings.IndexByte(branch, '/')+1:]
		args = []string{"worktree", "add", "--track", "-b", local, path, branch}
	default:
		args = []string{"worktree", "add", "-b", branch, path}
	}

	if _, err := s.run(ctx, repo.Dir, args...); err != nil {
		return "", err
	}
	return path, nil
}

// ListWorktrees returns the repository's worktrees as reported by
// `git worktree list --porcelain`.
func (s Service) ListWorktrees(ctx context.Context, repo Repo) ([]Worktree, error) {
	out, err := s.run(ctx, repo.Dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktreeList(out), nil
}

// RemoveWorktree removes the worktree identified by nameOrPath — either a
// worktree directory name (its handle) or a path. When force is true a
// worktree with local changes is removed anyway.
func (s Service) RemoveWorktree(
	ctx context.Context,
	repo Repo,
	nameOrPath string,
	force bool,
) error {
	target, err := s.resolveWorktree(ctx, repo, nameOrPath)
	if err != nil {
		return err
	}
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, target)
	_, err = s.run(ctx, repo.Dir, args...)
	return err
}

// resolveWorktree maps a worktree handle (directory name) or path to the
// canonical worktree path git knows about.
func (s Service) resolveWorktree(
	ctx context.Context,
	repo Repo,
	nameOrPath string,
) (string, error) {
	wts, err := s.ListWorktrees(ctx, repo)
	if err != nil {
		return "", err
	}
	want := nameOrPath
	if abs, err := filepath.Abs(nameOrPath); err == nil {
		want = abs
	}
	for _, wt := range wts {
		if wt.Path == want || wt.Name() == nameOrPath {
			return wt.Path, nil
		}
	}
	return "", fmt.Errorf("no worktree matching %q", nameOrPath)
}

// refExists reports whether ref resolves in the repository.
func (s Service) refExists(ctx context.Context, repo Repo, ref string) bool {
	_, err := s.run(ctx, repo.Dir, "show-ref", "--verify", "--quiet", ref)
	return err == nil
}

// parseWorktreeList parses the output of `git worktree list --porcelain`.
// Records are separated by blank lines; each starts with "worktree <path>"
// and may carry "branch refs/heads/<name>", "detached", or "bare".
func parseWorktreeList(out string) []Worktree {
	var (
		result []Worktree
		cur    Worktree
		active bool
	)
	flush := func() {
		if active {
			result = append(result, cur)
		}
		cur, active = Worktree{}, false
	}
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "worktree "):
			flush()
			cur.Path = strings.TrimPrefix(line, "worktree ")
			active = true
		case line == "bare":
			cur.Bare = true
		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		}
	}
	flush()
	return result
}
