package commands

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// resolveSocketPath returns the Unix socket path using the following precedence:
// 1. --sock flag value
// 2. $CRABSWARM_SOCK environment variable
// 3. ${XDG_RUNTIME_DIR:-/tmp}/crabswarm/default.sock
func resolveSocketPath(cmd *cobra.Command) string {
	if v, _ := cmd.Flags().GetString("sock"); v != "" {
		return v
	}
	if env := os.Getenv("CRABSWARM_SOCK"); env != "" {
		return env
	}
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = "/tmp"
	}
	return filepath.Join(runtimeDir, "crabswarm", "default.sock")
}
