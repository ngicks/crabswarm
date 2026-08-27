package server

import (
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestExpandHome(t *testing.T) {
	t.Setenv("HOME", "/home/someone")

	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{"bare tilde", "~", "/home/someone"},
		{"tilde path", "~/.local/state/crabswarm/chat.db",
			filepath.Join("/home/someone", ".local", "state", "crabswarm", "chat.db")},
		{"absolute path is untouched", "/var/lib/crabswarm/chat.db", "/var/lib/crabswarm/chat.db"},
		// "~user" is another user's home, which this expansion deliberately
		// does not resolve; leaving it alone fails at open rather than
		// silently pointing somewhere else.
		{"other user is untouched", "~other/chat.db", "~other/chat.db"},
		{"in-memory store is untouched", ":memory:", ":memory:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := expandHome(tc.in)
			assert.NilError(t, err)
			assert.Equal(t, got, tc.want)
		})
	}
}
