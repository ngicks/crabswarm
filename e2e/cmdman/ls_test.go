package cmdman_test

import (
	"strings"
	"testing"
)

func TestLs_Empty(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// Listing with no commands should return empty or null.
	entries := env.lsJSON()
	if len(entries) != 0 {
		t.Errorf("expected empty list, got %d entries", len(entries))
	}
}

func TestLs_ShowsRunningByDefault(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// Start a long-running command.
	id := env.run("run", "-n", "runner", "--", "/bin/sh", "-c", "sleep 300")
	t.Cleanup(func() { env.cleanupCommand(id) })
	env.waitForState("runner", "running", defaultTimeout)

	// Also start one that exits immediately.
	exitedID := env.run("run", "-n", "quitter", "--", "/bin/sh", "-c", "echo done")
	env.waitForState("quitter", "exited", defaultTimeout)
	t.Cleanup(func() { env.cleanupCommand(exitedID) })

	// Default ls (no -a) should only show the running command.
	stdout := env.run("ls", "--format", "json")
	if stdout == "null" || stdout == "" {
		t.Fatal("expected at least one entry in ls")
	}

	// The running command should appear.
	if !strings.Contains(stdout, "runner") {
		t.Error("running command 'runner' not found in ls output")
	}
}

func TestLs_AllFlag(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	id := env.run("run", "-n", "short-lived", "--", "/bin/sh", "-c", "echo done")
	t.Cleanup(func() { env.cleanupCommand(id) })
	env.waitForState("short-lived", "exited", defaultTimeout)

	// Without -a, exited commands may not appear.
	// With -a, it should appear.
	entries := env.lsJSON()
	found := false
	for _, e := range entries {
		if e["Name"] == "short-lived" {
			found = true
		}
	}
	if !found {
		t.Error("exited command not found in ls -a output")
	}
}

func TestLs_QuietMode(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	id := env.run("run", "--", "/bin/sh", "-c", "sleep 300")
	t.Cleanup(func() { env.cleanupCommand(id) })
	env.waitForState(id, "running", defaultTimeout)

	// Quiet mode should print only IDs.
	stdout := env.run("ls", "-q")
	if !strings.Contains(stdout, id) {
		t.Errorf("expected ID %s in quiet output, got %q", id, stdout)
	}
	// Should not contain table headers.
	if strings.Contains(stdout, "NAME") || strings.Contains(stdout, "STATE") {
		t.Error("quiet mode should not contain table headers")
	}
}

func TestLs_TableFormat(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	id := env.run("run", "-n", "table-test", "--", "/bin/sh", "-c", "sleep 300")
	t.Cleanup(func() { env.cleanupCommand(id) })
	env.waitForState("table-test", "running", defaultTimeout)

	// Default table format should contain headers.
	stdout := env.run("ls")
	if !strings.Contains(stdout, "ID") {
		t.Error("table output missing ID header")
	}
	if !strings.Contains(stdout, "NAME") {
		t.Error("table output missing NAME header")
	}
	if !strings.Contains(stdout, "STATE") {
		t.Error("table output missing STATE header")
	}
	if !strings.Contains(stdout, "table-test") {
		t.Error("table output missing command name")
	}
}

func TestLs_FilterByLabel(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// Start commands with different labels.
	id1 := env.run("run", "-l", "tier=frontend", "--", "/bin/sh", "-c", "sleep 300")
	id2 := env.run("run", "-l", "tier=backend", "--", "/bin/sh", "-c", "sleep 300")
	t.Cleanup(func() {
		env.cleanupCommand(id1)
		env.cleanupCommand(id2)
	})

	env.waitForState(id1, "running", defaultTimeout)
	env.waitForState(id2, "running", defaultTimeout)

	// Filter by label.
	stdout := env.run("ls", "-q", "-l", "tier=frontend")
	if !strings.Contains(stdout, id1) {
		t.Errorf("expected %s in label-filtered output", id1)
	}
	if strings.Contains(stdout, id2) {
		t.Errorf("did not expect %s in label-filtered output", id2)
	}
}
