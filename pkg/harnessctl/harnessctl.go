// Package harnessctl reaches a coding-agent harness from beside it: it
// recognises which CLI an MCP server is serving and hands back the channel a
// message reaches that agent through, and, where the harness has one, the feed
// its state is read from. It knows nothing about chat rooms; the crabswarm MCP
// server composes it with the room.
//
// A harness names itself once, in the MCP handshake, and everything about how
// it can be woken and how it is heard from follows from that name: Claude Code
// takes an upload on the loopback server its fakechat plugin runs and says what
// it is doing in the agents listing, Codex listens on its app server and
// reports on that server's feed, OpenCode relays through a plugin, and a
// harness nobody recognises takes nothing at all. The server asks here and
// attends as whatever comes back, so the daemon knows both what the member runs
// and whether it still has to type at it.
//
// A harness with no channel of its own is not a failure: it is the terminal
// one, which is how every agent was reached before a harness could deliver a
// mention itself. Its [Harness.Deliver] refuses, and the daemon does the
// waking.
//
// A harness that also implements [Prober] has a channel that can be asked
// whether it is there, and the server asks before the member attends and again
// for as long as it does, so a channel that was declared and never listened is
// a member that never appears, and one that stops listening is a member that
// leaves.
//
// A harness that also implements [StateSource] is where its member's state
// comes from. The server watches it for as long as the session runs and
// reports what it says, so nothing about that member's turns depends on a hook
// firing.
package harnessctl

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
// about the room rather than about one message. Room is the room this member
// attends, for the channels that label an event with where it came from — an
// agent can be looking at more than one crabswarm at a time.
type Notice struct {
	From string
	Room string
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

// Prober is a harness whose channel can be asked whether it is there before the
// member attends, which is a harness whose channel is something other than the
// session this server already holds.
//
// A channel built out of launch-time variables is a claim and nothing more: the
// variables say the launcher meant to wire one up, and no part of the session
// says it worked. A member that attends on a broken claim attends as native, so
// the daemon stops typing at it, and every mention it is then handed is
// dropped.
//
// The server is what joins the two ends: it probes a harness that implements
// this and refuses to attend while Probe fails, and asks again for as long as
// the member attends, so a channel nothing listens on is a member nobody sees
// rather than one nobody can reach — whether it never came up or went away
// mid-session. See [Harness] for the other half of what a harness does.
type Prober interface {
	// Probe answers nil while the channel is reachable, and otherwise says
	// which channel was looked for and where.
	//
	// An implementation bounds its own attempt. The context it is handed is the
	// server's, which lives as long as the session, so a channel that took the
	// connection and then said nothing would otherwise hold up the attendance
	// this gates for the rest of that session.
	Probe(ctx context.Context) error
}

// StateSource is a harness that says what its agent is doing without being
// asked, which is a harness whose CLI has a feed of its own to say it on.
//
// Claude Code has the agents listing and Codex has its app server, so both
// implement this. A harness with no feed says nothing here, and the daemon
// reads that member's terminal instead. A feed is preferred over a hook for
// the state: a hook only fires on the events its harness announces, so an
// interrupted turn would leave a member marked working with nothing to correct
// it.
//
// The server is what joins the two ends: it holds one harness per session, and
// a harness that implements this gets watched for as long as that session runs,
// with every state it reports handed to the daemon as this member's own. See
// [Harness] for the other half of what a harness does.
type StateSource interface {
	// Watch follows the harness until ctx is done, handing each state it
	// reports to report. It blocks, and it does not give up: a feed that
	// dropped is opened again, since an agent nobody hears about is one the
	// room stops delivering to.
	Watch(ctx context.Context, report func(chatv1.HarnessState))
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
// which is always a working way to be woken. Each constructor lives in the
// file of its harness.
var natives = map[chatv1.Harness]func(getenv func(string) string) Harness{
	chatv1.Harness_HARNESS_CLAUDE_CODE: newClaudeCode,
	chatv1.Harness_HARNESS_CODEX:       newCodex,
	chatv1.Harness_HARNESS_OPENCODE:    newOpenCode,
}

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
