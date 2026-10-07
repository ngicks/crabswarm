// Package codexproxy runs a remote Codex TUI through a WebSocket proxy that
// stamps the TUI's chat identity on every thread it starts.
//
// Several TUIs can attach to one `codex app-server`, and every thread's MCP
// servers run under the app server, so nothing on the MCP side can tell which
// TUI a thread belongs to. The proxy sits on the wire between one TUI and the
// app server and adds the identity to each thread's config, as the header the
// thread's HTTP MCP session sends.
package codexproxy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
	"github.com/ngicks/crabswarm/crabswarm/mcp"
	"github.com/ngicks/crabswarm/internal/cmdsignals"
)

// DefaultServer is the name the crabswarm MCP server is configured under in
// the app server's mcp_servers table.
const DefaultServer = mcp.SharedServerName

// Config is what [Run] runs.
type Config struct {
	// Token is the identity token to stamp, resolved with [cli.ResolveToken]:
	// empty falls back to $CRABSWARM_CHAT_TOKEN, then $CMDMAN_CMD_ID.
	Token string
	// Argv is the TUI command line. It names the app server with
	// `--remote unix://PATH` or `--remote=unix://PATH`.
	Argv []string
	// Server is the MCP server name the token is stamped for. Empty means
	// [DefaultServer].
	//
	// The app server has to declare it as a streamable-HTTP server. Codex
	// refuses a thread whose config sets http_headers on a server it does not
	// declare, or on a stdio one, so every thread the TUI starts fails then.
	Server string
	// SockDir is the directory the proxy serves the TUI's socket in, as
	// <pid>.sock. Run creates it and makes it private to its owner.
	//
	// Empty means $XDG_RUNTIME_DIR/crabswarm-codex-proxy, or the same
	// directory under [os.TempDir] where that variable is unset. The crabswarm
	// config derives its default from a runtime directory that also probes
	// /run/user/<uid>, so a caller holding that config passes its value here.
	SockDir string
}

// ExitError reports that the command ran and exited unsuccessfully. Code is
// the status a shell reports for it: the exit status, or 128 plus the signal
// number when a signal ended it.
type ExitError struct {
	Code int
	err  *exec.ExitError
}

func (e *ExitError) Error() string { return e.err.Error() }

func (e *ExitError) Unwrap() error { return e.err }

// ExitStatus is Code: the status a process standing in for the command exits
// with.
func (e *ExitError) ExitStatus() int { return e.Code }

// childWaitDelay bounds how long a cancelled Run waits for the TUI to leave
// after asking it to, before killing it.
const childWaitDelay = 5 * time.Second

// Run serves a private socket, runs cfg.Argv with its --remote address
// replaced by that socket, and relays every connection the command makes to
// the app server it named. It returns once the command exits: nil for a
// status of zero, an [*ExitError] for any other.
//
// The command inherits the process's stdio, and SIGINT and SIGTERM received
// while it runs are passed on to it.
func Run(ctx context.Context, logger *slog.Logger, cfg Config) error {
	remote, err := findRemote(cfg.Argv)
	if err != nil {
		return err
	}
	upstream, err := unixSocketPath(remote.addr)
	if err != nil {
		return err
	}

	r := newRelay(logger, upstream, cfg.Server)
	if r.server == "" {
		r.server = DefaultServer
	}
	// A missing token is no reason to keep Codex from running: its threads
	// then act as nobody in the room, which every chat tool says when called.
	if token, err := cli.ResolveToken(cfg.Token); err != nil {
		logger.Warn("codex proxy: threads start without a chat identity", "err", err)
	} else {
		r.token = token
	}

	sock, err := socketPath(cfg.SockDir)
	if err != nil {
		return err
	}
	ln, err := listen(sock)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var serving errgroup.Group
	// serve closes ln, which removes the socket file with it.
	serving.Go(func() error { return r.serve(ctx, ln) })

	err = runChild(ctx, withRemote(cfg.Argv, remote, "unix://"+sock))
	cancel()
	serveErr := serving.Wait()

	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return &ExitError{Code: exitCode(exitErr.ProcessState), err: exitErr}
	}
	if err != nil {
		return err
	}
	return serveErr
}

// socketPath is where the proxy serves the TUI: <pid>.sock in dir, which it
// creates. An empty dir is the fallback [Config.SockDir] describes.
func socketPath(dir string) (string, error) {
	if dir == "" {
		// Kept apart from the crabswarm config's runtime-directory derivation,
		// which lives in the config layer this package does not import.
		base := os.Getenv("XDG_RUNTIME_DIR")
		if base == "" {
			base = os.TempDir()
		}
		dir = filepath.Join(base, "crabswarm-codex-proxy")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("creating the proxy socket directory: %w", err)
	}
	// The directory can sit in one every user of the host shares, such as
	// /tmp. Taking the directory over refuses one somebody else made, who could
	// otherwise swap the socket and read the token off the wire.
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", fmt.Errorf("securing the proxy socket directory: %w", err)
	}
	return filepath.Join(dir, strconv.Itoa(os.Getpid())+".sock"), nil
}

// listen serves the unix socket at path. A socket already there that nobody
// answers on was left by a proxy that died with the same pid, and is replaced.
func listen(path string) (net.Listener, error) {
	ln, err := net.Listen("unix", path)
	if err == nil {
		return ln, nil
	}
	if conn, dialErr := net.Dial("unix", path); dialErr == nil {
		_ = conn.Close()
		return nil, fmt.Errorf("serving the proxy socket: %w", err)
	}
	if rmErr := os.Remove(path); rmErr != nil {
		return nil, fmt.Errorf("serving the proxy socket: %w", err)
	}
	ln, err = net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("serving the proxy socket: %w", err)
	}
	return ln, nil
}

// runChild runs argv on the process's own stdio and waits for it, passing on
// the exit signals the process receives meanwhile.
func runChild(ctx context.Context, argv []string) error {
	child := exec.CommandContext(ctx, argv[0], argv[1:]...)
	child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
	// A cancelled Run asks the TUI to leave rather than killing it, so it can
	// restore the terminal on its way out.
	child.Cancel = func() error { return child.Process.Signal(syscall.SIGTERM) }
	child.WaitDelay = childWaitDelay

	// The command owns the terminal, so a signal is its to act on. Left to
	// cmdsignals, the first one would cancel ctx and take the relay down under
	// a TUI that may mean to keep running.
	sigs := make(chan os.Signal, 1)
	notify := func() { signal.Notify(sigs, cmdsignals.ExitSignals[:]...) }
	paused := cmdsignals.Pause(ctx, notify)
	if !paused {
		notify()
	}
	defer func() {
		if paused && cmdsignals.Resume(ctx, func() { signal.Stop(sigs) }) {
			return
		}
		signal.Stop(sigs)
	}()

	if err := child.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", argv[0], err)
	}

	forwardCtx, stopForward := context.WithCancel(ctx)
	var forwarding errgroup.Group
	forwarding.Go(func() error {
		for {
			select {
			case sig := <-sigs:
				_ = child.Process.Signal(sig)
			case <-forwardCtx.Done():
				return nil
			}
		}
	})
	err := child.Wait()
	stopForward()
	_ = forwarding.Wait()
	return err
}

// exitCode is the status a shell reports for a command that exited as state
// says.
func exitCode(state *os.ProcessState) int {
	if code := state.ExitCode(); code >= 0 {
		return code
	}
	if ws, ok := state.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 128 + int(ws.Signal())
	}
	return 1
}
