// Package crabswarm exposes the unified configuration for the crabswarm CLI
// and its supporting services. All env-var reads and configuration-file
// unmarshaling live here so that code under ./cmd stays free of os.Getenv
// calls and ad-hoc file I/O.
package crabswarm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ngicks/crabswarm/pkg/crabswarm/hook/exec"
	"github.com/ngicks/crabswarm/pkg/filetype"
)

// Env-var names recognized by this package.
const (
	EnvSock           = "CRABSWARM_SOCK"
	EnvConf           = "CRABSWARM_CONF"
	EnvProjectDir     = "CLAUDE_PROJECT_DIR"
	EnvGitRepoBaseDir = "CRABSWARM_GIT_REPO_BASE_DIR"
	envXDGRuntime     = "XDG_RUNTIME_DIR"
	envXDGConfigHome  = "XDG_CONFIG_HOME"
	envHome           = "HOME"
)

// Config is the unified configuration for the crabswarm CLI and its
// services. Callers under ./cmd build a partial Config from CLI flags and
// call Load on it, which fills the remaining fields from environment
// variables and the configuration file.
type Config struct {
	// ConfPath is the path to the configuration file. When empty, Load
	// resolves it from $CRABSWARM_CONF or the std XDG location. After Load
	// returns, this field carries the path that was actually consulted (or
	// "" when none).
	ConfPath string `json:"-"`
	// Sock is the Unix-domain socket path used by the crabswarm server and
	// its hook clients.
	Sock string `json:"sock,omitzero"`
	// ProjectDir mirrors $CLAUDE_PROJECT_DIR — the Claude Code project root.
	ProjectDir string `json:"project_dir,omitzero"`
	// GitRepoBaseDir is the root under which `crabswarm git clone`
	// materializes repositories. Resolved from
	// $CRABSWARM_GIT_REPO_BASE_DIR, the config file, then $HOME/gitrepo.
	GitRepoBaseDir string `json:"git_repo_base_dir,omitzero"`
	// HookExec is the configuration consumed by `crabswarm hook exec`.
	HookExec exec.Config `json:"hook_exec,omitzero"`
}

// Load returns a Config assembled from (highest priority first):
//  1. fields already set on the receiver (typically from CLI flags),
//  2. environment variables ($CRABSWARM_SOCK, $CRABSWARM_CONF,
//     $CLAUDE_PROJECT_DIR, $CRABSWARM_GIT_REPO_BASE_DIR),
//  3. the configuration file at c.ConfPath, $CRABSWARM_CONF, or
//     ${XDG_CONFIG_HOME:-$HOME/.config}/crabswarm/config.json,
//  4. built-in defaults for path-like fields (e.g. $HOME/gitrepo for
//     GitRepoBaseDir).
//
// A missing default-resolved config file is tolerated (treated as empty),
// but if c.ConfPath was explicitly set the file must exist.
func (c Config) Load() (Config, error) {
	cfg := c

	if cfg.Sock == "" {
		cfg.Sock = os.Getenv(EnvSock)
	}
	if cfg.ProjectDir == "" {
		cfg.ProjectDir = os.Getenv(EnvProjectDir)
	}
	if cfg.GitRepoBaseDir == "" {
		cfg.GitRepoBaseDir = os.Getenv(EnvGitRepoBaseDir)
	}

	confFromFlag := cfg.ConfPath != ""
	if cfg.ConfPath == "" {
		cfg.ConfPath = os.Getenv(EnvConf)
	}
	if cfg.ConfPath == "" {
		cfg.ConfPath = defaultConfPath()
	}
	if cfg.ConfPath != "" {
		fileCfg, err := readConfigFile(cfg.ConfPath)
		switch {
		case err == nil:
			cfg = merge(cfg, fileCfg)
		case confFromFlag || !errors.Is(err, fs.ErrNotExist):
			return cfg, err
		}
	}

	if cfg.Sock == "" {
		cfg.Sock = defaultSockPath()
	}
	if cfg.GitRepoBaseDir == "" {
		cfg.GitRepoBaseDir = defaultGitRepoBaseDir()
	}

	return cfg, nil
}

// merge fills empty fields of primary from secondary. For HookExec, the
// embedded filetype.Config is merged key-by-key with primary winning on
// duplicates (so flag/env-supplied entries override file entries).
func merge(primary, secondary Config) Config {
	out := primary
	if out.Sock == "" {
		out.Sock = secondary.Sock
	}
	if out.ProjectDir == "" {
		out.ProjectDir = secondary.ProjectDir
	}
	if out.GitRepoBaseDir == "" {
		out.GitRepoBaseDir = secondary.GitRepoBaseDir
	}
	// Concatenate file-loaded leaves first, then flag/env leaves; later
	// leaves win on conflict, so flag/env entries override file entries.
	out.HookExec.Filetypes = append(
		append([]filetype.Config(nil), secondary.HookExec.Filetypes...),
		primary.HookExec.Filetypes...,
	)
	return out
}

func readConfigFile(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config %q: %w", path, err)
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("parsing config %q: %w", path, err)
	}
	return c, nil
}

func defaultSockPath() string {
	runtimeDir := os.Getenv(envXDGRuntime)
	if runtimeDir == "" {
		runtimeDir = "/tmp"
	}
	return filepath.Join(runtimeDir, "crabswarm", "default.sock")
}

func defaultGitRepoBaseDir() string {
	home := os.Getenv(envHome)
	if home == "" {
		return ""
	}
	return filepath.Join(home, "gitrepo")
}

func defaultConfPath() string {
	configHome := os.Getenv(envXDGConfigHome)
	if configHome == "" {
		home := os.Getenv(envHome)
		if home == "" {
			return ""
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "crabswarm", "config.json")
}
