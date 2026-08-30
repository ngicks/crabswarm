package notify

import (
	"errors"
	"strings"
	"testing"

	"github.com/ngicks/crabswarm/crabswarm/chat"
	"gotest.tools/v3/assert"
)

// The stub helpers ([stubCmdman], [stubCmdmanLogs], [stubArgs], [doneAgent])
// live in notify_test.go; same package, so these tests share them. See
// [stubCmdman] for why none of these call t.Parallel.

func TestCmdman_SendCommandTypesThenSubmits(t *testing.T) {
	bin := stubCmdmanLogs(t, idlePrompt)

	err := NewCmdman(bin, nil).SendCommand(t.Context(), doneAgent(), "/new")
	assert.NilError(t, err)

	// The text and the Enter are separate invocations on purpose: cmdman hands
	// a trailing key name in the same invocation to the terminal as pasted
	// text, and the line never submits.
	args := stubArgs(t, bin)
	assert.Equal(t, len(args), 3, "invocations: %v", args)
	assert.Equal(t, args[0], "logs --tail 40 0123456789abcdef")
	assert.Equal(t, args[1], "send-keys 0123456789abcdef /new")
	assert.Equal(t, args[2], "send-keys 0123456789abcdef Enter")
}

func TestCmdman_SendCommandDeclines(t *testing.T) {
	// Every guard reports the same way, so a caller can tell "there was nothing
	// to type into" apart from "the typing failed" with one errors.Is.
	idle := func(t *testing.T) string { return stubCmdmanLogs(t, idlePrompt) }
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
			func(m chat.Member) chat.Member { m.Token = "--tail"; return m },
		},
		{
			// Fail safe: no snapshot is no evidence the terminal is at a prompt.
			"snapshot unavailable",
			func(t *testing.T) string {
				return stubCmdman(t, logArgs+
					"if [ \"$1\" = logs ]; then exit 1; fi\nexit 0\n")
			},
			func(m chat.Member) chat.Member { return m },
		},
		{
			// Injecting into a dialog would answer it rather than type a line.
			// Which strings mean "dialog" is the guard's own business, so the
			// snapshot is built from the set rather than from a literal.
			"terminal shows a dialog",
			func(t *testing.T) string { return stubCmdmanLogs(t, dialogMarkers[0]) },
			func(m chat.Member) chat.Member { return m },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := tc.stub(t)

			err := NewCmdman(bin, nil).SendCommand(
				t.Context(), tc.member(doneAgent()), "/new")
			assert.Assert(t, errors.Is(err, ErrDeclined), "got %v", err)

			for _, a := range stubArgs(t, bin) {
				assert.Assert(t, !strings.HasPrefix(a, "send-keys "),
					"a declined send must type nothing, got %q", a)
			}
		})
	}
}

func TestCmdman_SendCommandReportsExecFailure(t *testing.T) {
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
				"if [ \"$1\" = logs ]; then printf '%s\\n' '"+idlePrompt+"'; exit 0; fi\n"+
				"if [ \"$3\" = '"+tc.failOn+"' ]; then "+
				"echo 'error: command is not running' >&2; exit 1; fi\n"+
				"exit 0\n")

			err := NewCmdman(bin, nil).SendCommand(t.Context(), doneAgent(), "/new")
			assert.Assert(t, err != nil, "want the failure reported")
			assert.Assert(t, !errors.Is(err, ErrDeclined), "got %v", err)
			assert.Assert(t, strings.Contains(err.Error(), "not running"), "got %v", err)
		})
	}
}

func TestNewCmdman_DefaultsToPathLookup(t *testing.T) {
	assert.Equal(t, NewCmdman("", nil).bin, "cmdman")
	assert.Equal(t, NewCmdman("/opt/bin/cmdman", nil).bin, "/opt/bin/cmdman")
}
