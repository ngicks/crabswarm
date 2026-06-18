// Package crabswarm implements the crabswarm service backing the binary of the
// same name: the server, the Claude Code hook handlers, and the git helpers.
// All env-var reads and configuration-file unmarshaling live in config.go so
// that code under ./cmd stays free of os.Getenv calls and ad-hoc file I/O.
package crabswarm

// Version is the human-readable version string. The release helper at
// internal/cmd/release rewrites this declaration when cutting a release,
// then bumps it to the next "-devel" version after tagging.
//
// Edit by hand only when the release helper is unavailable (e.g. cherry-pick
// of a release commit).
const Version = "v0.0.9-devel"
