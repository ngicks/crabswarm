package cmdman

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// CommandConfigJSON is the canonical command configuration stored in CommandConfig.JSON.
type CommandConfigJSON struct {
	// Argv is the command and its arguments.
	Argv []string `json:"argv"`
	// Dir is the working directory for the command.
	Dir string `json:"dir,omitempty"`
	// Env is environment variables for the command.
	Env []string `json:"env,omitempty"`
	// StartupKeys are keys to send to the PTY after the command starts.
	StartupKeys []string `json:"startup_keys,omitempty"`
	// RestartPolicy is one of "no", "on-failure", "always".
	RestartPolicy string `json:"restart_policy"`
	// ScrollbackBytes is the scrollback buffer size in bytes.
	ScrollbackBytes int `json:"scrollback_bytes"`
	// Labels are user-defined key-value metadata.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations are system metadata (e.g., auto-remove).
	Annotations map[string]string `json:"annotations,omitempty"`
	// CommandDir is the per-command directory path.
	CommandDir string `json:"command_dir"`
}

const (
	AnnotationAutoRemove = "crabswarm.auto-remove"
)

const (
	RestartNo        = "no"
	RestartOnFailure = "on-failure"
	RestartAlways    = "always"
)

const DefaultScrollbackBytes = 1048576 // 1 MiB

// CommandStateJSON stores mutable runtime fields in CommandState.JSON.
type CommandStateJSON struct {
	// MonitorPID is the PID of the monitor process.
	MonitorPID int `json:"monitor_pid,omitempty"`
	// SocketPath is the Unix socket path for the monitor gRPC server.
	SocketPath string `json:"socket_path,omitempty"`
	// StartedAt is the RFC3339 timestamp when the command started.
	StartedAt string `json:"started_at,omitempty"`
	// FinishedAt is the RFC3339 timestamp when the command finished.
	FinishedAt string `json:"finished_at,omitempty"`
	// RestartCount is how many times the command has been restarted.
	RestartCount int `json:"restart_count"`
	// Error contains error details when the command is in errored state.
	Error string `json:"error,omitempty"`
}

// MaterializeConfigJSON writes the config.json file to the command directory.
func MaterializeConfigJSON(cfg *CommandConfigJSON) error {
	dir := filepath.Dir(cfg.CommandDir + "/config.json")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cfg.CommandDir, "config.json"), data, 0o644)
}

// ReadConfigJSON reads config.json from the given command directory.
func ReadConfigJSON(commandDir string) (*CommandConfigJSON, error) {
	data, err := os.ReadFile(filepath.Join(commandDir, "config.json"))
	if err != nil {
		return nil, err
	}
	var cfg CommandConfigJSON
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
