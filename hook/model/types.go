// Package model defines the Claude Code hook I/O types.
package model

import "encoding/json"

// HookInput represents the JSON input received from Claude Code via stdin.
// This is the common structure for all hook types.
type HookInput struct {
	// HookName is the name of the hook being invoked (e.g., "PreToolUse", "PostToolUse").
	HookName string `json:"hook_name"`
	// SessionID is the unique identifier for the Claude Code session.
	SessionID string `json:"session_id"`
	// MessageID is the unique identifier for the current message.
	MessageID string `json:"message_id"`
	// ToolName is the name of the tool being used (only for tool-related hooks).
	ToolName string `json:"tool_name,omitempty"`
	// ToolInput contains the tool's input parameters as raw JSON.
	ToolInput json.RawMessage `json:"tool_input,omitempty"`
}

// HookOutput represents the JSON output sent to Claude Code via stdout.
type HookOutput struct {
	// Decision determines how Claude Code should proceed.
	// Valid values: "allow", "block", "allow_always"
	Decision string `json:"decision"`
	// Reason provides an optional explanation for the decision.
	// This is shown to Claude when the tool is blocked.
	Reason string `json:"reason,omitempty"`
}

// Decision constants for HookOutput.
const (
	// DecisionAllow allows the tool to proceed.
	DecisionAllow = "allow"
	// DecisionBlock prevents the tool from executing.
	DecisionBlock = "block"
	// DecisionAllowAlways allows this and similar future requests automatically.
	DecisionAllowAlways = "allow_always"
)
