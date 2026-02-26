package planreview

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestResolveLastPlanPath_DirectFormat(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	// Create a plan file.
	planFile := filepath.Join(plansDir, "my-plan.md")
	assert.NilError(t, os.WriteFile(planFile, []byte("# Plan"), 0o644))

	// Create transcript with a Write event using direct format.
	transcript := `{"type":"tool_use","tool_name":"Read","tool_input":{"file_path":"/some/file.go"}}
{"type":"tool_use","tool_name":"Write","tool_input":{"file_path":"` + planFile + `","content":"# Plan"}}
{"type":"tool_result","tool_name":"Write"}
`
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	assert.NilError(t, os.WriteFile(transcriptPath, []byte(transcript), 0o644))

	got, err := ResolveLastPlanPath(transcriptPath, plansDir)
	assert.NilError(t, err)
	assert.Equal(t, got, planFile)
}

func TestResolveLastPlanPath_NestedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	planFile := filepath.Join(plansDir, "nested-plan.md")
	assert.NilError(t, os.WriteFile(planFile, []byte("# Plan"), 0o644))

	// Create transcript with nested message.content format.
	transcript := `{"message":{"content":[{"type":"tool_use","name":"Write","input":{"file_path":"` + planFile + `","content":"# Plan"}}]}}
`
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	assert.NilError(t, os.WriteFile(transcriptPath, []byte(transcript), 0o644))

	got, err := ResolveLastPlanPath(transcriptPath, plansDir)
	assert.NilError(t, err)
	assert.Equal(t, got, planFile)
}

func TestResolveLastPlanPath_ReturnsLastMatch(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	plan1 := filepath.Join(plansDir, "plan-v1.md")
	plan2 := filepath.Join(plansDir, "plan-v2.md")
	assert.NilError(t, os.WriteFile(plan1, []byte("v1"), 0o644))
	assert.NilError(t, os.WriteFile(plan2, []byte("v2"), 0o644))

	transcript := `{"type":"tool_use","tool_name":"Write","tool_input":{"file_path":"` + plan1 + `","content":"v1"}}
{"type":"tool_use","tool_name":"Write","tool_input":{"file_path":"` + plan2 + `","content":"v2"}}
`
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	assert.NilError(t, os.WriteFile(transcriptPath, []byte(transcript), 0o644))

	got, err := ResolveLastPlanPath(transcriptPath, plansDir)
	assert.NilError(t, err)
	assert.Equal(t, got, plan2)
}

func TestResolveLastPlanPath_IgnoresNonPlanWrites(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	otherDir := filepath.Join(tmpDir, "other")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))
	assert.NilError(t, os.MkdirAll(otherDir, 0o755))

	otherFile := filepath.Join(otherDir, "code.go")
	assert.NilError(t, os.WriteFile(otherFile, []byte("package main"), 0o644))

	transcript := `{"type":"tool_use","tool_name":"Write","tool_input":{"file_path":"` + otherFile + `","content":"package main"}}
`
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	assert.NilError(t, os.WriteFile(transcriptPath, []byte(transcript), 0o644))

	got, err := ResolveLastPlanPath(transcriptPath, plansDir)
	assert.NilError(t, err)
	assert.Equal(t, got, "")
}

func TestResolveLastPlanPath_IgnoresNonMdFiles(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	txtFile := filepath.Join(plansDir, "notes.txt")
	assert.NilError(t, os.WriteFile(txtFile, []byte("notes"), 0o644))

	transcript := `{"type":"tool_use","tool_name":"Write","tool_input":{"file_path":"` + txtFile + `","content":"notes"}}
`
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	assert.NilError(t, os.WriteFile(transcriptPath, []byte(transcript), 0o644))

	got, err := ResolveLastPlanPath(transcriptPath, plansDir)
	assert.NilError(t, err)
	assert.Equal(t, got, "")
}

func TestResolveLastPlanPath_EmptyTranscript(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	assert.NilError(t, os.WriteFile(transcriptPath, []byte(""), 0o644))

	got, err := ResolveLastPlanPath(transcriptPath, plansDir)
	assert.NilError(t, err)
	assert.Equal(t, got, "")
}

func TestResolveLastPlanPath_MalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	planFile := filepath.Join(plansDir, "good.md")
	assert.NilError(t, os.WriteFile(planFile, []byte("plan"), 0o644))

	// Mix of malformed and valid lines. The function should skip malformed and find valid.
	transcript := `not valid json Write
{"type":"tool_use","tool_name":"Write","tool_input":{"file_path":"` + planFile + `","content":"plan"}}
also not valid Write json
`
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	assert.NilError(t, os.WriteFile(transcriptPath, []byte(transcript), 0o644))

	got, err := ResolveLastPlanPath(transcriptPath, plansDir)
	assert.NilError(t, err)
	assert.Equal(t, got, planFile)
}

func TestResolveLastPlanPath_NonexistentTranscript(t *testing.T) {
	_, err := ResolveLastPlanPath("/nonexistent/transcript.jsonl", "/tmp")
	assert.Assert(t, err != nil)
}

func TestExtractWritePath(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "direct format",
			line: `{"tool_name":"Write","tool_input":{"file_path":"/foo/bar.md","content":"x"}}`,
			want: "/foo/bar.md",
		},
		{
			name: "nested format",
			line: `{"message":{"content":[{"type":"tool_use","name":"Write","input":{"file_path":"/baz/qux.md"}}]}}`,
			want: "/baz/qux.md",
		},
		{
			name: "not a Write event",
			line: `{"tool_name":"Read","tool_input":{"file_path":"/foo/bar.md"}}`,
			want: "",
		},
		{
			name: "no Write in line",
			line: `{"tool_name":"Bash","tool_input":{"command":"ls"}}`,
			want: "",
		},
		{
			name: "malformed json",
			line: `not json but contains Write`,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWritePath(tt.line)
			assert.Equal(t, got, tt.want)
		})
	}
}
