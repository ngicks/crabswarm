package mcp

// Config is the persistent configuration of `crabswarm mcp` and its
// subcommands — the `mcp` block of the crabswarm global config. Its fields are
// value types carrying both json and yaml tags so the `config` subcommand can
// marshal it and a project can adopt either file format.
//
// There is no Default in this package, unlike the preview sub-config: the one
// field is host-derived, which only the parent config can do.
type Config struct {
	// CodexProxySockDir is the directory `crabswarm mcp codex-proxy` serves its
	// TUI's socket in, as <pid>.sock. The proxy creates it and makes it private
	// to its owner. A container mounts the crabswarm runtime directory
	// read-only, so the default is a directory of its own beside it.
	CodexProxySockDir string `json:"codex_proxy_sock_dir" yaml:"codex_proxy_sock_dir"`
}

// PartialConfig is the sparse mirror of [Config], used by the parent crabswarm
// config's merge layer: a nil field means "absent, leave the lower layer"; a
// non-nil pointer is an explicit value, including an explicit zero.
//
// The env tags hold only the bare names: the parent's MCP field carries
// envPrefix:"MCP_" and the parent's env parse applies CRABSWARM_ globally, so
// caarlos0/env composes both onto each name (CODEX_PROXY_SOCK_DIR ->
// CRABSWARM_MCP_CODEX_PROXY_SOCK_DIR).
//
// JSON tags use ",omitzero" (Go 1.24+) so a marshaled partial stays sparse;
// YAML has no omitzero, so its tags use ",omitempty".
//
//nolint:lll // triple json/yaml/env tags; one field per line, never wrap tags
type PartialConfig struct {
	CodexProxySockDir *string `json:"codex_proxy_sock_dir,omitzero" yaml:"codex_proxy_sock_dir,omitempty" env:"CODEX_PROXY_SOCK_DIR"`
}

// Apply overlays p's present fields onto base and returns the merged [Config].
// A non-nil pointer overwrites (explicit zero included); a nil pointer leaves
// the base untouched.
func (p PartialConfig) Apply(base Config) Config {
	if p.CodexProxySockDir != nil {
		base.CodexProxySockDir = *p.CodexProxySockDir
	}
	return base
}
