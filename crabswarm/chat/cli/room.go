package cli

import "path/filepath"

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
