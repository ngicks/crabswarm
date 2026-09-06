package issues

import (
	"errors"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestWhere(t *testing.T) {
	invocations := installFakeBd(t)
	dir := t.TempDir()

	loc, err := Where(t.Context(), dir)
	assert.NilError(t, err)
	assert.Equal(t, loc.BeadsPath, "/home/watage/gitrepo/github.com/ngicks/crabswarm/.beads")
	assert.Equal(
		t,
		loc.DatabasePath,
		"/home/watage/gitrepo/github.com/ngicks/crabswarm/.beads/embeddeddolt",
	)
	assert.Equal(t, loc.Prefix, "crabswarm")

	got := invocations()
	assert.Equal(t, len(got), 1)
	assert.Equal(t, got[0].args, "where --json")
	assert.Equal(t, got[0].dir, dir)
	// The envelope is what makes a missing workspace machine-readable.
	assert.Equal(t, got[0].envelope, "1")
}

func TestWhereNoBeads(t *testing.T) {
	installFakeBd(t)
	t.Setenv("FAKE_BD_NO_BEADS", "1")

	_, err := Where(t.Context(), t.TempDir())
	assert.Assert(t, errors.Is(err, ErrNoBeads), "got %v", err)
	assert.Assert(
		t,
		strings.Contains(err.Error(), "No active beads workspace found."),
		"got %v",
		err,
	)
}
