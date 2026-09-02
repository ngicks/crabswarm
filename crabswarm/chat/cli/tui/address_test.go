package tui

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// The first bare `@token` says who the message is for, and the message goes
// whole — the token with it, since it is also the mention that names who was
// asked. A backtick span and a `\@` are text, and text names nobody.
func TestParseAddressReadsTheFirstBareToken(t *testing.T) {
	for _, tc := range []struct {
		name   string
		text   string
		target cli.AdminTarget
		out    string
	}{
		{
			name:   "the first token addresses, the second is text",
			text:   "@alpha/ana ask @alpha/bob for the token",
			target: cli.AdminTarget{Team: "alpha", Name: "ana"},
			out:    "@alpha/ana ask @alpha/bob for the token",
		},
		{
			name:   "a backticked token is text, and the backticks stay",
			text:   "ask `@here` who owns it",
			target: cli.AdminTarget{Everyone: true},
			out:    "ask `@here` who owns it",
		},
		{
			name:   "an escaped @ is a literal one and loses its backslash",
			text:   `send it to ops\@corp.example`,
			target: cli.AdminTarget{Everyone: true},
			out:    "send it to ops@corp.example",
		},
		{
			name:   "a team is addressable whole",
			text:   "@beta/* rebase",
			target: cli.AdminTarget{Team: "beta"},
			out:    "@beta/* rebase",
		},
		{
			name:   "a bare name is left for the daemon to resolve",
			text:   "@ana hi",
			target: cli.AdminTarget{Name: "ana"},
			out:    "@ana hi",
		},
		{
			name:   "no @ at all is the whole room",
			text:   "standup in five",
			target: cli.AdminTarget{Everyone: true},
			out:    "standup in five",
		},
		{
			name:   "a token ending at the end of the text still addresses",
			text:   "@alpha/ana",
			target: cli.AdminTarget{Team: "alpha", Name: "ana"},
			out:    "@alpha/ana",
		},
		{
			name:   "an @ inside a word is text, so a mail address broadcasts",
			text:   "send it to ops@corp.example",
			target: cli.AdminTarget{Everyone: true},
			out:    "send it to ops@corp.example",
		},
		{
			name:   "a newline is whitespace, so a token after one addresses",
			text:   "here is the plan\n@alpha/bob review it",
			target: cli.AdminTarget{Team: "alpha", Name: "bob"},
			out:    "here is the plan\n@alpha/bob review it",
		},
		{
			name:   "the star addresses the room as it does on argv",
			text:   "@* standup in five",
			target: cli.AdminTarget{Everyone: true},
			out:    "@* standup in five",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target, out, err := parseAddress(tc.text)
			assert.NilError(t, err)
			assert.Equal(t, target, tc.target)
			assert.Equal(t, out, tc.out)
		})
	}
}

// A message that names half an addressee is refused here rather than sent: the
// daemon's answer to it names no member, which reads as the wrong problem.
func TestParseAddressRefusesAHalfWrittenAddress(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
	}{
		{name: "a bare @ addresses nobody", text: "@"},
		{name: "a bare @ before a space addresses nobody", text: "@ ana hi"},
		{name: "a team with no member", text: "@beta/ rebase"},
		{name: "a member with no team", text: "@/ana hi"},
		{name: "a path where a name goes", text: "@a/b/c hi"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := parseAddress(tc.text)
			assert.Assert(t, err != nil, "%q was accepted", tc.text)
		})
	}
}

// The admin has no member row, so being named is textual: they answer to
// `@admin` and to `@admin/admin`, and to neither inside backticks nor escaped.
func TestMentionsAdmin(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		want bool
	}{
		{name: "the bare name", text: "@admin can you look", want: true},
		{name: "the qualified name", text: "@admin/admin hi", want: true},
		{name: "named after something else", text: "@alpha/ana ping @admin", want: true},
		{name: "named at the end of the text", text: "ask @admin", want: true},
		{name: "escaped", text: `\@admin`},
		{name: "backticked", text: "`@admin`"},
		{name: "another member", text: "@alpha/ana hi"},
		{name: "a longer name that starts the same", text: "@administrator hi"},
		{name: "an @ inside a word", text: "x@admin"},
		{name: "nobody", text: "standup in five"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, mentionsAdmin(tc.text), tc.want)
		})
	}
}
