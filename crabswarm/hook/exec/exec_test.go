package exec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ngicks/crabswarm/internal/templateutil"
	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
	"github.com/ngicks/crabswarm/pkg/claudehook/types"
	"github.com/ngicks/crabswarm/pkg/filetype"
	"gotest.tools/v3/assert"
)

// marshalHookInput serializes a typed hook envelope.
func marshalHookInput(t *testing.T, in types.HookInput_Value) *bytes.Reader {
	t.Helper()
	data, err := json.Marshal(types.NewHookInput(in))
	assert.NilError(t, err)
	return bytes.NewReader(data)
}

// preToolUseEnvelope builds a PreToolUse envelope. tool and toolInput
// must agree per the SDK's discriminator → schema mapping.
func preToolUseEnvelope(
	t *testing.T,
	tool string,
	toolInput types.ToolInputSchemas_Value,
) *bytes.Reader {
	t.Helper()
	return marshalHookInput(t, &types.PreToolUseHookInput{
		BaseHookInput: types.BaseHookInput{
			SessionID:      "sess-1",
			TranscriptPath: "/tmp/x.jsonl",
			Cwd:            "/work",
		},
		HookEventName: types.HookEventPreToolUse,
		ToolName:      tool,
		ToolInput:     types.NewToolInputSchemas(toolInput),
		ToolUseID:     "toolu_1",
	})
}

// editEnvelope produces a PreToolUse + Edit envelope referencing filePath.
func editEnvelope(t *testing.T, filePath string) *bytes.Reader {
	t.Helper()
	return preToolUseEnvelope(t, types.ToolNameEdit, &types.FileEditInput{
		FilePath: filePath,
	})
}

// bashEnvelope produces a PreToolUse + Bash envelope.
func bashEnvelope(t *testing.T, command string) *bytes.Reader {
	t.Helper()
	return preToolUseEnvelope(t, types.ToolNameBash, &types.BashInput{
		Command: command,
	})
}

func touch(t *testing.T, path string) {
	t.Helper()
	assert.NilError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	assert.NilError(t, err)
	assert.NilError(t, f.Close())
}

func assertPassThrough(t *testing.T, err error) {
	t.Helper()
	var he *handler.HandlerError
	if !errors.As(err, &he) {
		t.Fatalf("expected *handler.HandlerError, got %T: %v", err, err)
	}
	if he.Output != nil {
		t.Fatalf("expected nil Output (pass-through), got %+v", he.Output)
	}
}

// goRustTables is an explicit filetype config used by tests that want to
// exercise the overlay path independently of the embedded defaults.
// Run / Render already layer Default underneath, so callers that just
// need "go" / "rust" detection can omit this and rely on defaults.
func goRustTables() []filetype.Config {
	return []filetype.Config{
		{
			Ext:      map[string]string{"go": "go"},
			Filename: map[string]string{"go.mod": "go"},
			RootMarkers: map[string][]filetype.MarkerGroup{
				"go": {{"go.mod"}, {".git"}},
			},
		},
		{
			Ext:      map[string]string{"rs": "rust"},
			Filename: map[string]string{"Cargo.toml": "rust"},
			RootMarkers: map[string][]filetype.MarkerGroup{
				"rust": {{"Cargo.toml"}, {".git"}},
			},
		},
	}
}

func TestRender_RendersTemplate(t *testing.T) {
	tmp := t.TempDir()
	touch(t, filepath.Join(tmp, "go.mod"))
	editPath := filepath.Join(tmp, "pkg", "foo.go")
	touch(t, editPath)

	cfg := Config{Filetypes: goRustTables()}
	opt := Option{Template: "ft={{ .Filetype }} root={{ .Root }} file={{ .File }}"}
	r := editEnvelope(t, editPath)

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "ft=go root="+tmp+" file="+editPath+"\n")
}

func TestRender_FlatBaseFields(t *testing.T) {
	cfg := Config{}
	opt := Option{Template: "cwd={{ .Cwd }} sid={{ .SessionID }}"}
	r := bashEnvelope(t, "ls")

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "cwd=/work sid=sess-1\n")
}

func TestRender_InputExposesTypedSubtype(t *testing.T) {
	// Confirm .Input is a typed *PreToolUseHookInput by reaching for
	// ToolUseID, which is unique to that subtype.
	cfg := Config{}
	opt := Option{Template: "uid={{ .Input.ToolUseID }}"}
	r := editEnvelope(t, "/x")

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "uid=toolu_1\n")
}

func TestRender_TemplateFuncs(t *testing.T) {
	tmp := t.TempDir()
	editPath := filepath.Join(tmp, "x.go")
	touch(t, editPath)

	cfg := Config{}
	opt := Option{Template: "{{ basename .File }}|{{ ext .File }}|{{ quote .File }}"}
	r := editEnvelope(t, editPath)

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "x.go|.go|'"+editPath+"'\n")
}

// commandName / commandArgs split the Bash tool's command reached through
// .Input, so a Bash-gated hook can dispatch on the invoked command.
func TestRender_CommandNameAndArgsOnBashInput(t *testing.T) {
	cfg := Config{}
	opt := Option{
		Template: `echo {{ commandName .Input.ToolInput.GetValue.Command }}` +
			`|{{ quoteJoin (commandArgs .Input.ToolInput.GetValue.Command) }}`,
	}
	r := bashEnvelope(t, "git commit -m 'two words'")

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), `echo git|"git" "commit" "-m" "two words"`+"\n")
}

func TestRender_QuoteEscapesSingleQuotes(t *testing.T) {
	cfg := Config{}
	opt := Option{Template: `{{ quote "it's" }}`}
	r := bashEnvelope(t, "ls")

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), `'it'\''s'`+"\n")
}

func TestRender_InvalidTemplateReturnsError(t *testing.T) {
	cfg := Config{}
	opt := Option{Template: "{{ .Unclosed"}
	r := bashEnvelope(t, "ls")

	var buf bytes.Buffer
	err := Render(context.Background(), r, &buf, cfg, opt)
	assert.Assert(t, err != nil)
	var he *handler.HandlerError
	assert.Assert(t, !errors.As(err, &he))
	assert.Assert(t, strings.Contains(err.Error(), "parsing template"))
}

func TestRender_InvalidJSONReturnsError(t *testing.T) {
	cfg := Config{}
	opt := Option{Template: "x"}
	var buf bytes.Buffer
	err := Render(context.Background(), strings.NewReader("not json"), &buf, cfg, opt)
	assert.Assert(t, err != nil)
	var he *handler.HandlerError
	assert.Assert(t, !errors.As(err, &he))
	assert.Assert(t, strings.Contains(err.Error(), "parsing hook input"))
}

func TestRender_MissingTemplateErrors(t *testing.T) {
	cfg := Config{}
	opt := Option{} // no Template
	var buf bytes.Buffer
	err := Render(context.Background(), bashEnvelope(t, "ls"), &buf, cfg, opt)
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "Template is required"))
}

func TestRender_DefaultFiletypesApplyWithoutCallerConfig(t *testing.T) {
	// Run/Render layer Default underneath. A .go file is detected even
	// when the caller passes Config{} (no leaves of their own).
	tmp := t.TempDir()
	touch(t, filepath.Join(tmp, "go.mod"))
	editPath := filepath.Join(tmp, "pkg", "x.go")
	touch(t, editPath)

	cfg := Config{}
	opt := Option{Template: "ft={{ .Filetype }} root={{ .Root }}"}
	r := editEnvelope(t, editPath)

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "ft=go root="+tmp+"\n")
}

func TestRender_CallerLeafOverridesDefault(t *testing.T) {
	// Caller-supplied leaves win on key conflicts. Re-mapping ".go" to
	// a different name proves Default is the lowest-priority overlay.
	tmp := t.TempDir()
	editPath := filepath.Join(tmp, "x.go")
	touch(t, editPath)

	cfg := Config{Filetypes: []filetype.Config{
		{Ext: map[string]string{"go": "golang-override"}},
	}}
	opt := Option{Template: "ft={{ .Filetype }}"}
	r := editEnvelope(t, editPath)

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "ft=golang-override\n")
}

func TestRun_RunsRenderedCommand(t *testing.T) {
	tmp := t.TempDir()
	marker := filepath.Join(tmp, "ran")

	cfg := Config{}
	opt := Option{Template: "touch " + marker}
	r := bashEnvelope(t, "ls")

	assertPassThrough(t, Run(context.Background(), r, cfg, opt))

	_, statErr := os.Stat(marker)
	assert.NilError(t, statErr)
}

func TestRun_QuotedArgsViaShellwords(t *testing.T) {
	tmp := t.TempDir()
	marker := filepath.Join(tmp, "has space.txt")

	cfg := Config{}
	opt := Option{Template: "touch {{ quote .File }}"}
	r := editEnvelope(t, marker)

	assertPassThrough(t, Run(context.Background(), r, cfg, opt))

	_, statErr := os.Stat(marker)
	assert.NilError(t, statErr)
}

func TestRun_SetsCwdToModuleRoot(t *testing.T) {
	tmp := t.TempDir()
	tmpResolved, err := filepath.EvalSymlinks(tmp)
	assert.NilError(t, err)
	touch(t, filepath.Join(tmpResolved, "go.mod"))
	editPath := filepath.Join(tmpResolved, "pkg", "foo.go")
	touch(t, editPath)

	out := filepath.Join(tmpResolved, "pwd.txt")
	cfg := Config{Filetypes: goRustTables()}
	// `sh -c 'pwd > out'` so we still get a value into the file even though
	// Run doesn't go through a shell — the rendered argv is [sh -c "pwd > out"].
	opt := Option{
		Template: "sh -c " + templateutil.ShellQuote("pwd > "+templateutil.ShellQuote(out)),
	}
	r := editEnvelope(t, editPath)

	assertPassThrough(t, Run(context.Background(), r, cfg, opt))

	got, err := os.ReadFile(out)
	assert.NilError(t, err)
	pwd, err := filepath.EvalSymlinks(strings.TrimSpace(string(got)))
	assert.NilError(t, err)
	assert.Equal(t, pwd, tmpResolved)
}

func TestRun_FailureReturnsBlockDecision(t *testing.T) {
	// `false` exits 1 with no output. Should produce a block HandlerError
	// so the hook protocol surfaces the failure to the agent.
	cfg := Config{}
	opt := Option{Template: "false"}
	r := bashEnvelope(t, "ls")

	err := Run(context.Background(), r, cfg, opt)
	var he *handler.HandlerError
	assert.Assert(t, errors.As(err, &he), "expected *handler.HandlerError, got %T", err)
	assert.Assert(t, he.Output != nil, "expected non-nil Output (block)")
	assert.Assert(t, he.Output.Decision != nil, "expected Decision set")
	assert.Equal(t, *he.Output.Decision, types.HookDecisionBlock)
	assert.Assert(t, he.Output.Reason != nil, "expected Reason set")
	assert.Assert(
		t,
		strings.Contains(*he.Output.Reason, "command failed: false"),
		"reason: %q", *he.Output.Reason,
	)
}

func TestRun_FailureCapturesStderrAsReason(t *testing.T) {
	cfg := Config{}
	opt := Option{Template: "sh -c " + templateutil.ShellQuote("echo lint-error >&2; exit 1")}
	r := bashEnvelope(t, "ls")

	err := Run(context.Background(), r, cfg, opt)
	var he *handler.HandlerError
	assert.Assert(t, errors.As(err, &he), "expected *handler.HandlerError, got %T", err)
	assert.Assert(t, he.Output != nil && he.Output.Reason != nil)
	reason := *he.Output.Reason
	assert.Assert(
		t,
		strings.Contains(reason, "lint-error"),
		"reason should include captured stderr; got %q", reason,
	)
	assert.Assert(
		t,
		strings.Contains(reason, "output:"),
		"reason should have an output: section; got %q", reason,
	)
}

func TestRun_FailureCapturesStdoutAsReason(t *testing.T) {
	cfg := Config{}
	opt := Option{Template: "sh -c " + templateutil.ShellQuote("echo on-stdout; exit 7")}
	r := bashEnvelope(t, "ls")

	err := Run(context.Background(), r, cfg, opt)
	var he *handler.HandlerError
	assert.Assert(t, errors.As(err, &he))
	reason := *he.Output.Reason
	assert.Assert(t, strings.Contains(reason, "on-stdout"), "reason: %q", reason)
}

func TestRun_NonExitErrorPropagatesAsPlainError(t *testing.T) {
	// Binary that doesn't exist — os/exec returns a non-ExitError. That
	// means the command never ran (vs ran-and-failed), so it should NOT
	// be a block decision; the parent should see a plain error.
	cfg := Config{}
	opt := Option{Template: "/no/such/binary-does-not-exist"}
	r := bashEnvelope(t, "ls")

	err := Run(context.Background(), r, cfg, opt)
	assert.Assert(t, err != nil)
	var he *handler.HandlerError
	assert.Assert(
		t,
		!errors.As(err, &he),
		"command-not-found should NOT produce a block HandlerError; got %T", err,
	)
	assert.Assert(t, strings.Contains(err.Error(), "running "), "err: %v", err)
}

func TestRun_SkipsEmptyRendered(t *testing.T) {
	// Empty rendered template (e.g., {{ if false }}cmd{{ end }}) is skipped
	// — passes through without execution.
	cfg := Config{}
	opt := Option{Template: "{{ if false }}/usr/bin/false{{ end }}"}
	r := bashEnvelope(t, "ls")

	assertPassThrough(t, Run(context.Background(), r, cfg, opt))
}

func TestRender_FilterAllows(t *testing.T) {
	tmp := t.TempDir()
	editPath := filepath.Join(tmp, "x.go")
	touch(t, editPath)

	cfg := Config{Filetypes: goRustTables()}
	opt := Option{
		Template: "ran:{{ .Filetype }}",
		Filter:   []string{"go", "rust"},
	}
	r := editEnvelope(t, editPath)

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "ran:go\n")
}

func TestRender_FilterBlocks(t *testing.T) {
	tmp := t.TempDir()
	editPath := filepath.Join(tmp, "x.go")
	touch(t, editPath)

	cfg := Config{Filetypes: goRustTables()}
	opt := Option{
		Template: "should-not-run",
		Filter:   []string{"rust"},
	}
	r := editEnvelope(t, editPath)

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "")
}

func TestRender_FilterBlocksUnknownFiletype(t *testing.T) {
	cfg := Config{Filetypes: goRustTables()}
	opt := Option{
		Template: "should-not-run",
		Filter:   []string{"go"},
	}
	r := bashEnvelope(t, "ls")

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "")
}

func TestRun_FilterBlocksExecution(t *testing.T) {
	tmp := t.TempDir()
	marker := filepath.Join(tmp, "should-not-exist")
	editPath := filepath.Join(tmp, "x.go")
	touch(t, editPath)

	cfg := Config{Filetypes: goRustTables()}
	opt := Option{
		Template: "touch " + marker,
		Filter:   []string{"rust"}, // detected will be "go"
	}
	r := editEnvelope(t, editPath)

	assertPassThrough(t, Run(context.Background(), r, cfg, opt))

	_, statErr := os.Stat(marker)
	assert.Assert(t, statErr != nil, "marker should NOT exist when filter blocks")
}

func TestDefault_HasCommonFiletypes(t *testing.T) {
	c := Default()
	merged := filetype.MergeAll(c.Filetypes)
	for _, want := range []string{"go", "rust", "python", "javascript", "typescript"} {
		_, ok := merged.RootMarkers[want]
		assert.Assert(t, ok, "default config missing filetype %q", want)
	}
	got, ok := merged.Detect("/x/y/foo.go")
	assert.Assert(t, ok)
	assert.Equal(t, got, "go")
}

func TestConfig_JSONRoundTrip(t *testing.T) {
	cfg := Config{
		Filetypes: []filetype.Config{
			{
				Ext:      map[string]string{"go": "go"},
				Filename: map[string]string{"go.mod": "go"},
				RootMarkers: map[string][]filetype.MarkerGroup{
					"go": {{"go.mod"}, {".git"}},
				},
			},
		},
	}
	data, err := json.Marshal(cfg)
	assert.NilError(t, err)

	var out Config
	assert.NilError(t, json.Unmarshal(data, &out))
	assert.DeepEqual(t, cfg, out)
}

func TestConfig_JSONShape(t *testing.T) {
	// Top-level: object with just "filetypes" — no more "templates".
	cfg := Config{
		Filetypes: []filetype.Config{
			{Ext: map[string]string{"go": "go"}},
		},
	}
	data, err := json.Marshal(cfg)
	assert.NilError(t, err)
	var top map[string]any
	assert.NilError(t, json.Unmarshal(data, &top))
	_, ok := top["filetypes"]
	assert.Assert(t, ok, "expected top-level 'filetypes' key")
	_, ok = top["templates"]
	assert.Assert(t, !ok, "should not have 'templates' key anymore")
	arr, ok := top["filetypes"].([]any)
	assert.Assert(t, ok, "filetypes should be a JSON array")
	assert.Equal(t, len(arr), 1)
	leaf, ok := arr[0].(map[string]any)
	assert.Assert(t, ok)
	_, ok = leaf["ext"]
	assert.Assert(t, ok, "leaf should have an ext map")
}

// Sanity check that .Input.Cwd works through the embedded BaseHookInput.
func TestRender_InputBaseFieldThroughEmbed(t *testing.T) {
	cfg := Config{}
	opt := Option{Template: "{{ .Input.Cwd }}"}
	r := bashEnvelope(t, "ls")

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "/work\n")
}

func TestRender_UnknownEvent(t *testing.T) {
	cfg := Config{}
	opt := Option{Template: "event={{ .Event }}"}
	r := marshalHookInput(t, &types.HookInputUnknown{
		UnknownUnion: types.UnknownUnion{
			Raw: json.RawMessage(`{` +
				`"session_id":"sess",` +
				`"transcript_path":"/tmp/x",` +
				`"cwd":"/w",` +
				`"hook_event_name":"SomeFutureEvent"` +
				`}`),
		},
	})

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "event=SomeFutureEvent\n")
}

// codexApplyPatchEnvelope builds a raw Codex PostToolUse + apply_patch
// envelope. Unlike the typed helpers above it emits JSON by hand so the
// shape matches what Codex actually sends — notably tool_response as a
// bare string, which is the payload that used to break parsing.
func codexApplyPatchEnvelope(t *testing.T, cwd, command, toolResponse string) *bytes.Reader {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"session_id":      "sess-codex",
		"transcript_path": "/tmp/x.jsonl",
		"cwd":             cwd,
		"hook_event_name": "PostToolUse",
		"tool_name":       "apply_patch",
		"tool_input":      map[string]any{"command": command},
		"tool_response":   toolResponse,
		"tool_use_id":     "call_codex",
	})
	assert.NilError(t, err)
	return bytes.NewReader(data)
}

// The report's exact scenario: --ft go 'golangci-lint fmt {{.File}}' on a
// Codex apply_patch that adds one .go file. .File must resolve to the
// path made absolute against cwd, with filetype and root detected from it.
func TestRender_CodexApplyPatch_SingleFile(t *testing.T) {
	tmp := t.TempDir()
	touch(t, filepath.Join(tmp, "go.mod"))
	rel := filepath.Join("pkg", "foo.go")
	touch(t, filepath.Join(tmp, rel))

	cfg := Config{Filetypes: goRustTables()}
	opt := Option{
		Template: "ft={{ .Filetype }} root={{ .Root }} file={{ .File }}",
		Filter:   []string{"go"},
	}
	cmd := "*** Begin Patch\n*** Add File: " + rel + "\n+package foo\n*** End Patch\n"
	r := codexApplyPatchEnvelope(t, tmp, cmd, "Exit code: 0\nOutput: ok\n")

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "ft=go root="+tmp+" file="+filepath.Join(tmp, rel)+"\n")
}

// apply_patch can touch several files at once. .Files exposes them all
// (resolved absolute), rename targets replace the old path, deletes are
// dropped, and .File is the first surviving path.
func TestRender_CodexApplyPatch_MultipleFilesAndRename(t *testing.T) {
	cfg := Config{}
	opt := Option{Template: "first={{ .File }}\n{{ range .Files }}f={{ . }}\n{{ end }}"}
	cmd := "*** Begin Patch\n" +
		"*** Update File: pkg/old.go\n*** Move to: pkg/new.go\n@@\n-x\n+y\n" +
		"*** Add File: cmd/added.go\n+package cmd\n" +
		"*** Delete File: pkg/gone.go\n" +
		"*** End Patch\n"
	r := codexApplyPatchEnvelope(t, "/repo", cmd, "ok")

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(
		t,
		buf.String(),
		"first=/repo/pkg/new.go\nf=/repo/pkg/new.go\nf=/repo/cmd/added.go\n\n",
	)
}

// A delete-only patch leaves no file to act on; with an --ft gate the
// invocation passes through without rendering — the report's "nothing to
// format" case.
func TestRender_CodexApplyPatch_DeleteOnlyFilteredPassThrough(t *testing.T) {
	cfg := Config{Filetypes: goRustTables()}
	opt := Option{Template: "fmt {{ .File }}", Filter: []string{"go"}}
	cmd := "*** Begin Patch\n*** Delete File: pkg/gone.go\n*** End Patch\n"
	r := codexApplyPatchEnvelope(t, "/repo", cmd, "ok")

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "")
}

// Regression: a string-valued tool_response (Codex apply_patch) must not
// abort parsing before the template runs.
func TestRender_CodexApplyPatch_StringToolResponseParses(t *testing.T) {
	cfg := Config{}
	opt := Option{Template: "ok={{ .ToolName }}"}
	cmd := "*** Begin Patch\n*** Add File: a.go\n+package a\n*** End Patch\n"
	r := codexApplyPatchEnvelope(t, "/repo", cmd, "Exit code: 0\nWall time: 0.5 seconds\n")

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "ok=apply_patch\n")
}

// Run executes the rendered command for a Codex apply_patch just like a
// Claude edit, with cwd set to the detected module root.
func TestRun_CodexApplyPatch_ExecutesAgainstResolvedFile(t *testing.T) {
	tmp := t.TempDir()
	tmpResolved, err := filepath.EvalSymlinks(tmp)
	assert.NilError(t, err)
	touch(t, filepath.Join(tmpResolved, "go.mod"))
	rel := filepath.Join("pkg", "foo.go")
	touch(t, filepath.Join(tmpResolved, rel))

	cfg := Config{Filetypes: goRustTables()}
	opt := Option{Template: "touch {{ quote .File }}.formatted", Filter: []string{"go"}}
	cmd := "*** Begin Patch\n*** Add File: " + rel + "\n+package foo\n*** End Patch\n"
	r := codexApplyPatchEnvelope(t, tmpResolved, cmd, "ok")

	assertPassThrough(t, Run(context.Background(), r, cfg, opt))
	_, statErr := os.Stat(filepath.Join(tmpResolved, rel) + ".formatted")
	assert.NilError(t, statErr)
}

// A Claude Write PostToolUse resolves .File the same way Edit does, now
// that the Write tool name dispatches to FileWriteInput. Before the wiring
// it deserialized to ToolInputUnknown and .File came back empty.
func TestRender_WriteToolResolvesFile(t *testing.T) {
	tmp := t.TempDir()
	touch(t, filepath.Join(tmp, "go.mod"))
	writePath := filepath.Join(tmp, "pkg", "w.go")
	touch(t, writePath)

	cfg := Config{Filetypes: goRustTables()}
	opt := Option{Template: "ft={{ .Filetype }} file={{ .File }}", Filter: []string{"go"}}
	r := marshalHookInput(t, &types.PostToolUseHookInput{
		BaseHookInput: types.BaseHookInput{SessionID: "s", TranscriptPath: "/t", Cwd: tmp},
		HookEventName: types.HookEventPostToolUse,
		ToolName:      types.ToolNameWrite,
		ToolInput: types.NewToolInputSchemas(
			&types.FileWriteInput{FilePath: writePath, Content: "package w\n"},
		),
		ToolUseID: "tu",
	})

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "ft=go file="+writePath+"\n")
}

// End-to-end through Run with the real captured Codex apply_patch payload
// (an Update File hunk; tool_response is a bare string — the exact shape
// from the failure report, kept verbatim in testdata/). Driven like
// `crabswarm hook exec --ft go ...`, it exercises the whole path on a real
// payload: parse (the string tool_response no longer aborts it),
// apply_patch file extraction, cwd resolution, template render, shellwords
// split, and actually spawning the process. Run executes argv directly (no
// shell), so we go through `sh -c` to redirect the resolved .File into a
// temp file and read it back.
func TestRun_CodexApplyPatchUpdateFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "codex_apply_patch_update.json"))
	assert.NilError(t, err)

	tmp := t.TempDir()
	out := filepath.Join(tmp, "ran.txt")

	cfg := Config{}
	opt := Option{
		Template: "sh -c " + templateutil.ShellQuote(
			"printf %s {{ .File }} > "+templateutil.ShellQuote(out),
		),
		Filter: []string{"go"},
	}

	assertPassThrough(t, Run(context.Background(), bytes.NewReader(raw), cfg, opt))

	got, err := os.ReadFile(out)
	assert.NilError(t, err)
	assert.Equal(t, string(got),
		"/home/watage/.dotfiles/snapshot_home/snapshotter/codex_hook_probe.go")
}

// Compile-time check that .Input is the SDK union variant interface.
var _ types.HookInput_Value = (Data{}).Input

// --- Option.OutputTemplate ---

// handlerErrorOf unwraps err as the hook result and requires it to carry an
// output; the output-template tests all assert on that output's shape.
func handlerErrorOf(t *testing.T, err error) *types.SyncHookJSONOutput {
	t.Helper()
	var he *handler.HandlerError
	if !errors.As(err, &he) {
		t.Fatalf("expected *handler.HandlerError, got %T: %v", err, err)
	}
	if he.Output == nil {
		t.Fatal("expected a non-nil Output from the output template")
	}
	return he.Output
}

func TestRun_OutputTemplate_SuccessEmitsHookSpecificOutput(t *testing.T) {
	// The success path the built-in behavior can't express at all: a zero
	// exit turns into additionalContext instead of a silent allow.
	cfg := Config{}
	opt := Option{
		Template: "sh -c " + templateutil.ShellQuote("echo captured"),
		OutputTemplate: `{{ if .Success }}` +
			`{{ context (printf "exit=%d out=%s" .ExitCode .Stdout) }}{{ end }}`,
	}
	r := bashEnvelope(t, "ls")

	out := handlerErrorOf(t, Run(context.Background(), r, cfg, opt))
	v := hsoAs[types.HookSpecificOutputPreToolUse](t, out)
	assert.Equal(t, deref(t, v.AdditionalContext), "exit=0 out=captured")
	assert.Assert(t, out.Decision == nil, "a successful command must not block")
}

func TestRun_OutputTemplate_FailureBlocksWithSplitCaptures(t *testing.T) {
	cfg := Config{}
	opt := Option{
		Template: "sh -c " + templateutil.ShellQuote("echo on-out; echo on-err >&2; exit 3"),
		OutputTemplate: `{{ if not .Success }}` +
			`{{ blockDecision (printf "exit=%d stderr=%s" .ExitCode .Stderr) }}` +
			`{{ systemMessage .Output }}{{ end }}`,
	}
	r := bashEnvelope(t, "ls")

	out := handlerErrorOf(t, Run(context.Background(), r, cfg, opt))
	assert.Equal(t, deref(t, out.Decision), types.HookDecisionBlock)
	assert.Equal(t, deref(t, out.Reason), "exit=3 stderr=on-err")

	// .Output is both streams; interleaving is best-effort so only
	// membership is asserted.
	combined := deref(t, out.SystemMessage)
	assert.Assert(t, strings.Contains(combined, "on-out"), "combined: %q", combined)
	assert.Assert(t, strings.Contains(combined, "on-err"), "combined: %q", combined)
}

func TestRun_OutputTemplate_StrayTextIsError(t *testing.T) {
	// Text outside the output functions fails the invocation rather than
	// silently degrading to allow-everything.
	cfg := Config{}
	opt := Option{Template: "true", OutputTemplate: "oops"}
	r := bashEnvelope(t, "ls")

	err := Run(context.Background(), r, cfg, opt)
	assert.Assert(t, err != nil)
	var he *handler.HandlerError
	assert.Assert(t, !errors.As(err, &he), "stray text should be a plain error; got %T", err)
	assert.Assert(
		t,
		strings.Contains(err.Error(), "may only produce output through the output functions"),
		"err: %v", err,
	)
}

func TestRun_OutputTemplate_SkippedByFilterGate(t *testing.T) {
	// A gated invocation never runs a command, so there is no result to
	// shape: the output template — which would hard-error on its stray
	// text — must not be rendered at all.
	tmp := t.TempDir()
	marker := filepath.Join(tmp, "should-not-exist")
	editPath := filepath.Join(tmp, "x.go")
	touch(t, editPath)

	cfg := Config{Filetypes: goRustTables()}
	opt := Option{
		Template:       "touch " + marker,
		OutputTemplate: "stray text that would fail the render",
		Filter:         []string{"rust"}, // detected will be "go"
	}
	r := editEnvelope(t, editPath)

	assertPassThrough(t, Run(context.Background(), r, cfg, opt))
	_, statErr := os.Stat(marker)
	assert.Assert(t, statErr != nil, "marker should NOT exist when filter blocks")
}

func TestRun_OutputTemplate_SkippedWhenRenderedEmpty(t *testing.T) {
	cfg := Config{}
	opt := Option{
		Template:       "{{ if false }}/usr/bin/false{{ end }}",
		OutputTemplate: "stray text that would fail the render",
	}
	r := bashEnvelope(t, "ls")

	assertPassThrough(t, Run(context.Background(), r, cfg, opt))
}

func TestRun_OutputTemplate_NonExitErrorPropagatesAsPlainError(t *testing.T) {
	// The command never ran, so — as in the built-in path — this is an
	// infrastructure failure, not something the output template gets to
	// reinterpret as a hook result.
	cfg := Config{}
	opt := Option{
		Template:       "/no/such/binary-does-not-exist",
		OutputTemplate: `{{ context "should never be reached" }}`,
	}
	r := bashEnvelope(t, "ls")

	err := Run(context.Background(), r, cfg, opt)
	assert.Assert(t, err != nil)
	var he *handler.HandlerError
	assert.Assert(t, !errors.As(err, &he), "expected a plain error; got %T", err)
	assert.Assert(t, strings.Contains(err.Error(), "running "), "err: %v", err)
}

func TestRender_OutputTemplate_SyntheticSuccess(t *testing.T) {
	// Dry-run has nothing to execute, so the output template previews
	// against a synthetic success: exit 0, empty captures, .Command set to
	// the command line that would have run.
	cfg := Config{}
	opt := Option{
		Template: "echo hi",
		OutputTemplate: `{{ if .Success }}` +
			`{{ context (printf "cmd=%s exit=%d out=%q" .Command .ExitCode .Output) }}{{ end }}`,
	}
	r := bashEnvelope(t, "ls")

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))

	want, err := json.Marshal(&types.SyncHookJSONOutput{
		HookSpecificOutput: types.NewHookSpecificOutput(&types.HookSpecificOutputPreToolUse{
			HookEventName:     types.HookEventPreToolUse,
			AdditionalContext: new(`cmd=echo hi exit=0 out=""`),
		}),
	})
	assert.NilError(t, err)
	assert.Equal(t, buf.String(), "echo hi\n"+string(want)+"\n")
}

// builtinBlockReasonTemplate restates the built-in block-on-failure behavior
// as an output template — the plan's "the built-in is expressible" claim in
// executable form. .Error supplies the run error the built-in prints on its
// `exit:` line, and the output: section is conditional exactly as blockReason's
// is.
const builtinBlockReasonTemplate = `{{- if not .Success -}}
{{- $r := printf "command failed: %s\nexit: %s\n" .Command .Error -}}
{{- if .Output }}{{ $r = printf "%soutput:\n%s\n" $r .Output }}{{ end -}}
{{- blockDecision $r -}}
{{- end -}}`

func TestRun_OutputTemplate_RestatesBuiltinBlockReason(t *testing.T) {
	// Single-stream output on purpose: the built-in path captures with
	// CombinedOutput while the template path multiplexes two pipes into one
	// buffer, so with both streams busy the interleaving — not the formatting
	// — would decide the bytes.
	for _, tc := range []struct {
		name              string
		command           string
		wantOutputSection bool
	}{
		{
			name:              "with output",
			command:           "sh -c " + templateutil.ShellQuote("echo boom; exit 1"),
			wantOutputSection: true,
		},
		{
			name:              "without output",
			command:           "false",
			wantOutputSection: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{}

			builtin := handlerErrorOf(t, Run(
				context.Background(), bashEnvelope(t, "ls"), cfg,
				Option{Template: tc.command},
			))
			templated := handlerErrorOf(t, Run(
				context.Background(), bashEnvelope(t, "ls"), cfg,
				Option{Template: tc.command, OutputTemplate: builtinBlockReasonTemplate},
			))

			reason := deref(t, builtin.Reason)
			assert.Equal(t, deref(t, templated.Reason), reason)
			assert.Equal(t, deref(t, templated.Decision), deref(t, builtin.Decision))

			// Guard the fixtures themselves: the comparison only proves
			// something if the two cases really differ in the section the
			// template has to reproduce conditionally.
			assert.Assert(t, strings.Contains(reason, "exit: exit status 1"), "reason: %q", reason)
			assert.Equal(t, strings.Contains(reason, "output:"), tc.wantOutputSection,
				"reason: %q", reason)
		})
	}
}

func TestRun_OutputTemplate_CancelledContextReachesTemplate(t *testing.T) {
	// Cancelling a child that already started is not an infrastructure
	// failure: os/exec kills it and the signal comes back as an ExitError, so
	// the template runs on a failed result instead of the call erroring out.
	//
	// The deadline has to outlast everything Run does before fork/exec —
	// reading the envelope, folding the filetype tables, rendering — because
	// Cmd.Start returns a plain ctx.Err() when the context is already done,
	// which is the other branch entirely. Generous on purpose; `sleep 5`
	// outlives it either way.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	cfg := Config{}
	opt := Option{
		Template:       "sleep 5",
		OutputTemplate: `{{ blockDecision (printf "%v|%d|%s" .Success .ExitCode .Error) }}`,
	}
	r := bashEnvelope(t, "ls")

	out := handlerErrorOf(t, Run(ctx, r, cfg, opt))
	assert.Equal(t, deref(t, out.Reason), "false|-1|signal: killed")
}

func TestRun_OutputTemplate_ConcurrentStreamsStayIntact(t *testing.T) {
	// os/exec copies stdout and stderr into the shared combined buffer from
	// two goroutines. Volume is what makes them actually overlap, so this
	// pushes thousands of lines down each stream — with the mutex removed it
	// trips `go test -race` and drops bytes.
	const lines = 2000
	script := "i=0; while [ $i -lt " + strconv.Itoa(lines) + " ]; do " +
		"echo out-$i; echo err-$i >&2; i=$((i+1)); done"

	cfg := Config{}
	opt := Option{
		Template:       "sh -c " + templateutil.ShellQuote(script),
		OutputTemplate: `{{ blockDecision .Stdout }}{{ systemMessage .Stderr }}{{ stop .Output }}`,
	}
	r := bashEnvelope(t, "ls")

	out := handlerErrorOf(t, Run(context.Background(), r, cfg, opt))
	stdout := deref(t, out.Reason)
	stderr := deref(t, out.SystemMessage)
	combined := deref(t, out.StopReason)

	assert.Equal(t, len(strings.Split(stdout, "\n")), lines)
	assert.Equal(t, len(strings.Split(stderr, "\n")), lines)
	assert.Equal(t, strings.Count(stdout, "out-"), lines)
	assert.Equal(t, strings.Count(stderr, "err-"), lines)
	// Counting rather than comparing byte length: every line has to survive
	// the shared buffer, but the two streams' interleaving in it does not.
	assert.Equal(t, strings.Count(combined, "out-"), lines)
	assert.Equal(t, strings.Count(combined, "err-"), lines)
}

func TestRender_SkipsWhitespaceOnlyRenderedCommand(t *testing.T) {
	// Run splits the rendered line and skips it when it yields no argv.
	// Render mirrors that gate, so a dry run shows what Run would do —
	// nothing — instead of a blank command line with a preview under it. The
	// stray-text output template proves the preview never ran: rendering it
	// would be a hard error.
	cfg := Config{}
	opt := Option{
		Template:       "   ",
		OutputTemplate: "stray text that would fail the render",
	}
	r := bashEnvelope(t, "ls")

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "")
}

func TestRender_UnparseableRenderedCommandErrors(t *testing.T) {
	// Splitting in Render also means a dry run now reports the unbalanced
	// quote that only Run used to catch.
	cfg := Config{}
	opt := Option{Template: `echo "unbalanced`}
	r := bashEnvelope(t, "ls")

	var buf bytes.Buffer
	err := Render(context.Background(), r, &buf, cfg, opt)
	assert.Assert(t, err != nil)
	var he *handler.HandlerError
	assert.Assert(t, !errors.As(err, &he), "expected a plain error; got %T", err)
	assert.Assert(t, strings.Contains(err.Error(), "parsing rendered command"), "err: %v", err)
	assert.Equal(t, buf.String(), "")
}

func TestRender_OutputTemplate_PlainAllowPreviewsAsNull(t *testing.T) {
	// A template that records nothing means plain allow; the preview spells
	// that as JSON null.
	cfg := Config{}
	opt := Option{
		Template:       "echo hi",
		OutputTemplate: `{{ if not .Success }}{{ blockDecision "unreachable" }}{{ end }}`,
	}
	r := bashEnvelope(t, "ls")

	var buf bytes.Buffer
	assertPassThrough(t, Render(context.Background(), r, &buf, cfg, opt))
	assert.Equal(t, buf.String(), "echo hi\nnull\n")
}
