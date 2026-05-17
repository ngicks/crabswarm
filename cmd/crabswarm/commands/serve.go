package commands

import (
	"log/slog"

	"github.com/ngicks/go-common/contextkey"
	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/pkg/crabswarm"
	"github.com/ngicks/crabswarm/pkg/crabswarm/server"
)

func serveCmd(parent *cobra.Command, flagSock *string) {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the crabswarm server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cmd, args, *flagSock)
		},
	}

	parent.AddCommand(cmd)
}

func runServe(cmd *cobra.Command, _ []string, flagSock string) error {
	ctx := cmd.Context()

	logger, _ := contextkey.ValueSlogLogger(ctx)
	if logger == nil {
		logger = slog.Default()
	}

	cfg, err := crabswarm.Config{Sock: flagSock}.Load()
	if err != nil {
		return err
	}

	srv := server.New(logger, cfg.Sock)
	return srv.Serve(ctx)
}
