package commands

import "github.com/spf13/cobra"

func gitWorktreeRepairCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "repair [path...]",
		Short: "Repair the administrative links between the repository and its worktrees",
		Long: `repair fixes up the administrative links between the repository detected from
the current directory and its worktrees after either side has been moved on
disk. With no arguments it repairs the worktrees the repository still knows
about; pass the new locations of moved worktrees so the repository can find
them again.`,
		Example: `  crabswarm git worktree repair
  crabswarm git worktree repair ../moved-feature`,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: gitWorktreeRepairCompletion,
		RunE:              runGitWorktreeRepair,
	}

	parent.AddCommand(cmd)
}

// gitWorktreeRepairCompletion leaves the path arguments to default file
// completion — the whole point of the paths is to name locations git does not
// yet track, so the existing-worktree completion would be misleading.
func gitWorktreeRepairCompletion(
	cmd *cobra.Command,
	args []string,
	toComplete string,
) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveDefault
}

func runGitWorktreeRepair(cmd *cobra.Command, args []string) error {
	svc, repo, err := gitRepoFromCwd(cmd)
	if err != nil {
		return err
	}

	if err := svc.RepairWorktrees(cmd.Context(), repo, args...); err != nil {
		return err
	}
	cmd.Println("Repaired worktree links")
	return nil
}
