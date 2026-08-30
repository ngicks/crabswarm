// Package preview implements `crabswarm preview`: a browser-based
// GitHub-flavored markdown previewer served over TCP so phones and tablets
// on the tailnet can reach it. This file holds only the persistent
// configuration — the `preview` block of the crabswarm global config.
package preview

// Default listen address and cmdman command name. Exposed through [Default]
// as the lowest-precedence layer; file/env/flag layers override them in turn.
const (
	defaultAddr       = "0.0.0.0:6419"
	defaultDaemonName = "crabswarm-preview"
)

// Config is the persistent configuration for the previewer — it's the
// `preview` block of the crabswarm global config. Its fields are value
// types carrying both json and yaml tags so the `config` subcommand can
// marshal it and a project can adopt either file format.
type Config struct {
	// Addr is the TCP listen address of the preview HTTP server. Default
	// "0.0.0.0:6419" — reachable over the tailnet by design.
	Addr string `json:"addr" yaml:"addr"`
	// DaemonName is the cmdman command name the previewer runs under.
	// Default "crabswarm-preview".
	DaemonName string `json:"daemon_name" yaml:"daemon_name"`
}

// PartialConfig is the sparse mirror of [Config], used by the parent
// crabswarm config's merge layer: a nil field means "absent, leave the
// lower layer"; a non-nil pointer is an explicit value, including an
// explicit zero.
//
// The env tags hold only the bare names: the parent's Preview field carries
// envPrefix:"PREVIEW_" and the parent's env parse applies CRABSWARM_
// globally, so caarlos0/env composes both onto each name (ADDR ->
// CRABSWARM_PREVIEW_ADDR, DAEMON_NAME -> CRABSWARM_PREVIEW_DAEMON_NAME).
//
// JSON tags use ",omitzero" (Go 1.24+) so a marshaled partial stays sparse;
// YAML has no omitzero, so its tags use ",omitempty".
//
//nolint:lll // triple json/yaml/env tags; one field per line, never wrap tags
type PartialConfig struct {
	Addr       *string `json:"addr,omitzero" yaml:"addr,omitempty" env:"ADDR"`
	DaemonName *string `json:"daemon_name,omitzero" yaml:"daemon_name,omitempty" env:"DAEMON_NAME"`
}

// Apply overlays p's present fields onto base and returns the merged
// Config. Each field is a scalar: a non-nil pointer overwrites (explicit
// zero included); a nil pointer leaves the base untouched.
func (p PartialConfig) Apply(base Config) Config {
	if p.Addr != nil {
		base.Addr = *p.Addr
	}
	if p.DaemonName != nil {
		base.DaemonName = *p.DaemonName
	}
	return base
}

// Default returns a fresh copy of the built-in configuration: the tailnet
// listen address and the cmdman command name. The parent crabswarm config
// layers Default underneath the file/env/flag layers so the built-ins apply
// out of the box and caller-supplied values win.
func Default() Config {
	return Config{
		Addr:       defaultAddr,
		DaemonName: defaultDaemonName,
	}
}
