package models_test

import (
	"encoding/json"
	"testing"

	"github.com/ngicks/crabswarm/pkg/claudesdk/models"
	"gotest.tools/v3/assert"
)

const preToolUseReadJSON = `{
  "session_id": "12345",
  "transcript_path": "/root/.config/claude/projects/-yay/12345.jsonl",
  "cwd": "/yay",
  "permission_mode": "default",
  "hook_event_name": "PreToolUse",
  "tool_name": "Read",
  "tool_input": {
    "file_path": "/yay/buf.gen.yaml"
  },
  "tool_use_id": "tooluse_12345"
}`

const preToolUseWriteJSON = `{
  "session_id": "d0404070",
  "transcript_path": "/root/.config/claude/projects/-home/d0404070.jsonl",
  "cwd": "/home/watage/gitrepo/github.com/ngicks/crabswarm",
  "permission_mode": "plan",
  "hook_event_name": "PreToolUse",
  "tool_name": "Write",
  "tool_input": {
    "file_path": "/root/.config/claude/plans/test.md",
    "content": "# Test Plan"
  },
  "tool_use_id": "toolu_01Ef96B1n6ZFjmrFhn3fTFSS"
}`

const preToolUseBashJSON = `{
  "session_id": "abc",
  "transcript_path": "/tmp/transcript.jsonl",
  "cwd": "/home/user",
  "hook_event_name": "PreToolUse",
  "tool_name": "Bash",
  "tool_input": {
    "command": "ls -la",
    "timeout": 30000,
    "description": "List files"
  },
  "tool_use_id": "toolu_bash123"
}`

func TestPreToolUseHookInput_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantTool string
	}{
		{"Read", preToolUseReadJSON, "Read"},
		{"Write", preToolUseWriteJSON, "Write"},
		{"Bash", preToolUseBashJSON, "Bash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m models.PreToolUseHookInput
			err := json.Unmarshal([]byte(tt.input), &m)
			assert.NilError(t, err)
			assert.Equal(t, m.ToolName, tt.wantTool)
			assert.Assert(t, len(m.ToolInput) > 0, "tool_input should not be empty")
		})
	}
}

func TestParseToolInput_Read(t *testing.T) {
	var m models.PreToolUseHookInput
	err := json.Unmarshal([]byte(preToolUseReadJSON), &m)
	assert.NilError(t, err)

	parsed, err := models.ParseToolInput(m.ToolName, m.ToolInput)
	assert.NilError(t, err)

	ri, ok := parsed.(*models.FileReadInput)
	assert.Assert(t, ok, "expected *FileReadInput, got %T", parsed)
	assert.Equal(t, ri.FilePath, "/yay/buf.gen.yaml")
	assert.Assert(t, ri.Offset == nil)
	assert.Assert(t, ri.Limit == nil)
}

func TestParseToolInput_Write(t *testing.T) {
	var m models.PreToolUseHookInput
	err := json.Unmarshal([]byte(preToolUseWriteJSON), &m)
	assert.NilError(t, err)

	parsed, err := models.ParseToolInput(m.ToolName, m.ToolInput)
	assert.NilError(t, err)

	wi, ok := parsed.(*models.FileWriteInput)
	assert.Assert(t, ok, "expected *FileWriteInput, got %T", parsed)
	assert.Equal(t, wi.FilePath, "/root/.config/claude/plans/test.md")
	assert.Equal(t, wi.Content, "# Test Plan")
}

func TestParseToolInput_Bash(t *testing.T) {
	var m models.PreToolUseHookInput
	err := json.Unmarshal([]byte(preToolUseBashJSON), &m)
	assert.NilError(t, err)

	parsed, err := models.ParseToolInput(m.ToolName, m.ToolInput)
	assert.NilError(t, err)

	bi, ok := parsed.(*models.BashInput)
	assert.Assert(t, ok, "expected *BashInput, got %T", parsed)
	assert.Equal(t, bi.Command, "ls -la")
	assert.Equal(t, *bi.Timeout, int32(30000))
	assert.Equal(t, *bi.Description, "List files")
	assert.Assert(t, bi.RunInBackground == nil)
}

func TestParseToolInput_UnknownMCP(t *testing.T) {
	raw := json.RawMessage(`{"custom_param": "value", "count": 42}`)
	parsed, err := models.ParseToolInput("mcp__myserver__custom_tool", raw)
	assert.NilError(t, err)

	m, ok := parsed.(map[string]any)
	assert.Assert(t, ok, "expected map[string]any, got %T", parsed)
	assert.Equal(t, m["custom_param"], "value")
	assert.Equal(t, m["count"], float64(42))
}

func TestParseToolInput_EnterPlanMode(t *testing.T) {
	parsed, err := models.ParseToolInput("EnterPlanMode", json.RawMessage(`{}`))
	assert.NilError(t, err)
	assert.Assert(t, parsed == nil, "EnterPlanMode should return nil")
}

func TestPreToolUseHookInput_RoundTripJSON(t *testing.T) {
	// Unmarshal from JSON
	var m models.PreToolUseHookInput
	err := json.Unmarshal([]byte(preToolUseReadJSON), &m)
	assert.NilError(t, err)

	// Marshal back to JSON
	data, err := json.Marshal(&m)
	assert.NilError(t, err)

	// Unmarshal again and compare
	var m2 models.PreToolUseHookInput
	err = json.Unmarshal(data, &m2)
	assert.NilError(t, err)

	assert.Equal(t, m.SessionID, m2.SessionID)
	assert.Equal(t, m.ToolName, m2.ToolName)
	assert.Equal(t, m.ToolUseID, m2.ToolUseID)
	assert.Equal(t, m.Cwd, m2.Cwd)

	// Verify tool_input is preserved
	parsed1, err := models.ParseToolInput(m.ToolName, m.ToolInput)
	assert.NilError(t, err)
	parsed2, err := models.ParseToolInput(m2.ToolName, m2.ToolInput)
	assert.NilError(t, err)
	assert.DeepEqual(t, parsed1, parsed2)
}

func TestPreToolUseHookInput_ModelToProtoRoundTrip(t *testing.T) {
	// JSON -> model
	var m models.PreToolUseHookInput
	err := json.Unmarshal([]byte(preToolUseReadJSON), &m)
	assert.NilError(t, err)

	// model -> proto
	p, err := models.PreToolUseHookInputToProto(&m)
	assert.NilError(t, err)
	assert.Equal(t, p.GetSessionId(), "12345")
	assert.Equal(t, p.GetToolName(), "Read")
	assert.Assert(t, p.GetToolInput() != nil)
	assert.Assert(t, p.GetToolInput().GetFileRead() != nil)
	assert.Equal(t, p.GetToolInput().GetFileRead().GetFilePath(), "/yay/buf.gen.yaml")

	// proto -> model
	m2, err := models.PreToolUseHookInputFromProto(p)
	assert.NilError(t, err)
	assert.Equal(t, m2.SessionID, m.SessionID)
	assert.Equal(t, m2.ToolName, m.ToolName)
	assert.Equal(t, m2.ToolUseID, m.ToolUseID)

	// Verify tool_input round-trips
	parsed, err := models.ParseToolInput(m2.ToolName, m2.ToolInput)
	assert.NilError(t, err)
	ri, ok := parsed.(*models.FileReadInput)
	assert.Assert(t, ok)
	assert.Equal(t, ri.FilePath, "/yay/buf.gen.yaml")
}

func TestPreToolUseHookInput_MarshalProducesFlat(t *testing.T) {
	m := models.PreToolUseHookInput{
		SessionID:      "sess1",
		TranscriptPath: "/tmp/t.jsonl",
		Cwd:            "/home",
		ToolName:       "Read",
		ToolInput:      json.RawMessage(`{"file_path":"/foo/bar.go"}`),
		HookEventName:  "PreToolUse",
		ToolUseID:      "tool1",
	}

	data, err := json.Marshal(&m)
	assert.NilError(t, err)

	// Parse as generic map to verify structure
	var raw map[string]any
	err = json.Unmarshal(data, &raw)
	assert.NilError(t, err)

	// tool_input should be a flat object, not wrapped in a oneof key
	ti, ok := raw["tool_input"].(map[string]any)
	assert.Assert(t, ok, "tool_input should be a map, got %T", raw["tool_input"])
	assert.Equal(t, ti["file_path"], "/foo/bar.go")

	// Should NOT have nested keys like "fileRead"
	_, hasFileRead := ti["fileRead"]
	assert.Assert(t, !hasFileRead, "tool_input should not have nested 'fileRead' key")
}
