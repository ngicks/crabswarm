//go:build unix

package codexproxy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/internal/cmdsignals"
)

// A SIGTERM the proxy receives while the command runs is the command's to act
// on: it reaches the command, whose status Run reports, and it does not cancel
// the context cmdsignals hands the CLI.
func TestRun_PassesASignalOnToTheCommand(t *testing.T) {
	for _, tc := range []struct {
		name      string
		cmdsignal bool
	}{
		{name: "under cmdsignals", cmdsignal: true},
		{name: "on a plain context"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_RUNTIME_DIR", shortDir(t))
			ready := filepath.Join(shortDir(t), "ready")
			t.Setenv("CODEXPROXY_TEST_READY", ready)

			ctx := t.Context()
			if tc.cmdsignal {
				blockOn, sigCtx, cancel := cmdsignals.NotifyContext(context.Background())
				var watching errgroup.Group
				watching.Go(func() error { blockOn(); return nil })
				t.Cleanup(func() {
					cancel(nil)
					_ = watching.Wait()
				})
				ctx = sigCtx
			}

			var signalling errgroup.Group
			signalling.Go(func() error {
				// The command touches ready once its trap is set, which is after Run
				// has taken the signals over.
				deadline := time.Now().Add(10 * time.Second)
				for {
					if _, err := os.Stat(ready); err == nil {
						return syscall.Kill(os.Getpid(), syscall.SIGTERM)
					}
					if time.Now().After(deadline) {
						return errors.New("the command never became ready")
					}
					time.Sleep(10 * time.Millisecond)
				}
			})

			err := runShWith(t, ctx,
				`trap 'exit 42' TERM
touch "$CODEXPROXY_TEST_READY"
while :; do sleep 0.05; done`,
				"--remote", "unix:///nowhere.sock")
			assert.NilError(t, signalling.Wait())
			exitErr, ok := errors.AsType[*ExitError](err)
			assert.Assert(t, ok, "Run returned %v", err)
			assert.Equal(t, exitErr.Code, 42)
			assert.NilError(t, ctx.Err())
		})
	}
}
