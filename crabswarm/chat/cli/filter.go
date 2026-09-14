package cli

import (
	"fmt"
	"strings"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// readCursors maps the words a read takes onto the cursor it means. The
// unspecified cursor has no word: leaving the cursor out is how a caller asks
// for the default of the read it is making, which the daemon fills in.
var readCursors = map[string]chatv1.ReadCursor{
	"unread": chatv1.ReadCursor_READ_CURSOR_UNREAD,
	"head":   chatv1.ReadCursor_READ_CURSOR_HEAD,
	"tail":   chatv1.ReadCursor_READ_CURSOR_TAIL,
}

// ReadCursorNames returns the cursor words a member read takes, in the order
// they are documented. The command wiring uses them for shell completion, so
// the offered words cannot drift from what [ParseReadCursor] accepts.
func ReadCursorNames() []string {
	return []string{"unread", "head", "tail"}
}

// HistoryCursorNames returns the cursor words an admin read takes: unread is
// measured from a role's read position, and an operator reading a room they do
// not attend has none.
func HistoryCursorNames() []string {
	return []string{"head", "tail"}
}

// ParseReadCursor maps a cursor word onto the enum. The empty string is no
// cursor at all, which leaves the default to the read being made.
func ParseReadCursor(s string) (chatv1.ReadCursor, error) {
	if s == "" {
		return chatv1.ReadCursor_READ_CURSOR_UNSPECIFIED, nil
	}
	cursor, ok := readCursors[s]
	if !ok {
		return chatv1.ReadCursor_READ_CURSOR_UNSPECIFIED,
			fmt.Errorf("unknown read cursor %q: want one of %s",
				s, strings.Join(ReadCursorNames(), ", "))
	}
	return cursor, nil
}

// ReadFlags is a read as its caller writes it: the five values the member read
// and the admin read both take, in the spelling a person types and a tool
// passes. [ReadFlags.Filter] turns it into the request field.
//
// Range is signed and counts from the cursor — forward when positive, backward
// when negative. Zero is left at zero rather than filled in here: the daemon
// reads it as ten in the cursor's own direction, which is the only place that
// knows which direction a cursor runs in.
type ReadFlags struct {
	Cursor string
	Range  int32
	To     string
	Since  int64
	Until  int64
}

// Filter builds the filter a member read carries.
func (f ReadFlags) Filter() (*chatv1.ReadFilter, error) {
	cursor, err := ParseReadCursor(f.Cursor)
	if err != nil {
		return nil, err
	}
	// An empty --to is no narrowing at all: the read keeps every target, posts
	// included. The same empty string means a board post on a send, which is
	// the wire's own answer to "who is this for" in both places.
	to, err := ParseTarget(f.To)
	if err != nil {
		return nil, err
	}
	return &chatv1.ReadFilter{
		Cursor: cursor,
		Range:  f.Range,
		To:     to,
		Since:  f.Since,
		Until:  f.Until,
	}, nil
}

// HistoryFilter builds the filter an admin read carries, which is the member
// one without the unread cursor.
//
// The refusal is made here rather than left to the daemon so it costs no
// challenge round trip, and so the operator is told in the words of the flag
// they typed.
func (f ReadFlags) HistoryFilter() (*chatv1.ReadFilter, error) {
	filter, err := f.Filter()
	if err != nil {
		return nil, err
	}
	if filter.GetCursor() == chatv1.ReadCursor_READ_CURSOR_UNREAD {
		return nil, fmt.Errorf(
			"an admin read attends no room and has no read position to count "+
				"unread from: write --cursor %s",
			strings.Join(HistoryCursorNames(), " or "))
	}
	return filter, nil
}
