package cli

import (
	"errors"
	"fmt"
	"strings"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// harnessStates maps the words the CLI accepts for a harness state onto the
// enum, mirroring the vocabulary of `cmdman status set working|waiting|done`.
// The unspecified state has no word: a hook that cannot say what its harness
// is doing must fail rather than report the one state a keystroke nudge is
// sent to on sight rather than only once the report has gone stale.
var harnessStates = map[string]chatv1.HarnessState{
	"working": chatv1.HarnessState_HARNESS_STATE_WORKING,
	"waiting": chatv1.HarnessState_HARNESS_STATE_WAITING,
	"done":    chatv1.HarnessState_HARNESS_STATE_DONE,
}

// HarnessStateNames returns the accepted harness-state words, in the order they
// are documented. The command wiring uses them for argument validation and
// shell completion, so the two cannot drift from what [ParseHarnessState]
// accepts.
func HarnessStateNames() []string {
	return []string{"working", "waiting", "done"}
}

// ParseHarnessState maps a state word onto the reported enum.
func ParseHarnessState(s string) (chatv1.HarnessState, error) {
	state, ok := harnessStates[s]
	if !ok {
		return chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED,
			fmt.Errorf("unknown harness state %q: want one of %s",
				s, strings.Join(HarnessStateNames(), ", "))
	}
	return state, nil
}

// HarnessStateName spells a reported state as the word [ParseHarnessState]
// takes, so a caller presenting a member's state and a caller setting one use
// the same three words. A state outside them — the unspecified one, which is
// what a member built without a state carries — reads as "unknown", since that
// is what it tells whoever is reading.
func HarnessStateName(state chatv1.HarnessState) string {
	for _, name := range HarnessStateNames() {
		if harnessStates[name] == state {
			return name
		}
	}
	return "unknown"
}

// ParseAddressedLine splits the "to: text" form a caller types when the
// address and the message arrive as one line instead of as two arguments — a
// screen with a single input line rather than a shell with a quoted argv.
//
// The address is whatever precedes the first colon: a bare name, a "team/name"
// pair, or the "*" the admin verbs take for everyone in the room. It is handed
// back untouched, the way `chat send` hands its argument on, because resolving
// an address is the daemon's job and its refusal names the form to retry with.
// Later colons belong to the message, which is where the ones in a URL or a
// timestamp end up.
func ParseAddressedLine(line string) (to, text string, err error) {
	to, text, ok := strings.Cut(line, ":")
	to, text = strings.TrimSpace(to), strings.TrimSpace(text)
	switch {
	case !ok:
		return "", "", fmt.Errorf(
			"%q names no addressee: write it as \"name: text\", \"team/name: text\" or \"*: text\"",
			line)
	case to == "":
		return "", "", errors.New(`no addressee before the ":"`)
	case text == "":
		return "", "", fmt.Errorf("nothing to say to %s", to)
	}
	return to, text, nil
}

// ParseQualifiedName splits the "team/name" form the admin verbs address a
// member by. Admin RPCs carry team and name as separate fields, but a member is
// written as one word everywhere else — in `chat members` output and in the
// address `chat send` takes — so the CLI keeps that spelling and splits here.
func ParseQualifiedName(s string) (team, name string, err error) {
	team, name, ok := strings.Cut(s, "/")
	if !ok || team == "" || name == "" || strings.Contains(name, "/") {
		return "", "", fmt.Errorf("member %q is not in \"team/name\" form", s)
	}
	return team, name, nil
}
