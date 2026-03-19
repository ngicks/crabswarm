package cmdman_test

import (
	"os"
	"testing"
	"time"
)

func TestRestart_NoPolicy(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// With --restart=no (default), the command should exit and stay exited.
	id := env.run("run", "--restart", "no", "--", "/bin/sh", "-c", "exit 1")
	t.Cleanup(func() { env.cleanupCommand(id) })
	env.waitForState(id, "exited", defaultTimeout)

	info := env.inspectJSON(id)

	// Should have exactly one exit history entry.
	history, _ := info["exit_history"].([]any)
	if len(history) != 1 {
		t.Errorf("expected 1 exit_history entry with restart=no, got %d", len(history))
	}

	exitCode, _ := info["exit_code"].(float64)
	if exitCode != 1 {
		t.Errorf("expected exit_code=1, got %v", exitCode)
	}
}

func TestRestart_OnFailure(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// Create a script that fails twice then succeeds.
	scriptDir := t.TempDir()
	counterFile := scriptDir + "/counter"

	script := scriptDir + "/countdown.sh"
	writeFile(t, script, `#!/bin/sh
count=$(cat "`+counterFile+`" 2>/dev/null || echo 0)
count=$((count + 1))
echo "$count" > "`+counterFile+`"
if [ "$count" -lt 3 ]; then
  exit 1
fi
exit 0
`)

	id := env.run("run", "--restart", "on-failure", "--", "/bin/sh", script)
	t.Cleanup(func() { env.cleanupCommand(id) })

	// Should eventually exit successfully after 3 runs (2 failures + 1 success).
	env.waitForState(id, "exited", defaultTimeout)

	info := env.inspectJSON(id)
	exitCode, _ := info["exit_code"].(float64)
	if exitCode != 0 {
		t.Errorf("expected final exit_code=0 with on-failure restart, got %v", exitCode)
	}

	// Should have 3 exit history entries.
	history, _ := info["exit_history"].([]any)
	if len(history) != 3 {
		t.Errorf("expected 3 exit_history entries, got %d", len(history))
	}

	// Verify restart count in state_detail.
	stateDetail, _ := info["state_detail"].(map[string]any)
	restartCount, _ := stateDetail["restart_count"].(float64)
	if restartCount != 2 {
		t.Errorf("expected restart_count=2, got %v", restartCount)
	}
}

func TestRestart_OnFailure_SuccessDoesNotRestart(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// A command that exits with 0 should not be restarted.
	id := env.run("run", "--restart", "on-failure", "--", "/bin/sh", "-c", "exit 0")
	t.Cleanup(func() { env.cleanupCommand(id) })
	env.waitForState(id, "exited", defaultTimeout)

	info := env.inspectJSON(id)
	history, _ := info["exit_history"].([]any)
	if len(history) != 1 {
		t.Errorf("expected 1 exit_history entry (no restart on success), got %d", len(history))
	}
}

func TestRestart_Always(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// A command with --restart=always keeps restarting even on success.
	// Use a command that sleeps briefly so it doesn't restart too fast.
	id := env.run("run", "-n", "always-restart", "--restart", "always",
		"--", "/bin/sh", "-c", "sleep 0.5; exit 0")
	t.Cleanup(func() { env.cleanupCommand(id) })

	// Wait for a few restart cycles.
	time.Sleep(3 * time.Second)

	info := env.inspectJSON("always-restart")

	// Should have multiple exit history entries.
	history, _ := info["exit_history"].([]any)
	if len(history) < 2 {
		t.Errorf("expected at least 2 exit_history entries with restart=always, got %d", len(history))
	}

	// Stop the always-restarting command.
	env.run("stop", "always-restart")
	env.waitForState("always-restart", "exited", defaultTimeout)
}

func TestRestart_AlwaysStoppedBySignal(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// Start a command that always restarts and sleeps.
	id := env.run("run", "-n", "always-sleep", "--restart", "always",
		"--", "/bin/sh", "-c", "sleep 300")
	t.Cleanup(func() { env.cleanupCommand(id) })
	env.waitForState("always-sleep", "running", defaultTimeout)

	// Stopping it should terminate it despite always-restart policy.
	env.run("stop", "always-sleep")
	env.waitForState("always-sleep", "exited", defaultTimeout)

	info := env.inspectJSON("always-sleep")
	if info["state"] != "exited" {
		t.Errorf("expected state=exited after stop, got %v", info["state"])
	}
}

// writeFile is a test helper that writes content to a file and makes it executable.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
