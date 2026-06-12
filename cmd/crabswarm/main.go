// Package main is the entry point for the crabswarm CLI.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/ngicks/crabswarm/cmd/crabswarm/commands"
	"github.com/ngicks/crabswarm/cmd/internal/cmdsignals"
	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
)

func main() {
	blockOn, ctx, cancel := cmdsignals.NotifyContext(context.Background())

	// blockOn watches ExitSignals and cancels ctx when one arrives; it must run
	// for signal propagation to work, so start it before Execute. cancel + Wait
	// tear the goroutine down afterwards — whether Execute returned on its own or
	// because a signal already cancelled ctx (cancel is a no-op in that case).
	var wg sync.WaitGroup
	wg.Go(blockOn)

	err := commands.Execute(ctx)

	// Recover the cancellation reason while ctx still reflects it. The guard is
	// errors.Is(err, ctx.Err()) — not the bare context.Canceled sentinel, which
	// any code may return without this ctx being cancelled — so it fires only
	// when *this* context was actually cancelled. Read it before cancel(nil)
	// below, or that cleanup call would set ctx.Err() and manufacture a false
	// positive. Execute surfaces only context.Canceled; the signal lives in the
	// cause as *SignalReceivedError.
	if err != nil && errors.Is(err, ctx.Err()) {
		if sigErr, ok := errors.AsType[*cmdsignals.SignalReceivedError](context.Cause(ctx)); ok {
			err = sigErr
		}
	}

	cancel(nil)
	wg.Wait()

	if err != nil {
		// HandlerError carries hook-protocol output and its own exit code; when
		// err is one, Handle writes it and exits the process. Otherwise (a plain
		// error or a recovered *SignalReceivedError) it returns and we fall
		// through to the generic report.
		handler.Handle(err)
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
