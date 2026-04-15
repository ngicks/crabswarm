package commands

import (
	"log/slog"

	"github.com/ngicks/crabswarm/pkg/crabswarm/server"
	"github.com/ngicks/go-common/contextkey"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

// serveCmd is the serve subcommand.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the crabswarm server",
	RunE:  runServeCmd,
}

func runServeCmd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	logger, _ := contextkey.ValueSlogLogger(ctx)
	if logger == nil {
		logger = slog.Default()
	}

	sockPath := resolveSocketPath(cmd)

	server := server.New(logger, sockPath)

	return server.Serve(ctx)
}
