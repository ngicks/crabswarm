package model

import (
	"encoding/json"
	"testing"

	sdktypes "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
)

// TestEnumRoundTrips tests that all enum conversions round-trip correctly.
func TestEnumRoundTrips(t *testing.T) {
	t.Run("HookEvent", func(t *testing.T) {
		tests := []struct {
			pb    sdktypes.HookEvent
			model HookEvent
		}{
			{sdktypes.HookEvent_HOOK_EVENT_PRE_TOOL_USE, HookEventPreToolUse},
			{sdktypes.HookEvent_HOOK_EVENT_POST_TOOL_USE, HookEventPostToolUse},
			{sdktypes.HookEvent_HOOK_EVENT_POST_TOOL_USE_FAILURE, HookEventPostToolUseFailure},
			{sdktypes.HookEvent_HOOK_EVENT_NOTIFICATION, HookEventNotification},
			{sdktypes.HookEvent_HOOK_EVENT_USER_PROMPT_SUBMIT, HookEventUserPromptSubmit},
			{sdktypes.HookEvent_HOOK_EVENT_SESSION_START, HookEventSessionStart},
			{sdktypes.HookEvent_HOOK_EVENT_SESSION_END, HookEventSessionEnd},
			{sdktypes.HookEvent_HOOK_EVENT_STOP, HookEventStop},
			{sdktypes.HookEvent_HOOK_EVENT_SUBAGENT_START, HookEventSubagentStart},
			{sdktypes.HookEvent_HOOK_EVENT_SUBAGENT_STOP, HookEventSubagentStop},
			{sdktypes.HookEvent_HOOK_EVENT_PRE_COMPACT, HookEventPreCompact},
			{sdktypes.HookEvent_HOOK_EVENT_PERMISSION_REQUEST, HookEventPermissionRequest},
		}
		for _, tt := range tests {
			got := HookEventFromProto(tt.pb)
			if got != tt.model {
				t.Errorf("HookEventFromProto(%v) = %q, want %q", tt.pb, got, tt.model)
			}
			back := got.ToProto()
			if back != tt.pb {
				t.Errorf("(%q).ToProto() = %v, want %v", got, back, tt.pb)
			}
		}
		// Unspecified
		if got := HookEventFromProto(sdktypes.HookEvent_HOOK_EVENT_UNSPECIFIED); got != "" {
			t.Errorf("HookEventFromProto(UNSPECIFIED) = %q, want empty", got)
		}
		if got := HookEvent("").ToProto(); got != sdktypes.HookEvent_HOOK_EVENT_UNSPECIFIED {
			t.Errorf(`HookEvent("").ToProto() = %v, want UNSPECIFIED`, got)
		}
	})

	t.Run("SessionStartSource", func(t *testing.T) {
		tests := []struct {
			pb    sdktypes.SessionStartSource
			model SessionStartSource
		}{
			{sdktypes.SessionStartSource_SESSION_START_SOURCE_STARTUP, SessionStartSourceStartup},
			{sdktypes.SessionStartSource_SESSION_START_SOURCE_RESUME, SessionStartSourceResume},
			{sdktypes.SessionStartSource_SESSION_START_SOURCE_CLEAR, SessionStartSourceClear},
			{sdktypes.SessionStartSource_SESSION_START_SOURCE_COMPACT, SessionStartSourceCompact},
		}
		for _, tt := range tests {
			got := SessionStartSourceFromProto(tt.pb)
			if got != tt.model {
				t.Errorf("SessionStartSourceFromProto(%v) = %q, want %q", tt.pb, got, tt.model)
			}
			if back := got.ToProto(); back != tt.pb {
				t.Errorf("(%q).ToProto() = %v, want %v", got, back, tt.pb)
			}
		}
	})

	t.Run("PreCompactTrigger", func(t *testing.T) {
		tests := []struct {
			pb    sdktypes.PreCompactTrigger
			model PreCompactTrigger
		}{
			{sdktypes.PreCompactTrigger_PRE_COMPACT_TRIGGER_MANUAL, PreCompactTriggerManual},
			{sdktypes.PreCompactTrigger_PRE_COMPACT_TRIGGER_AUTO, PreCompactTriggerAuto},
		}
		for _, tt := range tests {
			got := PreCompactTriggerFromProto(tt.pb)
			if got != tt.model {
				t.Errorf("PreCompactTriggerFromProto(%v) = %q, want %q", tt.pb, got, tt.model)
			}
			if back := got.ToProto(); back != tt.pb {
				t.Errorf("(%q).ToProto() = %v, want %v", got, back, tt.pb)
			}
		}
	})

	t.Run("PermissionMode", func(t *testing.T) {
		tests := []struct {
			pb    sdktypes.PermissionMode
			model PermissionMode
		}{
			{sdktypes.PermissionMode_PERMISSION_MODE_DEFAULT, PermissionModeDefault},
			{sdktypes.PermissionMode_PERMISSION_MODE_ACCEPT_EDITS, PermissionModeAcceptEdits},
			{sdktypes.PermissionMode_PERMISSION_MODE_BYPASS_PERMISSIONS, PermissionModeBypassPermissions},
			{sdktypes.PermissionMode_PERMISSION_MODE_PLAN, PermissionModePlan},
		}
		for _, tt := range tests {
			got := PermissionModeFromProto(tt.pb)
			if got != tt.model {
				t.Errorf("PermissionModeFromProto(%v) = %q, want %q", tt.pb, got, tt.model)
			}
			if back := got.ToProto(); back != tt.pb {
				t.Errorf("(%q).ToProto() = %v, want %v", got, back, tt.pb)
			}
		}
	})

	t.Run("PermissionBehavior", func(t *testing.T) {
		tests := []struct {
			pb    sdktypes.PermissionBehavior
			model PermissionBehavior
		}{
			{sdktypes.PermissionBehavior_PERMISSION_BEHAVIOR_ALLOW, PermissionBehaviorAllow},
			{sdktypes.PermissionBehavior_PERMISSION_BEHAVIOR_DENY, PermissionBehaviorDeny},
			{sdktypes.PermissionBehavior_PERMISSION_BEHAVIOR_ASK, PermissionBehaviorAsk},
		}
		for _, tt := range tests {
			got := PermissionBehaviorFromProto(tt.pb)
			if got != tt.model {
				t.Errorf("PermissionBehaviorFromProto(%v) = %q, want %q", tt.pb, got, tt.model)
			}
			if back := got.ToProto(); back != tt.pb {
				t.Errorf("(%q).ToProto() = %v, want %v", got, back, tt.pb)
			}
		}
	})

	t.Run("McpServerConnectionStatus", func(t *testing.T) {
		tests := []struct {
			pb    sdktypes.McpServerConnectionStatus
			model McpServerConnectionStatus
		}{
			{sdktypes.McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_CONNECTED, McpServerConnectionStatusConnected},
			{sdktypes.McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_FAILED, McpServerConnectionStatusFailed},
			{sdktypes.McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_NEEDS_AUTH, McpServerConnectionStatusNeedsAuth},
			{sdktypes.McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_PENDING, McpServerConnectionStatusPending},
		}
		for _, tt := range tests {
			got := McpServerConnectionStatusFromProto(tt.pb)
			if got != tt.model {
				t.Errorf("McpServerConnectionStatusFromProto(%v) = %q, want %q", tt.pb, got, tt.model)
			}
			if back := got.ToProto(); back != tt.pb {
				t.Errorf("(%q).ToProto() = %v, want %v", got, back, tt.pb)
			}
		}
	})

	t.Run("AgentModel", func(t *testing.T) {
		tests := []struct {
			pb    sdktypes.AgentModel
			model AgentModel
		}{
			{sdktypes.AgentModel_AGENT_MODEL_SONNET, AgentModelSonnet},
			{sdktypes.AgentModel_AGENT_MODEL_OPUS, AgentModelOpus},
			{sdktypes.AgentModel_AGENT_MODEL_HAIKU, AgentModelHaiku},
			{sdktypes.AgentModel_AGENT_MODEL_INHERIT, AgentModelInherit},
		}
		for _, tt := range tests {
			got := AgentModelFromProto(tt.pb)
			if got != tt.model {
				t.Errorf("AgentModelFromProto(%v) = %q, want %q", tt.pb, got, tt.model)
			}
			if back := got.ToProto(); back != tt.pb {
				t.Errorf("(%q).ToProto() = %v, want %v", got, back, tt.pb)
			}
		}
	})
}

// TestPreToolUseHookInputConversion tests message conversion round-trip.
func TestPreToolUseHookInputConversion(t *testing.T) {
	permMode := "default"
	pb := &sdktypes.PreToolUseHookInput{
		SessionId:      "sess-123",
		TranscriptPath: "/tmp/transcript",
		Cwd:            "/home/user",
		PermissionMode: &permMode,
		ToolName:       "Bash",
		ToolInput: &sdktypes.ToolInput{
			Input: &sdktypes.ToolInput_Bash{
				Bash: &sdktypes.BashInput{
					Command: "ls -la",
				},
			},
		},
	}

	m := PreToolUseHookInputFromProto(pb)
	if m.SessionID != "sess-123" {
		t.Errorf("SessionID = %q, want %q", m.SessionID, "sess-123")
	}
	if m.ToolName != "Bash" {
		t.Errorf("ToolName = %q, want %q", m.ToolName, "Bash")
	}
	if m.ToolInput == nil || m.ToolInput.Bash == nil {
		t.Fatal("ToolInput.Bash is nil")
	}
	if m.ToolInput.Bash.Command != "ls -la" {
		t.Errorf("ToolInput.Bash.Command = %q, want %q", m.ToolInput.Bash.Command, "ls -la")
	}

	// Round-trip
	pb2 := m.ToProto()
	if pb2.GetSessionId() != pb.GetSessionId() {
		t.Errorf("round-trip SessionId = %q, want %q", pb2.GetSessionId(), pb.GetSessionId())
	}
	if pb2.GetToolName() != pb.GetToolName() {
		t.Errorf("round-trip ToolName = %q, want %q", pb2.GetToolName(), pb.GetToolName())
	}
	bash := pb2.GetToolInput().GetBash()
	if bash == nil {
		t.Fatal("round-trip ToolInput.Bash is nil")
	}
	if bash.GetCommand() != "ls -la" {
		t.Errorf("round-trip Command = %q, want %q", bash.GetCommand(), "ls -la")
	}
}

// TestNilConversions tests that nil inputs produce nil outputs.
func TestNilConversions(t *testing.T) {
	if got := PreToolUseHookInputFromProto(nil); got != nil {
		t.Errorf("PreToolUseHookInputFromProto(nil) = %v, want nil", got)
	}
	if got := (*PreToolUseHookInput)(nil).ToProto(); got != nil {
		t.Errorf("nil.ToProto() = %v, want nil", got)
	}
	if got := ToolInputFromProto(nil); got != nil {
		t.Errorf("ToolInputFromProto(nil) = %v, want nil", got)
	}
	if got := (*ToolInput)(nil).ToProto(); got != nil {
		t.Errorf("nil.ToProto() = %v, want nil", got)
	}
	if got := HookInputFromProto(nil); got != nil {
		t.Errorf("HookInputFromProto(nil) = %v, want nil", got)
	}
	if got := (*HookInput)(nil).ToProto(); got != nil {
		t.Errorf("nil.ToProto() = %v, want nil", got)
	}
}

// TestJSONMarshalProducesSDKNames verifies that json.Marshal produces SDK-compatible strings.
func TestJSONMarshalProducesSDKNames(t *testing.T) {
	type hookEventWrapper struct {
		Event HookEvent `json:"event"`
	}

	w := hookEventWrapper{Event: HookEventPreToolUse}
	b, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	want := `{"event":"PreToolUse"}`
	if string(b) != want {
		t.Errorf("json.Marshal = %s, want %s", string(b), want)
	}

	// Unmarshal back
	var w2 hookEventWrapper
	if err := json.Unmarshal(b, &w2); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if w2.Event != HookEventPreToolUse {
		t.Errorf("json.Unmarshal event = %q, want %q", w2.Event, HookEventPreToolUse)
	}
}

// TestJSONMarshalHyphenEnum verifies enums with hyphens work correctly.
func TestJSONMarshalHyphenEnum(t *testing.T) {
	type statusWrapper struct {
		Status McpServerConnectionStatus `json:"status"`
	}

	w := statusWrapper{Status: McpServerConnectionStatusNeedsAuth}
	b, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	want := `{"status":"needs-auth"}`
	if string(b) != want {
		t.Errorf("json.Marshal = %s, want %s", string(b), want)
	}

	var w2 statusWrapper
	if err := json.Unmarshal(b, &w2); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if w2.Status != McpServerConnectionStatusNeedsAuth {
		t.Errorf("json.Unmarshal status = %q, want %q", w2.Status, McpServerConnectionStatusNeedsAuth)
	}
}

// TestHookInputOneofConversion tests oneof union conversions.
func TestHookInputOneofConversion(t *testing.T) {
	pb := &sdktypes.HookInput{
		Input: &sdktypes.HookInput_SessionStart{
			SessionStart: &sdktypes.SessionStartHookInput{
				SessionId:      "sess-456",
				TranscriptPath: "/tmp/t",
				Cwd:            "/home",
				Source:         sdktypes.SessionStartSource_SESSION_START_SOURCE_STARTUP,
			},
		},
	}

	m := HookInputFromProto(pb)
	if m.SessionStart == nil {
		t.Fatal("HookInput.SessionStart is nil")
	}
	if m.SessionStart.Source != SessionStartSourceStartup {
		t.Errorf("Source = %q, want %q", m.SessionStart.Source, SessionStartSourceStartup)
	}

	pb2 := m.ToProto()
	ss := pb2.GetSessionStart()
	if ss == nil {
		t.Fatal("round-trip SessionStart is nil")
	}
	if ss.GetSource() != sdktypes.SessionStartSource_SESSION_START_SOURCE_STARTUP {
		t.Errorf("round-trip Source = %v, want STARTUP", ss.GetSource())
	}
}

// TestPermissionUpdateConversion tests permission update oneof conversions.
func TestPermissionUpdateConversion(t *testing.T) {
	pb := &sdktypes.PermissionUpdate{
		Update: &sdktypes.PermissionUpdate_SetMode{
			SetMode: &sdktypes.SetModeUpdate{
				Mode:        sdktypes.PermissionMode_PERMISSION_MODE_PLAN,
				Destination: sdktypes.PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_SESSION,
			},
		},
	}

	m := PermissionUpdateFromProto(pb)
	if m.SetMode == nil {
		t.Fatal("PermissionUpdate.SetMode is nil")
	}
	if m.SetMode.Mode != PermissionModePlan {
		t.Errorf("Mode = %q, want %q", m.SetMode.Mode, PermissionModePlan)
	}
	if m.SetMode.Destination != PermissionUpdateDestinationSession {
		t.Errorf("Destination = %q, want %q", m.SetMode.Destination, PermissionUpdateDestinationSession)
	}

	pb2 := m.ToProto()
	sm := pb2.GetSetMode()
	if sm == nil {
		t.Fatal("round-trip SetMode is nil")
	}
	if sm.GetMode() != sdktypes.PermissionMode_PERMISSION_MODE_PLAN {
		t.Errorf("round-trip Mode = %v, want PLAN", sm.GetMode())
	}
}
