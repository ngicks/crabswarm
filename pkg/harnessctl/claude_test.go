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

// sent unmarshals the one notification the session took.
func (f *fakeSession) sent(t *testing.T) (method string, content string, meta map[string]string) {
	t.Helper()

	assert.Equal(t, len(f.frames), 1, "the session took %d notifications", len(f.frames))
	var params struct {
		Content string            `json:"content"`
		Meta    map[string]string `json:"meta"`
	}
	assert.NilError(t, json.Unmarshal(f.frames[0], &params))
	return f.methods[0], params.Content, params.Meta
}

// channelEnv is the environment of a Claude Code that was launched as a channel
// host, or of one that was not.
func channelEnv(value string) func(string) string {
	return func(name string) string {
		if name == ClaudeChannelEnv {
			return value
		}
		return ""
	}
}

// Whether this server is a registered channel is decided at launch and never
// said again, so the variable the launcher sets beside the flag is the whole of
// what the harness has to go on. Anything but the launcher's own value leaves
// the member terminal, where the daemon still reaches it.
func TestNewClaudeCode_TakesTheChannelOnlyFromItsLauncher(t *testing.T) {
	for _, value := range []string{"", "0", "true", "yes", "2"} {
		t.Run(value, func(t *testing.T) {
			assert.Assert(t, newClaudeCode(channelEnv(value), &fakeSession{}) == nil)
		})
	}

	h := newClaudeCode(channelEnv("1"), &fakeSession{})
	assert.Assert(t, h != nil)
	assert.Equal(t, h.Kind(), chatv1.Harness_HARNESS_CLAUDE_CODE)
	assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)
}

// A channel is the notification it pushes, so one with nowhere to push is no
// channel: the member attends as a terminal one rather than claiming a delivery
// it could never make.
func TestNewClaudeCode_RefusesASessionItCannotNotify(t *testing.T) {
	assert.Assert(t, newClaudeCode(channelEnv("1"), nil) == nil)
}

// metaKey is what Claude Code accepts in a meta key. A key outside it is
// dropped, which would cost the model the attribute that key carried.
var metaKey = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// A mention goes out as one channel event: the room's own wording as the
// content, and the sender and the room as the keys Claude Code renders around
// it.
func TestClaudeCode_DeliversAMentionAsAChannelEvent(t *testing.T) {
	session := &fakeSession{}
	h := newClaudeCode(channelEnv("1"), session)

	notice := Notice{
		From: "alpha/bob",
		Room: "/work/proj",
		Text: "[crabswarm chat] new message from alpha/bob",
	}
	assert.NilError(t, h.Deliver(t.Context(), notice))

	method, content, meta := session.sent(t)
	assert.Equal(t, method, "notifications/claude/channel")
	assert.Equal(t, content, notice.Text)
	assert.DeepEqual(t, meta, map[string]string{"from": "alpha/bob", "room": "/work/proj"})
	for key := range meta {
		assert.Assert(t, metaKey.MatchString(key), "%q is not a key Claude Code reads", key)
	}
}

// A notice about how much waits names no sender, and the key is left out rather
// than sent empty: the model would otherwise read from="" as the address of
// whoever wrote.
func TestClaudeCode_LeavesOutTheKeyItHasNothingFor(t *testing.T) {
	session := &fakeSession{}
	h := newClaudeCode(channelEnv("1"), session)

	waiting := "[crabswarm chat] 2 unread messages mention you"
	assert.NilError(t, h.Deliver(t.Context(), Notice{Room: "/work/proj", Text: waiting}))

	_, content, meta := session.sent(t)
	assert.Equal(t, content, waiting)
	assert.DeepEqual(t, meta, map[string]string{"room": "/work/proj"})
}

// A session that refused says so rather than reporting a delivery that never
// happened: the caller leaves the mention outstanding and comes back to it.
func TestClaudeCode_ReportsASessionThatRefused(t *testing.T) {
	gone := errors.New("the harness closed its end")
	h := newClaudeCode(channelEnv("1"), &fakeSession{err: gone})

	err := h.Deliver(t.Context(), Notice{Text: "hi"})
	assert.ErrorIs(t, err, gone)
	assert.ErrorContains(t, err, "notifications/claude/channel")
}

// The whole of it through [Detect], which is the one caller: a Claude Code
// launched as a channel host attends as a member the daemon never types at.
func TestDetect_TheClaudeChannelTakesTheDelivery(t *testing.T) {
	session := &fakeSession{}
	h := Detect("claude-code", channelEnv("1"), session)

	assert.Equal(t, h.Kind(), chatv1.Harness_HARNESS_CLAUDE_CODE)
	assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)
	assert.NilError(t, h.Deliver(t.Context(), Notice{Room: "/work/proj", Text: "hi"}))

	method, content, _ := session.sent(t)
	assert.Equal(t, method, "notifications/claude/channel")
	assert.Equal(t, content, "hi")
}
