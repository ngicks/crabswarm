package cli

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

func TestParseReadCursor(t *testing.T) {
	for word, want := range map[string]chatv1.ReadCursor{
		"unread": chatv1.ReadCursor_READ_CURSOR_UNREAD,
		"head":   chatv1.ReadCursor_READ_CURSOR_HEAD,
		"tail":   chatv1.ReadCursor_READ_CURSOR_TAIL,
		// No cursor at all leaves the default to the read being made: past the
		// caller's position for a member, the room's tail for an operator.
		"": chatv1.ReadCursor_READ_CURSOR_UNSPECIFIED,
	} {
		t.Run(word, func(t *testing.T) {
			got, err := ParseReadCursor(word)
			assert.NilError(t, err)
			assert.Equal(t, got, want)
		})
	}

	_, err := ParseReadCursor("newest")
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "unread"))
}

// The flags are carried as written, the range included: zero is ten in the
// cursor's own direction, which only the daemon knows how to count.
func TestReadFlags_Filter(t *testing.T) {
	got, err := ReadFlags{Cursor: "tail", To: "backend/alice", Since: 5, Until: 9}.Filter()
	assert.NilError(t, err)
	assert.Equal(t, got.GetCursor(), chatv1.ReadCursor_READ_CURSOR_TAIL)
	assert.Equal(t, got.GetRange(), int32(0))
	assert.Equal(t, TargetString(got.GetTo()), "backend/alice")
	assert.Equal(t, got.GetSince(), int64(5))
	assert.Equal(t, got.GetUntil(), int64(9))

	// An unwritten --to narrows nothing: the read keeps every target, posts
	// included, rather than keeping only posts.
	got, err = ReadFlags{Cursor: "unread", Range: 5}.Filter()
	assert.NilError(t, err)
	assert.Assert(t, got.GetTo() == nil)
	assert.Equal(t, got.GetRange(), int32(5))
}

func TestReadFlags_FilterRejectsAMalformedTarget(t *testing.T) {
	_, err := ReadFlags{Cursor: "head", To: "backend/"}.Filter()
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "team/name"))
}

// An admin read has no read position, so unread is refused before a challenge
// is spent on it, in the words of the flag the operator typed.
func TestReadFlags_HistoryFilterRefusesUnread(t *testing.T) {
	_, err := ReadFlags{Cursor: "unread"}.HistoryFilter()
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "--cursor head or tail"))

	got, err := ReadFlags{Cursor: "tail", Range: -20}.HistoryFilter()
	assert.NilError(t, err)
	assert.Equal(t, got.GetCursor(), chatv1.ReadCursor_READ_CURSOR_TAIL)
	assert.Equal(t, got.GetRange(), int32(-20))
}
