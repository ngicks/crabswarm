package commands

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/ngicks/go-common/contextkey"
	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/crabswarm"
	"github.com/ngicks/crabswarm/crabswarm/git"
)

func gitListCmd(parent *cobra.Command, flagConfig *string) {
	var (
		flagBaseDir  string
		flagWorktree bool
		flagFullPath bool
		flagIgnore   []string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List git repositories (or their worktrees) under the base directory",
		Long: `list walks the base directory and prints one repository root per line as
it is found, so the output streams straight into a fuzzy finder. A repository
root is any directory holding a ".git" entry — both crabswarm's bare-plus-
worktree layout and ordinary clones (ghq, plain git) are discovered, and a
repository's per-branch worktree subdirectories are not reported as separate
repositories. Dot-directories are skipped and symlinks are not followed.

A directory whose base name matches a --ignore-pattern glob (filepath.Match
syntax) is skipped and not descended into, so neither it nor anything beneath
it is listed. The flag is repeatable; when set it overrides the
git_list_ignore_patterns config entry, otherwise that config (if any) applies.

With --worktree the worktree directories of each repository are listed
instead (the bare entry excluded). Paths are printed base-relative
(ghq-style host/owner/repo) unless --full-path is given. A repository that
cannot be read is reported to stderr and the walk continues.

The base directory is taken from --base-dir, else $CRABSWARM_GIT_REPO_BASE_DIR,
else $HOME/gitrepo.`,
		Example: `  crabswarm git list
  crabswarm git list --worktree
  crabswarm git list --ignore-pattern node_modules --ignore-pattern 'tmp*'
  cd "$(crabswarm git list --full-path | fzf)"`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGitList(
				cmd,
				flagBaseDir,
				flagWorktree,
				flagFullPath,
				flagIgnore,
				*flagConfig,
			)
		},
	}

	cmd.Flags().StringVar(&flagBaseDir, "base-dir", "",
		"Base directory to walk (default $CRABSWARM_GIT_REPO_BASE_DIR or $HOME/gitrepo)")
	cmd.Flags().BoolVarP(&flagWorktree, "worktree", "w", false,
		"List worktree directories instead of repository roots (bare entry excluded)")
	cmd.Flags().BoolVar(&flagFullPath, "full-path", false,
		"Print absolute paths instead of base-relative (ghq-style) paths")
	cmd.Flags().StringArrayVar(&flagIgnore, "ignore-pattern", nil,
		"Glob pattern (filepath.Match) matched against directory names to skip without descending; "+
			"repeatable, overrides git_list_ignore_patterns config when set")

	parent.AddCommand(cmd)
}

func runGitList(
	cmd *cobra.Command,
	baseDir string,
	worktree, fullPath bool,
	ignorePatterns []string,
	flagConfig string,
) error {
	ctx := cmd.Context()

	logger, _ := contextkey.ValueSlogLogger(ctx)
	if logger == nil {
		logger = slog.Default()
	}

	cfg, err := crabswarm.LoadConfig(flagConfig)
	if err != nil {
		return err
	}
	if cmd.Flags().Changed("base-dir") {
		cfg.GitRepoBaseDir = baseDir
	}
	if cmd.Flags().Changed("ignore-pattern") {
		cfg.GitListIgnorePatterns = ignorePatterns
	}
	// A malformed glob is a usage error: reject it up front with a non-zero
	// exit instead of letting the walk surface it as a per-entry warning.
	if err := git.ValidateIgnorePatterns(cfg.GitListIgnorePatterns); err != nil {
		return err
	}

	svc := git.Service{
		BaseDir:        cfg.GitRepoBaseDir,
		IgnorePatterns: cfg.GitListIgnorePatterns,
		Logger:         logger,
	}

	seq := svc.WalkRepos
	if worktree {
		seq = svc.WalkWorktrees
	}

	out := cmd.OutOrStdout()
	for entry, err := range seq(ctx, cfg.GitRepoBaseDir) {
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v\n", err)
			continue
		}
		fmt.Fprintln(out, displayPath(cfg.GitRepoBaseDir, entry.Path, fullPath))
	}
	return nil
}

// displayPath renders an entry path for listing output: the absolute path when
// fullPath, otherwise the path relative to base (ghq-style host/owner/repo for
// repo roots, repo/<branch> for worktree directories). When the path cannot be
// made relative — e.g. it escaped base via a symlink — the absolute path is
// returned so nothing is silently dropped.
func displayPath(base, path string, fullPath bool) string {
	if fullPath {
		return path
	}
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return rel
}
