package chat

// Config is the persistent configuration of the chat broker — the `chat` block
// of the crabswarm global config. Its fields are value types carrying both json
// and yaml tags so the `config` subcommand can marshal it and a project can
// adopt either file format.
//
// There is no Default in this package, unlike the preview sub-config: every
// field is either host-derived, which only the parent config can do (the
// database path), or already meaningful when empty — "cmdman, resolved on
// PATH", and "no admin key, so no admin RPCs".
type Config struct {
	// Db is the path of the SQLite database holding rooms, members and
	// inboxes. A leading "~" is expanded when the daemon opens the store, so
	// the path stays as written wherever the config is printed back.
	Db string `json:"db" yaml:"db"`
	// CmdmanBin is the cmdman binary the team-info provider shells out to.
	// Empty means "cmdman", resolved on PATH; a non-standard install names an
	// absolute path here.
	CmdmanBin string `json:"cmdman_bin" yaml:"cmdman_bin"`
	// AdminRecipient is the age public key ("age1...") the daemon encrypts
	// admin challenge nonces to — the recipient of the identity file the host
	// operator keeps outside the mounts participants can see. Empty leaves the
	// admin RPCs refusing every call: with no key nothing can prove that
	// possession. A value that does not parse fails daemon startup.
	AdminRecipient string `json:"admin_recipient" yaml:"admin_recipient"`
	// AdminIdentityFile is the path of that age identity file, read by the
	// admin CLI on the host to answer the challenge. It is a client-side
	// setting the daemon never opens; keeping it in the same block is what
	// lets one config file describe both ends. A leading "~" is expanded by
	// the reader.
	AdminIdentityFile string `json:"admin_identity_file" yaml:"admin_identity_file"`
}

// PartialConfig is the sparse mirror of [Config], used by the parent crabswarm
// config's merge layer: a nil field means "absent, leave the lower layer"; a
// non-nil pointer is an explicit value, including an explicit zero. It is
// file-only (no env tags), like the preview sub-config.
//
// JSON tags use ",omitzero" (Go 1.24+) so a marshaled partial stays sparse;
// YAML has no omitzero, so its tags use ",omitempty".
//
//nolint:lll // dual json/yaml tags; one field per line, never wrap tags
type PartialConfig struct {
	Db                *string `json:"db,omitzero" yaml:"db,omitempty"`
	CmdmanBin         *string `json:"cmdman_bin,omitzero" yaml:"cmdman_bin,omitempty"`
	AdminRecipient    *string `json:"admin_recipient,omitzero" yaml:"admin_recipient,omitempty"`
	AdminIdentityFile *string `json:"admin_identity_file,omitzero" yaml:"admin_identity_file,omitempty"`
}

// Apply overlays p's present fields onto base and returns the merged [Config].
// Each field is a scalar: a non-nil pointer overwrites (explicit zero
// included); a nil pointer leaves the base untouched.
func (p PartialConfig) Apply(base Config) Config {
	if p.Db != nil {
		base.Db = *p.Db
	}
	if p.CmdmanBin != nil {
		base.CmdmanBin = *p.CmdmanBin
	}
	if p.AdminRecipient != nil {
		base.AdminRecipient = *p.AdminRecipient
	}
	if p.AdminIdentityFile != nil {
		base.AdminIdentityFile = *p.AdminIdentityFile
	}
	return base
}
