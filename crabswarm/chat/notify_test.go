package chat

import (
	"context"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// idlePrompt is what a harness waiting for a command leaves in its log: no
// dialog marker anywhere in it.
const idlePrompt = "> ready"

// stubCmdmanLogs writes a stand-in cmdman whose `logs` prints out and whose
// every other subcommand succeeds silently, recording all invocations. See
// [stubCmdman] for why these tests do not call t.Parallel.
func stubCmdmanLogs(t *testing.T, out string) string {
	t.Helper()
	return stubCmdman(t, logArgs+
		"if [ \"$1\" = logs ]; then printf '%s\\n' '"+out+"'; fi\nexit 0\n")
}

// idleAgent is a member in the one state that invites a nudge.
func idleAgent() Member {
	return Member{
		Token: "0123456789abcdef",
		Name:  "ana",
		Team:  "alpha",
		Room:  "/work",
		Kind:  KindAgent,
		State: StateDone,
	}
}

func bob() Sender {
	return Sender{Name: "bob", Team: "beta", Room: "/work"}
}

func TestSendKeysNotifier_NudgesIdleAgent(t *testing.T) {
	bin := stubCmdmanLogs(t, idlePrompt)

	err := NewSendKeysNotifier(bin, nil).Notify(t.Context(), idleAgent(), bob(), "hi")
	assert.NilError(t, err)

	// Both invocations are the contract with cmdman; pin them.
	args := stubArgs(t, bin)
	assert.Equal(t, len(args), 2, "invocations: %v", args)
	assert.Equal(t, args[0], "logs --tail 40 0123456789abcdef")
	assert.Equal(t, args[1], "send-keys 0123456789abcdef "+
		"[crabswarm chat] new message from beta/bob — run: crabswarm chat read Enter")
}

func TestSendKeysNotifier_SkipsBusyMember(t *testing.T) {
	// A harness mid-turn or sitting on a prompt must not be typed into. The
	// message stays in the inbox, so it is read at the end of the turn.
	for _, tc := range []struct {
		name  string
		state MemberState
	}{
		{"running", StateWorking},
		{"waiting for input", StateWaiting},
		// Only idle invites a nudge, so a state this notifier cannot read is
		// declined rather than assumed harmless.
		{"unset", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := stubCmdmanLogs(t, idlePrompt)
			m := idleAgent()
			m.State = tc.state

			err := NewSendKeysNotifier(bin, nil).Notify(t.Context(), m, bob(), "hi")
			assert.NilError(t, err)
			assert.Assert(t, stubArgs(t, bin) == nil, "cmdman must not be invoked")
		})
	}
}

func TestSendKeysNotifier_SkipsHuman(t *testing.T) {
	// A human's token is daemon-issued and names no cmdman command, so there is
	// no terminal to type into.
	bin := stubCmdmanLogs(t, idlePrompt)
	m := idleAgent()
	m.Kind = KindHuman

	err := NewSendKeysNotifier(bin, nil).Notify(t.Context(), m, bob(), "hi")
	assert.NilError(t, err)
	assert.Assert(t, stubArgs(t, bin) == nil, "cmdman must not be invoked")
}

func TestSendKeysNotifier_SkipsWhenTerminalShowsDialog(t *testing.T) {
	// A permission dialog on screen: injecting here would answer it instead of
	// typing a nudge. The whole marker set is covered by [TestDialogMarker];
	// this pins that a hit stops the injection.
	bin := stubCmdmanLogs(t, "Do you want to make this edit?\n❯ 1. Yes\n  2. No")

	err := NewSendKeysNotifier(bin, nil).Notify(t.Context(), idleAgent(), bob(), "hi")
	assert.NilError(t, err)

	args := stubArgs(t, bin)
	assert.Equal(t, len(args), 1, "invocations: %v", args)
	assert.Assert(t, strings.HasPrefix(args[0], "logs "), "got %v", args)
}

func TestDialogMarker(t *testing.T) {
	// Every marker must match whatever casing the harness prints it in, and an
	// idle prompt must match none of them — the guard declining forever would
	// be as silent a failure as it never declining.
	for _, marker := range dialogMarkers {
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

func TestSendKeysNotifier_SkipsWhenSnapshotFails(t *testing.T) {
	// Fail safe: no snapshot is no evidence the terminal is at a prompt.
	bin := stubCmdman(t, logArgs+
		"if [ \"$1\" = logs ]; then echo 'error: no log for command' >&2; exit 1; fi\nexit 0\n")

	err := NewSendKeysNotifier(bin, nil).Notify(t.Context(), idleAgent(), bob(), "hi")
	assert.NilError(t, err, "a declined nudge is not an error")

	args := stubArgs(t, bin)
	assert.Equal(t, len(args), 1, "invocations: %v", args)
	assert.Assert(t, strings.HasPrefix(args[0], "logs "), "got %v", args)
}

func TestSendKeysNotifier_InjectionFailureIsAnError(t *testing.T) {
	bin := stubCmdman(t, logArgs+
		"if [ \"$1\" = send-keys ]; then echo 'error: command is not running' >&2; exit 1; fi\n"+
		"exit 0\n")

	err := NewSendKeysNotifier(bin, nil).Notify(t.Context(), idleAgent(), bob(), "hi")
	assert.Assert(t, err != nil, "want the injection failure reported")
	assert.Assert(t, strings.Contains(err.Error(), "not running"), "got %v", err)
}

func TestSendKeysNotifier_SanitizesSenderAddress(t *testing.T) {
	for _, tc := range []struct {
		name string
		from Sender
		// want is the whole address the injected line should carry.
		want string
	}{
		{
			// A raw newline would submit the line early and leave the rest of
			// it running as a command in the recipient's terminal.
			"newline in name",
			Sender{Name: "bob\nrm -rf /", Team: "beta"},
			"beta/bobrm -rf /",
		},
		{
			"carriage return in team",
			Sender{Name: "bob", Team: "beta\rreset"},
			"betareset/bob",
		},
		{
			"over-long name is cut",
			Sender{Name: strings.Repeat("n", 200), Team: "beta"},
			"beta/" + strings.Repeat("n", maxNudgeAddrLen-len("beta/")),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := stubCmdmanLogs(t, idlePrompt)

			err := NewSendKeysNotifier(bin, nil).Notify(t.Context(), idleAgent(), tc.from, "hi")
			assert.NilError(t, err)

			// The stub logs one line per invocation, so a break inside the
			// injected text would show up as an extra line here.
			args := stubArgs(t, bin)
			assert.Equal(t, len(args), 2, "invocations: %v", args)
			assert.Equal(t, args[1], "send-keys 0123456789abcdef "+
				"[crabswarm chat] new message from "+tc.want+
				" — run: crabswarm chat read Enter")
		})
	}
}

func TestSendKeysNotifier_RejectsMalformedTokenWithoutExec(t *testing.T) {
	for _, token := range []string{"", "--tail", "tok id", "tok\nid"} {
		t.Run(token, func(t *testing.T) {
			bin := stubCmdmanLogs(t, idlePrompt)
			m := idleAgent()
			m.Token = token

			err := NewSendKeysNotifier(bin, nil).Notify(t.Context(), m, bob(), "hi")
			assert.NilError(t, err)
			assert.Assert(t, stubArgs(t, bin) == nil, "cmdman must not be invoked")
		})
	}
}

func TestSendKeysNotifier_NudgesAfterTheRequestIsCancelled(t *testing.T) {
	// The message is already stored by the time the notifier runs, so a sender
	// that walked away must not leave the recipient's terminal half-typed.
	bin := stubCmdmanLogs(t, idlePrompt)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := NewSendKeysNotifier(bin, nil).Notify(ctx, idleAgent(), bob(), "hi")
	assert.NilError(t, err)
	assert.Equal(t, len(stubArgs(t, bin)), 2)
}

func TestNewSendKeysNotifier_DefaultsToPathLookup(t *testing.T) {
	assert.Equal(t, NewSendKeysNotifier("", nil).bin, "cmdman")
	assert.Equal(t, NewSendKeysNotifier("/opt/bin/cmdman", nil).bin, "/opt/bin/cmdman")
}
