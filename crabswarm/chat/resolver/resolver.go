// Package resolver places the holder of an identity token in the chat
// topology: which room they talk in and which team namespaces their name.
//
// It is the boundary between crabswarm and whatever actually knows about
// running commands, so it stays free of the chat broker's own machinery — the
// broker consumes a resolver, never the other way round.
package resolver

import (
	"errors"
)

// ErrUnknownToken reports that a resolver cannot place the token it was given:
// either it names nothing the resolver knows about, or what it names carries
// no team coordination information.
//
// It is deliberately distinct from a lookup that merely failed. A caller may
// reject the join and reap the member behind an unknown token, but must keep
// the member across a failed lookup — a missing cmdman, a locked store or a
// cancelled context says nothing about whether the token is still valid.
var ErrUnknownToken = errors.New("unknown token")

// TeamInfo is where the holder of an identity token belongs in the chat
// topology.
type TeamInfo struct {
	// Room is the working directory of the command that reported the token.
	// Everything running in the same directory shares a room.
	Room string
	// Team is the name of the compose project the command belongs to. A team
	// is a name namespace inside a room, so a member whose bare name collides
	// with another team's is addressed as "<Team>/<name>".
	Team string
}
