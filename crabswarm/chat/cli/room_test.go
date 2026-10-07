package cli

import (
	"context"
	"errors"
	"testing"

	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// fakeLister answers a listing with the rooms named, or with err.
type fakeLister struct {
	rooms []string
	err   error
	calls int
}

func (f *fakeLister) Rooms(context.Context) ([]*chatv1.Room, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	listed := make([]*chatv1.Room, len(f.rooms))
	for i, name := range f.rooms {
		listed[i] = &chatv1.Room{Name: name}
	}
	return listed, nil
}

// The room an operator means is the nearest one at or above where they stand,
// read off one listing. Standing outside every room names none and says so,
// naming the directory that matched nothing.
func TestResolveRoom(t *testing.T) {
	lister := &fakeLister{rooms: []string{"/work/other", "/work/proj"}}

	room, notice, err := ResolveRoom(t.Context(), lister, "/work/proj/src")
	assert.NilError(t, err)
	assert.Equal(t, room, "/work/proj")
	assert.Equal(t, notice, "")
	assert.Equal(t, lister.calls, 1)

	room, notice, err = ResolveRoom(t.Context(), lister, "/home/alice")
	assert.NilError(t, err)
	assert.Equal(t, room, "")
	assert.Equal(t, notice, "cwd /home/alice matches no room")

	room, notice, err = ResolveRoom(t.Context(), &fakeLister{}, "/work/proj")
	assert.NilError(t, err)
	assert.Equal(t, room, "")
	assert.Equal(t, notice, "cwd /work/proj matches no room")

	refused := errors.New("decrypting the admin challenge")
	_, _, err = ResolveRoom(t.Context(), &fakeLister{err: refused}, "/work/proj")
	assert.ErrorIs(t, err, refused)
}

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
