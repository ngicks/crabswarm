package commands

import "github.com/spf13/cobra"

func gitWorktreePruneCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Prune worktree records whose directories have gone away",
		Long: `prune removes the administrative records of worktrees whose directories
were deleted out from under git, for the repository detected from the current
directory. Existing worktrees are left untouched.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE:              runGitWorktreePrune,
	}

	parent.AddCommand(cmd)
}

func runGitWorktreePrune(cmd *cobra.Command, _ []string) error {
	svc, repo, err := gitRepoFromCwd(cmd)
	if err != nil {
		return err
	}

	if err := svc.PruneWorktrees(cmd.Context(), repo); err != nil {
		return err
	}
	cmd.Println("Pruned stale worktree records")
	return nil
}
