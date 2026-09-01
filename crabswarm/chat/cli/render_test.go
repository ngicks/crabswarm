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

func render(t *testing.T, f func(w *strings.Builder) error) string {
	t.Helper()
	var b strings.Builder
	assert.NilError(t, f(&b))
	return b.String()
}

func TestRenderJoinedSentBroadcastLeft(t *testing.T) {
	self := member("backend", "alice", "/work/proj")

	got := render(t, func(b *strings.Builder) error { return RenderJoined(b, self) })
	assert.Equal(t, got, "joined /work/proj as backend/alice\n")

	got = render(t, func(b *strings.Builder) error { return RenderSent(b, self) })
	assert.Equal(t, got, "sent to backend/alice\n")

	got = render(t, func(b *strings.Builder) error { return RenderLeft(b) })
	assert.Equal(t, got, "left the room\n")

	got = render(t, func(b *strings.Builder) error { return RenderBroadcast(b, 3) })
	assert.Equal(t, got, "broadcast to 3 members\n")

	got = render(t, func(b *strings.Builder) error { return RenderBroadcast(b, 1) })
	assert.Equal(t, got, "broadcast to 1 member\n")

	// Nobody else attending is a successful broadcast, and has to read like one.
	got = render(t, func(b *strings.Builder) error { return RenderBroadcast(b, 0) })
	assert.Equal(t, got, "broadcast to 0 members\n")
}

func TestRenderMessages(t *testing.T) {
	sent := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	messages := []*chatv1.Message{
		{
			From:   member("backend", "alice", "/work"),
			Text:   "rebased onto main",
			SentAt: timestamppb.New(sent),
		},
		{
			From:   member("frontend", "bob", "/work"),
			Text:   "ack",
			SentAt: timestamppb.New(sent.Add(time.Minute)),
		},
	}

	got := render(t, func(b *strings.Builder) error { return RenderMessages(b, messages) })
	assert.Equal(t, got,
		"[2026-08-27T09:30:00Z] backend/alice: rebased onto main\n"+
			"[2026-08-27T09:31:00Z] frontend/bob: ack\n")
}

// A stamp is rendered in UTC whatever the reader's zone is, so two agents in
// different containers order the same conversation the same way.
func TestRenderMessages_StampIsUTC(t *testing.T) {
	zone := time.FixedZone("UTC+9", 9*60*60)
	messages := []*chatv1.Message{{
		From:   member("backend", "alice", "/work"),
		Text:   "hi",
		SentAt: timestamppb.New(time.Date(2026, 8, 27, 18, 30, 0, 0, zone)),
	}}

	got := render(t, func(b *strings.Builder) error { return RenderMessages(b, messages) })
	assert.Equal(t, got, "[2026-08-27T09:30:00Z] backend/alice: hi\n")
}

// An empty inbox reports itself: a caller polling for mail must be able to tell
// a successful empty read from a command that produced no output at all.
func TestRenderMessages_Empty(t *testing.T) {
	got := render(t, func(b *strings.Builder) error { return RenderMessages(b, nil) })
	assert.Equal(t, got, "no pending messages\n")
}

func TestRenderMessages_MissingTimestamp(t *testing.T) {
	messages := []*chatv1.Message{{From: member("backend", "alice", "/work"), Text: "hi"}}

	got := render(t, func(b *strings.Builder) error { return RenderMessages(b, messages) })
	assert.Equal(t, got, "[unknown time] backend/alice: hi\n")
}

// A transcript line names who was addressed, so a directed message is not read
// as something said to the whole room. A broadcast is addressed to "*", the
// same target the admin send verb takes for "everyone here".
func TestRenderHistory(t *testing.T) {
	sent := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	entries := []*chatv1.HistoryEntry{
		{
			From:   member("backend", "alice", "/work"),
			To:     member("frontend", "bob", "/work"),
			Text:   "rebased onto main",
			SentAt: timestamppb.New(sent),
		},
		{
			From:   member("frontend", "bob", "/work"),
			Text:   "standup in 5",
			SentAt: timestamppb.New(sent.Add(time.Minute)),
		},
		{From: member("backend", "alice", "/work"), Text: "when?"},
	}

	got := render(t, func(b *strings.Builder) error { return RenderHistory(b, entries) })
	assert.Equal(t, got,
		"[2026-08-27T09:30:00Z] backend/alice → frontend/bob: rebased onto main\n"+
			"[2026-08-27T09:31:00Z] frontend/bob → *: standup in 5\n"+
			"[unknown time] backend/alice → *: when?\n")
}

// A room nobody has spoken in says so, the way an empty inbox does.
func TestRenderHistory_Empty(t *testing.T) {
	got := render(t, func(b *strings.Builder) error { return RenderHistory(b, nil) })
	assert.Equal(t, got, "no messages yet\n")
}

func TestRenderMembers(t *testing.T) {
	members := []*chatv1.Member{
		member("backend", "alice", "/work"),
		member("frontend", "bob", "/work"),
	}

	got := render(t, func(b *strings.Builder) error { return RenderMembers(b, members) })
	assert.Equal(t, got, "backend/alice\nfrontend/bob\n")

	got = render(t, func(b *strings.Builder) error { return RenderMembers(b, nil) })
	assert.Equal(t, got, "no members\n")
}

func TestRenderRooms(t *testing.T) {
	rooms := []*chatv1.Room{
		{
			Name: "/work/proj",
			Members: []*chatv1.Member{
				member("backend", "alice", "/work/proj"),
				member("frontend", "bob", "/work/proj"),
				// A second member of an earlier team joins that team's block
				// rather than opening a new one.
				member("backend", "carol", "/work/proj"),
			},
		},
		{Name: "/work/other", Members: []*chatv1.Member{member("humans", "yuki", "/work/other")}},
	}

	got := render(t, func(b *strings.Builder) error { return RenderRooms(b, rooms) })
	assert.Equal(t, got,
		"room: /work/proj\n"+
			"  team: backend\n"+
			"    alice\n"+
			"    carol\n"+
			"  team: frontend\n"+
			"    bob\n"+
			"room: /work/other\n"+
			"  team: humans\n"+
			"    yuki\n")

	got = render(t, func(b *strings.Builder) error { return RenderRooms(b, nil) })
	assert.Equal(t, got, "no rooms\n")
}

func TestRenderMovedAndRegistered(t *testing.T) {
	moved := member("frontend", "alice", "/work/proj")
	got := render(t, func(b *strings.Builder) error { return RenderMoved(b, moved) })
	assert.Equal(t, got, "moved frontend/alice in room /work/proj\n")

	registered := member("humans", "yuki", "/work/proj")
	got = render(t, func(b *strings.Builder) error {
		return RenderRegistered(b, registered, "tok-secret")
	})
	assert.Equal(t, got,
		"registered humans/yuki in room /work/proj\ntoken: tok-secret\n")
}

func TestRenderAdminSent(t *testing.T) {
	got := render(t, func(b *strings.Builder) error {
		return RenderAdminSent(b, "/work/proj", "backend/alice", 1)
	})
	assert.Equal(t, got, "sent to backend/alice in room /work/proj: delivered to 1 member\n")

	got = render(t, func(b *strings.Builder) error {
		return RenderAdminSent(b, "/work/proj", "*", 3)
	})
	assert.Equal(t, got, "sent to * in room /work/proj: delivered to 3 members\n")
}
