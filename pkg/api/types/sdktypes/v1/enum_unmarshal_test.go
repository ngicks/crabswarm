package sdktypesv1

import (
	"encoding/json"
	"testing"
)

// enumRoundTrip unmarshals input into the union struct T (exercising its
// UnmarshalJSON) and asserts marshaling the result reproduces input verbatim.
func enumRoundTrip[T any](t *testing.T, input string) {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		t.Fatalf("unmarshal %s: %v", input, err)
	}
	got, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal %s: %v", input, err)
	}
	if string(got) != input {
		t.Fatalf("round-trip mismatch\nin:  %s\nout: %s", input, got)
	}
}

// TestEnumUnionJSONRoundTrip covers the in-band-discriminator unions that gained
// an UnmarshalJSON: shape-dispatched (SystemPrompt, ToolsConfig), type/behavior
// dispatched (ThinkingConfig, PermissionResult, McpServerConfig), and the
// type+subtype dispatched SDKMessage, plus unknown-variant preservation.
func TestEnumUnionJSONRoundTrip(t *testing.T) {
	enumRoundTrip[SystemPrompt](t, `"be helpful"`)
	enumRoundTrip[SystemPrompt](t, `{"type":"preset","preset":"claude_code"}`)
	enumRoundTrip[SystemPrompt](t, `{"type":"future","x":1}`)

	enumRoundTrip[ToolsConfig](t, `["Bash","Read"]`)
	enumRoundTrip[ToolsConfig](t, `{"type":"preset","preset":"core"}`)

	enumRoundTrip[ThinkingConfig](t, `{"type":"enabled","budgetTokens":2048}`)
	enumRoundTrip[ThinkingConfig](t, `{"type":"disabled"}`)
	enumRoundTrip[ThinkingConfig](t, `{"type":"adaptive","display":"summarized"}`)

	enumRoundTrip[PermissionResult](t, `{"behavior":"allow","updatedPermissions":[]}`)
	enumRoundTrip[PermissionResult](t, `{"behavior":"deny","message":"nope"}`)

	enumRoundTrip[McpServerConfig](t, `{"command":"srv","args":["--x"]}`)
	enumRoundTrip[McpServerConfig](t, `{"type":"sse","url":"https://x"}`)
	enumRoundTrip[McpServerConfig](t, `{"type":"sdk","name":"n","instance":{}}`)
	enumRoundTrip[McpServerConfig](t, `{"type":"weird","x":1}`)

	commandsChanged := `{"type":"system","subtype":"commands_changed",` +
		`"commands":[],"uuid":"u","session_id":"s"}`
	enumRoundTrip[SDKMessage](t, commandsChanged)
}

// TestEnumUnionGetters checks the typed accessors return the active variant.
func TestEnumUnionGetters(t *testing.T) {
	var sp SystemPrompt
	if err := json.Unmarshal([]byte(`"hi"`), &sp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if sp.GetValue() == nil {
		t.Fatal("GetValue: nil")
	}
	s, ok := sp.GetString()
	if !ok || s.Value != "hi" {
		t.Fatalf("GetString: ok=%v val=%q", ok, s.Value)
	}
	if _, ok := sp.GetPreset(); ok {
		t.Fatal("GetPreset: unexpected match")
	}

	var mc McpServerConfig
	if err := json.Unmarshal([]byte(`{"type":"future"}`), &mc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := mc.GetUnknown(); !ok {
		t.Fatal("GetUnknown: expected unknown variant")
	}
}
