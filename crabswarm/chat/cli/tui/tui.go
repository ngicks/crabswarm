// Package tui draws the admin's watch screen: one room's conversation as it
// happens, who is attending it and in what state, and a line to speak into it.
//
// It is the screen alone. What it needs from the daemon arrives through the
// three interfaces below, which the admin half of
// [github.com/ngicks/crabswarm/crabswarm/chat/cli] implements — the package
// dials, authenticates and speaks the schema, and this one decides what a
// terminal shows. Nothing here reaches for a socket or an identity file: both
// stay behind [Deps]. The clock is the screen's own, since keeping itself
// current is its job — it polls what it was handed on intervals it sets, and
// bounds each call so a daemon that stops answering is reported rather than
// waited on.
package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// LogReader reads a room's conversation. sinceID is the id of the last entry
// the caller already has, which asks for what was said after it; zero asks for
// the tail instead, the newest limit entries, which is where a reader with no
// cursor starts.
type LogReader interface {
	RoomLog(
		ctx context.Context,
		room string,
		sinceID int64,
		limit int32,
	) ([]*chatv1.AdminHistoryEntry, error)
}

// RosterLister reports every room the daemon knows and who attends it. The
// screen needs one room's attendance, but asks for all of them: that is the
// listing the daemon serves, and the answer also says whether the room the
// operator named exists at all.
type RosterLister interface {
	Rooms(ctx context.Context) ([]*chatv1.Room, error)
}

// AdminSender delivers a message into the room without attending it, addressed
// to one member or — as "*" — to everyone there.
type AdminSender interface {
	Send(ctx context.Context, room, target, text string) (delivered int32, err error)
}

// Deps is everything the screen needs from outside itself: the room it watches
// and the three reads and writes it makes against the daemon.
type Deps struct {
	// Room is the room being watched. The admin attends none, so this is
	// always named explicitly rather than derived from a credential.
	Room string
	// Log is where the conversation comes from.
	Log LogReader
	// Roster is where the attendance comes from.
	Roster RosterLister
	// Sender is where a typed line goes.
	Sender AdminSender
}

// Run draws the watch screen for the room deps names and blocks until the
// operator quits.
//
// It looks the room up before taking the terminal over, so an unreachable
// daemon, an identity the daemon will not accept and a room that does not
// exist are all reported as plain errors on the way in rather than as a screen
// that opens onto nothing.
//
// opts are passed to the bubbletea program after the ones Run sets, so a
// caller driving the screen without a terminal — a test, mostly — can hand it
// its own input and output.
func Run(ctx context.Context, deps Deps, opts ...tea.ProgramOption) error {
	roster, err := openRoom(ctx, deps)
	if err != nil {
		return err
	}
	program := tea.NewProgram(
		newModel(ctx, deps, roster),
		append([]tea.ProgramOption{tea.WithContext(ctx)}, opts...)...,
	)
	_, err = program.Run()
	return err
}

// openRoom reports who attends the room the screen is about to watch, and
// refuses a room the daemon does not know — naming the ones it does, since the
// admin can enumerate them anyway and a typo is the likely reason to be here.
func openRoom(ctx context.Context, deps Deps) ([]*chatv1.Member, error) {
	if deps.Room == "" {
		return nil, errors.New("no room to watch")
	}
	rooms, err := deps.Roster.Rooms(ctx)
	if err != nil {
		return nil, err
	}
	known := make([]string, 0, len(rooms))
	for _, r := range rooms {
		if r.GetName() == deps.Room {
			return r.GetMembers(), nil
		}
		known = append(known, r.GetName())
	}
	if len(known) == 0 {
		return nil, fmt.Errorf("no room %q: the daemon knows no rooms yet", deps.Room)
	}
	return nil, fmt.Errorf("no room %q: the daemon knows %s",
		deps.Room, strings.Join(known, ", "))
}
