package cmdman

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ngicks/crabswarm/crabswarm/chat"
	"gotest.tools/v3/assert"
)

// idlePrompt is what a harness waiting for a command leaves on its screen: no
// dialog marker anywhere in it.
const idlePrompt = "> ready"

// logArgs is a stub prologue that appends the invocation's arguments to
// "args.log" next to the stub. The stub locates the file relative to $0 rather
// than through the environment so tests need no t.Setenv and stay independent.
const logArgs = "printf '%s\\n' \"$*\" >> \"$(dirname \"$0\")/args.log\"\n"

// stubCmdman writes a stand-in cmdman whose body is the given shell script and
// returns its absolute path. [NewTerminal] takes the binary path directly, so
// nothing here touches PATH.
//
// The tests using it do not call t.Parallel: writing an executable in one test
// while another forks makes the child inherit the still-open write descriptor,
// and the exec then fails with ETXTBSY.
//
// The chat, notify and resolver packages keep their own copies rather than
// sharing this one: a test helper exported for another package's tests would
// have to live in non-test code, which is a worse trade than a few short
// functions.
func stubCmdman(t *testing.T, body string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "cmdman")
	assert.NilError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"+body), 0o755))
	return bin
}

// stubCmdmanScreen writes a stand-in cmdman whose `capture-screen` prints out
// and whose every other subcommand succeeds silently, recording all
// invocations. See [stubCmdman] for why these tests do not call t.Parallel.
func stubCmdmanScreen(t *testing.T, out string) string {
	t.Helper()
	return stubCmdman(t, logArgs+
		"if [ \"$1\" = capture-screen ]; then printf '%s\\n' '"+out+"'; fi\nexit 0\n")
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

// doneAgent is a member every guard here lets through. Its state is set for
// realism only: the state guard is nudge policy and lives in the notifier, so
// nothing in this package reads it.
func doneAgent() chat.Member {
	return chat.Member{
		Token: "0123456789abcdef",
		Name:  "ana",
		Team:  "alpha",
		Room:  "/work",
		Kind:  chat.KindAgent,
		State: chat.StateDone,
	}
}

func TestTerminal_SendCommandTypesThenSubmits(t *testing.T) {
	bin := stubCmdmanScreen(t, idlePrompt)

	err := NewTerminal(bin, nil).SendCommand(t.Context(), doneAgent(), "/new")
	assert.NilError(t, err)

	// The text and the Enter are separate invocations on purpose: cmdman hands
	// a trailing key name in the same invocation to the terminal as pasted
	// text, and the line never submits.
	args := stubArgs(t, bin)
	assert.Equal(t, len(args), 3, "invocations: %v", args)
	assert.Equal(t, args[0], "capture-screen 0123456789abcdef")
	assert.Equal(t, args[1], "send-keys 0123456789abcdef /new")
	assert.Equal(t, args[2], "send-keys 0123456789abcdef Enter")
}

func TestTerminal_SendCommandDeclines(t *testing.T) {
	// Every guard reports the same way, so a caller can tell "there was nothing
	// to type into" apart from "the typing failed" with one errors.Is.
	idle := func(t *testing.T) string { return stubCmdmanScreen(t, idlePrompt) }
	for _, tc := range []struct {
		name   string
		stub   func(t *testing.T) string
		member func(chat.Member) chat.Member
	}{
		{
			// A human's token is daemon-issued and names no cmdman command, so
			// there is no terminal to type into.
			"member runs no harness",
			idle,
			func(m chat.Member) chat.Member { m.Kind = chat.KindHuman; return m },
		},
		{
			"token cmdman cannot take",
			idle,
			func(m chat.Member) chat.Member { m.Token = "--start-line"; return m },
		},
		{
			// Fail safe: no snapshot is no evidence the terminal is at a prompt.
			"snapshot unavailable",
			func(t *testing.T) string {
				return stubCmdman(t, logArgs+
					"if [ \"$1\" = capture-screen ]; then exit 1; fi\nexit 0\n")
			},
			func(m chat.Member) chat.Member { return m },
		},
		{
			// Injecting into a dialog would answer it rather than type a line.
			// Which strings mean "dialog" is the guard's own business, so the
			// snapshot is built from the set rather than from a literal.
			"terminal shows a dialog",
			func(t *testing.T) string { return stubCmdmanScreen(t, DialogMarkers[0]) },
			func(m chat.Member) chat.Member { return m },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := tc.stub(t)

			err := NewTerminal(bin, nil).SendCommand(
				t.Context(), tc.member(doneAgent()), "/new")
			assert.Assert(t, errors.Is(err, ErrDeclined), "got %v", err)

			for _, a := range stubArgs(t, bin) {
				assert.Assert(t, !strings.HasPrefix(a, "send-keys "),
					"a declined send must type nothing, got %q", a)
			}
		})
	}
}

func TestTerminal_SendCommandReportsExecFailure(t *testing.T) {
	// A cmdman that ran and failed is not a decline: nothing about the member
	// said "do not type here", so the caller has a real failure to handle.
	for _, tc := range []struct {
		name string
		// failOn is the send-keys argument the stub refuses.
		failOn string
	}{
		{"text is refused", "/new"},
		// The text landed and only the submit failed, leaving the line sitting
		// unsubmitted in the recipient's prompt — the case most worth reporting.
		{"submit is refused", "Enter"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := stubCmdman(t, logArgs+
				"if [ \"$1\" = capture-screen ]; then printf '%s\\n' '"+idlePrompt+"'; exit 0; fi\n"+
				"if [ \"$3\" = '"+tc.failOn+"' ]; then "+
				"echo 'error: command is not running' >&2; exit 1; fi\n"+
				"exit 0\n")

			err := NewTerminal(bin, nil).SendCommand(t.Context(), doneAgent(), "/new")
			assert.Assert(t, err != nil, "want the failure reported")
			assert.Assert(t, !errors.Is(err, ErrDeclined), "got %v", err)
			assert.Assert(t, strings.Contains(err.Error(), "not running"), "got %v", err)
		})
	}
}

func TestDialogMarker(t *testing.T) {
	// Every marker must match whatever casing the harness prints it in, and an
	// idle prompt must match none of them — the guard declining forever would
	// be as silent a failure as it never declining.
	for _, marker := range DialogMarkers {
		t.Run(marker, func(t *testing.T) {
			for _, snapshot := range []string{
				"noise\n" + marker + "\nnoise",
				"noise\n" + strings.ToUpper(marker) + "\nnoise",
				"noise\n" + strings.ToLower(marker) + "\nnoise",
			} {
				got, found := dialogMarker(snapshot)
				assert.Assert(t, found, "no marker found in %q", snapshot)
				assert.Equal(t, got, marker)
			}
		})
	}

	_, found := dialogMarker("$ echo hello\nhello\n" + idlePrompt)
	assert.Assert(t, !found, "an idle prompt must not read as a dialog")
}

func TestNewTerminal_DefaultsToPathLookup(t *testing.T) {
	assert.Equal(t, NewTerminal("", nil).bin, "cmdman")
	assert.Equal(t, NewTerminal("/opt/bin/cmdman", nil).bin, "/opt/bin/cmdman")
}
