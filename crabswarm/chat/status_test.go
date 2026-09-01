package chat

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// agentMember is a member cmdman can be told about: an agent whose token is a
// plausible command ID. See [stubCmdman] for why these tests do not call
// t.Parallel.
func agentMember() Member {
	return Member{
		Token: "0123456789abcdef",
		Name:  "ana",
		Team:  "alpha",
		Room:  "/work",
		Kind:  KindAgent,
		State: StateDone,
	}
}

func TestCmdmanStatusMirror_SetPublishesTheState(t *testing.T) {
	for _, state := range []MemberState{StateWorking, StateWaiting, StateDone} {
		t.Run(string(state), func(t *testing.T) {
			bin := stubCmdman(t, logArgs+"exit 0\n")

			err := NewCmdmanStatusMirror(bin, nil).Set(t.Context(), agentMember(), state)
			assert.NilError(t, err)

			// The whole invocation is the contract with cmdman; pin it. The
			// state word reaches the CLI unmapped, which is the point of
			// sharing cmdman's vocabulary.
			args := stubArgs(t, bin)
			assert.Equal(t, len(args), 1, "invocations: %v", args)
			assert.Equal(t, args[0],
				"status set "+string(state)+" 0123456789abcdef --detail crabswarm chat")
		})
	}
}

func TestCmdmanStatusMirror_ClearWithdrawsTheStatus(t *testing.T) {
	bin := stubCmdman(t, logArgs+"exit 0\n")

	assert.NilError(t, NewCmdmanStatusMirror(bin, nil).Clear(t.Context(), agentMember()))

	args := stubArgs(t, bin)
	assert.Equal(t, len(args), 1, "invocations: %v", args)
	assert.Equal(t, args[0], "status delete 0123456789abcdef")
}

// A member that declared no harness reports no state to label a command with,
// and its token need not name a command at all, so it must never reach a
// cmdman command line — not even to be rejected there.
func TestCmdmanStatusMirror_SkipsMembersWithoutACommand(t *testing.T) {
	for _, tc := range []struct {
		name   string
		member func() Member
	}{
		{"a human", func() Member {
			m := agentMember()
			m.Kind = KindHuman
			return m
		}},
		{"an unknown kind", func() Member {
			m := agentMember()
			m.Kind = ""
			return m
		}},
		{"a token cmdman could not take", func() Member {
			m := agentMember()
			m.Token = "--detail"
			return m
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := stubCmdman(t, logArgs+"exit 0\n")
			mirror := NewCmdmanStatusMirror(bin, nil)

			assert.NilError(t, mirror.Set(t.Context(), tc.member(), StateWorking))
			assert.NilError(t, mirror.Clear(t.Context(), tc.member()))
			assert.Assert(t, stubArgs(t, bin) == nil, "cmdman must not be invoked")
		})
	}
}

// cmdman refuses to label a command that is not running, which is ordinary
// rather than exceptional — but the mirror still reports it, so the service can
// decide how loudly to say so.
func TestCmdmanStatusMirror_ReportsAFailedWrite(t *testing.T) {
	bin := stubCmdman(t, "echo 'error: command is not running' >&2\nexit 1\n")
	mirror := NewCmdmanStatusMirror(bin, nil)

	err := mirror.Set(t.Context(), agentMember(), StateWorking)
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "not running"), "got %v", err)

	err = mirror.Clear(t.Context(), agentMember())
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "not running"), "got %v", err)
}

func TestNewCmdmanStatusMirror_DefaultsToPathLookup(t *testing.T) {
	assert.Equal(t, NewCmdmanStatusMirror("", nil).bin, "cmdman")
	assert.Equal(t, NewCmdmanStatusMirror("/opt/bin/cmdman", nil).bin, "/opt/bin/cmdman")
}
