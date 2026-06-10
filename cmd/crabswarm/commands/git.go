package commands

import "github.com/spf13/cobra"

func gitCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "git",
		Short: "Git repository helpers (ghq-style bare clones and worktrees)",
	}

	gitCloneCmd(cmd)
	gitWorktreeCmd(cmd)

	parent.AddCommand(cmd)
}
