package crabswarm_test

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// followTimeout bounds the wait for one line of `chat follow`. A reconnect
// waits out the follower's backoff on top of a daemon restart, so it is
// generous.
const followTimeout = 30 * time.Second

// followLine is one line `chat follow` printed. The protocol fields are
// pointers so a line that left one out fails the comparison rather than
// reading as its zero value, and seq is an integer so a seq written as a JSON
// string fails to decode.
type followLine struct {
	Type         string `json:"type"`
	Message      string `json:"message"`
	Inject       *bool  `json:"inject"`
	MentionedYou *bool  `json:"mentioned_you"`
	ID           string `json:"id"`
	Seq          int64  `json:"seq"`
	From         *struct {
		Team string `json:"team"`
		Name string `json:"name"`
	} `json:"from"`
	Target json.RawMessage `json:"target"`
	Text   string          `json:"text"`
	SentAt string          `json:"sent_at"`
}

// chatFollower is a running `crabswarm chat follow`, its stdout read line by
// line as it is printed.
type chatFollower struct {
	cmd    *exec.Cmd
	lines  <-chan string
	exited <-chan error
	// stopped is set once stop has collected the exit.
	stopped bool
}

// startChatFollow starts `crabswarm chat <args...>` in dir, the test's own
// directory when empty, and returns once it is running. The command never
// ends on its own, so the cleanup kills whatever stop did not end.
//
// stdout is a pipe the test owns rather than cmd.StdoutPipe, which Wait closes
// as the process exits and so can drop a line the reader had not reached yet.
func startChatFollow(t *testing.T, dir string, args ...string) *chatFollower {
	t.Helper()
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("open the follow pipe: %v", err)
	}
	cmd := exec.Command(crabswarmBin, append([]string{"chat"}, args...)...)
	cmd.Env = chatEnviron()
	cmd.Dir = dir
	cmd.Stdout = pw
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start crabswarm chat follow: %v", err)
	}
	_ = pw.Close()

	lines := make(chan string, 64)
	go func() {
		defer close(lines)
		defer pr.Close()
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	f := &chatFollower{cmd: cmd, lines: lines, exited: exited}
	t.Cleanup(func() {
		if f.stopped {
			return
		}
		_ = cmd.Process.Kill()
		<-exited
	})
	return f
}

// next returns the next line, decoded. Every line has to be a JSON object.
func (f *chatFollower) next(t *testing.T) followLine {
	t.Helper()
	select {
	case raw, ok := <-f.lines:
		if !ok {
			t.Fatal("chat follow closed its stdout, want another line")
		}
		var l followLine
		if err := json.Unmarshal([]byte(raw), &l); err != nil {
			t.Fatalf("chat follow printed %q, which is not a line of the protocol: %v", raw, err)
		}
		return l
	case <-time.After(followTimeout):
		t.Fatalf("chat follow printed nothing within %s", followTimeout)
		return followLine{}
	}
}

// expectStatus reads the next line and fails unless it is the status message.
func (f *chatFollower) expectStatus(t *testing.T, message string) {
	t.Helper()
	l := f.next(t)
	if l.Type != "status" || l.Message != message {
		t.Fatalf("chat follow printed %+v, want the status %q", l, message)
	}
}

// expectMessage reads the next line and fails unless it is a message line
// showing shown and marked for injection as inject says. It returns the line
// for the assertions a case adds.
func (f *chatFollower) expectMessage(t *testing.T, shown string, inject bool) followLine {
	t.Helper()
	l := f.next(t)
	if l.Type != "message" || l.Message != shown {
		t.Fatalf("chat follow printed %+v, want the message %q", l, shown)
	}
	if l.Inject == nil || *l.Inject != inject {
		t.Errorf("message %q carries inject %v, want %v", shown, l.Inject, inject)
	}
	if l.MentionedYou == nil || *l.MentionedYou != inject {
		t.Errorf("message %q carries mentioned_you %v, want %v", shown, l.MentionedYou, inject)
	}
	if l.ID == "" || l.Seq <= 0 {
		t.Errorf("message %q carries id %q and seq %d, want both set", shown, l.ID, l.Seq)
	}
	if _, err := time.Parse(time.RFC3339, l.SentAt); err != nil {
		t.Errorf("message %q carries sent_at %q, which is not an RFC3339 instant: %v",
			shown, l.SentAt, err)
	}
	return l
}

// stop interrupts the command the way a reader shutting down does, and fails
// unless it exits cleanly. It returns whatever the command printed that the
// case had not read.
func (f *chatFollower) stop(t *testing.T) []string {
	t.Helper()
	if err := f.cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("interrupt chat follow: %v", err)
	}
	select {
	case err := <-f.exited:
		f.stopped = true
		if err != nil {
			t.Errorf("chat follow exited with %v on an interrupt, want status 0", err)
		}
	case <-time.After(followTimeout):
		t.Fatalf("chat follow did not exit within %s of an interrupt", followTimeout)
	}
	var rest []string
	for l := range f.lines {
		rest = append(rest, l)
	}
	return rest
}

// realTempDir is a fresh directory spelled the way the kernel reports a working
// directory, which is the spelling a room resolved from one is compared in.
func realTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolve the temporary directory: %v", err)
	}
	return dir
}

// A member follows its room with a token nobody attends under. The stream
// opens live, marks the messages that are for the follower, and puts nobody in
// the room.
func TestChatFollow_MemberFollowsWithoutAttending(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)
	bob := registerChatHuman(t, cfg, identity, chatRoom, "alpha", "bob")
	registerChatHuman(t, cfg, identity, chatRoom, "alpha", "ana")
	runChat(t, cfg, bob, "send", "everyone", "said before the follow")

	f := startChatFollow(t, "", "--config", cfg, "follow", "--token", "tok-cid")
	f.expectStatus(t, "following "+chatRoom)

	// everyone names the follower; a board post and a message to someone else
	// do not.
	runChat(t, cfg, bob, "send", "everyone", "main is red")
	first := f.expectMessage(t, "alpha/bob: main is red", true)
	if first.From == nil || first.From.Team != "alpha" || first.From.Name != "bob" {
		t.Errorf("message from = %+v, want alpha/bob", first.From)
	}
	if first.Text != "main is red" {
		t.Errorf("message text = %q, want %q", first.Text, "main is red")
	}
	if string(first.Target) != `{"everyone":{}}` {
		t.Errorf("message target = %s, want everyone", first.Target)
	}

	runChat(t, cfg, bob, "send", "", "fyi: rebased <main> & pushed")
	post := f.expectMessage(t, "alpha/bob: fyi: rebased <main> & pushed", false)
	if post.Target != nil {
		t.Errorf("board post target = %s, want none", post.Target)
	}
	runChat(t, cfg, bob, "send", "alpha/ana", "for ana alone")
	toAna := f.expectMessage(t, "alpha/bob: for ana alone", false)
	if first.Seq >= post.Seq || post.Seq >= toAna.Seq {
		t.Errorf("seqs = %d, %d, %d, want them increasing", first.Seq, post.Seq, toAna.Seq)
	}

	// Following is not attendance.
	members := memberAddresses(runChat(t, cfg, bob, "members"))
	if slices.Contains(members, chatBridgeCid) {
		t.Errorf("members = %v, want no %s: following attends nothing", members, chatBridgeCid)
	}

	if rest := f.stop(t); len(rest) != 0 {
		t.Errorf("chat follow printed %q after the last message, want nothing", rest)
	}
}

// An operator following from a subdirectory of a room follows that room, as
// nobody: nothing in it is for them. A directory under no room is followed
// itself, waiting for the agents that would make it one.
func TestChatFollow_AdminFollowsTheRoomOfItsDirectory(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)
	room := realTempDir(t)
	sub := filepath.Join(room, "src", "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("make the subdirectory: %v", err)
	}
	ana := registerChatHuman(t, cfg, identity, room, "alpha", "ana")

	f := startChatFollow(t, sub, "--config", cfg, "follow", "--admin", "--identity", identity)
	f.expectStatus(t, "following "+room)
	runChat(t, cfg, ana, "send", "everyone", "hello operator")
	f.expectMessage(t, "alpha/ana: hello operator", false)
	f.stop(t)

	elsewhere := realTempDir(t)
	g := startChatFollow(t, elsewhere, "--config", cfg, "follow", "--admin", "--identity", identity)
	g.expectStatus(t, "following "+elsewhere)
	g.stop(t)
}

// An operator's follow started before the daemon picks its room once the daemon
// answers, waiting the way a lost stream does rather than giving up.
func TestChatFollow_AdminWaitsForTheDaemonToPickItsRoom(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := writeChatConfig(t, 0, defaultStubCommands(), recipient)
	dir := realTempDir(t)

	f := startChatFollow(t, dir, "--config", cfg, "follow", "--admin", "--identity", identity)
	f.expectStatus(t, "reconnecting to the daemon")
	startChatServe(t, cfg)
	f.expectStatus(t, "following "+dir)

	if rest := f.stop(t); len(rest) != 0 {
		t.Errorf("chat follow printed %q after following, want nothing", rest)
	}
}

// A room nobody attends can be deleted under an operator's follow, and the
// messages said in it afterwards number from one again. The follow opens over
// on the restarted room and prints them, rather than holding each one back as
// a seq it already printed.
func TestChatFollow_FollowsARoomDeletedUnderIt(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)
	send := func(text string) {
		t.Helper()
		runChat(t, cfg, "", "admin", "send", chatRoom, "everyone", text, "--identity", identity)
	}
	send("said before the follow")
	send("also said before the follow")

	f := startChatFollow(t, "", "--config", cfg,
		"follow", "--admin", "--identity", identity, "--room", chatRoom)
	f.expectStatus(t, "following "+chatRoom)
	send("before the delete")
	if before := f.expectMessage(t, "admin: before the delete", false); before.Seq != 3 {
		t.Errorf("seq before the delete = %d, want 3", before.Seq)
	}

	runChat(t, cfg, "", "admin", "delete-room", chatRoom, "--identity", identity)
	send("after the delete")
	f.expectStatus(t, "following "+chatRoom)
	if after := f.expectMessage(t, "admin: after the delete", false); after.Seq != 1 {
		t.Errorf("seq after the delete = %d, want 1", after.Seq)
	}
	send("and again")
	if again := f.expectMessage(t, "admin: and again", false); again.Seq != 2 {
		t.Errorf("seq of the next message = %d, want 2", again.Seq)
	}

	if rest := f.stop(t); len(rest) != 0 {
		t.Errorf("chat follow printed %q after the last message, want nothing", rest)
	}
}

// sendWhileUnreachable sends text to everyone in chatRoom through a second
// daemon that shares cfgPath's database but listens on a socket of its own,
// then stops that daemon. A follower dialing cfgPath's socket cannot reach it,
// so the message can reach the follower only later, replayed by the resume
// after the last seq the follower saw.
func sendWhileUnreachable(t *testing.T, cfgPath, identity, text string) {
	t.Helper()
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read the daemon config: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("decode the daemon config: %v", err)
	}
	other := filepath.Join(t.TempDir(), "config.json")
	cfg["sock"] = chatSock(other)
	moved, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("encode the second daemon config: %v", err)
	}
	writeFile(t, other, string(moved))

	serve := startChatServe(t, other)
	runChat(t, other, "", "admin", "send", chatRoom, "everyone", text, "--identity", identity)
	stopProcess(t, serve)
}

// A follow outlives a daemon restart: it says it is reconnecting, follows the
// room again once the daemon is back, and prints every message once — the
// ones sent while it could not reach any daemon included, which only the
// resume after the last seq it printed can bring it.
func TestChatFollow_ReconnectsAcrossADaemonRestart(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := writeChatConfig(t, 0, defaultStubCommands(), recipient)
	serve := startChatServe(t, cfg)
	send := func(text string) {
		t.Helper()
		runChat(t, cfg, "", "admin", "send", chatRoom, "everyone", text, "--identity", identity)
	}

	f := startChatFollow(t, "", "--config", cfg, "follow", "--token", "tok-ana")
	f.expectStatus(t, "following "+chatRoom)
	send("before the restart")
	before := f.expectMessage(t, "admin: before the restart", true)

	stopProcess(t, serve)
	f.expectStatus(t, "reconnecting to the daemon")
	sendWhileUnreachable(t, cfg, identity, "while the follow was away")
	startChatServe(t, cfg)

	f.expectStatus(t, "following "+chatRoom)
	away := f.expectMessage(t, "admin: while the follow was away", true)
	send("after the reconnect")
	after := f.expectMessage(t, "admin: after the reconnect", true)
	if before.Seq >= away.Seq || away.Seq >= after.Seq {
		t.Errorf("seqs = %d, %d, %d, want them increasing", before.Seq, away.Seq, after.Seq)
	}

	if rest := f.stop(t); len(rest) != 0 {
		t.Errorf("chat follow printed %q after the last message, want nothing", rest)
	}
}

// A follow of a room nothing had been said in when it opened has no message to
// resume after. It resumes after the seq the room stood at, zero, and so still
// prints the message sent while it could not reach any daemon, once.
func TestChatFollow_ReconnectsAcrossADaemonRestartFromAnEmptyRoom(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := writeChatConfig(t, 0, defaultStubCommands(), recipient)
	serve := startChatServe(t, cfg)
	send := func(text string) {
		t.Helper()
		runChat(t, cfg, "", "admin", "send", chatRoom, "everyone", text, "--identity", identity)
	}

	f := startChatFollow(t, "", "--config", cfg, "follow", "--token", "tok-ana")
	f.expectStatus(t, "following "+chatRoom)

	stopProcess(t, serve)
	f.expectStatus(t, "reconnecting to the daemon")
	sendWhileUnreachable(t, cfg, identity, "while the follow was away")
	startChatServe(t, cfg)

	f.expectStatus(t, "following "+chatRoom)
	away := f.expectMessage(t, "admin: while the follow was away", true)
	send("after the reconnect")
	after := f.expectMessage(t, "admin: after the reconnect", true)
	if away.Seq != 1 || after.Seq != 2 {
		t.Errorf("seqs = %d, %d, want 1, 2", away.Seq, after.Seq)
	}

	if rest := f.stop(t); len(rest) != 0 {
		t.Errorf("chat follow printed %q after the last message, want nothing", rest)
	}
}
