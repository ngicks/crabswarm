package preview

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// cmdmanBin is the daemon-manager binary EnsureDaemon shells out to. It is
// expected on PATH (installed via mise, per the plan); tests inject a fake by
// prepending a directory to PATH.
const cmdmanBin = "cmdman"

// Health-poll budget: after (re)starting the daemon, EnsureDaemon polls
// /healthz until the server answers 200 or the deadline elapses.
const (
	healthzTimeout   = 10 * time.Second
	healthzInterval  = 200 * time.Millisecond
	healthzProbeWait = 2 * time.Second
)

// EnsureDaemon makes sure the preview daemon is running and reachable, then
// returns once it answers /healthz with 200.
//
// It checks the daemon's state with `cmdman inspect`; if the daemon is not
// already running it starts one with `cmdman run`, pointing cmdman at this same
// crabswarm executable's hidden `preview __serve` command. Regardless of
// whether it had to start the daemon, it finishes by polling /healthz so the
// caller can immediately make RPCs. logger may be nil.
func EnsureDaemon(ctx context.Context, logger *slog.Logger, cfg Config, configPath string) error {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	state, present := inspectDaemon(ctx, logger, cfg.DaemonName)
	if !present || state != "running" {
		if present {
			// A stale (exited/created) command still occupies the name;
			// `cmdman run --name` needs a free name, so drop it first.
			logger.Debug("removing stale preview daemon",
				"name", cfg.DaemonName, "state", state)
			removeDaemon(ctx, logger, cfg.DaemonName)
		}
		if err := runDaemon(ctx, logger, cfg, configPath); err != nil {
			return err
		}
	}

	return waitHealthy(ctx, logger, cfg.Addr)
}

// inspectDaemon queries cmdman for the daemon's runtime state. It returns the
// state string cmdman reports (lowercase, e.g. "running" or "exited") and
// whether the command is present at all.
//
// Any inspect failure — most commonly cmdman reporting no such command with a
// non-zero exit — is reported as "not present" with an empty state; a genuinely
// broken cmdman surfaces later when EnsureDaemon tries to `cmdman run`.
//
// Verified against cmdman 0.0.15: `cmdman inspect NAME --format '{{.State}}'`
// prints the state and exits 0 for a known command, and exits non-zero with
// `error: resolve command: no command found matching "NAME"` for an unknown
// one. cmdman has no `--json` flag; `--format` takes a Go text/template with a
// `.State` field, which is the narrowest signal EnsureDaemon needs.
func inspectDaemon(
	ctx context.Context,
	logger *slog.Logger,
	name string,
) (state string, present bool) {
	out, err := exec.CommandContext(ctx, cmdmanBin, "inspect", name, "--format", "{{.State}}").
		Output()
	if err != nil {
		logger.Debug("cmdman inspect: treating daemon as absent",
			"name", name, "error", err)
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

// runDaemon starts the preview server under cmdman. It resolves this running
// crabswarm binary via os.Executable so the daemon runs the exact same build,
// then runs `cmdman run --name <DaemonName> -- <self> preview __serve
// --addr <Addr> [--config <configPath>]`.
func runDaemon(ctx context.Context, logger *slog.Logger, cfg Config, configPath string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating crabswarm executable: %w", err)
	}

	args := []string{
		"run", "--name", cfg.DaemonName, "--",
		self, "preview", "__serve", "--addr", cfg.Addr,
	}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}

	logger.Debug("starting preview daemon",
		"name", cfg.DaemonName, "addr", cfg.Addr, "config", configPath)
	if out, err := exec.CommandContext(ctx, cmdmanBin, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("cmdman run %q: %w: %s",
			cfg.DaemonName, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// removeDaemon drops a stale cmdman command by name so the name can be reused.
// It is best-effort: any failure is logged at debug and swallowed, because the
// subsequent `cmdman run` reports the real problem if the name is still taken.
func removeDaemon(ctx context.Context, logger *slog.Logger, name string) {
	if out, err := exec.CommandContext(ctx, cmdmanBin, "rm", name).CombinedOutput(); err != nil {
		logger.Debug("cmdman rm (stale daemon)",
			"name", name, "error", err, "output", strings.TrimSpace(string(out)))
	}
}

// waitHealthy polls the daemon's /healthz endpoint until it answers 200 or the
// budget (healthzTimeout, or ctx cancellation, whichever is sooner) is spent.
func waitHealthy(ctx context.Context, logger *slog.Logger, addr string) error {
	ctx, cancel := context.WithTimeout(ctx, healthzTimeout)
	defer cancel()

	url := baseURL(addr) + "/healthz"
	ticker := time.NewTicker(healthzInterval)
	defer ticker.Stop()

	for {
		if probeHealthz(ctx, url) {
			logger.Debug("preview daemon healthy", "url", url)
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("preview daemon at %s did not become healthy within %s: %w",
				url, healthzTimeout, ctx.Err())
		case <-ticker.C:
		}
	}
}

// probeHealthz issues one GET and reports whether the daemon answered 200. Each
// attempt is bounded so a single stalled connection cannot consume the whole
// poll budget.
func probeHealthz(ctx context.Context, url string) bool {
	ctx, cancel := context.WithTimeout(ctx, healthzProbeWait)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode == http.StatusOK
}
