package sdk_typesv1

import (
	"encoding/json"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// jsonKeys returns all top-level keys of a JSON object.
func jsonKeys(t *testing.T, jsonBytes []byte) map[string]bool {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(jsonBytes, &m); err != nil {
		t.Fatalf("failed to unmarshal JSON keys: %v", err)
	}
	keys := make(map[string]bool, len(m))
	for k := range m {
		keys[k] = true
	}
	return keys
}

// assertJSONKeys marshals msg to JSON and checks that all expectedKeys are present.
func assertJSONKeys(t *testing.T, msg proto.Message, expectedKeys []string) []byte {
	t.Helper()
	marshaler := protojson.MarshalOptions{EmitUnpopulated: true}
	b, err := marshaler.Marshal(msg)
	if err != nil {
		t.Fatalf("protojson.Marshal failed: %v", err)
	}
	keys := jsonKeys(t, b)
	for _, k := range expectedKeys {
		if !keys[k] {
			t.Errorf("expected JSON key %q not found in: %s", k, string(b))
		}
	}
	return b
}

// assertRoundTrip marshals msg to JSON, then unmarshals back and checks equality.
func assertRoundTrip(t *testing.T, msg proto.Message) {
	t.Helper()
	marshaler := protojson.MarshalOptions{EmitUnpopulated: true}
	b, err := marshaler.Marshal(msg)
	if err != nil {
		t.Fatalf("protojson.Marshal failed: %v", err)
	}

	target := msg.ProtoReflect().New().Interface()
	if err := protojson.Unmarshal(b, target); err != nil {
		t.Fatalf("protojson.Unmarshal failed for %s: %v\nJSON: %s", msg.ProtoReflect().Descriptor().Name(), err, string(b))
	}

	if !proto.Equal(msg, target) {
		b2, _ := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(target)
		t.Errorf("round-trip mismatch:\n  original: %s\n  decoded:  %s", string(b), string(b2))
	}
}

// assertUnmarshalKeys unmarshals JSON into target and verifies it doesn't error.
func assertUnmarshalKeys(t *testing.T, jsonStr string, target proto.Message) {
	t.Helper()
	if err := protojson.Unmarshal([]byte(jsonStr), target); err != nil {
		t.Fatalf("protojson.Unmarshal failed: %v\nJSON: %s", err, jsonStr)
	}
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
func int32Ptr(i int32) *int32  { return &i }
func int64Ptr(i int64) *int64  { return &i }

// ===== Hook Input Messages =====

func TestPreToolUseHookInput_JSONName(t *testing.T) {
	msg := &PreToolUseHookInput{
		SessionId:      "sess-123",
		TranscriptPath: "/tmp/transcript.json",
		Cwd:            "/home/user",
		PermissionMode: strPtr("default"),
		ToolName:       "Bash",
		ToolInput:      &ToolInput{Input: &ToolInput_Bash{Bash: &BashInput{Command: "ls"}}},
		HookEventName:  "PreToolUse",
	}
	expectedKeys := []string{"session_id", "transcript_path", "cwd", "permission_mode", "tool_name", "tool_input", "hook_event_name"}
	b := assertJSONKeys(t, msg, expectedKeys)
	// Verify hook_event_name value
	if !strings.Contains(string(b), `"hook_event_name":"PreToolUse"`) {
		t.Errorf("hook_event_name value mismatch in: %s", string(b))
	}
	assertRoundTrip(t, msg)
}

func TestPostToolUseHookInput_JSONName(t *testing.T) {
	msg := &PostToolUseHookInput{
		SessionId:      "sess-123",
		TranscriptPath: "/tmp/transcript.json",
		Cwd:            "/home/user",
		PermissionMode: strPtr("default"),
		ToolName:       "Bash",
		ToolInput:      &ToolInput{Input: &ToolInput_Bash{Bash: &BashInput{Command: "ls"}}},
		ToolResponse:   &ToolOutput{Output: &ToolOutput_Bash{Bash: &BashOutput{Output: "file.txt", ExitCode: 0}}},
		HookEventName:  "PostToolUse",
	}
	expectedKeys := []string{"session_id", "transcript_path", "permission_mode", "tool_name", "tool_input", "tool_response", "hook_event_name"}
	assertJSONKeys(t, msg, expectedKeys)
	assertRoundTrip(t, msg)
}

func TestPostToolUseFailureHookInput_JSONName(t *testing.T) {
	msg := &PostToolUseFailureHookInput{
		SessionId:      "sess-123",
		TranscriptPath: "/tmp/transcript.json",
		Cwd:            "/home/user",
		PermissionMode: strPtr("default"),
		ToolName:       "Bash",
		ToolInput:      &ToolInput{Input: &ToolInput_Bash{Bash: &BashInput{Command: "ls"}}},
		Error:          "command failed",
		IsInterrupt:    boolPtr(true),
		HookEventName:  "PostToolUseFailure",
	}
	expectedKeys := []string{"session_id", "transcript_path", "permission_mode", "tool_name", "tool_input", "is_interrupt", "hook_event_name"}
	assertJSONKeys(t, msg, expectedKeys)
	assertRoundTrip(t, msg)
}

func TestNotificationHookInput_JSONName(t *testing.T) {
	msg := &NotificationHookInput{
		SessionId:      "sess-123",
		TranscriptPath: "/tmp/transcript.json",
		Cwd:            "/home/user",
		PermissionMode: strPtr("default"),
		Message:        "Build complete",
		Title:          strPtr("CI"),
		HookEventName:  "Notification",
	}
	expectedKeys := []string{"session_id", "transcript_path", "permission_mode", "hook_event_name"}
	assertJSONKeys(t, msg, expectedKeys)
	assertRoundTrip(t, msg)
}

func TestUserPromptSubmitHookInput_JSONName(t *testing.T) {
	msg := &UserPromptSubmitHookInput{
		SessionId:      "sess-123",
		TranscriptPath: "/tmp/transcript.json",
		Cwd:            "/home/user",
		PermissionMode: strPtr("default"),
		Prompt:         "help me",
		HookEventName:  "UserPromptSubmit",
	}
	expectedKeys := []string{"session_id", "transcript_path", "permission_mode", "hook_event_name"}
	assertJSONKeys(t, msg, expectedKeys)
	assertRoundTrip(t, msg)
}

func TestSessionStartHookInput_JSONName(t *testing.T) {
	msg := &SessionStartHookInput{
		SessionId:      "sess-123",
		TranscriptPath: "/tmp/transcript.json",
		Cwd:            "/home/user",
		PermissionMode: strPtr("default"),
		Source:         "startup",
		HookEventName:  "SessionStart",
	}
	expectedKeys := []string{"session_id", "transcript_path", "permission_mode", "hook_event_name"}
	assertJSONKeys(t, msg, expectedKeys)
	assertRoundTrip(t, msg)
}

func TestSessionEndHookInput_JSONName(t *testing.T) {
	msg := &SessionEndHookInput{
		SessionId:      "sess-123",
		TranscriptPath: "/tmp/transcript.json",
		Cwd:            "/home/user",
		PermissionMode: strPtr("default"),
		Reason:         "user_exit",
		HookEventName:  "SessionEnd",
	}
	expectedKeys := []string{"session_id", "transcript_path", "permission_mode", "hook_event_name"}
	assertJSONKeys(t, msg, expectedKeys)
	assertRoundTrip(t, msg)
}

func TestStopHookInput_JSONName(t *testing.T) {
	msg := &StopHookInput{
		SessionId:      "sess-123",
		TranscriptPath: "/tmp/transcript.json",
		Cwd:            "/home/user",
		PermissionMode: strPtr("default"),
		StopHookActive: true,
		HookEventName:  "Stop",
	}
	expectedKeys := []string{"session_id", "transcript_path", "permission_mode", "stop_hook_active", "hook_event_name"}
	assertJSONKeys(t, msg, expectedKeys)
	assertRoundTrip(t, msg)
}

func TestSubagentStartHookInput_JSONName(t *testing.T) {
	msg := &SubagentStartHookInput{
		SessionId:      "sess-123",
		TranscriptPath: "/tmp/transcript.json",
		Cwd:            "/home/user",
		PermissionMode: strPtr("default"),
		AgentId:        "agent-1",
		AgentType:      "Bash",
		HookEventName:  "SubagentStart",
	}
	expectedKeys := []string{"session_id", "transcript_path", "permission_mode", "agent_id", "agent_type", "hook_event_name"}
	assertJSONKeys(t, msg, expectedKeys)
	assertRoundTrip(t, msg)
}

func TestSubagentStopHookInput_JSONName(t *testing.T) {
	msg := &SubagentStopHookInput{
		SessionId:      "sess-123",
		TranscriptPath: "/tmp/transcript.json",
		Cwd:            "/home/user",
		PermissionMode: strPtr("default"),
		StopHookActive: true,
		HookEventName:  "SubagentStop",
	}
	expectedKeys := []string{"session_id", "transcript_path", "permission_mode", "stop_hook_active", "hook_event_name"}
	assertJSONKeys(t, msg, expectedKeys)
	assertRoundTrip(t, msg)
}

func TestPreCompactHookInput_JSONName(t *testing.T) {
	msg := &PreCompactHookInput{
		SessionId:          "sess-123",
		TranscriptPath:     "/tmp/transcript.json",
		Cwd:                "/home/user",
		PermissionMode:     strPtr("default"),
		Trigger:            "manual",
		CustomInstructions: strPtr("Be concise"),
		HookEventName:      "PreCompact",
	}
	expectedKeys := []string{"session_id", "transcript_path", "permission_mode", "custom_instructions", "hook_event_name"}
	assertJSONKeys(t, msg, expectedKeys)
	assertRoundTrip(t, msg)
}

func TestPermissionRequestHookInput_JSONName(t *testing.T) {
	msg := &PermissionRequestHookInput{
		SessionId:             "sess-123",
		TranscriptPath:        "/tmp/transcript.json",
		Cwd:                   "/home/user",
		PermissionMode:        strPtr("default"),
		ToolName:              "Bash",
		ToolInput:             &ToolInput{Input: &ToolInput_Bash{Bash: &BashInput{Command: "rm -rf /"}}},
		PermissionSuggestions: []*PermissionUpdate{},
		HookEventName:         "PermissionRequest",
	}
	expectedKeys := []string{"session_id", "transcript_path", "permission_mode", "tool_name", "tool_input", "permission_suggestions", "hook_event_name"}
	assertJSONKeys(t, msg, expectedKeys)
	assertRoundTrip(t, msg)
}

// ===== Hook Input Unmarshal from TS-style JSON =====

func TestHookInputUnmarshal_SnakeCaseJSON(t *testing.T) {
	jsonStr := `{
		"session_id": "s1",
		"transcript_path": "/tmp/t.json",
		"cwd": "/home",
		"permission_mode": "default",
		"tool_name": "Bash",
		"hook_event_name": "PreToolUse"
	}`
	msg := &PreToolUseHookInput{}
	assertUnmarshalKeys(t, jsonStr, msg)
	if msg.SessionId != "s1" {
		t.Errorf("session_id = %q, want %q", msg.SessionId, "s1")
	}
	if msg.HookEventName != "PreToolUse" {
		t.Errorf("hook_event_name = %q, want %q", msg.HookEventName, "PreToolUse")
	}
}

// ===== Tool Input Messages =====

func TestAgentInput_JSONName(t *testing.T) {
	msg := &AgentInput{
		Description:  "test task",
		Prompt:       "do something",
		SubagentType: "general-purpose",
	}
	assertJSONKeys(t, msg, []string{"subagent_type"})
	assertRoundTrip(t, msg)
}

func TestBashInput_JSONName(t *testing.T) {
	msg := &BashInput{
		Command:         "ls -la",
		Timeout:         int32Ptr(5000),
		Description:     strPtr("List files"),
		RunInBackground: boolPtr(true),
	}
	assertJSONKeys(t, msg, []string{"run_in_background"})
	assertRoundTrip(t, msg)
}

func TestBashOutputInput_JSONName(t *testing.T) {
	msg := &BashOutputInput{
		BashId: "shell-1",
		Filter: strPtr("error"),
	}
	assertJSONKeys(t, msg, []string{"bash_id"})
	assertRoundTrip(t, msg)
}

func TestFileEditInput_JSONName(t *testing.T) {
	msg := &FileEditInput{
		FilePath:   "/tmp/file.go",
		OldString:  "foo",
		NewString:  "bar",
		ReplaceAll: boolPtr(true),
	}
	assertJSONKeys(t, msg, []string{"file_path", "old_string", "new_string", "replace_all"})
	assertRoundTrip(t, msg)
}

func TestFileReadInput_JSONName(t *testing.T) {
	msg := &FileReadInput{
		FilePath: "/tmp/file.go",
		Offset:   int32Ptr(10),
		Limit:    int32Ptr(100),
	}
	assertJSONKeys(t, msg, []string{"file_path"})
	assertRoundTrip(t, msg)
}

func TestFileWriteInput_JSONName(t *testing.T) {
	msg := &FileWriteInput{
		FilePath: "/tmp/file.go",
		Content:  "hello world",
	}
	assertJSONKeys(t, msg, []string{"file_path"})
	assertRoundTrip(t, msg)
}

func TestGrepInput_JSONName(t *testing.T) {
	msg := &GrepInput{
		Pattern:         "TODO",
		Path:            strPtr("/src"),
		Glob:            strPtr("*.go"),
		Type:            strPtr("go"),
		OutputMode:      "content",
		CaseInsensitive: boolPtr(true),
		ShowLineNumbers: boolPtr(true),
		BeforeContext:   int32Ptr(3),
		AfterContext:    int32Ptr(3),
		Context:         int32Ptr(5),
		HeadLimit:       int32Ptr(100),
		Multiline:       boolPtr(false),
	}
	expectedKeys := []string{"output_mode", "-i", "-n", "-B", "-A", "-C", "head_limit"}
	b := assertJSONKeys(t, msg, expectedKeys)
	t.Logf("GrepInput JSON: %s", string(b))
	assertRoundTrip(t, msg)
}

func TestGrepInput_UnmarshalDashKeys(t *testing.T) {
	// Test that protojson can unmarshal JSON with dash-prefixed keys
	jsonStr := `{
		"pattern": "TODO",
		"output_mode": "content",
		"-i": true,
		"-n": true,
		"-B": 3,
		"-A": 3,
		"-C": 5,
		"head_limit": 100
	}`
	msg := &GrepInput{}
	assertUnmarshalKeys(t, jsonStr, msg)
	if msg.CaseInsensitive == nil || !*msg.CaseInsensitive {
		t.Errorf("-i: expected true, got %v", msg.CaseInsensitive)
	}
	if msg.ShowLineNumbers == nil || !*msg.ShowLineNumbers {
		t.Errorf("-n: expected true, got %v", msg.ShowLineNumbers)
	}
	if msg.BeforeContext == nil || *msg.BeforeContext != 3 {
		t.Errorf("-B: expected 3, got %v", msg.BeforeContext)
	}
	if msg.AfterContext == nil || *msg.AfterContext != 3 {
		t.Errorf("-A: expected 3, got %v", msg.AfterContext)
	}
	if msg.Context == nil || *msg.Context != 5 {
		t.Errorf("-C: expected 5, got %v", msg.Context)
	}
}

func TestNotebookEditInput_JSONName(t *testing.T) {
	msg := &NotebookEditInput{
		NotebookPath: "/tmp/notebook.ipynb",
		CellId:       strPtr("cell-1"),
		NewSource:    "print('hello')",
		CellType:     "code",
		EditMode:     "replace",
	}
	assertJSONKeys(t, msg, []string{"notebook_path", "cell_id", "new_source", "cell_type", "edit_mode"})
	assertRoundTrip(t, msg)
}

func TestWebSearchInput_JSONName(t *testing.T) {
	msg := &WebSearchInput{
		Query:          "golang protobuf",
		AllowedDomains: []string{"golang.org"},
		BlockedDomains: []string{"spam.com"},
	}
	assertJSONKeys(t, msg, []string{"allowed_domains", "blocked_domains"})
	assertRoundTrip(t, msg)
}

// ===== Tool Output Messages =====

func TestUsageInfo_JSONName(t *testing.T) {
	msg := &UsageInfo{
		InputTokens:              100,
		OutputTokens:             200,
		CacheCreationInputTokens: int32Ptr(50),
		CacheReadInputTokens:     int32Ptr(25),
	}
	assertJSONKeys(t, msg, []string{"input_tokens", "output_tokens", "cache_creation_input_tokens", "cache_read_input_tokens"})
	assertRoundTrip(t, msg)
}

func TestTaskOutput_JSONName(t *testing.T) {
	cost := 0.05
	msg := &TaskOutput{
		Result:       "done",
		Usage:        &UsageInfo{InputTokens: 100, OutputTokens: 200},
		TotalCostUsd: &cost,
		DurationMs:   int64Ptr(1500),
	}
	assertJSONKeys(t, msg, []string{"total_cost_usd", "duration_ms"})
	assertRoundTrip(t, msg)
}

func TestEditOutput_JSONName(t *testing.T) {
	msg := &EditOutput{
		Message:      "Edited successfully",
		Replacements: 2,
		FilePath:     "/tmp/file.go",
	}
	assertJSONKeys(t, msg, []string{"file_path"})
	assertRoundTrip(t, msg)
}

func TestTextFileOutput_JSONName(t *testing.T) {
	msg := &TextFileOutput{
		Content:       "hello",
		TotalLines:    100,
		LinesReturned: 50,
	}
	assertJSONKeys(t, msg, []string{"total_lines", "lines_returned"})
	assertRoundTrip(t, msg)
}

func TestImageFileOutput_JSONName(t *testing.T) {
	msg := &ImageFileOutput{
		Image:    "base64data",
		MimeType: "image/png",
		FileSize: 1024,
	}
	assertJSONKeys(t, msg, []string{"mime_type", "file_size"})
	assertRoundTrip(t, msg)
}

func TestPDFPageImage_JSONName(t *testing.T) {
	msg := &PDFPageImage{
		Image:    "base64data",
		MimeType: "image/png",
	}
	assertJSONKeys(t, msg, []string{"mime_type"})
	assertRoundTrip(t, msg)
}

func TestPDFPage_JSONName(t *testing.T) {
	msg := &PDFPage{
		PageNumber: 1,
		Text:       strPtr("Page content"),
		Images:     []*PDFPageImage{{Image: "data", MimeType: "image/png"}},
	}
	assertJSONKeys(t, msg, []string{"page_number"})
	assertRoundTrip(t, msg)
}

func TestPDFFileOutput_JSONName(t *testing.T) {
	msg := &PDFFileOutput{
		Pages:      []*PDFPage{{PageNumber: 1}},
		TotalPages: 5,
	}
	assertJSONKeys(t, msg, []string{"total_pages"})
	assertRoundTrip(t, msg)
}

func TestNotebookCell_JSONName(t *testing.T) {
	msg := &NotebookCell{
		CellType:       "code",
		Source:         "print('hi')",
		ExecutionCount: int32Ptr(1),
	}
	assertJSONKeys(t, msg, []string{"cell_type", "execution_count"})
	assertRoundTrip(t, msg)
}

func TestWriteOutput_JSONName(t *testing.T) {
	msg := &WriteOutput{
		Message:      "Written",
		BytesWritten: 1024,
		FilePath:     "/tmp/out.txt",
	}
	assertJSONKeys(t, msg, []string{"bytes_written", "file_path"})
	assertRoundTrip(t, msg)
}

func TestGlobOutput_JSONName(t *testing.T) {
	msg := &GlobOutput{
		Matches:    []string{"a.go", "b.go"},
		Count:      2,
		SearchPath: "/src",
	}
	assertJSONKeys(t, msg, []string{"search_path"})
	assertRoundTrip(t, msg)
}

func TestGrepMatch_JSONName(t *testing.T) {
	msg := &GrepMatch{
		File:          "main.go",
		LineNumber:    int32Ptr(42),
		Line:          "// TODO: fix",
		BeforeContext: []string{"line1"},
		AfterContext:  []string{"line3"},
	}
	assertJSONKeys(t, msg, []string{"line_number", "before_context", "after_context"})
	assertRoundTrip(t, msg)
}

func TestGrepContentOutput_JSONName(t *testing.T) {
	msg := &GrepContentOutput{
		Matches:      []*GrepMatch{{File: "a.go", Line: "match"}},
		TotalMatches: 1,
	}
	assertJSONKeys(t, msg, []string{"total_matches"})
	assertRoundTrip(t, msg)
}

func TestKillBashOutput_JSONName(t *testing.T) {
	msg := &KillBashOutput{
		Message: "Killed",
		ShellId: "shell-1",
	}
	assertJSONKeys(t, msg, []string{"shell_id"})
	assertRoundTrip(t, msg)
}

func TestNotebookEditOutput_JSONName(t *testing.T) {
	msg := &NotebookEditOutput{
		Message:    "Edited",
		EditType:   "replaced",
		CellId:     strPtr("cell-1"),
		TotalCells: 10,
	}
	assertJSONKeys(t, msg, []string{"edit_type", "cell_id", "total_cells"})
	assertRoundTrip(t, msg)
}

func TestWebFetchOutput_JSONName(t *testing.T) {
	msg := &WebFetchOutput{
		Response:   "content here",
		Url:        "https://example.com",
		FinalUrl:   strPtr("https://example.com/final"),
		StatusCode: int32Ptr(200),
	}
	assertJSONKeys(t, msg, []string{"final_url", "status_code"})
	assertRoundTrip(t, msg)
}

func TestWebSearchOutput_JSONName(t *testing.T) {
	msg := &WebSearchOutput{
		Results:      []*WebSearchResult{{Title: "Example", Url: "https://example.com", Snippet: "A test"}},
		TotalResults: 1,
		Query:        "test",
	}
	assertJSONKeys(t, msg, []string{"total_results"})
	assertRoundTrip(t, msg)
}

func TestTodoStats_JSONName(t *testing.T) {
	msg := &TodoStats{
		Total:      10,
		Pending:    3,
		InProgress: 4,
		Completed:  3,
	}
	assertJSONKeys(t, msg, []string{"in_progress"})
	assertRoundTrip(t, msg)
}

// ===== other.proto types =====

func TestModelUsage_CostUSD_JSONName(t *testing.T) {
	msg := &ModelUsage{
		InputTokens:              100,
		OutputTokens:             200,
		CacheReadInputTokens:     50,
		CacheCreationInputTokens: 25,
		WebSearchRequests:        5,
		CostUsd:                  0.05,
		ContextWindow:            100000,
	}
	b := assertJSONKeys(t, msg, []string{"costUSD"})
	// Ensure it's "costUSD" not "costUsd"
	if !strings.Contains(string(b), `"costUSD"`) {
		t.Errorf("expected costUSD key, got: %s", string(b))
	}
	if strings.Contains(string(b), `"costUsd"`) {
		t.Errorf("unexpected costUsd key (should be costUSD), got: %s", string(b))
	}
	assertRoundTrip(t, msg)
}

func TestNonNullableUsage_JSONName(t *testing.T) {
	msg := &NonNullableUsage{
		InputTokens:              100,
		OutputTokens:             200,
		CacheCreationInputTokens: int64Ptr(50),
		CacheReadInputTokens:     int64Ptr(25),
	}
	assertJSONKeys(t, msg, []string{"input_tokens", "output_tokens", "cache_creation_input_tokens", "cache_read_input_tokens"})
	assertRoundTrip(t, msg)
}

func TestUsage_JSONName(t *testing.T) {
	msg := &Usage{
		InputTokens:              int64Ptr(100),
		OutputTokens:             int64Ptr(200),
		CacheCreationInputTokens: int64Ptr(50),
		CacheReadInputTokens:     int64Ptr(25),
	}
	assertJSONKeys(t, msg, []string{"input_tokens", "output_tokens", "cache_creation_input_tokens", "cache_read_input_tokens"})
	assertRoundTrip(t, msg)
}

// ===== message.proto types =====

func TestSDKAssistantMessage_JSONName(t *testing.T) {
	msg := &SDKAssistantMessage{
		Uuid:              "uuid-1",
		SessionId:         "sess-1",
		ParentToolUseId:   strPtr("tool-1"),
	}
	assertJSONKeys(t, msg, []string{"session_id", "parent_tool_use_id"})
	assertRoundTrip(t, msg)
}

func TestSDKUserMessage_JSONName(t *testing.T) {
	msg := &SDKUserMessage{
		Uuid:            strPtr("uuid-1"),
		SessionId:       "sess-1",
		ParentToolUseId: strPtr("tool-1"),
	}
	assertJSONKeys(t, msg, []string{"session_id", "parent_tool_use_id"})
	assertRoundTrip(t, msg)
}

func TestSDKUserMessageReplay_JSONName(t *testing.T) {
	msg := &SDKUserMessageReplay{
		Uuid:            "uuid-1",
		SessionId:       "sess-1",
		ParentToolUseId: strPtr("tool-1"),
	}
	assertJSONKeys(t, msg, []string{"session_id", "parent_tool_use_id"})
	assertRoundTrip(t, msg)
}

func TestSDKPermissionDenial_JSONName(t *testing.T) {
	msg := &SDKPermissionDenial{
		ToolName:  "Bash",
		ToolUseId: "tu-1",
		ToolInput: &ToolInput{Input: &ToolInput_Bash{Bash: &BashInput{Command: "rm -rf /"}}},
	}
	assertJSONKeys(t, msg, []string{"tool_name", "tool_use_id", "tool_input"})
	assertRoundTrip(t, msg)
}

func TestSDKResultMessage_JSONName(t *testing.T) {
	msg := &SDKResultMessage{
		Subtype:      "success",
		Uuid:         "uuid-1",
		SessionId:    "sess-1",
		DurationMs:   5000,
		DurationApiMs: 4000,
		IsError:      false,
		NumTurns:     3,
		TotalCostUsd: 0.10,
		Usage:        &NonNullableUsage{InputTokens: 100, OutputTokens: 200},
		PermissionDenials: []*SDKPermissionDenial{},
		Result:       strPtr("success"),
	}
	assertJSONKeys(t, msg, []string{
		"session_id", "duration_ms", "duration_api_ms", "is_error",
		"num_turns", "total_cost_usd", "permission_denials",
	})
	assertRoundTrip(t, msg)

	// Verify structured_output json_name via unmarshal
	jsonWithStructuredOutput := `{
		"subtype": "success",
		"uuid": "uuid-1",
		"session_id": "sess-1",
		"duration_ms": "5000",
		"duration_api_ms": "4000",
		"is_error": false,
		"num_turns": 3,
		"total_cost_usd": 0.1,
		"usage": {"input_tokens": "100", "output_tokens": "200"},
		"permission_denials": [],
		"structured_output": "hello",
		"errors": []
	}`
	resultMsg := &SDKResultMessage{}
	assertUnmarshalKeys(t, jsonWithStructuredOutput, resultMsg)
	if resultMsg.StructuredOutput == nil {
		t.Errorf("structured_output should be populated after unmarshal")
	}
}

func TestSDKSystemMessage_JSONName(t *testing.T) {
	msg := &SDKSystemMessage{
		Uuid:           "uuid-1",
		SessionId:      "sess-1",
		ApiKeySource:   "user",
		Cwd:            "/home/user",
		Tools:          []string{"Bash", "Read"},
		McpServers:     []*SDKSystemMcpServer{{Name: "ctx7", Status: "connected"}},
		Model:          "claude-opus-4-6",
		PermissionMode: "default",
		SlashCommands:  []string{"commit", "review"},
		OutputStyle:    "markdown",
	}
	assertJSONKeys(t, msg, []string{"session_id", "mcp_servers", "slash_commands", "output_style"})
	assertRoundTrip(t, msg)
}

func TestSDKPartialAssistantMessage_JSONName(t *testing.T) {
	msg := &SDKPartialAssistantMessage{
		Uuid:            "uuid-1",
		SessionId:       "sess-1",
		ParentToolUseId: strPtr("tool-1"),
	}
	assertJSONKeys(t, msg, []string{"session_id", "parent_tool_use_id"})
	assertRoundTrip(t, msg)
}

func TestCompactMetadata_JSONName(t *testing.T) {
	msg := &CompactMetadata{
		Trigger:   "auto",
		PreTokens: 50000,
	}
	assertJSONKeys(t, msg, []string{"pre_tokens"})
	assertRoundTrip(t, msg)
}

func TestSDKCompactBoundaryMessage_JSONName(t *testing.T) {
	msg := &SDKCompactBoundaryMessage{
		Uuid:            "uuid-1",
		SessionId:       "sess-1",
		CompactMetadata: &CompactMetadata{Trigger: "auto", PreTokens: 50000},
	}
	assertJSONKeys(t, msg, []string{"session_id", "compact_metadata"})
	assertRoundTrip(t, msg)
}
