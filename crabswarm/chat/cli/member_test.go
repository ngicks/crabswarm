package cli

import (
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

func TestClient_Join(t *testing.T) {
	fake := &fakeChatService{self: member("backend", "alice", "/work/proj")}
	d := serveTestDaemon(t, fake, nil)

	var out strings.Builder
	assert.NilError(t, d.client.Join(t.Context(), &out, "tok-a", "alice"))
	assert.Equal(t, fake.join.GetName(), "alice")
	assert.Equal(t, out.String(), "joined /work/proj as backend/alice\n")

	// An unnamed join sends an empty name: naming the member is the daemon's
	// job when the caller declines to.
	assert.NilError(t, d.client.Join(t.Context(), &strings.Builder{}, "tok-a", ""))
	assert.Equal(t, fake.join.GetName(), "")
}

// The address is handed to the daemon exactly as typed — resolving a bare name
// against the caller's team, then the room, is a decision only the daemon can
// make.
func TestClient_SendPassesTheAddressThrough(t *testing.T) {
	fake := &fakeChatService{recipient: member("frontend", "bob", "/work/proj")}
	d := serveTestDaemon(t, fake, nil)

	var out strings.Builder
	assert.NilError(t, d.client.Send(t.Context(), &out, "tok-a", "bob", "ping"))
	assert.Equal(t, fake.send.GetTo(), "bob")
	assert.Equal(t, fake.send.GetText(), "ping")
	assert.Equal(t, out.String(), "sent to frontend/bob\n")

	assert.NilError(t,
		d.client.Send(t.Context(), &strings.Builder{}, "tok-a", "frontend/bob", "ping"))
	assert.Equal(t, fake.send.GetTo(), "frontend/bob")
}

func TestClient_Broadcast(t *testing.T) {
	fake := &fakeChatService{delivered: 2}
	d := serveTestDaemon(t, fake, nil)

	var out strings.Builder
	assert.NilError(t, d.client.Broadcast(t.Context(), &out, "tok-a", "standup in 5"))
	assert.Equal(t, fake.broadcast.GetText(), "standup in 5")
	assert.Equal(t, out.String(), "broadcast to 2 members\n")
}

// pendingMessage is the one message the read cases below hand over.
func pendingMessage() *chatv1.Message {
	return &chatv1.Message{
		From:   member("backend", "alice", "/work"),
		Text:   "ping",
		SentAt: timestamppb.New(time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)),
	}
}

const pendingMessageLine = "[2026-08-27T09:30:00Z] backend/alice: ping\n"

func TestClient_Read(t *testing.T) {
	fake := &fakeChatService{messages: []*chatv1.Message{pendingMessage()}}
	d := serveTestDaemon(t, fake, nil)

	var out strings.Builder
	assert.NilError(t, d.client.Read(t.Context(), &out, "tok-b", ReadOptions{}))
	assert.Equal(t, out.String(), pendingMessageLine)
	assert.DeepEqual(t, d.seenTokens(), []string{"tok-b"})
}

// The empty-inbox line is what a human wants and what a hook has to tell apart
// from mail, so --quiet is the only thing that removes it: output that is empty
// at all then means nothing arrived, with no wording to compare against.
func TestClient_ReadQuietPrintsNothingOnAnEmptyInbox(t *testing.T) {
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
			assert.NilError(t,
				d.client.Read(t.Context(), &out, "tok-b", ReadOptions{Quiet: tc.quiet}))
			assert.Equal(t, out.String(), tc.want)
			assert.Assert(t, fake.state == nil, "a read alone reports no state")
		})
	}
}

// A drain that found nothing ends the turn, so the same process reports the
// member idle — the state that lets the daemon nudge it when the next message
// arrives. Messages in hand mean the opposite: the turn is about to continue.
func TestClient_ReadIdleWhenEmpty(t *testing.T) {
	for _, tc := range []struct {
		name     string
		messages []*chatv1.Message
		want     string
		wantIdle bool
	}{
		{"an empty inbox reports idle", nil, "", true},
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
			assert.NilError(t, d.client.Read(t.Context(), &out, "tok-b",
				ReadOptions{Quiet: true, IdleWhenEmpty: true}))
			assert.Equal(t, out.String(), tc.want)

			if !tc.wantIdle {
				assert.Assert(t, fake.state == nil)
				return
			}
			assert.Equal(t, fake.state.GetState(), chatv1.HarnessState_HARNESS_STATE_IDLE)
			// The report rides on the read's own credential; nothing about the
			// caller is re-resolved on the way.
			assert.DeepEqual(t, d.seenTokens(), []string{"tok-b", "tok-b"})
		})
	}
}

func TestClient_ListMembersAndAddresses(t *testing.T) {
	fake := &fakeChatService{members: []*chatv1.Member{
		member("backend", "alice", "/work"),
		member("frontend", "bob", "/work"),
	}}
	d := serveTestDaemon(t, fake, nil)

	var out strings.Builder
	assert.NilError(t, d.client.ListMembers(t.Context(), &out, "tok-a"))
	assert.Equal(t, out.String(), "backend/alice\nfrontend/bob\n")

	// Completion needs the same strings as values rather than as a listing.
	addresses, err := d.client.MemberAddresses(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.DeepEqual(t, addresses, []string{"backend/alice", "frontend/bob"})
}

func TestClient_Leave(t *testing.T) {
	fake := &fakeChatService{}
	d := serveTestDaemon(t, fake, nil)

	var out strings.Builder
	assert.NilError(t, d.client.Leave(t.Context(), &out, "tok-a"))
	assert.Equal(t, fake.leaveCalls, 1)
	assert.Equal(t, out.String(), "left the room\n")
}

// ReportState is driven by harness hooks whose stdout the harness reads back,
// so it prints nothing at all.
func TestClient_ReportStateIsSilent(t *testing.T) {
	fake := &fakeChatService{}
	d := serveTestDaemon(t, fake, nil)

	assert.NilError(t, d.client.ReportState(t.Context(), "tok-a", "waiting_input"))
	assert.Equal(t, fake.state.GetState(), chatv1.HarnessState_HARNESS_STATE_WAITING_INPUT)
}

// An unknown state word never reaches the daemon: reporting the wrong state is
// worse than reporting none, since idle is the one state that invites a
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
