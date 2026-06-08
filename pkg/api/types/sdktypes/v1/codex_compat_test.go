package sdktypesv1

import (
	"encoding/json"
	"testing"
)

// Codex sends apply_patch's tool_response as a bare string rather than the
// JSON object Claude uses. UnknownUnion (and thus ToolOutputUnknown) must
// preserve any JSON value instead of failing the whole parse on a
// non-object — regression for the "cannot unmarshal string into
// map[string]json.RawMessage" hook-input failure.
func TestUnknownUnion_NonObjectValues(t *testing.T) {
	for _, raw := range []string{
		`"Exit code: 0\nWall time: 0.5 seconds"`,
		`42`,
		`["a","b"]`,
		`true`,
		`null`,
	} {
		var u UnknownUnion
		if err := json.Unmarshal([]byte(raw), &u); err != nil {
			t.Fatalf("UnknownUnion.UnmarshalJSON(%s): unexpected error: %v", raw, err)
		}
		if u.Discriminator != "" {
			t.Errorf(
				"UnknownUnion.UnmarshalJSON(%s): want empty discriminator, got %q",
				raw,
				u.Discriminator,
			)
		}
		if string(u.Raw) != raw {
			t.Errorf("UnknownUnion.UnmarshalJSON(%s): raw not preserved, got %s", raw, u.Raw)
		}
	}
}

// A Codex PostToolUse envelope for apply_patch: tool_name is unknown to the
// SDK and tool_response is a string. It must parse without error so the
// exec hook can decide whether it applies.
func TestUnmarshalHookInput_CodexApplyPatch(t *testing.T) {
	const payload = `{
		"session_id": "sess",
		"transcript_path": "/tmp/x.jsonl",
		"cwd": "/repo",
		"hook_event_name": "PostToolUse",
		"tool_name": "apply_patch",
		"tool_input": {"command": "*** Begin Patch\n*** Add File: a.go\n+package a\n*** End Patch\n"},
		"tool_response": "Exit code: 0\nOutput: ok\n",
		"tool_use_id": "call_1"
	}`

	var in HookInput
	if err := in.UnmarshalJSON([]byte(payload)); err != nil {
		t.Fatalf("HookInput.UnmarshalJSON: unexpected error: %v", err)
	}
	post, ok := in.GetPostToolUseHookInput()
	if !ok {
		t.Fatalf("want *PostToolUseHookInput, got %T", in.GetValue())
	}
	if post.ToolName != "apply_patch" {
		t.Errorf("tool_name: want apply_patch, got %q", post.ToolName)
	}
	out, ok := post.ToolResponse.GetToolOutputUnknown()
	if !ok {
		t.Fatalf("tool_response: want *ToolOutputUnknown, got %T", post.ToolResponse.GetValue())
	}
	if want := `"Exit code: 0\nOutput: ok\n"`; string(out.Raw) != want {
		t.Errorf("tool_response raw: want %s, got %s", want, out.Raw)
	}
	if _, ok := post.ToolInput.GetToolInputUnknown(); !ok {
		t.Fatalf("tool_input: want *ToolInputUnknown, got %T", post.ToolInput.GetValue())
	}
}
