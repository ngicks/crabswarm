package cmdman_test

import (
	"strings"
	"testing"
)

func TestLabels_MultipleLabelsANDLogic(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	// Create commands with different label combinations.
	id1 := env.run("run", "-l", "env=prod", "-l", "tier=web", "--", "/bin/sh", "-c", "sleep 300")
	id2 := env.run("run", "-l", "env=prod", "-l", "tier=api", "--", "/bin/sh", "-c", "sleep 300")
	id3 := env.run("run", "-l", "env=staging", "-l", "tier=web", "--", "/bin/sh", "-c", "sleep 300")
	t.Cleanup(func() {
		env.cleanupCommand(id1)
		env.cleanupCommand(id2)
		env.cleanupCommand(id3)
	})

	env.waitForState(id1, "running", defaultTimeout)
	env.waitForState(id2, "running", defaultTimeout)
	env.waitForState(id3, "running", defaultTimeout)

	// Filter by both labels (AND logic): env=prod AND tier=web.
	stdout := env.run("ls", "-q", "-l", "env=prod", "-l", "tier=web")
	if !strings.Contains(stdout, id1) {
		t.Errorf("expected %s (env=prod, tier=web) in output", id1)
	}
	if strings.Contains(stdout, id2) {
		t.Errorf("did not expect %s (env=prod, tier=api) in output", id2)
	}
	if strings.Contains(stdout, id3) {
		t.Errorf("did not expect %s (env=staging, tier=web) in output", id3)
	}
}

func TestLabels_SingleLabel(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	id1 := env.run("run", "-l", "role=worker", "--", "/bin/sh", "-c", "sleep 300")
	id2 := env.run("run", "-l", "role=scheduler", "--", "/bin/sh", "-c", "sleep 300")
	t.Cleanup(func() {
		env.cleanupCommand(id1)
		env.cleanupCommand(id2)
	})

	env.waitForState(id1, "running", defaultTimeout)
	env.waitForState(id2, "running", defaultTimeout)

	// Filter by single label.
	stdout := env.run("ls", "-q", "-l", "role=worker")
	if !strings.Contains(stdout, id1) {
		t.Errorf("expected %s in output for role=worker", id1)
	}
	if strings.Contains(stdout, id2) {
		t.Errorf("did not expect %s in output for role=worker", id2)
	}
}

func TestLabels_NoMatch(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	id := env.run("run", "-l", "color=blue", "--", "/bin/sh", "-c", "sleep 300")
	t.Cleanup(func() { env.cleanupCommand(id) })
	env.waitForState(id, "running", defaultTimeout)

	// Filter by a non-existent label value.
	entries := env.lsJSON("-l", "color=red")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for color=red, got %d", len(entries))
	}
}

func TestLabels_StopByLabel(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	id1 := env.run("run", "-l", "batch=morning", "--", "/bin/sh", "-c", "sleep 300")
	id2 := env.run("run", "-l", "batch=morning", "--", "/bin/sh", "-c", "sleep 300")
	id3 := env.run("run", "-l", "batch=evening", "--", "/bin/sh", "-c", "sleep 300")
	t.Cleanup(func() {
		env.cleanupCommand(id1)
		env.cleanupCommand(id2)
		env.cleanupCommand(id3)
	})

	env.waitForState(id1, "running", defaultTimeout)
	env.waitForState(id2, "running", defaultTimeout)
	env.waitForState(id3, "running", defaultTimeout)

	// Stop only the morning batch.
	env.run("stop", "-l", "batch=morning")

	env.waitForState(id1, "exited", defaultTimeout)
	env.waitForState(id2, "exited", defaultTimeout)

	// The evening one should still be running.
	info := env.inspectJSON(id3)
	if info["state"] != "running" {
		t.Errorf("expected id3 to still be running, got %v", info["state"])
	}
}

func TestLabels_RmByLabel(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	id1 := env.run("run", "-l", "disposable=yes", "--", "/bin/sh", "-c", "echo a")
	id2 := env.run("run", "-l", "disposable=yes", "--", "/bin/sh", "-c", "echo b")
	id3 := env.run("run", "-l", "disposable=no", "--", "/bin/sh", "-c", "echo c")

	env.waitForState(id1, "exited", defaultTimeout)
	env.waitForState(id2, "exited", defaultTimeout)
	env.waitForState(id3, "exited", defaultTimeout)

	// Remove only disposable=yes.
	env.run("rm", "-l", "disposable=yes")

	// Only id3 should remain.
	entries := env.lsJSON()
	for _, e := range entries {
		eid, _ := e["ID"].(string)
		if eid == id1 || eid == id2 {
			t.Errorf("labeled command %s still present after rm -l", eid)
		}
	}

	// Clean up the remaining one.
	env.run("rm", id3)
}

func TestLabels_InspectShowsLabels(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	id := env.run("run",
		"-l", "service=api",
		"-l", "version=v2",
		"--", "/bin/sh", "-c", "echo labeled-inspect",
	)
	t.Cleanup(func() { env.cleanupCommand(id) })
	env.waitForState(id, "exited", defaultTimeout)

	info := env.inspectJSON(id)
	cfg, _ := info["config"].(map[string]any)
	labels, _ := cfg["labels"].(map[string]any)

	if labels["service"] != "api" {
		t.Errorf("expected label service=api, got %v", labels["service"])
	}
	if labels["version"] != "v2" {
		t.Errorf("expected label version=v2, got %v", labels["version"])
	}
}
