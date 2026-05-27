// Package commands contains the cobra commands for crabswarm.
package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ngicks/go-common/contextkey"
	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/internal/loggerfactory"
)

// Execute runs the root command with the given context.
func Execute(ctx context.Context) error {
	return rootCmd().ExecuteContext(ctx)
}

func rootCmd() *cobra.Command {
	var (
		logConfig   *loggerfactory.Config
		flagSock    string
		flagVersion bool
	)

	cmd := &cobra.Command{
		Use:           "crabswarm",
		Short:         "crabswarm CLI",
		Long:          "crabswarm is a CLI tool for managing Claude Code hooks.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if err := loggerfactory.ReadEnv(logConfig, "crabswarm", os.Environ()); err != nil {
				fmt.Fprintln(os.Stderr, "warning:", err)
			}
			logger := loggerfactory.BuildLogger(logConfig)
			slog.SetDefault(logger)
			cmd.SetContext(contextkey.WithSlogLogger(cmd.Context(), logger))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagVersion {
				return runVersion(cmd, args)
			}
			return runRoot(cmd, args)
		},
	}

	logConfig = loggerfactory.RegisterFlags(cmd)
	cmd.PersistentFlags().StringVar(&flagSock, "sock", "", "Unix socket path")
	cmd.Flags().BoolVar(&flagVersion, "version", false, "alias for the version subcommand")

	versionCmd(cmd)
	serveCmd(cmd, &flagSock)
	hookCmd(cmd, &flagSock)
	statuslineCmd(cmd)

	return cmd
}

func runRoot(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
