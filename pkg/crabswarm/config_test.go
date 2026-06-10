package crabswarm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ngicks/crabswarm/pkg/crabswarm/hook/exec"
	"github.com/ngicks/crabswarm/pkg/filetype"
	"gotest.tools/v3/assert"
)

// setEnvBaseline clears every env var Load reads. Sub-tests opt back in
// with t.Setenv. Keeps cases hermetic regardless of the caller's shell.
func setEnvBaseline(t *testing.T) {
	t.Helper()
	keys := []string{
		EnvSock,
		EnvConf,
		EnvProjectDir,
		EnvGitRepoBaseDir,
		envXDGRuntime,
		envXDGConfigHome,
		envHome,
	}
	for _, k := range keys {
		t.Setenv(k, "")
	}
}

func TestLoad_FlagsWinOverEnv(t *testing.T) {
	setEnvBaseline(t)
	t.Setenv(EnvSock, "/env/sock")
	t.Setenv(EnvProjectDir, "/env/proj")

	cfg, err := Config{
		Sock:       "/flag/sock",
		ProjectDir: "/flag/proj",
	}.Load()
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, "/flag/sock")
	assert.Equal(t, cfg.ProjectDir, "/flag/proj")
}

func TestLoad_EnvFillsMissingFlags(t *testing.T) {
	setEnvBaseline(t)
	t.Setenv(EnvSock, "/env/sock")
	t.Setenv(EnvProjectDir, "/env/proj")

	cfg, err := Config{}.Load()
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, "/env/sock")
	assert.Equal(t, cfg.ProjectDir, "/env/proj")
}

func TestLoad_DefaultSockPathFromXDGRuntime(t *testing.T) {
	setEnvBaseline(t)
	t.Setenv(envXDGRuntime, "/run/user/1000")

	cfg, err := Config{}.Load()
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, filepath.Join("/run/user/1000", "crabswarm", "default.sock"))
}

func TestLoad_DefaultSockPathFallsBackToTmp(t *testing.T) {
	setEnvBaseline(t)

	cfg, err := Config{}.Load()
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, filepath.Join("/tmp", "crabswarm", "default.sock"))
}

func TestLoad_GitRepoBaseDirFlagWinsOverEnv(t *testing.T) {
	setEnvBaseline(t)
	t.Setenv(EnvGitRepoBaseDir, "/env/repos")

	cfg, err := Config{GitRepoBaseDir: "/flag/repos"}.Load()
	assert.NilError(t, err)
	assert.Equal(t, cfg.GitRepoBaseDir, "/flag/repos")
}

func TestLoad_GitRepoBaseDirFromEnv(t *testing.T) {
	setEnvBaseline(t)
	t.Setenv(EnvGitRepoBaseDir, "/env/repos")

	cfg, err := Config{}.Load()
	assert.NilError(t, err)
	assert.Equal(t, cfg.GitRepoBaseDir, "/env/repos")
}

func TestLoad_GitRepoBaseDirDefaultsToHomeGitrepo(t *testing.T) {
	setEnvBaseline(t)
	t.Setenv(envHome, "/home/someone")

	cfg, err := Config{}.Load()
	assert.NilError(t, err)
	assert.Equal(t, cfg.GitRepoBaseDir, filepath.Join("/home/someone", "gitrepo"))
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := json.Marshal(v)
	assert.NilError(t, err)
	assert.NilError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NilError(t, os.WriteFile(path, raw, 0o644))
}

func TestLoad_ConfigFileFillsEmptyFields(t *testing.T) {
	setEnvBaseline(t)
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")

	writeJSON(t, confPath, Config{
		Sock:       "/file/sock",
		ProjectDir: "/file/proj",
		HookExec: exec.Config{
			Filetypes: []filetype.Config{
				{Ext: map[string]string{"go": "go"}},
			},
		},
	})

	cfg, err := Config{ConfPath: confPath}.Load()
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, "/file/sock")
	assert.Equal(t, cfg.ProjectDir, "/file/proj")
	assert.Equal(t, len(cfg.HookExec.Filetypes), 1)
	assert.Equal(t, cfg.HookExec.Filetypes[0].Ext["go"], "go")
}

func TestLoad_FlagsWinOverFile(t *testing.T) {
	setEnvBaseline(t)
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	writeJSON(t, confPath, Config{Sock: "/file/sock"})

	cfg, err := Config{ConfPath: confPath, Sock: "/flag/sock"}.Load()
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, "/flag/sock")
}

func TestLoad_ConfPathFromEnv(t *testing.T) {
	setEnvBaseline(t)
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	writeJSON(t, confPath, Config{Sock: "/env-file/sock"})

	t.Setenv(EnvConf, confPath)

	cfg, err := Config{}.Load()
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, "/env-file/sock")
	assert.Equal(t, cfg.ConfPath, confPath)
}

func TestLoad_ConfPathFromXDGStdLocation(t *testing.T) {
	setEnvBaseline(t)
	tmp := t.TempDir()
	t.Setenv(envXDGConfigHome, tmp)

	confPath := filepath.Join(tmp, "crabswarm", "config.json")
	writeJSON(t, confPath, Config{Sock: "/xdg-file/sock"})

	cfg, err := Config{}.Load()
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, "/xdg-file/sock")
	assert.Equal(t, cfg.ConfPath, confPath)
}

func TestLoad_ConfPathFromHomeFallback(t *testing.T) {
	setEnvBaseline(t)
	tmp := t.TempDir()
	t.Setenv(envHome, tmp)

	confPath := filepath.Join(tmp, ".config", "crabswarm", "config.json")
	writeJSON(t, confPath, Config{Sock: "/home-file/sock"})

	cfg, err := Config{}.Load()
	assert.NilError(t, err)
	assert.Equal(t, cfg.Sock, "/home-file/sock")
	assert.Equal(t, cfg.ConfPath, confPath)
}

func TestLoad_MissingDefaultFileIsTolerated(t *testing.T) {
	setEnvBaseline(t)
	tmp := t.TempDir()
	t.Setenv(envXDGConfigHome, tmp) // file does not exist under here

	cfg, err := Config{}.Load()
	assert.NilError(t, err)
	// Defaults still applied:
	assert.Equal(t, cfg.Sock, filepath.Join("/tmp", "crabswarm", "default.sock"))
	assert.Equal(t, len(cfg.HookExec.Filetypes), 0)
}

func TestLoad_ExplicitMissingFileErrors(t *testing.T) {
	setEnvBaseline(t)
	cfg, err := Config{ConfPath: "/no/such/file.json"}.Load()
	assert.Assert(t, err != nil)
	assert.Equal(t, cfg.ConfPath, "/no/such/file.json")
}

func TestLoad_InvalidJSONErrors(t *testing.T) {
	setEnvBaseline(t)
	tmp := t.TempDir()
	confPath := filepath.Join(tmp, "config.json")
	assert.NilError(t, os.WriteFile(confPath, []byte("not json"), 0o644))

	_, err := Config{ConfPath: confPath}.Load()
	assert.Assert(t, err != nil)
}
