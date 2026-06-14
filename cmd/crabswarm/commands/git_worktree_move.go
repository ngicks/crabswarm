package commands

import "github.com/spf13/cobra"

func gitWorktreeMoveCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "move <worktree> <dest>",
		Aliases: []string{"mv"},
		Short:   "Move a worktree to a new path",
		Long: `move relocates a worktree of the repository detected from the current
directory. The <worktree> argument is a worktree directory name (its handle)
or a path, and completes against the repository's existing worktrees. A
relative <dest> resolves against the repository directory so the worktree
stays a sibling of the others; pass an absolute path to move it elsewhere.

git refuses to move a worktree that is locked or that contains submodules.`,
		Example: `  crabswarm git worktree move feature renamed-feature
  crabswarm git worktree move feature ../elsewhere/feature`,
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: gitWorktreeMoveCompletion,
		RunE:              runGitWorktreeMove,
	}

	parent.AddCommand(cmd)
}

// gitWorktreeMoveCompletion completes the worktree handle (first arg) and
// leaves the destination path (second arg) to default file completion.
func gitWorktreeMoveCompletion(
	cmd *cobra.Command,
	args []string,
	toComplete string,
) ([]string, cobra.ShellCompDirective) {
	if len(args) >= 1 {
		return nil, cobra.ShellCompDirectiveDefault
	}
	return completeGitWorktrees(cmd, args, toComplete)
}

func runGitWorktreeMove(cmd *cobra.Command, args []string) error {
	svc, repo, err := gitRepoFromCwd(cmd)
	if err != nil {
		return err
	}

	moved, err := svc.MoveWorktree(cmd.Context(), repo, args[0], args[1])
	if err != nil {
		return err
	}
	cmd.Printf("Moved worktree %q to %s\n", args[0], moved)
	return nil
}
