package cmdman

import (
	"os"
	"path/filepath"
)

const (
	ENV_CMDMAN_DATA_DIR    = "CMDMAN_DATA_DIR"
	ENV_CMDMAN_RUNTIME_DIR = "CMDMAN_RUNTIME_DIR"
)

// DataDir returns the base data directory for cmdman.
// Resolution order: $CMDMAN_DATA_DIR → $XDG_DATA_HOME/cmdman → ~/.local/share/cmdman.
func DataDir() string {
	if dir := os.Getenv(ENV_CMDMAN_DATA_DIR); dir != "" {
		return dir
	}
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, "cmdman")
}

// RuntimeDir returns the runtime directory for cmdman.
// Resolution order: $CMDMAN_RUNTIME_DIR → $XDG_RUNTIME_DIR/cmdman → /tmp/cmdman.
func RuntimeDir() string {
	if dir := os.Getenv(ENV_CMDMAN_RUNTIME_DIR); dir != "" {
		return dir
	}
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = "/tmp"
	}
	return filepath.Join(dir, "cmdman")
}

// DBPath returns the path to the SQLite database.
func DBPath() string {
	return filepath.Join(DataDir(), "commands.db")
}

// CommandDir returns the per-command directory for the given command ID.
func CommandDir(id string) string {
	return filepath.Join(DataDir(), "commands", id)
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
