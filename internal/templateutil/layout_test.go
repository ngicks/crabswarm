package templateutil

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestPadRune(t *testing.T) {
	for _, tc := range []struct {
		n                   int
		in                  string
		wantLeft, wantRight string
	}{
		{5, "ab", "   ab", "ab   "},
		{2, "ab", "ab", "ab"},
		{1, "abc", "abc", "abc"},
		{0, "ab", "ab", "ab"},
		{-3, "ab", "ab", "ab"},
		{3, "", "   ", "   "},
		// Terminal cells, not runes or bytes: each CJK character is two.
		{5, "日本", " 日本", "日本 "},
		{4, "日本", "日本", "日本"},
		{6, "日本語です", "日本語です", "日本語です"},
		{5, "🙂", "   🙂", "🙂   "},
		// A combining mark takes no cells of its own.
		{3, "e\u0301", "  e\u0301", "e\u0301  "},
	} {
		assert.Equal(t, PadRuneLeft(tc.n, tc.in), tc.wantLeft, "PadRuneLeft(%d, %q)", tc.n, tc.in)
		assert.Equal(
			t,
			PadRuneRight(tc.n, tc.in),
			tc.wantRight,
			"PadRuneRight(%d, %q)",
			tc.n,
			tc.in,
		)
	}
}

func TestTruncRune(t *testing.T) {
	for _, tc := range []struct {
		n                   int
		in                  string
		wantLeft, wantRight string
	}{
		{2, "abcd", "cd", "ab"},
		{4, "abcd", "abcd", "abcd"},
		{9, "abcd", "abcd", "abcd"},
		{0, "abcd", "", ""},
		{-1, "abcd", "", ""},
		{3, "", "", ""},
		// Terminal cells, not runes: 4 cells is two CJK characters.
		{4, "日本語です", "です", "日本"},
		{10, "日本語です", "日本語です", "日本語です"},
		// A wide character straddling the cut is dropped, never split. On
		// the left a space keeps the width exact; on the right the result
		// comes out one cell short.
		{3, "日本語です", " す", "日"},
		{5, "aé日🙂", "é日🙂", "aé日"},
		{3, "aé日🙂", " 🙂", "aé"},
	} {
		assert.Equal(
			t,
			TruncRuneLeft(tc.n, tc.in),
			tc.wantLeft,
			"TruncRuneLeft(%d, %q)",
			tc.n,
			tc.in,
		)
		assert.Equal(
			t,
			TruncRuneRight(tc.n, tc.in),
			tc.wantRight,
			"TruncRuneRight(%d, %q)",
			tc.n,
			tc.in,
		)
	}
}

func TestColumns(t *testing.T) {
	for _, tc := range []struct {
		env  string
		want int
	}{
		{"120", 120},
		{"0", 0},
		{"", 0},
		{"wide", 0},
		{"-5", 0},
		{"80.5", 0},
	} {
		t.Setenv("COLUMNS", tc.env)
		assert.Equal(t, Columns(), tc.want, "COLUMNS=%q", tc.env)
	}
}
