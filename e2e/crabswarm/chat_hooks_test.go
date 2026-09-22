package crabswarm_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// The hook wiring crabswarm-mcp ships, and the support every case that runs a
// shipped hook command is built on.
//
// One harness is hooked: Codex, through `.apm/hooks/codex-hooks.json`, and
// chat_codex_test.go is what pins the events that file wires. The cases here
// are the other half — how `crabswarm hook exec` behaves running those
// commands — so they read the same file and exercise it rather than restating
// what it declares.
//
// The Claude Code plugin ships no hook file, and apm_package_test.go is what
// keeps it that way. Its member reports what the session is doing off the
// agents listing its own `crabswarm mcp` server polls, and its messages arrive
// either as a channel notification or as a line the daemon types into the
// terminal, so a hook there would have nothing to add and one thing to get
// wrong: a report from a hook races a feed that is already right.
//
// Every case below runs the shipped command string verbatim rather than a Go
// paraphrase of it: the wiring is a text file no compiler ever sees, and a
// template that renders the wrong thing is exactly the bug that costs a
// message.

// chatHookConfig is the hook file's shape on both harnesses: events, each
// holding matcher groups, each holding the commands to run.
type chatHookConfig struct {
	Hooks map[string][]chatHookMatcher `json:"hooks"`
}

type chatHookMatcher struct {
	Matcher string          `json:"matcher"`
	Hooks   []chatHookEntry `json:"hooks"`
}

type chatHookEntry struct {
	Type          string `json:"type"`
	Command       string `json:"command"`
	Timeout       int    `json:"timeout"`
	StatusMessage string `json:"statusMessage"`
}

// commands returns every command wired to event, in file order.
func (c chatHookConfig) commands(event string) []string {
	var out []string
	for _, m := range c.Hooks[event] {
		for _, h := range m.Hooks {
			out = append(out, h.Command)
		}
	}
	return out
}

// command returns the single command wired to event, failing when the file
// wires none or several: a case that means to exercise "the Stop hook" must
// not silently pick one of two.
func (c chatHookConfig) command(t *testing.T, event string) string {
	t.Helper()
	got := c.commands(event)
	if len(got) != 1 {
		t.Fatalf("event %s wires %d commands, want exactly 1: %v", event, len(got), got)
	}
	return got[0]
}

// The envelopes a harness writes to a hook's stdin, one per event the file
// wires. Only the fields the hook commands actually read matter — the Stop flag
// above all — but each carries the envelope metadata a real harness sends,
// since `hook exec` parses the whole thing before it renders anything.
const (
	chatStopEnvelope = `{` +
		`"session_id":"sess-e2e",` +
		`"transcript_path":"/tmp/e2e.jsonl",` +
		`"cwd":"/tmp",` +
		`"hook_event_name":"Stop",` +
		`"stop_hook_active":false}`
	chatStopActiveEnvelope = `{` +
		`"session_id":"sess-e2e",` +
		`"transcript_path":"/tmp/e2e.jsonl",` +
		`"cwd":"/tmp",` +
		`"hook_event_name":"Stop",` +
		`"stop_hook_active":true}`
	// Codex's approval dialog, which no file here wires. It is the envelope
	// that says whether `hook exec` speaks the rest of Codex's surface at all.
	chatPermissionRequestEnvelope = `{` +
		`"session_id":"sess-e2e",` +
		`"transcript_path":"/tmp/e2e.jsonl",` +
		`"cwd":"/tmp",` +
		`"hook_event_name":"PermissionRequest",` +
		`"tool_name":"Bash",` +
		`"tool_input":{"command":"ls"}}`
)

// chatHookEnvelopes pairs each wired event with the stdin a harness would give
// it. TestChatCodex_HooksAreHarmlessWithoutADaemon keeps it exhaustive, so
// wiring a new event without an envelope fails rather than going untested.
var chatHookEnvelopes = map[string]string{
	"PostToolUse": postToolUseEnvelope,
	"Stop":        chatStopEnvelope,
}

// chatHookEnvelope is the stdin a harness gives the group event wires under a
// matcher. A group is what a caller has in hand, so the matcher is taken; no
// group of the one hook file left needs an envelope other than its event's.
func chatHookEnvelope(event, _ string) string {
	return chatHookEnvelopes[event]
}

// chatSentText is what a teammate sends in the delivery cases. The double quote
// and the line break are the point: they are what a hand-rolled encoder gets
// wrong, and a mangled hand-over is a message the agent never sees.
const chatSentText = "say \"hi\"\nand a second line"

// runChatHook runs one shipped hook command the way a harness does — through a
// shell, with the envelope on stdin — against the daemon cfgPath describes,
// acting as the holder of token.
//
// The built binary's directory leads the PATH so the `crabswarm` the command
// names is this checkout's, and $CRABSWARM_CONF reaches both halves of the
// invocation: the outer `hook exec` and the `crabswarm chat ...` it spawns.
func runChatHook(t *testing.T, cfgPath, token, command, envelope string) hookResult {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "sh", "-c", command)
	cmd.Stdin = strings.NewReader(envelope)
	cmd.Env = append(chatEnviron(),
		"PATH="+filepath.Dir(crabswarmBin)+string(os.PathListSeparator)+os.Getenv("PATH"),
		"CRABSWARM_CONF="+cfgPath,
		chatTokenEnvVar+"="+token,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if _, ok := errors.AsType[*exec.ExitError](err); !ok {
			t.Fatalf("run hook command %q: %v", command, err)
		}
	}
	return hookResult{
		stdout:   stdout.String(),
		stderr:   stderr.String(),
		exitCode: cmd.ProcessState.ExitCode(),
	}
}

// chatAbsentDaemonConfig names a socket nothing listens on, which is how the
// cases below stand a hook up against a daemon that is not running.
func chatAbsentDaemonConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	writeFile(t, cfgPath, fmt.Sprintf(`{"sock":%q}`, filepath.Join(dir, "absent.sock")))
	return cfgPath
}

// hookObject decodes a hook's output as a plain JSON object, leaving the key set
// for the case to pin: Codex rejects an output object carrying a field its
// schema does not know, so "and nothing else" is half of what these hooks have
// to get right. A struct with named fields cannot say that.
func hookObject(t *testing.T, s string) map[string]json.RawMessage {
	t.Helper()
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		t.Fatalf("decode hook output %q: %v", s, err)
	}
	return obj
}

// assertKeys reports an object whose key set is not exactly want.
func assertKeys(t *testing.T, obj map[string]json.RawMessage, want ...string) {
	t.Helper()
	got := slices.Sorted(maps.Keys(obj))
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("output keys = %v, want exactly %v", got, want)
	}
}

// hookString decodes one string field of a hook output object.
func hookString(t *testing.T, obj map[string]json.RawMessage, key string) string {
	t.Helper()
	var s string
	if err := json.Unmarshal(obj[key], &s); err != nil {
		t.Fatalf("decode %s of %v: %v", key, obj, err)
	}
	return s
}

// assertHookIsSilent pins what an allow looks like on the wire: nothing on
// stdout for the harness to read as a decision, nothing on stderr, exit 0.
func assertHookIsSilent(t *testing.T, res hookResult) {
	t.Helper()
	if res.exitCode != 0 {
		t.Errorf("exit code = %d, want 0\nstderr:\n%s", res.exitCode, res.stderr)
	}
	if res.stdout != "" {
		t.Errorf("stdout = %q, want nothing: an empty output is the allow", res.stdout)
	}
	if res.stderr != "" {
		t.Errorf("stderr = %q, want nothing", res.stderr)
	}
}

// startChatRoomWithMail brings up a daemon, attends as ana and bob, and leaves
// one message from bob unread for ana. It returns the config path the hooks run
// against; every case acts as ana, the member whose turn is ending.
func startChatRoomWithMail(t *testing.T) string {
	t.Helper()
	cfg := startChatDaemon(t)
	attendChatBridges(t, cfg, "tok-ana", "tok-bob")
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)
	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, chatSentText)
	return cfg
}

// chatMailLine is how the message [startChatRoomWithMail] leaves reads once a
// hook has handed it over: the sender, the role it named and the mention
// marker, with the sequence number and the stamp in front of them left out
// because neither is the same twice.
const chatMailLine = chatBridgeBob + " -> " + chatBridgeAna + ` [mentioned you]: say "hi"`

// assertInboxStillHoldsTheMail reads ana's inbox directly and reports an inbox
// a hook was supposed to leave alone.
func assertInboxStillHoldsTheMail(t *testing.T, cfg string) {
	t.Helper()
	if got := runChat(t, cfg, "tok-ana", "read"); !strings.Contains(got, "say \"hi\"") {
		t.Errorf("inbox after the hook = %q, want the message still waiting", got)
	}
}

// assertInboxWasConsumed reads ana's inbox directly and reports mail a hook
// handed over but left behind — the read that delivered it should have
// consumed it.
func assertInboxWasConsumed(t *testing.T, cfg string) {
	t.Helper()
	if got := runChat(t, cfg, "tok-ana", "read"); got != "no pending messages\n" {
		t.Errorf("inbox after the hook = %q, want it emptied by the delivery", got)
	}
}

// Nothing waiting: the turn ends as it would have without the hook. The done
// report the same read makes on the way is not observable from outside the
// daemon — TestClient_ReadDoneWhenEmpty pins that against the RPC — so what is
// asserted here is the half a harness sees.
//
// A board post is the second way a room has nothing for this member: it is in
// the log and mentions nobody, so the drain finds nothing to hand over and the
// turn ends. That half is worth its own case because it is the one a naive
// implementation gets wrong — the room is not quiet, the message simply is not
// addressed to anyone.
func TestChatHooks_StopAllowsWithNothingToDeliver(t *testing.T) {
	stop := readCodexHooks(t).command(t, "Stop")

	t.Run("the room said nothing at all", func(t *testing.T) {
		cfg := startChatDaemon(t)
		attendChatBridges(t, cfg, "tok-ana")

		assertHookIsSilent(t, runChatHook(t, cfg, "tok-ana", stop, chatStopEnvelope))
	})

	t.Run("the room posted to its board", func(t *testing.T) {
		cfg := startChatDaemon(t)
		attendChatBridges(t, cfg, "tok-ana", "tok-bob")
		waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)
		runChat(t, cfg, "tok-bob", "send", "", "fyi: rebased main")

		assertHookIsSilent(t, runChatHook(t, cfg, "tok-ana", stop, chatStopEnvelope))
	})
}

// The PostToolUse hook hands the messages over as additionalContext rather than
// announcing them: the read already consumed them, so injecting the text is the
// delivery. hookSpecificOutput and nothing else, for the same reason the Stop
// drain keeps to decision and reason.
func TestChatHooks_PostToolUseDeliversTheMessages(t *testing.T) {
	cfg := startChatRoomWithMail(t)
	command := readCodexHooks(t).command(t, "PostToolUse")

	res := runChatHook(t, cfg, "tok-ana", command, postToolUseEnvelope)
	if res.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s",
			res.exitCode, res.stdout, res.stderr)
	}
	if res.stderr != "" {
		t.Errorf("stderr = %q, want nothing", res.stderr)
	}

	obj := hookObject(t, res.stdout)
	assertKeys(t, obj, "hookSpecificOutput")
	inner := hookObject(t, string(obj["hookSpecificOutput"]))
	assertKeys(t, inner, "hookEventName", "additionalContext")
	if got := hookString(t, inner, "hookEventName"); got != "PostToolUse" {
		t.Errorf("hookEventName = %q, want %q", got, "PostToolUse")
	}
	injected := hookString(t, inner, "additionalContext")
	for _, want := range []string{
		chatMailLine,
		"and a second line",
		// The reply is a tool call, not a command line: every harness the room
		// reaches is served by the MCP server, and some of them decline to run a
		// command nobody asked them to run.
		"chat_send",
	} {
		if !strings.Contains(injected, want) {
			t.Errorf("additionalContext = %q, want it to carry %q", injected, want)
		}
	}
	assertInboxWasConsumed(t, cfg)
}

// This one runs after every single tool call, so anything short of a message
// actually having arrived ends in silence: an empty inbox and a daemon that is
// not there are neither of them news the agent needs mid-task. The daemon-down
// case is the one `.Output` would get wrong — the CLI's "start the daemon" hint
// goes to stderr, and injecting that as a message would be a delivery of
// something nobody sent.
func TestChatHooks_PostToolUseIsSilentWithoutMessages(t *testing.T) {
	command := readCodexHooks(t).command(t, "PostToolUse")

	t.Run("the inbox is empty", func(t *testing.T) {
		cfg := startChatDaemon(t)
		attendChatBridges(t, cfg, "tok-ana")
		assertHookIsSilent(t, runChatHook(t, cfg, "tok-ana", command, postToolUseEnvelope))
	})

	t.Run("the daemon is unreachable", func(t *testing.T) {
		cfg := chatAbsentDaemonConfig(t)
		assertHookIsSilent(t, runChatHook(t, cfg, "tok-ana", command, postToolUseEnvelope))
	})
}

// An event the file does not wire still parses: `hook exec` decodes the
// envelope into its typed variant, the command runs, and a template with
// nothing to record renders nothing. That is what says the envelope surface is
// the harness's rather than a list of the events this package happens to use —
// Codex announces its approval dialog this way, and a file that wires it later
// must not be the thing that discovers the parser cannot read it.
func TestChatHooks_AnUnwiredEventStillParses(t *testing.T) {
	cfg := startChatDaemon(t)
	attendChatBridges(t, cfg, "tok-ana")

	assertHookIsSilent(t, runChatHook(t, cfg, "tok-ana",
		readCodexHooks(t).command(t, "PostToolUse"), chatPermissionRequestEnvelope))
}

// No hook attends the room. Attendance is the MCP bridge's open stream, held
// for the whole session the harness runs the bridge in, so nothing is left for
// a `SessionStart` entry to declare — and a hook wired there would be a second
// thing claiming the same role, which the daemon refuses.
//
// It is asserted because re-adding a hook entry breaks nothing loudly: the
// session that lost the race simply never reaches its room.
func TestChatHooks_LeaveAttendanceToTheBridge(t *testing.T) {
	if groups, ok := readCodexHooks(t).Hooks["SessionStart"]; ok {
		t.Errorf("SessionStart is wired again (%d group(s)); the MCP bridge attends the room",
			len(groups))
	}
}

// Everything the hooks need ships inside `crabswarm hook exec`: no shell
// scripts to copy alongside the JSON, no `jq` to have installed, and no plugin
// root to resolve. The per-entry half of that is asserted over the file itself
// by TestChatCodex_HooksLeaveTheStateToTheAppServer; this is the directory
// half, which no command string would ever show.
func TestChatHooks_ShipNoScripts(t *testing.T) {
	if _, err := os.Stat(filepath.Join(
		repoRoot(), "apm-package", "crabswarm-mcp", "scripts")); err == nil {
		t.Error("apm-package/crabswarm-mcp/scripts exists again; the hooks ship no scripts")
	}
}

// assertSelfContainedHookEntry pins one entry's command against the four ways
// this wiring used to reach outside the binary, and against the timeout every
// entry needs: the PostToolUse hook runs after every tool call, and a wedged
// daemon must not stall the session.
func assertSelfContainedHookEntry(t *testing.T, event string, h chatHookEntry) {
	t.Helper()
	if !strings.HasPrefix(h.Command, "crabswarm hook exec ") {
		t.Errorf("%s command %q does not run through `crabswarm hook exec`", event, h.Command)
	}
	for _, banned := range []string{
		"jq",
		"CLAUDE_PLUGIN_ROOT",
		"scripts/",
		// The empty-inbox wording used to be the signal that mail arrived.
		// `chat read --quiet` prints nothing at all instead, so no hook has to
		// know a sentence the renderer is free to reword.
		"no pending messages",
	} {
		if strings.Contains(h.Command, banned) {
			t.Errorf("%s command %q still depends on %q", event, h.Command, banned)
		}
	}
	if h.Timeout <= 0 {
		t.Errorf("%s command %q carries no timeout", event, h.Command)
	}
	if h.Type != "command" {
		t.Errorf("%s entry type = %q, want %q", event, h.Type, "command")
	}
}

// The unparseable envelope is a plain error, not a decision: exit 1 with the
// reason on stderr and nothing on stdout. That is a louder failure than the
// shell scripts this wiring replaced — they treated an unreadable envelope as
// "leave the inbox alone" — but it is never a block, and a harness that sends
// unparseable JSON is broken in a way worth hearing about.
func TestChatHooks_UnparseableEnvelopeFailsWithoutBlocking(t *testing.T) {
	cfg := startChatRoomWithMail(t)
	stop := readCodexHooks(t).command(t, "Stop")

	res := runChatHook(t, cfg, "tok-ana", stop, "not an envelope at all")
	if res.exitCode != 1 {
		t.Errorf("exit code = %d, want 1\nstdout:\n%s\nstderr:\n%s",
			res.exitCode, res.stdout, res.stderr)
	}
	if res.stdout != "" {
		t.Errorf("stdout = %q, want nothing: a failed parse emits no hook decision", res.stdout)
	}
	if !strings.Contains(res.stderr, "parsing hook input") {
		t.Errorf("stderr = %q, want it to name the failed parse", res.stderr)
	}
	assertInboxStillHoldsTheMail(t, cfg)
}
