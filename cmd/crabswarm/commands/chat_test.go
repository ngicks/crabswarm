package commands

import (
	"bytes"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	chatcli "github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// chatHermeticEnv isolates a chat command from the host environment. It extends
// hermeticEnv with the two identity variables, which matter here and nowhere
// else: this suite may itself run under cmdman, whose CMDMAN_CMD_ID would
// otherwise be picked up as a perfectly good token.
func chatHermeticEnv(t *testing.T) {
	t.Helper()
	hermeticEnv(t)
	t.Setenv(chatcli.TokenEnvVar, "")
	t.Setenv(chatcli.CmdIDEnvVar, "")
}

// runChatCmd executes "chat <args...>" through the real root command, capturing
// stdout and stderr separately (see runConfigCmd).
func runChatCmd(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := rootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs(append([]string{"chat"}, args...))
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

// The chat tree is wired: every member verb, and the admin group with its own
// four children.
func TestChatCmd_Subcommands(t *testing.T) {
	root := rootCmd()
	chat, _, err := root.Find([]string{"chat"})
	assert.NilError(t, err)
	assert.Equal(t, chat.Name(), "chat")

	names := map[string]bool{}
	for _, c := range chat.Commands() {
		names[c.Name()] = true
	}
	for _, want := range []string{
		"join", "send", "broadcast", "read", "history", "members", "leave",
		"report-state", "admin",
	} {
		assert.Assert(t, names[want], "chat has no %q subcommand", want)
	}

	admin, _, err := root.Find([]string{"chat", "admin"})
	assert.NilError(t, err)
	assert.Equal(t, admin.Name(), "admin")
	adminNames := map[string]bool{}
	for _, c := range admin.Commands() {
		adminNames[c.Name()] = true
	}
	for _, want := range []string{"list", "register", "move", "send"} {
		assert.Assert(t, adminNames[want], "chat admin has no %q subcommand", want)
	}
}

// Without a token a member verb fails before it touches the network, and the
// message names every way to supply one.
func TestChatMemberVerbs_RequireAToken(t *testing.T) {
	for _, args := range [][]string{
		{"join"},
		{"read"},
		{"history"},
		{"members"},
		{"leave"},
		{"send", "alice", "hi"},
		{"broadcast", "hi"},
		{"report-state", "done"},
	} {
		t.Run(args[0], func(t *testing.T) {
			chatHermeticEnv(t)

			stdout, _, err := runChatCmd(t, args...)
			assert.Assert(t, err != nil)
			assert.Assert(t, strings.Contains(err.Error(), "--token"))
			assert.Equal(t, stdout, "")
		})
	}
}

// The --token flag outranks the environment, so a human can act as a registered
// member from inside a cmdman-managed shell. It is checked here through the
// wiring rather than only in the resolver: the flag is declared on the chat
// parent and has to reach every child.
func TestChatCmd_TokenFlagOutranksEnv(t *testing.T) {
	chatHermeticEnv(t)
	t.Setenv(chatcli.CmdIDEnvVar, "cmd-from-env")

	// The daemon is unreachable, so the call gets as far as dialing and no
	// further — which is exactly the point: it got past token resolution.
	_, _, err := runChatCmd(t, "read", "--token", "tok-flag", "--sock",
		t.TempDir()+"/absent.sock")
	assert.ErrorIs(t, err, chatcli.ErrDaemonUnreachable)
}

// The admin verbs are gated by the age identity file, not by a token, so they
// fail on a missing --identity even when a perfectly good token is exported.
func TestChatAdminVerbs_RequireAnIdentity(t *testing.T) {
	for _, args := range [][]string{
		{"admin", "list"},
		{"admin", "move", "/work", "backend/alice", "frontend"},
		{"admin", "register", "/work", "humans", "yuki"},
		{"admin", "send", "/work", "backend/alice", "hello"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			chatHermeticEnv(t)
			t.Setenv(chatcli.TokenEnvVar, "tok-a")

			stdout, _, err := runChatCmd(t, args...)
			assert.Assert(t, err != nil)
			assert.Assert(t, strings.Contains(err.Error(), "--identity"))
			assert.Equal(t, stdout, "")
		})
	}
}

// register's three coordinates are all required: a member registered into the
// wrong room is invisible to the people meant to talk to it.
func TestChatAdminRegister_RequiresRoomTeamAndName(t *testing.T) {
	chatHermeticEnv(t)

	_, _, err := runChatCmd(t, "admin", "register", "/work", "--identity", "/dev/null")
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "accepts 3 arg"))
}

func TestChatCmd_ArgumentShapes(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"send needs an address and a text", []string{"send", "alice"}},
		{"send takes no third argument", []string{"send", "alice", "hi", "extra"}},
		{"broadcast needs a text", []string{"broadcast"}},
		{"read takes no arguments", []string{"read", "extra"}},
		{"history takes no arguments", []string{"history", "extra"}},
		{"admin move needs three arguments", []string{"admin", "move", "/work", "backend/alice"}},
		{"admin send needs three arguments", []string{"admin", "send", "/work", "backend/alice"}},
		{"report-state needs a state", []string{"report-state"}},
		// An unknown state is rejected by the command itself, so a typo never
		// reaches the daemon as a report.
		{"report-state rejects an unknown state", []string{"report-state", "busy"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chatHermeticEnv(t)
			t.Setenv(chatcli.TokenEnvVar, "tok-a")

			_, _, err := runChatCmd(t, tc.args...)
			assert.Assert(t, err != nil)
		})
	}
}
