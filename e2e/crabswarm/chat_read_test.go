package crabswarm_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// `crabswarm chat read --quiet --done-when-empty` is the read the OpenCode TUI
// plugin runs each time a turn ends. What it prints becomes the session's next
// prompt, an empty output means the turn really ended, and a failed read means
// nothing is known. The cases below run the verb the way the plugin runs it,
// against a real daemon, so those three outcomes stay apart.

// chatSentText is what a teammate sends in the delivery cases. The double quote
// and the line break are the point: they are what a hand-rolled encoder gets
// wrong, and a mangled hand-over is a message the agent never sees.
const chatSentText = "say \"hi\"\nand a second line"

// chatTurnEndRead is the argument list the OpenCode TUI plugin runs
// `crabswarm chat` with when a turn ends (atIdle in opencode-tui.ts).
var chatTurnEndRead = []string{"read", "--quiet", "--done-when-empty"}

// chatAbsentDaemonConfig names a socket nothing listens on, which is how a case
// stands a client up against a daemon that is not running.
func chatAbsentDaemonConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	writeFile(t, cfgPath, fmt.Sprintf(`{"sock":%q}`, filepath.Join(dir, "absent.sock")))
	return cfgPath
}

// startChatRoomWithMail brings up a daemon, attends as ana and bob, and leaves
// one message from bob unread for ana. It returns the config path; every case
// reads as ana, the member whose turn is ending.
func startChatRoomWithMail(t *testing.T) string {
	t.Helper()
	cfg := startChatDaemon(t)
	attendChatBridges(t, cfg, "tok-ana", "tok-bob")
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)
	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, chatSentText)
	return cfg
}

// chatMailLine is how the message [startChatRoomWithMail] leaves reads once it
// is handed over: the sender, the role it named and the mention marker, with
// the sequence number and the stamp in front of them left out because neither
// is the same twice.
const chatMailLine = chatBridgeBob + " -> " + chatBridgeAna + ` [mentioned you]: say "hi"`

// chatMemberState is the state the room lists address in, read as ana. A
// report rides on the read's own RPC, and the daemon stores it before that RPC
// returns, so one listing after the verb exits already shows it.
func chatMemberState(t *testing.T, cfgPath, address string) string {
	t.Helper()
	listed := runChat(t, cfgPath, "tok-ana", "members")
	for _, l := range lines(listed) {
		fields := strings.Fields(l)
		if len(fields) > 2 && fields[0] == address {
			return fields[2]
		}
	}
	t.Fatalf("the room does not list %s:\n%s", address, listed)
	return ""
}

// Messages in hand when a turn ends: the read prints them and marks them read,
// and it leaves the member's state alone, because the prompt the plugin makes
// of them continues the turn. The read at the end of that continued turn finds
// nothing, and that read is the one that reports the member done.
func TestChatRead_TurnEndReadHandsTheMessagesOver(t *testing.T) {
	cfg := startChatRoomWithMail(t)
	runChat(t, cfg, "tok-ana", "report-state", "working")

	got := runChat(t, cfg, "tok-ana", chatTurnEndRead...)
	for _, want := range []string{chatMailLine, "and a second line"} {
		if !strings.Contains(got, want) {
			t.Errorf("read = %q, want it to carry %q", got, want)
		}
	}
	if state := chatMemberState(t, cfg, chatBridgeAna); state != "working" {
		t.Errorf("state after the read handed messages over = %q, want %q", state, "working")
	}

	if got := runChat(t, cfg, "tok-ana", chatTurnEndRead...); got != "" {
		t.Errorf("the next read = %q, want nothing: the first one consumed the message", got)
	}
	if state := chatMemberState(t, cfg, chatBridgeAna); state != "done" {
		t.Errorf("state after the empty read = %q, want %q", state, "done")
	}
}

// Nothing to hand over: the read prints nothing at all, which the plugin takes
// as the end of the turn, and it reports the member done on the way.
//
// A board post is the second way a room has nothing for this member: it is in
// the log and mentions nobody, so the read finds nothing to hand over and the
// turn ends. It is worth its own case because a naive implementation gets it
// wrong: the room is not quiet, the message simply is not addressed to anyone.
func TestChatRead_TurnEndReadIsSilentWithNothingToHandOver(t *testing.T) {
	for _, tc := range []struct {
		name string
		post string
	}{
		{"the room said nothing at all", ""},
		{"the room posted to its board", "fyi: rebased main"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := startChatDaemon(t)
			attendChatBridges(t, cfg, "tok-ana", "tok-bob")
			waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)
			runChat(t, cfg, "tok-ana", "report-state", "working")
			if tc.post != "" {
				runChat(t, cfg, "tok-bob", "send", "", tc.post)
			}

			stdout, stderr, err := execChat(t, cfg, "tok-ana", chatTurnEndRead...)
			if err != nil {
				t.Fatalf("read: %v\nstderr:\n%s", err, stderr)
			}
			if stdout != "" {
				t.Errorf(
					"stdout = %q, want nothing: an empty output is the end of the turn",
					stdout,
				)
			}
			if stderr != "" {
				t.Errorf("stderr = %q, want nothing", stderr)
			}
			if state := chatMemberState(t, cfg, chatBridgeAna); state != "done" {
				t.Errorf("state after the empty read = %q, want %q", state, "done")
			}
		})
	}
}

// chatMailSeq reads the sequence number of the message [startChatRoomWithMail]
// leaves, as bob: the tail read moves bob's position and leaves ana's alone.
func chatMailSeq(t *testing.T, cfgPath string) string {
	t.Helper()
	out := runChat(t, cfgPath, "tok-bob", "read", "--cursor", "tail", "--range", "-1")
	for _, l := range lines(out) {
		if strings.Contains(l, chatBridgeBob+" -> "+chatBridgeAna) {
			return strings.Fields(l)[0]
		}
	}
	t.Fatalf("bob's tail read does not show the mail:\n%s", out)
	return ""
}

// A client that already showed ana her mention another way marks it read by
// its sequence number. The skip prints what is left, a repeat of it changes
// nothing, and the next read hands over only what came after.
func TestChatRead_SkipMarksTheMentionRead(t *testing.T) {
	cfg := startChatRoomWithMail(t)
	seq := chatMailSeq(t, cfg)
	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "a later one")

	for range 2 {
		if got := runChat(t, cfg, "tok-ana", "read", "--skip", seq); got != "1 more unread\n" {
			t.Errorf("read --skip %s = %q, want %q", seq, got, "1 more unread\n")
		}
	}

	got := runChat(t, cfg, "tok-ana", "read")
	if strings.Contains(got, chatMailLine) {
		t.Errorf("read after the skip = %q, want the skipped mail left out", got)
	}
	if !strings.Contains(got, "a later one") {
		t.Errorf("read after the skip = %q, want the later message", got)
	}

	// A seq past the newest stops at the newest, and --quiet drops the line.
	if got := runChat(t, cfg, "tok-ana", "read", "--skip", "1000"); got != "no more unread\n" {
		t.Errorf("read --skip 1000 = %q, want %q", got, "no more unread\n")
	}
	if got := runChat(t, cfg, "tok-ana", "read", "--skip", "1000", "--quiet"); got != "" {
		t.Errorf("read --skip 1000 --quiet = %q, want nothing", got)
	}
}

// A skip beside a flag that shapes a shown read is refused before it reaches
// the daemon, so the mail it would have marked is still there to read.
func TestChatRead_SkipRefusesTheReadFlags(t *testing.T) {
	cfg := startChatRoomWithMail(t)

	for _, extra := range [][]string{
		{"--cursor", "tail"},
		{"--range", "5"},
		{"--to", "everyone"},
		{"--since", "1"},
		{"--until", "9"},
		{"--done-when-empty"},
	} {
		args := append([]string{"read", "--skip", "1000"}, extra...)
		stdout, stderr, err := execChat(t, cfg, "tok-ana", args...)
		if err == nil {
			t.Errorf("read %s succeeded; want a refusal", strings.Join(args, " "))
		}
		if stdout != "" {
			t.Errorf("read %s stdout = %q, want nothing", strings.Join(args, " "), stdout)
		}
		if !strings.Contains(stderr, extra[0]) {
			t.Errorf("read %s stderr = %q, want it to name %s",
				strings.Join(args, " "), stderr, extra[0])
		}
	}

	if got := runChat(t, cfg, "tok-ana", "read"); !strings.Contains(got, chatMailLine) {
		t.Errorf("read after the refusals = %q, want the mail still unread", got)
	}
}

// No daemon to ask: the read fails, and says why on stderr alone. The plugin
// tells a failed read from an empty one by the exit status, so a chat nobody
// is hosting never passes for a turn with nothing waiting, and the hint never
// reaches the session as a message nobody sent.
func TestChatRead_TurnEndReadFailsQuietlyWithoutADaemon(t *testing.T) {
	cfg := chatAbsentDaemonConfig(t)

	stdout, stderr, err := execChat(t, cfg, "tok-ana", chatTurnEndRead...)
	if err == nil {
		t.Error("the read succeeded against no daemon; want a failure the plugin can tell apart")
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want nothing: the plugin would prompt the session with it", stdout)
	}
	if !strings.Contains(stderr, "crabswarm serve") {
		t.Errorf("stderr = %q, want the hint that starts the daemon", stderr)
	}
}
