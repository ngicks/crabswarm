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
	// Silence cobra output for hook commands — we control stdout/stderr via HandlerError.Handle().
	SilenceErrors: true,
	SilenceUsage:  true,
}
