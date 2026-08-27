package chat

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// logArgs is a stub prologue that appends the invocation's arguments to
// "args.log" next to the stub. The stub locates the file relative to $0 rather
// than through the environment so tests need no t.Setenv and stay independent.
const logArgs = "printf '%s\\n' \"$*\" >> \"$(dirname \"$0\")/args.log\"\n"

// stubCmdman writes a stand-in cmdman whose body is the given shell script and
// returns its absolute path. [CmdmanComposeProvider] takes the binary path
// directly, so nothing here touches PATH — the same technique the e2e test
// will use against the real binary, minus the install.
//
// The tests below do not call t.Parallel: writing an executable in one test
// while another forks makes the child inherit the still-open write descriptor,
// and the exec then fails with ETXTBSY.
func stubCmdman(t *testing.T, body string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "cmdman")
	assert.NilError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"+body), 0o755))
	return bin
}

// stubArgs returns the argument lines the stub at bin recorded, one invocation
// per line. A missing log means the stub was never run.
func stubArgs(t *testing.T, bin string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(filepath.Dir(bin), "args.log"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	assert.NilError(t, err)
	var lines []string
	for l := range strings.SplitSeq(strings.TrimSpace(string(b)), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func TestCmdmanComposeProvider_Resolve_ComposeCommand(t *testing.T) {
	bin := stubCmdman(t, logArgs+`printf '%s\n' '{"argv":["claude"],"dir":"/work/repo",`+
		`"labels":{"cmdman.compose.project":"swarm","other":"x"}}'`+"\n")

	got, err := NewCmdmanComposeProvider(bin).
		Resolve(t.Context(), "0123456789abcdef0123456789abcdef")
	assert.NilError(t, err)
	assert.Equal(t, got.Room, "/work/repo")
	assert.Equal(t, got.Team, "swarm")

	// The inspect surface is the contract with cmdman; pin it.
	args := stubArgs(t, bin)
	assert.Equal(t, len(args), 1, "invocations: %v", args)
	assert.Equal(
		t,
		args[0],
		"inspect 0123456789abcdef0123456789abcdef --format {{json .Config}}",
	)
}

func TestCmdmanComposeProvider_Resolve_NoComposeLabel(t *testing.T) {
	bin := stubCmdman(t, `printf '%s\n' '{"dir":"/work/repo","labels":{"other":"x"}}'`+"\n")

	_, err := NewCmdmanComposeProvider(bin).Resolve(t.Context(), "deadbeef")
	assert.Assert(t, errors.Is(err, ErrUnknownToken), "got %v", err)
}

func TestCmdmanComposeProvider_Resolve_NoLabelsAtAll(t *testing.T) {
	bin := stubCmdman(t, `printf '%s\n' '{"dir":"/work/repo"}'`+"\n")

	_, err := NewCmdmanComposeProvider(bin).Resolve(t.Context(), "deadbeef")
	assert.Assert(t, errors.Is(err, ErrUnknownToken), "got %v", err)
}

func TestCmdmanComposeProvider_Resolve_NoWorkingDir(t *testing.T) {
	// cmdman omits "dir" when the command was created without one, so a
	// resolvable command can still have no room to place it in.
	bin := stubCmdman(t, `printf '%s\n' '{"labels":{"cmdman.compose.project":"swarm"}}'`+"\n")

	_, err := NewCmdmanComposeProvider(bin).Resolve(t.Context(), "deadbeef")
	assert.Assert(t, errors.Is(err, ErrUnknownToken), "got %v", err)
}

func TestCmdmanComposeProvider_Resolve_NoSuchCommand(t *testing.T) {
	bin := stubCmdman(t,
		"echo 'error: resolve command: no command found matching \"deadbeef\"' >&2\nexit 1\n")

	_, err := NewCmdmanComposeProvider(bin).Resolve(t.Context(), "deadbeef")
	assert.Assert(t, errors.Is(err, ErrUnknownToken), "got %v", err)
}

func TestCmdmanComposeProvider_Resolve_MalformedJSON(t *testing.T) {
	bin := stubCmdman(t, "printf 'not json at all\\n'\n")

	_, err := NewCmdmanComposeProvider(bin).Resolve(t.Context(), "deadbeef")
	assert.Assert(t, err != nil, "want a decode error")
	assert.Assert(t, !errors.Is(err, ErrUnknownToken), "got %v", err)
}

func TestCmdmanComposeProvider_Resolve_UnrelatedFailureIsNotUnknown(t *testing.T) {
	// A cmdman that fails for its own reasons must not read as "unknown
	// token": the caller reaps members on unknown, and this one is still
	// perfectly valid.
	bin := stubCmdman(t, "echo 'error: open store: database is locked' >&2\nexit 1\n")

	_, err := NewCmdmanComposeProvider(bin).Resolve(t.Context(), "deadbeef")
	assert.Assert(t, err != nil, "want a lookup error")
	assert.Assert(t, !errors.Is(err, ErrUnknownToken), "got %v", err)
	assert.Assert(t, strings.Contains(err.Error(), "database is locked"), "got %v", err)
}

func TestCmdmanComposeProvider_Resolve_MissingBinaryIsNotUnknown(t *testing.T) {
	// The reap guard: a cmdman that is not installed must never look like
	// every token being unknown.
	missing := filepath.Join(t.TempDir(), "cmdman")

	_, err := NewCmdmanComposeProvider(missing).Resolve(t.Context(), "deadbeef")
	assert.Assert(t, err != nil, "want a lookup error")
	assert.Assert(t, !errors.Is(err, ErrUnknownToken), "got %v", err)
	assert.Assert(t, errors.Is(err, fs.ErrNotExist), "got %v", err)
}

func TestCmdmanComposeProvider_Resolve_CanceledContextIsNotUnknown(t *testing.T) {
	bin := stubCmdman(t,
		`printf '%s\n' '{"dir":"/work/repo","labels":{"cmdman.compose.project":"swarm"}}'`+"\n")

	// Cancel up front: deterministic, and it exercises the same branch a
	// mid-flight cancellation takes.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := NewCmdmanComposeProvider(bin).Resolve(ctx, "deadbeef")
	assert.Assert(t, err != nil, "want a cancellation error")
	assert.Assert(t, errors.Is(err, context.Canceled), "got %v", err)
	assert.Assert(t, !errors.Is(err, ErrUnknownToken), "got %v", err)
}

func TestCmdmanComposeProvider_Resolve_RejectsMalformedTokenWithoutExec(t *testing.T) {
	for _, token := range []string{
		"",
		"   ",
		"--format",
		"-rf",
		"../../etc/passwd",
		"a b",
		"tok;rm -rf /",
		"tok\nid",
		strings.Repeat("a", maxTokenLen+1),
	} {
		t.Run(token, func(t *testing.T) {
			bin := stubCmdman(t, logArgs+"exit 0\n")

			_, err := NewCmdmanComposeProvider(bin).Resolve(t.Context(), token)
			assert.Assert(t, errors.Is(err, ErrUnknownToken), "got %v", err)
			assert.Assert(t, stubArgs(t, bin) == nil, "cmdman must not be invoked")
		})
	}
}

func TestNewCmdmanComposeProvider_DefaultsToPathLookup(t *testing.T) {
	assert.Equal(t, NewCmdmanComposeProvider("").bin, "cmdman")
	assert.Equal(t, NewCmdmanComposeProvider("/opt/bin/cmdman").bin, "/opt/bin/cmdman")
}
