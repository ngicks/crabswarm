package commands

import "github.com/spf13/cobra"

func hookPathCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Hooks that validate the file path a tool is about to edit",
	}

	hookPathWindowsCmd(cmd)

	parent.AddCommand(cmd)
}
