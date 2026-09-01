package cli

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// Every renderer here writes plain lines with no alignment padding, no color
// and no terminal control. Most of this output is read by an agent that has to
// act on it — the address it just learned goes straight back in as the argument
// of the next `chat send` — so a stable, greppable line beats a pretty table.
// Each builds its whole output first and writes once, so a renderer either
// reports the write error or has written everything.

// messageTimeFormat stamps a message with an unambiguous instant. UTC, not
// local time: a room spans containers that need not agree on a time zone, and
// the reader is usually comparing two agents' messages rather than the wall
// clock.
const messageTimeFormat = time.RFC3339

// qualify writes a member the way every chat command addresses one: "team/name",
// falling back to the bare name when the daemon reported no team.
func qualify(m *chatv1.Member) string {
	if m.GetTeam() == "" {
		return m.GetName()
	}
	return m.GetTeam() + "/" + m.GetName()
}

// RenderJoined reports the identity the daemon settled on. The room and team
// are the caller's news: it chose neither, they follow from its token.
func RenderJoined(w io.Writer, self *chatv1.Member) error {
	_, err := fmt.Fprintf(w, "joined %s as %s\n", self.GetRoom(), qualify(self))
	return err
}

// RenderSent reports which member an address resolved to, so a bare name that
// resolved room-wide shows whose inbox it actually landed in.
func RenderSent(w io.Writer, recipient *chatv1.Member) error {
	_, err := fmt.Fprintf(w, "sent to %s\n", qualify(recipient))
	return err
}

// RenderBroadcast reports how many inboxes the message reached. Zero is worth
// saying out loud: it means nobody else is attending, not that the send failed.
func RenderBroadcast(w io.Writer, delivered int32) error {
	_, err := fmt.Fprintf(w, "broadcast to %d %s\n", delivered, memberNoun(delivered))
	return err
}

// memberNoun agrees the noun with a count of recipients.
func memberNoun(delivered int32) string {
	if delivered == 1 {
		return "member"
	}
	return "members"
}

// RenderLeft confirms the withdrawal, which the daemon acknowledges with an
// empty response.
func RenderLeft(w io.Writer) error {
	_, err := fmt.Fprintln(w, "left the room")
	return err
}

// RenderMessages prints the messages a read consumed, oldest first, one line
// each: the instant, the team-qualified sender, then the text. An empty inbox
// says so on stdout rather than printing nothing, so a caller polling for mail
// can tell a successful empty read from a command that never ran.
//
// A message whose text spans lines keeps them; only the first line carries the
// prefix, since chopping a message up would misrepresent what was sent.
func RenderMessages(w io.Writer, messages []*chatv1.Message) error {
	if len(messages) == 0 {
		_, err := fmt.Fprintln(w, "no pending messages")
		return err
	}
	var b strings.Builder
	for _, m := range messages {
		sentAt := "unknown time"
		if ts := m.GetSentAt(); ts != nil {
			sentAt = ts.AsTime().UTC().Format(messageTimeFormat)
		}
		fmt.Fprintf(&b, "[%s] %s: %s\n", sentAt, qualify(m.GetFrom()), m.GetText())
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// broadcastTarget spells the addressee of an entry that went to the whole room.
// It is the "*" the admin send verb already takes for "everyone here", so the
// transcript names a target the reader can type back.
const broadcastTarget = "*"

// RenderHistory prints a room's conversation, oldest first, one line each: the
// instant, the team-qualified speaker, who it was said to, then the text. A
// broadcast is addressed to [broadcastTarget] rather than to a member.
//
// A room nobody has spoken in says so, the way an empty inbox does: the
// transcript is a read, and a read that printed nothing at all would be
// indistinguishable from a command that never ran.
func RenderHistory(w io.Writer, entries []*chatv1.HistoryEntry) error {
	if len(entries) == 0 {
		_, err := fmt.Fprintln(w, "no messages yet")
		return err
	}
	var b strings.Builder
	for _, e := range entries {
		sentAt := "unknown time"
		if ts := e.GetSentAt(); ts != nil {
			sentAt = ts.AsTime().UTC().Format(messageTimeFormat)
		}
		to := broadcastTarget
		if e.GetTo() != nil {
			to = qualify(e.GetTo())
		}
		fmt.Fprintf(&b, "[%s] %s → %s: %s\n", sentAt, qualify(e.GetFrom()), to, e.GetText())
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// RenderMembers lists the room's attendance, one team-qualified member per
// line. That spelling is the point: each line is exactly the address argument
// `chat send` takes, so the reader never has to assemble one.
func RenderMembers(w io.Writer, members []*chatv1.Member) error {
	if len(members) == 0 {
		_, err := fmt.Fprintln(w, "no members")
		return err
	}
	var b strings.Builder
	for _, m := range members {
		b.WriteString(qualify(m))
		b.WriteByte('\n')
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// RenderRooms prints the whole topology as an indented room → team → member
// tree. The admin listing is the one place where a member's three coordinates
// are all in play, and nesting shows the grouping that a flat "room/team/name"
// column would make the reader reconstruct.
//
// Teams appear in the order the daemon first mentions them and members in the
// order they arrive, so the tree mirrors the listing rather than imposing an
// order of its own.
func RenderRooms(w io.Writer, rooms []*chatv1.Room) error {
	if len(rooms) == 0 {
		_, err := fmt.Fprintln(w, "no rooms")
		return err
	}
	var b strings.Builder
	for _, r := range rooms {
		fmt.Fprintf(&b, "room: %s\n", r.GetName())
		for _, t := range groupByTeam(r.GetMembers()) {
			fmt.Fprintf(&b, "  team: %s\n", t.team)
			for _, name := range t.names {
				fmt.Fprintf(&b, "    %s\n", name)
			}
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// teamMembers is one team of a room and the names attending under it.
type teamMembers struct {
	team  string
	names []string
}

// groupByTeam buckets a room's members by team, keeping first-mention order.
func groupByTeam(members []*chatv1.Member) []teamMembers {
	var grouped []teamMembers
	for _, m := range members {
		i := slices.IndexFunc(grouped, func(t teamMembers) bool { return t.team == m.GetTeam() })
		if i < 0 {
			grouped = append(grouped, teamMembers{team: m.GetTeam()})
			i = len(grouped) - 1
		}
		grouped[i].names = append(grouped[i].names, m.GetName())
	}
	return grouped
}

// RenderMoved reports the member's new placement, read back from the daemon
// rather than echoed from the request.
func RenderMoved(w io.Writer, member *chatv1.Member) error {
	_, err := fmt.Fprintf(w, "moved %s in room %s\n", qualify(member), member.GetRoom())
	return err
}

// RenderAdminSent reports an admin delivery, echoing the room and the target it
// was addressed to. Both come back from the request rather than from the daemon,
// which answers with a count alone — and an address that resolves to nobody
// fails the call instead of reporting zero, so a rendered count is at least 1.
func RenderAdminSent(w io.Writer, room, target string, delivered int32) error {
	_, err := fmt.Fprintf(w, "sent to %s in room %s: delivered to %d %s\n",
		target, room, delivered, memberNoun(delivered))
	return err
}

// RenderRegistered prints the new member and, on a line of its own, the token
// it presents from then on. The token is shown once and stored nowhere, so the
// line is kept bare enough to copy or cut out of a pipe.
func RenderRegistered(w io.Writer, member *chatv1.Member, token string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "registered %s in room %s\n", qualify(member), member.GetRoom())
	fmt.Fprintf(&b, "token: %s\n", token)
	_, err := io.WriteString(w, b.String())
	return err
}
