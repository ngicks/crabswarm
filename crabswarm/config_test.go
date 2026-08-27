package crabswarm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ngicks/crabswarm/crabswarm/chat"
	"github.com/ngicks/crabswarm/crabswarm/hook/exec"
	"github.com/ngicks/crabswarm/crabswarm/preview"
	"github.com/ngicks/crabswarm/pkg/filetype"
	"gotest.tools/v3/assert"
)

// Env-var names used by the tests. They mirror the OS environment verbatim;
// CLAUDE_PROJECT_DIR is the unprefixed external contract set by Claude Code.
const (
	envSock           = "CRABSWARM_SOCK"
	envProjectDir     = "CLAUDE_PROJECT_DIR"
	envGitRepoBaseDir = "CRABSWARM_GIT_REPO_BASE_DIR"
	envXDGRuntime     = "XDG_RUNTIME_DIR"
)

// baseEnv unsets every env var LoadConfig and the default resolvers read, so
// cases are hermetic regardless of the caller's shell. t.Setenv registers
// restoration of the original value; the immediate os.Unsetenv removes it for
// the duration of the test.
func baseEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		envSock,
		envConfVar,
		envProjectDir,
		envGitRepoBaseDir,
		envXDGRuntime,
		"XDG_CONFIG_HOME",
		"XDG_STATE_HOME",
		"HOME",
	}
	for _, k := range keys {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
}

// configDir points os.UserConfigDir at a fresh empty temp dir so configPath
// resolves to an absent config.json (ENOENT tolerated). Returns the dir.
func configDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := json.Marshal(v)
	assert.NilError(t, err)
	assert.NilError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NilError(t, os.WriteFile(path, raw, 0o644))
}

func TestLoadConfig_Defaults(t *testing.T) {
	baseEnv(t)
	configDir(t)
	t.Setenv(envXDGRuntime, "/run/user/1000")
	t.Setenv("HOME", "/home/someone")

	cfg, err := LoadConfig("")
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, filepath.Join("/run/user/1000", "crabswarm", "default.sock"))
	assert.Equal(t, cfg.GitRepoBaseDir, filepath.Join("/home/someone", "gitrepo"))
	assert.Equal(t, cfg.ProjectDir, "")
}

func TestLoadConfig_SockDefaultFallsBackToTmp(t *testing.T) {
	baseEnv(t)
	configDir(t) // no XDG_RUNTIME_DIR

	cfg, err := LoadConfig("")
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, filepath.Join("/tmp", "crabswarm", "default.sock"))
}

func TestLoadConfig_EnvOverridesDefault(t *testing.T) {
	baseEnv(t)
	configDir(t)
	t.Setenv(envSock, "/env/sock")
	t.Setenv(envProjectDir, "/env/proj")
	t.Setenv(envGitRepoBaseDir, "/env/repos")

	cfg, err := LoadConfig("")
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, "/env/sock")
	assert.Equal(t, cfg.ProjectDir, "/env/proj")
	assert.Equal(t, cfg.GitRepoBaseDir, "/env/repos")
}

func TestLoadConfig_FileFillsFields(t *testing.T) {
	baseEnv(t)
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	writeJSON(t, confPath, Config{
		Sock:           "/file/sock",
		ProjectDir:     "/file/proj",
		GitRepoBaseDir: "/file/repos",
		HookExec: exec.Config{
			Filetypes: []filetype.Config{
				{Ext: map[string]string{"go": "go"}},
			},
		},
	})

	cfg, err := LoadConfig(confPath)
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, "/file/sock")
	assert.Equal(t, cfg.ProjectDir, "/file/proj")
	assert.Equal(t, cfg.GitRepoBaseDir, "/file/repos")
	assert.Equal(t, len(cfg.HookExec.Filetypes), 1)
	assert.Equal(t, cfg.HookExec.Filetypes[0].Ext["go"], "go")
}

// An omitted key in the sparse file stays nil and must not clobber a lower
// layer — the default here.
func TestLoadConfig_FileSparseKeepsDefault(t *testing.T) {
	baseEnv(t)
	t.Setenv(envXDGRuntime, "/run/user/1000")
	t.Setenv("HOME", "/home/someone")
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	assert.NilError(t, os.WriteFile(confPath, []byte(`{"sock":"/file/sock"}`), 0o644))

	cfg, err := LoadConfig(confPath)
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, "/file/sock")
	// git_repo_base_dir omitted, so the default survives.
	assert.Equal(t, cfg.GitRepoBaseDir, filepath.Join("/home/someone", "gitrepo"))
}

// Under the PartialConfig model an explicit-zero file key APPLIES (it no longer
// means "unset"): a present "sock":"" sets the pointer to an explicit empty
// string, which overwrites the default. This is the intended semantics change.
func TestLoadConfig_FileExplicitZeroApplies(t *testing.T) {
	baseEnv(t)
	t.Setenv(envXDGRuntime, "/run/user/1000")
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	assert.NilError(t, os.WriteFile(confPath, []byte(`{"sock":""}`), 0o644))

	cfg, err := LoadConfig(confPath)
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, "")
}

// A present-but-NON-empty env var applies and wins over the file.
func TestLoadConfig_EnvValueOverridesFile(t *testing.T) {
	baseEnv(t)
	t.Setenv(envSock, "/env/sock")
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	assert.NilError(t, os.WriteFile(confPath, []byte(`{"sock":"/file/sock"}`), 0o644))

	cfg, err := LoadConfig(confPath)
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, "/env/sock")
}

// LIBRARY CONSTRAINT: for the CRABSWARM_-prefixed fields, caarlos0/env's
// setField skips a value of "" unconditionally, so a present-but-empty
// CRABSWARM_SOCK= is indistinguishable from an absent var — the pointer stays
// nil and the lower (file) layer survives. This is the library's behavior, not
// a custom policy, and differs from the unprefixed CLAUDE_PROJECT_DIR handling
// (read via os.LookupEnv, which honors present-empty — see ProjectDir tests).
func TestLoadConfig_EnvEmptyTreatedAsAbsent(t *testing.T) {
	baseEnv(t)
	t.Setenv(envSock, "") // present but empty
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	assert.NilError(t, os.WriteFile(confPath, []byte(`{"sock":"/file/sock"}`), 0o644))

	cfg, err := LoadConfig(confPath)
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, "/file/sock")
}

// CLAUDE_PROJECT_DIR is read unprefixed (no CRABSWARM_ prefix). A present-but-
// empty CLAUDE_PROJECT_DIR= is an explicit empty value that applies, mirroring
// the prefixed env fields' present-based semantics.
func TestLoadConfig_ProjectDirUnprefixed(t *testing.T) {
	baseEnv(t)
	configDir(t)
	// The prefixed name must be ignored for ProjectDir.
	t.Setenv("CRABSWARM_PROJECT_DIR", "/wrong/prefixed")
	t.Setenv(envProjectDir, "/claude/proj")

	cfg, err := LoadConfig("")
	assert.NilError(t, err)
	assert.Equal(t, cfg.ProjectDir, "/claude/proj")
}

// A present-but-empty CLAUDE_PROJECT_DIR= overrides a file-set project_dir,
// staying present-based.
func TestLoadConfig_ProjectDirEmptyOverridesFile(t *testing.T) {
	baseEnv(t)
	t.Setenv(envProjectDir, "") // present but empty
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	assert.NilError(t, os.WriteFile(confPath, []byte(`{"project_dir":"/file/proj"}`), 0o644))

	cfg, err := LoadConfig(confPath)
	assert.NilError(t, err)
	assert.Equal(t, cfg.ProjectDir, "")
}

// An absent CLAUDE_PROJECT_DIR leaves the file/default layer intact (nil ->
// absent, no overwrite).
func TestLoadConfig_ProjectDirAbsentKeepsFile(t *testing.T) {
	baseEnv(t)
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	assert.NilError(t, os.WriteFile(confPath, []byte(`{"project_dir":"/file/proj"}`), 0o644))

	cfg, err := LoadConfig(confPath)
	assert.NilError(t, err)
	assert.Equal(t, cfg.ProjectDir, "/file/proj")
}

func TestLoadConfig_ConfPathFromEnv(t *testing.T) {
	baseEnv(t)
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	assert.NilError(t, os.WriteFile(confPath, []byte(`{"sock":"/env-file/sock"}`), 0o644))
	t.Setenv(envConfVar, confPath)

	cfg, err := LoadConfig("")
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, "/env-file/sock")
}

func TestLoadConfig_ConfPathFromUserConfigDir(t *testing.T) {
	baseEnv(t)
	dir := configDir(t)
	confPath := filepath.Join(dir, "crabswarm", "config.json")
	assert.NilError(t, os.MkdirAll(filepath.Dir(confPath), 0o755))
	assert.NilError(t, os.WriteFile(confPath, []byte(`{"sock":"/xdg-file/sock"}`), 0o644))

	cfg, err := LoadConfig("")
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, "/xdg-file/sock")
}

func TestLoadConfig_MissingFileTolerated(t *testing.T) {
	baseEnv(t)
	configDir(t) // empty dir, no config.json

	cfg, err := LoadConfig("")
	assert.NilError(t, err)
	// Defaults still applied.
	assert.Equal(t, cfg.Sock, filepath.Join("/tmp", "crabswarm", "default.sock"))
	assert.Equal(t, len(cfg.HookExec.Filetypes), 0)
}

// ENOENT is tolerated even for an explicit --config path (the canonical
// unmarshalConfigFile contract); a typo'd path falls through to defaults rather
// than erroring.
func TestLoadConfig_ExplicitMissingFileTolerated(t *testing.T) {
	baseEnv(t)
	configDir(t)

	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "absent.json"))
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, filepath.Join("/tmp", "crabswarm", "default.sock"))
}

func TestLoadConfig_InvalidJSONErrors(t *testing.T) {
	baseEnv(t)
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	assert.NilError(t, os.WriteFile(confPath, []byte("not json"), 0o644))

	_, err := LoadConfig(confPath)
	assert.Assert(t, err != nil)
}

// With no --config, no $CRABSWARM_CONF, and no HOME/XDG_CONFIG_HOME,
// os.UserConfigDir cannot resolve a directory and LoadConfig propagates the
// error.
func TestLoadConfig_UnresolvableConfigDirErrors(t *testing.T) {
	baseEnv(t)

	_, err := LoadConfig("")
	assert.Assert(t, err != nil)
}

// The nested HookExec sub-config deep-merges: env carries no HookExec field, so
// the file's Filetypes survives the env layer.
func TestLoadConfig_HookExecFromFile(t *testing.T) {
	baseEnv(t)
	t.Setenv(envSock, "/env/sock") // env present, but does not touch HookExec
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	writeJSON(t, confPath, Config{
		HookExec: exec.Config{
			Filetypes: []filetype.Config{
				{Ext: map[string]string{"go": "go"}},
			},
		},
	})

	cfg, err := LoadConfig(confPath)
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, "/env/sock")
	assert.Equal(t, len(cfg.HookExec.Filetypes), 1)
	assert.Equal(t, cfg.HookExec.Filetypes[0].Ext["go"], "go")
}

// git_list_ignore_patterns is a file-settable []string; a present value loads
// into the resolved config.
func TestLoadConfig_GitListIgnorePatternsFromFile(t *testing.T) {
	baseEnv(t)
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	writeJSON(t, confPath, Config{
		GitListIgnorePatterns: []string{"node_modules", "tmp*"},
	})

	cfg, err := LoadConfig(confPath)
	assert.NilError(t, err)
	assert.DeepEqual(t, cfg.GitListIgnorePatterns, []string{"node_modules", "tmp*"})
}

// CRABSWARM_GIT_LIST_IGNORE_PATTERNS is parsed by caarlos0/env as a
// comma-separated list and overrides the file layer.
func TestLoadConfig_GitListIgnorePatternsFromEnv(t *testing.T) {
	baseEnv(t)
	configDir(t)
	t.Setenv("CRABSWARM_GIT_LIST_IGNORE_PATTERNS", "vendor,node_modules")

	cfg, err := LoadConfig("")
	assert.NilError(t, err)
	assert.DeepEqual(t, cfg.GitListIgnorePatterns, []string{"vendor", "node_modules"})
}

// PartialConfig.Apply: a non-nil GitListIgnorePatterns slice overwrites the
// base wholesale; a nil slice leaves the base.
func TestApply_GitListIgnorePatternsSliceOverwrite(t *testing.T) {
	base := Config{GitListIgnorePatterns: []string{"a", "b"}}

	overwrite := PartialConfig{GitListIgnorePatterns: []string{"c"}}
	got := overwrite.Apply(base)
	assert.DeepEqual(t, got.GitListIgnorePatterns, []string{"c"})

	keep := PartialConfig{}
	got = keep.Apply(base)
	assert.DeepEqual(t, got.GitListIgnorePatterns, []string{"a", "b"})
}

// PartialConfig.Apply: a non-nil Filetypes slice overwrites the base wholesale
// (no element-wise merge); a nil slice leaves the base.
func TestApply_FiletypesSliceOverwrite(t *testing.T) {
	base := Config{
		HookExec: exec.Config{
			Filetypes: []filetype.Config{
				{Ext: map[string]string{"go": "go"}},
				{Ext: map[string]string{"rs": "rust"}},
			},
		},
	}

	// Non-nil incoming slice replaces wholesale.
	overwrite := PartialConfig{
		HookExec: exec.PartialConfig{
			Filetypes: []filetype.Config{
				{Ext: map[string]string{"py": "python"}},
			},
		},
	}
	got := overwrite.Apply(base)
	assert.Equal(t, len(got.HookExec.Filetypes), 1)
	assert.Equal(t, got.HookExec.Filetypes[0].Ext["py"], "python")

	// nil incoming slice leaves the base untouched.
	keep := PartialConfig{}
	got = keep.Apply(base)
	assert.Equal(t, len(got.HookExec.Filetypes), 2)
}

// Apply is pure with respect to its base: it returns a new Config and does not
// mutate the base value or its nested slice.
func TestApply_PurityBaseNotMutated(t *testing.T) {
	base := Config{
		Sock: "/base/sock",
		HookExec: exec.Config{
			Filetypes: []filetype.Config{
				{Ext: map[string]string{"go": "go"}},
			},
		},
	}

	newSock := "/new/sock"
	p := PartialConfig{
		Sock: &newSock,
		HookExec: exec.PartialConfig{
			Filetypes: []filetype.Config{
				{Ext: map[string]string{"rs": "rust"}},
			},
		},
	}
	got := p.Apply(base)

	// The result reflects the overlay.
	assert.Equal(t, got.Sock, "/new/sock")
	assert.Equal(t, got.HookExec.Filetypes[0].Ext["rs"], "rust")

	// The base is unchanged.
	assert.Equal(t, base.Sock, "/base/sock")
	assert.Equal(t, len(base.HookExec.Filetypes), 1)
	assert.Equal(t, base.HookExec.Filetypes[0].Ext["go"], "go")
}

// With no file and no env, the preview sub-config resolves to its built-in
// defaults (preview.Default).
func TestLoadConfig_PreviewDefaults(t *testing.T) {
	baseEnv(t)
	configDir(t)

	cfg, err := LoadConfig("")
	assert.NilError(t, err)
	assert.Equal(t, cfg.Preview.Addr, "0.0.0.0:6419")
	assert.Equal(t, cfg.Preview.DaemonName, "crabswarm-preview")
}

// A full preview section in the file overrides both scalar defaults.
func TestLoadConfig_PreviewFromFile(t *testing.T) {
	baseEnv(t)
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	assert.NilError(
		t,
		os.WriteFile(
			confPath,
			[]byte(`{"preview":{"addr":"127.0.0.1:9999","daemon_name":"my-preview"}}`),
			0o644,
		),
	)

	cfg, err := LoadConfig(confPath)
	assert.NilError(t, err)
	assert.Equal(t, cfg.Preview.Addr, "127.0.0.1:9999")
	assert.Equal(t, cfg.Preview.DaemonName, "my-preview")
}

// A sparse preview section overlays field-by-field: an omitted daemon_name
// keeps the default while addr is overridden.
func TestLoadConfig_PreviewSparseKeepsDefault(t *testing.T) {
	baseEnv(t)
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	assert.NilError(
		t,
		os.WriteFile(confPath, []byte(`{"preview":{"addr":"127.0.0.1:9999"}}`), 0o644),
	)

	cfg, err := LoadConfig(confPath)
	assert.NilError(t, err)
	assert.Equal(t, cfg.Preview.Addr, "127.0.0.1:9999")
	assert.Equal(t, cfg.Preview.DaemonName, "crabswarm-preview")
}

// PartialConfig.Apply: the preview sub-partial deep-merges field-by-field —
// a non-nil pointer overwrites, a nil one leaves the base — and does not
// mutate the base.
func TestApply_PreviewOverlay(t *testing.T) {
	base := Config{
		Preview: preview.Config{Addr: "0.0.0.0:6419", DaemonName: "crabswarm-preview"},
	}

	addr := "127.0.0.1:9999"
	overlay := PartialConfig{
		Preview: preview.PartialConfig{Addr: &addr},
	}
	got := overlay.Apply(base)
	assert.Equal(t, got.Preview.Addr, "127.0.0.1:9999")
	assert.Equal(t, got.Preview.DaemonName, "crabswarm-preview")

	// The base is unchanged.
	assert.Equal(t, base.Preview.Addr, "0.0.0.0:6419")

	// A zero sub-partial merges nothing.
	keep := PartialConfig{}
	got = keep.Apply(base)
	assert.Equal(t, got.Preview.Addr, "0.0.0.0:6419")
	assert.Equal(t, got.Preview.DaemonName, "crabswarm-preview")
}

// The chat database path is derived from $XDG_STATE_HOME.
func TestLoadConfig_ChatDbDefault(t *testing.T) {
	baseEnv(t)
	configDir(t)
	t.Setenv("XDG_STATE_HOME", "/state")

	cfg, err := LoadConfig("")
	assert.NilError(t, err)
	assert.Equal(t, cfg.Chat.Db, filepath.Join("/state", "crabswarm", "chat.db"))
	assert.Equal(t, cfg.Chat.CmdmanBin, "")
}

// Without XDG_STATE_HOME the default follows the XDG fallback under $HOME.
func TestLoadConfig_ChatDbDefaultFallsBackToHome(t *testing.T) {
	baseEnv(t)
	configDir(t)
	t.Setenv("HOME", "/home/someone")

	cfg, err := LoadConfig("")
	assert.NilError(t, err)
	assert.Equal(
		t,
		cfg.Chat.Db,
		filepath.Join("/home/someone", ".local", "state", "crabswarm", "chat.db"),
	)
}

// A chat section in the file overrides the derived default.
func TestLoadConfig_ChatFromFile(t *testing.T) {
	baseEnv(t)
	t.Setenv("XDG_STATE_HOME", "/state")
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	assert.NilError(
		t,
		os.WriteFile(
			confPath,
			[]byte(`{"chat":{"db":"/file/chat.db","cmdman_bin":"/opt/bin/cmdman"}}`),
			0o644,
		),
	)

	cfg, err := LoadConfig(confPath)
	assert.NilError(t, err)
	assert.Equal(t, cfg.Chat.Db, "/file/chat.db")
	assert.Equal(t, cfg.Chat.CmdmanBin, "/opt/bin/cmdman")
}

// PartialConfig.Apply: the chat sub-partial deep-merges field-by-field — a
// non-nil pointer overwrites, a nil one leaves the base — and does not mutate
// the base.
func TestApply_ChatOverlay(t *testing.T) {
	base := Config{
		Chat: chat.Config{Db: "/state/crabswarm/chat.db"},
	}

	db := "/elsewhere/chat.db"
	overlay := PartialConfig{Chat: chat.PartialConfig{Db: &db}}
	got := overlay.Apply(base)
	assert.Equal(t, got.Chat.Db, "/elsewhere/chat.db")
	assert.Equal(t, got.Chat.CmdmanBin, "")

	// The base is unchanged.
	assert.Equal(t, base.Chat.Db, "/state/crabswarm/chat.db")

	// A zero sub-partial merges nothing.
	keep := PartialConfig{}
	got = keep.Apply(base)
	assert.Equal(t, got.Chat.Db, "/state/crabswarm/chat.db")
}
