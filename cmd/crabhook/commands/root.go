// Package commands contains the cobra commands for crabhook.
package commands

import (
	"context"

	"github.com/spf13/cobra"
)

// Execute runs the root command with the given context.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

// rootCmd is the root command for crabhook.
var rootCmd = &cobra.Command{
	Use:   "crabhook",
	Short: "crabhook CLI",
	Long:  `crabhook is a CLI tool for managing Claude Code hooks.`,
}

var sock string

func init() {
	rootCmd.PersistentFlags().StringVar(&sock, "sock", "", "Unix socket path")
}
