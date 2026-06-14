package commands

import "github.com/spf13/cobra"

func gitWorktreeUnlockCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "unlock <worktree>",
		Short: "Unlock a worktree, reversing lock",
		Long: `unlock removes the lock from a worktree of the repository detected from the
current directory, reversing lock. The <worktree> argument is a worktree
directory name (its handle) or a path, and completes against the repository's
existing worktrees.`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: gitWorktreeUnlockCompletion,
		RunE:              runGitWorktreeUnlock,
	}

	parent.AddCommand(cmd)
}

// gitWorktreeUnlockCompletion completes the single worktree argument and
// offers nothing once it is provided.
func gitWorktreeUnlockCompletion(
	cmd *cobra.Command,
	args []string,
	toComplete string,
) ([]string, cobra.ShellCompDirective) {
	if len(args) >= 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeGitWorktrees(cmd, args, toComplete)
}

func runGitWorktreeUnlock(cmd *cobra.Command, args []string) error {
	svc, repo, err := gitRepoFromCwd(cmd)
	if err != nil {
		return err
	}

	if err := svc.UnlockWorktree(cmd.Context(), repo, args[0]); err != nil {
		return err
	}
	cmd.Printf("Unlocked worktree %q\n", args[0])
	return nil
}
