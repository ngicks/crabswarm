package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/ngicks/crabswarm/cmd/cmdman/commands"
	cmdsignals "github.com/ngicks/crabswarm/cmd/internal/signals"
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
	_ = logger

	if err := commands.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
