package mcp

import (
	"log/slog"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// launchedAsTheChannel is the environment of a server whose launcher registered
// the server itself as the channel of the Claude Code session beside it.
func launchedAsTheChannel(name string) string {
	if name == harnessctl.ClaudeChannelEnv {
		return "direct"
	}
	return ""
}

// A server registered as the channel declares the capability Claude Code
// listens on, caps the revision Claude Code still delivers channel events on,
// and tells the model what an event from it is. Its member attends as one the
// daemon never types at: the server pushes the mention itself.
func TestServer_DeclaresTheDirectChannel(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	bridge := newTestBridgeIn(t, fake, launchedAsTheChannel)
	session := serveBridgeAs(t, bridge, "claude-code")

	res := session.InitializeResult()
	_, declared := experimental(res)[harnessctl.ClaudeChannelCapability]
	assert.Assert(t, declared, "the handshake declares %v", experimental(res))
	assert.Assert(t, res.ProtocolVersion <= maxProtocolVersion,
		"the session settled on %s, above the cap", res.ProtocolVersion)
	for _, want := range []string{serverName, "chat_read", "chat_send", "Do not reply"} {
		assert.Assert(t, strings.Contains(res.Instructions, want),
			"the instructions do not name %q:\n%s", want, res.Instructions)
	}

	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })
	assert.Equal(t, fake.lastAttend().GetNudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)
}

// A channel declaration the launcher would not write is refused at startup
// rather than read as no channel: a misspelled one would leave the member typed
// at while its launcher believes otherwise.
func TestNewServer_RefusesAnUnknownChannel(t *testing.T) {
	_, err := newServer(slog.New(slog.DiscardHandler), "/nonexistent.sock", testToken,
		func(name string) string {
			if name == harnessctl.ClaudeChannelEnv {
				return "telegram"
			}
			return ""
		})
	assert.ErrorContains(t, err, harnessctl.ClaudeChannelEnv)
}
