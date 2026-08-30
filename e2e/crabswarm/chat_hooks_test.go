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
)

// The one hook file the package installs into every harness: its stem carries
// no target token, so apm hands the same wiring to Claude Code and to Codex,
// and each ignores the events it does not know. The commands inside are
// `crabswarm hook exec` invocations, so every case below runs the shipped
// command string verbatim rather than a Go paraphrase of it: the wiring is a
// text file no compiler ever sees, and a template that renders the wrong thing
// is exactly the bug that costs a message.
var chatHooksPath = []string{".apm", "hooks", "report-state.json"}

// chatHookConfig is the hook file both harnesses read: events, each holding
// matcher groups, each holding the commands to run.
type chatHookConfig struct {
	Version int                          `json:"version"`
	Hooks   map[string][]chatHookMatcher `json:"hooks"`
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

// readChatHooks decodes the package's hook file out of the checkout under test.
func readChatHooks(t *testing.T) chatHookConfig {
	t.Helper()
	path := filepath.Join(append(
		[]string{repoRoot(), "apm-package", "crabswarm-chat"}, chatHooksPath...)...)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read hook file %s: %v", path, err)
	}
	var cfg chatHookConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("decode hook file %s: %v", path, err)
	}
	if len(cfg.Hooks) == 0 {
		t.Fatalf("hook file %s wires no events", path)
	}
	return cfg
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

// The envelopes the harnesses write to a hook's stdin, one per event this
// package wires. Only the fields the hook commands actually read matter — the
// Stop flag above all — but each carries the envelope metadata a real harness
// sends, since `hook exec` parses the whole thing before it renders anything.
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
	chatSessionStartEnvelope = `{` +
		`"session_id":"sess-e2e",` +
		`"transcript_path":"/tmp/e2e.jsonl",` +
		`"cwd":"/tmp",` +
		`"hook_event_name":"SessionStart",` +
		`"source":"startup"}`
	chatUserPromptSubmitEnvelope = `{` +
		`"session_id":"sess-e2e",` +
		`"transcript_path":"/tmp/e2e.jsonl",` +
		`"cwd":"/tmp",` +
		`"hook_event_name":"UserPromptSubmit",` +
		`"prompt":"do the thing"}`
	chatNotificationEnvelope = `{` +
		`"session_id":"sess-e2e",` +
		`"transcript_path":"/tmp/e2e.jsonl",` +
		`"cwd":"/tmp",` +
		`"hook_event_name":"Notification",` +
		`"message":"Claude needs your permission to run a command"}`
	// The approval dialog both harnesses announce. Codex has no other event for
	// it, and it is the envelope that says whether `hook exec` speaks Codex's
	// half of the surface at all.
	chatPermissionRequestEnvelope = `{` +
		`"session_id":"sess-e2e",` +
		`"transcript_path":"/tmp/e2e.jsonl",` +
		`"cwd":"/tmp",` +
		`"hook_event_name":"PermissionRequest",` +
		`"tool_name":"Bash",` +
		`"tool_input":{"command":"ls"}}`
)

// chatHookEnvelopes pairs each wired event with the stdin a harness would give
// it. TestChatHooks_EveryWiredEventIsExercised keeps it exhaustive, so wiring a
// new event without an envelope fails rather than going untested.
var chatHookEnvelopes = map[string]string{
	"SessionStart":      chatSessionStartEnvelope,
	"UserPromptSubmit":  chatUserPromptSubmitEnvelope,
	"Notification":      chatNotificationEnvelope,
	"PermissionRequest": chatPermissionRequestEnvelope,
	"PostToolUse":       postToolUseEnvelope,
	"Stop":              chatStopEnvelope,
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
// one message from bob waiting in ana's inbox. It returns the config path the
// hooks run against; every case acts as ana, the member whose turn is ending.
func startChatRoomWithMail(t *testing.T) string {
	t.Helper()
	cfg := startChatDaemon(t)
	runChat(t, cfg, "tok-ana", "join", "--name", "ana")
	runChat(t, cfg, "tok-bob", "join", "--name", "bob")
	runChat(t, cfg, "tok-bob", "send", "ana", chatSentText)
	return cfg
}

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

// Messages in hand at turn end: the stop is blocked so they reach the agent,
// carried through `hook exec`'s JSON intact — quotes and line breaks included.
// decision and reason and nothing else, since a rejected output after a
// consuming read is exactly the lost message this hook exists to prevent. The
// inbox is empty afterwards: the block is the delivery.
func TestChatHooks_StopBlocksWithTheMessages(t *testing.T) {
	cfg := startChatRoomWithMail(t)
	hooks := readChatHooks(t)

	res := runChatHook(t, cfg, "tok-ana", hooks.command(t, "Stop"), chatStopEnvelope)
	if res.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s",
			res.exitCode, res.stdout, res.stderr)
	}
	if res.stderr != "" {
		t.Errorf("stderr = %q, want nothing", res.stderr)
	}

	obj := hookObject(t, res.stdout)
	assertKeys(t, obj, "decision", "reason")
	if got := hookString(t, obj, "decision"); got != "block" {
		t.Errorf("decision = %q, want %q", got, "block")
	}
	reason := hookString(t, obj, "reason")
	for _, want := range []string{
		"alpha/bob: say \"hi\"",
		"and a second line",
		"crabswarm chat send",
	} {
		if !strings.Contains(reason, want) {
			t.Errorf("reason = %q, want it to carry %q", reason, want)
		}
	}
	assertInboxWasConsumed(t, cfg)
}

// stop_hook_active means an earlier Stop hook already blocked this turn.
// Reading again would either loop the agent or, once the harness stops honoring
// the block, consume messages it never shows — so the hook does not read at
// all, and the mail is still there afterwards.
func TestChatHooks_StopLeavesTheInboxAloneWhenAlreadyBlocking(t *testing.T) {
	cfg := startChatRoomWithMail(t)
	hooks := readChatHooks(t)

	res := runChatHook(t, cfg, "tok-ana", hooks.command(t, "Stop"), chatStopActiveEnvelope)
	assertHookIsSilent(t, res)
	assertInboxStillHoldsTheMail(t, cfg)
}

// Nothing waiting: the turn ends as it would have without the hook. The done
// report the same read makes on the way is not observable from outside the
// daemon — TestClient_ReadDoneWhenEmpty pins that against the RPC — so what is
// asserted here is the half a harness sees.
func TestChatHooks_StopAllowsWithNothingToDeliver(t *testing.T) {
	cfg := startChatDaemon(t)
	runChat(t, cfg, "tok-ana", "join", "--name", "ana")
	hooks := readChatHooks(t)

	res := runChatHook(t, cfg, "tok-ana", hooks.command(t, "Stop"), chatStopEnvelope)
	assertHookIsSilent(t, res)
}

// The PostToolUse hook hands the messages over as additionalContext rather than
// announcing them: the read already consumed them, so injecting the text is the
// delivery. hookSpecificOutput and nothing else, for the same reason the Stop
// drain keeps to decision and reason.
func TestChatHooks_PostToolUseDeliversTheMessages(t *testing.T) {
	cfg := startChatRoomWithMail(t)
	hooks := readChatHooks(t)

	res := runChatHook(t, cfg, "tok-ana",
		hooks.commands("PostToolUse")[0], postToolUseEnvelope)
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
		"alpha/bob: say \"hi\"",
		"and a second line",
		"crabswarm chat send",
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
	hooks := readChatHooks(t)
	command := hooks.commands("PostToolUse")[0]

	t.Run("the inbox is empty", func(t *testing.T) {
		cfg := startChatDaemon(t)
		runChat(t, cfg, "tok-ana", "join", "--name", "ana")
		assertHookIsSilent(t, runChatHook(t, cfg, "tok-ana", command, postToolUseEnvelope))
	})

	t.Run("the daemon is unreachable", func(t *testing.T) {
		cfg := chatAbsentDaemonConfig(t)
		assertHookIsSilent(t, runChatHook(t, cfg, "tok-ana", command, postToolUseEnvelope))
	})
}

// Every shipped command, on every event, against a daemon that is not running:
// none of them says anything and none of them fails. That is the whole
// degradation story of this package — a chat nobody is hosting must not break a
// session start, a turn, or a tool call — and it is asserted over the file
// itself so a newly wired command cannot skip it.
func TestChatHooks_EveryCommandIsHarmlessWithoutADaemon(t *testing.T) {
	cfg := chatAbsentDaemonConfig(t)
	hooks := readChatHooks(t)
	for _, event := range slices.Sorted(maps.Keys(hooks.Hooks)) {
		for i, command := range hooks.commands(event) {
			t.Run(fmt.Sprintf("%s/%d", event, i), func(t *testing.T) {
				assertHookIsSilent(t,
					runChatHook(t, cfg, "tok-ana", command, chatHookEnvelopes[event]))
			})
		}
	}
}

// The fire-and-forget hooks are silent by design, which makes "it did nothing"
// and "it worked" look alike from the outside. SessionStart is the one whose
// effect the room reports back, so it is checked against a live daemon: the
// hook's bare `chat join` takes the name the daemon derives from the token.
func TestChatHooks_SessionStartAttendsTheRoom(t *testing.T) {
	cfg := startChatDaemon(t)
	runChat(t, cfg, "tok-ana", "join", "--name", "ana")
	hooks := readChatHooks(t)

	res := runChatHook(t, cfg, "tok-cid", hooks.command(t, "SessionStart"),
		chatSessionStartEnvelope)
	assertHookIsSilent(t, res)

	members := lines(runChat(t, cfg, "tok-ana", "members"))
	if len(members) != 2 {
		t.Fatalf("members = %v, want ana plus the member the hook joined", members)
	}
	if !slices.ContainsFunc(members, func(m string) bool {
		return strings.HasPrefix(m, "beta/")
	}) {
		t.Errorf("members = %v, want the hook's join to have attended under team beta", members)
	}
}

// An envelope `hook exec` cannot parse is a plain error, not a decision: exit 1
// with the reason on stderr and nothing on stdout. That is a louder failure
// than the shell scripts this wiring replaced — they treated an unreadable
// envelope as "leave the inbox alone" — but it is never a block, and a harness
// that sends unparseable JSON is broken in a way worth hearing about.
func TestChatHooks_UnparseableEnvelopeFailsWithoutBlocking(t *testing.T) {
	cfg := startChatRoomWithMail(t)
	hooks := readChatHooks(t)

	res := runChatHook(t, cfg, "tok-ana", hooks.command(t, "Stop"), "not an envelope at all")
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

// Everything the hooks need now ships inside `crabswarm hook exec`: no shell
// scripts to copy alongside the JSON, no `jq` to have installed, and no plugin
// root to resolve. This is asserted because losing it is invisible — a command
// that reaches back out to a script keeps working on the author's machine and
// breaks on every consumer that installed only the hook file.
func TestChatHooks_AreSelfContained(t *testing.T) {
	if _, err := os.Stat(filepath.Join(
		repoRoot(), "apm-package", "crabswarm-chat", "scripts")); err == nil {
		t.Error("apm-package/crabswarm-chat/scripts exists again; the hooks ship no scripts")
	}
	for event, groups := range readChatHooks(t).Hooks {
		for _, g := range groups {
			for _, h := range g.Hooks {
				assertSelfContainedHookEntry(t, event, h)
			}
		}
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

// One file wires the union of what the two harnesses announce, and each drops
// what it does not know: Codex's config loader ignores `Notification`, Claude
// Code runs `PermissionRequest` natively. So both waiting events carry the same
// report — byte for byte, since a waiting report reworded in one place and not
// the other is a member stuck in the wrong state on one harness — and
// `PostToolUse` reports working again after the delivery, which is Codex's only
// signal that a dialog resolved and Claude Code's way back out of `waiting`
// once a permission is granted.
func TestChatHooks_WireTheUnionOfBothHarnesses(t *testing.T) {
	hooks := readChatHooks(t)

	notification := hooks.command(t, "Notification")
	if !strings.Contains(notification, "report-state waiting") {
		t.Errorf("Notification command = %q, want it to report waiting", notification)
	}
	if got := hooks.command(t, "PermissionRequest"); got != notification {
		t.Errorf("PermissionRequest command = %q, want the Notification one %q",
			got, notification)
	}

	postToolUse := hooks.commands("PostToolUse")
	if len(postToolUse) != 2 {
		t.Fatalf("PostToolUse wires %d commands, want the delivery plus the "+
			"working report: %v", len(postToolUse), postToolUse)
	}
	if !strings.Contains(postToolUse[0], "chat read --quiet") {
		t.Errorf("first PostToolUse command = %q, want the delivering read", postToolUse[0])
	}
	if got, want := postToolUse[1], hooks.command(t, "UserPromptSubmit"); got != want {
		t.Errorf("second PostToolUse command = %q, want the UserPromptSubmit one %q",
			got, want)
	}
}

// Every event the file wires has an envelope in chatHookEnvelopes, so
// TestChatHooks_EveryCommandIsHarmlessWithoutADaemon really does feed each one
// what its harness would. Without this, wiring a new event would quietly hand
// that case an empty stdin and pass for the wrong reason.
func TestChatHooks_EveryWiredEventIsExercised(t *testing.T) {
	for event := range readChatHooks(t).Hooks {
		if _, ok := chatHookEnvelopes[event]; !ok {
			t.Errorf("event %s is wired but has no envelope in chatHookEnvelopes", event)
		}
	}
}
