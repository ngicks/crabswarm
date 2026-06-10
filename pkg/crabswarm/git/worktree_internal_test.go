package git

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseWorktreeList(t *testing.T) {
	out := "worktree /repo/.bare\n" +
		"bare\n" +
		"\n" +
		"worktree /repo/main\n" +
		"HEAD abc123\n" +
		"branch refs/heads/main\n" +
		"\n" +
		"worktree /repo/detached\n" +
		"HEAD def456\n" +
		"detached\n"

	got := parseWorktreeList(out)
	assert.Equal(t, len(got), 3)

	assert.Equal(t, got[0].Path, "/repo/.bare")
	assert.Assert(t, got[0].Bare)
	assert.Equal(t, got[0].Branch, "")

	assert.Equal(t, got[1].Path, "/repo/main")
	assert.Equal(t, got[1].Branch, "main")
	assert.Equal(t, got[1].Name(), "main")
	assert.Assert(t, !got[1].Bare)

	assert.Equal(t, got[2].Path, "/repo/detached")
	assert.Equal(t, got[2].Branch, "")
	assert.Assert(t, !got[2].Bare)
}
