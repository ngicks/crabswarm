package sdktypesv1

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
		ToolInput:     &FileWriteInput{FilePath: "/x/y.go", Content: "package y\n"},
		ToolResponse: &FileWriteOutput{
			Type:     "create",
			FilePath: "/x/y.go",
			Content:  "package y\n",
		},
		ToolUseID: "tu-1",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := UnmarshalHookInput(data)
	if err != nil {
		t.Fatalf("UnmarshalHookInput: %v", err)
	}
	post, ok := got.(*PostToolUseHookInput)
	if !ok {
		t.Fatalf("want *PostToolUseHookInput, got %T", got)
	}
	if post.ToolName != ToolNameWrite {
		t.Errorf("tool_name: want %q, got %q", ToolNameWrite, post.ToolName)
	}

	in, ok := post.ToolInput.(*FileWriteInput)
	if !ok {
		t.Fatalf("tool_input: want *FileWriteInput, got %T", post.ToolInput)
	}
	if in.FilePath != "/x/y.go" {
		t.Errorf("tool_input.file_path: want /x/y.go, got %q", in.FilePath)
	}

	out, ok := post.ToolResponse.(*FileWriteOutput)
	if !ok {
		t.Fatalf("tool_response: want *FileWriteOutput, got %T", post.ToolResponse)
	}
	if out.FilePath != "/x/y.go" {
		t.Errorf("tool_response.filePath: want /x/y.go, got %q", out.FilePath)
	}
}

// Dispatch checks independent of the hook envelope.
func TestUnmarshalToolSchemas_Write(t *testing.T) {
	in, err := UnmarshalToolInputSchemas(
		ToolNameWrite,
		[]byte(`{"file_path":"/a.go","content":"x"}`),
	)
	if err != nil {
		t.Fatalf("UnmarshalToolInputSchemas: %v", err)
	}
	if _, ok := in.(*FileWriteInput); !ok {
		t.Fatalf("want *FileWriteInput, got %T", in)
	}

	out, err := UnmarshalToolOutputSchemas(
		ToolNameWrite,
		[]byte(`{"type":"create","filePath":"/a.go","content":"x"}`),
	)
	if err != nil {
		t.Fatalf("UnmarshalToolOutputSchemas: %v", err)
	}
	if _, ok := out.(*FileWriteOutput); !ok {
		t.Fatalf("want *FileWriteOutput, got %T", out)
	}
}
