package statusline

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// render renders tmpl against the JSON in raw and returns the output.
func render(t *testing.T, tmpl, raw string) string {
	t.Helper()
	var buf bytes.Buffer
	assert.NilError(t, Render(strings.NewReader(raw), &buf, tmpl))
	return buf.String()
}

// renderInput marshals in to JSON and renders tmpl against it.
func renderInput(t *testing.T, tmpl string, in Input) string {
	t.Helper()
	raw, err := json.Marshal(in)
	assert.NilError(t, err)
	return render(t, tmpl, string(raw))
}

func TestRender_BasicFields(t *testing.T) {
	in := Input{
		Cwd:       "/work/project",
		SessionId: "sess-1",
		Version:   "2.1.90",
		Model:     Model{Id: "claude-opus-4-7", DisplayName: "Opus"},
	}
	got := renderInput(
		t,
		"{{ .Model.DisplayName }} v{{ .Version }} sid={{ .SessionId }}",
		in,
	)
	assert.Equal(t, got, "Opus v2.1.90 sid=sess-1")
}

func TestRender_NoTrailingNewline(t *testing.T) {
	// The template controls its own output; Render adds nothing.
	got := renderInput(t, "{{ .Version }}", Input{Version: "1.0.0"})
	assert.Equal(t, got, "1.0.0")
}

func TestRender_WorkspaceAndRepo(t *testing.T) {
	in := Input{
		Workspace: Workspace{
			CurrentDir:  "/work/project",
			GitWorktree: "feature-xyz",
			Repo:        &Repo{Host: "github.com", Owner: "anthropics", Name: "claude-code"},
		},
	}
	got := renderInput(
		t,
		"{{ with .Workspace.Repo }}{{ .Owner }}/{{ .Name }}{{ end }}"+
			" wt={{ .Workspace.GitWorktree }}",
		in,
	)
	assert.Equal(t, got, "anthropics/claude-code wt=feature-xyz")
}

func TestRender_OptionalObjectsPresent(t *testing.T) {
	in := Input{
		Agent: &Agent{Name: "security-reviewer"},
		Vim:   &Vim{Mode: "NORMAL"},
		Pr:    &Pr{Number: 1234, ReviewState: "pending"},
	}
	got := renderInput(
		t,
		"{{ with .Agent }}[{{ .Name }}]{{ end }}"+
			"{{ with .Vim }} {{ .Mode }}{{ end }}"+
			"{{ with .Pr }} PR#{{ .Number }} {{ .ReviewState }}{{ end }}",
		in,
	)
	assert.Equal(t, got, "[security-reviewer] NORMAL PR#1234 pending")
}

func TestRender_OptionalObjectsAbsent(t *testing.T) {
	// Nil pointers make `{{ with }}` / `{{ if }}` render nothing — no
	// nil-pointer evaluation error.
	got := renderInput(
		t,
		"a{{ with .Agent }}{{ .Name }}{{ end }}"+
			"{{ if .Pr }}pr{{ end }}"+
			"{{ if .Effort }}{{ .Effort.Level }}{{ end }}z",
		Input{},
	)
	assert.Equal(t, got, "az")
}

func TestRender_IntegerPercentageRendersClean(t *testing.T) {
	// context_window.used_percentage is documented as an integer (8); the
	// float64 field renders it without a decimal point.
	got := render(
		t,
		"{{ .ContextWindow.UsedPercentage }}% used",
		`{"context_window":{"used_percentage":8}}`,
	)
	assert.Equal(t, got, "8% used")
}

func TestRender_FractionalPercentageParses(t *testing.T) {
	// rate_limits percentages are fractional; they must parse and render.
	got := render(
		t,
		"{{ .RateLimits.FiveHour.UsedPercentage }}",
		`{"rate_limits":{"five_hour":{"used_percentage":23.5,"resets_at":1738425600}}}`,
	)
	assert.Equal(t, got, "23.5")
}

func TestRender_AddedDirsRange(t *testing.T) {
	in := Input{
		Workspace: Workspace{AddedDirs: []string{"/a", "/b"}},
	}
	got := renderInput(t, "{{ range .Workspace.AddedDirs }}{{ . }},{{ end }}", in)
	assert.Equal(t, got, "/a,/b,")
}

func TestRender_Funcs(t *testing.T) {
	in := Input{Workspace: Workspace{CurrentDir: "/work/project/sub"}}
	got := renderInput(
		t,
		"{{ basename .Workspace.CurrentDir }}|{{ dirname .Workspace.CurrentDir }}",
		in,
	)
	assert.Equal(t, got, "sub|/work/project")
}

func TestRender_EnvFunc(t *testing.T) {
	t.Setenv("CRABSWARM_STATUSLINE_TEST", "xyz")
	got := renderInput(t, `{{ env "CRABSWARM_STATUSLINE_TEST" }}`, Input{})
	assert.Equal(t, got, "xyz")
}

func TestRender_NarrowTerminalFallback(t *testing.T) {
	// The documented narrow-terminal shape: trim the cwd by runes when
	// $COLUMNS is known, otherwise keep its last three components.
	const tmpl = `{{ padRuneRight 6 .Model.DisplayName }}|` +
		`{{ if columns }}{{ truncRuneLeft 10 .Workspace.CurrentDir }}` +
		`{{ else }}{{ splitPath .Workspace.CurrentDir | lastN 3 | join "/" }}{{ end }}`
	in := Input{
		Model:     Model{DisplayName: "Opus"},
		Workspace: Workspace{CurrentDir: "/home/me/src/github.com/org/repo"},
	}

	t.Setenv("COLUMNS", "60")
	assert.Equal(t, renderInput(t, tmpl, in), "Opus  |m/org/repo")

	t.Setenv("COLUMNS", "")
	assert.Equal(t, renderInput(t, tmpl, in), "Opus  |github.com/org/repo")
}

func TestRender_UnknownFieldsIgnored(t *testing.T) {
	// Forward compatibility: a future Claude Code field must not break parsing.
	got := render(
		t,
		"{{ .Version }}",
		`{"version":"9.9.9","some_future_field":{"nested":true}}`,
	)
	assert.Equal(t, got, "9.9.9")
}

func TestRender_InvalidJSONErrors(t *testing.T) {
	var buf bytes.Buffer
	err := Render(strings.NewReader("not json"), &buf, "{{ .Version }}")
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "parsing statusline input"))
	assert.Equal(t, buf.String(), "")
}

func TestRender_InvalidTemplateErrors(t *testing.T) {
	var buf bytes.Buffer
	err := Render(strings.NewReader(`{}`), &buf, "{{ .Unclosed")
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "parsing template"))
	assert.Equal(t, buf.String(), "")
}

func TestRender_ExecutionErrorLeavesNoOutput(t *testing.T) {
	// Referencing a field that doesn't exist on Input is an execution error;
	// because Render buffers, nothing is written to w.
	var buf bytes.Buffer
	err := Render(strings.NewReader(`{}`), &buf, "ok{{ .NoSuchField }}")
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "rendering template"))
	assert.Equal(t, buf.String(), "")
}

// TestRender_FullFixture renders a representative template against the
// documented payload to exercise every block end-to-end.
func TestRender_FullFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "statusline_full.json"))
	assert.NilError(t, err)

	tmpl := "{{ .Model.DisplayName }}" +
		" [{{ .Workspace.Repo.Owner }}/{{ .Workspace.Repo.Name }}]" +
		" {{ basename .Workspace.CurrentDir }}" +
		"{{ with .Workspace.GitWorktree }} ({{ . }}){{ end }}" +
		" {{ .ContextWindow.UsedPercentage }}%" +
		"{{ with .Effort }} effort={{ .Level }}{{ end }}" +
		"{{ if .Thinking.Enabled }} thinking{{ end }}" +
		"{{ with .Agent }} agent={{ .Name }}{{ end }}" +
		"{{ with .Pr }} PR#{{ .Number }}{{ end }}" +
		"{{ with .Vim }} {{ .Mode }}{{ end }}" +
		" 5h={{ .RateLimits.FiveHour.UsedPercentage }}"

	var buf bytes.Buffer
	assert.NilError(t, Render(bytes.NewReader(raw), &buf, tmpl))
	assert.Equal(
		t,
		buf.String(),
		"Opus [anthropics/claude-code] directory (feature-xyz) 8%"+
			" effort=high thinking agent=security-reviewer PR#1234 NORMAL 5h=23.5",
	)
}

// TestRender_FixtureRoundTrips confirms the documented payload deserializes
// into Input with the expected typed values.
func TestRender_FixtureRoundTrips(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "statusline_full.json"))
	assert.NilError(t, err)

	var in Input
	assert.NilError(t, json.Unmarshal(raw, &in))

	assert.Equal(t, in.Model.Id, "claude-opus-4-7")
	assert.Equal(t, in.Cost.TotalCostUsd, 0.01234)
	assert.Equal(t, in.Cost.TotalDurationMs, int64(45000))
	assert.Equal(t, in.ContextWindow.ContextWindowSize, 200000)
	assert.Equal(t, in.ContextWindow.CurrentUsage.CacheReadInputTokens, 2000)
	assert.Equal(t, in.Exceeds200kTokens, false)
	assert.Assert(t, in.Effort != nil)
	assert.Equal(t, in.Effort.Level, "high")
	assert.Equal(t, in.Thinking.Enabled, true)
	assert.Assert(t, in.RateLimits.SevenDay != nil)
	assert.Equal(t, in.RateLimits.SevenDay.ResetsAt, int64(1738857600))
	assert.Assert(t, in.Pr != nil)
	assert.Equal(t, in.Pr.Url, "https://github.com/anthropics/claude-code/pull/1234")
	assert.Assert(t, in.Worktree != nil)
	assert.Equal(t, in.Worktree.OriginalBranch, "main")
}
