package server

import (
	"encoding/json"
	"fmt"
	"strings"

	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
)

// AskUserQuestionInput is a minimal representation of AskUserQuestion tool input.
type AskUserQuestionInput struct {
	Questions []AskQuestion     `json:"questions"`
	Answers   map[string]string `json:"answers,omitempty"`
}

// AskQuestion represents a single question in an AskUserQuestion tool input.
type AskQuestion struct {
	Question    string      `json:"question"`
	Header      string      `json:"header"`
	Options     []AskOption `json:"options"`
	MultiSelect bool        `json:"multiSelect"`
}

// AskOption represents a single option in an AskUserQuestion question.
type AskOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// ParseAskUserInput parses AskUserQuestion tool input JSON.
func ParseAskUserInput(toolInputJSON string) (AskUserQuestionInput, error) {
	var input AskUserQuestionInput
	if err := json.Unmarshal([]byte(toolInputJSON), &input); err != nil {
		return AskUserQuestionInput{}, fmt.Errorf("failed to parse AskUserQuestion input: %w", err)
	}
	return input, nil
}

// ResolveAnswers converts numeric choices to option labels.
func ResolveAnswers(choice string, options []AskOption) string {
	parts := strings.Split(choice, ",")
	var labels []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		var idx int
		if _, err := fmt.Sscanf(p, "%d", &idx); err == nil && idx >= 1 && idx <= len(options) {
			labels = append(labels, options[idx-1].Label)
		}
	}
	if len(labels) == 0 {
		return choice
	}
	return strings.Join(labels, ", ")
}

// ExitPlanModeInput is a minimal representation of ExitPlanMode tool input.
type ExitPlanModeInput struct {
	AllowedPrompts []AllowedPrompt `json:"allowedPrompts,omitempty"`
	PushToRemote   bool            `json:"pushToRemote,omitempty"`
}

// AllowedPrompt represents a prompt-based permission that the plan will request.
type AllowedPrompt struct {
	Tool   string `json:"tool"`
	Prompt string `json:"prompt"`
}

// ParseExitPlanModeInput parses ExitPlanMode tool input JSON.
func ParseExitPlanModeInput(toolInputJSON string) (ExitPlanModeInput, error) {
	var input ExitPlanModeInput
	if err := json.Unmarshal([]byte(toolInputJSON), &input); err != nil {
		return ExitPlanModeInput{}, fmt.Errorf("failed to parse ExitPlanMode input: %w", err)
	}
	return input, nil
}

// BuildPermissionResponse builds a PermissionResponse for a standard permission decision.
func BuildPermissionResponse(req *pb.PermissionRequest, decision pb.PermissionDecision, reason string) *pb.PermissionResponse {
	return &pb.PermissionResponse{
		ShouldContinue: true,
		HookSpecificOutput: &pb.HookSpecificOutput{
			HookEventName:           req.HookEventName,
			PermissionDecision:      decision,
			PermissionDecisionReason: reason,
		},
	}
}

// BuildAskUserResponse builds a PermissionResponse for an AskUserQuestion with answers filled in.
func BuildAskUserResponse(req *pb.PermissionRequest, input AskUserQuestionInput, answers map[string]string) (*pb.PermissionResponse, error) {
	input.Answers = answers
	updatedJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated input: %w", err)
	}

	return &pb.PermissionResponse{
		ShouldContinue: true,
		HookSpecificOutput: &pb.HookSpecificOutput{
			HookEventName:      req.HookEventName,
			PermissionDecision: pb.PermissionDecision_PERMISSION_DECISION_ALLOW,
			UpdatedInputJson:   string(updatedJSON),
		},
	}, nil
}
