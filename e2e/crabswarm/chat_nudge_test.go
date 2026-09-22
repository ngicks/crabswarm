package crabswarm_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	chatcli "github.com/ngicks/crabswarm/crabswarm/chat/cli"
	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// attendNative puts an agent in the room that delivers its own mentions, and
// holds that attendance until the test ends.
//
// It attends through the client the verbs dial with rather than through
// `crabswarm mcp`, which keeps the case below about what the daemon does with a
// declared delivery and nothing else: a bridge would bring its own deliverer
// along, and what is asserted here is who is typed at. The case that drives a
// bridge's own delivery is below it. Everything under this is the real thing —
// the daemon the other members talk to, over its own socket.
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
		chatBridgeAna + "  agent  done  other  terminal",
		chatBridgeBob + "  agent  done  other  terminal",
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

// chatNoticeSink is where a bridge writes what it delivered, one line per
// notice. The real channels talk to a running Claude Code, Codex or OpenCode,
// none of which this suite can start, so the sink is what makes the delivery
// path something a process-level test can watch.
func chatNoticeSink(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "notices.log")
}

// chatSinkNotices is what the bridge delivered, oldest first. A file that is not
// there is one nothing was delivered to.
func chatSinkNotices(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("reading the harness sink: %v", err)
	}
	return lines(string(b))
}

// waitChatSinkNotices blocks until the bridge has delivered exactly want, so a
// case pins both what reached the agent and that nothing else did.
func waitChatSinkNotices(t *testing.T, path string, want ...string) {
	t.Helper()
	var got []string
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		got = chatSinkNotices(t, path)
		if slices.Equal(got, want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("the harness was handed\n%q\nwant\n%q", got, want)
}

// A bridge whose harness carries a channel of its own attends as a member the
// daemon never types at, and hands that member its mentions itself — while the
// harness is idle, never mid-turn, and once the turn ends as a count of what
// piled up meanwhile.
func TestChat_ABridgeDeliversTheMentionsItsDaemonNoLongerTypes(t *testing.T) {
	cfg := startChatDaemon(t)
	sink := chatNoticeSink(t)
	// Appended after the scrubbed environment, which drops every CRABSWARM_
	// variable the suite is itself running with.
	startChatBridgeAs(t, cfg, "tok-ana", "claude-code",
		append(chatEnviron(), harnessctl.SinkEnv+"="+sink))
	attendChatBridges(t, cfg, "tok-bob")
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)

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

	// Mentioned while idle: the notice reaches it through its own harness, and
	// no keystroke is typed anywhere.
	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "the migration needs you")
	arrival := "[crabswarm chat] new message from " + chatBridgeBob +
		" — read it with the chat_read tool and respond with chat_send," +
		" both from crabswarm-mcp"
	waitChatSinkNotices(t, sink, arrival)
	// A nudge is typed while the send is being served, so by now there would be
	// one to read.
	if keys := stubSendKeys(t, cfg); keys != nil {
		t.Errorf("cmdman send-keys invocations = %q, want none", keys)
	}

	// Mentioned mid-turn: nothing is pushed at a harness that is working, and
	// the mention waits in the room where it already is.
	runChat(t, cfg, "tok-ana", "report-state", "working")
	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "and the rebase after it")

	// The turn ends and what waited is delivered as one line. Both messages are
	// still unread — a notice is not a read — so the count is two.
	runChat(t, cfg, "tok-ana", "report-state", "done")
	waitChatSinkNotices(t, sink, arrival,
		"[crabswarm chat] 2 unread messages mention you"+
			" — read them with the chat_read tool and respond with chat_send,"+
			" both from crabswarm-mcp")
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
