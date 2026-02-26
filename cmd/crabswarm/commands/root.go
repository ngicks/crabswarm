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

var k = &struct{}{}

var rootCmd = &cobra.Command{
	Use:   "crabswarm",
	Short: "crabswarm CLI",
	Long:  `crabswarm is a CLI tool for managing Claude Code hooks.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(cmd.Context())
		ctx = context.WithValue(ctx, k, cancel)
		cmd.SetContext(ctx)
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		cancel := cmd.Context().Value(k).(context.CancelFunc)
		cancel()
	},
}

func init() {
	rootCmd.PersistentFlags().String("sock", "", "Unix socket path")
}
