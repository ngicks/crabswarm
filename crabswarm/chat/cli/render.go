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
// act on it — the target it just read goes straight back in as the argument of
// the next `chat send` — so a stable, greppable line beats a pretty table.
// Each builds its whole output first and writes once, so a renderer either
// reports the write error or has written everything.

// messageTimeFormat stamps a message with an unambiguous instant. UTC, not
// local time: a room spans containers that need not agree on a time zone, and
// the reader is usually comparing two agents' messages rather than the wall
// clock.
const messageTimeFormat = time.RFC3339

// unknownTime stands in for a message the daemon sent no stamp with. It is one
// word because a message line is read by cutting it on spaces, and it is not
// the epoch the timestamp would otherwise decode to, which would read as a real
// instant in 1970.
const unknownTime = "unknown-time"

// RenderRead prints what a read handed over and how much of the caller's unread
// is left, which is what tells a reader whether to read again.
//
// A read that found nothing says so on stdout rather than printing nothing, so
// a caller polling for messages can tell a successful empty read from a command
// that never ran. [ReadOptions.Quiet] is what removes that line.
func RenderRead(w io.Writer, resp *chatv1.ReadResponse) error {
	messages := resp.GetMessages()
	if len(messages) == 0 {
		_, err := fmt.Fprintln(w, "no pending messages")
		return err
	}
	var b strings.Builder
	writeMessages(&b, messages)
	if remaining := resp.GetRemainingUnread(); remaining > 0 {
		fmt.Fprintf(&b, "%d more unread\n", remaining)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// RenderHistory prints a room's conversation, which is the same transcript a
// read prints: an operator comparing what a room shows its members with what it
// shows the host should be reading one text, not two.
//
// A room nobody has spoken in says so, the way an empty read does: a listing
// that printed nothing at all would be indistinguishable from a command that
// never ran.
func RenderHistory(w io.Writer, messages []*chatv1.Message) error {
	if len(messages) == 0 {
		_, err := fmt.Fprintln(w, "no messages yet")
		return err
	}
	var b strings.Builder
	writeMessages(&b, messages)
	_, err := io.WriteString(w, b.String())
	return err
}

// writeMessages writes the transcript itself, oldest first, one line each:
//
//	<seq> <time> <from> -> <target> [mentioned you]: <text>
//
// The sequence number leads because it is what --since and --until take, so a
// reader who wants what came after a line can read the argument off it. The
// target is there because a room is one conversation: a line addressed to
// somebody else is not a line to answer, and only the target says which is
// which.
//
// A message whose text spans lines keeps them; only the first line carries the
// prefix, since chopping a message up would misrepresent what was sent.
func writeMessages(b *strings.Builder, messages []*chatv1.Message) {
	for _, m := range messages {
		sentAt := unknownTime
		if ts := m.GetSentAt(); ts != nil {
			sentAt = ts.AsTime().UTC().Format(messageTimeFormat)
		}
		fmt.Fprintf(b, "%d %s %s -> %s",
			m.GetSeq(), sentAt, Address(m.GetFrom()), TargetString(m.GetTarget()))
		if m.GetMentionedYou() {
			b.WriteString(" [mentioned you]")
		}
		fmt.Fprintf(b, ": %s\n", m.GetText())
	}
}

// sendOutcome is what reporting a send needs, and what the member send and the
// admin send answer alike: who the target resolved to, and which of them nobody
// is attending under.
type sendOutcome interface {
	GetMentioned() []*chatv1.Member
	GetAbsent() []*chatv1.Member
}

// RenderSent reports a send: one line per role it mentioned, and a warning per
// role nobody is attending under. A send to everyone and a board post name no
// role in particular and so print nothing, which is the quiet the sender wants
// — the message is in the room either way.
//
// The warnings go to warn rather than to out because they are not the answer to
// the command: the message was accepted and waits at that role's read position
// until somebody attends under it. A caller piping the delivery lines wants the
// warning in front of the person, not in the pipe.
func RenderSent[R sendOutcome](out, warn io.Writer, resp R) error {
	var mentioned strings.Builder
	for _, m := range resp.GetMentioned() {
		fmt.Fprintf(&mentioned, "mentioned %s\n", Address(m))
	}
	if _, err := io.WriteString(out, mentioned.String()); err != nil {
		return err
	}
	var absent strings.Builder
	for _, m := range resp.GetAbsent() {
		fmt.Fprintf(&absent, "warning: %s is not attending; the mention waits\n", Address(m))
	}
	_, err := io.WriteString(warn, absent.String())
	return err
}

// RenderMembers lists the room's attendance, one member per line: the
// team-qualified address, the kind and the harness state, separated by two
// spaces. The first column is the point: it is exactly the role a target names,
// so the reader never has to assemble one, and it stays first and unpadded so a
// line still cuts cleanly on whitespace.
//
// The kind is there because it says whether a message reaches the member on its
// own: an agent is typed into when one arrives, a human is only ever handed its
// messages when it asks. Whoever is waiting for an answer reads that off the
// roster rather than guessing from the name.
func RenderMembers(w io.Writer, members []*chatv1.Member) error {
	if len(members) == 0 {
		_, err := fmt.Fprintln(w, "no members")
		return err
	}
	var b strings.Builder
	for _, m := range members {
		fmt.Fprintf(&b, "%s  %s  %s\n",
			Address(m), MemberKindName(m.GetKind()), HarnessStateName(m.GetState()))
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// RenderRooms prints the whole topology as an indented room → team → member
// tree, each member followed by its kind. The admin listing is the one place
// where a member's three coordinates are all in play, and nesting shows the
// grouping that a flat "room/team/name" column would make the reader
// reconstruct; the kind is what tells an operator which of those members a
// message reaches on its own.
//
// A room nobody is in is listed with a note in place of its teams rather than
// left out: its conversation and its read positions outlive the sessions that
// made them, and an operator looking for a room to read or to delete is looking
// for exactly those.
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
		if len(r.GetMembers()) == 0 {
			b.WriteString("  (nobody attending)\n")
			continue
		}
		for _, t := range groupByTeam(r.GetMembers()) {
			fmt.Fprintf(&b, "  team: %s\n", t.team)
			for _, m := range t.members {
				fmt.Fprintf(&b, "    %s  %s\n", m.GetName(), MemberKindName(m.GetKind()))
			}
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// teamMembers is one team of a room and the members attending under it.
type teamMembers struct {
	team    string
	members []*chatv1.Member
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
		grouped[i].members = append(grouped[i].members, m)
	}
	return grouped
}

// RenderRegistered prints the new member and, on a line of its own, the token
// it presents from then on. The token is shown once and stored nowhere, so the
// line is kept bare enough to copy or cut out of a pipe.
func RenderRegistered(w io.Writer, member *chatv1.Member, token string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "registered %s in room %s\n", Address(member), member.GetRoom())
	fmt.Fprintf(&b, "token: %s\n", token)
	_, err := io.WriteString(w, b.String())
	return err
}

// RenderDeletedRoom reports what the deletion took with it. The count is worth
// saying: it is the only thing left of a room that is gone, and it tells an
// operator who deleted the wrong one how much was in it.
func RenderDeletedRoom(w io.Writer, room string, messages int64) error {
	_, err := fmt.Fprintf(w, "deleted room %s and %d %s\n",
		room, messages, messageNoun(messages))
	return err
}

// messageNoun agrees the noun with a count of messages.
func messageNoun(n int64) string {
	if n == 1 {
		return "message"
	}
	return "messages"
}
