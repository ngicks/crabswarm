package harnessctl

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// fakeSession stands in for the MCP session the server holds with its harness.
// It keeps each notification as the JSON it would have gone out as, which is
// the only shape the harness on the other end ever sees.
type fakeSession struct {
	// err, when set, is what the session refuses every notification with — a
	// harness whose stream is already gone.
	err error

	methods []string
	frames  [][]byte
}

func (f *fakeSession) Notify(_ context.Context, method string, params any) error {
	if f.err != nil {
		return f.err
	}
	frame, err := json.Marshal(params)
	if err != nil {
		return err
	}
	f.methods = append(f.methods, method)
	f.frames = append(f.frames, frame)
	return nil
}

// sent unmarshals the one notification the session took. The meta comes back
// as the raw member so a case can tell an absent one from an empty object.
func (f *fakeSession) sent(t *testing.T) (method, content string, meta json.RawMessage) {
	t.Helper()

	assert.Equal(t, len(f.frames), 1, "the session took %d notifications", len(f.frames))
	var params struct {
		Content string          `json:"content"`
		Meta    json.RawMessage `json:"meta"`
	}
	assert.NilError(t, json.Unmarshal(f.frames[0], &params))
	return f.methods[0], params.Content, params.Meta
}

// directOn is what a server spawned as the session's own channel hands
// [Detect].
func directOn(session Session) Options {
	return Options{ClaudeChannel: ClaudeChannel{Kind: ClaudeChannelDirect}, Session: session}
}

// metaKey is what Claude Code accepts in a meta key. A key outside it is
// dropped, which would cost the model the attribute that key carried.
var metaKey = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// A Claude Code that registered this server as its channel attends as a member
// the daemon never types at, and is asked nothing before it attends: the
// channel is the session the server already holds.
func TestDetect_TheDirectChannelTakesTheDelivery(t *testing.T) {
	session := &fakeSession{}
	h := Detect("claude-code", noEnv, directOn(session))

	assert.Equal(t, h.Kind(), chatv1.Harness_HARNESS_CLAUDE_CODE)
	assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)
	_, probed := h.(Prober)
	assert.Assert(t, !probed, "%T is probed for a channel that is its own session", h)
}

// A mention goes out as one channel event: the room's own wording as the
// content, and the sender and the room as the keys Claude Code renders around
// it.
func TestDirectChannel_DeliversAMentionAsAChannelEvent(t *testing.T) {
	session := &fakeSession{}
	h := Detect("claude-code", noEnv, directOn(session))

	notice := Notice{
		From: "alpha/bob",
		Room: "/work/proj",
		Text: "[crabswarm chat] new message from alpha/bob",
	}
	assert.NilError(t, h.Deliver(t.Context(), notice))

	method, content, raw := session.sent(t)
	assert.Equal(t, method, "notifications/claude/channel")
	assert.Equal(t, content, notice.Text)
	var meta map[string]string
	assert.NilError(t, json.Unmarshal(raw, &meta))
	assert.DeepEqual(t, meta, map[string]string{"from": "alpha/bob", "room": "/work/proj"})
	for key := range meta {
		assert.Assert(t, metaKey.MatchString(key), "%q is not a key Claude Code reads", key)
	}
}

// A notice about how much waits names no sender, and the key is left out rather
// than sent empty: the model would otherwise read from="" as the address of
// whoever wrote. A notice with neither key carries no meta at all.
func TestDirectChannel_LeavesOutTheKeyItHasNothingFor(t *testing.T) {
	session := &fakeSession{}
	h := Detect("claude-code", noEnv, directOn(session))

	waiting := "[crabswarm chat] 2 unread messages mention you"
	assert.NilError(t, h.Deliver(t.Context(), Notice{Room: "/work/proj", Text: waiting}))

	_, content, raw := session.sent(t)
	assert.Equal(t, content, waiting)
	var meta map[string]string
	assert.NilError(t, json.Unmarshal(raw, &meta))
	assert.DeepEqual(t, meta, map[string]string{"room": "/work/proj"})

	bare := &fakeSession{}
	h = Detect("claude-code", noEnv, directOn(bare))
	assert.NilError(t, h.Deliver(t.Context(), Notice{Text: "hi"}))
	_, _, raw = bare.sent(t)
	assert.Assert(t, raw == nil, "a notice with no keys carried meta %s", raw)
}

// A session that refused says so rather than reporting a delivery that never
// happened: the caller leaves the mention outstanding and comes back to it.
func TestDirectChannel_ReportsASessionThatRefused(t *testing.T) {
	gone := errors.New("the harness closed its end")
	h := Detect("claude-code", noEnv, directOn(&fakeSession{err: gone}))

	err := h.Deliver(t.Context(), Notice{Text: "hi"})
	assert.ErrorIs(t, err, gone)
	assert.ErrorContains(t, err, "notifications/claude/channel")
}

// A direct channel is the notification it pushes, so one with nowhere to push
// is no channel: the member attends as a terminal one rather than claiming a
// delivery it could never make.
func TestDetect_ADirectChannelWithNoSessionIsTerminal(t *testing.T) {
	h := Detect("claude-code", noEnv, directOn(nil))

	assert.Equal(t, h.Kind(), chatv1.Harness_HARNESS_CLAUDE_CODE)
	assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL)
	assert.ErrorIs(t, h.Deliver(t.Context(), Notice{Text: "hi"}), errNoChannel)
}
