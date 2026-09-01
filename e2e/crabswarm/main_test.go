package crabswarm_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// crabswarmBin is the path to the built crabswarm binary, set once by TestMain.
var crabswarmBin string

func TestMain(m *testing.M) {
	// Build the crabswarm binary into a temp directory.
	tmp, err := os.MkdirTemp("", "crabswarm-e2e-bin-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	bin := filepath.Join(tmp, "crabswarm")
	build := exec.Command("go", "build", "-o", bin, "./cmd/crabswarm")
	// Build from the repository root.
	build.Dir = repoRoot()
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build crabswarm: %v\n", err)
		os.Exit(1)
	}
	crabswarmBin = bin

	os.Exit(m.Run())
}

func repoRoot() string {
	// Walk up from the test file location to find go.mod.
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("could not find repo root")
		}
		dir = parent
	}
}

// freeAddr reserves an ephemeral loopback port and returns it as host:port. The
// listener is closed before returning so the port is free for the process under
// test to bind. A small race window (another process could grab the port in
// between) is acceptable for e2e: the preview `__serve` command binds a concrete
// port rather than port 0, which would otherwise be unknowable to the caller.
func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve free port: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close reserved listener: %v", err)
	}
	return addr
}

// writeFile writes content to path, failing the test on error.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// waitHealthz polls the preview server's /healthz endpoint until it answers 200
// or the timeout elapses. addr is the server's loopback listen address.
func waitHealthz(ctx context.Context, t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	url := "http://" + addr + "/healthz"
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			t.Fatalf("build healthz request: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("preview server at %s did not become healthy within %s", addr, timeout)
}

// postToolUseEnvelope is the hook input every `hook exec` case below is fed: a
// Claude PostToolUse + Bash envelope. Bash carries no file path, so filetype
// and module-root detection stay out of the picture and the child process runs
// in the test's own working directory.
const postToolUseEnvelope = `{` +
	`"session_id":"sess-e2e",` +
	`"transcript_path":"/tmp/e2e.jsonl",` +
	`"cwd":"/tmp",` +
	`"hook_event_name":"PostToolUse",` +
	`"tool_name":"Bash",` +
	`"tool_input":{"command":"ls"},` +
	`"tool_use_id":"toolu_e2e"}`

// hookResult is a hook invocation's full outcome. A hook decision rides on
// stdout, but a misconfiguration still shows up as an exit code and a stderr
// report, so all three are captured.
type hookResult struct {
	stdout   string
	stderr   string
	exitCode int
}

// runHookExec runs `crabswarm hook exec <args...>` with envelope on stdin.
// Unlike runCrabswarm it does not fail on a non-zero exit: a hook decision is
// always exit 0, so a non-zero code is a plain error (exit 1) and the cases
// below assert on it. Only a failure to run the binary at all is fatal.
//
// An empty temp --config keeps the invocation off the developer's real
// $XDG_CONFIG_HOME/crabswarm/config.json.
func runHookExec(ctx context.Context, t *testing.T, envelope string, args ...string) hookResult {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	writeFile(t, cfgPath, "{}")

	full := append([]string{"hook", "exec", "--config", cfgPath}, args...)
	cmd := exec.CommandContext(ctx, crabswarmBin, full...)
	cmd.Stdin = strings.NewReader(envelope)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if _, ok := errors.AsType[*exec.ExitError](err); !ok {
			t.Fatalf("crabswarm %s: %v\nstdout:\n%s\nstderr:\n%s",
				strings.Join(full, " "), err, stdout.String(), stderr.String())
		}
	}
	return hookResult{
		stdout:   stdout.String(),
		stderr:   stderr.String(),
		exitCode: cmd.ProcessState.ExitCode(),
	}
}

// hookOutput mirrors the wire shape of the hook's JSON stdout. It is spelled
// out here rather than reusing types.SyncHookJSONOutput so the assertions read
// the bytes a hook consumer would see, instead of round-tripping through the
// very types that produced them.
type hookOutput struct {
	Decision           string `json:"decision"`
	Reason             string `json:"reason"`
	SystemMessage      string `json:"systemMessage"`
	HookSpecificOutput struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
	} `json:"hookSpecificOutput"`
}

// decodeHookOutput parses one JSON hook output line.
func decodeHookOutput(t *testing.T, s string) hookOutput {
	t.Helper()
	var out hookOutput
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		t.Fatalf("decode hook output %q: %v", s, err)
	}
	return out
}

// A successful command plus an output template that records additionalContext:
// the success path the built-in behavior cannot express at all. JSON on
// stdout, exit 0.
func TestHookExec_OutputTemplateSuccess(t *testing.T) {
	res := runHookExec(t.Context(), t, postToolUseEnvelope,
		"true",
		`{{ if .Success }}{{ context (printf "ran %s exit=%d" .Command .ExitCode) }}{{ end }}`,
	)

	if res.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s",
			res.exitCode, res.stdout, res.stderr)
	}
	if res.stderr != "" {
		t.Errorf("stderr = %q, want empty", res.stderr)
	}

	out := decodeHookOutput(t, res.stdout)
	if got := out.HookSpecificOutput.HookEventName; got != "PostToolUse" {
		t.Errorf("hookEventName = %q, want %q", got, "PostToolUse")
	}
	if got, want := out.HookSpecificOutput.AdditionalContext, "ran true exit=0"; got != want {
		t.Errorf("additionalContext = %q, want %q", got, want)
	}
	if out.Decision != "" {
		t.Errorf("decision = %q, want empty: a successful command must not block", out.Decision)
	}
}

// hookJSONLine returns the single JSON line a hook wrote to stdout, failing
// when stdout is not exactly one line. A hook decision is always one crafted
// JSON object, so anything else on the stream is a leak the block cases below
// must not tolerate.
func hookJSONLine(t *testing.T, res hookResult) string {
	t.Helper()
	if res.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s",
			res.exitCode, res.stdout, res.stderr)
	}
	if res.stderr != "" {
		t.Errorf("stderr = %q, want empty", res.stderr)
	}
	line, ok := strings.CutSuffix(res.stdout, "\n")
	if !ok || strings.Contains(line, "\n") {
		t.Fatalf("stdout should be exactly one JSON line; got:\n%s", res.stdout)
	}
	return line
}

// A failing command plus a blockDecision-only output template: the block rides
// on stdout as JSON with exit 0, like every other hook output. Nothing reaches
// stderr and no exit code carries meaning.
func TestHookExec_OutputTemplateBlocksOnFailure(t *testing.T) {
	res := runHookExec(t.Context(), t, postToolUseEnvelope,
		"false",
		`{{ if not .Success }}`+
			`{{ blockDecision (printf "false failed with exit=%d" .ExitCode) }}{{ end }}`,
	)

	out := decodeHookOutput(t, hookJSONLine(t, res))
	if out.Decision != "block" {
		t.Errorf("decision = %q, want %q", out.Decision, "block")
	}
	if want := "false failed with exit=1"; out.Reason != want {
		t.Errorf("reason = %q, want %q", out.Reason, want)
	}
}

// A block combined with any other field keeps every field: the whole output
// goes out as one JSON object, with nothing dropped on the way.
func TestHookExec_OutputTemplateMixedStaysJSON(t *testing.T) {
	res := runHookExec(t.Context(), t, postToolUseEnvelope,
		"false",
		`{{ blockDecision "blocked" }}{{ systemMessage "note for the user" }}`,
	)

	if res.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s",
			res.exitCode, res.stdout, res.stderr)
	}
	if res.stderr != "" {
		t.Errorf("stderr = %q, want empty", res.stderr)
	}

	out := decodeHookOutput(t, res.stdout)
	if out.Decision != "block" {
		t.Errorf("decision = %q, want %q", out.Decision, "block")
	}
	if out.Reason != "blocked" {
		t.Errorf("reason = %q, want %q", out.Reason, "blocked")
	}
	if want := "note for the user"; out.SystemMessage != want {
		t.Errorf("systemMessage = %q, want %q", out.SystemMessage, want)
	}
}

// An output template that records nothing is a plain allow: nothing on
// stdout, exit 0. The command is a failing one on purpose — this is the
// fire-and-forget idiom the shipped chat hooks use, where the template's job
// is to keep a failed report from blocking the turn the built-in behavior
// would have blocked.
func TestHookExec_OutputTemplateRecordingNothingIsPlainAllow(t *testing.T) {
	res := runHookExec(t.Context(), t, postToolUseEnvelope,
		"false",
		`{{/* records nothing, so a failed command never blocks */}}`,
	)

	if res.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s",
			res.exitCode, res.stdout, res.stderr)
	}
	if res.stdout != "" {
		t.Errorf("stdout = %q, want empty: a plain allow emits no hook output", res.stdout)
	}
	if res.stderr != "" {
		t.Errorf("stderr = %q, want empty", res.stderr)
	}
}

// An event-scoped function called for the wrong event is a misconfiguration,
// not a hook decision: the render fails and the CLI exits 1 with the mismatch
// reported on stderr.
func TestHookExec_OutputTemplateEventMismatch(t *testing.T) {
	res := runHookExec(t.Context(), t, postToolUseEnvelope,
		"true",
		`{{ permissionAllow }}`, // PermissionRequest-only, on a PostToolUse envelope
	)

	if res.exitCode != 1 {
		t.Fatalf("exit code = %d, want 1\nstdout:\n%s\nstderr:\n%s",
			res.exitCode, res.stdout, res.stderr)
	}
	if res.stdout != "" {
		t.Errorf("stdout = %q, want empty: a failed render emits no hook output", res.stdout)
	}
	// text/template decorates func errors with parse positions; only the
	// mismatch itself is pinned.
	for _, want := range []string{"permissionAllow", "not supported for hook event", "PostToolUse"} {
		if !strings.Contains(res.stderr, want) {
			t.Errorf("stderr missing %q; got:\n%s", want, res.stderr)
		}
	}
}

// --dry-run prints the rendered command line and, underneath it, the JSON the
// output template would produce against a synthetic success result — without
// running anything.
func TestHookExec_OutputTemplateDryRun(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "marker")

	res := runHookExec(t.Context(), t, postToolUseEnvelope,
		"--dry-run",
		"touch "+marker,
		`{{ if .Success }}{{ context "would have run" }}{{ end }}`,
	)

	if res.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s",
			res.exitCode, res.stdout, res.stderr)
	}
	if res.stderr != "" {
		t.Errorf("stderr = %q, want empty", res.stderr)
	}

	lines := strings.Split(strings.TrimSuffix(res.stdout, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("stdout should be the command line then the preview JSON; got:\n%s", res.stdout)
	}
	if want := "touch " + marker; lines[0] != want {
		t.Errorf("rendered command = %q, want %q", lines[0], want)
	}
	out := decodeHookOutput(t, lines[1])
	if got, want := out.HookSpecificOutput.AdditionalContext, "would have run"; got != want {
		t.Errorf("preview additionalContext = %q, want %q", got, want)
	}

	if _, err := os.Stat(marker); err == nil {
		t.Errorf("--dry-run executed the command: %s exists", marker)
	}
}

// Without an output template the built-in behavior is unchanged: a non-zero
// exit blocks with the formatted failure as the reason. blockReason omits the
// `output:` section when the command captured nothing, as `false` does.
func TestHookExec_NoOutputTemplateBlocksOnFailure(t *testing.T) {
	res := runHookExec(t.Context(), t, postToolUseEnvelope, "false")

	line := hookJSONLine(t, res)
	// Compared as the raw bytes a hook consumer reads, not field by field: the
	// omitted `output:` section is half of what this case pins down, and the
	// reason's own trailing newline — blockReason's, now carried inside the
	// JSON string rather than written to stderr — is the other half.
	want := `{"decision":"block","reason":"command failed: false\nexit: exit status 1\n"}`
	if line != want {
		t.Errorf("stdout = %q, want %q", line, want)
	}
	out := decodeHookOutput(t, line)
	if out.Decision != "block" {
		t.Errorf("decision = %q, want %q", out.Decision, "block")
	}
}
