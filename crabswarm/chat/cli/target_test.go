package cli

import (
	"testing"

	"gotest.tools/v3/assert"
)

// The four written forms name the four things an operator can address, and each
// is refused rather than sent where a half of it is missing: the daemon answers
// an address nothing holds with NotFound, which reads as "no such member".
func TestParseAdminTarget(t *testing.T) {
	for _, tc := range []struct {
		target string
		want   AdminTarget
	}{
		{"*", AdminTarget{Everyone: true}},
		{"backend/*", AdminTarget{Team: "backend"}},
		{"backend/alice", AdminTarget{Team: "backend", Name: "alice"}},
		{"alice", AdminTarget{Name: "alice"}},
	} {
		t.Run(tc.target, func(t *testing.T) {
			got, err := ParseAdminTarget(tc.target)
			assert.NilError(t, err)
			assert.DeepEqual(t, got, tc.want)
			// Every parsed target spells itself back as what was written, which is
			// what lets a rendered line be typed back in.
			assert.Equal(t, got.String(), tc.target)
		})
	}

	for _, target := range []string{"", "/alice", "backend/", "/", "a/b/c", "backend/*/x"} {
		t.Run("rejects "+target, func(t *testing.T) {
			_, err := ParseAdminTarget(target)
			assert.Assert(t, err != nil)
		})
	}
}
