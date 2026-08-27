package cli

import (
	"fmt"
	"strings"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// harnessStates maps the words the CLI accepts for a harness state onto the
// enum, mirroring the vocabulary of `cmdman status set working|waiting|done`.
// The unspecified state has no word: a hook that cannot say what its harness
// is doing must fail rather than report the one state that invites a
// keystroke nudge.
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
