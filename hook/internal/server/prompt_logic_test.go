package server

import (
	"encoding/json"
	"testing"

	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
)

func TestResolveAnswers(t *testing.T) {
	options := []AskOption{
		{Label: "Alpha", Description: "first"},
		{Label: "Beta", Description: "second"},
		{Label: "Gamma", Description: "third"},
	}

	tests := []struct {
		name   string
		choice string
		want   string
	}{
		{"single numeric", "1", "Alpha"},
		{"second option", "2", "Beta"},
		{"multi-select", "1,3", "Alpha, Gamma"},
		{"free text passthrough", "my custom answer", "my custom answer"},
		{"out of range falls back", "99", "99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveAnswers(tt.choice, options)
			if got != tt.want {
				t.Errorf("ResolveAnswers(%q) = %q, want %q", tt.choice, got, tt.want)
			}
		})
	}
}

func TestParseAskUserInput(t *testing.T) {
	inputJSON := `{"questions":[{"question":"Pick one?","header":"Q","options":[{"label":"A"},{"label":"B"}],"multiSelect":false}]}`
	input, err := ParseAskUserInput(inputJSON)
	if err != nil {
		t.Fatalf("ParseAskUserInput error: %v", err)
	}
	if len(input.Questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(input.Questions))
	}
	if input.Questions[0].Question != "Pick one?" {
		t.Errorf("question = %q, want %q", input.Questions[0].Question, "Pick one?")
	}
}

func TestParseAskUserInput_Invalid(t *testing.T) {
	_, err := ParseAskUserInput("not json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestBuildPermissionResponse(t *testing.T) {
	req := &pb.PermissionRequest{
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
	}

	resp := BuildPermissionResponse(req, pb.PermissionDecision_PERMISSION_DECISION_ALLOW, "")
	if !resp.ShouldContinue {
		t.Error("expected ShouldContinue=true")
	}
	if resp.HookSpecificOutput.PermissionDecision != pb.PermissionDecision_PERMISSION_DECISION_ALLOW {
		t.Error("expected ALLOW decision")
	}
	if resp.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Errorf("HookEventName = %q, want %q", resp.HookSpecificOutput.HookEventName, "PreToolUse")
	}

	resp = BuildPermissionResponse(req, pb.PermissionDecision_PERMISSION_DECISION_DENY, "unsafe")
	if resp.HookSpecificOutput.PermissionDecision != pb.PermissionDecision_PERMISSION_DECISION_DENY {
		t.Error("expected DENY decision")
	}
	if resp.HookSpecificOutput.PermissionDecisionReason != "unsafe" {
		t.Errorf("reason = %q, want %q", resp.HookSpecificOutput.PermissionDecisionReason, "unsafe")
	}
}

func TestParseExitPlanModeInput(t *testing.T) {
	inputJSON := `{"allowedPrompts":[{"tool":"Bash","prompt":"run tests"},{"tool":"Bash","prompt":"install dependencies"}],"pushToRemote":true}`
	input, err := ParseExitPlanModeInput(inputJSON)
	if err != nil {
		t.Fatalf("ParseExitPlanModeInput error: %v", err)
	}
	if len(input.AllowedPrompts) != 2 {
		t.Fatalf("expected 2 allowedPrompts, got %d", len(input.AllowedPrompts))
	}
	if input.AllowedPrompts[0].Tool != "Bash" {
		t.Errorf("first prompt tool = %q, want %q", input.AllowedPrompts[0].Tool, "Bash")
	}
	if input.AllowedPrompts[0].Prompt != "run tests" {
		t.Errorf("first prompt = %q, want %q", input.AllowedPrompts[0].Prompt, "run tests")
	}
	if input.AllowedPrompts[1].Prompt != "install dependencies" {
		t.Errorf("second prompt = %q, want %q", input.AllowedPrompts[1].Prompt, "install dependencies")
	}
	if !input.PushToRemote {
		t.Error("expected pushToRemote=true")
	}
}

func TestParseExitPlanModeInput_Empty(t *testing.T) {
	inputJSON := `{}`
	input, err := ParseExitPlanModeInput(inputJSON)
	if err != nil {
		t.Fatalf("ParseExitPlanModeInput error: %v", err)
	}
	if len(input.AllowedPrompts) != 0 {
		t.Errorf("expected 0 allowedPrompts, got %d", len(input.AllowedPrompts))
	}
	if input.PushToRemote {
		t.Error("expected pushToRemote=false")
	}
}

func TestParseExitPlanModeInput_Invalid(t *testing.T) {
	_, err := ParseExitPlanModeInput("not json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestBuildAskUserResponse(t *testing.T) {
	req := &pb.PermissionRequest{
		HookEventName: "PreToolUse",
		ToolName:      "AskUserQuestion",
		ToolInputJson: `{"questions":[{"question":"Which?","header":"Q","options":[{"label":"A"},{"label":"B"}],"multiSelect":false}]}`,
	}

	input, err := ParseAskUserInput(req.ToolInputJson)
	if err != nil {
		t.Fatal(err)
	}

	answers := map[string]string{"Which?": "A"}
	resp, err := BuildAskUserResponse(req, input, answers)
	if err != nil {
		t.Fatal(err)
	}

	if resp.HookSpecificOutput.PermissionDecision != pb.PermissionDecision_PERMISSION_DECISION_ALLOW {
		t.Error("expected ALLOW")
	}

	var output AskUserQuestionInput
	if err := json.Unmarshal([]byte(resp.HookSpecificOutput.UpdatedInputJson), &output); err != nil {
		t.Fatalf("failed to unmarshal updatedInput: %v", err)
	}

	if output.Answers["Which?"] != "A" {
		t.Errorf("answer = %q, want %q", output.Answers["Which?"], "A")
	}
}
