// Package main is the entry point for the crabswarm CLI.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/ngicks/crabswarm/cmd/crabswarm/commands"
	"github.com/ngicks/crabswarm/cmd/internal/cmdsignals"
	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		cmdsignals.ExitSignals[:]...,
	)
	defer stop()

	if err := commands.Execute(ctx); err != nil {
		// HandlerError carries hook-protocol output and its own exit code.
		handler.Handle(err)
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
