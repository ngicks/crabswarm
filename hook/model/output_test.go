package model

import (
	"encoding/json"
	"testing"
)

func TestAllowHelpers(t *testing.T) {
	t.Run("Allow", func(t *testing.T) {
		output := Allow()
		if output.Decision != DecisionAllow {
			t.Errorf("Expected allow, got %s", output.Decision)
		}
	})

	t.Run("AllowWithMessage", func(t *testing.T) {
		output := AllowWithMessage("test message")
		if output.Decision != DecisionAllow {
			t.Errorf("Expected allow, got %s", output.Decision)
		}
		if output.OutputToModel != "test message" {
			t.Errorf("Expected message 'test message', got '%s'", output.OutputToModel)
		}
	})

	t.Run("AllowAlways", func(t *testing.T) {
		output := AllowAlways()
		if output.Decision != DecisionAllowAlways {
			t.Errorf("Expected allow_always, got %s", output.Decision)
		}
	})
}

func TestBlockHelpers(t *testing.T) {
	t.Run("Block", func(t *testing.T) {
		output := Block("test reason")
		if output.Decision != DecisionBlock {
			t.Errorf("Expected block, got %s", output.Decision)
		}
		if output.Reason != "test reason" {
			t.Errorf("Expected reason 'test reason', got '%s'", output.Reason)
		}
	})

	t.Run("BlockWithMessage", func(t *testing.T) {
		output := BlockWithMessage("test reason", "test message")
		if output.Decision != DecisionBlock {
			t.Errorf("Expected block, got %s", output.Decision)
		}
		if output.Reason != "test reason" {
			t.Errorf("Expected reason 'test reason', got '%s'", output.Reason)
		}
		if output.OutputToModel != "test message" {
			t.Errorf("Expected message 'test message', got '%s'", output.OutputToModel)
		}
	})
}

func TestAllowWithModifiedInput(t *testing.T) {
	input := &BashInput{
		Command:     "echo sanitized",
		Description: "Safe command",
	}

	output, err := AllowWithModifiedInput(input)
	if err != nil {
		t.Fatalf("AllowWithModifiedInput failed: %v", err)
	}

	if output.Decision != DecisionAllow {
		t.Errorf("Expected allow, got %s", output.Decision)
	}

	if len(output.ModifiedInput) == 0 {
		t.Fatal("Expected non-empty ModifiedInput")
	}

	// Verify the JSON is correct
	var parsed BashInput
	if err := json.Unmarshal(output.ModifiedInput, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal ModifiedInput: %v", err)
	}

	if parsed.Command != "echo sanitized" {
		t.Errorf("Expected command 'echo sanitized', got '%s'", parsed.Command)
	}
}

func TestMustAllowWithModifiedInput(t *testing.T) {
	input := &BashInput{Command: "test"}

	// Should not panic
	output := MustAllowWithModifiedInput(input)
	if output.Decision != DecisionAllow {
		t.Errorf("Expected allow, got %s", output.Decision)
	}
}

func TestHookOutputChaining(t *testing.T) {
	output := Allow().
		WithReason("extra reason").
		WithOutputToModel("extra message").
		WithSuppressOutput()

	if output.Decision != DecisionAllow {
		t.Errorf("Expected allow, got %s", output.Decision)
	}
	if output.Reason != "extra reason" {
		t.Errorf("Expected reason 'extra reason', got '%s'", output.Reason)
	}
	if output.OutputToModel != "extra message" {
		t.Errorf("Expected message 'extra message', got '%s'", output.OutputToModel)
	}
	if !output.SuppressOutput {
		t.Error("Expected SuppressOutput to be true")
	}
}

func TestHookSpecificOutputHelpers(t *testing.T) {
	t.Run("UserPromptSubmitOutput", func(t *testing.T) {
		output := UserPromptSubmitOutput("modified prompt")
		if output.ModifiedPrompt != "modified prompt" {
			t.Errorf("Expected modified prompt, got '%s'", output.ModifiedPrompt)
		}
	})

	t.Run("NotificationOutput", func(t *testing.T) {
		output := NotificationOutput(true)
		if !output.Dismiss {
			t.Error("Expected Dismiss to be true")
		}
	})

	t.Run("PreCompactOutput", func(t *testing.T) {
		output := PreCompactOutput("summary to preserve")
		if output.PreserveSummary != "summary to preserve" {
			t.Errorf("Expected preserve summary, got '%s'", output.PreserveSummary)
		}
	})
}

func TestHookOutputJSONMarshaling(t *testing.T) {
	output := Block("not allowed")
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	expected := `{"decision":"block","reason":"not allowed"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}
