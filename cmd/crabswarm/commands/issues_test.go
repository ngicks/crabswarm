package commands

import (
	"testing"

	"gotest.tools/v3/assert"
)

// The issues command tree is wired: `lint` sits under `issues` and carries
// every flag the documented command line offers, -C among them.
func TestIssuesCmd_Lint(t *testing.T) {
	root := rootCmd()
	lint, _, err := root.Find([]string{"issues", "lint"})
	assert.NilError(t, err)
	assert.Equal(t, lint.Name(), "lint")

	for _, name := range []string{"dir", "all", "limit", "json"} {
		assert.Assert(t, lint.Flags().Lookup(name) != nil, "flag --%s missing", name)
	}
	shorthand := lint.Flags().ShorthandLookup("C")
	assert.Assert(t, shorthand != nil, "-C missing")
	assert.Equal(t, shorthand.Name, "dir")
}
