package cmdman_test

import (
	"testing"
	"time"
)

// TestLifecycle_RunStopRm verifies the full lifecycle:
// run → verify running → stop → verify exited → rm → verify gone.
func TestLifecycle_RunStopRm(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// Step 1: Run a long-lived command.
	id := env.run("run", "-n", "lifecycle-cmd", "--", "/bin/sh", "-c", "sleep 300")

	// Step 2: Wait for running state.
	env.waitForState("lifecycle-cmd", "running", defaultTimeout)

	// Step 3: Verify it appears in ls.
	entries := env.lsJSON()
	found := false
	for _, e := range entries {
		if e["Name"] == "lifecycle-cmd" {
			found = true
			if e["State"] != "running" {
				t.Errorf("expected state=running in ls, got %v", e["State"])
			}
		}
	}
	if !found {
		t.Fatal("lifecycle-cmd not found in ls output")
	}

	// Step 4: Inspect while running.
	info := env.inspectJSON("lifecycle-cmd")
	if info["state"] != "running" {
		t.Errorf("expected state=running in inspect, got %v", info["state"])
	}
	liveStatus, _ := info["live_status"].(map[string]any)
	if liveStatus == nil {
		t.Error("expected live_status for running command")
	}

	// Step 5: Stop the command.
	env.run("stop", "lifecycle-cmd")

	// Step 6: Wait for exited state.
	env.waitForState("lifecycle-cmd", "exited", defaultTimeout)

	// Step 7: Verify exited state in inspect.
	info = env.inspectJSON("lifecycle-cmd")
	if info["state"] != "exited" {
		t.Errorf("expected state=exited after stop, got %v", info["state"])
	}

	// Step 8: Remove.
	env.run("rm", "lifecycle-cmd")

	// Step 9: Verify gone from ls.
	entries = env.lsJSON()
	for _, e := range entries {
		if e["ID"] == id {
			t.Error("command still found in ls after rm")
		}
	}

	// Step 10: Inspect should fail.
	_, _ = env.runExpectFail("inspect", "lifecycle-cmd")
}

// TestLifecycle_RunAutoRemove verifies run with --rm:
// run --rm → verify running → command exits → verify auto-removed.
func TestLifecycle_RunAutoRemove(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	id := env.run("run", "--rm", "-n", "auto-rm-lifecycle", "--", "/bin/sh", "-c", "echo done")

	// Wait for auto-removal.
	waitUntil(t, defaultTimeout, func() bool {
		entries := env.lsJSON()
		for _, e := range entries {
			if e["ID"] == id {
				return false
			}
		}
		return true
	}, "command %s was not auto-removed", id)
}

// TestLifecycle_RunRestartStop verifies restart + stop:
// run --restart=always → verify restarts → stop → verify exited.
func TestLifecycle_RunRestartStop(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// Start a command that exits immediately but always restarts.
	id := env.run("run", "-n", "restart-lifecycle",
		"--restart", "always",
		"--", "/bin/sh", "-c", "echo restarting; exit 0")
	t.Cleanup(func() { env.cleanupCommand(id) })

	// Wait for multiple restarts.
	time.Sleep(2 * time.Second)

	// Verify it has restarted multiple times.
	info := env.inspectJSON("restart-lifecycle")
	history, _ := info["exit_history"].([]any)
	if len(history) < 2 {
		t.Errorf("expected at least 2 exit_history entries, got %d", len(history))
	}

	// Stop it.
	env.run("stop", "restart-lifecycle")
	env.waitForState("restart-lifecycle", "exited", defaultTimeout)

	info = env.inspectJSON("restart-lifecycle")
	if info["state"] != "exited" {
		t.Errorf("expected state=exited after stop, got %v", info["state"])
	}
}

// TestLifecycle_MultipleCommands verifies managing multiple commands simultaneously.
func TestLifecycle_MultipleCommands(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// Start three commands.
	id1 := env.run("run", "-n", "multi-1", "-l", "group=multi", "--", "/bin/sh", "-c", "sleep 300")
	id2 := env.run("run", "-n", "multi-2", "-l", "group=multi", "--", "/bin/sh", "-c", "sleep 300")
	id3 := env.run("run", "-n", "multi-3", "-l", "group=multi", "--", "/bin/sh", "-c", "sleep 300")
	t.Cleanup(func() {
		env.cleanupCommand(id1)
		env.cleanupCommand(id2)
		env.cleanupCommand(id3)
	})

	env.waitForState(id1, "running", defaultTimeout)
	env.waitForState(id2, "running", defaultTimeout)
	env.waitForState(id3, "running", defaultTimeout)

	// All three should appear in ls.
	entries := env.lsJSON("-l", "group=multi")
	if len(entries) != 3 {
		t.Errorf("expected 3 entries with group=multi, got %d", len(entries))
	}

	// Stop all with label.
	env.run("stop", "-l", "group=multi")

	env.waitForState(id1, "exited", defaultTimeout)
	env.waitForState(id2, "exited", defaultTimeout)
	env.waitForState(id3, "exited", defaultTimeout)

	// Remove all with label.
	env.run("rm", "-l", "group=multi")

	entries = env.lsJSON("-l", "group=multi")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after rm, got %d", len(entries))
	}
}
