package chat

import (
	"errors"
	"fmt"
	"testing"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"
)

func TestService_SendDeliversAndNotifies(t *testing.T) {
	svc, provider, notifier := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")
	seedAgent(t, svc, provider, "tok-b", "/work", "alpha", "bob")

	sent, err := svc.Send(callCtx(t, "tok-a"), &chatv1.SendRequest{To: "bob", Text: "ping"})
	assert.NilError(t, err)
	assert.Equal(t, sent.GetRecipient().GetName(), "bob")

	got := notifier.notified()
	assert.Equal(t, len(got), 1)
	assert.Equal(t, got[0].recipient.Name, "bob")
	assert.Equal(t, got[0].from.Name, "ana")
	assert.Equal(t, got[0].text, "ping")

	read, err := svc.Read(callCtx(t, "tok-b"), &chatv1.ReadRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(read.GetMessages()), 1)
	assert.Equal(t, read.GetMessages()[0].GetText(), "ping")
	assert.Equal(t, read.GetMessages()[0].GetFrom().GetName(), "ana")
	assert.Equal(t, read.GetMessages()[0].GetFrom().GetTeam(), "alpha")
	assert.Assert(t, read.GetMessages()[0].GetSentAt() != nil)

	// Reading drains: nothing is handed out twice.
	read, err = svc.Read(callCtx(t, "tok-b"), &chatv1.ReadRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(read.GetMessages()), 0)
}

func TestService_SendMapsAddressErrors(t *testing.T) {
	svc, provider, _ := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")
	seedAgent(t, svc, provider, "tok-b", "/work", "beta", "bob")
	seedAgent(t, svc, provider, "tok-c", "/work", "gamma", "bob")

	_, err := svc.Send(callCtx(t, "tok-a"), &chatv1.SendRequest{To: "nobody", Text: "hi"})
	assert.Equal(t, status.Code(err), codes.NotFound)

	// "bob" exists in two other teams, so the bare name needs a team prefix.
	_, err = svc.Send(callCtx(t, "tok-a"), &chatv1.SendRequest{To: "bob", Text: "hi"})
	assert.Equal(t, status.Code(err), codes.InvalidArgument)

	res, err := svc.Send(callCtx(t, "tok-a"), &chatv1.SendRequest{To: "beta/bob", Text: "hi"})
	assert.NilError(t, err)
	assert.Equal(t, res.GetRecipient().GetTeam(), "beta")

	_, err = svc.Send(callCtx(t, "tok-a"), &chatv1.SendRequest{To: "beta/bob", Text: ""})
	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestService_BroadcastExcludesSender(t *testing.T) {
	svc, provider, notifier := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")
	seedAgent(t, svc, provider, "tok-b", "/work", "alpha", "bob")
	seedAgent(t, svc, provider, "tok-c", "/work", "beta", "cid")
	seedAgent(t, svc, provider, "tok-d", "/elsewhere", "alpha", "dee")

	res, err := svc.Broadcast(callCtx(t, "tok-a"), &chatv1.BroadcastRequest{Text: "standup"})
	assert.NilError(t, err)
	assert.Equal(t, res.GetDeliveredCount(), int32(2))
	assert.Equal(t, len(notifier.notified()), 2)

	// The sender keeps an empty inbox; the other room hears nothing.
	own, err := svc.Read(callCtx(t, "tok-a"), &chatv1.ReadRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(own.GetMessages()), 0)
	other, err := svc.Read(callCtx(t, "tok-d"), &chatv1.ReadRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(other.GetMessages()), 0)
}

func TestService_HistoryShowsTheWholeRoom(t *testing.T) {
	svc, provider, _ := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")
	seedAgent(t, svc, provider, "tok-b", "/work", "alpha", "bob")
	seedAgent(t, svc, provider, "tok-c", "/work", "beta", "cid")

	_, err := svc.Send(callCtx(t, "tok-a"), &chatv1.SendRequest{To: "bob", Text: "ping"})
	assert.NilError(t, err)
	_, err = svc.Broadcast(callCtx(t, "tok-a"), &chatv1.BroadcastRequest{Text: "standup"})
	assert.NilError(t, err)
	// Draining an inbox is not supposed to change what the room remembers.
	_, err = svc.Read(callCtx(t, "tok-b"), &chatv1.ReadRequest{})
	assert.NilError(t, err)

	// cid was never addressed by the send, and still reads it: the transcript
	// belongs to the room rather than to a recipient.
	res, err := svc.History(callCtx(t, "tok-c"), &chatv1.HistoryRequest{})
	assert.NilError(t, err)
	entries := res.GetEntries()
	assert.Equal(t, len(entries), 2)
	assert.Equal(t, entries[0].GetText(), "ping")
	assert.Equal(t, entries[0].GetFrom().GetName(), "ana")
	assert.Equal(t, entries[0].GetTo().GetName(), "bob")
	assert.Equal(t, entries[0].GetTo().GetTeam(), "alpha")
	assert.Assert(t, entries[0].GetSentAt() != nil)
	assert.Equal(t, entries[1].GetText(), "standup")
	assert.Assert(t, entries[1].GetTo() == nil)

	// Asking again returns the same window: nothing was consumed.
	again, err := svc.History(callCtx(t, "tok-c"), &chatv1.HistoryRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(again.GetEntries()), 2)
}

func TestService_HistoryWindowIsBounded(t *testing.T) {
	svc, provider, _ := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")

	for i := range defaultHistoryWindow + 2 {
		_, err := svc.Broadcast(callCtx(t, "tok-a"),
			&chatv1.BroadcastRequest{Text: fmt.Sprintf("line %d", i)})
		assert.NilError(t, err)
	}

	// No limit asked for: the server's own window, ending at the newest line.
	res, err := svc.History(callCtx(t, "tok-a"), &chatv1.HistoryRequest{})
	assert.NilError(t, err)
	entries := res.GetEntries()
	assert.Equal(t, len(entries), defaultHistoryWindow)
	assert.Equal(t, entries[len(entries)-1].GetText(),
		fmt.Sprintf("line %d", defaultHistoryWindow+1))

	// A smaller window is honoured, a larger one answers with what the room
	// keeps rather than failing.
	res, err = svc.History(callCtx(t, "tok-a"), &chatv1.HistoryRequest{Limit: 3})
	assert.NilError(t, err)
	assert.Equal(t, len(res.GetEntries()), 3)
	res, err = svc.History(callCtx(t, "tok-a"), &chatv1.HistoryRequest{Limit: 100_000})
	assert.NilError(t, err)
	assert.Equal(t, len(res.GetEntries()), defaultHistoryWindow+2)
}

func TestService_HistoryRequiresMembership(t *testing.T) {
	svc, provider, _ := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")

	// No token at all, and a token the provider vouches for but that never
	// joined: neither has a room to read.
	_, err := svc.History(t.Context(), &chatv1.HistoryRequest{})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
	provider.vouch("tok-outsider", "/work", "alpha")
	_, err = svc.History(callCtx(t, "tok-outsider"), &chatv1.HistoryRequest{})
	assert.Equal(t, status.Code(err), codes.Unauthenticated)

	// A member of a room where nothing was said reads an empty transcript.
	res, err := svc.History(callCtx(t, "tok-a"), &chatv1.HistoryRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(res.GetEntries()), 0)
}

func TestService_NotifierFailureDoesNotFailSend(t *testing.T) {
	svc, provider, notifier := newTestService(t)
	seedAgent(t, svc, provider, "tok-a", "/work", "alpha", "ana")
	seedAgent(t, svc, provider, "tok-b", "/work", "alpha", "bob")
	notifier.fail = true
	notifier.err = errors.New("send-keys: no such command")

	_, err := svc.Send(callCtx(t, "tok-a"), &chatv1.SendRequest{To: "bob", Text: "ping"})
	assert.NilError(t, err)

	// The message is stored regardless of the failed nudge.
	read, err := svc.Read(callCtx(t, "tok-b"), &chatv1.ReadRequest{})
	assert.NilError(t, err)
	assert.Equal(t, len(read.GetMessages()), 1)
}
