package cli

import (
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

func member(team, name, room string) *chatv1.Member {
	return &chatv1.Member{Team: team, Name: name, Room: room}
}

// memberWith is member() plus the two things the roster renderers show beside
// an address: what attends, and what its harness last reported.
func memberWith(
	team, name, room string,
	kind chatv1.MemberKind,
	state chatv1.HarnessState,
) *chatv1.Member {
	m := member(team, name, room)
	m.Kind = kind
	m.State = state
	return m
}

// runningOn is what an attendee declared about itself: the CLI it runs, and the
// route a mention takes to it.
func runningOn(
	m *chatv1.Member,
	harness chatv1.Harness,
	nudge chatv1.NudgeDelivery,
) *chatv1.Member {
	m.Harness = harness
	m.Nudge = nudge
	return m
}

// targeting builds the target a message carries, written the way a sender
// writes one.
func targeting(t *testing.T, written string) *chatv1.Target {
	t.Helper()
	target, err := ParseTarget(written)
	assert.NilError(t, err)
	return target
}

func render(t *testing.T, f func(w *strings.Builder) error) string {
	t.Helper()
	var b strings.Builder
	assert.NilError(t, f(&b))
	return b.String()
}

// A transcript line names the sequence number the reader would page from, who
// spoke, and who it was for, so a line addressed to somebody else is not read
// as one to answer.
func TestRenderRead(t *testing.T) {
	sent := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	resp := &chatv1.ReadResponse{Messages: []*chatv1.Message{
		{
			Seq:          11,
			From:         member("backend", "alice", "/work"),
			Target:       targeting(t, "frontend/bob,ops/carol"),
			Text:         "rebased onto main",
			SentAt:       timestamppb.New(sent),
			MentionedYou: true,
		},
		{
			Seq:    12,
			From:   member("frontend", "bob", "/work"),
			Target: targeting(t, "everyone"),
			Text:   "standup in 5",
			SentAt: timestamppb.New(sent.Add(time.Minute)),
		},
		{
			Seq:    13,
			From:   member("", "admin", "/work"),
			Text:   "deploy is frozen",
			SentAt: timestamppb.New(sent.Add(2 * time.Minute)),
		},
	}}

	got := render(t, func(b *strings.Builder) error { return RenderRead(b, resp) })
	assert.Equal(t, got,
		"11 2026-08-27T09:30:00Z backend/alice -> frontend/bob,ops/carol "+
			"[mentioned you]: rebased onto main\n"+
			"12 2026-08-27T09:31:00Z frontend/bob -> everyone: standup in 5\n"+
			"13 2026-08-27T09:32:00Z admin -> -: deploy is frozen\n")
}

// What the read left behind is the only thing telling the reader that another
// read would hand over more; a read that left none says nothing about it.
func TestRenderRead_RemainingUnread(t *testing.T) {
	messages := []*chatv1.Message{{
		Seq:    4,
		From:   member("backend", "alice", "/work"),
		Target: targeting(t, "everyone"),
		Text:   "ping",
		SentAt: timestamppb.New(time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)),
	}}
	const line = "4 2026-08-27T09:30:00Z backend/alice -> everyone: ping\n"

	got := render(t, func(b *strings.Builder) error {
		return RenderRead(b, &chatv1.ReadResponse{Messages: messages, RemainingUnread: 12})
	})
	assert.Equal(t, got, line+"12 more unread\n")

	got = render(t, func(b *strings.Builder) error {
		return RenderRead(b, &chatv1.ReadResponse{Messages: messages})
	})
	assert.Equal(t, got, line)
}

// A stamp is rendered in UTC whatever the reader's zone is, so two agents in
// different containers order the same conversation the same way.
func TestRenderRead_StampIsUTC(t *testing.T) {
	zone := time.FixedZone("UTC+9", 9*60*60)
	resp := &chatv1.ReadResponse{Messages: []*chatv1.Message{{
		Seq:    1,
		From:   member("backend", "alice", "/work"),
		Target: targeting(t, "everyone"),
		Text:   "hi",
		SentAt: timestamppb.New(time.Date(2026, 8, 27, 18, 30, 0, 0, zone)),
	}}}

	got := render(t, func(b *strings.Builder) error { return RenderRead(b, resp) })
	assert.Equal(t, got, "1 2026-08-27T09:30:00Z backend/alice -> everyone: hi\n")
}

// A read that found nothing reports itself: a caller polling for messages must
// be able to tell a successful empty read from a command that produced no
// output at all.
func TestRenderRead_Empty(t *testing.T) {
	got := render(t, func(b *strings.Builder) error {
		return RenderRead(b, &chatv1.ReadResponse{})
	})
	assert.Equal(t, got, "no pending messages\n")
}

// A narrowed read answers about what it was asked for, so it can come back
// empty while mentions of the caller wait outside it. The trailer is what keeps
// that from reading as an empty room.
func TestRenderRead_EmptyWithRemainingUnread(t *testing.T) {
	got := render(t, func(b *strings.Builder) error {
		return RenderRead(b, &chatv1.ReadResponse{RemainingUnread: 2})
	})
	assert.Equal(t, got, "no pending messages\n2 more unread\n")
}

// A message with no stamp still fills the field, in one word: a reader cutting
// the line on spaces would otherwise find the sender where the time should be.
func TestRenderRead_MissingTimestamp(t *testing.T) {
	resp := &chatv1.ReadResponse{Messages: []*chatv1.Message{{
		Seq:    2,
		From:   member("backend", "alice", "/work"),
		Target: targeting(t, "everyone"),
		Text:   "hi",
	}}}

	got := render(t, func(b *strings.Builder) error { return RenderRead(b, resp) })
	assert.Equal(t, got, "2 unknown-time backend/alice -> everyone: hi\n")
	assert.Equal(t, len(strings.Fields(strings.Split(got, "\n")[0])), 6)
}

// The admin reads the transcript the members read: the same lines, down to the
// wording of a silent room.
func TestRenderHistory(t *testing.T) {
	sent := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	messages := []*chatv1.Message{{
		Seq:    41,
		From:   member("", "admin", "/work"),
		Target: targeting(t, "frontend/bob"),
		Text:   "deploy is frozen",
		SentAt: timestamppb.New(sent),
	}}

	got := render(t, func(b *strings.Builder) error { return RenderHistory(b, messages) })
	assert.Equal(t, got,
		"41 2026-08-27T09:30:00Z admin -> frontend/bob: deploy is frozen\n")

	got = render(t, func(b *strings.Builder) error { return RenderHistory(b, nil) })
	assert.Equal(t, got, "no messages yet\n")
}

// A send says who it mentioned, and warns about the ones nobody is attending
// under — on stderr, since the message was accepted either way and the warning
// is for the person rather than for the pipe.
func TestRenderSent(t *testing.T) {
	resp := &chatv1.SendResponse{
		Mentioned: []*chatv1.Member{
			member("devenv", "claude-1", "/work"),
			member("devenv", "claude-3", "/work"),
		},
		Absent: []*chatv1.Member{member("devenv", "claude-3", "/work")},
	}

	var out, warn strings.Builder
	assert.NilError(t, RenderSent(&out, &warn, resp))
	assert.Equal(t, out.String(),
		"mentioned devenv/claude-1\nmentioned devenv/claude-3\n")
	assert.Equal(t, warn.String(),
		"warning: devenv/claude-3 is not attending; the mention waits\n")
}

// Everyone and a board post name no role in particular, so there is nothing to
// report: the message is in the room, and a line saying so would be noise in
// front of the agent reading it.
func TestRenderSent_NamesNobody(t *testing.T) {
	var out, warn strings.Builder
	assert.NilError(t, RenderSent(&out, &warn, &chatv1.SendResponse{}))
	assert.Equal(t, out.String(), "")
	assert.Equal(t, warn.String(), "")
}

// The admin send answers in the same two lists and reads the same way: an
// operator and a member comparing notes are reading one text.
func TestRenderSent_Admin(t *testing.T) {
	resp := &chatv1.AdminSendResponse{
		Mentioned: []*chatv1.Member{member("backend", "alice", "/work")},
		Absent:    []*chatv1.Member{member("backend", "alice", "/work")},
	}

	var out, warn strings.Builder
	assert.NilError(t, RenderSent(&out, &warn, resp))
	assert.Equal(t, out.String(), "mentioned backend/alice\n")
	assert.Equal(t, warn.String(),
		"warning: backend/alice is not attending; the mention waits\n")
}

func TestRenderMembers(t *testing.T) {
	members := []*chatv1.Member{
		runningOn(memberWith("backend", "alice", "/work",
			chatv1.MemberKind_MEMBER_KIND_AGENT,
			chatv1.HarnessState_HARNESS_STATE_WORKING),
			chatv1.Harness_HARNESS_CLAUDE_CODE,
			chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE),
		// An agent whose server could not name its harness still says how it
		// wants to be reached: the two are declared separately for this.
		runningOn(memberWith("backend", "dave", "/work",
			chatv1.MemberKind_MEMBER_KIND_AGENT,
			chatv1.HarnessState_HARNESS_STATE_DONE),
			chatv1.Harness_HARNESS_UNSPECIFIED,
			chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL),
		// A person runs no harness and is never nudged, so both columns say
		// nothing was declared rather than naming something they are not.
		memberWith("frontend", "bob", "/work",
			chatv1.MemberKind_MEMBER_KIND_HUMAN,
			chatv1.HarnessState_HARNESS_STATE_DONE),
		// A member the daemon reported none of it for still gets its columns, so
		// the line a caller splits has the same shape for everyone.
		member("ops", "carol", "/work"),
	}

	got := render(t, func(b *strings.Builder) error { return RenderMembers(b, members) })
	assert.Equal(t, got,
		"backend/alice  agent  working  claude-code  native\n"+
			"backend/dave  agent  done  -  terminal\n"+
			"frontend/bob  human  done  -  -\n"+
			"ops/carol  unknown  unknown  -  -\n")

	// The first column is the role a target names, undecorated: whoever reads a
	// line cuts it on whitespace and sends to what comes first.
	assert.Equal(t, strings.Fields(strings.Split(got, "\n")[0])[0], "backend/alice")

	got = render(t, func(b *strings.Builder) error { return RenderMembers(b, nil) })
	assert.Equal(t, got, "no members\n")
}

func TestRenderRooms(t *testing.T) {
	agent := chatv1.MemberKind_MEMBER_KIND_AGENT
	human := chatv1.MemberKind_MEMBER_KIND_HUMAN
	unreported := chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED
	rooms := []*chatv1.Room{
		{
			Name: "/work/proj",
			Members: []*chatv1.Member{
				runningOn(memberWith("backend", "alice", "/work/proj", agent, unreported),
					chatv1.Harness_HARNESS_CODEX,
					chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL),
				memberWith("frontend", "bob", "/work/proj", agent, unreported),
				// A second member of an earlier team joins that team's block
				// rather than opening a new one.
				memberWith("backend", "carol", "/work/proj", human, unreported),
			},
		},
		// A room outlives the sessions that spoke in it, and an operator looking
		// for one to read or to delete is looking for exactly this one.
		{Name: "/work/done"},
	}

	got := render(t, func(b *strings.Builder) error { return RenderRooms(b, rooms) })
	assert.Equal(t, got,
		"room: /work/proj\n"+
			"  team: backend\n"+
			"    alice  agent  codex  terminal\n"+
			"    carol  human  -  -\n"+
			"  team: frontend\n"+
			"    bob  agent  -  -\n"+
			"room: /work/done\n"+
			"  (nobody attending)\n")

	got = render(t, func(b *strings.Builder) error { return RenderRooms(b, nil) })
	assert.Equal(t, got, "no rooms\n")
}

func TestRenderRegistered(t *testing.T) {
	registered := member("humans", "yuki", "/work/proj")
	got := render(t, func(b *strings.Builder) error {
		return RenderRegistered(b, registered, "tok-secret")
	})
	assert.Equal(t, got,
		"registered humans/yuki in room /work/proj\ntoken: tok-secret\n")
}

// The count is the only thing left of a room that is gone, which is what tells
// an operator who deleted the wrong one how much was in it.
func TestRenderDeletedRoom(t *testing.T) {
	got := render(t, func(b *strings.Builder) error {
		return RenderDeletedRoom(b, "/work/proj", 42)
	})
	assert.Equal(t, got, "deleted room /work/proj and 42 messages\n")

	got = render(t, func(b *strings.Builder) error {
		return RenderDeletedRoom(b, "/work/proj", 1)
	})
	assert.Equal(t, got, "deleted room /work/proj and 1 message\n")
}
