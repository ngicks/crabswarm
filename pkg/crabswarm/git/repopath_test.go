package git

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseRepoPath(t *testing.T) {
	for _, tc := range []struct {
		in, want string
	}{
		{"https://github.com/owner/repo", "github.com/owner/repo"},
		{"https://github.com/owner/repo.git", "github.com/owner/repo"},
		{"https://github.com/owner/repo/", "github.com/owner/repo"},
		{"http://example.com/owner/repo", "example.com/owner/repo"},
		{"https://user@github.com/owner/repo.git", "github.com/owner/repo"},
		{"https://github.com:443/owner/repo.git", "github.com/owner/repo"},
		{"ssh://git@github.com/owner/repo.git", "github.com/owner/repo"},
		{"ssh://git@github.com:22/owner/repo.git", "github.com/owner/repo"},
		{"git@github.com:owner/repo.git", "github.com/owner/repo"},
		{"git@github.com:owner/repo", "github.com/owner/repo"},
		{"github.com/owner/repo", "github.com/owner/repo"},
		{"github.com/owner/repo.git", "github.com/owner/repo"},
		// Deep namespaces (GitLab subgroups) survive.
		{"https://gitlab.com/group/subgroup/repo.git", "gitlab.com/group/subgroup/repo"},
		{"git@gitlab.com:group/subgroup/repo.git", "gitlab.com/group/subgroup/repo"},
		{"  https://github.com/owner/repo  ", "github.com/owner/repo"},
	} {
		got, err := ParseRepoPath(tc.in)
		assert.NilError(t, err, "input %q", tc.in)
		assert.Equal(t, got, tc.want, "input %q", tc.in)
	}
}

func TestParseRepoPath_Errors(t *testing.T) {
	for _, in := range []string{
		"",
		"   ",
		"https://github.com/",      // no path
		"https://github.com",       // no path
		"file:///tmp/local/origin", // no host
		"owner",                    // no host/path split
	} {
		_, err := ParseRepoPath(in)
		assert.Assert(t, err != nil, "expected error for %q", in)
	}
}

func TestIsSCPLike(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"git@github.com:owner/repo", true},
		{"github.com:owner/repo", true},
		{"https://github.com/owner/repo", false},
		{"ssh://git@github.com/owner/repo", false},
		{"github.com/owner/repo", false},
		{"plainstring", false},
	} {
		assert.Equal(t, isSCPLike(tc.in), tc.want, "input %q", tc.in)
	}
}
