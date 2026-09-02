package cli

import (
	"os"
	"strings"
)

// VisualEnvVar and EditorEnvVar name the editor the operator has already told
// their shell about. VISUAL is read first: the convention is that EDITOR may
// name a line editor for a dumb terminal, and the screen hands its draft to a
// full-screen program on a terminal it has just released.
const (
	VisualEnvVar = "VISUAL"
	EditorEnvVar = "EDITOR"
)

// EditorFromEnv is the editor command the admin TUI's ctrl+g runs: $VISUAL,
// else $EDITOR, else "" — which the screen reports rather than guessing at a
// vi that may not be installed.
//
// The value is a command line, not a program: `code -w` and `emacsclient -nw`
// are both ordinary contents of these variables, so whoever runs it splits it
// the way a shell would.
//
// This is the one place in the binary those two variables are read, which is
// what keeps ./cmd free of os.Getenv (see crabswarm/config.go).
func EditorFromEnv() string {
	for _, name := range []string{VisualEnvVar, EditorEnvVar} {
		// Trimmed because a variable set to spaces is a variable set to
		// nothing, and the caller's only test is whether this is empty.
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	return ""
}
