package cmdman

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestRestartPolicyOnFailure(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	commandDir := filepath.Join(dir, "cmd-restart")

	store, err := OpenStore(dbPath)
	assert.NilError(t, err)
	defer store.Close()

	id := "test-restart-1"

	// Write a script that fails twice then succeeds.
	scriptPath := filepath.Join(dir, "countdown.sh")
	os.WriteFile(scriptPath, []byte(`#!/bin/sh
COUNTER_FILE="$1"
count=$(cat "$COUNTER_FILE" 2>/dev/null || echo 0)
count=$((count + 1))
echo "$count" > "$COUNTER_FILE"
if [ "$count" -lt 3 ]; then
  exit 1
fi
exit 0
`), 0o755)

	counterFile := filepath.Join(dir, "counter")

	cfg := &CommandConfigJSON{
		Argv:            []string{"/bin/sh", scriptPath, counterFile},
		Dir:             dir,
		RestartPolicy:   RestartPolicyOnFailure,
		ScrollbackBytes: 4096,
		CommandDir:      commandDir,
	}

	assert.NilError(t, store.InsertCommandConfig(id, "", cfg))
	assert.NilError(t, cfg.Write())
	assert.NilError(t, store.InsertCommandState(id, StateCreated, &CommandStateJSON{}))

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = RunMonitor(ctx, id, commandDir, dbPath, logger)
	assert.NilError(t, err)

	// Should have exited successfully after 3 runs.
	state, exitCode, _, err := store.GetCommandState(id)
	assert.NilError(t, err)
	assert.Equal(t, state, StateExited)
	assert.Assert(t, exitCode != nil)
	assert.Equal(t, *exitCode, 0)

	// Should have 3 exit history entries.
	history, err := store.GetExitHistory(id)
	assert.NilError(t, err)
	assert.Equal(t, len(history), 3)
}

func TestRestartPolicyAlways(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	commandDir := filepath.Join(dir, "cmd-always")

	store, err := OpenStore(dbPath)
	assert.NilError(t, err)
	defer store.Close()

	id := "test-restart-always"
	cfg := &CommandConfigJSON{
		Argv:            []string{"/bin/sh", "-c", "true"},
		Dir:             dir,
		RestartPolicy:   RestartPolicyAlways,
		ScrollbackBytes: 4096,
		CommandDir:      commandDir,
	}

	assert.NilError(t, store.InsertCommandConfig(id, "", cfg))
	assert.NilError(t, cfg.Write())
	assert.NilError(t, store.InsertCommandState(id, StateCreated, &CommandStateJSON{}))

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// Cancel after a short time to stop the always-restart loop.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = RunMonitor(ctx, id, commandDir, dbPath, logger)
	assert.NilError(t, err)

	// Should have multiple exit history entries.
	history, err := store.GetExitHistory(id)
	assert.NilError(t, err)
	assert.Assert(t, len(history) >= 2, "expected at least 2 restarts, got %d", len(history))
}
