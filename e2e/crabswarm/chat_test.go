package crabswarm_test

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// chatRoom is the working directory the stub cmdman reports for every command
// it knows, and therefore the room all the members below share.
const chatRoom = "/work/proj"

// stubCmdmanScript answers the one cmdman surface the chat broker uses:
// `inspect <ID> --format '{{json .Config}}'`. Each token it knows maps to a
// compose project, which becomes the member's team; anything else fails the way
// cmdman fails on an unknown ID, since the daemon reads that exact wording as
// "this token names nothing" rather than "the lookup broke".
const stubCmdmanScript = `#!/bin/sh
if [ "$1" != "inspect" ]; then
	echo "stub cmdman: unsupported invocation: $*" >&2
	exit 1
fi
case "$2" in
	tok-ana) project=alpha ;;
	tok-bob) project=alpha ;;
	tok-cid) project=beta ;;
	*)
		echo "error: resolve command: no command found matching \"$2\"" >&2
		exit 1
		;;
esac
printf '{"dir":"%s","labels":{"cmdman.compose.project":"%s"}}\n' "` + chatRoom + `" "$project"
`

// chatEnviron is the environment the crabswarm processes below run with: the
// test's own, minus everything that would override the config file this test
// hands them or supply an identity token behind its back. The suite may itself
// run under cmdman, which exports CMDMAN_CMD_ID.
func chatEnviron() []string {
	var env []string
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(name, "CRABSWARM_") || name == "CMDMAN_CMD_ID" {
			continue
		}
		env = append(env, kv)
	}
	return env
}

// startChatDaemon writes a config naming a private socket, database and stub
// cmdman, starts `crabswarm serve` on it, and returns the config path.
func startChatDaemon(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	stub := filepath.Join(dir, "cmdman")
	writeFile(t, stub, stubCmdmanScript)
	if err := os.Chmod(stub, 0o755); err != nil {
		t.Fatalf("chmod stub cmdman: %v", err)
	}

	sock := filepath.Join(dir, "chat.sock")
	cfgPath := filepath.Join(dir, "config.json")
	writeFile(t, cfgPath, fmt.Sprintf(
		`{"sock":%q,"chat":{"db":%q,"cmdman_bin":%q}}`,
		sock, filepath.Join(dir, "chat.db"), stub))

	serve := exec.Command(crabswarmBin, "serve", "--config", cfgPath)
	serve.Env = chatEnviron()
	serve.Stdout = os.Stderr
	serve.Stderr = os.Stderr
	if err := serve.Start(); err != nil {
		t.Fatalf("start crabswarm serve: %v", err)
	}
	t.Cleanup(func() { stopProcess(t, serve) })

	waitSocket(t, sock, 30*time.Second)
	return cfgPath
}

// waitSocket blocks until something accepts on the Unix socket at path.
func waitSocket(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("unix", path)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("crabswarm serve did not listen on %s within %s", path, timeout)
}

// runChat runs `crabswarm chat ...` as the holder of token and returns its
// stdout. A non-zero exit fails the test.
func runChat(t *testing.T, cfgPath, token string, args ...string) string {
	t.Helper()
	stdout, stderr, err := execChat(t, cfgPath, token, args...)
	if err != nil {
		t.Fatalf("crabswarm chat %s: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout, stderr)
	}
	return stdout
}

// execChat is runChat without the failure check, for the cases that assert on
// one.
func execChat(
	t *testing.T,
	cfgPath, token string,
	args ...string,
) (stdout, stderr string, err error) {
	t.Helper()
	full := append([]string{"chat", "--config", cfgPath}, args...)
	if token != "" {
		full = append(full, "--token", token)
	}
	cmd := exec.CommandContext(t.Context(), crabswarmBin, full...)
	cmd.Env = chatEnviron()
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	if err != nil {
		if _, ok := errors.AsType[*exec.ExitError](err); !ok {
			t.Fatalf("run crabswarm chat %s: %v", strings.Join(args, " "), err)
		}
	}
	return outBuf.String(), errBuf.String(), err
}

// lines splits rendered output into its non-empty lines.
func lines(s string) []string {
	var out []string
	for l := range strings.SplitSeq(s, "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

// TestChat drives the member verbs against a real daemon over its Unix socket,
// with a stub cmdman standing in for the team-info provider. It covers the
// whole life of an attendance: join, address a teammate, read (and consume) the
// mail, broadcast across teams, report a harness state, and leave.
func TestChat(t *testing.T) {
	cfg := startChatDaemon(t)

	// Join: the room and the team come from the token, not from the caller.
	got := runChat(t, cfg, "tok-ana", "join", "--name", "ana")
	if want := "joined " + chatRoom + " as alpha/ana\n"; got != want {
		t.Errorf("join = %q, want %q", got, want)
	}
	runChat(t, cfg, "tok-bob", "join", "--name", "bob")
	runChat(t, cfg, "tok-cid", "join", "--name", "cid")

	// Members: everyone in the room, across teams, spelled as an address.
	members := lines(runChat(t, cfg, "tok-ana", "members"))
	slices.Sort(members)
	want := []string{"alpha/ana", "alpha/bob", "beta/cid"}
	if !slices.Equal(members, want) {
		t.Errorf("members = %v, want %v", members, want)
	}

	// Send: a bare name resolves within the sender's own team.
	got = runChat(t, cfg, "tok-ana", "send", "bob", "ping")
	if want := "sent to alpha/bob\n"; got != want {
		t.Errorf("send = %q, want %q", got, want)
	}

	// Read: the message arrives team-qualified and stamped.
	got = runChat(t, cfg, "tok-bob", "read")
	if !strings.Contains(got, "alpha/ana: ping") {
		t.Errorf("read = %q, want it to carry the sender and text", got)
	}
	if !strings.HasPrefix(got, "[") {
		t.Errorf("read = %q, want it to open with a timestamp", got)
	}

	// Reading consumed it: a message is handed out exactly once.
	got = runChat(t, cfg, "tok-bob", "read")
	if want := "no pending messages\n"; got != want {
		t.Errorf("second read = %q, want %q", got, want)
	}

	// Broadcast reaches the other team too, and not the sender.
	got = runChat(t, cfg, "tok-ana", "broadcast", "standup in 5")
	if want := "broadcast to 2 members\n"; got != want {
		t.Errorf("broadcast = %q, want %q", got, want)
	}
	got = runChat(t, cfg, "tok-cid", "read")
	if !strings.Contains(got, "alpha/ana: standup in 5") {
		t.Errorf("read after broadcast = %q, want the broadcast text", got)
	}

	// report-state is driven by harness hooks, so it stays silent.
	if got := runChat(t, cfg, "tok-ana", "report-state", "idle"); got != "" {
		t.Errorf("report-state wrote %q, want nothing", got)
	}

	// Leave withdraws the attendance, and the room reflects it.
	if got := runChat(t, cfg, "tok-cid", "leave"); got != "left the room\n" {
		t.Errorf("leave = %q, want %q", got, "left the room\n")
	}
	members = lines(runChat(t, cfg, "tok-ana", "members"))
	if slices.Contains(members, "beta/cid") {
		t.Errorf("members after leave = %v, want no beta/cid", members)
	}
}

// A refused request is reported in the daemon's own words: the CLI unwraps the
// gRPC status so the reader gets the sentence that names what went wrong, not
// the "rpc error: code = ..." envelope around it.
func TestChat_ServerErrorIsReportedPlainly(t *testing.T) {
	cfg := startChatDaemon(t)
	runChat(t, cfg, "tok-ana", "join", "--name", "ana")

	_, stderr, err := execChat(t, cfg, "tok-ana", "send", "nobody", "hi")
	if err == nil {
		t.Fatal("send to an absent member succeeded, want a failure")
	}
	if !strings.Contains(stderr, "nobody") {
		t.Errorf("stderr = %q, want it to name the unresolved address", stderr)
	}
	if strings.Contains(stderr, "rpc error") {
		t.Errorf("stderr = %q, want the daemon's message without the gRPC envelope", stderr)
	}
}

// A token no provider vouches for cannot attend, and the refusal reaches the
// user rather than being swallowed into an empty success.
func TestChat_UnknownTokenIsRejected(t *testing.T) {
	cfg := startChatDaemon(t)

	stdout, stderr, err := execChat(t, cfg, "tok-stranger", "join", "--name", "who")
	if err == nil {
		t.Fatal("join with an unknown token succeeded, want a failure")
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want nothing", stdout)
	}
	if !strings.Contains(stderr, "error:") {
		t.Errorf("stderr = %q, want a reported error", stderr)
	}
}
