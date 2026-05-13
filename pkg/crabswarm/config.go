// Package crabswarm exposes shared configuration for the crabswarm CLI and
// its supporting services. Env-var reads for the project live here so that
// code under ./cmd stays free of os.Getenv calls.
package crabswarm

import (
	"os"
	"path/filepath"
)

// Env-var names recognized by this package.
const (
	EnvSock       = "CRABSWARM_SOCK"
	EnvProjectDir = "CLAUDE_PROJECT_DIR"
	envXDGRuntime = "XDG_RUNTIME_DIR"
)

// SocketPath resolves the Unix-domain socket path used by crabswarm.
//
// Precedence:
//  1. override (e.g. the --sock flag); returned as-is when non-empty.
//  2. $CRABSWARM_SOCK.
//  3. ${XDG_RUNTIME_DIR:-/tmp}/crabswarm/default.sock.
func SocketPath(override string) string {
	if override != "" {
		return override
	}
	if env := os.Getenv(EnvSock); env != "" {
		return env
	}
	runtimeDir := os.Getenv(envXDGRuntime)
	if runtimeDir == "" {
		runtimeDir = "/tmp"
	}
	return filepath.Join(runtimeDir, "crabswarm", "default.sock")
}

// ProjectDir returns the Claude Code project directory from $CLAUDE_PROJECT_DIR.
// Returns an empty string when the variable is unset.
func ProjectDir() string {
	return os.Getenv(EnvProjectDir)
}
