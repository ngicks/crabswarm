package chat

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// sendAs is [Service.Send] for token, failing the test on error.
func sendAs(
	t *testing.T,
	svc *Service,
	token string,
	target *chatv1.Target,
	text string,
) *chatv1.SendResponse {
	t.Helper()
	res, err := svc.Send(callCtx(t, token), &chatv1.SendRequest{Target: target, Text: text})
	assert.NilError(t, err)
	return res
}

// addresses renders a list of members the way the tests name them.
func addresses(members []*chatv1.Member) []string {
	out := make([]string, len(members))
	for i, m := range members {
		out[i] = address(m)
	}
	return out
}

// targetAddresses renders the roles a message was written to.
func targetAddresses(t *chatv1.Target) []string {
	roles := t.GetRoles().GetRoles()
	out := make([]string, len(roles))
	for i, r := range roles {
		out[i] = r.GetTeam() + "/" + r.GetName()
	}
	return out
}

// messageTexts is what a read returned, for comparing a whole window at once.
func messageTexts(messages []*chatv1.Message) []string {
	out := make([]string, len(messages))
	for i, m := range messages {
		out[i] = m.GetText()
	}
	return out
}

// Only the named roles are mentioned, and only the ones that are attending
// agents are worth interrupting: a person reads at their own pace, and a role
// nobody is attending under has no terminal to type into.
func TestService_SendToRolesNudgesTheAttendingAgents(t *testing.T) {
	svc, provider, notifier := newTestService(t)
	agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")
	agent(t, svc, provider, "tok-b", testRoom, "alpha", "bob")
	human(t, svc, provider, "tok-h", testRoom, "alpha", "hana")
	// Attended once and gone: the role exists, so it can be addressed, and the
	// mention waits at its read position.
	carl := agent(t, svc, provider, "tok-c", testRoom, "alpha", "carl")
	assert.Assert(t, carl.close(t) != nil)

	res := sendAs(t, svc, "tok-a", to("bob", "hana", "carl"), "standup")
	assert.DeepEqual(t, addresses(res.GetMentioned()),
		[]string{"alpha/bob", "alpha/hana", "alpha/carl"})
	assert.DeepEqual(t, addresses(res.GetAbsent()), []string{"alpha/carl"})
	assert.DeepEqual(t, notifier.nudged(), []string{"alpha/bob"})

	got := notifier.notified()[0]
	assert.Equal(t, got.from.Name, "ana")
	assert.Equal(t, got.text, "standup")
}

// A sender that names its own role is mentioned, since that is what it wrote,
// but not nudged: its own message is never unread for it, so the wake-up would
// buy it an empty read.
func TestService_SendToRolesSkipsTheSender(t *testing.T) {
	svc, provider, notifier := newTestService(t)
	agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")
	agent(t, svc, provider, "tok-b", testRoom, "alpha", "bob")

	res := sendAs(t, svc, "tok-a", to("ana", "bob"), "ana and bob, both")
	assert.DeepEqual(t, addresses(res.GetMentioned()), []string{"alpha/ana", "alpha/bob"})
	assert.DeepEqual(t, notifier.nudged(), []string{"alpha/bob"})
}

func TestService_SendToEveryoneNudgesEveryAgentButTheSender(t *testing.T) {
	svc, provider, notifier := newTestService(t)
	agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")
	agent(t, svc, provider, "tok-b", testRoom, "alpha", "bob")
	agent(t, svc, provider, "tok-d", testRoom, "alpha", "dave")
	human(t, svc, provider, "tok-h", testRoom, "alpha", "hana")
	agent(t, svc, provider, "tok-far", "/work/elsewhere", "alpha", "stranger")

	// Nobody in particular is named, so nobody is mentioned or absent; the
	// wake-up still goes out, minus the sender, who already knows.
	res := sendAs(t, svc, "tok-a", everyone(), "deploying")
	assert.Equal(t, len(res.GetMentioned()), 0)
	assert.Equal(t, len(res.GetAbsent()), 0)
	assert.DeepEqual(t, notifier.nudged(), []string{"alpha/bob", "alpha/dave"})
}

// A board post is addressed to nobody: it sits in the room for whoever comes
// looking, and interrupts no one on the way.
func TestService_SendPostNudgesNobody(t *testing.T) {
	svc, provider, notifier := newTestService(t)
	agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")
	agent(t, svc, provider, "tok-b", testRoom, "alpha", "bob")

	res := sendAs(t, svc, "tok-a", nil, "notes are in the wiki")
	assert.Equal(t, len(res.GetMentioned()), 0)
	assert.Equal(t, len(notifier.nudged()), 0)

	// In the room all the same, and nobody's unread.
	read, err := svc.Read(callCtx(t, "tok-b"), &chatv1.ReadRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(read.GetMessages()), 0)

	posted, err := svc.store.ReadRoom(t.Context(), testRoom, ReadFilter{})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(posted), []string{"notes are in the wiki"})
}

func TestService_SendMapsTargetErrors(t *testing.T) {
	svc, provider, _ := newTestService(t)
	agent(t, svc, provider, "tok-a", testRoom, "gamma", "ana")
	agent(t, svc, provider, "tok-b", testRoom, "alpha", "bob")
	agent(t, svc, provider, "tok-b2", testRoom, "beta", "bob")

	_, err := svc.Send(callCtx(t, "tok-a"), &chatv1.SendRequest{
		Target: to("nobody"), Text: "hi",
	})
	assert.Equal(t, status.Code(err), codes.NotFound)

	// The sender is in neither team that carries the name, so nothing breaks
	// the tie and the caller is told to write it as "team/name".
	_, err = svc.Send(callCtx(t, "tok-a"), &chatv1.SendRequest{
		Target: to("bob"), Text: "hi",
	})
	assert.Equal(t, status.Code(err), codes.InvalidArgument)

	_, err = svc.Send(callCtx(t, "tok-a"), &chatv1.SendRequest{
		Target: to("alpha/bob"), Text: "",
	})
	assert.Equal(t, status.Code(err), codes.InvalidArgument)

	// A roles target naming nobody is the address that was left half written.
	_, err = svc.Send(callCtx(t, "tok-a"), &chatv1.SendRequest{
		Target: &chatv1.Target{
			Target: &chatv1.Target_Roles{Roles: &chatv1.Roles{}},
		},
		Text: "hi",
	})
	assert.Equal(t, status.Code(err), codes.InvalidArgument)

	// Nothing was recorded by any of them.
	said, err := svc.store.ReadRoom(t.Context(), testRoom, ReadFilter{})
	assert.NilError(t, err)
	assert.Equal(t, len(said), 0)
}

func TestService_ReadDefaultsToWhatTheCallerHasNotSeen(t *testing.T) {
	svc, provider, _ := newTestService(t)
	agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")
	agent(t, svc, provider, "tok-b", testRoom, "alpha", "bob")

	sendAs(t, svc, "tok-a", to("bob"), "one")
	sendAs(t, svc, "tok-a", everyone(), "two")
	sendAs(t, svc, "tok-a", nil, "three")
	sendAs(t, svc, "tok-b", everyone(), "four")

	// Addressed to bob or to everyone, not sent by bob, and not a post.
	read, err := svc.Read(callCtx(t, "tok-b"), &chatv1.ReadRequest{})
	assert.NilError(t, err)
	assert.DeepEqual(t, messageTexts(read.GetMessages()), []string{"one", "two"})
	assert.Equal(t, read.GetRemainingUnread(), int32(0))

	first := read.GetMessages()[0]
	assert.Assert(t, first.GetId() != "")
	assert.Equal(t, first.GetSeq(), int64(1))
	assert.Equal(t, address(first.GetFrom()), "alpha/ana")
	assert.Assert(t, first.GetMentionedYou())
	assert.DeepEqual(t, targetAddresses(first.GetTarget()), []string{"alpha/bob"})
	assert.Assert(t, read.GetMessages()[1].GetTarget().GetEveryone() != nil)

	// Having been shown, they are not shown again.
	read, err = svc.Read(callCtx(t, "tok-b"), &chatv1.ReadRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(read.GetMessages()), 0)
}

func TestService_ReadHonoursTheFilter(t *testing.T) {
	svc, provider, _ := newTestService(t)
	agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")
	agent(t, svc, provider, "tok-b", testRoom, "alpha", "bob")

	sendAs(t, svc, "tok-a", to("bob"), "one")
	sendAs(t, svc, "tok-a", everyone(), "two")
	sendAs(t, svc, "tok-a", nil, "three")
	sendAs(t, svc, "tok-a", to("bob"), "four")

	for _, tc := range []struct {
		name   string
		filter *chatv1.ReadFilter
		want   []string
	}{
		{
			// The tail reads the room rather than an inbox: posts and all.
			"the tail counts backward",
			&chatv1.ReadFilter{
				Cursor: chatv1.ReadCursor_READ_CURSOR_TAIL, Range: -2,
			},
			[]string{"three", "four"},
		},
		{
			"the head counts forward",
			&chatv1.ReadFilter{
				Cursor: chatv1.ReadCursor_READ_CURSOR_HEAD, Range: 2,
			},
			[]string{"one", "two"},
		},
		{
			"a target narrows to what named it",
			&chatv1.ReadFilter{
				Cursor: chatv1.ReadCursor_READ_CURSOR_HEAD, To: to("alpha/bob"),
			},
			[]string{"one", "four"},
		},
		{
			"since and until bound the seqs, both exclusive",
			&chatv1.ReadFilter{
				Cursor: chatv1.ReadCursor_READ_CURSOR_HEAD, Since: 1, Until: 4,
			},
			[]string{"two", "three"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A fresh reader per case: a read moves the position, so cases
			// sharing one would read what the case before it left behind.
			svc, provider, _ := newTestService(t)
			agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")
			agent(t, svc, provider, "tok-b", testRoom, "alpha", "bob")
			sendAs(t, svc, "tok-a", to("bob"), "one")
			sendAs(t, svc, "tok-a", everyone(), "two")
			sendAs(t, svc, "tok-a", nil, "three")
			sendAs(t, svc, "tok-a", to("bob"), "four")

			read, err := svc.Read(callCtx(t, "tok-b"),
				&chatv1.ReadRequest{Filter: tc.filter})
			assert.NilError(t, err)
			assert.DeepEqual(t, messageTexts(read.GetMessages()), tc.want)
		})
	}
}

// Nothing lies before the head or after the tail, and unread counts forward
// only, so a range running against its cursor asks for what cannot exist.
func TestService_ReadRejectsARangeAgainstItsCursor(t *testing.T) {
	svc, provider, _ := newTestService(t)
	agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")

	for _, filter := range []*chatv1.ReadFilter{
		{Cursor: chatv1.ReadCursor_READ_CURSOR_HEAD, Range: -1},
		{Cursor: chatv1.ReadCursor_READ_CURSOR_TAIL, Range: 1},
		{Cursor: chatv1.ReadCursor_READ_CURSOR_UNREAD, Range: -1},
	} {
		_, err := svc.Read(callCtx(t, "tok-a"), &chatv1.ReadRequest{Filter: filter})
		assert.Equal(t, status.Code(err), codes.InvalidArgument,
			"cursor %s range %d", filter.GetCursor(), filter.GetRange())
	}
}

// The message is already in the room by the time the nudge is attempted, so a
// terminal that could not be typed into costs its reader a late read.
func TestService_NotifierFailureDoesNotFailSend(t *testing.T) {
	svc, provider, notifier := newTestService(t)
	notifier.fail = true
	notifier.err = errors.New("cmdman: send-keys declined")
	agent(t, svc, provider, "tok-a", testRoom, "alpha", "ana")
	agent(t, svc, provider, "tok-b", testRoom, "alpha", "bob")

	sendAs(t, svc, "tok-a", to("bob"), "hi")
	assert.DeepEqual(t, notifier.nudged(), []string{"alpha/bob"})

	read, err := svc.Read(callCtx(t, "tok-b"), &chatv1.ReadRequest{})
	assert.NilError(t, err)
	assert.DeepEqual(t, messageTexts(read.GetMessages()), []string{"hi"})
}
