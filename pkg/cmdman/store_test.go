package cmdman

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := OpenStore(dbPath, true)
	assert.NilError(t, err)
	t.Cleanup(func() { store.Close() })
	return store
}

func TestSchemaCreation(t *testing.T) {
	store := testStore(t)

	// Verify tables exist by querying them.
	var count int
	err := store.db.QueryRow(`SELECT count(*) FROM CommandConfig`).Scan(&count)
	assert.NilError(t, err)
	assert.Equal(t, count, 0)

	err = store.db.QueryRow(`SELECT count(*) FROM CommandState`).Scan(&count)
	assert.NilError(t, err)

	err = store.db.QueryRow(`SELECT count(*) FROM CommandExitCode`).Scan(&count)
	assert.NilError(t, err)
}

func TestExitCodeRangeCheck(t *testing.T) {
	store := testStore(t)

	cfg := &CommandConfigJSON{Argv: []string{"/bin/true"}, CommandDir: "/tmp/test"}
	assert.NilError(t, store.InsertCommandConfig("test-1", "", cfg))
	assert.NilError(t, store.InsertCommandState("test-1", StateCreated, &CommandStateJSON{}))

	// Valid exit codes: -1 to 255.
	ec := 0
	assert.NilError(t, store.UpdateCommandState("test-1", StateExited, &ec, &CommandStateJSON{}))
	ec = 255
	assert.NilError(t, store.UpdateCommandState("test-1", StateExited, &ec, &CommandStateJSON{}))
	ec = -1
	assert.NilError(t, store.UpdateCommandState("test-1", StateExited, &ec, &CommandStateJSON{}))

	// Invalid exit codes.
	ec = 256
	err := store.UpdateCommandState("test-1", StateExited, &ec, &CommandStateJSON{})
	assert.Assert(t, err != nil, "exit code 256 should be rejected")

	ec = -2
	err = store.UpdateCommandState("test-1", StateExited, &ec, &CommandStateJSON{})
	assert.Assert(t, err != nil, "exit code -2 should be rejected")
}

func TestDeferredForeignKey(t *testing.T) {
	store := testStore(t)

	// Insert state before config (deferred FK allows this within a transaction).
	tx, err := store.db.Begin()
	assert.NilError(t, err)

	_, err = tx.Exec(`INSERT INTO CommandState (ID, State, JSON) VALUES (?, ?, ?)`,
		"deferred-1", StateCreated, "{}")
	assert.NilError(t, err)

	_, err = tx.Exec(`INSERT INTO CommandConfig (ID, Name, JSON) VALUES (?, ?, ?)`,
		"deferred-1", nil, `{"argv":["/bin/true"],"command_dir":"/tmp/test","restart_policy":"no","scrollback_bytes":1048576}`)
	assert.NilError(t, err)

	assert.NilError(t, tx.Commit())
}

func TestInsertAndGetCommandConfig(t *testing.T) {
	store := testStore(t)

	cfg := &CommandConfigJSON{
		Argv:            []string{"/bin/bash", "-c", "echo hello"},
		Dir:             "/tmp",
		RestartPolicy:   RestartPolicyNo,
		ScrollbackBytes: DefaultScrollbackBytes,
		Labels:          map[string]string{"app": "test", "env": "dev"},
		CommandDir:      "/tmp/cmd/test-1",
	}

	assert.NilError(t, store.InsertCommandConfig("test-1", "mycommand", cfg))

	id, name, got, err := store.GetCommandConfig("test-1")
	assert.NilError(t, err)
	assert.Equal(t, id, "test-1")
	assert.Equal(t, name, "mycommand")
	assert.Equal(t, got.Argv[0], "/bin/bash")

	// Lookup by name.
	id2, _, _, err := store.GetCommandConfig("mycommand")
	assert.NilError(t, err)
	assert.Equal(t, id2, "test-1")
}

func TestListCommandsWithLabels(t *testing.T) {
	store := testStore(t)

	cfg1 := &CommandConfigJSON{
		Argv:            []string{"/bin/true"},
		RestartPolicy:   RestartPolicyNo,
		ScrollbackBytes: DefaultScrollbackBytes,
		Labels:          map[string]string{"app": "web", "env": "prod"},
		CommandDir:      "/tmp/cmd/1",
	}
	cfg2 := &CommandConfigJSON{
		Argv:            []string{"/bin/true"},
		RestartPolicy:   RestartPolicyNo,
		ScrollbackBytes: DefaultScrollbackBytes,
		Labels:          map[string]string{"app": "api", "env": "prod"},
		CommandDir:      "/tmp/cmd/2",
	}

	assert.NilError(t, store.InsertCommandConfig("id-1", "web", cfg1))
	assert.NilError(t, store.InsertCommandState("id-1", StateRunning, &CommandStateJSON{}))
	assert.NilError(t, store.InsertCommandConfig("id-2", "api", cfg2))
	assert.NilError(t, store.InsertCommandState("id-2", StateRunning, &CommandStateJSON{}))

	// Filter by label.
	entries, err := store.ListCommands(true, map[string]string{"app": "web"})
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 1)
	assert.Equal(t, entries[0].ID, "id-1")

	// Filter by env=prod should return both.
	entries, err = store.ListCommands(true, map[string]string{"env": "prod"})
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 2)
}

func TestDeleteCommand(t *testing.T) {
	store := testStore(t)

	cfg := &CommandConfigJSON{
		Argv:            []string{"/bin/true"},
		RestartPolicy:   RestartPolicyNo,
		ScrollbackBytes: DefaultScrollbackBytes,
		CommandDir:      "/tmp/cmd/del-1",
	}
	assert.NilError(t, store.InsertCommandConfig("del-1", "", cfg))
	assert.NilError(t, store.InsertCommandState("del-1", StateExited, &CommandStateJSON{}))
	assert.NilError(t, store.InsertCommandExitCode("del-1", 0))

	assert.NilError(t, store.DeleteCommand("del-1"))

	_, err := store.ResolveID("del-1")
	assert.Assert(t, err != nil, "should not find deleted command")
}

func TestConfigJSONMaterialization(t *testing.T) {
	dir := t.TempDir()
	commandDir := filepath.Join(dir, "cmd-1")

	cfg := &CommandConfigJSON{
		Argv:            []string{"/bin/echo", "hello"},
		Dir:             "/tmp",
		RestartPolicy:   RestartPolicyNo,
		ScrollbackBytes: DefaultScrollbackBytes,
		CommandDir:      commandDir,
	}

	assert.NilError(t, cfg.Write())

	// Verify file was written.
	data, err := os.ReadFile(filepath.Join(commandDir, "config.json"))
	assert.NilError(t, err)
	assert.Assert(t, len(data) > 0)

	// Read it back.
	got, err := ReadCommandConfig(commandDir)
	assert.NilError(t, err)
	assert.Equal(t, got.Argv[0], "/bin/echo")
	assert.Equal(t, got.Dir, "/tmp")
}
