package types

import (
	"encoding/json"
	"testing"
)

// The Write tool's types (FileWriteInput / FileWriteOutput) are defined and
// documented (typescript#write, #write-2) but were never wired into the
// tool-name dispatch, so a "Write" payload silently fell back to
// ToolInputUnknown / ToolOutputUnknown. Regression: a PostToolUse Write
// envelope must round-trip into the typed Write shapes.
func TestHookInput_WriteToolRoundTrip(t *testing.T) {
	original := &PostToolUseHookInput{
		BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "/t.jsonl", Cwd: "/w"},
		HookEventName: HookEventPostToolUse,
		ToolName:      ToolNameWrite,
		ToolInput: NewToolInputSchemas(
			&FileWriteInput{FilePath: "/x/y.go", Content: "package y\n"},
		),
		ToolResponse: NewToolOutputSchemas(&FileWriteOutput{
			Type:     "create",
			FilePath: "/x/y.go",
			Content:  "package y\n",
		}),
		ToolUseID: "tu-1",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got HookInput
	if err := got.UnmarshalJSON(data); err != nil {
		t.Fatalf("HookInput.UnmarshalJSON: %v", err)
	}
	post, ok := got.GetPostToolUseHookInput()
	if !ok {
		t.Fatalf("want *PostToolUseHookInput, got %T", got.GetValue())
	}
	if post.ToolName != ToolNameWrite {
		t.Errorf("tool_name: want %q, got %q", ToolNameWrite, post.ToolName)
	}

	in, ok := post.ToolInput.GetFileWriteInput()
	if !ok {
		t.Fatalf("tool_input: want *FileWriteInput, got %T", post.ToolInput.GetValue())
	}
	if in.FilePath != "/x/y.go" {
		t.Errorf("tool_input.file_path: want /x/y.go, got %q", in.FilePath)
	}

	out, ok := post.ToolResponse.GetFileWriteOutput()
	if !ok {
		t.Fatalf("tool_response: want *FileWriteOutput, got %T", post.ToolResponse.GetValue())
	}
	if out.FilePath != "/x/y.go" {
		t.Errorf("tool_response.filePath: want /x/y.go, got %q", out.FilePath)
	}
}

// Dispatch checks independent of the hook envelope.
func TestUnmarshalToolSchemas_Write(t *testing.T) {
	var in ToolInputSchemas
	if err := in.UnmarshalForTool(
		ToolNameWrite,
		[]byte(`{"file_path":"/a.go","content":"x"}`),
	); err != nil {
		t.Fatalf("ToolInputSchemas.UnmarshalForTool: %v", err)
	}
	if _, ok := in.GetFileWriteInput(); !ok {
		t.Fatalf("want *FileWriteInput, got %T", in.GetValue())
	}

	var out ToolOutputSchemas
	if err := out.UnmarshalForTool(
		ToolNameWrite,
		[]byte(`{"type":"create","filePath":"/a.go","content":"x"}`),
	); err != nil {
		t.Fatalf("ToolOutputSchemas.UnmarshalForTool: %v", err)
	}
	if _, ok := out.GetFileWriteOutput(); !ok {
		t.Fatalf("want *FileWriteOutput, got %T", out.GetValue())
	}
}
