package commands

import "github.com/spf13/cobra"

func gitWorktreeCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "worktree",
		Aliases: []string{"wt"},
		Short:   "Manage worktrees of the repository detected from the current directory",
	}

	gitWorktreeAddCmd(cmd)
	gitWorktreeListCmd(cmd)
	gitWorktreeRemoveCmd(cmd)
	gitWorktreeMoveCmd(cmd)
	gitWorktreePruneCmd(cmd)
	gitWorktreeLockCmd(cmd)
	gitWorktreeUnlockCmd(cmd)
	gitWorktreeRepairCmd(cmd)

	parent.AddCommand(cmd)
}
