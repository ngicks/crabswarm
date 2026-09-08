package resolver

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
// returns its absolute path. [CmdmanCompose] takes the binary path
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

// stubCmdmanReplying writes a stand-in cmdman that prints stdout, or fails the
// way cmdman fails when stderr is not empty. The reply is written beside the
// stub and read back by it, so a recorded line reaches the resolver byte for
// byte instead of surviving shell quoting.
func stubCmdmanReplying(t *testing.T, stdout, stderr string) string {
	t.Helper()
	bin := stubCmdman(t, `d=$(dirname "$0")
if [ -s "$d/stderr" ]; then
	cat "$d/stderr" >&2
	exit 1
fi
cat "$d/stdout"
`)
	dir := filepath.Dir(bin)
	assert.NilError(t, os.WriteFile(filepath.Join(dir, "stdout"), []byte(stdout+"\n"), 0o644))
	assert.NilError(t, os.WriteFile(filepath.Join(dir, "stderr"), []byte(stderr), 0o644))
	return bin
}

func TestCmdmanCompose_Resolve_ComposeCommand(t *testing.T) {
	bin := stubCmdman(t, logArgs+`printf '%s\n' 'running {"argv":["claude"],"dir":"/work/repo",`+
		`"labels":{"cmdman.compose.project":"swarm","other":"x"}}'`+"\n")

	got, err := NewCmdmanCompose(bin).
		Resolve(t.Context(), "0123456789abcdef0123456789abcdef")
	assert.NilError(t, err)
	assert.Equal(t, got.Room, "/work/repo")
	assert.Equal(t, got.Team, "swarm")
	// No naming labels at all, so the resolver derives no name.
	assert.Equal(t, got.Name, "")

	// The inspect surface is the contract with cmdman; pin it.
	args := stubArgs(t, bin)
	assert.Equal(t, len(args), 1, "invocations: %v", args)
	assert.Equal(
		t,
		args[0],
		"inspect 0123456789abcdef0123456789abcdef --format {{.State}} {{json .Config}}",
	)
}

func TestCmdmanCompose_Resolve_NameFromCommandAndScaleIndex(t *testing.T) {
	bin := stubCmdman(t, `printf '%s\n' 'running {"dir":"/work/repo","labels":{`+
		`"cmdman.compose.project":"swarm","cmdman.compose.command":"worker",`+
		`"cmdman.compose.scale-index":"2"}}'`+"\n")

	got, err := NewCmdmanCompose(bin).Resolve(t.Context(), "deadbeef")
	assert.NilError(t, err)
	assert.Equal(t, got.Name, "worker-2")
}

func TestCmdmanCompose_Resolve_NameFromCommandOnly(t *testing.T) {
	// cmdman-compose always stamps a scale index, so this is the defensive
	// shape: without one the command name has to stand on its own rather than
	// leave the member unnamed.
	bin := stubCmdman(t, `printf '%s\n' 'running {"dir":"/work/repo","labels":{`+
		`"cmdman.compose.project":"swarm","cmdman.compose.command":"worker"}}'`+"\n")

	got, err := NewCmdmanCompose(bin).Resolve(t.Context(), "deadbeef")
	assert.NilError(t, err)
	assert.Equal(t, got.Name, "worker")
}

func TestCmdmanCompose_Resolve_NameNeedsCommandLabel(t *testing.T) {
	// A scale index without a command names nothing on its own; the caller
	// must fall back to its own default rather than be handed "-2".
	bin := stubCmdman(t, `printf '%s\n' 'running {"dir":"/work/repo","labels":{`+
		`"cmdman.compose.project":"swarm","cmdman.compose.scale-index":"2"}}'`+"\n")

	got, err := NewCmdmanCompose(bin).Resolve(t.Context(), "deadbeef")
	assert.NilError(t, err)
	assert.Equal(t, got.Name, "")
}

// Only the first run of whitespace separates the state from the config, so a
// space inside the config survives: a command whose working directory has one
// resolves to that whole directory rather than to the half before the space.
func TestCmdmanCompose_Resolve_ConfigCarryingASpace(t *testing.T) {
	bin := stubCmdmanReplying(t, `running {"argv":["claude","--dir","/work/my repo"],`+
		`"dir":"/work/my repo","labels":{"cmdman.compose.project":"swarm"}}`, "")

	got, err := NewCmdmanCompose(bin).Resolve(t.Context(), "deadbeef")
	assert.NilError(t, err)
	assert.Equal(t, got.Room, "/work/my repo")
	assert.Equal(t, got.Team, "swarm")
}

func TestCmdmanCompose_Resolve_NoComposeLabel(t *testing.T) {
	bin := stubCmdman(t, `printf '%s\n' 'running {"dir":"/work/repo","labels":{"other":"x"}}'`+"\n")

	_, err := NewCmdmanCompose(bin).Resolve(t.Context(), "deadbeef")
	assert.Assert(t, errors.Is(err, ErrUnknownToken), "got %v", err)
}

func TestCmdmanCompose_Resolve_NoLabelsAtAll(t *testing.T) {
	bin := stubCmdman(t, `printf '%s\n' 'running {"dir":"/work/repo"}'`+"\n")

	_, err := NewCmdmanCompose(bin).Resolve(t.Context(), "deadbeef")
	assert.Assert(t, errors.Is(err, ErrUnknownToken), "got %v", err)
}

func TestCmdmanCompose_Resolve_NoWorkingDir(t *testing.T) {
	// cmdman omits "dir" when the command was created without one, so a
	// resolvable command can still have no room to place it in.
	bin := stubCmdman(
		t,
		`printf '%s\n' 'running {"labels":{"cmdman.compose.project":"swarm"}}'`+"\n",
	)

	_, err := NewCmdmanCompose(bin).Resolve(t.Context(), "deadbeef")
	assert.Assert(t, errors.Is(err, ErrUnknownToken), "got %v", err)
}

func TestCmdmanCompose_Resolve_NoSuchCommand(t *testing.T) {
	bin := stubCmdman(t,
		"echo 'error: resolve command: no command found matching \"deadbeef\"' >&2\nexit 1\n")

	_, err := NewCmdmanCompose(bin).Resolve(t.Context(), "deadbeef")
	assert.Assert(t, errors.Is(err, ErrUnknownToken), "got %v", err)
}

func TestCmdmanCompose_Resolve_MalformedJSON(t *testing.T) {
	bin := stubCmdman(t, "printf 'not json at all\\n'\n")

	_, err := NewCmdmanCompose(bin).Resolve(t.Context(), "deadbeef")
	assert.Assert(t, err != nil, "want a decode error")
	assert.Assert(t, !errors.Is(err, ErrUnknownToken), "got %v", err)
}

func TestCmdmanCompose_Resolve_UnrelatedFailureIsNotUnknown(t *testing.T) {
	// A cmdman that fails for its own reasons must not read as "unknown
	// token": the caller reaps members on unknown, and this one is still
	// perfectly valid.
	bin := stubCmdman(t, "echo 'error: open store: database is locked' >&2\nexit 1\n")

	_, err := NewCmdmanCompose(bin).Resolve(t.Context(), "deadbeef")
	assert.Assert(t, err != nil, "want a lookup error")
	assert.Assert(t, !errors.Is(err, ErrUnknownToken), "got %v", err)
	assert.Assert(t, strings.Contains(err.Error(), "database is locked"), "got %v", err)
}

func TestCmdmanCompose_Resolve_MissingBinaryIsNotUnknown(t *testing.T) {
	// The reap guard: a cmdman that is not installed must never look like
	// every token being unknown.
	missing := filepath.Join(t.TempDir(), "cmdman")

	_, err := NewCmdmanCompose(missing).Resolve(t.Context(), "deadbeef")
	assert.Assert(t, err != nil, "want a lookup error")
	assert.Assert(t, !errors.Is(err, ErrUnknownToken), "got %v", err)
	assert.Assert(t, errors.Is(err, fs.ErrNotExist), "got %v", err)
}

func TestCmdmanCompose_Resolve_CanceledContextIsNotUnknown(t *testing.T) {
	bin := stubCmdman(t,
		`printf '%s\n' 'running {"dir":"/work/repo","labels":{"cmdman.compose.project":"swarm"}}'`+
			"\n")

	// Cancel up front: deterministic, and it exercises the same branch a
	// mid-flight cancellation takes.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := NewCmdmanCompose(bin).Resolve(ctx, "deadbeef")
	assert.Assert(t, err != nil, "want a cancellation error")
	assert.Assert(t, errors.Is(err, context.Canceled), "got %v", err)
	assert.Assert(t, !errors.Is(err, ErrUnknownToken), "got %v", err)
}

func TestCmdmanCompose_Resolve_RejectsMalformedTokenWithoutExec(t *testing.T) {
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

			_, err := NewCmdmanCompose(bin).Resolve(t.Context(), token)
			assert.Assert(t, errors.Is(err, ErrUnknownToken), "got %v", err)
			assert.Assert(t, stubArgs(t, bin) == nil, "cmdman must not be invoked")
		})
	}
}

// inspectFixture is what a recorded `cmdman inspect` printed at one phase of a
// command's life: the stdout line of a lookup that succeeded, or the stderr
// line of one that failed.
type inspectFixture struct {
	stdout string
	stderr string
}

// loadInspectFixtures reads the recorded cmdman probe under testdata and
// returns what each phase printed, keyed by phase name.
//
// The probe asked cmdman for the exit code as well, which the resolver never
// does, so the recorded state and config are reassembled into the line the
// resolver's own format produces: the state word, one space, and the config
// from its first "{". Everything the probe put between the two is dropped, and
// the two halves the resolver reads stay the bytes cmdman actually printed.
func loadInspectFixtures(t *testing.T) map[string]inspectFixture {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "cmdman-inspect-states.txt"))
	assert.NilError(t, err)

	fixtures := map[string]inspectFixture{}
	for line := range strings.SplitSeq(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "---") {
			// What follows is the probe's closing `cmdman ls` table, not an
			// inspect of any phase.
			break
		}
		phase, rest, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}
		switch {
		case strings.HasPrefix(rest, "error:"):
			fixtures[phase] = inspectFixture{stderr: rest}
		case strings.Contains(rest, "{"):
			state, _, _ := strings.Cut(rest, " ")
			fixtures[phase] = inspectFixture{
				stdout: state + " " + rest[strings.Index(rest, "{"):],
			}
		}
	}
	return fixtures
}

func TestCmdmanCompose_Resolve_RecordedCmdmanPhases(t *testing.T) {
	fixtures := loadInspectFixtures(t)

	// The recorded command carries the compose labels project "p", command "c"
	// and the scale index of the phase's own replica.
	for _, tc := range []struct {
		phase   string
		want    TeamInfo
		unknown string // the reap the phase must produce, by a fragment of its text
	}{
		{phase: "running", want: TeamInfo{Room: "/tmp/tmp.JT9T17P8Xt", Team: "p", Name: "c-2"}},
		{phase: "created", unknown: `is created, not running`},
		{phase: "stopped", unknown: `is exited, not running`},
		{phase: "exited", unknown: `is exited, not running`},
		{phase: "removed", unknown: `knows no command`},
	} {
		t.Run(tc.phase, func(t *testing.T) {
			f, ok := fixtures[tc.phase]
			assert.Assert(t, ok, "the recorded probe holds no %q phase", tc.phase)
			bin := stubCmdmanReplying(t, f.stdout, f.stderr)

			got, err := NewCmdmanCompose(bin).Resolve(t.Context(), "probe")
			if tc.unknown != "" {
				assert.Assert(t, errors.Is(err, ErrUnknownToken), "got %v", err)
				assert.Assert(t, strings.Contains(err.Error(), tc.unknown), "got %v", err)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, got.Room, tc.want.Room)
			assert.Equal(t, got.Team, tc.want.Team)
			assert.Equal(t, got.Name, tc.want.Name)
		})
	}
}

func TestCmdmanCompose_Resolve_UnreadableOutputIsNotUnknown(t *testing.T) {
	// The reap guard again, this time against a cmdman that stopped printing
	// what this package reads: output no state can be taken from must fail the
	// lookup rather than report every token gone.
	for _, tc := range []struct {
		name   string
		stdout string
	}{
		{name: "config without a state", stdout: `{"dir":"/work/repo",` +
			`"labels":{"cmdman.compose.project":"swarm"}}`},
		{name: "state without a config", stdout: "running"},
		{name: "no output at all", stdout: ""},
		{name: "garbage", stdout: "cmdman: something else entirely"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := stubCmdmanReplying(t, tc.stdout, "")

			_, err := NewCmdmanCompose(bin).Resolve(t.Context(), "deadbeef")
			assert.Assert(t, err != nil, "want a lookup error")
			assert.Assert(t, !errors.Is(err, ErrUnknownToken), "got %v", err)
		})
	}
}

func TestCmdmanCompose_Resolve_UnrecognisedStateIsNotUnknown(t *testing.T) {
	// The same guard where it is easiest to trip: a state word this package has
	// never seen decides nothing at all. A cmdman that renames or re-cases its
	// states must fail the lookup and keep the member, not read as every agent
	// having ended at once.
	config := `{"dir":"/work/repo","labels":{"cmdman.compose.project":"swarm"}}`
	for _, state := range []string{"Running", "up", "starting"} {
		t.Run(state, func(t *testing.T) {
			bin := stubCmdmanReplying(t, state+" "+config, "")

			_, err := NewCmdmanCompose(bin).Resolve(t.Context(), "deadbeef")
			assert.Assert(t, err != nil, "want a lookup error")
			assert.Assert(t, !errors.Is(err, ErrUnknownToken), "got %v", err)
			assert.Assert(t, strings.Contains(err.Error(), state), "got %v", err)
		})
	}
}

func TestNewCmdmanCompose_DefaultsToPathLookup(t *testing.T) {
	assert.Equal(t, NewCmdmanCompose("").bin, "cmdman")
	assert.Equal(t, NewCmdmanCompose("/opt/bin/cmdman").bin, "/opt/bin/cmdman")
}
