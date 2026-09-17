// Package harness recognises the CLI an MCP server is serving and hands back
// the channel a mention reaches its agent through.
//
// A harness names itself once, in the MCP handshake, and everything about how
// it can be woken follows from that name: Claude Code takes a channel
// notification, Codex listens on its app server, OpenCode relays through a
// plugin, and a harness nobody recognises takes nothing at all. The server asks
// here and attends as whatever comes back, so the daemon knows both what the
// member runs and whether it still has to type at it.
//
// A harness with no channel of its own is not a failure: it is the terminal
// one, which is how every agent was reached before a harness could deliver a
// mention itself. Its [Harness.Deliver] refuses, and the daemon does the
// waking.
package harness

import (
	"context"
	"errors"
	"fmt"
	"os"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// Notice is one thing the agent should be told. Text is the whole line it will
// see — the room's own wording, built by
// [github.com/ngicks/crabswarm/crabswarm/chat/nudge] so that a mention reads
// the same however it arrives. From is who wrote, for the deliverers whose
// channel carries a sender beside the text; it is empty where the notice is
// about the room rather than about one message.
type Notice struct {
	From string
	Text string
}

// Harness is one agent's CLI as the channel a mention takes to it.
type Harness interface {
	// Kind names the CLI, for the daemon to record and a roster to show.
	Kind() chatv1.Harness
	// Nudge is how a mention reaches this agent, which is what the daemon
	// reads: it types at a terminal one and leaves a native one to its own
	// server.
	Nudge() chatv1.NudgeDelivery
	// Deliver hands one notice to the agent. A harness with no channel of its
	// own refuses, since a caller that believed otherwise would leave the agent
	// waiting on a delivery nobody could make.
	Deliver(ctx context.Context, n Notice) error
}

// SinkEnv names a file every notice is appended to, one line each.
//
// It is how the end-to-end suite drives the delivery path: the real deliverers
// speak to a running Claude Code, Codex or OpenCode, none of which the suite can
// start, so without a sink the whole native half of a nudge would be exercised
// nowhere. It lives in production code for that reason, and setting it makes
// this member a native one whatever it runs.
const SinkEnv = "CRABSWARM_HARNESS_SINK"

// Detect maps the name a client gave itself in the MCP handshake onto the
// harness behind it, reading what that harness needs from getenv — the process
// environment the harness started the server in, [os.Getenv] outside a test.
//
// A name nothing recognises, the empty one included, is [chatv1.Harness]'s
// other: the server is serving something, and saying so is more use to whoever
// reads a roster than leaving the field blank.
func Detect(clientName string, getenv func(string) string) Harness {
	kind := kindOf(clientName)
	if getenv == nil {
		getenv = os.Getenv
	}
	if path := getenv(SinkEnv); path != "" {
		return sink{kind: kind, path: path}
	}
	if newNative := natives[kind]; newNative != nil {
		if h := newNative(getenv); h != nil {
			return h
		}
	}
	return terminal{kind: kind}
}

// natives is what builds the channel of each harness that has one, keyed by the
// harness it belongs to. A constructor answers nil when the environment does
// not carry what its channel needs — the harness was started without the
// variable that points at it — and the member then attends as a terminal one,
// which is always a working way to be woken.
//
// It is empty for now: every deliverer is a channel of its own, and each lands
// with the protocol work it needs.
var natives = map[chatv1.Harness]func(getenv func(string) string) Harness{}

// The names the harnesses give themselves, as they wrote them in the first
// frame of a real MCP session. e2e/crabswarm/testdata/harness holds those
// frames, and the suite checks these against them.
const (
	clientClaudeCode = "claude-code"
	clientCodex      = "codex-mcp-client"
	clientOpenCode   = "opencode"
)

// kindOf reads the harness off the name its client gave itself.
func kindOf(clientName string) chatv1.Harness {
	switch clientName {
	case clientClaudeCode:
		return chatv1.Harness_HARNESS_CLAUDE_CODE
	case clientCodex:
		return chatv1.Harness_HARNESS_CODEX
	case clientOpenCode:
		return chatv1.Harness_HARNESS_OPENCODE
	default:
		return chatv1.Harness_HARNESS_OTHER
	}
}

// terminal is a harness the server cannot deliver to itself, which is every
// harness whose channel is not wired up yet.
type terminal struct {
	kind chatv1.Harness
}

func (t terminal) Kind() chatv1.Harness { return t.kind }

func (terminal) Nudge() chatv1.NudgeDelivery {
	return chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL
}

func (terminal) Deliver(context.Context, Notice) error {
	return errNoChannel
}

// errNoChannel is what a terminal harness refuses a delivery with. A caller
// that asked is one that believed this member reached its agent itself, so the
// refusal says who does reach it instead.
var errNoChannel = errors.New(
	"this harness has no channel to deliver a mention through; " +
		"the daemon types at its terminal instead")

// sink is the deliverer that writes what it was handed to a file, for the tests
// that drive delivery end to end. See [SinkEnv].
type sink struct {
	kind chatv1.Harness
	path string
}

func (s sink) Kind() chatv1.Harness { return s.kind }

func (sink) Nudge() chatv1.NudgeDelivery {
	return chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE
}

// Deliver appends the notice as one line. Appending rather than rewriting: the
// file is the record of everything that reached the agent, and a reader wants
// the order as much as the last one.
func (s sink) Deliver(_ context.Context, n Notice) error {
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("opening the harness sink %s: %w", s.path, err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(n.Text + "\n"); err != nil {
		return fmt.Errorf("writing to the harness sink %s: %w", s.path, err)
	}
	return f.Close()
}
