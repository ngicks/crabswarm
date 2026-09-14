package cli

import (
	"errors"
	"fmt"
	"strings"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// EveryoneTarget addresses the whole room: everyone attending it when the
// message lands, and every role that reads it afterwards.
//
// It is a word rather than a star so the shell hands it over untouched, and so
// a target is never mistaken for a pattern matched against anything.
const EveryoneTarget = "everyone"

// PostTarget is how a message addressed to nobody reads back. Nothing parses
// it: a board post is written as the empty target, and this is only how one is
// shown, so a rendered line has a target column even when it names no one.
const PostTarget = "-"

// ParseTarget maps the target a message is written with onto the one the wire
// carries, for a member send and an admin send alike — both address a room the
// same way, and a second grammar would be a second thing to learn.
//
// [EveryoneTarget] is the whole room. Anything else is a comma-separated list
// of roles, each "team/name" or a bare "name" the daemon resolves — within the
// sender's team first, then across the room. The empty string is a board post:
// it reaches the room's log and mentions nobody, so it interrupts no one.
//
// Spaces around a role are dropped, since a list written for a human to read
// has them. An empty item, a repeated role and the old star grammar are refused
// here rather than sent: the daemon answers an address nothing holds with
// NotFound, which reads as "no such member" and sends the writer looking for
// the wrong mistake.
func ParseTarget(s string) (*chatv1.Target, error) {
	if s == "" {
		return nil, nil
	}
	// Trimmed for the whole-room word alone: a target written as spaces is a
	// role left out rather than a post, and the empty string above is the only
	// spelling of a post there is.
	if strings.TrimSpace(s) == EveryoneTarget {
		return &chatv1.Target{
			Target: &chatv1.Target_Everyone{Everyone: &chatv1.Everyone{}},
		}, nil
	}
	written := strings.Split(s, ",")
	roles := make([]*chatv1.MemberTarget, 0, len(written))
	seen := make(map[string]struct{}, len(written))
	for _, item := range written {
		role, err := parseRole(strings.TrimSpace(item))
		if err != nil {
			return nil, err
		}
		addr := roleAddress(role.GetTeam(), role.GetName())
		if _, dup := seen[addr]; dup {
			return nil, fmt.Errorf("target %q names %s twice", s, addr)
		}
		seen[addr] = struct{}{}
		roles = append(roles, role)
	}
	return &chatv1.Target{
		Target: &chatv1.Target_Roles{Roles: &chatv1.Roles{Roles: roles}},
	}, nil
}

// parseRole maps one written role onto the target field pair. An empty team is
// a bare name, which the daemon resolves; it is not a missing half.
func parseRole(item string) (*chatv1.MemberTarget, error) {
	switch {
	case item == "":
		return nil, errors.New(
			`a target is "` + EveryoneTarget + `", roles as "team/name" or "name" ` +
				`separated by commas, or nothing at all for a board post`)
	case strings.Contains(item, "*"):
		return nil, fmt.Errorf(
			"%q is not a target: write %q for the whole room, or name the roles",
			item, EveryoneTarget)
	case item == EveryoneTarget:
		return nil, fmt.Errorf(
			"%q is the whole room and cannot stand among named roles", EveryoneTarget)
	}
	team, name, qualified := strings.Cut(item, "/")
	if !qualified {
		return &chatv1.MemberTarget{Name: item}, nil
	}
	if team == "" || name == "" || strings.Contains(name, "/") {
		return nil, fmt.Errorf(`role %q is not in "team/name" or "name" form`, item)
	}
	return &chatv1.MemberTarget{Team: team, Name: name}, nil
}

// TargetString spells a target the way [ParseTarget] takes it, so a rendered
// line names a target the reader can type back. It is the one authority on that
// spelling: a roster row, a completion and a transcript all read it from here
// rather than assembling one each.
func TargetString(t *chatv1.Target) string {
	switch tt := t.GetTarget().(type) {
	case *chatv1.Target_Everyone:
		return EveryoneTarget
	case *chatv1.Target_Roles:
		written := tt.Roles.GetRoles()
		if len(written) == 0 {
			// A roles case naming nobody addresses nobody, which is what a post
			// is; spelling it as an empty column would lose the target field.
			return PostTarget
		}
		addrs := make([]string, len(written))
		for i, r := range written {
			addrs[i] = roleAddress(r.GetTeam(), r.GetName())
		}
		return strings.Join(addrs, ",")
	default:
		return PostTarget
	}
}

// Address spells a member the way every chat verb addresses one — "team/name",
// or the bare name where there is no team, which is the host operator and
// nothing else: no attendee is in no team.
//
// It is exported for the callers that present a roster themselves instead of
// printing [RenderMembers]'s lines: they owe their reader the same address, and
// assembling a second one would be a second spelling to keep in step.
func Address(m *chatv1.Member) string {
	return roleAddress(m.GetTeam(), m.GetName())
}

// roleAddress is [Address] for a team and name held apart, which is how a
// target carries a role.
func roleAddress(team, name string) string {
	if team == "" {
		return name
	}
	return team + "/" + name
}
