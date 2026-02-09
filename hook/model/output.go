package model

import "encoding/json"

// Decision represents the hook decision type.
type Decision string

// Decision constants for HookOutput.
const (
	// DecisionAllow allows the tool to proceed.
	DecisionAllow Decision = "allow"

	// DecisionBlock prevents the tool from executing.
	DecisionBlock Decision = "block"

	// DecisionAllowAlways allows this and similar future requests automatically.
	DecisionAllowAlways Decision = "allow_always"
)

// HookOutput represents the JSON output sent to Claude Code via stdout.
type HookOutput struct {
	// Decision determines how Claude Code should proceed.
	// Valid values: "allow", "block", "allow_always"
	Decision Decision `json:"decision"`

	// Reason provides an optional explanation for the decision.
	// This is shown to Claude when the tool is blocked.
	Reason string `json:"reason,omitempty"`

	// OutputToModel contains optional content to add to the model's context.
	OutputToModel string `json:"output_to_model,omitempty"`

	// SuppressOutput prevents the tool output from being shown to the model if true.
	SuppressOutput bool `json:"suppress_output,omitempty"`

	// ModifiedInput can replace the original tool input.
	ModifiedInput json.RawMessage `json:"modified_input,omitempty"`
}

// Allow creates a HookOutput that allows the operation.
func Allow() HookOutput {
	return HookOutput{Decision: DecisionAllow}
}

// AllowWithMessage creates a HookOutput that allows with a message to the model.
func AllowWithMessage(message string) HookOutput {
	return HookOutput{
		Decision:      DecisionAllow,
		OutputToModel: message,
	}
}

// AllowAlways creates a HookOutput that allows this and similar future requests.
func AllowAlways() HookOutput {
	return HookOutput{Decision: DecisionAllowAlways}
}

// Block creates a HookOutput that blocks the operation.
func Block(reason string) HookOutput {
	return HookOutput{
		Decision: DecisionBlock,
		Reason:   reason,
	}
}

// BlockWithMessage creates a HookOutput that blocks with a message to the model.
func BlockWithMessage(reason, message string) HookOutput {
	return HookOutput{
		Decision:      DecisionBlock,
		Reason:        reason,
		OutputToModel: message,
	}
}

// AllowWithModifiedInput creates a HookOutput that allows with modified input.
func AllowWithModifiedInput(input interface{}) (HookOutput, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return HookOutput{}, err
	}
	return HookOutput{
		Decision:      DecisionAllow,
		ModifiedInput: data,
	}, nil
}

// MustAllowWithModifiedInput creates a HookOutput that allows with modified input.
// Panics if marshaling fails.
func MustAllowWithModifiedInput(input interface{}) HookOutput {
	result, err := AllowWithModifiedInput(input)
	if err != nil {
		panic(err)
	}
	return result
}

// WithReason adds a reason to the HookOutput.
func (h HookOutput) WithReason(reason string) HookOutput {
	h.Reason = reason
	return h
}

// WithOutputToModel adds a message to the model.
func (h HookOutput) WithOutputToModel(message string) HookOutput {
	h.OutputToModel = message
	return h
}

// WithSuppressOutput suppresses the tool output.
func (h HookOutput) WithSuppressOutput() HookOutput {
	h.SuppressOutput = true
	return h
}

// WithModifiedInput sets the modified input.
func (h HookOutput) WithModifiedInput(input json.RawMessage) HookOutput {
	h.ModifiedInput = input
	return h
}

// HookSpecificOutput represents hook-specific output data for non-tool hooks.
type HookSpecificOutput struct {
	// For UserPromptSubmit: can modify the prompt
	ModifiedPrompt string `json:"modified_prompt,omitempty"`

	// For Notification hooks: whether to dismiss the notification
	Dismiss bool `json:"dismiss,omitempty"`

	// For PreCompact: summary text to preserve
	PreserveSummary string `json:"preserve_summary,omitempty"`
}

// UserPromptSubmitOutput creates output for UserPromptSubmit hook.
func UserPromptSubmitOutput(modifiedPrompt string) HookSpecificOutput {
	return HookSpecificOutput{
		ModifiedPrompt: modifiedPrompt,
	}
}

// NotificationOutput creates output for Notification hook.
func NotificationOutput(dismiss bool) HookSpecificOutput {
	return HookSpecificOutput{
		Dismiss: dismiss,
	}
}

// PreCompactOutput creates output for PreCompact hook.
func PreCompactOutput(preserveSummary string) HookSpecificOutput {
	return HookSpecificOutput{
		PreserveSummary: preserveSummary,
	}
}
