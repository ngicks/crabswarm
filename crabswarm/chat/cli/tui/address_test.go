package tui

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// Every bare `@token` says who the message is for, and the message goes whole —
// the tokens with it, since they are also the mentions that name who was asked.
// A backtick span and a `\@` are text, and text names nobody.
func TestParseAddressReadsEveryBareToken(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		// target is how the parsed target reads back; a board post reads as "-"
		// rather than as the nothing it carries.
		target string
		out    string
	}{
		{
			name:   "two tokens are one list, in the order they were written",
			text:   "@alpha/ana ask @alpha/bob for the token",
			target: "alpha/ana,alpha/bob",
			out:    "@alpha/ana ask @alpha/bob for the token",
		},
		{
			name:   "the same role twice is named once",
			text:   "@ana and @ana again",
			target: "ana",
			out:    "@ana and @ana again",
		},
		{
			name:   "everyone is the whole room",
			text:   "@everyone standup in five",
			target: "everyone",
			out:    "@everyone standup in five",
		},
		{
			name:   "a backticked token is text, and the backticks stay",
			text:   "ask `@here` who owns it",
			target: "-",
			out:    "ask `@here` who owns it",
		},
		{
			name:   "an escaped @ is a literal one and loses its backslash",
			text:   `send it to ops\@corp.example`,
			target: "-",
			out:    "send it to ops@corp.example",
		},
		{
			name:   "a bare name is left for the daemon to resolve",
			text:   "@ana hi",
			target: "ana",
			out:    "@ana hi",
		},
		{
			name:   "no @ at all names nobody, which is a board post",
			text:   "standup in five",
			target: "-",
			out:    "standup in five",
		},
		{
			name:   "a token ending at the end of the text still addresses",
			text:   "@alpha/ana",
			target: "alpha/ana",
			out:    "@alpha/ana",
		},
		{
			name:   "an @ inside a word is text, so a mail address is a post",
			text:   "send it to ops@corp.example",
			target: "-",
			out:    "send it to ops@corp.example",
		},
		{
			name:   "a newline is whitespace, so a token after one addresses",
			text:   "here is the plan\n@alpha/bob review it",
			target: "alpha/bob",
			out:    "here is the plan\n@alpha/bob review it",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target, out, err := parseAddress(tc.text)
			assert.NilError(t, err)
			assert.Equal(t, cli.TargetString(target), tc.target)
			assert.Equal(t, out, tc.out)
		})
	}
}

// A board post carries no target at all, which is how the wire says "for
// nobody": the column it reads back as is the rendering, not the value.
func TestParseAddressPostCarriesNoTarget(t *testing.T) {
	target, out, err := parseAddress("standup in five")
	assert.NilError(t, err)
	assert.Assert(t, target == nil)
	assert.Equal(t, out, "standup in five")
}

// A message that names half an addressee is refused here rather than sent: the
// daemon's answer to it names no member, which reads as the wrong problem. The
// star the screen used to take is refused in the `@` spelling of what replaced
// it, since the operator typing one is typing the old grammar.
func TestParseAddressRefusesAHalfWrittenAddress(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		// says is what the message has to name, so the operator knows what to
		// fix.
		says string
	}{
		{name: "a bare @ addresses nobody", text: "@", says: "@everyone"},
		{name: "a bare @ before a space", text: "@ ana hi", says: "@everyone"},
		{name: "a team with no member", text: "@beta/ rebase", says: "team/name"},
		{name: "a member with no team", text: "@/ana hi", says: "team/name"},
		{name: "a path where a name goes", text: "@a/b/c hi", says: "team/name"},
		{name: "a whole team", text: "@beta/* rebase", says: "@everyone"},
		{name: "the old star", text: "@* standup in five", says: "@everyone"},
		{
			name: "everyone among named roles",
			text: "@everyone @alpha/ana hi",
			says: "everyone",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := parseAddress(tc.text)
			assert.Assert(t, err != nil, "%q was accepted", tc.text)
			assert.Assert(t, strings.Contains(err.Error(), tc.says),
				"error %q does not name %q", err, tc.says)
		})
	}
}

// The admin has no member row, so being named is textual: they answer to
// `@admin`, and to neither a backticked one nor an escaped one.
func TestMentionsAdmin(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		want bool
	}{
		{name: "the bare name", text: "@admin can you look", want: true},
		{name: "named after something else", text: "@alpha/ana ping @admin", want: true},
		{name: "named at the end of the text", text: "ask @admin", want: true},
		{name: "escaped", text: `\@admin`},
		{name: "backticked", text: "`@admin`"},
		{name: "another member", text: "@alpha/ana hi"},
		{name: "a longer name that starts the same", text: "@administrator hi"},
		// The operator is in no team, so a qualified spelling is a role in a
		// team called admin and names somebody else entirely.
		{name: "a qualified name", text: "@admin/admin hi"},
		{name: "an @ inside a word", text: "x@admin"},
		{name: "nobody", text: "standup in five"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, mentionsAdmin(tc.text), tc.want)
		})
	}
}
