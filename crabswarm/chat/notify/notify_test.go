package notify

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ngicks/crabswarm/crabswarm/chat"
	"github.com/ngicks/crabswarm/crabswarm/chat/internal/cmdman"
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
// returns its absolute path. [NewSendKeys] takes the binary path directly, so
// nothing here touches PATH.
//
// The tests using it do not call t.Parallel: writing an executable in one test
// while another forks makes the child inherit the still-open write descriptor,
// and the exec then fails with ETXTBSY.
//
// The chat, cmdman and resolver packages keep their own copies rather than
// sharing this one: a test helper exported for another package's tests would
// have to live in non-test code, which is a worse trade than a few short
// functions.
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

// stubCmdmanScreen writes a stand-in cmdman whose `capture-screen` prints out
// and whose every other subcommand succeeds silently, recording all
// invocations. See [stubCmdman] for why these tests do not call t.Parallel.
func stubCmdmanScreen(t *testing.T, out string) string {
	t.Helper()
	return stubCmdman(t, logArgs+
		"if [ \"$1\" = capture-screen ]; then printf '%s\\n' '"+out+"'; fi\nexit 0\n")
}

// doneAgent is a member whose last report — made just now — invites a nudge.
func doneAgent() chat.Member {
	return chat.Member{
		Token:           "0123456789abcdef",
		Name:            "ana",
		Team:            "alpha",
		Room:            "/work",
		Kind:            chat.KindAgent,
		State:           chat.StateDone,
		StateReportedAt: time.Now(),
	}
}

// staleReport is a report time old enough that the state it carries is no
// longer believed, with a minute to spare so a slow test cannot land inside
// the threshold.
func staleReport() time.Time {
	return time.Now().Add(-staleStateAfter - time.Minute)
}

func bob() chat.Sender {
	return chat.Sender{Name: "bob", Team: "beta", Room: "/work"}
}

func TestSendKeys_NudgesDoneAgent(t *testing.T) {
	bin := stubCmdmanScreen(t, idlePrompt)

	err := NewSendKeys(bin, nil).Notify(t.Context(), doneAgent(), bob(), "hi")
	assert.NilError(t, err)

	// All three invocations are the contract with cmdman; pin them. The text
	// and the Enter are separate sends: cmdman hands a trailing key name in the
	// same invocation to the terminal as pasted text, and the line never
	// submits.
	args := stubArgs(t, bin)
	assert.Equal(t, len(args), 3, "invocations: %v", args)
	assert.Equal(t, args[0], "capture-screen 0123456789abcdef")
	assert.Equal(t, args[1], "send-keys 0123456789abcdef "+
		"[crabswarm chat] new message from beta/bob — run: crabswarm chat read")
	assert.Equal(t, args[2], "send-keys 0123456789abcdef Enter")
}

func TestSendKeys_SkipsBusyMember(t *testing.T) {
	// A harness mid-turn or sitting on a prompt must not be typed into. The
	// message stays in the inbox, so it is read at the end of the turn.
	for _, tc := range []struct {
		name  string
		state chat.MemberState
		// reportedAt is when the state was reported: a report this fresh is
		// still believed, so it blocks.
		reportedAt time.Time
	}{
		{"working", chat.StateWorking, time.Now()},
		{"waiting", chat.StateWaiting, time.Now()},
		// A state this notifier cannot read is declined rather than assumed
		// harmless, and age does not rescue it: the zero report time is as old
		// as they come, and it still must not be nudged.
		{"unset", "", time.Time{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := stubCmdmanScreen(t, idlePrompt)
			m := doneAgent()
			m.State = tc.state
			m.StateReportedAt = tc.reportedAt

			err := NewSendKeys(bin, nil).Notify(t.Context(), m, bob(), "hi")
			assert.NilError(t, err)
			assert.Assert(t, stubArgs(t, bin) == nil, "cmdman must not be invoked")
		})
	}
}

func TestSendKeys_NudgesMemberWedgedInABusyState(t *testing.T) {
	// A state only changes when a harness hook reports the change, so a hook
	// that never fired — the user interrupted the session, or the harness has
	// no idle notification to hook — would leave the member busy forever and
	// never nudged again. Past the threshold the report stops being believed.
	for _, state := range []chat.MemberState{chat.StateWorking, chat.StateWaiting} {
		t.Run(string(state), func(t *testing.T) {
			bin := stubCmdmanScreen(t, idlePrompt)
			m := doneAgent()
			m.State = state
			m.StateReportedAt = staleReport()

			err := NewSendKeys(bin, nil).Notify(t.Context(), m, bob(), "hi")
			assert.NilError(t, err)

			args := stubArgs(t, bin)
			assert.Equal(t, len(args), 3, "invocations: %v", args)
			assert.Equal(t, args[0], "capture-screen 0123456789abcdef")
		})
	}
}

func TestSendKeys_SkipsStaleBusyMemberShowingDialog(t *testing.T) {
	// Age lifts the state guard, not the one that looks at the terminal: a
	// stale report is the reason to ask the screen, not to skip asking it.
	bin := stubCmdmanScreen(t, "some dialog\n"+cmdman.DialogMarkers[0]+"\nmore text")
	m := doneAgent()
	m.State = chat.StateWorking
	m.StateReportedAt = staleReport()

	err := NewSendKeys(bin, nil).Notify(t.Context(), m, bob(), "hi")
	assert.NilError(t, err)

	args := stubArgs(t, bin)
	assert.Equal(t, len(args), 1, "invocations: %v", args)
	assert.Assert(t, strings.HasPrefix(args[0], "capture-screen "), "got %v", args)
}

func TestSendKeys_SkipsHuman(t *testing.T) {
	// A human's token is daemon-issued and names no cmdman command, so there is
	// no terminal to type into.
	bin := stubCmdmanScreen(t, idlePrompt)
	m := doneAgent()
	m.Kind = chat.KindHuman

	err := NewSendKeys(bin, nil).Notify(t.Context(), m, bob(), "hi")
	assert.NilError(t, err)
	assert.Assert(t, stubArgs(t, bin) == nil, "cmdman must not be invoked")
}

func TestSendKeys_SkipsWhenTerminalShowsDialog(t *testing.T) {
	// A dialog on screen: injecting here would answer it instead of typing a
	// nudge. The whole marker set is covered by the cmdman package's own
	// TestDialogMarker; this pins that a hit stops the injection, so the
	// snapshot is built from the set rather than from a literal — the set is
	// the guard's own to change.
	bin := stubCmdmanScreen(t, "some dialog\n"+cmdman.DialogMarkers[0]+"\nmore text")

	err := NewSendKeys(bin, nil).Notify(t.Context(), doneAgent(), bob(), "hi")
	assert.NilError(t, err)

	args := stubArgs(t, bin)
	assert.Equal(t, len(args), 1, "invocations: %v", args)
	assert.Assert(t, strings.HasPrefix(args[0], "capture-screen "), "got %v", args)
}

func TestSendKeys_SkipsWhenSnapshotFails(t *testing.T) {
	// Fail safe: no snapshot is no evidence the terminal is at a prompt.
	bin := stubCmdman(t, logArgs+
		"if [ \"$1\" = capture-screen ]; then "+
		"echo 'error: command has no tty' >&2; exit 1; fi\nexit 0\n")

	err := NewSendKeys(bin, nil).Notify(t.Context(), doneAgent(), bob(), "hi")
	assert.NilError(t, err, "a declined nudge is not an error")

	args := stubArgs(t, bin)
	assert.Equal(t, len(args), 1, "invocations: %v", args)
	assert.Assert(t, strings.HasPrefix(args[0], "capture-screen "), "got %v", args)
}

func TestSendKeys_InjectionFailureIsAnError(t *testing.T) {
	bin := stubCmdman(t, logArgs+
		"if [ \"$1\" = send-keys ]; then echo 'error: command is not running' >&2; exit 1; fi\n"+
		"exit 0\n")

	err := NewSendKeys(bin, nil).Notify(t.Context(), doneAgent(), bob(), "hi")
	assert.Assert(t, err != nil, "want the injection failure reported")
	assert.Assert(t, strings.Contains(err.Error(), "not running"), "got %v", err)
}

func TestSendKeys_SanitizesSenderAddress(t *testing.T) {
	for _, tc := range []struct {
		name string
		from chat.Sender
		// want is the whole address the injected line should carry.
		want string
	}{
		{
			// A raw newline would submit the line early and leave the rest of
			// it running as a command in the recipient's terminal.
			"newline in name",
			chat.Sender{Name: "bob\nrm -rf /", Team: "beta"},
			"beta/bobrm -rf /",
		},
		{
			"carriage return in team",
			chat.Sender{Name: "bob", Team: "beta\rreset"},
			"betareset/bob",
		},
		{
			"over-long name is cut",
			chat.Sender{Name: strings.Repeat("n", 200), Team: "beta"},
			"beta/" + strings.Repeat("n", maxNudgeAddrLen-len("beta/")),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := stubCmdmanScreen(t, idlePrompt)

			err := NewSendKeys(bin, nil).Notify(t.Context(), doneAgent(), tc.from, "hi")
			assert.NilError(t, err)

			// The stub logs one line per invocation, so a break inside the
			// injected text would show up as an extra line here.
			args := stubArgs(t, bin)
			assert.Equal(t, len(args), 3, "invocations: %v", args)
			assert.Equal(t, args[1], "send-keys 0123456789abcdef "+
				"[crabswarm chat] new message from "+tc.want+
				" — run: crabswarm chat read")
		})
	}
}

func TestSendKeys_RejectsMalformedTokenWithoutExec(t *testing.T) {
	for _, token := range []string{"", "--start-line", "tok id", "tok\nid"} {
		t.Run(token, func(t *testing.T) {
			bin := stubCmdmanScreen(t, idlePrompt)
			m := doneAgent()
			m.Token = token

			err := NewSendKeys(bin, nil).Notify(t.Context(), m, bob(), "hi")
			assert.NilError(t, err)
			assert.Assert(t, stubArgs(t, bin) == nil, "cmdman must not be invoked")
		})
	}
}

func TestSendKeys_NudgesAfterTheRequestIsCancelled(t *testing.T) {
	// The message is already stored by the time the notifier runs, so a sender
	// that walked away must not leave the recipient's terminal half-typed.
	bin := stubCmdmanScreen(t, idlePrompt)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := NewSendKeys(bin, nil).Notify(ctx, doneAgent(), bob(), "hi")
	assert.NilError(t, err)
	assert.Equal(t, len(stubArgs(t, bin)), 3)
}

func TestNewSendKeys_DefaultsToPathLookup(t *testing.T) {
	assert.Equal(t, NewSendKeys("", nil).terminal.Bin(), "cmdman")
	assert.Equal(t, NewSendKeys("/opt/bin/cmdman", nil).terminal.Bin(), "/opt/bin/cmdman")
}
