package commands

import (
	"log/slog"

	"github.com/ngicks/go-common/contextkey"
	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/crabswarm"
	"github.com/ngicks/crabswarm/crabswarm/git"
)

func gitCloneCmd(parent *cobra.Command, flagConfig *string) {
	var flagBaseDir string

	cmd := &cobra.Command{
		Use:   "clone <url>",
		Short: "Bare-clone a repository and check out its default branch as a worktree",
		Long: `clone places a repository under the base directory using a ghq-style
path derived from its URL, clones it in --bare mode, and checks out the
remote's default branch as a worktree.

The resulting layout for, e.g., https://github.com/owner/repo is:

  <base>/github.com/owner/repo/
    .bare/   the bare clone
    .git     file pointing at .bare
    main/    worktree of the default branch

The base directory is taken from --base-dir, else
$CRABSWARM_GIT_REPO_BASE_DIR, else $HOME/gitrepo. Accepted URL forms include
https://, ssh://, scp-like git@host:owner/repo, and scheme-less
host/owner/repo (with or without a trailing .git).`,
		Example: `  crabswarm git clone https://github.com/ngicks/crabswarm
  crabswarm git clone git@github.com:ngicks/crabswarm.git`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGitClone(cmd, args, flagBaseDir, *flagConfig)
		},
	}

	cmd.Flags().StringVar(&flagBaseDir, "base-dir", "",
		"Base directory to clone under (default $CRABSWARM_GIT_REPO_BASE_DIR or $HOME/gitrepo)")

	parent.AddCommand(cmd)
}

func runGitClone(cmd *cobra.Command, args []string, baseDir, flagConfig string) error {
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

	svc := git.Service{
		BaseDir:  cfg.GitRepoBaseDir,
		Logger:   logger,
		Progress: cmd.ErrOrStderr(),
	}
	res, err := svc.Clone(ctx, args[0])
	if err != nil {
		return err
	}

	cmd.Printf("Cloned %s\n  repo:     %s\n  branch:   %s\n  worktree: %s\n",
		args[0], res.RepoDir, res.DefaultBranch, res.WorktreePath)
	return nil
}
