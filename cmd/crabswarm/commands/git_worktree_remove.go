package commands

import "github.com/spf13/cobra"

func gitWorktreeRemoveCmd(parent *cobra.Command) {
	var flagForce bool

	cmd := &cobra.Command{
		Use:     "remove <worktree>",
		Aliases: []string{"rm"},
		Short:   "Remove a worktree by its directory name (or path)",
		Long: `remove deletes a worktree of the repository detected from the current
directory. The <worktree> argument is a worktree directory name (its handle)
or a path, and completes against the repository's existing worktrees.`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: gitWorktreeRemoveCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGitWorktreeRemove(cmd, args, flagForce)
		},
	}

	cmd.Flags().BoolVar(&flagForce, "force", false,
		"Remove even when the worktree has local modifications or untracked files")

	parent.AddCommand(cmd)
}

// gitWorktreeRemoveCompletion completes the single worktree argument and
// offers nothing once it is provided.
func gitWorktreeRemoveCompletion(
	cmd *cobra.Command,
	args []string,
	toComplete string,
) ([]string, cobra.ShellCompDirective) {
	if len(args) >= 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeGitWorktrees(cmd, args, toComplete)
}

func runGitWorktreeRemove(cmd *cobra.Command, args []string, force bool) error {
	svc, repo, err := gitRepoFromCwd(cmd)
	if err != nil {
		return err
	}

	if err := svc.RemoveWorktree(cmd.Context(), repo, args[0], force); err != nil {
		return err
	}
	cmd.Printf("Removed worktree %q\n", args[0])
	return nil
}
