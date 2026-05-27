package commands

import "github.com/spf13/cobra"

func statuslineCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "statusline",
		Short: "Status line helpers for Claude Code",
	}

	statuslineRenderCmd(cmd)

	parent.AddCommand(cmd)
}
