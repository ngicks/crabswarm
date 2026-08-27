package crabswarm_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The two hook scripts the plugin wires into a live harness: the Stop-hook
// inbox drain and the PostToolUse delivery. They are exercised here as the
// harness runs them — piped an envelope on stdin, with a `crabswarm` on PATH —
// because nothing else in the build compiles or type-checks a shell script.
const (
	chatDrainScript = "chat-stop-drain.sh"
	chatHintScript  = "chat-inbox-hint.sh"
)

// chatEmptyInbox is what `crabswarm chat read` prints when nothing is waiting,
// which is the one thing both scripts read out of the CLI's output rather than
// out of its exit code. TestChatHookScripts_EmptyInboxLiteralMatchesTheCLI pins
// it against the binary at both ends.
const chatEmptyInbox = "no pending messages"

// chatDrainedMessages stands in for what a read hands over. It carries a double
// quote and a line break on purpose: those are what a shell script piping text
// through jq gets wrong, and a mangled hand-over is a message the agent never
// sees.
const chatDrainedMessages = "[2026-01-02T03:04:05Z] alpha/ana: say \"hi\"\n" +
	"[2026-01-02T03:04:06Z] alpha/bob: and a second line"

// The invocations the scripts may make, spelled as the stub records them.
const (
	callRead = "chat read"
	callIdle = "chat report-state idle"
)

// stubCrabswarmScript is the `crabswarm` the hook scripts call. It records
// every invocation — a read consumes, so which calls a script did *not* make is
// as much of the contract as its output — and answers `chat read` from the
// environment the case set up. Every other verb succeeds silently, which is all
// `chat report-state idle` needs.
const stubCrabswarmScript = `#!/bin/sh
printf '%s\n' "$*" >>"$CRABSWARM_STUB_LOG"
if [ "$1" = chat ] && [ "$2" = read ]; then
	if [ -n "$CRABSWARM_STUB_READ" ]; then
		printf '%s\n' "$CRABSWARM_STUB_READ"
	fi
	exit "$CRABSWARM_STUB_READ_EXIT"
fi
exit 0
`

// hookScriptRun is one invocation: which script, the envelope its harness
// writes to stdin, how the stub `crabswarm` behind it answers `chat read`, and
// whether jq is on the PATH it runs with.
type hookScriptRun struct {
	script   string
	stdin    string
	readOut  string // what `crabswarm chat read` prints; empty for nothing at all
	readExit int    // and how it exits; non-zero stands in for an absent daemon
	noJQ     bool
}

// hookScriptResult is what one run produced. The harness reads stdout and the
// exit code; calls is every `crabswarm` invocation the script made on the way
// there.
type hookScriptResult struct {
	stdout   string
	stderr   string
	exitCode int
	calls    []string
}

// chatHookScriptPath locates a hook script inside the checkout under test,
// which is the very file plugin/hooks/hooks.json points a harness at.
func chatHookScriptPath(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(repoRoot(), "plugin", "scripts", name)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("locate hook script %s: %v", name, err)
	}
	return path
}

// requireJQ skips a case that cannot run without jq. The scripts treat a
// missing jq as "leave the inbox alone", so a machine without it can still say
// something about the gate — just not about the encoding behind it.
func requireJQ(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skipf("jq is not installed: %v", err)
	}
}

// runHookScript runs one hook script under `sh`, the way both harnesses invoke
// it, against a stub `crabswarm` that shadows any real one on the PATH so no
// case can reach a daemon of the developer's.
func runHookScript(t *testing.T, run hookScriptRun) hookScriptResult {
	t.Helper()
	dir := t.TempDir()
	stub := filepath.Join(dir, "crabswarm")
	writeFile(t, stub, stubCrabswarmScript)
	if err := os.Chmod(stub, 0o755); err != nil {
		t.Fatalf("chmod stub crabswarm: %v", err)
	}
	log := filepath.Join(dir, "calls.log")

	// Dropping the rest of the PATH is how the jq gate is reached. `cat` is
	// linked in alongside the stub because the drain script reads its stdin with
	// it before it ever looks for jq, and a script that died on the read would
	// pass the gate's assertions for the wrong reason.
	path := dir + string(os.PathListSeparator) + os.Getenv("PATH")
	if run.noJQ {
		linkTool(t, dir, "cat")
		path = dir
	}

	cmd := exec.CommandContext(t.Context(), "sh", chatHookScriptPath(t, run.script))
	cmd.Stdin = strings.NewReader(run.stdin)
	cmd.Env = append(chatEnviron(),
		"PATH="+path,
		"CRABSWARM_STUB_LOG="+log,
		"CRABSWARM_STUB_READ="+run.readOut,
		fmt.Sprintf("CRABSWARM_STUB_READ_EXIT=%d", run.readExit),
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if _, ok := errors.AsType[*exec.ExitError](err); !ok {
			t.Fatalf("run %s: %v", run.script, err)
		}
	}
	return hookScriptResult{
		stdout:   stdout.String(),
		stderr:   stderr.String(),
		exitCode: cmd.ProcessState.ExitCode(),
		calls:    readStubCalls(t, log),
	}
}

// linkTool links one of the shell's external commands into dir, so a PATH
// holding dir alone still has it.
func linkTool(t *testing.T, dir, name string) {
	t.Helper()
	src, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s is not installed: %v", name, err)
	}
	if err := os.Symlink(src, filepath.Join(dir, name)); err != nil {
		t.Fatalf("link %s into %s: %v", name, dir, err)
	}
}

// readStubCalls returns the stub's recorded invocations. No log file means the
// script never called crabswarm at all.
func readStubCalls(t *testing.T, log string) []string {
	t.Helper()
	b, err := os.ReadFile(log)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("read stub call log: %v", err)
	}
	return lines(string(b))
}

// hookObject decodes a hook script's output as a plain JSON object, leaving the
// key set for the case to pin: Codex rejects an output object carrying a field
// its schema does not know, so "and nothing else" is half of what these scripts
// have to get right. The struct the Go hook cases decode into cannot say that.
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

// assertStopAllowed pins what every path that lets the stop through looks like:
// nothing on stdout for the harness to read as a decision, exit 0, and exactly
// the calls the path is allowed to make — the idle report among them, since
// that is what re-arms the daemon's nudge for a member about to go quiet.
func assertStopAllowed(t *testing.T, res hookScriptResult, wantCalls ...string) {
	t.Helper()
	if res.exitCode != 0 {
		t.Errorf("exit code = %d, want 0\nstderr:\n%s", res.exitCode, res.stderr)
	}
	if res.stdout != "" {
		t.Errorf("stdout = %q, want nothing: an empty output lets the stop through",
			res.stdout)
	}
	if res.stderr != "" {
		t.Errorf("stderr = %q, want nothing", res.stderr)
	}
	if !slices.Equal(res.calls, wantCalls) {
		t.Errorf("crabswarm calls = %v, want %v", res.calls, wantCalls)
	}
}

// stop_hook_active means an earlier Stop hook already blocked this turn.
// Reading again would either loop the agent or, once the harness stops honoring
// the block, consume messages it never shows — so the inbox is left alone. The
// same goes for an envelope that cannot be read at all: an inbox left full
// costs a late read, a drain nobody delivers costs the message.
func TestChatHookScripts_DrainLeavesTheInboxAlone(t *testing.T) {
	for _, tc := range []struct {
		name  string
		stdin string
	}{
		{"an earlier stop hook is still blocking", `{"stop_hook_active":true}`},
		{"the envelope is not JSON", "not an envelope at all"},
		{"the envelope is empty", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireJQ(t)
			res := runHookScript(t, hookScriptRun{
				script:  chatDrainScript,
				stdin:   tc.stdin,
				readOut: chatDrainedMessages,
			})
			assertStopAllowed(t, res, callIdle)
		})
	}
}

// Without jq the drained text could not be encoded back to the harness, so the
// messages are left where they are for the next turn rather than read out into
// nowhere.
func TestChatHookScripts_DrainWithoutJQLeavesTheInboxAlone(t *testing.T) {
	res := runHookScript(t, hookScriptRun{
		script:  chatDrainScript,
		stdin:   `{"stop_hook_active":false}`,
		readOut: chatDrainedMessages,
		noJQ:    true,
	})
	assertStopAllowed(t, res, callIdle)
}

// Nothing waiting, or nobody to ask: the turn ends as it would have without the
// hook, and the member is reported idle so the daemon may nudge it when the
// next message arrives.
func TestChatHookScripts_DrainAllowsTheStopWithNothingToDeliver(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  hookScriptRun
	}{
		{"the inbox is empty", hookScriptRun{readOut: chatEmptyInbox}},
		{"the read printed nothing", hookScriptRun{}},
		{"the daemon is unreachable", hookScriptRun{readExit: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireJQ(t)
			tc.run.script = chatDrainScript
			tc.run.stdin = `{"stop_hook_active":false}`
			assertStopAllowed(t, runHookScript(t, tc.run), callRead, callIdle)
		})
	}
}

// Messages in hand: the stop is blocked so they reach the agent, with the text
// carried through jq intact — quotes and line breaks included. decision and
// reason and nothing else, since a rejected output after a consuming read is
// exactly the lost message this script exists to prevent. No idle is reported
// either: the turn is about to continue, so the member is running.
func TestChatHookScripts_DrainBlocksWithTheMessages(t *testing.T) {
	requireJQ(t)
	res := runHookScript(t, hookScriptRun{
		script:  chatDrainScript,
		stdin:   `{"stop_hook_active":false}`,
		readOut: chatDrainedMessages,
	})

	if res.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr:\n%s", res.exitCode, res.stderr)
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
	if !strings.Contains(reason, chatDrainedMessages) {
		t.Errorf("reason = %q, want it to carry the drained messages verbatim:\n%s",
			reason, chatDrainedMessages)
	}
	if !strings.Contains(reason, "crabswarm chat send") {
		t.Errorf("reason = %q, want it to name how the agent replies", reason)
	}
	if want := []string{callRead}; !slices.Equal(res.calls, want) {
		t.Errorf("crabswarm calls = %v, want %v: a blocked stop is not idle",
			res.calls, want)
	}
}

// The PostToolUse hook hands the messages over as additionalContext rather than
// announcing them: the read already consumed them, so injecting the text is the
// delivery. hookSpecificOutput and nothing else, for the same reason the drain
// keeps to decision and reason.
func TestChatHookScripts_HintDeliversTheMessages(t *testing.T) {
	requireJQ(t)
	res := runHookScript(t, hookScriptRun{
		script:  chatHintScript,
		stdin:   postToolUseEnvelope,
		readOut: chatDrainedMessages,
	})

	if res.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr:\n%s", res.exitCode, res.stderr)
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
	if !strings.Contains(injected, chatDrainedMessages) {
		t.Errorf("additionalContext = %q, want it to carry the messages verbatim:\n%s",
			injected, chatDrainedMessages)
	}
	if want := []string{callRead}; !slices.Equal(res.calls, want) {
		t.Errorf("crabswarm calls = %v, want %v", res.calls, want)
	}
}

// This one runs after every single tool call, so anything short of a message
// actually having arrived ends in silence: an empty inbox, a daemon that is not
// there and a missing jq are none of them news the agent needs mid-task. It
// reports no state either — a member calling tools is plainly not idle.
func TestChatHookScripts_HintIsSilentWithoutMessages(t *testing.T) {
	for _, tc := range []struct {
		name      string
		run       hookScriptRun
		wantCalls []string
	}{
		{"the inbox is empty", hookScriptRun{readOut: chatEmptyInbox}, []string{callRead}},
		{"the read printed nothing", hookScriptRun{}, []string{callRead}},
		{"the daemon is unreachable", hookScriptRun{readExit: 1}, []string{callRead}},
		{"jq is missing", hookScriptRun{readOut: chatDrainedMessages, noJQ: true}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.run.noJQ {
				requireJQ(t)
			}
			tc.run.script = chatHintScript
			tc.run.stdin = postToolUseEnvelope
			res := runHookScript(t, tc.run)

			if res.exitCode != 0 {
				t.Errorf("exit code = %d, want 0\nstderr:\n%s", res.exitCode, res.stderr)
			}
			if res.stdout != "" {
				t.Errorf("stdout = %q, want nothing", res.stdout)
			}
			if res.stderr != "" {
				t.Errorf("stderr = %q, want nothing", res.stderr)
			}
			if !slices.Equal(res.calls, tc.wantCalls) {
				t.Errorf("crabswarm calls = %v, want %v", res.calls, tc.wantCalls)
			}
		})
	}
}

// Both scripts decide whether anything arrived by comparing the read's output
// against a literal sentence, and no Go build step knows that. So the wording is
// pinned from both ends here: what the binary actually prints on an empty
// inbox, and the string each script tests for. Reword the renderer's empty case
// without touching the scripts and this fails, instead of every quiet turn
// silently blocking on "no pending messages" as if it were mail.
func TestChatHookScripts_EmptyInboxLiteralMatchesTheCLI(t *testing.T) {
	cfg := startChatDaemon(t)
	runChat(t, cfg, "tok-ana", "join", "--name", "ana")

	printed := strings.TrimSuffix(runChat(t, cfg, "tok-ana", "read"), "\n")
	if printed != chatEmptyInbox {
		t.Fatalf("chat read on an empty inbox = %q, want %q", printed, chatEmptyInbox)
	}
	for _, name := range []string{chatDrainScript, chatHintScript} {
		b, err := os.ReadFile(chatHookScriptPath(t, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if want := `"` + printed + `"`; !strings.Contains(string(b), want) {
			t.Errorf("%s does not compare the read against %s, "+
				"which is what `crabswarm chat read` prints on an empty inbox",
				name, want)
		}
	}
}
