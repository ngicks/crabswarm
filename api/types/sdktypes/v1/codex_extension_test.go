package sdktypesv1

import (
	"encoding/json"
	"testing"
)

// TestToolInputSchemas_UnmarshalForTool_Codex checks tool-name dispatch routes
// Codex-specific inputs to their typed variants while Codex tools that reuse a
// Claude shape (Bash, mcp__*) still decode with the Claude type.
func TestToolInputSchemas_UnmarshalForTool_Codex(t *testing.T) {
	t.Run("apply_patch", func(t *testing.T) {
		const raw = `{"command":"*** Begin Patch\n*** Add File: a.go\n+x\n*** End Patch\n"}`
		var in ToolInputSchemas
		if err := in.UnmarshalForTool("apply_patch", []byte(raw)); err != nil {
			t.Fatal(err)
		}
		c, ok := in.GetCodexApplyPatchInput()
		if !ok {
			t.Fatalf("want *CodexApplyPatchInput, got %T", in.GetValue())
		}
		if files := c.Patch.EditedFiles(); len(files) != 1 || files[0] != "a.go" {
			t.Fatalf("edited files: want [a.go], got %v", files)
		}
	})

	t.Run("update_plan", func(t *testing.T) {
		const raw = `{"plan":[{"step":"s","status":"in_progress"}]}`
		var in ToolInputSchemas
		if err := in.UnmarshalForTool("update_plan", []byte(raw)); err != nil {
			t.Fatal(err)
		}
		c, ok := in.GetCodexUpdatePlanInput()
		if !ok {
			t.Fatalf("want *CodexUpdatePlanInput, got %T", in.GetValue())
		}
		if len(c.Plan) != 1 || c.Plan[0].Status != "in_progress" {
			t.Fatalf("plan: got %v", c.Plan)
		}
	})

	t.Run("bash reuses Claude BashInput", func(t *testing.T) {
		var in ToolInputSchemas
		if err := in.UnmarshalForTool("Bash", []byte(`{"command":"ls"}`)); err != nil {
			t.Fatal(err)
		}
		if _, ok := in.GetBashInput(); !ok {
			t.Fatalf("want *BashInput, got %T", in.GetValue())
		}
	})
}

// TestToolOutputSchemas_UnmarshalForTool_Codex checks the output dispatch: the
// Codex string-shaped responses route to Codex variants, the Bash object form
// stays Claude's BashOutput, and MCP output stays ToolOutputUnknown.
func TestToolOutputSchemas_UnmarshalForTool_Codex(t *testing.T) {
	t.Run("apply_patch string", func(t *testing.T) {
		const raw = `"Exit code: 0\nWall time: 0.1 seconds\nOutput:\nupdated\n"`
		var out ToolOutputSchemas
		if err := out.UnmarshalForTool("apply_patch", []byte(raw)); err != nil {
			t.Fatal(err)
		}
		c, ok := out.GetCodexApplyPatchOutput()
		if !ok {
			t.Fatalf("want *CodexApplyPatchOutput, got %T", out.GetValue())
		}
		if c.ExitCode == nil || *c.ExitCode != 0 {
			t.Fatalf("exit code: got %v", c.ExitCode)
		}
	})

	t.Run("bash string routes to CodexShellOutput", func(t *testing.T) {
		var out ToolOutputSchemas
		if err := out.UnmarshalForTool("Bash", []byte(`"raw output\n"`)); err != nil {
			t.Fatal(err)
		}
		c, ok := out.GetCodexShellOutput()
		if !ok {
			t.Fatalf("want *CodexShellOutput, got %T", out.GetValue())
		}
		if c.Output != "raw output\n" {
			t.Fatalf("output: got %q", c.Output)
		}
	})

	t.Run("bash object stays Claude BashOutput", func(t *testing.T) {
		var out ToolOutputSchemas
		if err := out.UnmarshalForTool("Bash", []byte(`{"stdout":"x"}`)); err != nil {
			t.Fatal(err)
		}
		if _, ok := out.GetBashOutput(); !ok {
			t.Fatalf("want *BashOutput, got %T", out.GetValue())
		}
	})

	t.Run("mcp output stays ToolOutputUnknown", func(t *testing.T) {
		const raw = `{"content":[{"type":"text","text":"x"}]}`
		var out ToolOutputSchemas
		if err := out.UnmarshalForTool("mcp__srv__tool", []byte(raw)); err != nil {
			t.Fatal(err)
		}
		if _, ok := out.GetToolOutputUnknown(); !ok {
			t.Fatalf("want *ToolOutputUnknown, got %T", out.GetValue())
		}
	})
}

// TestCodexToolInputProtoRoundTrip proves Codex tool inputs convey through the
// gRPC ToolInput message (typed Codex oneof variant) without loss.
func TestCodexToolInputProtoRoundTrip(t *testing.T) {
	cases := []struct {
		name, toolName, raw string
	}{
		{
			name:     "apply_patch",
			toolName: "apply_patch",
			raw:      `{"command":"*** Begin Patch\n*** Add File: a.go\n+x\n*** End Patch\n"}`,
		},
		{
			name:     "update_plan",
			toolName: "update_plan",
			raw:      `{"explanation":"e","plan":[{"step":"s","status":"pending"}]}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var in ToolInputSchemas
			if err := in.UnmarshalForTool(tc.toolName, []byte(tc.raw)); err != nil {
				t.Fatal(err)
			}
			p := rawToolInputToProto(in, tc.toolName)
			if p == nil {
				t.Fatal("rawToolInputToProto returned nil")
			}
			back, err := rawFromToolInputProto(p, tc.toolName)
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.Marshal(back)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.raw {
				t.Fatalf("proto round-trip mismatch\nwant: %s\ngot:  %s", tc.raw, got)
			}
		})
	}
}

// TestCodexToolOutputProtoRoundTrip proves Codex tool responses convey through
// the gRPC ToolOutput message without loss.
func TestCodexToolOutputProtoRoundTrip(t *testing.T) {
	cases := []struct {
		name, toolName, raw string
	}{
		{
			name:     "apply_patch",
			toolName: "apply_patch",
			raw:      `"Exit code: 0\nWall time: 0.1 seconds\nOutput:\nupdated\n"`,
		},
		{
			name:     "bash shell",
			toolName: "Bash",
			raw:      `"total 0\nfile.go\n"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out ToolOutputSchemas
			if err := out.UnmarshalForTool(tc.toolName, []byte(tc.raw)); err != nil {
				t.Fatal(err)
			}
			p := rawToolOutputToProto(out, tc.toolName)
			if p == nil {
				t.Fatal("rawToolOutputToProto returned nil")
			}
			back, err := rawFromToolOutputProto(p, tc.toolName)
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.Marshal(back)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.raw {
				t.Fatalf("proto round-trip mismatch\nwant: %s\ngot:  %s", tc.raw, got)
			}
		})
	}
}

// TestHookInputProtoRoundTrip_CodexApplyPatch proves a full Codex PostToolUse
// apply_patch envelope survives HookInput proto conversion (i.e. is conveyable
// through gRPC) with the decoded patch and exec output intact.
func TestHookInputProtoRoundTrip_CodexApplyPatch(t *testing.T) {
	const payload = `{"session_id":"s","transcript_path":"t","cwd":"/repo",` +
		`"hook_event_name":"PostToolUse","tool_name":"apply_patch",` +
		`"tool_input":{"command":"*** Begin Patch\n*** Add File: a.go\n+x\n*** End Patch\n"},` +
		`"tool_response":"Exit code: 0\nWall time: 0.1 seconds\nOutput:\nupdated\n",` +
		`"tool_use_id":"call_1"}`

	var in HookInput
	if err := in.UnmarshalJSON([]byte(payload)); err != nil {
		t.Fatal(err)
	}
	p, err := HookInputToProto(in)
	if err != nil {
		t.Fatal(err)
	}
	back, err := HookInputFromProto(p)
	if err != nil {
		t.Fatal(err)
	}
	post, ok := back.GetPostToolUseHookInput()
	if !ok {
		t.Fatalf("want *PostToolUseHookInput, got %T", back.GetValue())
	}
	inputVar, ok := post.ToolInput.GetCodexApplyPatchInput()
	if !ok {
		t.Fatalf("tool_input: want *CodexApplyPatchInput, got %T", post.ToolInput.GetValue())
	}
	if files := inputVar.Patch.EditedFiles(); len(files) != 1 || files[0] != "a.go" {
		t.Fatalf("edited files: want [a.go], got %v", files)
	}
	outVar, ok := post.ToolResponse.GetCodexApplyPatchOutput()
	if !ok {
		t.Fatalf("tool_response: want *CodexApplyPatchOutput, got %T", post.ToolResponse.GetValue())
	}
	if outVar.ExitCode == nil || *outVar.ExitCode != 0 {
		t.Fatalf("exit code: got %v", outVar.ExitCode)
	}
}
