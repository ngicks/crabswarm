package crabswarm_test

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ngicks/crabswarm/crabswarm/mcp/harness"
)

// claudeChannelMethod carries one channel event. It is spelled here rather than
// imported because what this file is about is the wire: a rename that the
// harness package and this test made together would be one nothing caught.
const claudeChannelMethod = "notifications/claude/channel"

// channelWatcher wraps the bridge's transport so the channel events it pushes
// can be read by the test.
//
// They are taken off the stream rather than passed along: the MCP client has no
// vocabulary for a method outside the specification, and one handed to it would
// be refused as unknown. What Claude Code does with an event is its own
// business, and none of it happens here — the bridge's side of the contract is
// the frame, which is all this reads.
type channelWatcher struct {
	inner  mcp.Transport
	events chan []byte
}

func (w *channelWatcher) Connect(ctx context.Context) (mcp.Connection, error) {
	conn, err := w.inner.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &channelConn{Connection: conn, events: w.events}, nil
}

type channelConn struct {
	mcp.Connection
	events chan []byte
}

// Read hands the client everything but the channel events, which go to the test
// as the bytes they arrived as.
func (c *channelConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	for {
		msg, err := c.Connection.Read(ctx)
		if err != nil {
			return nil, err
		}
		req, ok := msg.(*jsonrpc.Request)
		if !ok || req.Method != claudeChannelMethod {
			return msg, nil
		}
		frame, err := jsonrpc.EncodeMessage(req)
		if err != nil {
			return nil, err
		}
		select {
		case c.events <- frame:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// startChannelBridge starts `crabswarm mcp` the way a Claude Code launched as a
// channel host starts it: named as Claude Code in the handshake, and carrying
// the variable the launcher sets beside the flag that registers the channel.
//
// It answers with the session and the events the bridge pushed, oldest first.
func startChannelBridge(
	t *testing.T, cfgPath, token string,
) (*mcp.ClientSession, chan []byte) {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), crabswarmBin,
		"mcp", "--config", cfgPath, "--token", token)
	// Appended after the scrubbed environment, which drops every CRABSWARM_
	// variable the suite is itself running with.
	cmd.Env = append(chatEnviron(), harness.ClaudeChannelEnv+"=1")
	cmd.Stderr = os.Stderr

	// Buffered well past what one case pushes, so the bridge never waits on a
	// test that has not looked yet.
	events := make(chan []byte, 16)
	client := mcp.NewClient(&mcp.Implementation{Name: "claude-code", Version: "v0"}, nil)
	session, err := client.Connect(t.Context(),
		&channelWatcher{inner: &mcp.CommandTransport{Command: cmd}, events: events}, nil)
	if err != nil {
		t.Fatalf("connect to the channel bridge for %q: %v", token, err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session, events
}

// nextChannelEvent waits for one channel event and returns its content and meta.
func nextChannelEvent(t *testing.T, events chan []byte) (content string, meta map[string]string) {
	t.Helper()

	select {
	case frame := <-events:
		return channelEventFields(t, frame)
	case <-time.After(30 * time.Second):
		t.Fatal("the bridge pushed no channel event")
		return "", nil
	}
}

// channelEventFields reads one channel frame the way the harness on the other
// end does: off the wire, as JSON.
func channelEventFields(t *testing.T, frame []byte) (content string, meta map[string]string) {
	t.Helper()

	var event struct {
		Params struct {
			Content string            `json:"content"`
			Meta    map[string]string `json:"meta"`
		} `json:"params"`
	}
	if err := json.Unmarshal(frame, &event); err != nil {
		t.Fatalf("reading the channel event %s: %v", frame, err)
	}
	return event.Params.Content, event.Params.Meta
}

// assertNoChannelEvent fails when anything was pushed, having given the bridge
// time to push it.
func assertNoChannelEvent(t *testing.T, events chan []byte, why string) {
	t.Helper()

	select {
	case frame := <-events:
		t.Fatalf("%s, yet the bridge pushed %s", why, frame)
	case <-time.After(2 * time.Second):
	}
}

// frameKeys is the set of keys one JSON object holds, sorted.
func frameKeys(t *testing.T, raw json.RawMessage) []string {
	t.Helper()

	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatalf("reading %s as an object: %v", raw, err)
	}
	return slices.Sorted(maps.Keys(object))
}

// A Claude Code the launcher registered this server as a channel of attends as
// a member the daemon never types at, and is handed its mentions as channel
// events of the session it already holds — while it is idle, never mid-turn,
// and once the turn ends as a count of what piled up meanwhile.
func TestChat_TheClaudeChannelTakesTheMentionsItsDaemonNoLongerTypes(t *testing.T) {
	cfg := startChatDaemon(t)
	session, events := startChannelBridge(t, cfg, "tok-ana")
	attendChatBridges(t, cfg, "tok-bob")
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)

	// The handshake is what registers the listener: without the capability
	// Claude Code never looks, and without the instructions the model reads a
	// turn that appeared from nowhere.
	res := session.InitializeResult()
	if res.Capabilities == nil {
		t.Fatal("the bridge declared no capabilities at all")
	}
	if _, declared := res.Capabilities.Experimental[harness.ClaudeChannelCapability]; !declared {
		t.Errorf("the handshake declares no %q capability: %+v",
			harness.ClaudeChannelCapability, res.Capabilities.Experimental)
	}
	for _, want := range []string{"crabswarm-mcp", "chat_read", "chat_send"} {
		if !strings.Contains(res.Instructions, want) {
			t.Errorf("the instructions do not name %q:\n%s", want, res.Instructions)
		}
	}

	// The daemon has it from the handshake: which CLI the member runs, and that
	// its mentions are its own server's to deliver.
	roster := lines(runChat(t, cfg, "tok-bob", "members"))
	slices.Sort(roster)
	want := []string{
		chatBridgeAna + "  agent  done  claude-code  native",
		chatBridgeBob + "  agent  done  other  terminal",
	}
	if !slices.Equal(roster, want) {
		t.Errorf("members = %v, want %v", roster, want)
	}

	// Mentioned while idle: the notice reaches it as a channel event naming who
	// wrote and which room, and no keystroke is typed anywhere.
	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "the migration needs you")
	arrival := "[crabswarm chat] new message from " + chatBridgeBob +
		" — read it with the chat_read tool"
	content, meta := nextChannelEvent(t, events)
	if content != arrival {
		t.Errorf("the channel event carried %q, want %q", content, arrival)
	}
	if got := map[string]string{"from": chatBridgeBob, "room": chatRoom}; !maps.Equal(meta, got) {
		t.Errorf("the channel event's meta = %v, want %v", meta, got)
	}
	// A nudge is typed while the send is being served, so by now there would be
	// one to read.
	if keys := stubSendKeys(t, cfg); keys != nil {
		t.Errorf("cmdman send-keys invocations = %q, want none", keys)
	}

	// Mentioned mid-turn: nothing is pushed at a harness that is working, and
	// the mention waits in the room where it already is.
	runChat(t, cfg, "tok-ana", "report-state", "working")
	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "and the rebase after it")
	assertNoChannelEvent(t, events, "the harness is mid-turn")

	// The turn ends and what waited is delivered as one line. Both messages are
	// still unread — an event is not a read — so the count is two.
	runChat(t, cfg, "tok-ana", "report-state", "done")
	content, meta = nextChannelEvent(t, events)
	waiting := "[crabswarm chat] 2 unread messages mention you — read them with the chat_read tool"
	if content != waiting {
		t.Errorf("the channel event carried %q, want %q", content, waiting)
	}
	// A count names no sender, so the key is left out rather than sent empty.
	if got := map[string]string{"room": chatRoom}; !maps.Equal(meta, got) {
		t.Errorf("the channel event's meta = %v, want %v", meta, got)
	}
	if keys := stubSendKeys(t, cfg); keys != nil {
		t.Errorf("cmdman send-keys invocations = %q, want none", keys)
	}

	// Delivered is not read: both messages are waiting where the agent's own
	// read finds them.
	read := runChat(t, cfg, "tok-ana", "read")
	for _, want := range []string{"the migration needs you", "and the rebase after it"} {
		if !strings.Contains(read, want) {
			t.Errorf("read as %s = %q, want it to carry %q", chatBridgeAna, read, want)
		}
	}
}

// What the bridge writes has the shape of the frame a working channel server
// wrote into a registered Claude Code session: the same method, the same
// top-level keys, and the same params — a content and a meta whose keys that
// recording already carried.
func TestChat_TheClaudeChannelWritesTheRecordedFrame(t *testing.T) {
	cfg := startChatDaemon(t)
	_, events := startChannelBridge(t, cfg, "tok-ana")
	attendChatBridges(t, cfg, "tok-bob")
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)

	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "the migration needs you")
	var written []byte
	select {
	case written = <-events:
	case <-time.After(30 * time.Second):
		t.Fatal("the bridge pushed no channel event")
	}

	recorded := harnessFixtureBytes(t, "claude-channel-notification.jsonl")
	recorded = []byte(strings.TrimSpace(string(recorded)))

	type frame struct {
		Jsonrpc string          `json:"jsonrpc"`
		Id      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}
	var got, want frame
	if err := json.Unmarshal(written, &got); err != nil {
		t.Fatalf("reading what the bridge wrote: %v", err)
	}
	if err := json.Unmarshal(recorded, &want); err != nil {
		t.Fatalf("reading the recorded frame: %v", err)
	}

	if got.Jsonrpc != want.Jsonrpc || got.Method != want.Method {
		t.Errorf("the bridge wrote %s %s, want %s %s",
			got.Jsonrpc, got.Method, want.Jsonrpc, want.Method)
	}
	// A notification carries no id, which is what says the harness answers it
	// with a turn rather than with a response.
	if got.Id != nil {
		t.Errorf("the bridge wrote an id of %s, so it asked for an answer", got.Id)
	}
	if k, w := frameKeys(t, written), frameKeys(t, recorded); !slices.Equal(k, w) {
		t.Errorf("the bridge wrote the keys %v, want %v", k, w)
	}
	if k, w := frameKeys(t, got.Params), frameKeys(t, want.Params); !slices.Equal(k, w) {
		t.Errorf("the bridge wrote the params keys %v, want %v", k, w)
	}

	_, gotMeta := channelEventFields(t, written)
	_, wantMeta := channelEventFields(t, recorded)
	for key := range wantMeta {
		if _, ok := gotMeta[key]; !ok {
			t.Errorf("the bridge wrote the meta keys %v, which lack the recorded %q",
				slices.Sorted(maps.Keys(gotMeta)), key)
		}
	}
}
