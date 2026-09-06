package issues

import (
	"testing"

	"gotest.tools/v3/assert"
)

// fixtureBeadsPath is the .beads directory testdata/where.json reports, which
// the fake bd replays for every directory.
const fixtureBeadsPath = "/home/watage/gitrepo/github.com/ngicks/crabswarm/.beads"

func TestSourceStoreAdd(t *testing.T) {
	invocations := installFakeBd(t)
	dir := t.TempDir()

	store := NewSourceStore()
	src, added, err := store.Add(t.Context(), dir)
	assert.NilError(t, err)
	assert.Assert(t, added)
	assert.Equal(t, src.BeadsPath, fixtureBeadsPath)
	assert.Equal(t, src.Prefix, "crabswarm")
	assert.Equal(t, src.Dir, dir)
	assert.Equal(t, src.ID, sourceID(fixtureBeadsPath))

	inv := invocations()
	assert.Equal(t, len(inv), 1)
	assert.Equal(t, inv[0].args, "where --json")
	assert.Equal(t, inv[0].dir, dir)
	// The envelope is what turns "no beads database" into ErrNoBeads.
	assert.Equal(t, inv[0].envelope, "1")
}

func TestSourceStoreAddSameDatabaseOnce(t *testing.T) {
	installFakeBd(t)

	store := NewSourceStore()
	first, _, err := store.Add(t.Context(), t.TempDir())
	assert.NilError(t, err)

	// A second worktree of one repository reports the same .beads directory,
	// so it registers as the same source rather than a second one.
	second, added, err := store.Add(t.Context(), t.TempDir())
	assert.NilError(t, err)
	assert.Assert(t, !added)
	assert.Equal(t, second.ID, first.ID)
	assert.Equal(t, second.Dir, first.Dir)
	assert.Equal(t, len(store.List()), 1)
}

func TestSourceStoreAddNoBeads(t *testing.T) {
	installFakeBd(t)
	t.Setenv("FAKE_BD_NO_BEADS", "1")

	_, _, err := NewSourceStore().Add(t.Context(), t.TempDir())
	assert.ErrorIs(t, err, ErrNoBeads)
}

func TestSourceStoreGetRemove(t *testing.T) {
	installFakeBd(t)

	store := NewSourceStore()
	src, _, err := store.Add(t.Context(), t.TempDir())
	assert.NilError(t, err)

	got, ok := store.Get(src.ID)
	assert.Assert(t, ok)
	assert.Equal(t, got.ID, src.ID)

	removed, ok := store.Remove(src.ID)
	assert.Assert(t, ok)
	assert.Equal(t, removed.ID, src.ID)
	assert.Equal(t, len(store.List()), 0)

	_, ok = store.Remove(src.ID)
	assert.Assert(t, !ok)
}
