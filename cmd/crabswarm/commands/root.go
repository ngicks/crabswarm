// Package commands contains the cobra commands for crabswarm.
package commands

import (
	"context"
	"errors"

	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
	"github.com/spf13/cobra"
)

// Execute runs the root command with the given context.
func Execute(ctx context.Context) error {
	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		if he, ok := errors.AsType[*handler.HandlerError](err); ok {
			he.Handle() // writes output, calls os.Exit
		}
	}
	return err
}

// rootCmd is the root command for crabswarm.
var rootCmd = &cobra.Command{
	Use:   "crabswarm",
	Short: "crabswarm CLI",
	Long:  `crabswarm is a CLI tool for managing Claude Code hooks.`,
}

func init() {
	rootCmd.PersistentFlags().String("sock", "", "Unix socket path")
}
