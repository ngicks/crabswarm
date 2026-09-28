package crabswarm_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `crabswarm hook exec` fed what a harness hands a hook besides the PostToolUse
// envelope the cases in main_test.go use: the Stop envelope
// crabswarm-issues-lint is wired to, an event no shipped file wires, and stdin
// that is no envelope at all. The hook files the apm packages ship are pinned
// at the end: every entry in them runs through `hook exec`.

// The envelopes a harness writes to a hook's stdin. Only the fields a template
// reads matter, the Stop flag above all, but each carries the envelope metadata
// a real harness sends, since `hook exec` parses the whole thing before it
// renders anything.
const (
	stopEnvelope = `{` +
		`"session_id":"sess-e2e",` +
		`"transcript_path":"/tmp/e2e.jsonl",` +
		`"cwd":"/tmp",` +
		`"hook_event_name":"Stop",` +
		`"stop_hook_active":false}`
	stopActiveEnvelope = `{` +
		`"session_id":"sess-e2e",` +
		`"transcript_path":"/tmp/e2e.jsonl",` +
		`"cwd":"/tmp",` +
		`"hook_event_name":"Stop",` +
		`"stop_hook_active":true}`
	// Codex's approval dialog, which no shipped file wires. It is the envelope
	// that says whether `hook exec` speaks the rest of a harness's surface at all.
	permissionRequestEnvelope = `{` +
		`"session_id":"sess-e2e",` +
		`"transcript_path":"/tmp/e2e.jsonl",` +
		`"cwd":"/tmp",` +
		`"hook_event_name":"PermissionRequest",` +
		`"tool_name":"Bash",` +
		`"tool_input":{"command":"ls"}}`
)

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

// commandRan reports whether the command a case rendered left marker behind.
func commandRan(marker string) bool {
	_, err := os.Stat(marker)
	return err == nil
}

// The shape crabswarm-issues-lint ships: a Stop hook with a command and no
// output template. A command that succeeds lets the turn end, and one that
// fails blocks it with the failure as the reason, which keeps the turn going
// until the command is satisfied.
func TestHookExec_StopBlocksOnFailure(t *testing.T) {
	t.Run("the command succeeds", func(t *testing.T) {
		assertHookIsSilent(t, runHookExec(t.Context(), t, stopEnvelope, "true"))
	})

	t.Run("the command fails", func(t *testing.T) {
		line := hookJSONLine(t, runHookExec(t.Context(), t, stopEnvelope, "false"))
		want := `{"decision":"block","reason":"command failed: false\nexit: exit status 1\n"}`
		if line != want {
			t.Errorf("stdout = %q, want %q", line, want)
		}
	})
}

// A Stop envelope's stop_hook_active reaches the command template, so a Stop
// hook can stand aside on the turn its own earlier block bought instead of
// blocking it again. With the flag set the command renders empty, and an empty
// command runs nothing and allows.
func TestHookExec_StopHookActiveReachesTheTemplate(t *testing.T) {
	for _, tc := range []struct {
		name     string
		envelope string
		wantRan  bool
	}{
		{"the first stop", stopEnvelope, true},
		{"an earlier block is in force", stopActiveEnvelope, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "marker")
			assertHookIsSilent(t, runHookExec(t.Context(), t, tc.envelope,
				"{{ if not .Input.StopHookActive }}touch "+marker+"{{ end }}"))
			if ran := commandRan(marker); ran != tc.wantRan {
				t.Errorf("the command ran = %v, want %v", ran, tc.wantRan)
			}
		})
	}
}

// An event no shipped file wires still parses: `hook exec` decodes the envelope
// into its typed variant and runs the command. The envelope surface is the
// harness's, not a list of the events the shipped files happen to use, so a
// file that wires Codex's approval dialog later does not discover that the
// parser cannot read it.
func TestHookExec_AnUnwiredEventStillParses(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "marker")

	assertHookIsSilent(t, runHookExec(t.Context(), t, permissionRequestEnvelope, "touch "+marker))
	if !commandRan(marker) {
		t.Error("the command never ran")
	}
}

// Stdin that is no envelope is a plain error, not a decision: exit 1 with the
// reason on stderr, nothing on stdout, and the command never run. That is
// never a block, and a harness that sends unparseable JSON is broken in a way
// worth hearing about.
func TestHookExec_UnparseableEnvelopeFailsWithoutBlocking(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "marker")

	res := runHookExec(t.Context(), t, "not an envelope at all", "touch "+marker)
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
	if commandRan(marker) {
		t.Error("the command ran on an envelope that did not parse")
	}
}

// hookFile is a hook file's shape on both harnesses: events, each holding
// matcher groups, each holding the commands to run.
type hookFile struct {
	Hooks map[string][]hookGroup `json:"hooks"`
}

type hookGroup struct {
	Matcher string      `json:"matcher"`
	Hooks   []hookEntry `json:"hooks"`
}

type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

// apmHookFiles is every hook file the apm packages ship: the plugin's file of
// each package apm_package_test.go says hooks Claude Code, and the Codex file
// of each package it says hooks Codex.
func apmHookFiles() []string {
	var files []string
	for _, name := range apmPluginHookPackages {
		files = append(files, filepath.Join(apmSkillPluginDir(name), "hooks", "hooks.json"))
	}
	for _, name := range apmCodexHookPackages {
		files = append(
			files,
			filepath.Join(apmPackageDir(name), ".apm", "hooks", "codex-hooks.json"),
		)
	}
	return files
}

// Everything a shipped hook needs ships inside `crabswarm hook exec`: no shell
// scripts to copy alongside the JSON, no `jq` to have installed, and no plugin
// root to resolve. Each entry also carries a timeout, so a command that wedges
// cannot hold the session for as long as it runs.
func TestApmHooks_RunThroughHookExec(t *testing.T) {
	for _, path := range apmHookFiles() {
		rel, err := filepath.Rel(repoRoot(), path)
		if err != nil {
			t.Fatalf("relate %s to the checkout: %v", path, err)
		}
		t.Run(rel, func(t *testing.T) {
			var file hookFile
			readJSONFile(t, path, &file)
			if len(file.Hooks) == 0 {
				t.Fatalf("%s wires no events", rel)
			}
			for event, groups := range file.Hooks {
				for _, group := range groups {
					for _, h := range group.Hooks {
						assertSelfContainedHookEntry(t, event, h)
					}
				}
			}
		})
	}
}

// The directory half of the same rule, which no command string would ever
// show: no package ships a scripts directory beside its hook files.
func TestApmHooks_ShipNoScripts(t *testing.T) {
	for _, name := range apmPackages {
		if _, err := os.Stat(filepath.Join(apmPackageDir(name), "scripts")); err == nil {
			t.Errorf("apm-package/%s/scripts exists; hooks run through `crabswarm hook exec`", name)
		}
	}
}

// assertSelfContainedHookEntry pins one entry's command against the ways hook
// wiring can reach outside the binary, and against a missing timeout.
func assertSelfContainedHookEntry(t *testing.T, event string, h hookEntry) {
	t.Helper()
	if !strings.HasPrefix(h.Command, "crabswarm hook exec ") {
		t.Errorf("%s command %q does not run through `crabswarm hook exec`", event, h.Command)
	}
	for _, banned := range []string{"jq", "CLAUDE_PLUGIN_ROOT", "scripts/"} {
		if strings.Contains(h.Command, banned) {
			t.Errorf("%s command %q depends on %q", event, h.Command, banned)
		}
	}
	if h.Timeout <= 0 {
		t.Errorf("%s command %q carries no timeout", event, h.Command)
	}
	if h.Type != "command" {
		t.Errorf("%s entry type = %q, want %q", event, h.Type, "command")
	}
}
