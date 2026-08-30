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
	// AdminRecipients are the age public keys ("age1...") the daemon encrypts
	// admin challenge nonces to — the recipients of the identity files the host
	// operators keep outside the mounts participants can see. Every challenge
	// goes to all of them, so the holder of any one listed key's identity can
	// answer it. An empty list leaves the admin RPCs refusing every call: with
	// no key nothing can prove that possession. A key that does not parse fails
	// daemon startup.
	AdminRecipients []string `json:"admin_recipients" yaml:"admin_recipients"`
	// AdminIdentityFile is the path of that age identity file, read by the
	// admin CLI on the host to answer the challenge. It is a client-side
	// setting the daemon never opens; keeping it in the same block is what
	// lets one config file describe both ends. A leading "~" is expanded by
	// the reader.
	AdminIdentityFile string `json:"admin_identity_file" yaml:"admin_identity_file"`
}

// PartialConfig is the sparse mirror of [Config], used by the parent crabswarm
// config's merge layer: a nil field means "absent, leave the lower layer"; a
// non-nil pointer is an explicit value, including an explicit zero.
//
// The env tags hold only the bare names: the parent's Chat field carries
// envPrefix:"CHAT_" and the parent's env parse applies CRABSWARM_ globally, so
// caarlos0/env composes both onto each name (DB -> CRABSWARM_CHAT_DB,
// CMDMAN_BIN -> CRABSWARM_CHAT_CMDMAN_BIN, ...).
//
// AdminRecipients is a []string rather than a pointer: it overwrites wholesale
// on Apply (a non-nil incoming slice replaces the base, a nil one leaves it),
// and it is env-shaped, so caarlos0/env parses
// CRABSWARM_CHAT_ADMIN_RECIPIENTS as a comma-separated list.
//
// JSON tags use ",omitzero" (Go 1.24+) so a marshaled partial stays sparse;
// YAML has no omitzero, so its tags use ",omitempty".
//
//nolint:lll // triple json/yaml/env tags; one field per line, never wrap tags
type PartialConfig struct {
	Db                *string  `json:"db,omitzero" yaml:"db,omitempty" env:"DB"`
	CmdmanBin         *string  `json:"cmdman_bin,omitzero" yaml:"cmdman_bin,omitempty" env:"CMDMAN_BIN"`
	AdminRecipients   []string `json:"admin_recipients,omitzero" yaml:"admin_recipients,omitempty" env:"ADMIN_RECIPIENTS"`
	AdminIdentityFile *string  `json:"admin_identity_file,omitzero" yaml:"admin_identity_file,omitempty" env:"ADMIN_IDENTITY_FILE"`
}

// Apply overlays p's present fields onto base and returns the merged [Config].
// A scalar field is a non-nil pointer that overwrites (explicit zero included),
// a nil pointer that leaves the base untouched; AdminRecipients is a slice that
// overwrites wholesale when non-nil, an explicit empty list included.
func (p PartialConfig) Apply(base Config) Config {
	if p.Db != nil {
		base.Db = *p.Db
	}
	if p.CmdmanBin != nil {
		base.CmdmanBin = *p.CmdmanBin
	}
	if p.AdminRecipients != nil {
		base.AdminRecipients = p.AdminRecipients
	}
	if p.AdminIdentityFile != nil {
		base.AdminIdentityFile = *p.AdminIdentityFile
	}
	return base
}
