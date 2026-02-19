package commands

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(hookCmd)
}

// hookCmd is the parent group command for hook subcommands.
var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Hook management commands",
}
