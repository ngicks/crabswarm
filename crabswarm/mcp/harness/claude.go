package harness

import (
	"context"
	"fmt"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// The Claude Code channel contract, as a registered session speaks it.
//
// A channel is a server Claude Code lets push turns into a running session. The
// server declares [ClaudeChannelCapability] among its experimental capabilities
// and then pushes [claudeChannelMethod] notifications; the model sees each one
// as a <channel> element naming the server, the sender and the room, and starts
// a turn on it when it is idle.
const (
	// ClaudeChannelEnv is set to "1" by whoever launched Claude Code with the
	// flag that registers this server as a channel.
	//
	// Registration is decided at launch and nothing in the MCP session reports
	// it, so the server cannot tell a registered session from an ordinary one.
	// The launcher sets this beside the flag, which makes the variable the only
	// honest answer available: without it the notifications would be dropped
	// unseen and the member would wait on a delivery nobody could make.
	ClaudeChannelEnv = "CRABSWARM_CLAUDE_CHANNEL"

	// ClaudeChannelCapability is the experimental capability a channel server
	// declares. Declaring it is what makes Claude Code listen, so a server that
	// was not launched as a channel declares nothing.
	ClaudeChannelCapability = "claude/channel"

	// claudeChannelMethod carries one channel event.
	claudeChannelMethod = "notifications/claude/channel"
)

// The meta keys one channel event carries. Claude Code renders them as
// attributes of the <channel> element the model reads, and accepts
// [A-Za-z0-9_] in a key.
const (
	claudeMetaFrom = "from"
	claudeMetaRoom = "room"
)

// ClaudeChannelEnabled reports whether this process was launched as a channel of
// the Claude Code session that started it.
func ClaudeChannelEnabled(getenv func(string) string) bool {
	return getenv(ClaudeChannelEnv) == "1"
}

// newClaudeCode builds the claude channel, or answers nil for a session that is
// not a registered channel and so has to be woken through its terminal.
//
// A nil session is the same case: a channel is the notification it pushes, and
// one with nowhere to push would claim a delivery it could never make.
func newClaudeCode(getenv func(string) string, session Session) Harness {
	if !ClaudeChannelEnabled(getenv) || session == nil {
		return nil
	}
	return claudeCode{session: session}
}

// claudeCode delivers a notice as a channel event of the MCP session the server
// already holds with Claude Code. There is no second connection: the harness
// spawned this server, and the stream it spawned it on is the channel.
type claudeCode struct {
	session Session
}

func (claudeCode) Kind() chatv1.Harness { return chatv1.Harness_HARNESS_CLAUDE_CODE }

func (claudeCode) Nudge() chatv1.NudgeDelivery {
	return chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE
}

// claudeChannelParams is one channel event: the line the model reads, and the
// keys Claude Code renders around it.
type claudeChannelParams struct {
	Content string            `json:"content"`
	Meta    map[string]string `json:"meta,omitzero"`
}

// Deliver pushes the notice into the session as a channel event.
func (c claudeCode) Deliver(ctx context.Context, n Notice) error {
	params := claudeChannelParams{Content: n.Text}
	// Only the keys there is something to say with — a notice about how much
	// waits names no sender, and an empty value would render as from="" in the
	// element the model reads. The map stays nil until one of them lands, so a
	// notice with neither carries no meta at all rather than an empty object.
	put := func(key, value string) {
		if value == "" {
			return
		}
		if params.Meta == nil {
			params.Meta = map[string]string{}
		}
		params.Meta[key] = value
	}
	put(claudeMetaFrom, n.From)
	put(claudeMetaRoom, n.Room)
	if err := c.session.Notify(ctx, claudeChannelMethod, params); err != nil {
		return fmt.Errorf("pushing a %s notification: %w", claudeChannelMethod, err)
	}
	return nil
}
