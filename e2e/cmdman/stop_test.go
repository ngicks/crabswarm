package cmdman_test

import (
	"testing"
)

func TestStop_RunningCommand(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// Start a long-running command.
	id := env.run("run", "-n", "sleeper", "--", "/bin/sh", "-c", "sleep 300")
	t.Cleanup(func() { env.cleanupCommand(id) })

	env.waitForState("sleeper", "running", defaultTimeout)

	// Stop it.
	env.run("stop", "sleeper")

	// Wait for it to reach exited state.
	env.waitForState("sleeper", "exited", defaultTimeout)

	info := env.inspectJSON("sleeper")
	if info["state"] != "exited" {
		t.Errorf("expected state=exited after stop, got %v", info["state"])
	}
}

func TestStop_ByID(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	id := env.run("run", "--", "/bin/sh", "-c", "sleep 300")
	t.Cleanup(func() { env.cleanupCommand(id) })

	env.waitForState(id, "running", defaultTimeout)

	// Stop by ID.
	env.run("stop", id)

	env.waitForState(id, "exited", defaultTimeout)
}

func TestStop_WithSignal(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	id := env.run("run", "-n", "sig-test", "--", "/bin/sh", "-c", "sleep 300")
	t.Cleanup(func() { env.cleanupCommand(id) })

	env.waitForState("sig-test", "running", defaultTimeout)

	// Send SIGKILL explicitly.
	env.run("stop", "-s", "SIGKILL", "sig-test")

	env.waitForState("sig-test", "exited", defaultTimeout)
}

func TestStop_AlreadyExited(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	id := env.run("run", "--", "/bin/sh", "-c", "echo done")
	env.waitForState(id, "exited", defaultTimeout)

	// Stopping an already-exited command prints an error per-command
	// but the stop command itself may return 0.
	// The important thing is the command remains in exited state.
	stdout, stderr, _ := env.exec("stop", id)
	combined := stdout + " " + stderr
	// Should indicate an error (connection refused, no socket, etc.)
	if combined == " " {
		t.Log("stop on exited command produced no output (error was silent)")
	}

	// State should still be exited.
	info := env.inspectJSON(id)
	if info["state"] != "exited" {
		t.Errorf("expected state=exited, got %v", info["state"])
	}
}

func TestStop_WithLabels(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// Start two commands with the same label.
	id1 := env.run("run", "-l", "group=workers", "--", "/bin/sh", "-c", "sleep 300")
	id2 := env.run("run", "-l", "group=workers", "--", "/bin/sh", "-c", "sleep 300")
	t.Cleanup(func() {
		env.cleanupCommand(id1)
		env.cleanupCommand(id2)
	})

	env.waitForState(id1, "running", defaultTimeout)
	env.waitForState(id2, "running", defaultTimeout)

	// Stop all commands with the label.
	env.run("stop", "-l", "group=workers")

	env.waitForState(id1, "exited", defaultTimeout)
	env.waitForState(id2, "exited", defaultTimeout)
}
