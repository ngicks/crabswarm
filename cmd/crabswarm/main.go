// Package main is the entry point for the crabswarm CLI.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/ngicks/crabswarm/cmd/crabswarm/commands"
	cmdsignals "github.com/ngicks/crabswarm/cmd/internal/signals"
	"github.com/ngicks/go-common/contextkey"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), cmdsignals.ExitSignals[:]...)
	defer stop()

	logger := slog.New(
		slog.NewJSONHandler(
			os.Stderr,
			&slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelDebug,
			},
		),
	)
	ctx = contextkey.WithSlogLogger(ctx, logger)

	if err := commands.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
