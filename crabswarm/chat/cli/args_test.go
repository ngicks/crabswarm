package cli

import (
	"testing"

	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

func TestParseHarnessState(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want chatv1.HarnessState
	}{
		{"working", chatv1.HarnessState_HARNESS_STATE_WORKING},
		{"waiting", chatv1.HarnessState_HARNESS_STATE_WAITING},
		{"done", chatv1.HarnessState_HARNESS_STATE_DONE},
	} {
		got, err := ParseHarnessState(tc.in)
		assert.NilError(t, err)
		assert.Equal(t, got, tc.want)
	}

	// The words this vocabulary replaced parse as nothing at all: a hook still
	// wired to them fails loudly instead of reporting a state it did not mean.
	for _, bad := range []string{
		"", "DONE", "running", "idle", "waiting_input", "waiting-input", "unspecified", "busy",
	} {
		_, err := ParseHarnessState(bad)
		assert.Assert(t, err != nil, "state %q should not parse", bad)
	}
}

// The names offered for completion and argument validation must be exactly the
// ones the parser accepts, or a shell-completed argument would be rejected.
func TestHarnessStateNamesMatchParser(t *testing.T) {
	names := HarnessStateNames()
	assert.DeepEqual(t, names, []string{"working", "waiting", "done"})
	assert.Equal(t, len(names), len(harnessStates))
	for _, name := range names {
		_, err := ParseHarnessState(name)
		assert.NilError(t, err)
	}
}
