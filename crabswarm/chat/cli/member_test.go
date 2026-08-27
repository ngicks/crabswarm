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

func TestClient_Read(t *testing.T) {
	fake := &fakeChatService{messages: []*chatv1.Message{{
		From:   member("backend", "alice", "/work"),
		Text:   "ping",
		SentAt: timestamppb.New(time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)),
	}}}
	d := serveTestDaemon(t, fake, nil)

	var out strings.Builder
	assert.NilError(t, d.client.Read(t.Context(), &out, "tok-b"))
	assert.Equal(t, out.String(), "[2026-08-27T09:30:00Z] backend/alice: ping\n")
	assert.DeepEqual(t, d.seenTokens(), []string{"tok-b"})
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
