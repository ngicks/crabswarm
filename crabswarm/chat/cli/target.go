package cli

import (
	"errors"
	"fmt"
	"strings"
)

// BroadcastTarget spells the addressee of a message that went to the whole
// room. It is the "*" the admin send verb takes, so a transcript names a target
// the reader can type back.
//
// It also spells a team-wide send: the daemon records one with no recipient for
// now, so a rendered line cannot tell "everyone here" from "everyone in that
// team" and says "*" for both.
const BroadcastTarget = "*"

// AdminTarget is who an admin send is for: exactly one of the three cases the
// request carries. It is a struct rather than an interface because the CLI
// builds one from a written word and hands it straight on — [ParseAdminTarget]
// is the only thing that decides which case a spelling means.
//
// Everyone is the whole room. Otherwise a Name names a member, with Team
// narrowing it — an empty Team leaves the name for the daemon to resolve across
// the room, exactly as a member's own bare address is resolved. A Team with no
// Name is the team itself, everyone attending it when the send is counted.
type AdminTarget struct {
	Everyone bool
	Team     string // TeamTarget when set and Name is empty
	Name     string // MemberTarget with Team (may be empty) when set
}

// ParseAdminTarget maps the grammar `chat admin send` takes onto the case it
// means: "*" is everyone in the room, "team/*" that whole team, "team/name"
// that member of that team, and a bare name the member the daemon resolves it
// to across the room.
//
// A half-written address is refused here rather than sent: an empty team or
// name reaches the daemon as an address nothing answers to, and NotFound is a
// poor way to be told that a word was left out.
func ParseAdminTarget(s string) (AdminTarget, error) {
	switch s {
	case "":
		return AdminTarget{}, errors.New(
			`no target: write "*", "team/*", "team/name" or "name"`)
	case BroadcastTarget:
		return AdminTarget{Everyone: true}, nil
	}
	team, name, qualified := strings.Cut(s, "/")
	if !qualified {
		return AdminTarget{Name: s}, nil
	}
	if team == "" || name == "" || strings.Contains(name, "/") {
		return AdminTarget{}, fmt.Errorf(
			`target %q is not in "team/name" or "team/*" form`, s)
	}
	if name == BroadcastTarget {
		return AdminTarget{Team: team}, nil
	}
	return AdminTarget{Team: team, Name: name}, nil
}

// String spells the target the way [ParseAdminTarget] takes it, so a rendered
// line and a log entry name a target the reader can type back. It is also how
// anything that offers an address — a roster row, a completion — spells one, so
// there is one authority on the spelling rather than a `+ "/*"` per caller.
func (t AdminTarget) String() string {
	switch {
	case t.Everyone:
		return BroadcastTarget
	case t.Name == "":
		return t.Team + "/" + BroadcastTarget
	case t.Team == "":
		return t.Name
	default:
		return t.Team + "/" + t.Name
	}
}
