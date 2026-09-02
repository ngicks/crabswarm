package templateutil

import (
	"os"
	"strconv"

	"github.com/mattn/go-runewidth"
)

// This file holds the terminal-layout helpers used by the status line
// renderer: fixed-width padding and truncation, and the terminal width from
// $COLUMNS. Widths are terminal cells as computed by go-runewidth, not runes
// or bytes: a CJK character or an emoji occupies two cells, a combining mark
// zero, and East-Asian-ambiguous characters follow the locale ($LC_ALL,
// $LANG, $RUNEWIDTH_EASTASIAN).

// PadRuneLeft returns s prepended with spaces until it occupies n terminal
// cells, i.e. right-aligned in a field n cells wide. A string already n
// cells or wider is returned unchanged; padding never truncates.
func PadRuneLeft(n int, s string) string {
	return runewidth.FillLeft(s, n)
}

// PadRuneRight returns s followed by spaces until it occupies n terminal
// cells, i.e. left-aligned in a field n cells wide. A string already n cells
// or wider is returned unchanged; padding never truncates.
func PadRuneRight(n int, s string) string {
	return runewidth.FillRight(s, n)
}

// TruncRuneLeft drops characters from the left of s until it fits in n
// terminal cells, i.e. it keeps the rightmost n cells. It is the counterpart
// of PadRuneLeft: both act on the left edge of s. The cut never splits a
// wide character; when one straddles the cut it is dropped and a leading
// space keeps the result exactly n cells wide. A string already n cells or
// narrower is returned unchanged; n <= 0 yields "".
func TruncRuneLeft(n int, s string) string {
	if n <= 0 {
		return ""
	}
	drop := runewidth.StringWidth(s) - n
	if drop <= 0 {
		return s
	}
	return runewidth.TruncateLeft(s, drop, "")
}

// TruncRuneRight drops characters from the right of s until it fits in n
// terminal cells, i.e. it keeps the leftmost n cells. It is the counterpart
// of PadRuneRight: both act on the right edge of s. The cut never splits a
// wide character; when one straddles the cut it is dropped, so the result
// may be one cell narrower than n. A string already n cells or narrower is
// returned unchanged; n <= 0 yields "".
func TruncRuneRight(n int, s string) string {
	if n <= 0 {
		return ""
	}
	return runewidth.Truncate(s, n, "")
}

// Columns returns the terminal width advertised by the COLUMNS environment
// variable as an integer, or 0 when it is unset, empty or not a non-negative
// integer. Claude Code sets COLUMNS (and LINES) to the terminal dimensions
// before running the status line command; an interactive shell keeps them as
// unexported variables, so a hand-run render usually sees 0. The int return
// (rather than the string [os.Getenv] gives) lets a template gate on it with
// {{ if columns }} and compare it with lt/gt.
func Columns() int {
	n, err := strconv.Atoi(os.Getenv("COLUMNS"))
	if err != nil || n < 0 {
		return 0
	}
	return n
}
