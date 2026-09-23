package crabswarm_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	chatcli "github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// screenPollInterval is how often the daemon under these cases reads a screen.
// Short enough that a case walks a whole sequence in well under a second, and
// still far longer than one stub invocation takes.
const screenPollInterval = 200 * time.Millisecond

// screenPollScript is the order the stub cmdman hands a member's screens back
// in: at its prompt, then mid-turn, then holding a permission dialog, then back
// at its prompt. The last one is held once the script runs out, the way a
// terminal keeps showing its last screen.
//
// They are Claude Code recordings although the member walking them attends as
// the harness nobody recognised: the markers the daemon reads a screen with
// were read off those recordings, and a harness with no name has no recording
// of its own to stand in for it.
var screenPollScript = []string{
	"claude-screen-idle.txt",
	"claude-screen-working.txt",
	"claude-screen-dialog.txt",
	"claude-screen-idle.txt",
}

// startScreenPollDaemon writes a config naming a private socket, database and
// stub cmdman together with the screen-poll interval, plants the recorded
// screens the stub answers `capture-screen` from, starts `crabswarm serve` on
// it and returns the config path.
//
// It is not [startChatDaemonWith] with an argument added: the poll interval is
// the only thing these cases configure that no other case does, and the stub
// here answers `capture-screen` from a script rather than with one fixed idle
// prompt.
func startScreenPollDaemon(t *testing.T, commands []stubCommand) string {
	t.Helper()

	dir := t.TempDir()
	stub := filepath.Join(dir, "cmdman")
	writeFile(t, stub, screenPollStubScript(commands))
	if err := os.Chmod(stub, 0o755); err != nil {
		t.Fatalf("chmod stub cmdman: %v", err)
	}
	for i, fixture := range screenPollScript {
		writeFile(t, filepath.Join(dir, fmt.Sprintf("screen-%d.txt", i+1)),
			harnessFixture(t, fixture))
	}

	cfgPath := filepath.Join(dir, "config.json")
	writeFile(t, cfgPath, fmt.Sprintf(
		`{"sock":%q,"chat":{"db":%q,"cmdman_bin":%q,"screen_poll_interval":%d}}`,
		chatSock(cfgPath), filepath.Join(dir, "chat.db"), stub,
		screenPollInterval.Nanoseconds()))
	startChatServe(t, cfgPath)
	return cfgPath
}

// harnessFixture reads one recorded harness screen out of the suite's testdata.
func harnessFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "harness", name))
	if err != nil {
		t.Fatalf("reading the harness fixture %q: %v", name, err)
	}
	return string(b)
}

// screenPollStubScript renders a stub cmdman whose `capture-screen` walks the
// planted screens, one per capture per token, and holds the last once it runs
// out. Every capture is recorded beside the stub, so a case can read back whose
// screen the daemon asked for — and whose it never did.
//
// The screens are read from files rather than written into the script: they
// carry quotes and box-drawing characters, and a shell that had to assemble
// them would be quoting them rather than reproducing them.
func screenPollStubScript(commands []stubCommand) string {
	var b strings.Builder
	b.WriteString(`#!/bin/sh
d=$(dirname "$0")
if [ "$1" = "status" ]; then
	shift
	printf '%s\n' "$*" >> "$d/status.log"
	exit 0
fi
if [ "$1" = "send-keys" ]; then
	shift
	printf '%s\n' "$*" >> "$d/send-keys.log"
	exit 0
fi
if [ "$1" = "capture-screen" ]; then
	printf '%s\n' "$2" >> "$d/capture-screen.log"
	n=0
	if [ -f "$d/taken-$2" ]; then n=$(cat "$d/taken-$2"); fi
	next=$((n + 1))
	if [ ! -f "$d/screen-$next.txt" ]; then next=$n; fi
	printf '%s\n' "$next" > "$d/taken-$2"
	cat "$d/screen-$next.txt"
	exit 0
fi
if [ "$1" != "inspect" ]; then
	echo "stub cmdman: unsupported invocation: $*" >&2
	exit 1
fi
case "$2" in
`)
	for _, c := range commands {
		fmt.Fprintf(&b, "\t%s) state=%s; dir=%s; labels='%s' ;;\n",
			c.token, stubState(c), c.dir, stubLabels(c))
	}
	b.WriteString(`	*)
		echo "error: resolve command: no command found matching \"$2\"" >&2
		exit 1
		;;
esac
printf '%s {"dir":"%s","labels":%s}\n' "$state" "$dir" "$labels"
`)
	return b.String()
}

// stubCaptures returns the tokens the daemon asked for a screen of, oldest
// first. No log file means it never asked for one at all.
func stubCaptures(t *testing.T, cfgPath string) []string {
	t.Helper()
	return stubLog(t, cfgPath, "capture-screen.log")
}

// attendScreenPoll puts an agent running harness in the room and holds that
// attendance until the test ends.
//
// It attends through the client the verbs dial with rather than through
// `crabswarm mcp`: what the poller reads is the terminal, not the bridge, and
// the bridge would only stand between the case and the harness it declares.
func attendScreenPoll(
	t *testing.T, cfgPath, token string, harness chatv1.Harness,
) *chatcli.Attendance {
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

	attendance, err := client.Attend(ctx, token, "",
		chatv1.MemberKind_MEMBER_KIND_AGENT, harness,
		chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL)
	if err != nil {
		t.Fatalf("attend as %q: %v", token, err)
	}
	return attendance
}

// screenPollStates collects the state changes the room announced, in order,
// until want of them have arrived.
//
// The feed rather than the roster: a state the poller holds for one interval is
// one a case polling `chat members` can miss entirely, while the feed carries
// every change the daemon published.
func screenPollStates(
	t *testing.T, a *chatcli.Attendance, want int, timeout time.Duration,
) []string {
	t.Helper()

	// Closed by the reader when the stream ends, so a feed that went away reads
	// as that rather than as a case that simply ran out of time. The sends are
	// guarded by the test's own context, which ends the reader with the test
	// instead of leaving it blocked on a channel nobody drains any more.
	ctx := t.Context()
	changes := make(chan string, 64)
	go func() {
		defer close(changes)
		for {
			ev, err := a.Recv()
			if err != nil {
				return
			}
			changed := ev.GetMemberStateChanged()
			if changed == nil {
				continue
			}
			select {
			case changes <- changed.GetState().String():
			case <-ctx.Done():
				return
			}
		}
	}()

	var states []string
	deadline := time.After(timeout)
	for len(states) < want {
		select {
		case s, ok := <-changes:
			if !ok {
				t.Fatalf("the attendance ended after %d of %d state changes: %v",
					len(states), want, states)
			}
			states = append(states, s)
		case <-deadline:
			t.Fatalf("only %d of %d state changes arrived within %s: %v",
				len(states), want, timeout, states)
		}
	}
	return states
}

// memberRow returns the `chat members` row for address, split into its columns.
func memberRow(t *testing.T, cfgPath, token, address string) []string {
	t.Helper()
	rendered := runChat(t, cfgPath, token, "members")
	for _, l := range lines(rendered) {
		fields := strings.Fields(l)
		if len(fields) > 0 && fields[0] == address {
			return fields
		}
	}
	t.Fatalf("members = %q, want a row for %s", rendered, address)
	return nil
}

// A session attending as the harness nobody recognised reports nothing here —
// it has no feed of its own, and nothing calls report-state — and the daemon
// still follows it from its prompt into a turn, into the permission dialog that
// turn raised, and back to its prompt. Reading the screen is the whole of how
// it knows.
func TestChat_ScreenPollFollowsAHarnessWithNoFeed(t *testing.T) {
	cfg := startScreenPollDaemon(t, []stubCommand{
		{token: "tok-ana", dir: chatRoom, project: "alpha"},
	})
	attendance := attendScreenPoll(t, cfg, "tok-ana", chatv1.Harness_HARNESS_OTHER)

	// Attendance opens at done, so the first screen — the session at its prompt
	// — changes nothing and announces nothing. What follows is the sequence.
	self := attendance.Self()
	if self.GetState() != chatv1.HarnessState_HARNESS_STATE_DONE {
		t.Fatalf("attended as %s, want %s",
			self.GetState(), chatv1.HarnessState_HARNESS_STATE_DONE)
	}
	address := self.GetTeam() + "/" + self.GetName()

	got := screenPollStates(t, attendance, 3, 30*time.Second)
	want := []string{
		"HARNESS_STATE_WORKING",
		"HARNESS_STATE_WAITING",
		"HARNESS_STATE_DONE",
	}
	if !slices.Equal(got, want) {
		t.Errorf("announced states = %v, want %v", got, want)
	}

	// The roster the other members read says the same as the feed.
	row := memberRow(t, cfg, "tok-ana", address)
	if len(row) < 3 || row[2] != "done" {
		t.Errorf("members row = %v, want its state column to read done", row)
	}

	// Nothing was typed at the session: reading a screen is not nudging one.
	if keys := stubSendKeys(t, cfg); len(keys) != 0 {
		t.Errorf("cmdman send-keys invocations = %q, want none", keys)
	}
}

// A harness the daemon has a name for is left to the feed it reports on.
// Reading its screen beside that feed would only lay a wrong "done" over what
// the feed said: the composer line stands empty while a subagent running in the
// background keeps the session working.
func TestChat_ScreenPollLeavesHarnessesWithAFeedAlone(t *testing.T) {
	cfg := startScreenPollDaemon(t, []stubCommand{
		{token: "tok-ana", dir: chatRoom, project: "alpha"},
		{token: "tok-bob", dir: chatRoom, project: "alpha"},
		{token: "tok-cid", dir: chatRoom, project: "alpha"},
	})
	// The two harnesses with a feed attend first, so every sweep that read the
	// unrecognised member's screen had both of them on the roster already.
	claude := attendScreenPoll(t, cfg, "tok-bob", chatv1.Harness_HARNESS_CLAUDE_CODE)
	codex := attendScreenPoll(t, cfg, "tok-cid", chatv1.Harness_HARNESS_CODEX)
	other := attendScreenPoll(t, cfg, "tok-ana", chatv1.Harness_HARNESS_OTHER)

	// The unrecognised member walking its whole script is what says the poller
	// has been round several times by now, so the other two had not simply gone
	// unread so far.
	screenPollStates(t, other, 3, 30*time.Second)

	captured := stubCaptures(t, cfg)
	if !slices.Contains(captured, "tok-ana") {
		t.Fatalf("capture-screen was never asked for tok-ana; it captured %q", captured)
	}
	// The stub answers `capture-screen` for whatever token it is handed, so a
	// token missing from its log is the daemon declining to read that screen
	// rather than the stub refusing to show one.
	for _, token := range []string{"tok-bob", "tok-cid"} {
		if slices.Contains(captured, token) {
			t.Errorf("capture-screen was asked for %s; it captured %q", token, captured)
		}
	}

	// Both are still where their own feeds left them.
	for _, m := range []struct {
		token      string
		attendance *chatcli.Attendance
	}{{"tok-bob", claude}, {"tok-cid", codex}} {
		self := m.attendance.Self()
		row := memberRow(t, cfg, m.token, self.GetTeam()+"/"+self.GetName())
		if len(row) < 3 || row[2] != "done" {
			t.Errorf("members row for %s = %v, want its state column to read done",
				m.token, row)
		}
	}
}
