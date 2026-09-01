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
// command and scaleIndex are the other two compose labels, the ones a joiner
// that names itself nothing is named after: the name the compose file declares
// the command under and the replica index that tells one instance of a scaled
// command from another. Either may be empty, which is a command whose labels do
// not say.
type stubCommand struct {
	token      string
	dir        string
	project    string
	command    string
	scaleIndex string
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

// stubCmdmanScript renders a stub cmdman answering the two surfaces the chat
// broker uses: `inspect <ID> --format '{{json .Config}}'` for placement, and
// `status set|delete` for the state display. Each token in commands reports its
// own directory and compose project; an unknown ID fails the way cmdman fails
// on one, since the daemon reads that exact wording as "this token names
// nothing" rather than "the lookup broke".
//
// Status invocations are recorded beside the stub rather than answered, so a
// test can read back what the daemon published. The stub finds the log relative
// to $0, which keeps it independent of the environment the daemon runs it with.
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
if [ "$1" != "inspect" ]; then
	echo "stub cmdman: unsupported invocation: $*" >&2
	exit 1
fi
case "$2" in
`)
	for _, c := range commands {
		fmt.Fprintf(&b, "\t%s) dir=%s; labels='%s' ;;\n", c.token, c.dir, stubLabels(c))
	}
	b.WriteString(`	*)
		echo "error: resolve command: no command found matching \"$2\"" >&2
		exit 1
		;;
esac
printf '{"dir":"%s","labels":%s}\n' "$dir" "$labels"
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
	b, err := os.ReadFile(filepath.Join(filepath.Dir(cfgPath), "status.log"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("reading stub status log: %v", err)
	}
	var lines []string
	for l := range strings.SplitSeq(strings.TrimSpace(string(b)), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

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
	dir := t.TempDir()

	stub := filepath.Join(dir, "cmdman")
	writeFile(t, stub, stubCmdmanScript(commands))
	if err := os.Chmod(stub, 0o755); err != nil {
		t.Fatalf("chmod stub cmdman: %v", err)
	}

	recipients, err := json.Marshal(adminRecipients)
	if err != nil {
		t.Fatalf("marshal admin recipients: %v", err)
	}

	sock := filepath.Join(dir, "chat.sock")
	cfgPath := filepath.Join(dir, "config.json")
	writeFile(t, cfgPath, fmt.Sprintf(
		`{"sock":%q,"chat":{"db":%q,"cmdman_bin":%q,"admin_recipients":%s}}`,
		sock, filepath.Join(dir, "chat.db"), stub, recipients))

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
// printed, which is the only time the daemon reveals it.
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

// The addresses the bridge cases below spell. A bridge joins with no name at
// all — an agent is named by whoever registered it, not by the harness it runs
// under — so the daemon derives one from the token, and that derivation is what
// makes these addresses writable in a test.
const (
	chatBridgeAna = "alpha/agent-tok-ana"
	chatBridgeBob = "alpha/agent-tok-bob"
)

// startChatBridge starts `crabswarm chat mcp` the way a configured harness
// does — as a stdio subprocess spoken to over MCP — and returns the session
// that harness would hold. Connecting is the handshake, so a bridge that failed
// to serve one fails the test here.
//
// The bridge is the only thing that ever declares this token's attendance: no
// case below runs `chat join` for a token it hands to one.
func startChatBridge(t *testing.T, cfgPath, token string) *mcp.ClientSession {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), crabswarmBin,
		"chat", "mcp", "--config", cfgPath, "--token", token)
	cmd.Env = chatEnviron()
	// Everything the bridge says goes to stderr by design, since stdout carries
	// the protocol; forwarding it is what makes a failing case readable.
	cmd.Stderr = os.Stderr

	client := mcp.NewClient(&mcp.Implementation{Name: "crabswarm-e2e", Version: "v0"}, nil)
	session, err := client.Connect(t.Context(), &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect to the chat bridge for %s: %v", token, err)
	}
	// Closing the session shuts the subprocess down the way the stdio transport
	// is meant to: stdin first, then a wait for the process to go.
	t.Cleanup(func() { _ = session.Close() })
	return session
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
	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	text := chatToolText(t, res)
	if res.IsError {
		t.Fatalf("%s reported an error: %s", name, text)
	}
	return text
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
// It is deliberately not a tool call: every tool declares attendance itself
// before it acts, so calling one would prove nothing about the join the bridge
// makes on its own — which is the only automatic join a consumer gets now that
// the session-start hook is gone.
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

// chatMessageBody strips the stamp off a rendered message and returns the rest.
// The instant is the one part of the rendering a case cannot pin, so it is
// checked for being an instant at all and then dropped.
func chatMessageBody(t *testing.T, rendered string) string {
	t.Helper()
	stamped, ok := strings.CutPrefix(rendered, "[")
	if !ok {
		t.Fatalf("read = %q, want it to open with a timestamp", rendered)
	}
	stamp, body, ok := strings.Cut(stamped, "] ")
	if !ok {
		t.Fatalf("read = %q, want a timestamped message line", rendered)
	}
	if _, err := time.Parse(time.RFC3339, stamp); err != nil {
		t.Fatalf("read = %q carries %q, which is not an RFC3339 instant: %v", rendered, stamp, err)
	}
	return body
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
	if got := runChat(t, cfg, "tok-ana", "report-state", "done"); got != "" {
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

// A command cmdman knows but that runs outside any compose project is refused
// just as an unknown token is: it has a working directory, so it could be
// placed in a room, but nothing says which team it coordinates with.
func TestChat_NonComposeTokenIsRejected(t *testing.T) {
	cfg := startChatDaemonWith(t, []stubCommand{
		{token: "tok-loner", dir: chatRoom},
	})

	stdout, stderr, err := execChat(t, cfg, "tok-loner", "join", "--name", "loner")
	if err == nil {
		t.Fatal("join without a compose project succeeded, want a failure")
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want nothing", stdout)
	}
	if !strings.Contains(stderr, "not part of a compose project") {
		t.Errorf("stderr = %q, want it to name what the token is missing", stderr)
	}
}

// A joiner that names itself nothing is named after the compose labels of the
// command it runs under: the declared command name, suffixed with the replica
// index that tells one instance of a scaled command apart from its siblings. A
// compose author therefore addresses an agent by the name their compose file
// already gives it, without every command template having to pass --name.
func TestChat_JoinWithoutNameTakesComposeLabels(t *testing.T) {
	cfg := startChatDaemonWith(t, []stubCommand{
		{token: "tok-worker", dir: chatRoom, project: "alpha",
			command: "worker", scaleIndex: "2"},
		{token: "tok-solo", dir: chatRoom, project: "alpha", command: "solo"},
		// Exactly eight characters, which is as much of a token as a
		// token-derived name carries, so the fallback below is spelled out
		// whole.
		{token: "tok-bare", dir: chatRoom, project: "alpha"},
	})

	got := runChat(t, cfg, "tok-worker", "join")
	if want := "joined " + chatRoom + " as alpha/worker-2\n"; got != want {
		t.Errorf("join = %q, want %q", got, want)
	}

	// An unscaled command carries no replica index to append, so the declared
	// name stands on its own.
	got = runChat(t, cfg, "tok-solo", "join")
	if want := "joined " + chatRoom + " as alpha/solo\n"; got != want {
		t.Errorf("join of an unscaled command = %q, want %q", got, want)
	}

	// Nothing in the labels names this one, so the daemon falls back to the
	// token, as it did before the labels were read at all.
	got = runChat(t, cfg, "tok-bare", "join")
	if want := "joined " + chatRoom + " as alpha/agent-tok-bare\n"; got != want {
		t.Errorf("join without naming labels = %q, want %q", got, want)
	}

	// The derived names are the ones a teammate sees and addresses.
	members := lines(runChat(t, cfg, "tok-worker", "members"))
	slices.Sort(members)
	want := []string{"alpha/agent-tok-bare", "alpha/solo", "alpha/worker-2"}
	if !slices.Equal(members, want) {
		t.Errorf("members = %v, want %v", members, want)
	}

	// An explicit name still wins over the labels: the request is the first
	// thing consulted, not a default the labels override.
	cfg = startChatDaemonWith(t, []stubCommand{
		{token: "tok-worker", dir: chatRoom, project: "alpha",
			command: "worker", scaleIndex: "2"},
	})
	got = runChat(t, cfg, "tok-worker", "join", "--name", "chosen")
	if want := "joined " + chatRoom + " as alpha/chosen\n"; got != want {
		t.Errorf("join with an explicit name = %q, want %q", got, want)
	}
}

// One name carried by several teams of a room: a bare name means the caller's
// own teammate first, then the room-wide member when only one carries it, and
// is refused when two other teams both do.
func TestChat_NameCollisionAddressing(t *testing.T) {
	cfg := startChatDaemonWith(t, []stubCommand{
		{token: "tok-alpha-sam", dir: chatRoom, project: "alpha"},
		{token: "tok-beta-sam", dir: chatRoom, project: "beta"},
		{token: "tok-alpha-uniq", dir: chatRoom, project: "alpha"},
		{token: "tok-asker", dir: chatRoom, project: "gamma"},
		{token: "tok-gamma-sam", dir: chatRoom, project: "gamma"},
	})

	runChat(t, cfg, "tok-alpha-sam", "join", "--name", "sam")
	runChat(t, cfg, "tok-beta-sam", "join", "--name", "sam")
	runChat(t, cfg, "tok-alpha-uniq", "join", "--name", "uniq")
	runChat(t, cfg, "tok-asker", "join", "--name", "asker")

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
	if want := "sent to beta/sam\n"; got != want {
		t.Errorf("qualified send = %q, want %q", got, want)
	}
	got = runChat(t, cfg, "tok-beta-sam", "read")
	if !strings.Contains(got, "gamma/asker: for beta") {
		t.Errorf("beta/sam read = %q, want the qualified message", got)
	}
	if got := runChat(t, cfg, "tok-alpha-sam", "read"); got != "no pending messages\n" {
		t.Errorf("alpha/sam read = %q, want nothing: beta was addressed, not alpha", got)
	}

	// A bare name only one member of the room carries needs no qualification,
	// even across teams.
	got = runChat(t, cfg, "tok-asker", "send", "uniq", "hi")
	if want := "sent to alpha/uniq\n"; got != want {
		t.Errorf("unique bare send = %q, want %q", got, want)
	}

	// Once the caller's own team carries the name too, that one wins over the
	// other teams' — and over the ambiguity the same address gave a moment ago.
	runChat(t, cfg, "tok-gamma-sam", "join", "--name", "sam")
	got = runChat(t, cfg, "tok-asker", "send", "sam", "for my own team")
	if want := "sent to gamma/sam\n"; got != want {
		t.Errorf("own-team send = %q, want %q", got, want)
	}
	got = runChat(t, cfg, "tok-gamma-sam", "read")
	if !strings.Contains(got, "gamma/asker: for my own team") {
		t.Errorf("gamma/sam read = %q, want the own-team message", got)
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

	runChat(t, cfg, "tok-ana", "join", "--name", "ana")
	got := runChat(t, cfg, "tok-zed", "join", "--name", "zed")
	if want := "joined " + chatOtherRoom + " as alpha/zed\n"; got != want {
		t.Errorf("join = %q, want %q", got, want)
	}

	// Each side lists its own room only, though both are team alpha.
	if members := lines(runChat(t, cfg, "tok-ana", "members")); !slices.Equal(
		members, []string{"alpha/ana"}) {
		t.Errorf("members = %v, want only alpha/ana", members)
	}
	if members := lines(runChat(t, cfg, "tok-zed", "members")); !slices.Equal(
		members, []string{"alpha/zed"}) {
		t.Errorf("members of the other room = %v, want only alpha/zed", members)
	}

	// Neither spelling of the address reaches across.
	for _, addr := range []string{"zed", "alpha/zed"} {
		_, stderr, err := execChat(t, cfg, "tok-ana", "send", addr, "hello other room")
		if err == nil {
			t.Fatalf("send to %q in another room succeeded, want a failure", addr)
		}
		if !strings.Contains(stderr, "member not found") {
			t.Errorf("stderr for %q = %q, want an unresolved address", addr, stderr)
		}
	}

	// Nor does a broadcast, which addresses nobody by name at all.
	got = runChat(t, cfg, "tok-ana", "broadcast", "anyone?")
	if want := "broadcast to 0 members\n"; got != want {
		t.Errorf("broadcast = %q, want %q", got, want)
	}
	if got := runChat(t, cfg, "tok-zed", "read"); got != "no pending messages\n" {
		t.Errorf("read in the other room = %q, want nothing", got)
	}
}

// The MCP bridge attends as it starts, and a harness may start more than one of
// them against the same token, so joining twice is not an error: the second
// join is answered from the stored membership, leaving the name, the attendance
// and the inbox as they were.
func TestChat_JoinIsIdempotent(t *testing.T) {
	cfg := startChatDaemon(t)

	wantJoined := "joined " + chatRoom + " as alpha/ana\n"
	if got := runChat(t, cfg, "tok-ana", "join", "--name", "ana"); got != wantJoined {
		t.Errorf("first join = %q, want %q", got, wantJoined)
	}
	runChat(t, cfg, "tok-bob", "join", "--name", "bob")
	runChat(t, cfg, "tok-bob", "send", "ana", "before the second join")

	// Even a differently spelled name changes nothing: the first join settled
	// this token's identity.
	if got := runChat(t, cfg, "tok-ana", "join", "--name", "renamed"); got != wantJoined {
		t.Errorf("second join = %q, want %q", got, wantJoined)
	}

	members := lines(runChat(t, cfg, "tok-ana", "members"))
	slices.Sort(members)
	if want := []string{"alpha/ana", "alpha/bob"}; !slices.Equal(members, want) {
		t.Errorf("members = %v, want %v: the re-join must not add a member", members, want)
	}

	// The inbox survived: nothing waiting was dropped with the re-join.
	got := runChat(t, cfg, "tok-ana", "read")
	if !strings.Contains(got, "alpha/bob: before the second join") {
		t.Errorf("read = %q, want the message queued before the second join", got)
	}
}

// A human is registered by the host rather than vouched for by cmdman, and
// takes part with the token that registration printed. No provider can ever
// resolve that token — which must not cost the human their place in the room.
func TestChat_RegisteredHumanParticipates(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)

	runChat(t, cfg, "tok-ana", "join", "--name", "ana")
	token := registerChatHuman(t, cfg, identity, chatRoom, "humans", "yuki")

	// Registration is attendance: the room already lists the human.
	members := lines(runChat(t, cfg, "tok-ana", "members"))
	slices.Sort(members)
	if want := []string{"alpha/ana", "humans/yuki"}; !slices.Equal(members, want) {
		t.Errorf("members = %v, want %v", members, want)
	}

	// Joining as an already-registered human is answered from the store, so the
	// token the daemon minted needs no provider to back it.
	got := runChat(t, cfg, token, "join", "--name", "yuki")
	if want := "joined " + chatRoom + " as humans/yuki\n"; got != want {
		t.Errorf("human join = %q, want %q", got, want)
	}

	// The human writes with the token in the environment, the way a shell that
	// ran `chat admin register` holds it.
	stdout, stderr, err := execChatTokenEnv(t, cfg, token, "send", "ana", "from the host")
	if err != nil {
		t.Fatalf("send with $%s: %v\nstderr:\n%s", chatTokenEnvVar, err, stderr)
	}
	if want := "sent to alpha/ana\n"; stdout != want {
		t.Errorf("send with $%s = %q, want %q", chatTokenEnvVar, stdout, want)
	}
	agentRead := runChat(t, cfg, "tok-ana", "read")
	if !strings.Contains(agentRead, "humans/yuki: from the host") {
		t.Errorf("agent read = %q, want the human's message", agentRead)
	}

	// And reads what the room sends back.
	runChat(t, cfg, "tok-ana", "send", "humans/yuki", "welcome")
	if got := runChat(t, cfg, token, "read"); !strings.Contains(got, "alpha/ana: welcome") {
		t.Errorf("human read = %q, want the agent's message", got)
	}

	// Every one of these passes the daemon's liveness check, which for an agent
	// would ask the stub cmdman about the token and reap a member it does not
	// know. The human is left alone across all of them.
	for range 3 {
		runChat(t, cfg, token, "report-state", "done")
		if got := runChat(t, cfg, token, "read"); got != "no pending messages\n" {
			t.Errorf("human read = %q, want an empty inbox", got)
		}
		if got := lines(runChat(t, cfg, token, "members")); !slices.Contains(got, "humans/yuki") {
			t.Errorf("members seen by the human = %v, want it to still list humans/yuki", got)
		}
	}
	if got := lines(runChat(t, cfg, "tok-ana", "members")); !slices.Contains(got, "humans/yuki") {
		t.Errorf("members = %v, want the human still attending", got)
	}
}

// The admin verbs are gated by possession of the age identity file the daemon
// encrypts its challenge to, proven per call: the right file reads the whole
// topology, another key reads nothing, and no file at all is refused before the
// daemon is even dialed.
func TestChat_AdminIdentityGatesRoomList(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)

	runChat(t, cfg, "tok-ana", "join", "--name", "ana")
	runChat(t, cfg, "tok-cid", "join", "--name", "cid")

	got := runChat(t, cfg, "", "admin", "list", "--identity", identity)
	for _, want := range []string{"room: " + chatRoom, "team: alpha", "ana", "team: beta", "cid"} {
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

// Moving a member is an operator's edit to the room's team formation, and the
// room reads it back immediately: the listing shows the new team and addressing
// follows it, on both the new spelling and the stale one.
func TestChat_AdminMovesMemberBetweenTeams(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)

	runChat(t, cfg, "tok-ana", "join", "--name", "ana")
	runChat(t, cfg, "tok-cid", "join", "--name", "cid")

	got := runChat(t, cfg, "", "admin", "move", chatRoom, "alpha/ana", "beta",
		"--identity", identity)
	if want := "moved beta/ana in room " + chatRoom + "\n"; got != want {
		t.Errorf("admin move = %q, want %q", got, want)
	}

	members := lines(runChat(t, cfg, "tok-cid", "members"))
	slices.Sort(members)
	if want := []string{"beta/ana", "beta/cid"}; !slices.Equal(members, want) {
		t.Errorf("members after the move = %v, want %v", members, want)
	}

	// ana is cid's teammate now, so cid's bare name resolves inside beta.
	got = runChat(t, cfg, "tok-cid", "send", "ana", "same team now")
	if want := "sent to beta/ana\n"; got != want {
		t.Errorf("send after the move = %q, want %q", got, want)
	}
	got = runChat(t, cfg, "tok-ana", "read")
	if !strings.Contains(got, "beta/cid: same team now") {
		t.Errorf("read after the move = %q, want the message", got)
	}

	// And the team it left no longer names it.
	_, stderr, err := execChat(t, cfg, "tok-cid", "send", "alpha/ana", "stale address")
	if err == nil {
		t.Fatal("send to the team the member left succeeded, want a failure")
	}
	if !strings.Contains(stderr, "member not found") {
		t.Errorf("stderr = %q, want an unresolved address", stderr)
	}
}

// The host speaks into a room it does not attend: the message lands in the
// addressed inbox under the reserved "admin" identity, "*" reaches everyone
// there, and none of it leaves a member behind for the room to talk back to.
func TestChat_AdminSendsWithoutAttending(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)

	runChat(t, cfg, "tok-ana", "join", "--name", "ana")
	runChat(t, cfg, "tok-bob", "join", "--name", "bob")
	runChat(t, cfg, "tok-cid", "join", "--name", "cid")

	// A named target is addressed the way `chat send` addresses one, and the
	// count is echoed back so "*" and a name read alike.
	got := runChat(t, cfg, "", "admin", "send", chatRoom, "alpha/ana",
		"deploy is frozen", "--identity", identity)
	if want := "sent to alpha/ana in room " + chatRoom + ": delivered to 1 member\n"; got != want {
		t.Errorf("admin send = %q, want %q", got, want)
	}

	// The message names the host rather than a peer: the sender's team repeats
	// the reserved name, so it renders as an address that no member could hold.
	got = runChat(t, cfg, "tok-ana", "read")
	if !strings.Contains(got, "admin/admin: deploy is frozen") {
		t.Errorf("read = %q, want it attributed to the reserved admin identity", got)
	}
	if got := runChat(t, cfg, "tok-bob", "read"); got != "no pending messages\n" {
		t.Errorf("bystander read = %q, want nothing: ana was addressed, not bob", got)
	}

	// Speaking into the room did not join it, under any spelling of the name.
	members := lines(runChat(t, cfg, "tok-bob", "members"))
	slices.Sort(members)
	if want := []string{"alpha/ana", "alpha/bob", "beta/cid"}; !slices.Equal(members, want) {
		t.Errorf("members after the admin send = %v, want %v", members, want)
	}
	for _, m := range members {
		if strings.Contains(m, "admin") {
			t.Errorf("members = %v, want no member row for the admin sender", members)
		}
	}

	// "*" is the whole room, across teams — the admin attends none of it, so
	// there is no sender to leave out the way a member broadcast leaves itself.
	got = runChat(t, cfg, "", "admin", "send", chatRoom, "*",
		"standup in five", "--identity", identity)
	if want := "sent to * in room " + chatRoom + ": delivered to 3 members\n"; got != want {
		t.Errorf("admin send to * = %q, want %q", got, want)
	}
	for _, token := range []string{"tok-ana", "tok-bob", "tok-cid"} {
		if got := runChat(t, cfg, token, "read"); !strings.Contains(
			got, "admin/admin: standup in five") {
			t.Errorf("read as %s = %q, want the room-wide message", token, got)
		}
	}

	// And send is gated by the identity file like every other admin verb: it is
	// refused before the daemon is dialed, naming the file it wants.
	_, stderr, err := execChat(t, cfg, "", "admin", "send", chatRoom, "alpha/ana", "unsigned")
	if err == nil {
		t.Fatal("admin send without an identity succeeded, want a failure")
	}
	if !strings.Contains(stderr, "no admin age identity file") {
		t.Errorf("stderr = %q, want it to ask for an identity file", stderr)
	}
}

// The verbs that moved under `admin` are gone from the chat parent, and saying
// them there has to fail. A group parent that cannot run answers anything it
// does not recognize with its own help and a success exit, which would let a
// stale `chat register ...` in a script look like it worked; these two are the
// spellings most likely to be typed from memory.
func TestChat_RemovedSpellingsAreRejected(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"register moved under admin", []string{"chat", "register", chatRoom, "humans", "yuki"}},
		{"team is gone entirely", []string{"chat", "team", "list"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := execChatEnv(t, nil, tc.args...)
			if err == nil {
				t.Fatalf("crabswarm %s succeeded, want a failure", strings.Join(tc.args, " "))
			}
			want := fmt.Sprintf("unknown command %q for \"crabswarm chat\"", tc.args[1])
			if !strings.Contains(stderr, want) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, want)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want nothing: the help text is not an answer here", stdout)
			}
		})
	}
}

// The state an agent reports reaches cmdman's status display, which is where an
// operator watching their commands sees it. The daemon drives the real CLI
// surface — `cmdman status set <state> <id> --detail ...` and
// `cmdman status delete <id>` — so the stub records invocations verbatim rather
// than the test asserting against a Go paraphrase of them.
func TestChat_MirrorsMemberStateOntoCmdmanStatus(t *testing.T) {
	cfg := startChatDaemon(t)

	runChat(t, cfg, "tok-ana", "join", "--name", "ana")
	runChat(t, cfg, "tok-ana", "report-state", "working")
	runChat(t, cfg, "tok-ana", "report-state", "waiting")
	runChat(t, cfg, "tok-ana", "leave")

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

// A human is registered by the host, so their token is the daemon's own secret
// and names no cmdman command. It must never reach a cmdman command line, not
// even to be rejected there.
func TestChat_NeverPublishesAHumanToken(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)

	token := registerChatHuman(t, cfg, identity, chatRoom, "humans", "yuki")
	runChat(t, cfg, token, "join", "--name", "yuki")
	runChat(t, cfg, token, "report-state", "working")
	runChat(t, cfg, token, "leave")

	// An agent doing the same thing proves the recording works at all, so the
	// human half below cannot pass by nothing being recorded.
	runChat(t, cfg, "tok-ana", "join", "--name", "ana")

	published := stubStatus(t, cfg)
	if want := []string{
		"set done tok-ana --detail crabswarm chat",
	}; !slices.Equal(
		published,
		want,
	) {
		t.Errorf("cmdman status invocations = %q, want only the agent's %q", published, want)
	}
	for _, line := range published {
		if strings.Contains(line, token) {
			t.Errorf("cmdman status invocation %q carries the human's token", line)
		}
	}
}

// A container that comes back brings its token with it, so two bridges can be
// live on one identity at once. Both have to serve — a harness whose MCP
// subprocess died during the handshake has no chat at all — and the room must
// still hold a single member: attendance follows the token, not the process
// that declared it.
func TestChat_TwoBridgesOnOneTokenAttendOnce(t *testing.T) {
	cfg := startChatDaemon(t)

	bridges := []*mcp.ClientSession{
		startChatBridge(t, cfg, "tok-ana"),
		startChatBridge(t, cfg, "tok-ana"),
	}
	for i, session := range bridges {
		if got := session.InitializeResult().ServerInfo.Name; got != "crabswarm-chat" {
			t.Errorf("bridge %d announced itself as %q, want %q", i, got, "crabswarm-chat")
		}
	}

	// Attendance is observed before anything is asked of either bridge, so what
	// passes here is the join a bridge makes on its own rather than the one a
	// tool call would have made on its way to answering.
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)

	// Asking both settles both joins, and the room names the member once: the
	// second was answered from the stored membership rather than attending
	// again beside it.
	want := chatBridgeAna + "\n"
	for i, session := range bridges {
		if got := callChatTool(t, session, "chat_members", nil); got != want {
			t.Errorf("bridge %d chat_members = %q, want %q", i, got, want)
		}
	}
}

// The whole path a configured harness takes: two agents, each with a bridge of
// its own started from the command line the apm package declares, talking
// through the tools alone. Nothing here runs `chat join` — the bridges attend,
// which is the only automatic join left — and what a tool hands back is
// compared against what the CLI verb prints for the same message, since a
// member wired through MCP is meant to read its room in the same words as one
// typing commands.
func TestChat_BridgeToolsCarryTheRoom(t *testing.T) {
	cfg := startChatDaemon(t)

	ana := startChatBridge(t, cfg, "tok-ana")
	bob := startChatBridge(t, cfg, "tok-bob")

	// Both are asked for the roster before either is addressed. A bridge
	// declares attendance as it starts, but a message that overtook that join
	// would name a member the daemon does not have yet.
	callChatTool(t, ana, "chat_members", nil)
	members := lines(callChatTool(t, bob, "chat_members", nil))
	slices.Sort(members)
	if want := []string{chatBridgeAna, chatBridgeBob}; !slices.Equal(members, want) {
		t.Errorf("chat_members = %v, want %v", members, want)
	}

	// A bare name resolves inside the sender's own team, and the tool reports
	// whose inbox it landed in.
	got := callChatTool(t, ana, "chat_send", map[string]any{
		"to": "agent-tok-bob", "message": "the bridge is up",
	})
	if want := "sent to " + chatBridgeBob + "\n"; got != want {
		t.Errorf("chat_send = %q, want %q", got, want)
	}

	throughBridge := chatMessageBody(t, callChatTool(t, bob, "chat_read", nil))
	if want := chatBridgeAna + ": the bridge is up\n"; throughBridge != want {
		t.Errorf("chat_read = %q, want %q", throughBridge, want)
	}

	// The same message read the other way. The second copy is sent after the
	// first read rather than beside it: a read hands over the whole inbox, and
	// would have taken both.
	callChatTool(t, ana, "chat_send", map[string]any{
		"to": "agent-tok-bob", "message": "the bridge is up",
	})
	throughCLI := chatMessageBody(t, runChat(t, cfg, "tok-bob", "read"))
	if throughCLI != throughBridge {
		t.Errorf("`chat read` printed %q, want the tool's %q", throughCLI, throughBridge)
	}

	// The attendance the bridges declared is the daemon's, not something the
	// tools keep between themselves: a member verb typed at the CLI, which the
	// daemon answers for members alone, reads back the same room.
	roster := lines(runChat(t, cfg, "tok-ana", "members"))
	slices.Sort(roster)
	if want := []string{chatBridgeAna, chatBridgeBob}; !slices.Equal(roster, want) {
		t.Errorf("members = %v, want %v", roster, want)
	}
}
