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

func TestMonitorRunAndExit(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	commandDir := filepath.Join(dir, "cmd-1")

	store, err := OpenStore(dbPath)
	assert.NilError(t, err)
	defer store.Close()

	id := "test-monitor-1"
	cfg := &CommandConfigJSON{
		Argv:            []string{"/bin/sh", "-c", "echo hello from monitor"},
		Dir:             dir,
		RestartPolicy:   RestartPolicyNo,
		ScrollbackBytes: 4096,
		CommandDir:      commandDir,
	}

	assert.NilError(t, store.InsertCommandConfig(id, "test-echo", cfg))
	assert.NilError(t, cfg.Write())
	assert.NilError(t, store.InsertCommandState(id, StateCreated, &CommandStateJSON{}))

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Run monitor synchronously (in this test process, not detached).
	err = RunMonitor(ctx, id, commandDir, dbPath, logger)
	assert.NilError(t, err)

	// Verify final state.
	state, exitCode, _, err := store.GetCommandState(id)
	assert.NilError(t, err)
	assert.Equal(t, state, StateExited)
	assert.Assert(t, exitCode != nil)
	assert.Equal(t, *exitCode, 0)

	// Verify exit history.
	history, err := store.GetExitHistory(id)
	assert.NilError(t, err)
	assert.Assert(t, len(history) > 0)
	assert.Equal(t, history[0].ExitCode, 0)
}

func TestMonitorNonZeroExit(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	commandDir := filepath.Join(dir, "cmd-2")

	store, err := OpenStore(dbPath)
	assert.NilError(t, err)
	defer store.Close()

	id := "test-monitor-2"
	cfg := &CommandConfigJSON{
		Argv:            []string{"/bin/sh", "-c", "exit 42"},
		Dir:             dir,
		RestartPolicy:   RestartPolicyNo,
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

	state, exitCode, _, err := store.GetCommandState(id)
	assert.NilError(t, err)
	assert.Equal(t, state, StateExited)
	assert.Assert(t, exitCode != nil)
	assert.Equal(t, *exitCode, 42)
}

func TestMonitorAutoRemove(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	commandDir := filepath.Join(dir, "cmd-3")

	store, err := OpenStore(dbPath)
	assert.NilError(t, err)
	defer store.Close()

	id := "test-monitor-3"
	cfg := &CommandConfigJSON{
		Argv:            []string{"/bin/sh", "-c", "true"},
		Dir:             dir,
		RestartPolicy:   RestartPolicyNo,
		ScrollbackBytes: 4096,
		Annotations:     map[string]string{AnnotationAutoRemove: "true"},
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

	// Command should be auto-removed.
	_, resolveErr := store.ResolveID(id)
	assert.Assert(t, resolveErr != nil, "command should be removed")

	// Command dir should be removed.
	_, err = os.Stat(commandDir)
	assert.Assert(t, os.IsNotExist(err), "command dir should be removed")
}

func TestMonitorGracefulShutdown(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	commandDir := filepath.Join(dir, "cmd-4")

	store, err := OpenStore(dbPath)
	assert.NilError(t, err)
	defer store.Close()

	id := "test-monitor-4"
	cfg := &CommandConfigJSON{
		Argv:            []string{"/bin/sh", "-c", "sleep 60"},
		Dir:             dir,
		RestartPolicy:   RestartPolicyNo,
		ScrollbackBytes: 4096,
		CommandDir:      commandDir,
	}

	assert.NilError(t, store.InsertCommandConfig(id, "", cfg))
	assert.NilError(t, cfg.Write())
	assert.NilError(t, store.InsertCommandState(id, StateCreated, &CommandStateJSON{}))

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Cancel after a short delay to simulate SIGTERM.
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	err = RunMonitor(ctx, id, commandDir, dbPath, logger)
	// Should exit with an error from the killed process.
	// The monitor should handle this gracefully.
	assert.NilError(t, err)

	state, _, _, err := store.GetCommandState(id)
	assert.NilError(t, err)
	assert.Equal(t, state, StateExited)
}

func TestStaleEntryCleanup(t *testing.T) {
	store := testStore(t)

	cfg := &CommandConfigJSON{
		Argv:            []string{"/bin/true"},
		RestartPolicy:   RestartPolicyNo,
		ScrollbackBytes: DefaultScrollbackBytes,
		CommandDir:      "/tmp/cmd/stale-1",
	}
	assert.NilError(t, store.InsertCommandConfig("stale-1", "", cfg))
	// Set a PID that's definitely not alive (PID 1 is init, but use a very high PID).
	stateJSON := &CommandStateJSON{MonitorPID: 99999999}
	assert.NilError(t, store.InsertCommandState("stale-1", StateRunning, stateJSON))

	assert.NilError(t, CleanStaleEntries(store))

	state, _, _, err := store.GetCommandState("stale-1")
	assert.NilError(t, err)
	assert.Equal(t, state, StateFailed)
}
