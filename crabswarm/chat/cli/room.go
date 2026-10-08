package cli

import (
	"context"
	"fmt"
	"path/filepath"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// RoomLister reports every room the daemon knows. [AdminClient] is one.
type RoomLister interface {
	Rooms(ctx context.Context) ([]*chatv1.Room, error)
}

// OpeningRoom returns the room a watch screen opens on and what to tell the
// operator about how it was chosen.
//
// A named room is returned as given, with nothing listed and no directory read:
// whether the daemon knows it is the screen's to check. With no room named, the
// working directory getwd reports is resolved by [ResolveRoom]. getwd is passed
// in so the choice can be driven without a process working directory to set.
func OpeningRoom(
	ctx context.Context,
	lister RoomLister,
	room string,
	getwd func() (string, error),
) (open, notice string, err error) {
	if room != "" {
		return room, "", nil
	}
	dir, err := getwd()
	if err != nil {
		return "", "", fmt.Errorf("reading the working directory: %w", err)
	}
	return ResolveRoom(ctx, lister, dir)
}

// RoomToFollow returns the room an operator's follow streams: room when named,
// else the room [ResolveRoom] finds for the working directory getwd reports.
//
// A directory under no room is followed itself rather than refused. The daemon
// follows a room nobody has used yet, and the agents that make it one may start
// in that directory after the follow did.
func RoomToFollow(
	ctx context.Context,
	lister RoomLister,
	room string,
	getwd func() (string, error),
) (string, error) {
	if room != "" {
		return room, nil
	}
	dir, err := getwd()
	if err != nil {
		return "", fmt.Errorf("reading the working directory: %w", err)
	}
	found, _, err := ResolveRoom(ctx, lister, dir)
	if err != nil {
		return "", err
	}
	if found == "" {
		return dir, nil
	}
	return found, nil
}

// ResolveRoom lists the rooms once and returns the one an operator standing in
// dir means, by [RoomForDir].
//
// No room at or above dir returns an empty room and a notice naming dir. The
// empty room leaves the choice to whatever the caller does when no room is
// named, and the notice is for the operator, who expected their project's room
// and is about to be shown another.
func ResolveRoom(
	ctx context.Context,
	lister RoomLister,
	dir string,
) (room, notice string, err error) {
	listed, err := lister.Rooms(ctx)
	if err != nil {
		return "", "", err
	}
	names := make([]string, len(listed))
	for i, r := range listed {
		names[i] = r.GetName()
	}
	if room, ok := RoomForDir(dir, names); ok {
		return room, "", nil
	}
	return "", "cwd " + dir + " matches no room", nil
}

// RoomForDir returns the nearest of dir and its ancestors that is one of rooms,
// and whether one matched.
//
// A room is named by the directory its members run in, and an admin standing
// in a subdirectory of that project still means the project's room, so the
// ancestors are walked rather than dir alone.
//
// Paths are compared cleaned, so a trailing slash on either side does not keep
// a room from matching. The room is returned as listed, not cleaned: the daemon
// keys a room by the working directory it was recorded with, and a respelled
// one names a room that does not exist. Where two listed rooms clean to the
// same path, the first listed is returned.
func RoomForDir(dir string, rooms []string) (string, bool) {
	cur := filepath.Clean(dir)
	for {
		for _, room := range rooms {
			if filepath.Clean(room) == cur {
				return room, true
			}
		}
		// Stopping where the parent is the directory itself, rather than at "/",
		// ends the walk for a relative dir too, which climbs to "." instead.
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", false
		}
		cur = parent
	}
}
