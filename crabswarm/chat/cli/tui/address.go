package tui

import (
	"errors"
	"strings"
	"unicode"

	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// The screen addresses with `@`, and this file is the whole of that grammar:
// one tokenizer read two ways — once to find who a message being sent is for,
// and once to find where an arriving message names the admin.
//
// The admin holds no member row — they are the one at the screen — so being
// named is textual: they answer to `@admin` and to `@admin/admin`, which is how
// the daemon spells them in the log.
const (
	adminName          = "admin"
	adminQualifiedName = "admin/admin"
)

// parseAddress reads a written message the way the room will: left to right, a
// backtick opening a span whose content is text, `\@` a literal `@` whose
// backslash does not travel, and the first bare `@token` naming who the message
// is for. A token ends at whitespace or at the end of the text, and later `@`s
// are text — one message has one addressee.
//
// The text comes back whole, the target's token included: it doubles as the
// mention that names who was asked, which is what the room reads.
//
// A message with no bare `@` in it is for everyone in the room, which is what
// the operator writing to the room without naming anyone means. A bare `@` with
// nothing after it, or a token that is not an address, is refused here rather
// than sent: the daemon would answer NotFound, which reads as "nobody by that
// name" and sends the operator looking for a member instead of for the typo.
func parseAddress(text string) (cli.AdminTarget, string, error) {
	var (
		runes  = []rune(text)
		b      strings.Builder
		span   bool
		target cli.AdminTarget
		found  bool
	)
	for i := 0; i < len(runes); i++ {
		switch r := runes[i]; {
		case r == '`':
			span = !span
			b.WriteRune(r)
		case r == '\\' && i+1 < len(runes) && runes[i+1] == '@':
			b.WriteRune('@')
			i++
		case r == '@' && !span && !found:
			j := tokenEnd(runes, i+1)
			written := string(runes[i+1 : j])
			if written == "" {
				return cli.AdminTarget{}, "", errors.New(
					`a bare "@" addresses nobody: write @name, @team/name or ` +
						`@team/*, or \@ for a literal @`)
			}
			parsed, err := cli.ParseAdminTarget(written)
			if err != nil {
				return cli.AdminTarget{}, "", err
			}
			target, found = parsed, true
			b.WriteString(string(runes[i:j]))
			i = j - 1
		default:
			b.WriteRune(r)
		}
	}
	if !found {
		// Nobody was named, so the message is the room's.
		target = cli.AdminTarget{Everyone: true}
	}
	return target, b.String(), nil
}

// mentionsAdmin reports whether a message names the admin — a bare `@admin` or
// `@admin/admin` token. Bare by [parseAddress]'s rules, since a mention is the
// same `@` a message is addressed with: a backticked or `\@`-escaped occurrence
// is text and names nobody.
func mentionsAdmin(text string) bool {
	return len(adminMentions([]rune(text))) > 0
}

// adminMentions is where a message names the admin: the rune ranges of its bare
// `@admin` and `@admin/admin` tokens, each token's `@` included, so the pane can
// draw them apart from the rest of the line.
func adminMentions(runes []rune) [][2]int {
	var spans [][2]int
	var span bool
	for i := 0; i < len(runes); i++ {
		switch r := runes[i]; {
		case r == '`':
			span = !span
		case r == '\\' && i+1 < len(runes) && runes[i+1] == '@':
			i++
		case r == '@' && !span:
			j := tokenEnd(runes, i+1)
			if tok := string(runes[i+1 : j]); tok == adminName || tok == adminQualifiedName {
				spans = append(spans, [2]int{i, j})
			}
			i = j - 1
		}
	}
	return spans
}

// tokenEnd is where the token starting at from ends: at the first space, or at
// the end of the text.
func tokenEnd(runes []rune, from int) int {
	for from < len(runes) && !unicode.IsSpace(runes[from]) {
		from++
	}
	return from
}
