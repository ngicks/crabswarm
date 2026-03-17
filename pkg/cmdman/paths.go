package cmdman

import (
	"os"
	"path/filepath"
)

// DataDir returns the base data directory for crabswarm commands.
// Uses $XDG_DATA_HOME/crabswarm, falling back to ~/.local/share/crabswarm.
func DataDir() string {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, "crabswarm")
}

// RuntimeDir returns the runtime directory for crabswarm.
// Uses $XDG_RUNTIME_DIR/crabswarm, falling back to /tmp/crabswarm.
func RuntimeDir() string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = "/tmp"
	}
	return filepath.Join(dir, "crabswarm")
}

// DBPath returns the path to the SQLite database.
func DBPath() string {
	return filepath.Join(DataDir(), "commands.db")
}

// CommandDir returns the per-command directory for the given command ID.
func CommandDir(id string) string {
	return filepath.Join(DataDir(), "commands", id)
}

// CommandConfigPath returns the path to config.json for a command.
func CommandConfigPath(id string) string {
	return filepath.Join(CommandDir(id), "config.json")
}

// MonitorRuntimeDir returns the per-command runtime directory.
func MonitorRuntimeDir(id string) string {
	return filepath.Join(RuntimeDir(), "cmd", id)
}

// MonitorSocketPath returns the Unix socket path for a command's monitor.
func MonitorSocketPath(id string) string {
	return filepath.Join(MonitorRuntimeDir(id), "monitor.sock")
}

// MonitorPIDPath returns the PID file path for a command's monitor.
func MonitorPIDPath(id string) string {
	return filepath.Join(MonitorRuntimeDir(id), "pid")
}
