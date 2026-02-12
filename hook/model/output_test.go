package model

import (
	"encoding/json"
	"testing"
)

func TestAllowHelpers(t *testing.T) {
	t.Run("Allow", func(t *testing.T) {
		output := Allow()
		if output.HookSpecificOutput != nil {
			t.Error("Expected nil HookSpecificOutput for empty allow")
		}
	})

	t.Run("AllowWithEvent", func(t *testing.T) {
		output := AllowWithEvent(HookEventPreToolUse)
		if output.HookSpecificOutput == nil {
			t.Fatal("Expected non-nil HookSpecificOutput")
		}
		if output.HookSpecificOutput.PermissionDecision != PermissionAllow {
			t.Errorf("Expected allow, got %s", output.HookSpecificOutput.PermissionDecision)
		}
		if output.HookSpecificOutput.HookEventName != HookEventPreToolUse {
			t.Errorf("Expected PreToolUse, got %s", output.HookSpecificOutput.HookEventName)
		}
	})

	t.Run("AllowWithSystemMessage", func(t *testing.T) {
		output := AllowWithSystemMessage("test message")
		if output.SystemMessage != "test message" {
			t.Errorf("Expected message 'test message', got '%s'", output.SystemMessage)
		}
	})
}

func TestDenyHelpers(t *testing.T) {
	t.Run("Deny", func(t *testing.T) {
		output := Deny(HookEventPreToolUse, "test reason")
		if output.HookSpecificOutput == nil {
			t.Fatal("Expected non-nil HookSpecificOutput")
		}
		if output.HookSpecificOutput.PermissionDecision != PermissionDeny {
			t.Errorf("Expected deny, got %s", output.HookSpecificOutput.PermissionDecision)
		}
		if output.HookSpecificOutput.PermissionDecisionReason != "test reason" {
			t.Errorf("Expected reason 'test reason', got '%s'", output.HookSpecificOutput.PermissionDecisionReason)
		}
	})
}

func TestAskHelper(t *testing.T) {
	output := Ask(HookEventPreToolUse)
	if output.HookSpecificOutput == nil {
		t.Fatal("Expected non-nil HookSpecificOutput")
	}
	if output.HookSpecificOutput.PermissionDecision != PermissionAsk {
		t.Errorf("Expected ask, got %s", output.HookSpecificOutput.PermissionDecision)
	}
}

func TestStopHelper(t *testing.T) {
	output := Stop("stopping now")
	if output.Continue == nil || *output.Continue != false {
		t.Error("Expected Continue to be false")
	}
	if output.StopReason != "stopping now" {
		t.Errorf("Expected stop reason 'stopping now', got '%s'", output.StopReason)
	}
}

func TestAllowWithUpdatedInput(t *testing.T) {
	input := &BashInput{
		Command:     "echo sanitized",
		Description: "Safe command",
	}

	output, err := AllowWithUpdatedInput(HookEventPreToolUse, input)
	if err != nil {
		t.Fatalf("AllowWithUpdatedInput failed: %v", err)
	}

	if output.HookSpecificOutput == nil {
		t.Fatal("Expected non-nil HookSpecificOutput")
	}
	if output.HookSpecificOutput.PermissionDecision != PermissionAllow {
		t.Errorf("Expected allow, got %s", output.HookSpecificOutput.PermissionDecision)
	}
	if len(output.HookSpecificOutput.UpdatedInput) == 0 {
		t.Fatal("Expected non-empty UpdatedInput")
	}

	var parsed BashInput
	if err := json.Unmarshal(output.HookSpecificOutput.UpdatedInput, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal UpdatedInput: %v", err)
	}
	if parsed.Command != "echo sanitized" {
		t.Errorf("Expected command 'echo sanitized', got '%s'", parsed.Command)
	}
}

func TestMustAllowWithUpdatedInput(t *testing.T) {
	input := &BashInput{Command: "test"}

	// Should not panic
	output := MustAllowWithUpdatedInput(HookEventPreToolUse, input)
	if output.HookSpecificOutput == nil {
		t.Fatal("Expected non-nil HookSpecificOutput")
	}
	if output.HookSpecificOutput.PermissionDecision != PermissionAllow {
		t.Errorf("Expected allow, got %s", output.HookSpecificOutput.PermissionDecision)
	}
}

func TestHookOutputChaining(t *testing.T) {
	output := AllowWithEvent(HookEventPreToolUse).
		WithSystemMessage("extra message").
		WithSuppressOutput()

	if output.SystemMessage != "extra message" {
		t.Errorf("Expected message 'extra message', got '%s'", output.SystemMessage)
	}
	if !output.SuppressOutput {
		t.Error("Expected SuppressOutput to be true")
	}
}

func TestHookOutputJSONMarshaling(t *testing.T) {
	output := Deny(HookEventPreToolUse, "not allowed")
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	hso, ok := result["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatal("Expected hookSpecificOutput in JSON")
	}
	if hso["permissionDecision"] != "deny" {
		t.Errorf("Expected deny, got %v", hso["permissionDecision"])
	}
	if hso["permissionDecisionReason"] != "not allowed" {
		t.Errorf("Expected 'not allowed', got %v", hso["permissionDecisionReason"])
	}
	if hso["hookEventName"] != "PreToolUse" {
		t.Errorf("Expected PreToolUse, got %v", hso["hookEventName"])
	}
}

func TestEmptyOutputJSON(t *testing.T) {
	output := EmptyOutput()
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	if string(data) != "{}" {
		t.Errorf("Expected {}, got %s", string(data))
	}
}

func TestWithAdditionalContext(t *testing.T) {
	output := AllowWithEvent(HookEventPreToolUse).WithAdditionalContext("some context")
	if output.HookSpecificOutput.AdditionalContext != "some context" {
		t.Errorf("Expected 'some context', got '%s'", output.HookSpecificOutput.AdditionalContext)
	}
}
