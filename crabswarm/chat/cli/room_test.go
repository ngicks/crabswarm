package cli

import (
	"testing"

	"gotest.tools/v3/assert"
)

// A directory belongs to the nearest room at or above it, and the room comes
// back spelled as listed so it can be handed to the daemon unchanged.
func TestRoomForDir(t *testing.T) {
	for _, tc := range []struct {
		name  string
		dir   string
		rooms []string
		want  string
		ok    bool
	}{
		{"exact match", "/work/proj", []string{"/work/other", "/work/proj"}, "/work/proj", true},
		{"ancestor match", "/work/proj/src/pkg", []string{"/work/proj"}, "/work/proj", true},
		{
			// The farther ancestor is listed first, so depth decides rather than
			// list order.
			"nearest of two ancestors wins",
			"/work/proj/src",
			[]string{"/work", "/work/proj"},
			"/work/proj",
			true,
		},
		{"no match", "/home/alice", []string{"/work/proj"}, "", false},
		{
			// A shared name prefix is not an ancestor.
			"sibling with a common prefix",
			"/work/project",
			[]string{"/work/proj"},
			"",
			false,
		},
		{"trailing slash in dir", "/work/proj/", []string{"/work/proj"}, "/work/proj", true},
		{"trailing slash in rooms", "/work/proj/src", []string{"/work/proj/"}, "/work/proj/", true},
		{"root as a room", "/etc/x", []string{"/"}, "/", true},
		{"root dir in root room", "/", []string{"/"}, "/", true},
		{"empty rooms", "/work/proj", nil, "", false},
		// A relative dir climbs to "." rather than "/", and the walk still ends.
		{"relative dir", "src/pkg", []string{"/work/proj"}, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := RoomForDir(tc.dir, tc.rooms)
			assert.Equal(t, got, tc.want)
			assert.Equal(t, ok, tc.ok)
		})
	}
}
