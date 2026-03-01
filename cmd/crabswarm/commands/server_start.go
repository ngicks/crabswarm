package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/ngicks/crabswarm/pkg/mux/tmux"
	"github.com/spf13/cobra"
)

const defaultSessionName = "crabswarm"

func init() {
	serverCmd.AddCommand(serverStartCmd)
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the crabswarm server in a detached tmux session",
	RunE:  runServerStartCmd,
}

// shellQuote wraps a string in single quotes, escaping embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func runServerStartCmd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Resolve binary path.
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("eval symlinks: %w", err)
	}

	sockPath := resolveSocketPath(cmd)

	// Check if server is already running by trying to acquire the lock.
	lockPath := sockPath + ".lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err == nil {
		err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err != nil {
			// Lock is held — server is running.
			lockFile.Close()
			fmt.Fprintln(cmd.OutOrStdout(), "server already running")
			return nil
		}
		// We got the lock — release it so serve can acquire it.
		syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		lockFile.Close()
	}

	// Build the serve command string.
	serveCmd := shellQuote(exe) + " serve --sock " + shellQuote(sockPath)

	// Create or reuse a tmux session (DisallowReuse defaults to false).
	_, err = tmux.New(ctx, tmux.Config{
		Name:        defaultSessionName,
		StartupKeys: []string{serveCmd, "Enter"},
	})
	if err != nil {
		return fmt.Errorf("create tmux session: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "server started in tmux session %q\n", defaultSessionName)
	return nil
}
