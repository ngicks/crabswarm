package commands

import "github.com/spf13/cobra"

func gitWorktreeLockCmd(parent *cobra.Command) {
	var flagReason string

	cmd := &cobra.Command{
		Use:   "lock <worktree>",
		Short: "Lock a worktree so prune and non-forced remove leave it alone",
		Long: `lock marks a worktree of the repository detected from the current directory
as locked, so prune and a non-forced remove will not touch it — useful for a
worktree on removable media or otherwise temporarily absent. The <worktree>
argument is a worktree directory name (its handle) or a path, and completes
against the repository's existing worktrees.`,
		Example: `  crabswarm git worktree lock feature
  crabswarm git worktree lock feature --reason "on a USB drive"`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: gitWorktreeLockCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGitWorktreeLock(cmd, args, flagReason)
		},
	}

	cmd.Flags().StringVar(&flagReason, "reason", "",
		"Reason recorded with the lock, shown by `git worktree list`")

	parent.AddCommand(cmd)
}

// gitWorktreeLockCompletion completes the single worktree argument and offers
// nothing once it is provided.
func gitWorktreeLockCompletion(
	cmd *cobra.Command,
	args []string,
	toComplete string,
) ([]string, cobra.ShellCompDirective) {
	if len(args) >= 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeGitWorktrees(cmd, args, toComplete)
}

func runGitWorktreeLock(cmd *cobra.Command, args []string, reason string) error {
	svc, repo, err := gitRepoFromCwd(cmd)
	if err != nil {
		return err
	}

	if err := svc.LockWorktree(cmd.Context(), repo, args[0], reason); err != nil {
		return err
	}
	cmd.Printf("Locked worktree %q\n", args[0])
	return nil
}
