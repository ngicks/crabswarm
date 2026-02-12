package model

import "encoding/json"

// PermissionDecision represents the hook permission decision type.
type PermissionDecision string

// PermissionDecision constants for HookSpecificOutput.
const (
	// PermissionAllow auto-approves the tool execution.
	PermissionAllow PermissionDecision = "allow"

	// PermissionDeny blocks the tool execution.
	PermissionDeny PermissionDecision = "deny"

	// PermissionAsk prompts the user for confirmation.
	PermissionAsk PermissionDecision = "ask"
)

// HookOutput represents the JSON output sent to Claude Code via stdout.
// This matches the official Claude Agent SDK hooks protocol.
// See https://platform.claude.com/docs/en/agent-sdk/hooks for the official protocol.
type HookOutput struct {
	// Continue indicates whether the agent should continue after this hook (default: true).
	Continue *bool `json:"continue,omitempty"`

	// StopReason is the message shown when Continue is false.
	StopReason string `json:"stopReason,omitempty"`

	// SuppressOutput hides stdout from the transcript if true.
	SuppressOutput bool `json:"suppressOutput,omitempty"`

	// SystemMessage is a message injected into the conversation for Claude to see.
	SystemMessage string `json:"systemMessage,omitempty"`

	// HookSpecificOutput contains hook-specific output data.
	HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// HookSpecificOutput contains hook-specific output data nested inside HookOutput.
type HookSpecificOutput struct {
	// HookEventName identifies which hook type the output is for.
	HookEventName HookEventName `json:"hookEventName"`

	// PermissionDecision controls whether the tool executes (PreToolUse).
	// Valid values: "allow", "deny", "ask"
	PermissionDecision PermissionDecision `json:"permissionDecision,omitempty"`

	// PermissionDecisionReason provides an explanation for the decision.
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`

	// UpdatedInput contains modified tool input (requires PermissionDecision "allow").
	UpdatedInput json.RawMessage `json:"updatedInput,omitempty"`

	// AdditionalContext is context added to the conversation.
	// Available for PreToolUse, PostToolUse, UserPromptSubmit, SessionStart, SubagentStart.
	AdditionalContext string `json:"additionalContext,omitempty"`
}

// EmptyOutput creates an empty HookOutput that allows the operation without changes.
func EmptyOutput() HookOutput {
	return HookOutput{}
}

// Allow creates a HookOutput that allows the operation.
func Allow() HookOutput {
	return HookOutput{}
}

// AllowWithEvent creates a HookOutput that explicitly allows the operation for a specific hook event.
func AllowWithEvent(event HookEventName) HookOutput {
	return HookOutput{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:      event,
			PermissionDecision: PermissionAllow,
		},
	}
}

// AllowWithReason creates a HookOutput that allows with an explicit reason.
func AllowWithReason(event HookEventName, reason string) HookOutput {
	return HookOutput{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:           event,
			PermissionDecision:      PermissionAllow,
			PermissionDecisionReason: reason,
		},
	}
}

// Deny creates a HookOutput that denies the operation.
func Deny(event HookEventName, reason string) HookOutput {
	return HookOutput{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:           event,
			PermissionDecision:      PermissionDeny,
			PermissionDecisionReason: reason,
		},
	}
}

// Ask creates a HookOutput that prompts the user for confirmation.
func Ask(event HookEventName) HookOutput {
	return HookOutput{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:      event,
			PermissionDecision: PermissionAsk,
		},
	}
}

// Stop creates a HookOutput that stops the agent.
func Stop(reason string) HookOutput {
	f := false
	return HookOutput{
		Continue:   &f,
		StopReason: reason,
	}
}

// AllowWithSystemMessage creates a HookOutput that allows with a system message for Claude.
func AllowWithSystemMessage(message string) HookOutput {
	return HookOutput{
		SystemMessage: message,
	}
}

// AllowWithUpdatedInput creates a HookOutput that allows with modified input.
func AllowWithUpdatedInput(event HookEventName, input any) (HookOutput, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return HookOutput{}, err
	}
	return HookOutput{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:      event,
			PermissionDecision: PermissionAllow,
			UpdatedInput:       data,
		},
	}, nil
}

// MustAllowWithUpdatedInput creates a HookOutput that allows with modified input.
// Panics if marshaling fails.
func MustAllowWithUpdatedInput(event HookEventName, input any) HookOutput {
	result, err := AllowWithUpdatedInput(event, input)
	if err != nil {
		panic(err)
	}
	return result
}

// WithSystemMessage adds a system message to the HookOutput.
func (h HookOutput) WithSystemMessage(message string) HookOutput {
	h.SystemMessage = message
	return h
}

// WithSuppressOutput suppresses the tool output.
func (h HookOutput) WithSuppressOutput() HookOutput {
	h.SuppressOutput = true
	return h
}

// WithAdditionalContext adds additional context to the hook-specific output.
func (h HookOutput) WithAdditionalContext(context string) HookOutput {
	if h.HookSpecificOutput == nil {
		h.HookSpecificOutput = &HookSpecificOutput{}
	}
	h.HookSpecificOutput.AdditionalContext = context
	return h
}

// WithUpdatedInput sets the updated input on the hook-specific output.
func (h HookOutput) WithUpdatedInput(input json.RawMessage) HookOutput {
	if h.HookSpecificOutput == nil {
		h.HookSpecificOutput = &HookSpecificOutput{}
	}
	h.HookSpecificOutput.UpdatedInput = input
	return h
}
