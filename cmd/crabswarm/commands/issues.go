package commands

import "github.com/spf13/cobra"

func issuesCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "issues",
		Short: "Work with the beads issue backlog",
	}

	issuesLintCmd(cmd)

	parent.AddCommand(cmd)
}
