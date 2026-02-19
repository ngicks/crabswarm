// Package main is the entry point for the crabhook CLI.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/ngicks/crabswarm/cmd/crabhook/commands"
	"github.com/ngicks/go-common/contextkey"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx = contextkey.WithSlogLogger(ctx, logger)

	if err := commands.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
