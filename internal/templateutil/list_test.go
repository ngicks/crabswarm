package templateutil

import (
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestSplitPath(t *testing.T) {
	sep := string(filepath.Separator)
	for _, tc := range []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"/", []string{"", ""}},
		{"/a/b/c", []string{"", "a", "b", "c"}},
		{"/a/b/c/", []string{"", "a", "b", "c"}},
		{"a/b//c/../d", []string{"a", "b", "d"}},
		{"rel", []string{"rel"}},
		{".", []string{"."}},
	} {
		in := filepath.FromSlash(tc.in)
		got := SplitPath(in)
		assert.DeepEqual(t, got, tc.want)
		if tc.in != "" {
			// A cleaned path round-trips through join.
			assert.Equal(t, strings.Join(got, sep), filepath.Clean(in), "round trip of %q", tc.in)
		}
	}
}

func TestLastN(t *testing.T) {
	for _, tc := range []struct {
		n    int
		in   any
		want any
	}{
		{3, []string{"a", "b", "c", "d"}, []string{"b", "c", "d"}},
		{4, []string{"a", "b", "c", "d"}, []string{"a", "b", "c", "d"}},
		{9, []string{"a", "b"}, []string{"a", "b"}},
		{0, []string{"a", "b"}, []string{}},
		{-2, []string{"a", "b"}, []string{}},
		{2, []int{1, 2, 3}, []int{2, 3}},
		{2, [3]string{"x", "y", "z"}, []string{"y", "z"}},
		{1, []string{}, []string{}},
		{1, nil, nil},
	} {
		got, err := LastN(tc.n, tc.in)
		assert.NilError(t, err, "LastN(%d, %v)", tc.n, tc.in)
		assert.DeepEqual(t, got, tc.want)
	}
}

func TestJoin(t *testing.T) {
	for _, tc := range []struct {
		sep  string
		in   any
		want string
	}{
		{"/", []string{"a", "b", "c"}, "a/b/c"},
		{"/", []string{"", "a"}, "/a"},
		{", ", []int{1, 2}, "1, 2"},
		{"-", [2]string{"x", "y"}, "x-y"},
		{"/", []string{}, ""},
		{"/", nil, ""},
	} {
		got, err := Join(tc.sep, tc.in)
		assert.NilError(t, err, "Join(%q, %v)", tc.sep, tc.in)
		assert.Equal(t, got, tc.want)
	}
}

func TestJoin_NonListErrors(t *testing.T) {
	_, err := Join("/", 42)
	assert.ErrorContains(t, err, "join: want a slice or array")
}

func TestLastN_NonListErrors(t *testing.T) {
	_, err := LastN(1, "not a list")
	assert.ErrorContains(t, err, "lastN: want a slice or array")
}
