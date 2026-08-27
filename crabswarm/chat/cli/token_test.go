package cli

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestResolveToken_Precedence(t *testing.T) {
	env := func(pairs map[string]string) func(string) (string, bool) {
		return func(name string) (string, bool) {
			v, ok := pairs[name]
			return v, ok
		}
	}

	for _, tc := range []struct {
		name  string
		flag  string
		pairs map[string]string
		want  string
	}{
		{
			name: "flag wins over both variables",
			flag: "from-flag",
			pairs: map[string]string{
				TokenEnvVar: "from-chat-token",
				CmdIDEnvVar: "from-cmd-id",
			},
			want: "from-flag",
		},
		{
			name:  "chat token wins over the cmdman id",
			pairs: map[string]string{TokenEnvVar: "from-chat-token", CmdIDEnvVar: "from-cmd-id"},
			want:  "from-chat-token",
		},
		{
			name:  "the cmdman id is the last resort",
			pairs: map[string]string{CmdIDEnvVar: "from-cmd-id"},
			want:  "from-cmd-id",
		},
		{
			// An exported-but-empty variable is not an identity; fall through
			// to the next source rather than sending an empty token.
			name:  "an empty variable is skipped",
			pairs: map[string]string{TokenEnvVar: "", CmdIDEnvVar: "from-cmd-id"},
			want:  "from-cmd-id",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveToken(tc.flag, env(tc.pairs))
			assert.NilError(t, err)
			assert.Equal(t, got, tc.want)
		})
	}
}

// With nothing to go on the error has to say what to do about it, since a human
// hitting this has a token to paste and an agent has a broken cmdman env.
func TestResolveToken_NoSourceNamesBoth(t *testing.T) {
	_, err := resolveToken("", func(string) (string, bool) { return "", false })
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "--token"))
	assert.Assert(t, strings.Contains(err.Error(), TokenEnvVar))
	assert.Assert(t, strings.Contains(err.Error(), CmdIDEnvVar))
}

// ResolveToken reads the process environment. Both variables are emptied rather
// than assumed absent: this suite may itself run under cmdman, which exports
// CMDMAN_CMD_ID.
func TestResolveToken_ReadsProcessEnv(t *testing.T) {
	t.Setenv(TokenEnvVar, "")
	t.Setenv(CmdIDEnvVar, "")
	_, err := ResolveToken("")
	assert.Assert(t, err != nil)

	t.Setenv(CmdIDEnvVar, "cmd-1234")
	got, err := ResolveToken("")
	assert.NilError(t, err)
	assert.Equal(t, got, "cmd-1234")
}
