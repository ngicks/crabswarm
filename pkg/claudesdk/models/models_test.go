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

const postToolUseReadJSON = `{
  "session_id": "post-1",
  "transcript_path": "/tmp/transcript.jsonl",
  "cwd": "/home/user",
  "hook_event_name": "PostToolUse",
  "tool_name": "Read",
  "tool_input": {
    "file_path": "/tmp/a.txt"
  },
  "tool_response": {
    "text_file": {
      "content": "hello",
      "total_lines": 1,
      "lines_returned": 1
    }
  },
  "tool_use_id": "toolu_post_1"
}`

const permissionRequestJSON = `{
  "session_id": "perm-1",
  "transcript_path": "/tmp/transcript.jsonl",
  "cwd": "/home/user",
  "hook_event_name": "PermissionRequest",
  "tool_name": "Write",
  "tool_input": {
    "file_path": "/tmp/a.txt",
    "content": "hello"
  },
  "permission_suggestions": [
    {
      "add_rules": {
        "rules": [{"tool_name":"Write","rule_content":"/tmp/**"}],
        "behavior": "allow",
        "destination": "session"
      }
    }
  ]
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
			assert.Assert(t, m.ToolInput != nil, "tool_input should not be nil")
		})
	}
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

	// Verify tool_input is preserved (compare semantically, ignoring whitespace).
	raw1, err := json.Marshal(m.ToolInput)
	assert.NilError(t, err)
	raw2, err := json.Marshal(m2.ToolInput)
	assert.NilError(t, err)
	var ti1, ti2 map[string]any
	err = json.Unmarshal(raw1, &ti1)
	assert.NilError(t, err)
	err = json.Unmarshal(raw2, &ti2)
	assert.NilError(t, err)
	assert.DeepEqual(t, ti1, ti2)
}

func TestPreToolUseHookInput_ModelToProtoRoundTrip(t *testing.T) {
	// JSON -> model
	var m models.PreToolUseHookInput
	err := json.Unmarshal([]byte(preToolUseReadJSON), &m)
	assert.NilError(t, err)

	// model -> proto
	p, err := m.ToProto()
	assert.NilError(t, err)
	assert.Equal(t, p.GetSessionId(), "12345")
	assert.Equal(t, p.GetToolName(), "Read")
	assert.Assert(t, p.GetToolInput() != nil)
	assert.Assert(t, p.GetToolInput().GetFileRead() != nil)
	assert.Equal(t, p.GetToolInput().GetFileRead().GetFilePath(), "/yay/buf.gen.yaml")

	// proto -> model
	var m2 models.PreToolUseHookInput
	err = m2.FromProto(p)
	assert.NilError(t, err)
	assert.Equal(t, m2.SessionID, m.SessionID)
	assert.Equal(t, m2.ToolName, m.ToolName)
	assert.Equal(t, m2.ToolUseID, m.ToolUseID)

	// Verify tool_input round-trips.
	ri, ok := m2.ToolInput.(models.FileReadInput)
	assert.Assert(t, ok, "expected FileReadInput, got %T", m2.ToolInput)
	assert.Equal(t, ri.FilePath, "/yay/buf.gen.yaml")
}

func TestPreToolUseHookInput_MarshalProducesFlat(t *testing.T) {
	var m models.PreToolUseHookInput
	m.SessionID = "sess1"
	m.TranscriptPath = "/tmp/t.jsonl"
	m.Cwd = "/home"
	m.ToolName = "Read"
	m.ToolInput = models.FileReadInput{FilePath: "/foo/bar.go"}
	m.HookEventName = "PreToolUse"
	m.ToolUseID = "tool1"

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

func TestPostToolUseHookInput_ModelToProtoRoundTrip(t *testing.T) {
	var m models.PostToolUseHookInput
	err := json.Unmarshal([]byte(postToolUseReadJSON), &m)
	assert.NilError(t, err)

	p, err := m.ToProto()
	assert.NilError(t, err)
	assert.Equal(t, p.GetToolName(), "Read")
	assert.Assert(t, p.GetToolInput().GetFileRead() != nil)
	assert.Assert(t, p.GetToolResponse().GetRead() != nil)
	assert.Assert(t, p.GetToolResponse().GetRead().GetTextFile() != nil)
	assert.Equal(t, p.GetToolResponse().GetRead().GetTextFile().GetContent(), "hello")

	var m2 models.PostToolUseHookInput
	err = m2.FromProto(p)
	assert.NilError(t, err)
	assert.Equal(t, m2.ToolName, "Read")
	assert.Assert(t, m2.ToolInput != nil)
	assert.Assert(t, m2.ToolResponse != nil)
}

func TestPermissionRequestHookInput_ModelToProtoRoundTrip(t *testing.T) {
	var m models.PermissionRequestHookInput
	err := json.Unmarshal([]byte(permissionRequestJSON), &m)
	assert.NilError(t, err)
	assert.Equal(t, len(m.PermissionSuggestions), 1)

	p, err := m.ToProto()
	assert.NilError(t, err)
	assert.Equal(t, p.GetToolName(), "Write")
	assert.Equal(t, len(p.GetPermissionSuggestions()), 1)
	assert.Assert(t, p.GetPermissionSuggestions()[0].GetAddRules() != nil)
	assert.Equal(t, p.GetPermissionSuggestions()[0].GetAddRules().GetBehavior(), "allow")

	var m2 models.PermissionRequestHookInput
	err = m2.FromProto(p)
	assert.NilError(t, err)
	assert.Equal(t, m2.ToolName, "Write")
	assert.Equal(t, len(m2.PermissionSuggestions), 1)
	assert.Assert(t, m2.PermissionSuggestions[0].AddRules != nil)
	assert.Equal(t, m2.PermissionSuggestions[0].AddRules.Behavior, "allow")
}
