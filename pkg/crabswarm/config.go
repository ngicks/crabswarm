package crabswarm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v11"

	"github.com/ngicks/crabswarm/pkg/crabswarm/hook/exec"
)

// Config is the materialized configuration the service consumes, after every
// layer (defaults < file < env < flags) is applied. Its fields are value types
// (the merged config is always concrete) and carry both json and yaml tags so
// the `config` subcommand can marshal it and a project can adopt either file
// format without touching fields. The file is NOT decoded into Config —
// PartialConfig is the decode target; Config only ever holds a fully-merged
// result. No omit option here: the `config` subcommand prints the full resolved
// config, so nothing is omitted.
type Config struct {
	// Sock is the Unix-domain socket path used by the crabswarm server and
	// its hook clients.
	Sock string `json:"sock" yaml:"sock"`
	// ProjectDir mirrors $CLAUDE_PROJECT_DIR — the Claude Code project root.
	ProjectDir string `json:"project_dir" yaml:"project_dir"`
	// GitRepoBaseDir is the root under which `crabswarm git clone`
	// materializes repositories.
	GitRepoBaseDir string `json:"git_repo_base_dir" yaml:"git_repo_base_dir"`
	// GitListIgnorePatterns are glob patterns (filepath.Match syntax) that
	// `crabswarm git list` matches against each directory's base name while
	// walking GitRepoBaseDir. A directory whose name matches any pattern is
	// skipped and not descended into, so neither it nor anything beneath it is
	// listed. Empty disables ignoring.
	GitListIgnorePatterns []string `json:"git_list_ignore_patterns" yaml:"git_list_ignore_patterns"`
	// HookExec is the configuration consumed by `crabswarm hook exec`
	// (nested sub-config: deep-merged).
	HookExec exec.Config `json:"hook_exec" yaml:"hook_exec"`
}

// DefaultConfig is the lowest-precedence layer. The path-like defaults are
// derived from the runtime environment (the platform runtime dir for the
// socket, the home dir for the repo base) so they are correct per host; file,
// env, and flag layers override them in turn.
//
// These env reads (XDG_RUNTIME_DIR, the user home dir) are DEFAULT DERIVATION,
// not crabswarm config overrides: they compute a per-host default rather than
// reading a crabswarm-owned variable, so they live here in DefaultConfig and
// are not routed through PartialConfig / caarlos0/env.
func DefaultConfig() Config {
	return Config{
		Sock:           defaultSockPath(),
		GitRepoBaseDir: defaultGitRepoBaseDir(),
	}
}

// PartialConfig is the exported sparse mirror of Config's serialized shape (the
// same keys): a nil/zero field means "absent, leave the lower layer"; a set
// field is an explicit value, including an explicit zero. It is the decode
// target for the config file AND the struct LoadConfig fills via caarlos0/env,
// so file and env merge through one method, Apply. Exported so other code can
// build or inspect partial overrides.
//
// Under this model a present-but-empty env value and an explicit-zero file key
// APPLY (they no longer mean "unset", unlike the previous zero-means-unset
// overlay): nil alone means absent, distinguishable because every scalar is a
// pointer.
//
// Carries three tag sets, kept in sync with Config field-for-field: json + yaml
// for the file decode, and env / envPrefix for caarlos0/env. The CRABSWARM_
// prefix is applied once via envOptions, so the tags hold only the bare names
// (SOCK -> CRABSWARM_SOCK, GIT_REPO_BASE_DIR -> CRABSWARM_GIT_REPO_BASE_DIR).
//
// ProjectDir has NO env tag here: its variable is CLAUDE_PROJECT_DIR — an
// external contract set by Claude Code that must NOT carry the CRABSWARM_
// prefix. caarlos0/env's Prefix option applies to every tagged field, so this
// field cannot ride the prefixed parse; LoadConfig reads it separately (see
// there). It keeps the file (json/yaml) tags so the config file can still set
// project_dir.
//
// GitListIgnorePatterns is a []string: it overwrites wholesale on Apply (a
// non-nil incoming slice replaces the base, a nil one leaves it). Unlike
// HookExec it IS env-shaped, so it carries an env tag — caarlos0/env parses
// CRABSWARM_GIT_LIST_IGNORE_PATTERNS as a comma-separated list.
//
// HookExec is a nested sub-config with no env tags: a []filetype.Config is not
// env-shaped, so it is file-only (env-unsettable), which the skill permits.
//
// JSON tags use ",omitzero" (Go 1.24+) so a marshaled partial stays sparse
// while preserving an explicit empty []/{}; YAML has no omitzero, so its tags
// use ",omitempty".
//
//nolint:lll // triple json/yaml/env tags; one field per line, never wrap tags
type PartialConfig struct {
	Sock                  *string            `json:"sock,omitzero" yaml:"sock,omitempty" env:"SOCK"`
	ProjectDir            *string            `json:"project_dir,omitzero" yaml:"project_dir,omitempty"`
	GitRepoBaseDir        *string            `json:"git_repo_base_dir,omitzero" yaml:"git_repo_base_dir,omitempty" env:"GIT_REPO_BASE_DIR"`
	GitListIgnorePatterns []string           `json:"git_list_ignore_patterns,omitzero" yaml:"git_list_ignore_patterns,omitempty" env:"GIT_LIST_IGNORE_PATTERNS"`
	HookExec              exec.PartialConfig `json:"hook_exec,omitzero" yaml:"hook_exec,omitempty"`
}

// Apply overlays p's present fields onto base and returns the merged Config.
// Merge rules by field kind:
//   - scalar:        non-nil pointer overwrites (explicit zero included).
//   - slice:         non-nil slice overwrites wholesale; nil leaves the base.
//   - nested struct: deep-merged via the sub-partial's Apply — always called; a
//     zero sub-partial merges nothing. Inside it, the Filetypes
//     slice overwrites wholesale when non-nil.
func (p PartialConfig) Apply(base Config) Config {
	if p.Sock != nil {
		base.Sock = *p.Sock
	}
	if p.ProjectDir != nil {
		base.ProjectDir = *p.ProjectDir
	}
	if p.GitRepoBaseDir != nil {
		base.GitRepoBaseDir = *p.GitRepoBaseDir
	}
	if p.GitListIgnorePatterns != nil {
		base.GitListIgnorePatterns = p.GitListIgnorePatterns
	}
	base.HookExec = p.HookExec.Apply(base.HookExec)
	return base
}

// envOptions configures caarlos0/env for the env layer in LoadConfig. The
// variable names live in the env: tags on PartialConfig; the CRABSWARM_ prefix
// is applied here, yielding CRABSWARM_SOCK, CRABSWARM_GIT_REPO_BASE_DIR, etc.
// ProjectDir is deliberately omitted from this parse (see LoadConfig).
var envOptions = env.Options{Prefix: "CRABSWARM_"}

// LoadConfig assembles defaults < config file < environment through Apply. The
// ./cmd layer applies explicitly-set flags on top (flags win). flagPath is the
// --config value ("" when the flag is unset).
//
// The env layer fills a PartialConfig with caarlos0/env: a scalar is set
// (non-nil) only when its variable is present AND non-empty; absent ones — and,
// per caarlos0/env's behavior, present-but-empty ones — stay nil so Apply
// leaves the lower layer untouched. (caarlos0/env's setField skips a value of
// "" unconditionally, so for the prefixed fields a present-but-empty
// CRABSWARM_SOCK= is indistinguishable from an absent one; this is the library's
// behavior, not a custom policy.) ParseWithOptions errors only when a present
// value fails to parse — a hard error that aborts startup. t.Setenv +
// LoadConfig keeps the layer unit-testable.
func LoadConfig(flagPath string) (Config, error) {
	cfg := DefaultConfig()

	path, err := configPath(flagPath)
	if err != nil {
		return cfg, err
	}
	filePartial, err := unmarshalConfigFile(path)
	if err != nil {
		return cfg, err
	}
	cfg = filePartial.Apply(cfg)

	// The prefixed env layer: caarlos0/env applies CRABSWARM_ to every tagged
	// field. ProjectDir carries no env tag, so it is untouched here.
	var envPartial PartialConfig
	if err := env.ParseWithOptions(&envPartial, envOptions); err != nil {
		return cfg, err
	}

	// CLAUDE_PROJECT_DIR is an external contract set by Claude Code and must NOT
	// carry the CRABSWARM_ prefix; caarlos0/env's Prefix applies to every tagged
	// field, so this variable cannot ride the prefixed parse above. It is read
	// by hand with os.LookupEnv — not caarlos0/env — precisely so it stays
	// genuinely present-based: caarlos0/env skips any value of "", which would
	// silently drop a present-but-empty CLAUDE_PROJECT_DIR=, whereas this
	// contract requires that an explicit empty value APPLY (the agent setting
	// the variable to empty is a deliberate "no project dir"). os.LookupEnv
	// distinguishes present-empty (set the pointer to "") from absent (leave it
	// nil, lower layer survives). This is the one env read by hand besides the
	// config-file path — the variable is unprefixed and demands strict presence
	// semantics that the prescribed library cannot express.
	if v, ok := os.LookupEnv(envProjectDirVar); ok {
		envPartial.ProjectDir = &v
	}

	cfg = envPartial.Apply(cfg)

	return cfg, nil
}

// unmarshalConfigFile only reads + decodes; it never merges. It decodes into a
// fresh zero PartialConfig (all nil) and returns the zero value when the file
// does not exist. A non-ENOENT read error or a JSON parse error aborts.
//
// Decoding into a zero value — never a defaults-populated struct — sidesteps
// the v1 encoding/json merge edge cases that decoding into a populated struct
// hits; Apply does the merge afterward.
func unmarshalConfigFile(path string) (PartialConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return PartialConfig{}, nil
		}
		return PartialConfig{}, fmt.Errorf("read config %q: %w", path, err)
	}
	var p PartialConfig
	if err := json.Unmarshal(b, &p); err != nil {
		return PartialConfig{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	return p, nil
}

// envConfVar names the config-file-path override; envProjectDirVar names the
// Claude Code project-dir contract. These are the two env vars read by hand:
// the config path is needed before parsing and is not a Config field, and
// CLAUDE_PROJECT_DIR is unprefixed and needs strict present-based semantics
// caarlos0/env cannot express (see LoadConfig). Every other crabswarm variable
// lives in PartialConfig's env tags. MixedCaps, so no naming-lint directive is
// needed.
const (
	envConfVar       = "CRABSWARM_CONF"
	envProjectDirVar = "CLAUDE_PROJECT_DIR"
)

// configPath resolves the file path: --config (flagPath), else $CRABSWARM_CONF,
// else os.UserConfigDir()/crabswarm/config.json. UserConfigDir already consults
// $XDG_CONFIG_HOME and is platform-native; its error (no config dir resolvable)
// is propagated.
func configPath(flagPath string) (string, error) {
	if flagPath != "" {
		return flagPath, nil
	}
	if p, ok := os.LookupEnv(envConfVar); ok && p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "crabswarm", "config.json"), nil
}

// defaultSockPath derives the default socket path from $XDG_RUNTIME_DIR, falling
// back to /tmp when it is unset. This is default derivation (see DefaultConfig),
// not a config override.
func defaultSockPath() string {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = "/tmp"
	}
	return filepath.Join(runtimeDir, "crabswarm", "default.sock")
}

// defaultGitRepoBaseDir derives $HOME/gitrepo via the platform-native home dir.
// It returns "" when no home dir is resolvable, leaving the field empty. This is
// default derivation (see DefaultConfig), not a config override.
func defaultGitRepoBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, "gitrepo")
}
