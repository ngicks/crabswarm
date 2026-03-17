package commands

import (
	"context"

	"github.com/spf13/cobra"
)

// Execute runs the root command with the given context.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

var rootCmd = &cobra.Command{
	Use:   "cmdman",
	Short: "Command monitor manager",
	Long:  "cmdman manages long-running commands via per-command monitor processes.",
}
