package cli

import (
	"testing"

	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

func TestParseQualifiedName(t *testing.T) {
	for _, tc := range []struct {
		in         string
		team, name string
		wantErr    bool
	}{
		{in: "backend/alice", team: "backend", name: "alice"},
		{in: "a/b", team: "a", name: "b"},
		{in: "alice", wantErr: true},
		{in: "/alice", wantErr: true},
		{in: "backend/", wantErr: true},
		{in: "", wantErr: true},
		// Three segments are ambiguous rather than "team plus a name with a
		// slash": a name cannot hold one, so this is a typo, not an address.
		{in: "a/b/c", wantErr: true},
	} {
		t.Run(tc.in, func(t *testing.T) {
			team, name, err := ParseQualifiedName(tc.in)
			if tc.wantErr {
				assert.Assert(t, err != nil)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, team, tc.team)
			assert.Equal(t, name, tc.name)
		})
	}
}

func TestParseHarnessState(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want chatv1.HarnessState
	}{
		{"idle", chatv1.HarnessState_HARNESS_STATE_IDLE},
		{"running", chatv1.HarnessState_HARNESS_STATE_RUNNING},
		{"waiting_input", chatv1.HarnessState_HARNESS_STATE_WAITING_INPUT},
	} {
		got, err := ParseHarnessState(tc.in)
		assert.NilError(t, err)
		assert.Equal(t, got, tc.want)
	}

	for _, bad := range []string{"", "IDLE", "waiting-input", "unspecified", "busy"} {
		_, err := ParseHarnessState(bad)
		assert.Assert(t, err != nil, "state %q should not parse", bad)
	}
}

// The names offered for completion and argument validation must be exactly the
// ones the parser accepts, or a shell-completed argument would be rejected.
func TestHarnessStateNamesMatchParser(t *testing.T) {
	names := HarnessStateNames()
	assert.Equal(t, len(names), len(harnessStates))
	for _, name := range names {
		_, err := ParseHarnessState(name)
		assert.NilError(t, err)
	}
}
