package crabswarm_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	chatcli "github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// attendNative puts an agent in the room that delivers its own mentions, and
// holds that attendance until the test ends.
//
// It attends through the client the verbs dial with rather than through
// `crabswarm mcp`, because the bridge declares the terminal delivery for every
// session it opens: reading the harness off the MCP handshake and pushing a
// mention through it is work the bridge does not do yet, so there is no process
// the suite could start that would attend this way. Everything under it is the
// real thing — the daemon the other members talk to, over its own socket.
func attendNative(t *testing.T, cfgPath, token string) *chatcli.Attendance {
	t.Helper()

	client, err := chatcli.Dial(chatSock(cfgPath))
	if err != nil {
		t.Fatalf("dial the daemon as %q: %v", token, err)
	}
	t.Cleanup(func() { _ = client.Close() })

	// The attendance lasts as long as this context, so it is cancelled on the
	// way out rather than left holding a role the daemon would keep listing.
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	// The empty name takes the one the daemon derives from the token, which is
	// what a bridge passes too — so the role reads the same either way.
	attendance, err := client.Attend(ctx, token, "",
		chatv1.MemberKind_MEMBER_KIND_AGENT,
		chatv1.Harness_HARNESS_CLAUDE_CODE,
		chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)
	if err != nil {
		t.Fatalf("attend as %q: %v", token, err)
	}
	return attendance
}

// An agent whose own server delivers its mentions is never typed at, while one
// that is reached through its terminal still is. Both are in the room, both are
// mentioned, and only one of them costs a keystroke — which is the whole of what
// the declared delivery decides.
func TestChat_ANativeAgentIsMentionedButNeverTypedAt(t *testing.T) {
	cfg := startChatDaemon(t)
	attendChatBridges(t, cfg, "tok-ana", "tok-bob")
	attendNative(t, cfg, "tok-cid")
	waitChatRosterHas(t, cfg, "tok-ana", chatBridgeBob, 30*time.Second)
	waitChatRosterHas(t, cfg, "tok-ana", chatBridgeCid, 30*time.Second)

	// The roster says which of the two is typed at, so an operator wondering why
	// a member never woke can read the answer rather than infer it.
	roster := lines(runChat(t, cfg, "tok-ana", "members"))
	slices.Sort(roster)
	want := []string{
		chatBridgeAna + "  agent  done  -  terminal",
		chatBridgeBob + "  agent  done  -  terminal",
		chatBridgeCid + "  agent  done  claude-code  native",
	}
	if !slices.Equal(roster, want) {
		t.Errorf("members = %v, want %v", roster, want)
	}

	// Named together: both are mentioned, and only the terminal one is typed at.
	got := runChat(t, cfg, "tok-ana", "send", chatBridgeBob+","+chatBridgeCid, "you two own it")
	if want := "mentioned " + chatBridgeBob + "\nmentioned " + chatBridgeCid + "\n"; got != want {
		t.Errorf("send = %q, want %q", got, want)
	}
	typed := chatNudgeKeys("tok-bob", chatBridgeAna)
	if keys := stubSendKeys(t, cfg); !slices.Equal(keys, typed) {
		t.Errorf("cmdman send-keys invocations =\n%q\nwant\n%q", keys, typed)
	}

	// A message to the whole room walks the other list and reaches the same
	// conclusion.
	if out := runChat(t, cfg, "tok-ana", "send", "everyone", "standup in five"); out != "" {
		t.Errorf("send to everyone = %q, want nothing", out)
	}
	typed = append(typed, chatNudgeKeys("tok-bob", chatBridgeAna)...)
	if keys := stubSendKeys(t, cfg); !slices.Equal(keys, typed) {
		t.Errorf("cmdman send-keys invocations =\n%q\nwant\n%q", keys, typed)
	}

	// Not typed at is not undelivered: both messages are waiting to be read,
	// which is what the member's own server hands it.
	read := runChat(t, cfg, "tok-cid", "read")
	for _, want := range []string{"you two own it", "standup in five"} {
		if !strings.Contains(read, want) {
			t.Errorf("read as %s = %q, want it to carry %q", chatBridgeCid, read, want)
		}
	}
}
