package chat

import (
	"errors"
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
