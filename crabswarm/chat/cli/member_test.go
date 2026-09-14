package cli

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// Attendance lands before the feed is read: the caller learns its own room,
// team and name from the first event, which is what it matches later events
// against.
func TestClient_Attend(t *testing.T) {
	fake := &fakeChatService{self: member("backend", "alice", "/work/proj")}
	d := serveTestDaemon(t, fake, nil)

	att, err := d.client.Attend(t.Context(), "tok-a", "alice",
		chatv1.MemberKind_MEMBER_KIND_AGENT)
	assert.NilError(t, err)
	assert.Equal(t, fake.attendRequest().GetName(), "alice")
	assert.Equal(t, fake.attendRequest().GetKind(), chatv1.MemberKind_MEMBER_KIND_AGENT)
	assert.DeepEqual(t, d.seenTokens(), []string{"tok-a"})

	assert.Equal(t, att.Self().GetTeam(), "backend")
	assert.Equal(t, att.Self().GetName(), "alice")
	assert.Equal(t, att.Self().GetRoom(), "/work/proj")
}

// An unnamed attendance sends an empty name: naming the member is the daemon's
// job when the caller declines to.
func TestClient_AttendUnnamed(t *testing.T) {
	fake := &fakeChatService{self: member("backend", "alice", "/work/proj")}
	d := serveTestDaemon(t, fake, nil)

	_, err := d.client.Attend(t.Context(), "tok-a", "",
		chatv1.MemberKind_MEMBER_KIND_HUMAN)
	assert.NilError(t, err)
	assert.Equal(t, fake.attendRequest().GetName(), "")
	assert.Equal(t, fake.attendRequest().GetKind(), chatv1.MemberKind_MEMBER_KIND_HUMAN)
}

// The stream is lazy, so a refusal arrives at the first event rather than at the
// call that opened it. It still has to reach the caller as the daemon's own
// words, since that is where "somebody is already attending as this role" is
// said.
func TestClient_AttendSurfacesTheRefusal(t *testing.T) {
	const msg = `"backend/alice" is already attending`
	fake := &fakeChatService{err: status.Error(codes.AlreadyExists, msg)}
	d := serveTestDaemon(t, fake, nil)

	_, err := d.client.Attend(t.Context(), "tok-a", "alice",
		chatv1.MemberKind_MEMBER_KIND_AGENT)
	assert.Assert(t, err != nil)
	assert.Equal(t, err.Error(), msg)
}

// A caller that gave up while attending was in flight hears its own
// cancellation: a bridge shutting down would otherwise read its exit as a
// refusal to attend, and try again on the way out.
func TestClient_AttendReportsTheCancellation(t *testing.T) {
	fake := &fakeChatService{self: member("backend", "alice", "/work")}
	d := serveTestDaemon(t, fake, nil)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := d.client.Attend(ctx, "tok-a", "alice",
		chatv1.MemberKind_MEMBER_KIND_AGENT)
	assert.ErrorIs(t, err, context.Canceled)
}

// The first event is what says the attendance landed; anything else means the
// caller holds no membership it could act on, and saying so beats handing back
// a member nobody named.
func TestClient_AttendRejectsAnOpeningThatNamesNoMember(t *testing.T) {
	fake := &fakeChatService{openWith: &chatv1.RoomEvent{
		Event: &chatv1.RoomEvent_MemberJoined{
			MemberJoined: &chatv1.MemberJoined{
				Member: member("backend", "bob", "/work/proj"),
			},
		},
	}}
	d := serveTestDaemon(t, fake, nil)

	_, err := d.client.Attend(t.Context(), "tok-a", "alice",
		chatv1.MemberKind_MEMBER_KIND_AGENT)
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "MemberJoined"))
}

// Forward hands the room's feed over event by event and stops when the caller
// says so, which is how a bridge ends a feed it is still being served.
func TestAttendance_ForwardStopsOnTheCallersError(t *testing.T) {
	joined := &chatv1.RoomEvent{
		Event: &chatv1.RoomEvent_MemberJoined{
			MemberJoined: &chatv1.MemberJoined{Member: member("frontend", "bob", "/work")},
		},
	}
	appended := &chatv1.RoomEvent{
		Event: &chatv1.RoomEvent_MessageAppended{
			MessageAppended: &chatv1.MessageAppended{Message: &chatv1.Message{Seq: 4}},
		},
	}
	fake := &fakeChatService{
		self:   member("backend", "alice", "/work"),
		events: []*chatv1.RoomEvent{joined, appended},
	}
	d := serveTestDaemon(t, fake, nil)

	att, err := d.client.Attend(t.Context(), "tok-a", "alice",
		chatv1.MemberKind_MEMBER_KIND_AGENT)
	assert.NilError(t, err)

	enough := errors.New("seen enough")
	var seen []string
	err = att.Forward(t.Context(), func(ev *chatv1.RoomEvent) error {
		switch ev.GetEvent().(type) {
		case *chatv1.RoomEvent_MemberJoined:
			seen = append(seen, "joined")
		case *chatv1.RoomEvent_MessageAppended:
			seen = append(seen, "appended")
			return enough
		}
		return nil
	})
	assert.ErrorIs(t, err, enough)
	assert.DeepEqual(t, seen, []string{"joined", "appended"})
}

// A feed the caller ended itself reports the cancellation rather than the
// transport failure the stream raises on the way down: a caller that retries a
// dropped attendance must not retry its own shutdown.
func TestAttendance_ForwardReportsTheCancellation(t *testing.T) {
	fake := &fakeChatService{self: member("backend", "alice", "/work")}
	d := serveTestDaemon(t, fake, nil)

	ctx, cancel := context.WithCancel(t.Context())
	att, err := d.client.Attend(ctx, "tok-a", "alice",
		chatv1.MemberKind_MEMBER_KIND_AGENT)
	assert.NilError(t, err)

	cancel()
	err = att.Forward(ctx, func(*chatv1.RoomEvent) error { return nil })
	assert.ErrorIs(t, err, context.Canceled)
}

// The target reaches the daemon as it was written, and the answer says who it
// resolved to — which the caller needs whole, since a role nobody attends is in
// it as well.
func TestClient_Send(t *testing.T) {
	fake := &fakeChatService{
		mentioned: []*chatv1.Member{
			member("frontend", "bob", "/work"),
			member("ops", "carol", "/work"),
		},
		absent: []*chatv1.Member{member("ops", "carol", "/work")},
	}
	d := serveTestDaemon(t, fake, nil)

	target, err := ParseTarget("frontend/bob,ops/carol")
	assert.NilError(t, err)
	resp, err := d.client.Send(t.Context(), "tok-a", target, "please review")
	assert.NilError(t, err)
	assert.Equal(t, fake.send.GetText(), "please review")
	assert.Equal(t, TargetString(fake.send.GetTarget()), "frontend/bob,ops/carol")
	assert.Equal(t, len(resp.GetMentioned()), 2)
	assert.Equal(t, Address(resp.GetAbsent()[0]), "ops/carol")
}

// A board post carries no target at all, which is how the wire spells "for
// nobody": the message is in the room and mentions no one.
func TestClient_SendPostCarriesNoTarget(t *testing.T) {
	fake := &fakeChatService{}
	d := serveTestDaemon(t, fake, nil)

	target, err := ParseTarget("")
	assert.NilError(t, err)
	_, err = d.client.Send(t.Context(), "tok-a", target, "fyi: rebased main")
	assert.NilError(t, err)
	assert.Assert(t, fake.send.GetTarget() == nil)
}

// pendingMessage is the one message the read cases below hand over.
func pendingMessage() *chatv1.Message {
	return &chatv1.Message{
		Seq:  7,
		From: member("backend", "alice", "/work"),
		Target: &chatv1.Target{
			Target: &chatv1.Target_Everyone{Everyone: &chatv1.Everyone{}},
		},
		Text:         "ping",
		SentAt:       timestamppb.New(time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)),
		MentionedYou: true,
	}
}

const pendingMessageLine = "7 2026-08-27T09:30:00Z backend/alice -> everyone " +
	"[mentioned you]: ping\n"

func TestClient_ReadInto(t *testing.T) {
	fake := &fakeChatService{messages: []*chatv1.Message{pendingMessage()}}
	d := serveTestDaemon(t, fake, nil)

	var out strings.Builder
	assert.NilError(t, d.client.ReadInto(t.Context(), &out, "tok-b", ReadOptions{}))
	assert.Equal(t, out.String(), pendingMessageLine)
	assert.DeepEqual(t, d.seenTokens(), []string{"tok-b"})
	// No filter is no filter: the daemon's own default is the first ten unread.
	assert.Assert(t, fake.read.GetFilter() == nil)
}

// The written flags reach the daemon as the one read shape it takes, so a read
// asks for what was typed rather than for what the client thought it meant.
func TestClient_ReadCarriesTheFilter(t *testing.T) {
	fake := &fakeChatService{}
	d := serveTestDaemon(t, fake, nil)

	filter, err := ReadFlags{
		Cursor: "head",
		Range:  20,
		To:     "everyone",
		Since:  120,
		Until:  180,
	}.Filter()
	assert.NilError(t, err)

	_, err = d.client.Read(t.Context(), "tok-b", filter)
	assert.NilError(t, err)
	sent := fake.read.GetFilter()
	assert.Equal(t, sent.GetCursor(), chatv1.ReadCursor_READ_CURSOR_HEAD)
	assert.Equal(t, sent.GetRange(), int32(20))
	assert.Equal(t, TargetString(sent.GetTo()), "everyone")
	assert.Equal(t, sent.GetSince(), int64(120))
	assert.Equal(t, sent.GetUntil(), int64(180))
}

// What a read left behind is worth a line: it is the only thing telling the
// reader that another read would hand over more.
func TestClient_ReadIntoReportsTheRemainingUnread(t *testing.T) {
	fake := &fakeChatService{
		messages:  []*chatv1.Message{pendingMessage()},
		remaining: 3,
	}
	d := serveTestDaemon(t, fake, nil)

	var out strings.Builder
	assert.NilError(t, d.client.ReadInto(t.Context(), &out, "tok-b", ReadOptions{}))
	assert.Equal(t, out.String(), pendingMessageLine+"3 more unread\n")
}

// The empty-read line is what a human wants and what a hook has to tell apart
// from messages, so --quiet is the only thing that removes it: output that is
// empty at all then means nothing arrived, with no wording to compare against.
func TestClient_ReadIntoQuietPrintsNothingOnAnEmptyRead(t *testing.T) {
	for _, tc := range []struct {
		name  string
		quiet bool
		want  string
	}{
		{"a plain read says so", false, "no pending messages\n"},
		{"a quiet read says nothing", true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeChatService{}
			d := serveTestDaemon(t, fake, nil)

			var out strings.Builder
			assert.NilError(t, d.client.ReadInto(t.Context(), &out, "tok-b",
				ReadOptions{Quiet: tc.quiet}))
			assert.Equal(t, out.String(), tc.want)
			assert.Assert(t, fake.state == nil, "a read alone reports no state")
		})
	}
}

// A drain that found nothing ends the turn, so the same process reports the
// member done — the state that lets the daemon nudge it when the next message
// arrives. Messages in hand mean the opposite: the turn is about to continue.
func TestClient_ReadIntoDoneWhenEmpty(t *testing.T) {
	for _, tc := range []struct {
		name     string
		messages []*chatv1.Message
		want     string
		wantDone bool
	}{
		{"an empty read reports done", nil, "", true},
		{
			"messages report nothing",
			[]*chatv1.Message{pendingMessage()},
			pendingMessageLine,
			false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeChatService{messages: tc.messages}
			d := serveTestDaemon(t, fake, nil)

			var out strings.Builder
			assert.NilError(t, d.client.ReadInto(t.Context(), &out, "tok-b",
				ReadOptions{Quiet: true, DoneWhenEmpty: true}))
			assert.Equal(t, out.String(), tc.want)

			if !tc.wantDone {
				assert.Assert(t, fake.state == nil)
				return
			}
			assert.Equal(t, fake.state.GetState(), chatv1.HarnessState_HARNESS_STATE_DONE)
			// The report rides on the read's own credential; nothing about the
			// caller is re-resolved on the way.
			assert.DeepEqual(t, d.seenTokens(), []string{"tok-b", "tok-b"})
		})
	}
}

func TestClient_ListMembersAndAddresses(t *testing.T) {
	fake := &fakeChatService{members: []*chatv1.Member{
		memberWith("backend", "alice", "/work",
			chatv1.MemberKind_MEMBER_KIND_AGENT,
			chatv1.HarnessState_HARNESS_STATE_WORKING),
		memberWith("frontend", "bob", "/work",
			chatv1.MemberKind_MEMBER_KIND_HUMAN,
			chatv1.HarnessState_HARNESS_STATE_DONE),
	}}
	d := serveTestDaemon(t, fake, nil)

	var out strings.Builder
	assert.NilError(t, d.client.ListMembers(t.Context(), &out, "tok-a"))
	assert.Equal(t, out.String(),
		"backend/alice  agent  working\nfrontend/bob  human  done\n")

	// Completion needs the same strings as values rather than as a listing.
	addresses, err := d.client.MemberAddresses(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.DeepEqual(t, addresses, []string{"backend/alice", "frontend/bob"})
}

// ReportState is driven by harness hooks whose stdout the harness reads back,
// so it prints nothing at all.
func TestClient_ReportStateIsSilent(t *testing.T) {
	fake := &fakeChatService{}
	d := serveTestDaemon(t, fake, nil)

	assert.NilError(t, d.client.ReportState(t.Context(), "tok-a", "waiting"))
	assert.Equal(t, fake.state.GetState(), chatv1.HarnessState_HARNESS_STATE_WAITING)
}

// An unknown state word never reaches the daemon: reporting the wrong state is
// worse than reporting none, since done is the one state that invites a
// keystroke nudge.
func TestClient_ReportStateRejectsUnknownStateLocally(t *testing.T) {
	fake := &fakeChatService{}
	d := serveTestDaemon(t, fake, nil)

	err := d.client.ReportState(t.Context(), "tok-a", "busy")
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "busy"))
	assert.Assert(t, fake.state == nil)
	assert.Equal(t, len(d.seenTokens()), 0)
}
