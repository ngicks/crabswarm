package server

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
)

func TestPromptAskUserQuestion_AnswerKeys(t *testing.T) {
	inputData := AskUserQuestionInput{
		Questions: []AskQuestion{
			{
				Question: "How should I format the output?",
				Header:   "Format",
				Options: []AskOption{
					{Label: "Summary", Description: "Brief summary"},
					{Label: "Detailed", Description: "Full details"},
				},
				MultiSelect: false,
			},
			{
				Question: "Which language?",
				Header:   "Language",
				Options: []AskOption{
					{Label: "Go", Description: "Golang"},
					{Label: "Rust", Description: "Rust lang"},
				},
				MultiSelect: false,
			},
		},
	}

	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		userInput  string
		wantAnswer map[string]string
	}{
		{
			name:      "numeric selection uses option labels",
			userInput: "1\n2\n",
			wantAnswer: map[string]string{
				"How should I format the output?": "Summary",
				"Which language?":                 "Rust",
			},
		},
		{
			name:      "free text passthrough",
			userInput: "plain text answer\nTypeScript\n",
			wantAnswer: map[string]string{
				"How should I format the output?": "plain text answer",
				"Which language?":                 "TypeScript",
			},
		},
		{
			name:      "zero prompts for custom then uses typed text",
			userInput: "0\nmy custom format\n0\nPython\n",
			wantAnswer: map[string]string{
				"How should I format the output?": "my custom format",
				"Which language?":                 "Python",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.userInput)
			var writer bytes.Buffer

			prompter := NewPlainPrompter(reader, &writer)

			req := &pb.PermissionRequest{
				HookEventName: "on_tool_use",
				ToolName:      "AskUserQuestion",
				ToolInputJson: string(inputJSON),
				SessionId:     "test-session",
				MessageId:     "test-message",
			}

			resp, err := prompter.promptAskUserQuestion(context.Background(), req)
			if err != nil {
				t.Fatalf("promptAskUserQuestion error: %v", err)
			}

			if resp.HookSpecificOutput.PermissionDecision != pb.PermissionDecision_PERMISSION_DECISION_ALLOW {
				t.Error("expected ALLOW decision")
			}

			// Parse the updatedInput to verify answer keys
			var output AskUserQuestionInput
			if err := json.Unmarshal([]byte(resp.HookSpecificOutput.UpdatedInputJson), &output); err != nil {
				t.Fatalf("failed to unmarshal updatedInput: %v", err)
			}

			for key, want := range tt.wantAnswer {
				got, ok := output.Answers[key]
				if !ok {
					t.Errorf("missing answer key %q; got keys: %v", key, mapKeys(output.Answers))
				} else if got != want {
					t.Errorf("answer[%q] = %q, want %q", key, got, want)
				}
			}

			// Ensure no old-style question_N keys
			for key := range output.Answers {
				if strings.HasPrefix(key, "question_") {
					t.Errorf("found old-style key %q in answers", key)
				}
			}
		})
	}
}

func TestPromptAskUserQuestion_MultiSelect(t *testing.T) {
	inputData := AskUserQuestionInput{
		Questions: []AskQuestion{
			{
				Question: "Which features?",
				Header:   "Features",
				Options: []AskOption{
					{Label: "Auth"},
					{Label: "Logging"},
					{Label: "Metrics"},
				},
				MultiSelect: true,
			},
		},
	}

	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		t.Fatal(err)
	}

	reader := strings.NewReader("1,3\n")
	var writer bytes.Buffer
	prompter := NewPlainPrompter(reader, &writer)

	req := &pb.PermissionRequest{
		HookEventName: "on_tool_use",
		ToolName:      "AskUserQuestion",
		ToolInputJson: string(inputJSON),
		SessionId:     "test-session",
		MessageId:     "test-message",
	}

	resp, err := prompter.promptAskUserQuestion(context.Background(), req)
	if err != nil {
		t.Fatalf("promptAskUserQuestion error: %v", err)
	}

	var output AskUserQuestionInput
	if err := json.Unmarshal([]byte(resp.HookSpecificOutput.UpdatedInputJson), &output); err != nil {
		t.Fatalf("failed to unmarshal updatedInput: %v", err)
	}

	want := "Auth, Metrics"
	got := output.Answers["Which features?"]
	if got != want {
		t.Errorf("multi-select answer = %q, want %q", got, want)
	}
}

func TestPromptExitPlanMode_Allow(t *testing.T) {
	reader := strings.NewReader("a\n")
	var writer bytes.Buffer
	prompter := NewPlainPrompter(reader, &writer)

	req := &pb.PermissionRequest{
		HookEventName: "PreToolUse",
		ToolName:      "ExitPlanMode",
		ToolInputJson: `{"allowedPrompts":[{"tool":"Bash","prompt":"run tests"}]}`,
		SessionId:     "test-session",
		MessageId:     "test-message",
	}

	resp, err := prompter.promptExitPlanMode(context.Background(), req)
	if err != nil {
		t.Fatalf("promptExitPlanMode error: %v", err)
	}
	if resp.HookSpecificOutput.PermissionDecision != pb.PermissionDecision_PERMISSION_DECISION_ALLOW {
		t.Error("expected ALLOW decision")
	}
	output := writer.String()
	if !strings.Contains(output, "Plan Approval") {
		t.Error("expected 'Plan Approval' in output")
	}
	if !strings.Contains(output, "[Bash] run tests") {
		t.Error("expected '[Bash] run tests' in output")
	}
}

func TestPromptExitPlanMode_Deny(t *testing.T) {
	reader := strings.NewReader("d\ntoo risky\n")
	var writer bytes.Buffer
	prompter := NewPlainPrompter(reader, &writer)

	req := &pb.PermissionRequest{
		HookEventName: "PreToolUse",
		ToolName:      "ExitPlanMode",
		ToolInputJson: `{"allowedPrompts":[{"tool":"Bash","prompt":"run tests"}]}`,
		SessionId:     "test-session",
		MessageId:     "test-message",
	}

	resp, err := prompter.promptExitPlanMode(context.Background(), req)
	if err != nil {
		t.Fatalf("promptExitPlanMode error: %v", err)
	}
	if resp.HookSpecificOutput.PermissionDecision != pb.PermissionDecision_PERMISSION_DECISION_DENY {
		t.Error("expected DENY decision")
	}
	if resp.HookSpecificOutput.PermissionDecisionReason != "too risky" {
		t.Errorf("reason = %q, want %q", resp.HookSpecificOutput.PermissionDecisionReason, "too risky")
	}
}

func TestPromptExitPlanMode_Ask(t *testing.T) {
	reader := strings.NewReader("k\n")
	var writer bytes.Buffer
	prompter := NewPlainPrompter(reader, &writer)

	req := &pb.PermissionRequest{
		HookEventName: "PreToolUse",
		ToolName:      "ExitPlanMode",
		ToolInputJson: `{"allowedPrompts":[]}`,
		SessionId:     "test-session",
		MessageId:     "test-message",
	}

	resp, err := prompter.promptExitPlanMode(context.Background(), req)
	if err != nil {
		t.Fatalf("promptExitPlanMode error: %v", err)
	}
	if resp.HookSpecificOutput.PermissionDecision != pb.PermissionDecision_PERMISSION_DECISION_ASK {
		t.Error("expected ASK decision")
	}
}

func TestPromptExitPlanMode_ParseFailFallback(t *testing.T) {
	reader := strings.NewReader("a\n")
	var writer bytes.Buffer
	prompter := NewPlainPrompter(reader, &writer)

	req := &pb.PermissionRequest{
		HookEventName: "PreToolUse",
		ToolName:      "ExitPlanMode",
		ToolInputJson: "not valid json",
		SessionId:     "test-session",
		MessageId:     "test-message",
	}

	resp, err := prompter.promptExitPlanMode(context.Background(), req)
	if err != nil {
		t.Fatalf("promptExitPlanMode error: %v", err)
	}
	// Falls back to promptStandard which only has Allow/Deny
	if resp.HookSpecificOutput.PermissionDecision != pb.PermissionDecision_PERMISSION_DECISION_ALLOW {
		t.Error("expected ALLOW decision from fallback")
	}
	output := writer.String()
	if !strings.Contains(output, "falling back to standard prompt") {
		t.Error("expected fallback message in output")
	}
}

func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
