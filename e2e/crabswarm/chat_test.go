package crabswarm_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// chatRoom is the working directory the stub cmdman reports for most of the
// commands it knows, and therefore the room those members share. chatOtherRoom
// is a second working directory, so a case can put a member somewhere the first
// room must not be able to see.
const (
	chatRoom      = "/work/proj"
	chatOtherRoom = "/work/other"
)

// chatTokenEnvVar carries a registered human's identity token to the CLI. It is
// spelled out rather than taken from the chat package: an e2e case reaches the
// binary the way its user does, and the name of the variable is part of that
// surface.
const chatTokenEnvVar = "CRABSWARM_CHAT_TOKEN"

// stubCommand is one command the stub cmdman knows: the token that names it,
// the working directory that becomes its holder's room, and the compose project
// that becomes its team. An empty project is a command started outside any
// compose project, which carries no team coordination information at all.
//
// command and scaleIndex are the other two compose labels, the ones an attendee
// that names itself nothing is named after: the name the compose file declares
// the command under and the replica index that tells one instance of a scaled
// command from another. Either may be empty, which is a command whose labels do
// not say.
//
// state is what cmdman reports the command's process to be doing. An empty one
// is a command that is running, which is what every case here wants.
type stubCommand struct {
	token      string
	dir        string
	project    string
	command    string
	scaleIndex string
	state      string
}

// stubState is the state the stub cmdman reports for c, defaulting to running.
func stubState(c stubCommand) string {
	if c.state == "" {
		return "running"
	}
	return c.state
}

// defaultStubCommands is the roster the plain member cases run against: two
// teammates and one member of another team, all in one room.
func defaultStubCommands() []stubCommand {
	return []stubCommand{
		{token: "tok-ana", dir: chatRoom, project: "alpha"},
		{token: "tok-bob", dir: chatRoom, project: "alpha"},
		{token: "tok-cid", dir: chatRoom, project: "beta"},
	}
}

// stubCmdmanScript renders a stub cmdman answering the surfaces the chat broker
// uses: `inspect <ID> --format '{{.State}} {{json .Config}}'` for placement,
// `status set|delete` for the state display, and `capture-screen` / `send-keys`
// for the terminal nudge. Each token in commands reports its own process state,
// directory and compose project; an unknown ID fails the way cmdman fails on
// one, since the daemon reads that exact wording as "this token names nothing"
// rather than "the lookup broke".
//
// A command whose state is not running is still answered for, the way cmdman
// answers about a command whose process has ended until it is removed. The
// daemon is what draws the line between the two.
//
// Status and send-keys invocations are recorded beside the stub rather than
// answered, so a test can read back what the daemon published and what it typed.
// The stub finds the logs relative to $0, which keeps it independent of the
// environment the daemon runs it with. The screen it captures is a bare prompt:
// the nudge declines on anything that looks like a dialog, and every case here
// wants the terminal to be idle.
//
// The label object of each command is rendered in Go and embedded whole: the
// stub only has to echo it back, so the shell never assembles JSON.
func stubCmdmanScript(commands []stubCommand) string {
	var b strings.Builder
	b.WriteString(`#!/bin/sh
if [ "$1" = "status" ]; then
	shift
	printf '%s\n' "$*" >> "$(dirname "$0")/status.log"
	exit 0
fi
if [ "$1" = "send-keys" ]; then
	shift
	printf '%s\n' "$*" >> "$(dirname "$0")/send-keys.log"
	exit 0
fi
if [ "$1" = "capture-screen" ]; then
	printf 'stub@terminal:~$ \n'
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

// stubLabels renders one stub command's compose labels as the JSON object a
// cmdman config carries them in. A label the command leaves empty is omitted
// rather than emitted blank: that is how a command whose compose file declares
// none looks to the daemon.
func stubLabels(c stubCommand) string {
	labels := map[string]string{}
	for _, l := range []struct{ name, value string }{
		{"cmdman.compose.project", c.project},
		{"cmdman.compose.command", c.command},
		{"cmdman.compose.scale-index", c.scaleIndex},
	} {
		if l.value != "" {
			labels[l.name] = l.value
		}
	}
	b, err := json.Marshal(labels)
	if err != nil {
		panic(err) // a map[string]string always marshals
	}
	return string(b)
}

// stubStatus returns the `cmdman status` invocations the daemon made through
// the stub, oldest first, with the leading "status" already stripped. No log
// file means the daemon never published anything.
func stubStatus(t *testing.T, cfgPath string) []string {
	t.Helper()
	return stubLog(t, cfgPath, "status.log")
}

// stubSendKeys returns the `cmdman send-keys` invocations the daemon made
// through the stub, oldest first, with the leading "send-keys" already
// stripped. No log file means no terminal was typed into at all.
func stubSendKeys(t *testing.T, cfgPath string) []string {
	t.Helper()
	return stubLog(t, cfgPath, "send-keys.log")
}

// stubLog reads back one of the logs the stub cmdman appends its invocations
// to, oldest first. A log that is not there is one nothing was recorded in.
func stubLog(t *testing.T, cfgPath, name string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(filepath.Dir(cfgPath), name))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("reading stub %s: %v", name, err)
	}
	var lines []string
	for l := range strings.SplitSeq(strings.TrimSpace(string(b)), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

// waitStubStatus blocks until the daemon has published want through the stub
// cmdman. A status the daemon publishes as a session ends is written after the
// command that ended it has already returned, so a case reading the log
// straight away would read it before the line is there.
func waitStubStatus(t *testing.T, cfgPath, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var published []string
	for time.Now().Before(deadline) {
		published = stubStatus(t, cfgPath)
		if slices.Contains(published, want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("cmdman status never carried %q within %s; it carried %q",
		want, timeout, published)
}

// chatEnviron is the environment the crabswarm processes below run with: the
// test's own, minus everything that would override the config file this test
// hands them or supply an identity token behind its back. The suite may itself
// run under cmdman, which exports CMDMAN_CMD_ID.
//
// FAKECHAT_PORT goes with them. The suite may be run beside a live Claude Code
// whose fakechat plugin is listening on the port that environment names, and a
// bridge started here that inherited it would post the suite's notices into that
// session. The cases that want a plugin name a port of their own.
func chatEnviron() []string {
	var env []string
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(name, "CRABSWARM_") || name == "CMDMAN_CMD_ID" ||
			name == "FAKECHAT_PORT" {
			continue
		}
		env = append(env, kv)
	}
	return env
}

// startChatDaemon starts a daemon knowing the default roster and holding no
// admin key, which is all the member verbs need.
func startChatDaemon(t *testing.T) string {
	t.Helper()
	return startChatDaemonWith(t, defaultStubCommands())
}

// startChatDaemonWith writes a config naming a private socket, database and
// stub cmdman, starts `crabswarm serve` on it, and returns the config path.
// commands is the roster the stub cmdman vouches for; adminRecipients are the
// age public keys the daemon encrypts admin challenges to, none for a daemon
// that was never given one and therefore refuses every admin verb.
func startChatDaemonWith(t *testing.T, commands []stubCommand, adminRecipients ...string) string {
	t.Helper()
	return startChatDaemonKeeping(t, 0, commands, adminRecipients...)
}

// startChatDaemonKeeping is startChatDaemonWith with the per-room message cap
// the config names, for the cases that assert on what a host's
// chat.history_limit does to a live room. Zero is the config's own "unset", so
// the daemon keeps its default.
func startChatDaemonKeeping(
	t *testing.T,
	historyLimit int,
	commands []stubCommand,
	adminRecipients ...string,
) string {
	t.Helper()

	cfgPath := writeChatConfig(t, historyLimit, commands, adminRecipients...)
	startChatServe(t, cfgPath)
	return cfgPath
}

// writeChatConfig writes the config a daemon and everything talking to it
// share — the socket, the database, the stub cmdman and the admin recipients —
// and returns its path. It starts nothing, so a case can hand the config to a
// client that has to cope with a daemon that is not there yet.
func writeChatConfig(
	t *testing.T,
	historyLimit int,
	commands []stubCommand,
	adminRecipients ...string,
) string {
	t.Helper()
	dir := t.TempDir()
	return writeChatConfigOn(t, dir, chatSock(filepath.Join(dir, "config.json")),
		historyLimit, commands, adminRecipients...)
}

// writeChatConfigOn is writeChatConfig with the directory it works in and the
// socket it names given by the caller, for the case that has the bridge derive
// the socket path from its environment instead of reading it out of a config.
func writeChatConfigOn(
	t *testing.T,
	dir, sock string,
	historyLimit int,
	commands []stubCommand,
	adminRecipients ...string,
) string {
	t.Helper()

	stub := filepath.Join(dir, "cmdman")
	writeFile(t, stub, stubCmdmanScript(commands))
	if err := os.Chmod(stub, 0o755); err != nil {
		t.Fatalf("chmod stub cmdman: %v", err)
	}

	recipients, err := json.Marshal(adminRecipients)
	if err != nil {
		t.Fatalf("marshal admin recipients: %v", err)
	}

	cfgPath := filepath.Join(dir, "config.json")
	writeFile(t, cfgPath, fmt.Sprintf(
		`{"sock":%q,"chat":{"db":%q,"cmdman_bin":%q,"admin_recipients":%s,"history_limit":%d}}`,
		sock, filepath.Join(dir, "chat.db"), stub, recipients, historyLimit))
	return cfgPath
}

// startChatServe starts `crabswarm serve` on cfgPath and returns once its
// socket answers. The process is returned so a case can end it and start
// another on the same config, which is what a daemon restart is.
func startChatServe(t *testing.T, cfgPath string) *exec.Cmd {
	t.Helper()
	return startChatServeOn(t, cfgPath, chatSock(cfgPath))
}

// startChatServeOn is startChatServe for a config naming a socket somewhere
// other than beside it, which is the path the daemon then has to be waited on.
func startChatServeOn(t *testing.T, cfgPath, sock string) *exec.Cmd {
	t.Helper()

	serve := exec.Command(crabswarmBin, "serve", "--config", cfgPath)
	serve.Env = chatEnviron()
	serve.Stdout = os.Stderr
	serve.Stderr = os.Stderr
	if err := serve.Start(); err != nil {
		t.Fatalf("start crabswarm serve: %v", err)
	}
	t.Cleanup(func() { stopProcess(t, serve) })

	waitSocket(t, sock, 30*time.Second)
	return serve
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
// one. An empty token passes no --token at all, which is what the admin verbs
// want: they authenticate by identity file instead.
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
	return execChatEnv(t, nil, full...)
}

// execChatTokenEnv is execChat with the identity carried in the environment
// instead of on the command line — how a registered human's shell holds the
// token `chat admin register` printed.
func execChatTokenEnv(
	t *testing.T,
	cfgPath, token string,
	args ...string,
) (stdout, stderr string, err error) {
	t.Helper()
	full := append([]string{"chat", "--config", cfgPath}, args...)
	return execChatEnv(t, []string{chatTokenEnvVar + "=" + token}, full...)
}

// execChatEnv runs the built binary with the suite's scrubbed environment plus
// extraEnv, which is appended last so a variable set here survives the scrub.
func execChatEnv(
	t *testing.T,
	extraEnv []string,
	args ...string,
) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), crabswarmBin, args...)
	cmd.Env = append(chatEnviron(), extraEnv...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	if err != nil {
		if _, ok := errors.AsType[*exec.ExitError](err); !ok {
			t.Fatalf("run crabswarm %s: %v", strings.Join(args, " "), err)
		}
	}
	return outBuf.String(), errBuf.String(), err
}

// newChatIdentityFile writes a fresh age identity the way the age CLI does —
// one private key per line — and returns its path together with the recipient
// string a daemon is configured with to challenge its holder.
func newChatIdentityFile(t *testing.T) (path, recipient string) {
	t.Helper()
	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate age identity: %v", err)
	}
	path = filepath.Join(t.TempDir(), "chat_admin.key")
	writeFile(t, path, "# created by the crabswarm e2e suite\n"+id.String()+"\n")
	return path, id.Recipient().String()
}

// registerChatHuman runs the admin register verb and returns the token it
// printed, which is the only time the daemon reveals it. Registration is also
// the attendance: the person is in the room from here until the daemon
// restarts, with no stream of their own to hold it.
func registerChatHuman(t *testing.T, cfgPath, identity, room, team, name string) string {
	t.Helper()
	out := runChat(t, cfgPath, "", "admin", "register",
		room, team, name, "--identity", identity)
	want := fmt.Sprintf("registered %s/%s in room %s", team, name, room)
	if !strings.Contains(out, want) {
		t.Errorf("register = %q, want it to report %q", out, want)
	}
	var token string
	for _, l := range lines(out) {
		if rest, ok := strings.CutPrefix(l, "token: "); ok {
			token = rest
		}
	}
	if token == "" {
		t.Fatalf("register printed no token line; got:\n%s", out)
	}
	return token
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

// memberAddresses is the first column of a `chat members` listing — the role
// `chat send` addresses — for the assertions about who attends rather than
// about how a member is rendered.
//
// A room with nobody in it prints one sentence saying so rather than an empty
// listing, and reading a word of that sentence as an address would report a
// member nobody can write to. The empty room is the empty roster.
func memberAddresses(s string) []string {
	if strings.TrimSpace(s) == emptyRosterLine {
		return nil
	}
	var out []string
	for _, l := range lines(s) {
		fields := strings.Fields(l)
		if len(fields) == 0 {
			continue
		}
		out = append(out, fields[0])
	}
	return out
}

// emptyRosterLine is what `chat members` prints for a room nobody attends.
const emptyRosterLine = "no members"

// The addresses the bridge cases below spell. A bridge attends under no name at
// all — an agent is named by whoever registered it, not by the harness it runs
// under — so the daemon derives one from the token, and that derivation is what
// makes these addresses writable in a test.
const (
	chatBridgeAna = "alpha/agent-tok-ana"
	chatBridgeBob = "alpha/agent-tok-bob"
	chatBridgeCid = "beta/agent-tok-cid"
)

// startChatBridge starts `crabswarm mcp` the way a configured harness
// does — as a stdio subprocess spoken to over MCP — and returns the session
// that harness would hold. Connecting is the handshake, so a bridge that failed
// to serve one fails the test here.
//
// The bridge is what attends: it holds one Attend stream for the whole session,
// and there is no verb a command line could declare attendance with.
func startChatBridge(t *testing.T, cfgPath, token string) *mcp.ClientSession {
	t.Helper()
	return startChatBridgeIn(t, cfgPath, token, chatEnviron())
}

// startChatBridgeIn is [startChatBridge] with the environment the harness hands
// the bridge, for the cases about what a bridge can and cannot resolve out of
// it. An empty token passes no --token at all, and an empty config path no
// --config — a bridge left to resolve the daemon socket for itself.
func startChatBridgeIn(
	t *testing.T, cfgPath, token string, env []string,
) *mcp.ClientSession {
	t.Helper()
	return startChatBridgeAs(t, cfgPath, token, "crabswarm-e2e", env)
}

// startChatBridgeAs is [startChatBridgeIn] under the name the harness gives
// itself in the MCP handshake, which is all the bridge has to go on when it
// decides which harness it is serving and whether it can deliver a mention
// itself.
func startChatBridgeAs(
	t *testing.T, cfgPath, token, clientName string, env []string,
) *mcp.ClientSession {
	t.Helper()
	args := []string{"mcp"}
	if cfgPath != "" {
		args = append(args, "--config", cfgPath)
	}
	if token != "" {
		args = append(args, "--token", token)
	}
	cmd := exec.CommandContext(t.Context(), crabswarmBin, args...)
	cmd.Env = env
	// Everything the bridge says goes to stderr by design, since stdout carries
	// the protocol; forwarding it is what makes a failing case readable.
	cmd.Stderr = os.Stderr

	client := mcp.NewClient(&mcp.Implementation{Name: clientName, Version: "v0"}, nil)
	session, err := client.Connect(t.Context(), &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect to the chat bridge for %q: %v", token, err)
	}
	// Closing the session shuts the subprocess down the way the stdio transport
	// is meant to: stdin first, then a wait for the process to go.
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// stopChatBridge ends a bridge's session, which is what ends the attendance it
// was holding: the stream goes with the process. The cleanup closes it again,
// which a closed session takes without complaint.
func stopChatBridge(t *testing.T, session *mcp.ClientSession) {
	t.Helper()
	if err := session.Close(); err != nil {
		t.Fatalf("close the chat bridge: %v", err)
	}
}

// callChatTool calls one of the bridge's tools and returns the text it answered
// with. A tool that reported a failure fails the test with the words the model
// would have read, which is where a refusal from the daemon ends up.
func callChatTool(
	t *testing.T,
	session *mcp.ClientSession,
	name string,
	args map[string]any,
) string {
	t.Helper()
	text, failed := chatToolResult(t, session, name, args)
	if failed {
		t.Fatalf("%s reported an error: %s", name, text)
	}
	return text
}

// chatToolResult is callChatTool for the cases that are about a tool failing:
// it hands back the text and whether the tool reported it as a failure.
func chatToolResult(
	t *testing.T,
	session *mcp.ClientSession,
	name string,
	args map[string]any,
) (text string, failed bool) {
	t.Helper()
	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	return chatToolText(t, res), res.IsError
}

// chatToolText unwraps the one text block a chat tool answers with.
func chatToolText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("tool answered with %d content blocks, want exactly one", len(res.Content))
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("tool content is %T, want text", res.Content[0])
	}
	return text.Text
}

// waitChatAttendance blocks until token attends, observed by a member verb the
// daemon answers for members alone.
//
// It is deliberately not a tool call: a tool waits for the bridge's own
// attendance before it acts, so it would report the same thing from inside the
// process being asked about. This reads the daemon instead, which is what an
// agent's hooks talk to.
func waitChatAttendance(t *testing.T, cfgPath, token string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var refusal string
	for time.Now().Before(deadline) {
		_, stderr, err := execChat(t, cfgPath, token, "members")
		if err == nil {
			return
		}
		refusal = stderr
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("the bridge for %s did not attend within %s; last refusal:\n%s",
		token, timeout, refusal)
}

// waitChatRosterHas blocks until the member holding observer lists address in
// its room, which is how one member watches another arrive.
func waitChatRosterHas(
	t *testing.T, cfgPath, observer, address string, timeout time.Duration,
) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var listed []string
	for time.Now().Before(deadline) {
		listed = memberAddresses(runChat(t, cfgPath, observer, "members"))
		if slices.Contains(listed, address) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("%s did not appear in the room within %s; the room held %v",
		address, timeout, listed)
}

// attendChatBridges starts a bridge per token and returns once every one of
// them is attending, so what follows acts on a room that is fully assembled.
func attendChatBridges(t *testing.T, cfgPath string, tokens ...string) []*mcp.ClientSession {
	t.Helper()
	sessions := make([]*mcp.ClientSession, len(tokens))
	for i, token := range tokens {
		sessions[i] = startChatBridge(t, cfgPath, token)
	}
	for _, token := range tokens {
		waitChatAttendance(t, cfgPath, token, 30*time.Second)
	}
	return sessions
}

// restartChatDaemon ends the daemon and starts another on the same config,
// which is what an operator restarting the service does. The database is left
// where it is: the rooms, their logs and every role's read position are the
// durable half of a room, and a case that removed them would be playing a fresh
// install rather than a restart.
func restartChatDaemon(t *testing.T, cfgPath string, serve *exec.Cmd) *exec.Cmd {
	t.Helper()
	stopProcess(t, serve)
	return startChatServe(t, cfgPath)
}

// chatMessageBody strips the sequence number and the stamp off the first
// rendered message line and returns the rest — the sender, the target, the
// mention marker and the text. Those two are the parts a case cannot pin: the
// seq depends on everything else said in the room, and the instant on when the
// case ran. Both are checked for being what they claim and then dropped.
func chatMessageBody(t *testing.T, rendered string) string {
	t.Helper()
	first, _, _ := strings.Cut(rendered, "\n")
	seq, stamped, ok := strings.Cut(first, " ")
	if !ok {
		t.Fatalf("read = %q, want a rendered message line", rendered)
	}
	if _, err := strconv.ParseInt(seq, 10, 64); err != nil {
		t.Fatalf("read = %q opens with %q, which is not a sequence number: %v",
			rendered, seq, err)
	}
	stamp, body, ok := strings.Cut(stamped, " ")
	if !ok {
		t.Fatalf("read = %q, want a timestamped message line", rendered)
	}
	if _, err := time.Parse(time.RFC3339, stamp); err != nil {
		t.Fatalf("read = %q carries %q, which is not an RFC3339 instant: %v", rendered, stamp, err)
	}
	return body
}

// TestChat drives the member verbs against a real daemon over its Unix socket,
// with a stub cmdman standing in for the team-info provider. It covers a room
// at work: three agents attending through their bridges, a message to one of
// them, a read that consumes it, a message to everyone, a board post that
// mentions nobody, and a harness state report.
func TestChat(t *testing.T) {
	cfg := startChatDaemon(t)
	attendChatBridges(t, cfg, "tok-ana", "tok-bob", "tok-cid")
	waitChatRosterHas(t, cfg, "tok-ana", chatBridgeBob, 30*time.Second)
	waitChatRosterHas(t, cfg, "tok-ana", chatBridgeCid, 30*time.Second)

	// Members: everyone in the room, across teams. The first column is the role
	// a target names, and the kind and the state follow it — the kind says
	// whether a message reaches that member on its own, which is what the sender
	// wants to know before waiting for an answer. The last two say which CLI the
	// member runs and how a mention gets to it: this suite's own client is no
	// harness the bridge recognises, so it is other, and a harness with no
	// channel of its own is typed at through its terminal.
	roster := lines(runChat(t, cfg, "tok-ana", "members"))
	slices.Sort(roster)
	want := []string{
		chatBridgeAna + "  agent  done  other  terminal",
		chatBridgeBob + "  agent  done  other  terminal",
		chatBridgeCid + "  agent  done  other  terminal",
	}
	if !slices.Equal(roster, want) {
		t.Errorf("members = %v, want %v", roster, want)
	}

	// Send: a bare name resolves within the sender's own team, and the answer
	// names the role it resolved to.
	got := runChat(t, cfg, "tok-ana", "send", "agent-tok-bob", "ping")
	if want := "mentioned " + chatBridgeBob + "\n"; got != want {
		t.Errorf("send = %q, want %q", got, want)
	}

	// Read: the message arrives with its place in the room, its sender, who it
	// was for, and the marker that says it was for the reader.
	body := chatMessageBody(t, runChat(t, cfg, "tok-bob", "read"))
	if want := chatBridgeAna + " -> " + chatBridgeBob + " [mentioned you]: ping"; body != want {
		t.Errorf("read = %q, want %q", body, want)
	}

	// The read moved the position past it, so it is not unread twice.
	if got := runChat(t, cfg, "tok-bob", "read"); got != "no pending messages\n" {
		t.Errorf("second read = %q, want %q", got, "no pending messages\n")
	}

	// everyone reaches the other team as well, and names no role in particular,
	// so there is nothing to report about who it was delivered to.
	if got := runChat(t, cfg, "tok-ana", "send", "everyone", "standup in 5"); got != "" {
		t.Errorf("send to everyone = %q, want nothing", got)
	}
	body = chatMessageBody(t, runChat(t, cfg, "tok-cid", "read"))
	if want := chatBridgeAna + " -> everyone [mentioned you]: standup in 5"; body != want {
		t.Errorf("read after the room-wide send = %q, want %q", body, want)
	}
	runChat(t, cfg, "tok-bob", "read")

	// A board post is in the room and mentions nobody, so it interrupts no one
	// and leaves every read position where it was.
	if got := runChat(t, cfg, "tok-ana", "send", "", "fyi: rebased main"); got != "" {
		t.Errorf("board post = %q, want nothing", got)
	}
	if got := runChat(t, cfg, "tok-bob", "read"); got != "no pending messages\n" {
		t.Errorf("read after the board post = %q, want nothing unread", got)
	}

	// report-state is driven by harness hooks, so it stays silent.
	if got := runChat(t, cfg, "tok-ana", "report-state", "done"); got != "" {
		t.Errorf("report-state wrote %q, want nothing", got)
	}
}

// A role is what a target names, and a role outlives the session that declared
// it. So the two ways a name can fail to reach somebody are not the same thing:
// a role that has attended and is not attending now takes the message and warns
// that nobody is there to be woken by it, while a name the room has never
// carried is refused outright, since a message addressed to nobody is one the
// sender meant to go somewhere.
//
// The refusal is reported in the daemon's own words: the CLI unwraps the gRPC
// status, so the reader gets the sentence that names what went wrong rather
// than the "rpc error: code = ..." envelope around it.
func TestChat_AnAbsentRoleKeepsItsMentionAndAnUnknownOneIsRefused(t *testing.T) {
	cfg := startChatDaemon(t)
	sessions := attendChatBridges(t, cfg, "tok-ana", "tok-bob")
	waitChatRosterHas(t, cfg, "tok-ana", chatBridgeBob, 30*time.Second)

	// Bob's session ends, which ends its attendance and nothing else.
	stopChatBridge(t, sessions[1])
	waitStubStatus(t, cfg, "delete tok-bob", 30*time.Second)

	// The send is accepted: the role resolved, so the message names it and the
	// answer says so. The warning is on stderr rather than in the answer — the
	// message was taken either way, and a caller piping the delivery lines wants
	// the warning in front of the person instead.
	stdout, stderr, err := execChat(t, cfg, "tok-ana", "send", "agent-tok-bob", "when you are back")
	if err != nil {
		t.Fatalf("send to an absent role: %v\nstderr:\n%s", err, stderr)
	}
	if want := "mentioned " + chatBridgeBob + "\n"; stdout != want {
		t.Errorf("send = %q, want %q", stdout, want)
	}
	if want := "warning: " + chatBridgeBob +
		" is not attending; the mention waits\n"; stderr != want {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}

	// Nobody was typed at: a nudge is keystrokes into a session's terminal, and
	// there is no session.
	if typed := stubSendKeys(t, cfg); typed != nil {
		t.Errorf("cmdman send-keys invocations = %q, want none: nobody was there", typed)
	}

	// A name the room has never carried is a different answer entirely.
	_, stderr, err = execChat(t, cfg, "tok-ana", "send", "nobody", "hi")
	if err == nil {
		t.Fatal("send to a role the room has never had succeeded, want a failure")
	}
	for _, want := range []string{"nobody", "unknown role"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr = %q, want it to carry %q", stderr, want)
		}
	}
	if strings.Contains(stderr, "rpc error") {
		t.Errorf("stderr = %q, want the daemon's message without the gRPC envelope", stderr)
	}

	// And the mention really did wait: a session attending under the same role
	// is handed it, once.
	attendChatBridges(t, cfg, "tok-bob")
	body := chatMessageBody(t, runChat(t, cfg, "tok-bob", "read"))
	if want := chatBridgeAna + " -> " + chatBridgeBob +
		" [mentioned you]: when you are back"; body != want {
		t.Errorf("read = %q, want %q", body, want)
	}
	if got := runChat(t, cfg, "tok-bob", "read"); got != "no pending messages\n" {
		t.Errorf("second read = %q, want the mention handed over once", got)
	}
}

// A token no provider vouches for cannot attend, and the refusal reaches the
// agent rather than being swallowed: the bridge keeps a live server whose tools
// report why the room is out of reach.
func TestChat_UnknownTokenIsRejected(t *testing.T) {
	cfg := startChatDaemon(t)
	session := startChatBridge(t, cfg, "tok-stranger")

	text, failed := chatToolResult(t, session, "chat_members", nil)
	if !failed {
		t.Fatalf("chat_members answered %q, want it to report the refused attendance", text)
	}
	if !strings.Contains(text, "no team information for this token") {
		t.Errorf("chat_members = %q, want it to name what the token is missing", text)
	}
}

// A command cmdman knows but that runs outside any compose project is refused
// just as an unknown token is: it has a working directory, so it could be
// placed in a room, but nothing says which team it coordinates with.
func TestChat_NonComposeTokenIsRejected(t *testing.T) {
	cfg := startChatDaemonWith(t, []stubCommand{
		{token: "tok-loner", dir: chatRoom},
	})
	session := startChatBridge(t, cfg, "tok-loner")

	text, failed := chatToolResult(t, session, "chat_members", nil)
	if !failed {
		t.Fatalf("chat_members answered %q, want it to report the refused attendance", text)
	}
	if !strings.Contains(text, "not part of a compose project") {
		t.Errorf("chat_members = %q, want it to name what the token is missing", text)
	}
}

// An attendee that names itself nothing is named after the compose labels of
// the command it runs under: the declared command name, suffixed with the
// replica index that tells one instance of a scaled command apart from its
// siblings. A compose author therefore addresses an agent by the name their
// compose file already gives it, and a bridge passes no name at all.
func TestChat_AttendanceWithoutNameTakesComposeLabels(t *testing.T) {
	cfg := startChatDaemonWith(t, []stubCommand{
		{token: "tok-worker", dir: chatRoom, project: "alpha",
			command: "worker", scaleIndex: "2"},
		{token: "tok-solo", dir: chatRoom, project: "alpha", command: "solo"},
		// Exactly eight characters, which is as much of a token as a
		// token-derived name carries, so the fallback below is spelled out
		// whole.
		{token: "tok-bare", dir: chatRoom, project: "alpha"},
	})
	attendChatBridges(t, cfg, "tok-worker", "tok-solo", "tok-bare")

	// An unscaled command carries no replica index to append, so the declared
	// name stands on its own; a command whose labels name it nothing falls back
	// to the token, prefixed by the kind that attended so the name does not say
	// the member is something it is not.
	for _, address := range []string{"alpha/solo", "alpha/agent-tok-bare"} {
		waitChatRosterHas(t, cfg, "tok-worker", address, 30*time.Second)
	}
	members := memberAddresses(runChat(t, cfg, "tok-worker", "members"))
	slices.Sort(members)
	want := []string{"alpha/agent-tok-bare", "alpha/solo", "alpha/worker-2"}
	if !slices.Equal(members, want) {
		t.Errorf("members = %v, want %v", members, want)
	}
}

// One name carried by several teams of a room: a bare name means the caller's
// own teammate first, then the room-wide role when only one team carries it,
// and is refused when two other teams both do.
func TestChat_NameCollisionAddressing(t *testing.T) {
	cfg := startChatDaemonWith(t, []stubCommand{
		{token: "tok-alpha-sam", dir: chatRoom, project: "alpha", command: "sam"},
		{token: "tok-beta-sam", dir: chatRoom, project: "beta", command: "sam"},
		{token: "tok-alpha-uniq", dir: chatRoom, project: "alpha", command: "uniq"},
		{token: "tok-asker", dir: chatRoom, project: "gamma", command: "asker"},
		{token: "tok-gamma-sam", dir: chatRoom, project: "gamma", command: "sam"},
	})
	attendChatBridges(t, cfg, "tok-alpha-sam", "tok-beta-sam", "tok-alpha-uniq", "tok-asker")
	for _, address := range []string{"alpha/sam", "beta/sam", "alpha/uniq"} {
		waitChatRosterHas(t, cfg, "tok-asker", address, 30*time.Second)
	}

	// Two other teams carry the name and the caller's own does not, so there is
	// nothing to prefer: the refusal names both teams and the form to retry
	// with. Asserted before gamma has a sam of its own, which would resolve it.
	_, stderr, err := execChat(t, cfg, "tok-asker", "send", "sam", "which one?")
	if err == nil {
		t.Fatal("send to an ambiguous bare name succeeded, want a failure")
	}
	for _, want := range []string{"alpha", "beta", `address it as "<team>/sam"`} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr missing %q; got:\n%s", want, stderr)
		}
	}

	// The qualified form picks exactly one of them.
	got := runChat(t, cfg, "tok-asker", "send", "beta/sam", "for beta")
	if want := "mentioned beta/sam\n"; got != want {
		t.Errorf("qualified send = %q, want %q", got, want)
	}
	body := chatMessageBody(t, runChat(t, cfg, "tok-beta-sam", "read"))
	if want := "gamma/asker -> beta/sam [mentioned you]: for beta"; body != want {
		t.Errorf("beta/sam read = %q, want %q", body, want)
	}
	if got := runChat(t, cfg, "tok-alpha-sam", "read"); got != "no pending messages\n" {
		t.Errorf("alpha/sam read = %q, want nothing: beta was addressed, not alpha", got)
	}

	// A bare name only one team of the room carries needs no qualification.
	got = runChat(t, cfg, "tok-asker", "send", "uniq", "hi")
	if want := "mentioned alpha/uniq\n"; got != want {
		t.Errorf("unique bare send = %q, want %q", got, want)
	}

	// Once the caller's own team carries the name too, that one wins over the
	// other teams' — and over the ambiguity the same address gave a moment ago.
	attendChatBridges(t, cfg, "tok-gamma-sam")
	waitChatRosterHas(t, cfg, "tok-asker", "gamma/sam", 30*time.Second)
	got = runChat(t, cfg, "tok-asker", "send", "sam", "for my own team")
	if want := "mentioned gamma/sam\n"; got != want {
		t.Errorf("own-team send = %q, want %q", got, want)
	}
	body = chatMessageBody(t, runChat(t, cfg, "tok-gamma-sam", "read"))
	if want := "gamma/asker -> gamma/sam [mentioned you]: for my own team"; body != want {
		t.Errorf("gamma/sam read = %q, want %q", body, want)
	}
}

// A room is the working directory its members' commands run in, and nothing
// crosses it: members of another room are neither listed nor addressable, not
// even under a team name both rooms happen to use.
func TestChat_RoomsAreIsolated(t *testing.T) {
	cfg := startChatDaemonWith(t, []stubCommand{
		{token: "tok-ana", dir: chatRoom, project: "alpha"},
		{token: "tok-zed", dir: chatOtherRoom, project: "alpha"},
	})
	attendChatBridges(t, cfg, "tok-ana", "tok-zed")

	// Each side lists its own room only, though both are team alpha.
	if members := memberAddresses(runChat(t, cfg, "tok-ana", "members")); !slices.Equal(
		members, []string{chatBridgeAna}) {
		t.Errorf("members = %v, want only %s", members, chatBridgeAna)
	}
	if members := memberAddresses(runChat(t, cfg, "tok-zed", "members")); !slices.Equal(
		members, []string{"alpha/agent-tok-zed"}) {
		t.Errorf("members of the other room = %v, want only alpha/agent-tok-zed", members)
	}

	// Neither spelling of the role reaches across.
	for _, addr := range []string{"agent-tok-zed", "alpha/agent-tok-zed"} {
		_, stderr, err := execChat(t, cfg, "tok-ana", "send", addr, "hello other room")
		if err == nil {
			t.Fatalf("send to %q in another room succeeded, want a failure", addr)
		}
		if !strings.Contains(stderr, "unknown role") {
			t.Errorf("stderr for %q = %q, want an unresolved role", addr, stderr)
		}
	}

	// Nor does a message to everyone, which names nobody at all.
	if got := runChat(t, cfg, "tok-ana", "send", "everyone", "anyone?"); got != "" {
		t.Errorf("send to everyone = %q, want nothing", got)
	}
	if got := runChat(t, cfg, "tok-zed", "read"); got != "no pending messages\n" {
		t.Errorf("read in the other room = %q, want nothing", got)
	}
}

// A read position is a cursor over the room rather than a receipt per message:
// it moves to the newest message the read showed, whichever cursor asked for
// it. A reader that jumps to the tail has therefore read past everything behind
// it, and the unread read that follows repeats nothing.
func TestChat_ATailReadMovesThePositionPastTheRoom(t *testing.T) {
	cfg := startChatDaemon(t)
	attendChatBridges(t, cfg, "tok-ana", "tok-bob")
	waitChatRosterHas(t, cfg, "tok-ana", chatBridgeBob, 30*time.Second)

	for i := range 3 {
		runChat(t, cfg, "tok-ana", "send", "agent-tok-bob", fmt.Sprintf("line %d", i))
	}

	tail := lines(runChat(t, cfg, "tok-bob", "read", "--cursor", "tail", "--range", "-1"))
	if len(tail) != 1 {
		t.Fatalf("tail read = %v, want the newest message alone", tail)
	}
	want := chatBridgeAna + " -> " + chatBridgeBob + " [mentioned you]: line 2"
	if !strings.Contains(tail[0], want) {
		t.Errorf("tail read = %q, want it to carry %q", tail[0], want)
	}

	if got := runChat(t, cfg, "tok-bob", "read"); got != "no pending messages\n" {
		t.Errorf("unread read after the tail = %q, want %q", got, "no pending messages\n")
	}
}

// --to narrows a read to the messages naming a role, and a message naming two
// roles names each of them: a reader filtering by one of a pair is shown the
// message that asked them both, and not the one that asked only the other.
//
// Narrowing is not reading away, though — the count of what is still waiting
// says so, and the next unfiltered read hands it over.
func TestChat_ReadToKeepsOnlyWhatNamesTheRole(t *testing.T) {
	cfg := startChatDaemon(t)
	attendChatBridges(t, cfg, "tok-ana", "tok-bob", "tok-cid")
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeCid, 30*time.Second)

	runChat(t, cfg, "tok-bob", "send", chatBridgeAna+","+chatBridgeCid, "the pair owns the rebase")
	runChat(t, cfg, "tok-bob", "send", chatBridgeCid, "and cid alone owns the deploy")

	got := lines(runChat(t, cfg, "tok-cid", "read",
		"--cursor", "head", "--range", "100", "--to", chatBridgeAna))
	if len(got) != 2 {
		t.Fatalf("filtered read = %v, want the message naming ana and the count left over", got)
	}
	if want := "the pair owns the rebase"; !strings.Contains(got[0], want) {
		t.Errorf("filtered read = %q, want it to carry %q", got[0], want)
	}
	if got[1] != "1 more unread" {
		t.Errorf("filtered read reported %q, want %q", got[1], "1 more unread")
	}

	if out := runChat(t, cfg, "tok-cid", "read"); !strings.Contains(
		out, "and cid alone owns the deploy") {
		t.Errorf("read after the filtered one = %q, want the message the filter left", out)
	}
}

// chatNudgeKeys is the pair of `cmdman send-keys` invocations one nudge makes:
// the line typed into the recipient's terminal, then the Enter that submits it.
// token names the command being typed into and from is the sender the line
// names.
func chatNudgeKeys(token, from string) []string {
	return []string{
		token + " [crabswarm chat] new message from " + from +
			" — read it with the chat_read tool and respond with chat_send," +
			" both from crabswarm-mcp",
		token + " Enter",
	}
}

// everyone is the whole room minus the one saying it: a sender already knows
// what it said, so the message is neither unread for it nor typed into its
// terminal. Everyone else has it waiting, and the agents among them are woken
// for it — a person is delivered to and never typed at, since they read their
// room when they choose to.
//
// A list of roles is the same rule read the other way: it reaches exactly the
// roles it names, and wakes the agents among those.
func TestChat_EveryoneReachesTheRoomButItsSenderAndNudgesOnlyAgents(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)
	attendChatBridges(t, cfg, "tok-ana", "tok-bob", "tok-cid")
	waitChatRosterHas(t, cfg, "tok-ana", chatBridgeBob, 30*time.Second)
	waitChatRosterHas(t, cfg, "tok-ana", chatBridgeCid, 30*time.Second)
	human := registerChatHuman(t, cfg, identity, chatRoom, "humans", "yuki")

	if got := runChat(t, cfg, "tok-ana", "send", "everyone", "standup in five"); got != "" {
		t.Errorf("send to everyone = %q, want nothing: it names no role in particular", got)
	}

	// Typed at: the attending agents, in the order the room lists them. Not the
	// sender, not the person.
	typed := slices.Concat(
		chatNudgeKeys("tok-bob", chatBridgeAna),
		chatNudgeKeys("tok-cid", chatBridgeAna))
	if got := stubSendKeys(t, cfg); !slices.Equal(got, typed) {
		t.Errorf("cmdman send-keys invocations =\n%q\nwant\n%q", got, typed)
	}

	if got := runChat(t, cfg, "tok-ana", "read"); got != "no pending messages\n" {
		t.Errorf("the sender's read = %q, want nothing: it already knows what it said", got)
	}
	for _, reader := range []struct{ address, token string }{
		{chatBridgeBob, "tok-bob"},
		{chatBridgeCid, "tok-cid"},
		{"humans/yuki", human},
	} {
		body := chatMessageBody(t, runChat(t, cfg, reader.token, "read"))
		if want := chatBridgeAna + " -> everyone [mentioned you]: standup in five"; body != want {
			t.Errorf("read as %s = %q, want %q", reader.address, body, want)
		}
	}

	// A list of roles mentions each of them and nobody else, and the person
	// named beside the agent is delivered to without a keystroke.
	got := runChat(t, cfg, "tok-ana", "send", chatBridgeBob+",humans/yuki", "you two own it")
	if want := "mentioned " + chatBridgeBob + "\nmentioned humans/yuki\n"; got != want {
		t.Errorf("send to a list of roles = %q, want %q", got, want)
	}
	typed = append(typed, chatNudgeKeys("tok-bob", chatBridgeAna)...)
	if keys := stubSendKeys(t, cfg); !slices.Equal(keys, typed) {
		t.Errorf("cmdman send-keys invocations =\n%q\nwant\n%q", keys, typed)
	}
	if out := runChat(t, cfg, "tok-cid", "read"); out != "no pending messages\n" {
		t.Errorf("read as %s = %q, want nothing: cid was not named", chatBridgeCid, out)
	}
	if out := runChat(t, cfg, human, "read"); !strings.Contains(out, "you two own it") {
		t.Errorf("the person's read = %q, want the message nobody typed", out)
	}
}

// The cap the host writes into chat.history_limit reaches the room the members
// talk in: a daemon told to keep three messages answers with the three newest,
// however many were said. The environment spelling of the same setting,
// $CRABSWARM_CHAT_HISTORY_LIMIT, lands on the field this config file sets, and
// the config layers pin that separately.
func TestChat_ConfiguredHistoryLimitBoundsTheRoom(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonKeeping(t, 3, defaultStubCommands(), recipient)
	attendChatBridges(t, cfg, "tok-ana", "tok-bob")
	waitChatRosterHas(t, cfg, "tok-ana", chatBridgeBob, 30*time.Second)

	for i := range 5 {
		runChat(t, cfg, "tok-ana", "send", "agent-tok-bob", fmt.Sprintf("line %d", i))
	}

	got := lines(runChat(t, cfg, "", "admin", "log", chatRoom,
		"--cursor", "head", "--range", "100", "--identity", identity))
	if len(got) != 3 {
		t.Fatalf("admin log = %v, want the three messages the cap keeps", got)
	}
	for i, want := range []string{"line 2", "line 3", "line 4"} {
		suffix := chatBridgeAna + " -> " + chatBridgeBob + ": " + want
		if !strings.Contains(got[i], suffix) {
			t.Errorf("message %d = %q, want it to carry %q", i, got[i], suffix)
		}
	}
}

// A human is registered by the host rather than vouched for by cmdman, and
// takes part with the token that registration printed. No provider can ever
// resolve that token — which must not cost the human their place in the room.
//
// The attendance registration opens is the one no stream holds, so it ends the
// only way it can: with the daemon it lives in. The person registers again
// afterwards and picks the conversation up where they left it, because the role
// and its read position were in the database all along.
func TestChat_RegisteredHumanOutlivesShellsButNotARestart(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := writeChatConfig(t, 0, defaultStubCommands(), recipient)
	serve := startChatServe(t, cfg)
	attendChatBridges(t, cfg, "tok-ana")

	token := registerChatHuman(t, cfg, identity, chatRoom, "humans", "yuki")

	// Registration is attendance, held by nothing: the room lists the human
	// straight away, with no stream of their own behind it.
	members := memberAddresses(runChat(t, cfg, "tok-ana", "members"))
	slices.Sort(members)
	if want := []string{chatBridgeAna, "humans/yuki"}; !slices.Equal(members, want) {
		t.Errorf("members = %v, want %v", members, want)
	}

	// The human writes with the token in the environment, the way a shell that
	// ran `chat admin register` holds it.
	stdout, stderr, err := execChatTokenEnv(t, cfg, token,
		"send", "agent-tok-ana", "from the host")
	if err != nil {
		t.Fatalf("send with $%s: %v\nstderr:\n%s", chatTokenEnvVar, err, stderr)
	}
	if want := "mentioned " + chatBridgeAna + "\n"; stdout != want {
		t.Errorf("send with $%s = %q, want %q", chatTokenEnvVar, stdout, want)
	}
	body := chatMessageBody(t, runChat(t, cfg, "tok-ana", "read"))
	if want := "humans/yuki -> " + chatBridgeAna + " [mentioned you]: from the host"; body != want {
		t.Errorf("agent read = %q, want %q", body, want)
	}

	// And reads what the room sends back.
	runChat(t, cfg, "tok-ana", "send", "humans/yuki", "welcome")
	body = chatMessageBody(t, runChat(t, cfg, token, "read"))
	if want := chatBridgeAna + " -> humans/yuki [mentioned you]: welcome"; body != want {
		t.Errorf("human read = %q, want %q", body, want)
	}

	// The attendance outlives every command the person runs: it is held by the
	// registration rather than by a session, so the shell may come and go.
	for range 3 {
		runChat(t, cfg, token, "report-state", "done")
		if got := runChat(t, cfg, token, "read"); got != "no pending messages\n" {
			t.Errorf("human read = %q, want an empty inbox", got)
		}
		if got := memberAddresses(runChat(t, cfg, token, "members")); !slices.Contains(
			got, "humans/yuki") {
			t.Errorf("members seen by the human = %v, want it to still list humans/yuki", got)
		}
	}
	seenByAgent := memberAddresses(runChat(t, cfg, "tok-ana", "members"))
	if !slices.Contains(seenByAgent, "humans/yuki") {
		t.Errorf("members = %v, want the human still attending", seenByAgent)
	}

	// The restart is where it ends. Nothing held that attendance open, so there
	// is nothing to re-open it: the topology no longer carries the person, and
	// the token they were given answers for nobody.
	restartChatDaemon(t, cfg, serve)
	if listed := runChat(t, cfg, "", "admin", "list", "--identity", identity); strings.Contains(
		listed, "yuki") {
		t.Errorf("admin list after the restart =\n%s\nwant the person gone with the daemon",
			listed)
	}
	if _, stderr, err := execChatTokenEnv(t, cfg, token, "read"); err == nil {
		t.Error("the old token still read the room, want it to answer for nobody")
	} else if !strings.Contains(stderr, "not attending") {
		t.Errorf("stderr = %q, want it to say the token attends nothing", stderr)
	}

	// The role is not gone, though — the room has carried it — so it is still
	// addressable, and the send says nobody is there to be woken by it.
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)
	stdout, stderr, err = execChat(t, cfg, "tok-ana", "send", "humans/yuki", "the deploy is yours")
	if err != nil {
		t.Fatalf("send to the unregistered person: %v\nstderr:\n%s", err, stderr)
	}
	if want := "mentioned humans/yuki\n"; stdout != want {
		t.Errorf("send = %q, want %q", stdout, want)
	}
	if want := "warning: humans/yuki is not attending; the mention waits\n"; stderr != want {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}

	// Registering the same role again picks up its read position rather than
	// starting one: what waited is unread exactly once, and what the person read
	// before the restart stays read.
	again := registerChatHuman(t, cfg, identity, chatRoom, "humans", "yuki")
	body = chatMessageBody(t, runChat(t, cfg, again, "read"))
	if want := chatBridgeAna +
		" -> humans/yuki [mentioned you]: the deploy is yours"; body != want {
		t.Errorf("read after registering again = %q, want %q", body, want)
	}
	if got := runChat(t, cfg, again, "read"); got != "no pending messages\n" {
		t.Errorf("second read = %q, want the mention handed over once", got)
	}
	if listed := runChat(t, cfg, "", "admin", "list", "--identity", identity); !strings.Contains(
		listed, "yuki  human") {
		t.Errorf("admin list =\n%s\nwant the person back in the room", listed)
	}
}

// The admin verbs are gated by possession of the age identity file the daemon
// encrypts its challenge to, proven per call: the right file reads the whole
// topology, another key reads nothing, and no file at all is refused before the
// daemon is even dialed.
func TestChat_AdminIdentityGatesRoomList(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)
	attendChatBridges(t, cfg, "tok-ana")
	registerChatHuman(t, cfg, identity, chatRoom, "humans", "yuki")

	// The tree names each member's kind beside it, so the operator reading the
	// topology can tell which members a message reaches on its own.
	got := runChat(t, cfg, "", "admin", "list", "--identity", identity)
	for _, want := range []string{
		"room: " + chatRoom, "team: alpha", "agent-tok-ana  agent",
		"team: humans", "yuki  human",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("admin list missing %q; got:\n%s", want, got)
		}
	}

	// A challenge is spent by the call that answers it, so the second listing
	// runs a whole round of its own rather than reusing the first one's nonce.
	if second := runChat(t, cfg, "", "admin", "list", "--identity", identity); second != got {
		t.Errorf("second admin list = %q, want the same listing as the first %q", second, got)
	}

	// Another key cannot read the challenge, so it cannot answer it.
	other, _ := newChatIdentityFile(t)
	_, stderr, err := execChat(t, cfg, "", "admin", "list", "--identity", other)
	if err == nil {
		t.Fatal("admin list with the wrong identity succeeded, want a failure")
	}
	if !strings.Contains(stderr, "decrypting the admin challenge") {
		t.Errorf("stderr = %q, want it to name the failed decryption", stderr)
	}

	// With no identity configured or passed, the CLI says which one it wants.
	_, stderr, err = execChat(t, cfg, "", "admin", "list")
	if err == nil {
		t.Fatal("admin list without an identity succeeded, want a failure")
	}
	if !strings.Contains(stderr, "no admin age identity file") {
		t.Errorf("stderr = %q, want it to ask for an identity file", stderr)
	}
}

// The host speaks into a room it does not attend: the message mentions the
// roles it names under the reserved "admin" sender, everyone reaches the whole
// room, an empty target is a board post that mentions nobody, and none of it
// leaves a member behind for the room to talk back to.
func TestChat_AdminSendsWithoutAttending(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)
	attendChatBridges(t, cfg, "tok-ana", "tok-bob", "tok-cid")
	waitChatRosterHas(t, cfg, "tok-ana", chatBridgeBob, 30*time.Second)
	waitChatRosterHas(t, cfg, "tok-ana", chatBridgeCid, 30*time.Second)

	// A named role is addressed the way `chat send` addresses one, and the
	// answer names it back.
	got := runChat(t, cfg, "", "admin", "send", chatRoom, chatBridgeAna,
		"deploy is frozen", "--identity", identity)
	if want := "mentioned " + chatBridgeAna + "\n"; got != want {
		t.Errorf("admin send = %q, want %q", got, want)
	}

	// The message names the host rather than a peer: the sender carries no team
	// at all, which no member of a room ever does.
	body := chatMessageBody(t, runChat(t, cfg, "tok-ana", "read"))
	if want := "admin -> " + chatBridgeAna + " [mentioned you]: deploy is frozen"; body != want {
		t.Errorf("read = %q, want %q", body, want)
	}
	if got := runChat(t, cfg, "tok-bob", "read"); got != "no pending messages\n" {
		t.Errorf("bystander read = %q, want nothing: ana was addressed, not bob", got)
	}

	// Speaking into the room did not join it.
	members := memberAddresses(runChat(t, cfg, "tok-bob", "members"))
	slices.Sort(members)
	if want := []string{chatBridgeAna, chatBridgeBob, chatBridgeCid}; !slices.Equal(
		members, want) {
		t.Errorf("members after the admin send = %v, want %v", members, want)
	}
	for _, m := range members {
		if strings.Contains(m, "admin") {
			t.Errorf("members = %v, want no member row for the admin sender", members)
		}
	}

	// everyone is the whole room, across teams, and names no role in particular.
	if got := runChat(t, cfg, "", "admin", "send", chatRoom, "everyone",
		"standup in five", "--identity", identity); got != "" {
		t.Errorf("admin send to everyone = %q, want nothing", got)
	}
	for _, token := range []string{"tok-ana", "tok-bob", "tok-cid"} {
		read := runChat(t, cfg, token, "read")
		if !strings.Contains(read, "admin -> everyone") ||
			!strings.Contains(read, "standup in five") {
			t.Errorf("read as %s = %q, want the room-wide message", token, read)
		}
	}

	// A list of roles mentions each of them and nobody else.
	got = runChat(t, cfg, "", "admin", "send", chatRoom, chatBridgeAna+","+chatBridgeBob,
		"alpha owns the rebase", "--identity", identity)
	if want := "mentioned " + chatBridgeAna + "\nmentioned " + chatBridgeBob + "\n"; got != want {
		t.Errorf("admin send to a list of roles = %q, want %q", got, want)
	}
	for _, token := range []string{"tok-ana", "tok-bob"} {
		if read := runChat(t, cfg, token, "read"); !strings.Contains(
			read, "alpha owns the rebase") {
			t.Errorf("read as %s = %q, want the message addressed to the pair", token, read)
		}
	}
	if got := runChat(t, cfg, "tok-cid", "read"); got != "no pending messages\n" {
		t.Errorf("read as tok-cid = %q, want nothing: cid was not named", got)
	}

	// A board post is in the room and mentions nobody, so nothing is left
	// unread by it.
	if got := runChat(t, cfg, "", "admin", "send", chatRoom, "",
		"the room is quiet", "--identity", identity); got != "" {
		t.Errorf("admin board post = %q, want nothing", got)
	}
	if got := runChat(t, cfg, "tok-cid", "read"); got != "no pending messages\n" {
		t.Errorf("read after the board post = %q, want nothing unread", got)
	}

	// And send is gated by the identity file like every other admin verb: it is
	// refused before the daemon is dialed, naming the file it wants.
	_, stderr, err := execChat(t, cfg, "", "admin", "send", chatRoom, chatBridgeAna, "unsigned")
	if err == nil {
		t.Fatal("admin send without an identity succeeded, want a failure")
	}
	if !strings.Contains(stderr, "no admin age identity file") {
		t.Errorf("stderr = %q, want it to ask for an identity file", stderr)
	}
}

// The host reads a room it does not attend, and reads the whole of it: what the
// members said to each other and what the host itself said into the room, still
// there after every member has read everything addressed to it.
func TestChat_AdminReadsTheRoomLog(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)
	attendChatBridges(t, cfg, "tok-ana", "tok-bob")
	waitChatRosterHas(t, cfg, "tok-ana", chatBridgeBob, 30*time.Second)

	runChat(t, cfg, "", "admin", "send", chatRoom, chatBridgeAna,
		"deploy is frozen", "--identity", identity)
	runChat(t, cfg, "tok-ana", "send", "agent-tok-bob", "understood")
	runChat(t, cfg, "tok-bob", "send", "everyone", "back in five")

	// Nothing is unread anywhere from here on; the log is not a read position.
	for _, token := range []string{"tok-ana", "tok-bob"} {
		runChat(t, cfg, token, "read")
	}

	got := lines(runChat(t, cfg, "", "admin", "log", chatRoom,
		"--cursor", "head", "--range", "100", "--identity", identity))
	if len(got) != 3 {
		t.Fatalf("admin log = %v, want the three utterances of the room", got)
	}
	for i, want := range []string{
		"admin -> " + chatBridgeAna + ": deploy is frozen",
		chatBridgeAna + " -> " + chatBridgeBob + ": understood",
		chatBridgeBob + " -> everyone: back in five",
	} {
		if !strings.Contains(got[i], want) {
			t.Errorf("message %d = %q, want it to carry %q", i, got[i], want)
		}
	}
	// Every line opens with its place in the room, which is what --since and
	// --until take.
	if _, err := strconv.ParseInt(strings.Fields(got[0])[0], 10, 64); err != nil {
		t.Errorf("first message = %q, want it to open with a sequence number", got[0])
	}

	// Reading it again changes nothing: an admin read moves no position.
	if again := lines(runChat(t, cfg, "", "admin", "log", chatRoom,
		"--cursor", "head", "--range", "100", "--identity", identity)); !slices.Equal(
		again, got) {
		t.Errorf("second admin log = %v, want the same transcript as %v", again, got)
	}

	// The tail cursor takes the newest, and a room nothing was said in reports
	// itself rather than failing as an unknown one.
	if last := lines(runChat(t, cfg, "", "admin", "log", chatRoom,
		"--cursor", "tail", "--range", "-1", "--identity", identity)); !slices.Equal(
		last, got[2:]) {
		t.Errorf("admin log of the tail = %v, want the newest message %v", last, got[2:])
	}
	if silent := runChat(
		t, cfg, "", "admin", "log", "/work/nowhere", "--identity", identity,
	); silent != "no messages yet\n" {
		t.Errorf("admin log of a room nothing was said in = %q, want %q",
			silent, "no messages yet\n")
	}

	// The unread cursor belongs to a member read: an operator attends no room
	// and has no position to count unread from.
	_, stderr, err := execChat(t, cfg, "", "admin", "log", chatRoom,
		"--cursor", "unread", "--identity", identity)
	if err == nil {
		t.Fatal("admin log --cursor unread succeeded, want a failure")
	}
	if !strings.Contains(stderr, "unread") {
		t.Errorf("stderr = %q, want it to name the cursor it refuses", stderr)
	}

	// And it is gated by the identity file like every other admin verb.
	_, stderr, err = execChat(t, cfg, "", "admin", "log", chatRoom)
	if err == nil {
		t.Fatal("admin log without an identity succeeded, want a failure")
	}
	if !strings.Contains(stderr, "no admin age identity file") {
		t.Errorf("stderr = %q, want it to ask for an identity file", stderr)
	}
}

// A room is emptied by its sessions ending, which takes no verb: there is none
// to leave with, and a bridge that is simply gone is a member the room no
// longer has. Deleting is refused until then, because a room somebody is in is
// not a room the operator is done with.
func TestChat_DeleteRoomWaitsForTheRoomToEmpty(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)
	sessions := attendChatBridges(t, cfg, "tok-ana")
	runChat(t, cfg, "tok-ana", "send", "", "fyi: rebased main")

	// Refused while somebody is in it, and the refusal names who.
	_, stderr, err := execChat(t, cfg, "", "admin", "delete-room", chatRoom,
		"--identity", identity)
	if err == nil {
		t.Fatal("delete-room of an attended room succeeded, want a failure")
	}
	for _, want := range []string{chatBridgeAna, "room is attended"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr = %q, want it to carry %q", stderr, want)
		}
	}

	// The session ends. Nothing else happens — no restart, no cleanup verb —
	// and the room is empty.
	stopChatBridge(t, sessions[0])
	waitStubStatus(t, cfg, "delete tok-ana", 30*time.Second)

	listed := runChat(t, cfg, "", "admin", "list", "--identity", identity)
	if want := "room: " + chatRoom + "\n  (nobody attending)\n"; listed != want {
		t.Errorf("admin list = %q, want %q", listed, want)
	}

	// Now it goes, and takes the one thing said in it along.
	want := "deleted room " + chatRoom + " and 1 message\n"
	if got := runChat(t, cfg, "", "admin", "delete-room", chatRoom,
		"--identity", identity); got != want {
		t.Errorf("delete-room = %q, want %q", got, want)
	}
	if got := runChat(t, cfg, "", "admin", "list", "--identity", identity); got != "no rooms\n" {
		t.Errorf("admin list after the deletion = %q, want %q", got, "no rooms\n")
	}
}

// Nothing but `admin delete-room` removes a room. Its row is made by the first
// attendance or message in it and outlives both: the session that made it ends,
// the history cap throws the conversation away message by message, the daemon
// restarts — and the operator still finds the room, with nobody in it. That is
// what makes it findable at all, since the read positions a returning role
// reads from hang off exactly that row.
func TestChat_ARoomOutlivesItsMembersAndItsLog(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	// A cap of one: every send prunes everything below the message it just
	// appended, so what a room of three utterances keeps is the last of them.
	cfg := writeChatConfig(t, 1, defaultStubCommands(), recipient)
	serve := startChatServe(t, cfg)
	sessions := attendChatBridges(t, cfg, "tok-ana")

	for i := range 3 {
		runChat(t, cfg, "tok-ana", "send", "", fmt.Sprintf("note %d", i))
	}
	stopChatBridge(t, sessions[0])
	waitStubStatus(t, cfg, "delete tok-ana", 30*time.Second)

	restartChatDaemon(t, cfg, serve)

	listed := runChat(t, cfg, "", "admin", "list", "--identity", identity)
	if want := "room: " + chatRoom + "\n  (nobody attending)\n"; listed != want {
		t.Errorf("admin list after the restart = %q, want %q", listed, want)
	}

	logged := lines(runChat(t, cfg, "", "admin", "log", chatRoom,
		"--cursor", "head", "--range", "100", "--identity", identity))
	if len(logged) != 1 || !strings.Contains(logged[0], "note 2") {
		t.Fatalf("admin log = %v, want only the newest message the cap kept", logged)
	}

	want := "deleted room " + chatRoom + " and 1 message\n"
	if got := runChat(t, cfg, "", "admin", "delete-room", chatRoom,
		"--identity", identity); got != want {
		t.Errorf("delete-room = %q, want %q", got, want)
	}
	if got := runChat(t, cfg, "", "admin", "list", "--identity", identity); got != "no rooms\n" {
		t.Errorf("admin list after the deletion = %q, want %q", got, "no rooms\n")
	}
}

// The state an agent reports reaches cmdman's status display, which is where an
// operator watching their commands sees it. The daemon drives the real CLI
// surface — `cmdman status set <state> <id> --detail ...` and
// `cmdman status delete <id>` — so the stub records invocations verbatim rather
// than the test asserting against a Go paraphrase of them.
func TestChat_MirrorsMemberStateOntoCmdmanStatus(t *testing.T) {
	cfg := startChatDaemon(t)
	sessions := attendChatBridges(t, cfg, "tok-ana")

	runChat(t, cfg, "tok-ana", "report-state", "working")
	runChat(t, cfg, "tok-ana", "report-state", "waiting")

	// Ending the session ends the attendance: the stream is the membership, so
	// the daemon withdraws the published state as the bridge goes.
	stopChatBridge(t, sessions[0])
	waitStubStatus(t, cfg, "delete tok-ana", 30*time.Second)

	got := stubStatus(t, cfg)
	want := []string{
		// A fresh member is done: attendance is declared before there is work.
		"set done tok-ana --detail crabswarm chat",
		"set working tok-ana --detail crabswarm chat",
		"set waiting tok-ana --detail crabswarm chat",
		"delete tok-ana",
	}
	if !slices.Equal(got, want) {
		t.Errorf("cmdman status invocations =\n%q\nwant\n%q", got, want)
	}
}

// Typing into a terminal is what the agent kind asks for, and the only thing it
// changes. A registered human is delivered to like anyone else and simply never
// typed at: they read their room when they choose to, which is what a person at
// a shell wants and what a harness cannot wait for.
//
// Two lines per nudge is the injection itself — the text, then the Enter that
// submits it — read off the same stub cmdman the status cases use.
func TestChat_OnlyAnAgentIsTypedAt(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)
	attendChatBridges(t, cfg, "tok-ana")
	token := registerChatHuman(t, cfg, identity, chatRoom, "humans", "yuki")

	if _, stderr, err := execChatTokenEnv(t, cfg, token,
		"send", "agent-tok-ana", "PR is ready"); err != nil {
		t.Fatalf("the human's send failed: %v\nstderr:\n%s", err, stderr)
	}
	runChat(t, cfg, "tok-ana", "send", "humans/yuki", "looking now")

	got := stubSendKeys(t, cfg)
	want := []string{
		"tok-ana [crabswarm chat] new message from humans/yuki" +
			" — read it with the chat_read tool and respond with chat_send," +
			" both from crabswarm-mcp",
		"tok-ana Enter",
	}
	if !slices.Equal(got, want) {
		t.Errorf("cmdman send-keys invocations =\n%q\nwant\n%q", got, want)
	}

	// The human was not woken, but the message is theirs all the same.
	if out := runChat(t, cfg, token, "read"); !strings.Contains(out, "looking now") {
		t.Errorf("the human's read = %q, want it to carry the message nobody typed", out)
	}
}

// A human is registered by the host, so their token is the daemon's own secret
// and names no cmdman command. It must never reach a cmdman command line, not
// even to be rejected there.
func TestChat_NeverPublishesAHumanToken(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)

	token := registerChatHuman(t, cfg, identity, chatRoom, "humans", "yuki")
	runChat(t, cfg, token, "report-state", "working")

	// An agent doing the same thing proves the recording works at all, so the
	// human half below cannot pass by nothing being recorded.
	attendChatBridges(t, cfg, "tok-ana")

	published := stubStatus(t, cfg)
	if want := []string{
		"set done tok-ana --detail crabswarm chat",
	}; !slices.Equal(published, want) {
		t.Errorf("cmdman status invocations = %q, want only the agent's %q", published, want)
	}
	for _, line := range published {
		if strings.Contains(line, token) {
			t.Errorf("cmdman status invocation %q carries the human's token", line)
		}
	}
}

// The whole path a configured harness takes: two agents, each with a bridge of
// its own started from the command line the apm package declares, talking
// through the tools alone. What a tool hands back is compared against what the
// CLI verb prints for the same message, since a member wired through MCP is
// meant to read its room in the same words as one typing commands.
func TestChat_BridgeToolsCarryTheRoom(t *testing.T) {
	cfg := startChatDaemon(t)
	sessions := attendChatBridges(t, cfg, "tok-ana", "tok-bob")
	ana, bob := sessions[0], sessions[1]
	waitChatRosterHas(t, cfg, "tok-ana", chatBridgeBob, 30*time.Second)

	members := memberAddresses(callChatTool(t, bob, "chat_members", nil))
	slices.Sort(members)
	if want := []string{chatBridgeAna, chatBridgeBob}; !slices.Equal(members, want) {
		t.Errorf("chat_members = %v, want %v", members, want)
	}

	// A bare name resolves inside the sender's own team, and the tool reports
	// the role it resolved to.
	got := callChatTool(t, ana, "chat_send", map[string]any{
		"to": "agent-tok-bob", "message": "the bridge is up",
	})
	if want := "mentioned " + chatBridgeBob + "\n"; got != want {
		t.Errorf("chat_send = %q, want %q", got, want)
	}

	throughBridge := chatMessageBody(t, callChatTool(t, bob, "chat_read", nil))
	want := chatBridgeAna + " -> " + chatBridgeBob + " [mentioned you]: the bridge is up"
	if throughBridge != want {
		t.Errorf("chat_read = %q, want %q", throughBridge, want)
	}

	// The same message read the other way. The second copy is sent after the
	// first read rather than beside it: a read moves the position past
	// everything it showed, and would have taken both.
	callChatTool(t, ana, "chat_send", map[string]any{
		"to": "agent-tok-bob", "message": "the bridge is up",
	})
	throughCLI := chatMessageBody(t, runChat(t, cfg, "tok-bob", "read"))
	if throughCLI != throughBridge {
		t.Errorf("`chat read` printed %q, want the tool's %q", throughCLI, throughBridge)
	}

	// The attendance the bridges hold is the daemon's, not something the tools
	// keep between themselves: a member verb typed at the CLI, which the daemon
	// answers for members alone, reads back the same room.
	roster := memberAddresses(runChat(t, cfg, "tok-ana", "members"))
	slices.Sort(roster)
	if want := []string{chatBridgeAna, chatBridgeBob}; !slices.Equal(roster, want) {
		t.Errorf("members = %v, want %v", roster, want)
	}
}

// A harness starts its MCP subprocesses before the services they talk to, so a
// bridge regularly comes up against a socket nothing is listening on. It keeps
// asking, and the room holds the member shortly after the daemon is there —
// with no tool call to prompt it, since the agent has no reason to make one and
// its hooks are refused until it does.
func TestChat_BridgeAttendsOnceTheDaemonComesUp(t *testing.T) {
	cfg := writeChatConfig(t, 0, defaultStubCommands())

	// The handshake is answered by a bridge with no daemon to attend.
	bridge := startChatBridge(t, cfg, "tok-ana")
	if got := bridge.InitializeResult().ServerInfo.Name; got != "crabswarm-mcp" {
		t.Errorf("bridge announced itself as %q, want %q", got, "crabswarm-mcp")
	}

	startChatServe(t, cfg)

	// Watched from another member rather than through the bridge itself, so
	// what passes here is the attendance the bridge opens on its own.
	attendChatBridges(t, cfg, "tok-bob")
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)
}

// A daemon can go away and come back under a live bridge — restarted by its
// operator while every agent keeps running. The bridge attends again on its
// own, which is what the agent's hooks need: they report harness state through
// the CLI, and every report is refused for as long as the room is missing the
// member.
//
// What was waiting is still waiting afterwards. Attendance is held in memory
// and dies with the daemon, but the role and its read position are in the
// database, so a mention sent before the restart is unread exactly once after
// it — which is the whole reason a restart is survivable rather than a reset.
func TestChat_BridgeReattendsAfterTheDaemonRestarts(t *testing.T) {
	cfg := writeChatConfig(t, 0, defaultStubCommands())
	serve := startChatServe(t, cfg)
	attendChatBridges(t, cfg, "tok-ana", "tok-bob")
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)
	runChat(t, cfg, "tok-bob", "send", "agent-tok-ana", "look at the migration")

	restartChatDaemon(t, cfg, serve)

	// Attendance read through a member verb, which the daemon answers for
	// members alone. Nothing has called a tool on the bridge.
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)

	body := chatMessageBody(t, runChat(t, cfg, "tok-ana", "read"))
	want := chatBridgeBob + " -> " + chatBridgeAna + " [mentioned you]: look at the migration"
	if body != want {
		t.Errorf("read after the restart = %q, want %q", body, want)
	}
	if got := runChat(t, cfg, "tok-ana", "read"); got != "no pending messages\n" {
		t.Errorf("second read = %q, want the mention handed over once", got)
	}

	// The hook path is the other half: a state report reaches cmdman's status
	// display again, on the member the bridge attended for a second time.
	runChat(t, cfg, "tok-ana", "report-state", "working")
	waitStubStatus(t, cfg, "set working tok-ana --detail crabswarm chat", 30*time.Second)
}

// A harness that strips the environment leaves the bridge with no identity to
// resolve: no --token, no $CRABSWARM_CHAT_TOKEN and no $CMDMAN_CMD_ID. It
// answers the handshake all the same — a subprocess that exits during startup
// reaches the user as a closed connection with no reason attached — and its
// tools report what is missing in the words every member verb uses.
func TestChat_BridgeWithoutAnIdentityStillServes(t *testing.T) {
	cfg := writeChatConfig(t, 0, defaultStubCommands())
	startChatServe(t, cfg)

	// What a stripped harness leaves behind: enough to find a home directory
	// and a binary, and nothing that names a member. No $XDG_RUNTIME_DIR
	// either, which is why the daemon socket comes from --config.
	session := startChatBridgeIn(t, cfg, "", []string{
		"HOME=" + os.Getenv("HOME"),
		"PATH=" + os.Getenv("PATH"),
	})
	if got := session.InitializeResult().ServerInfo.Name; got != "crabswarm-mcp" {
		t.Errorf("bridge announced itself as %q, want %q", got, "crabswarm-mcp")
	}

	text, failed := chatToolResult(t, session, "chat_members", nil)
	if !failed {
		t.Fatalf("chat_members answered %q, want it to report the missing identity", text)
	}
	for _, want := range []string{
		"no chat identity token", "--token", chatTokenEnvVar, "CMDMAN_CMD_ID",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("chat_members = %q, want it to name %q", text, want)
		}
	}
}

// A harness that spawns the bridge with a fixed environment whitelist forwards
// only the variables its own MCP configuration names, which is what the apm
// package's env_vars list is for. Handed those three and nothing else, the
// bridge resolves both halves of what it needs by itself: an identity token,
// and — with no --config and no --sock — the daemon socket under
// $XDG_RUNTIME_DIR. So it attends and its tools answer for the room.
func TestChat_BridgeFindsTheDaemonThroughTheRuntimeDir(t *testing.T) {
	// The runtime dir is made short on purpose: a Unix socket path is bounded
	// at a little over a hundred bytes, and a t.TempDir() spends much of that
	// on the test's own name before the derived crabswarm/default.sock is
	// appended.
	runtimeDir, err := os.MkdirTemp("", "crabswarm-rt")
	if err != nil {
		t.Fatalf("make a runtime dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(runtimeDir) })

	// Where the bridge derives the socket from $XDG_RUNTIME_DIR, and therefore
	// where the daemon has to listen for the derivation to be worth anything.
	// Its crabswarm/ directory is left for the daemon to create, as on a fresh
	// boot.
	sock := filepath.Join(runtimeDir, "crabswarm", "default.sock")
	cfg := writeChatConfigOn(t, t.TempDir(), sock, 0, defaultStubCommands())
	startChatServeOn(t, cfg, sock)

	// A home of the test's own rather than the suite's: the bridge reads a
	// config file out of $HOME when one is there, and a developer's host may
	// hold one naming a socket of its own — which would answer the question
	// this case is asking. Everything else is what the harness forwards, with
	// the token in the spelling an agent under cmdman inherits.
	session := startChatBridgeIn(t, "", "", []string{
		"HOME=" + t.TempDir(),
		"PATH=" + os.Getenv("PATH"),
		"XDG_RUNTIME_DIR=" + runtimeDir,
		"CMDMAN_CMD_ID=tok-ana",
		chatTokenEnvVar + "=",
	})

	// Attendance is read through a member verb, not a tool: a tool waits for
	// the bridge's own attendance, so it would report from inside the process
	// being asked about.
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)

	members := memberAddresses(callChatTool(t, session, "chat_members", nil))
	if want := []string{chatBridgeAna}; !slices.Equal(members, want) {
		t.Errorf("chat_members = %v, want %v", members, want)
	}
}
