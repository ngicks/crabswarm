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
	var m models.PreToolUseHookInput
	err := json.Unmarshal([]byte(preToolUseReadJSON), &m)
	assert.NilError(t, err)

	data, err := json.Marshal(&m)
	assert.NilError(t, err)

	var m2 models.PreToolUseHookInput
	err = json.Unmarshal(data, &m2)
	assert.NilError(t, err)

	assert.Equal(t, m.SessionID, m2.SessionID)
	assert.Equal(t, m.ToolName, m2.ToolName)
	assert.Equal(t, m.ToolUseID, m2.ToolUseID)
	assert.Equal(t, m.Cwd, m2.Cwd)

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

	var raw map[string]any
	err = json.Unmarshal(data, &raw)
	assert.NilError(t, err)

	ti, ok := raw["tool_input"].(map[string]any)
	assert.Assert(t, ok, "tool_input should be a map, got %T", raw["tool_input"])
	assert.Equal(t, ti["file_path"], "/foo/bar.go")

	_, hasFileRead := ti["fileRead"]
	assert.Assert(t, !hasFileRead, "tool_input should not have nested 'fileRead' key")
}

// --- SyncHookJSONOutput round-trip tests ---

func TestSyncHookJSONOutput_RoundTrip_PreToolUse(t *testing.T) {
	input := `{
		"continue": true,
		"decision": "approve",
		"hookSpecificOutput": {
			"hookEventName": "PreToolUse",
			"permissionDecision": "allow",
			"permissionDecisionReason": "trusted tool"
		}
	}`
	var m models.SyncHookJSONOutput
	err := json.Unmarshal([]byte(input), &m)
	assert.NilError(t, err)
	assert.Assert(t, m.Continue != nil && *m.Continue)
	assert.Assert(t, m.HookSpecificOutput != nil)

	data, err := json.Marshal(&m)
	assert.NilError(t, err)

	var m2 models.SyncHookJSONOutput
	err = json.Unmarshal(data, &m2)
	assert.NilError(t, err)
	assert.Assert(t, m2.Continue != nil && *m2.Continue)
	assert.Assert(t, m2.HookSpecificOutput != nil)
}

func TestSyncHookJSONOutput_RoundTrip_PermissionRequest(t *testing.T) {
	input := `{
		"hookSpecificOutput": {
			"hookEventName": "PermissionRequest",
			"decision": {
				"behavior": "deny",
				"message": "not allowed"
			}
		}
	}`
	var m models.SyncHookJSONOutput
	err := json.Unmarshal([]byte(input), &m)
	assert.NilError(t, err)
	assert.Assert(t, m.HookSpecificOutput != nil)

	data, err := json.Marshal(&m)
	assert.NilError(t, err)

	var m2 models.SyncHookJSONOutput
	err = json.Unmarshal(data, &m2)
	assert.NilError(t, err)
	assert.Assert(t, m2.HookSpecificOutput != nil)
}

// --- PermissionUpdate round-trip tests ---

func TestPermissionUpdate_AddRules_RoundTrip(t *testing.T) {
	input := `{"type":"addRules","rules":[{"tool_name":"Bash","rule_content":"ls"}],"behavior":"allow","destination":"session"}`
	var v models.PermissionUpdateAddRules
	err := json.Unmarshal([]byte(input), &v)
	assert.NilError(t, err)
	assert.Equal(t, v.Type, "addRules")
	assert.Equal(t, len(v.Rules), 1)
	assert.Equal(t, v.Rules[0].ToolName, "Bash")
	assert.Equal(t, string(v.Behavior), "allow")
	assert.Equal(t, string(v.Destination), "session")

	data, err := json.Marshal(&v)
	assert.NilError(t, err)

	var v2 models.PermissionUpdateAddRules
	err = json.Unmarshal(data, &v2)
	assert.NilError(t, err)
	assert.Equal(t, v.Type, v2.Type)
	assert.Equal(t, len(v.Rules), len(v2.Rules))
}

func TestPermissionUpdate_SetMode_RoundTrip(t *testing.T) {
	input := `{"type":"setMode","mode":"plan","destination":"userSettings"}`
	var v models.PermissionUpdateSetMode
	err := json.Unmarshal([]byte(input), &v)
	assert.NilError(t, err)
	assert.Equal(t, v.Type, "setMode")
	assert.Equal(t, string(v.Mode), "plan")
	assert.Equal(t, string(v.Destination), "userSettings")

	data, err := json.Marshal(&v)
	assert.NilError(t, err)

	var v2 models.PermissionUpdateSetMode
	err = json.Unmarshal(data, &v2)
	assert.NilError(t, err)
	assert.Equal(t, v.Type, v2.Type)
	assert.Equal(t, string(v.Mode), string(v2.Mode))
}

func TestPermissionUpdate_AddDirectories_RoundTrip(t *testing.T) {
	input := `{"type":"addDirectories","directories":["/home/user","/tmp"],"destination":"localSettings"}`
	var v models.PermissionUpdateAddDirectories
	err := json.Unmarshal([]byte(input), &v)
	assert.NilError(t, err)
	assert.Equal(t, v.Type, "addDirectories")
	assert.Equal(t, len(v.Directories), 2)

	data, err := json.Marshal(&v)
	assert.NilError(t, err)

	var v2 models.PermissionUpdateAddDirectories
	err = json.Unmarshal(data, &v2)
	assert.NilError(t, err)
	assert.DeepEqual(t, v.Directories, v2.Directories)
}

// --- AgentOutput discriminated union tests ---

func TestAgentOutput_Completed(t *testing.T) {
	input := `{
		"status": "completed",
		"agentId": "agent-1",
		"content": [{"type": "text", "text": "hello"}],
		"totalToolUseCount": 5,
		"totalDurationMs": 1234,
		"totalTokens": 100,
		"usage": {
			"input_tokens": 50,
			"output_tokens": 50,
			"cache_creation_input_tokens": null,
			"cache_read_input_tokens": null,
			"server_tool_use": null,
			"service_tier": "standard",
			"cache_creation": null
		},
		"prompt": "do something"
	}`
	var v models.AgentOutputCompleted
	err := json.Unmarshal([]byte(input), &v)
	assert.NilError(t, err)
	assert.Equal(t, v.Status, "completed")
	assert.Equal(t, v.AgentID, "agent-1")
	assert.Equal(t, len(v.Content), 1)
	assert.Equal(t, v.Content[0].Text, "hello")
	assert.Equal(t, v.Prompt, "do something")

	// Round-trip
	data, err := json.Marshal(&v)
	assert.NilError(t, err)

	var v2 models.AgentOutputCompleted
	err = json.Unmarshal(data, &v2)
	assert.NilError(t, err)
	assert.Equal(t, v2.Status, "completed")
	assert.Equal(t, v2.AgentID, v.AgentID)
}

func TestAgentOutput_AsyncLaunched(t *testing.T) {
	input := `{
		"status": "async_launched",
		"agentId": "agent-2",
		"description": "background task",
		"prompt": "run tests",
		"outputFile": "/tmp/output.txt",
		"canReadOutputFile": true
	}`
	var v models.AgentOutputAsyncLaunched
	err := json.Unmarshal([]byte(input), &v)
	assert.NilError(t, err)
	assert.Equal(t, v.Status, "async_launched")
	assert.Equal(t, v.AgentID, "agent-2")
	assert.Assert(t, v.CanReadOutputFile != nil && *v.CanReadOutputFile)

	data, err := json.Marshal(&v)
	assert.NilError(t, err)

	var v2 models.AgentOutputAsyncLaunched
	err = json.Unmarshal(data, &v2)
	assert.NilError(t, err)
	assert.Equal(t, v2.Status, "async_launched")
}

func TestAgentOutput_SubAgentEntered(t *testing.T) {
	input := `{
		"status": "sub_agent_entered",
		"description": "entering sub-agent",
		"message": "processing"
	}`
	var v models.AgentOutputSubAgentEntered
	err := json.Unmarshal([]byte(input), &v)
	assert.NilError(t, err)
	assert.Equal(t, v.Status, "sub_agent_entered")
	assert.Equal(t, v.Description, "entering sub-agent")
}

// --- FileReadOutput discriminated union tests ---

func TestFileReadOutput_Text(t *testing.T) {
	input := `{
		"type": "text",
		"file": {
			"filePath": "/tmp/test.go",
			"content": "package main",
			"numLines": 1,
			"startLine": 1,
			"totalLines": 1
		}
	}`
	var v models.FileReadOutputText
	err := json.Unmarshal([]byte(input), &v)
	assert.NilError(t, err)
	assert.Equal(t, v.Type, "text")
	assert.Equal(t, v.File.FilePath, "/tmp/test.go")
	assert.Equal(t, v.File.Content, "package main")

	data, err := json.Marshal(&v)
	assert.NilError(t, err)

	var v2 models.FileReadOutputText
	err = json.Unmarshal(data, &v2)
	assert.NilError(t, err)
	assert.Equal(t, v2.Type, "text")
	assert.Equal(t, v2.File.FilePath, v.File.FilePath)
}

func TestFileReadOutput_Image(t *testing.T) {
	input := `{
		"type": "image",
		"file": {
			"base64": "aGVsbG8=",
			"type": "image/png",
			"originalSize": 1024,
			"dimensions": {
				"originalWidth": 800,
				"originalHeight": 600
			}
		}
	}`
	var v models.FileReadOutputImage
	err := json.Unmarshal([]byte(input), &v)
	assert.NilError(t, err)
	assert.Equal(t, v.Type, "image")
	assert.Equal(t, v.File.Base64, "aGVsbG8=")
	assert.Equal(t, v.File.Type, "image/png")
	assert.Assert(t, v.File.Dimensions != nil)
}

func TestFileReadOutput_Pdf(t *testing.T) {
	input := `{
		"type": "pdf",
		"file": {
			"filePath": "/tmp/test.pdf",
			"base64": "cGRm",
			"originalSize": 2048
		}
	}`
	var v models.FileReadOutputPdf
	err := json.Unmarshal([]byte(input), &v)
	assert.NilError(t, err)
	assert.Equal(t, v.Type, "pdf")
	assert.Equal(t, v.File.FilePath, "/tmp/test.pdf")
}

func TestFileReadOutput_Parts(t *testing.T) {
	input := `{
		"type": "parts",
		"file": {
			"filePath": "/tmp/large.bin",
			"originalSize": 999999,
			"count": 10,
			"outputDir": "/tmp/parts"
		}
	}`
	var v models.FileReadOutputParts
	err := json.Unmarshal([]byte(input), &v)
	assert.NilError(t, err)
	assert.Equal(t, v.Type, "parts")
	assert.Equal(t, v.File.OutputDir, "/tmp/parts")
}

// --- Flat structure verification ---

func TestAgentOutputCompleted_NoNestedDiscriminator(t *testing.T) {
	v := models.AgentOutputCompleted{
		AgentID: "a1",
		Content: []models.AgentOutputContentBlock{{Type: "text", Text: "hi"}},
		Prompt:  "test",
		Usage: models.AgentOutputUsage{
			InputTokens:  "10",
			OutputTokens: "20",
		},
		TotalToolUseCount: "1",
		TotalDurationMs:   "100",
		TotalTokens:       "30",
	}

	data, err := json.Marshal(&v)
	assert.NilError(t, err)

	var raw map[string]any
	err = json.Unmarshal(data, &raw)
	assert.NilError(t, err)

	// status should be flat, not wrapped
	assert.Equal(t, raw["status"], "completed")
	assert.Equal(t, raw["agentId"], "a1")

	// Should NOT have nested wrapper keys
	_, hasCompleted := raw["completed"]
	assert.Assert(t, !hasCompleted, "should not have nested 'completed' key")
}

func TestPermissionUpdateAddRules_NoNestedDiscriminator(t *testing.T) {
	rc := "ls"
	v := models.PermissionUpdateAddRules{
		Rules:       []models.PermissionRuleValue{{ToolName: "Bash", RuleContent: &rc}},
		Behavior:    models.PermissionBehaviorAllow,
		Destination: models.PermissionUpdateDestinationSession,
	}

	data, err := json.Marshal(&v)
	assert.NilError(t, err)

	var raw map[string]any
	err = json.Unmarshal(data, &raw)
	assert.NilError(t, err)

	assert.Equal(t, raw["type"], "addRules")
	assert.Equal(t, raw["behavior"], "allow")

	_, hasAddRules := raw["addRules"]
	assert.Assert(t, !hasAddRules, "should not have nested 'addRules' key")
}

// --- Enum helper tests ---

func TestIsPermissionBehavior(t *testing.T) {
	assert.Assert(t, models.IsPermissionBehavior("allow"))
	assert.Assert(t, models.IsPermissionBehavior("deny"))
	assert.Assert(t, models.IsPermissionBehavior("ask"))
	assert.Assert(t, !models.IsPermissionBehavior("invalid"))
}

func TestIsPermissionMode(t *testing.T) {
	assert.Assert(t, models.IsPermissionMode("default"))
	assert.Assert(t, models.IsPermissionMode("plan"))
	assert.Assert(t, models.IsPermissionMode("bypassAll"))
	assert.Assert(t, !models.IsPermissionMode("invalid"))
}

func TestIsPermissionUpdateDestination(t *testing.T) {
	assert.Assert(t, models.IsPermissionUpdateDestination("userSettings"))
	assert.Assert(t, models.IsPermissionUpdateDestination("session"))
	assert.Assert(t, !models.IsPermissionUpdateDestination("invalid"))
}

// --- Simple tool output round-trip tests ---

func TestBashOutput_RoundTrip(t *testing.T) {
	input := `{"stdout":"hello","stderr":"","interrupted":false,"isImage":true}`
	var v models.BashOutput
	err := json.Unmarshal([]byte(input), &v)
	assert.NilError(t, err)
	assert.Equal(t, v.Stdout, "hello")
	assert.Assert(t, v.IsImage != nil && *v.IsImage)

	data, err := json.Marshal(&v)
	assert.NilError(t, err)

	var v2 models.BashOutput
	err = json.Unmarshal(data, &v2)
	assert.NilError(t, err)
	assert.Equal(t, v2.Stdout, v.Stdout)
}

func TestGlobOutput_RoundTrip(t *testing.T) {
	input := `{"durationMs":42,"numFiles":2,"filenames":["a.go","b.go"],"truncated":false}`
	var v models.GlobOutput
	err := json.Unmarshal([]byte(input), &v)
	assert.NilError(t, err)
	assert.Equal(t, len(v.Filenames), 2)
	assert.Equal(t, v.Truncated, false)

	data, err := json.Marshal(&v)
	assert.NilError(t, err)

	var v2 models.GlobOutput
	err = json.Unmarshal(data, &v2)
	assert.NilError(t, err)
	assert.DeepEqual(t, v.Filenames, v2.Filenames)
}
