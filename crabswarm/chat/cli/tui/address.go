package tui

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// The screen addresses with `@`, and this file is the whole of that grammar:
// one tokenizer read two ways — once to find who a message being sent is for,
// and once to find where an arriving message names the admin.
//
// The admin holds no member row — they are the one at the screen — so being
// named is textual: they answer to `@admin`, which is how the daemon spells
// them in the log. There is no qualified spelling to answer to, since the
// operator is in no team.
const adminName = "admin"

// parseAddress reads a written message the way the room will: left to right, a
// backtick opening a span whose content is text, `\@` a literal `@` whose
// backslash does not travel, and every bare `@token` naming somebody the
// message is for. A token starts a word — the `@` at the start of the text or
// right after whitespace — and ends at whitespace or at the end of the text.
//
// The text comes back whole, the tokens included: they double as the mentions
// that name who was asked, which is what the room reads.
//
// One `@everyone` is the whole room. Any other run of tokens is the list of
// roles the message names, in the order they were written and without repeats.
// A message with no bare `@` in it names nobody, which is a board post: it
// reaches the room's log and interrupts no one — an `@` inside a word, as in an
// email address, is one of those.
//
// A bare `@` with nothing after it, and a token that is not a role, are refused
// here rather than sent: the daemon would answer NotFound, which reads as
// "nobody by that name" and sends the operator looking for a member instead of
// for the typo.
func parseAddress(text string) (*chatv1.Target, string, error) {
	var (
		runes   = []rune(text)
		b       strings.Builder
		span    bool
		written []string
		seen    = map[string]struct{}{}
	)
	for i := 0; i < len(runes); i++ {
		switch r := runes[i]; {
		case r == '`':
			span = !span
			b.WriteRune(r)
		case r == '\\' && i+1 < len(runes) && runes[i+1] == '@':
			b.WriteRune('@')
			i++
		case r == '@' && !span && wordStart(runes, i):
			j := tokenEnd(runes, i+1)
			token := string(runes[i+1 : j])
			if err := checkToken(token); err != nil {
				return nil, "", err
			}
			if _, dup := seen[token]; !dup {
				seen[token] = struct{}{}
				written = append(written, token)
			}
			b.WriteString(string(runes[i:j]))
			i = j - 1
		default:
			b.WriteRune(r)
		}
	}
	if len(written) == 0 {
		// Nobody was named, which is what a board post is. The empty target is
		// how the wire says so, and the empty written target is what parses
		// into it.
		return nil, b.String(), nil
	}
	// The tokens are handed over as the one comma-separated target the rest of
	// the CLI writes, so what a message addresses and what `chat admin send`
	// takes are read by the same parser rather than by two that may drift.
	target, err := cli.ParseTarget(strings.Join(written, ","))
	if err != nil {
		return nil, "", err
	}
	return target, b.String(), nil
}

// checkToken refuses the two tokens that are not addresses. The star gets its
// own refusal rather than the one [cli.ParseTarget] gives, because the operator
// typing it is typing the grammar this screen used to have and needs the `@`
// spelling of what replaced it.
func checkToken(token string) error {
	switch {
	case token == "":
		return errors.New(
			`a bare "@" addresses nobody: write @` + cli.EveryoneTarget +
				`, @name or @team/name, or \@ for a literal @`)
	case strings.Contains(token, "*"):
		return fmt.Errorf(
			"@%s is not a target: write @%s for the whole room, or name the roles",
			token, cli.EveryoneTarget)
	}
	return nil
}

// mentionsAdmin reports whether a message names the admin — a bare `@admin`
// token. Bare by [parseAddress]'s rules, since a mention is the same `@` a
// message is addressed with: a backticked or `\@`-escaped occurrence is text
// and names nobody, and so is an `@` inside a word.
func mentionsAdmin(text string) bool {
	return len(adminMentions([]rune(text))) > 0
}

// adminMentions is where a message names the admin: the rune ranges of its bare
// `@admin` tokens, each token's `@` included, so the pane can draw them apart
// from the rest of the line.
func adminMentions(runes []rune) [][2]int {
	var spans [][2]int
	var span bool
	for i := 0; i < len(runes); i++ {
		switch r := runes[i]; {
		case r == '`':
			span = !span
		case r == '\\' && i+1 < len(runes) && runes[i+1] == '@':
			i++
		case r == '@' && !span && wordStart(runes, i):
			j := tokenEnd(runes, i+1)
			if string(runes[i+1:j]) == adminName {
				spans = append(spans, [2]int{i, j})
			}
			i = j - 1
		}
	}
	return spans
}

// wordStart reports whether the rune at i opens a word: it is the first of the
// text, or whitespace is in front of it. This is the rule the completion reads
// a token by — everything back to the last space on the line — so what the
// screen offers to complete is what the grammar takes as an address, and
// `ops@corp.example` is one word and one piece of text.
func wordStart(runes []rune, i int) bool {
	return i == 0 || unicode.IsSpace(runes[i-1])
}

// tokenEnd is where the token starting at from ends: at the first space, or at
// the end of the text.
func tokenEnd(runes []rune, from int) int {
	for from < len(runes) && !unicode.IsSpace(runes[from]) {
		from++
	}
	return from
}
