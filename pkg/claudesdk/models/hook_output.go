package models

import "encoding/json"

// AsyncHookJSONOutput represents the output of an async hook.
type AsyncHookJSONOutput struct {
	AsyncTimeout *int32 `json:"async_timeout,omitempty"`
}

// PreToolUseHookSpecificOutput represents hook-specific output for pre-tool-use hooks.
// UpdatedInput is json.RawMessage because it has the same oneof issue as ToolInput.
type PreToolUseHookSpecificOutput struct {
	PermissionDecision       *string         `json:"permissionDecision,omitempty"`
	PermissionDecisionReason *string         `json:"permissionDecisionReason,omitempty"`
	UpdatedInput             json.RawMessage `json:"updatedInput,omitempty"`
}

// UserPromptSubmitHookSpecificOutput represents hook-specific output for user-prompt-submit hooks.
type UserPromptSubmitHookSpecificOutput struct {
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// SessionStartHookSpecificOutput represents hook-specific output for session-start hooks.
type SessionStartHookSpecificOutput struct {
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// PostToolUseHookSpecificOutput represents hook-specific output for post-tool-use hooks.
type PostToolUseHookSpecificOutput struct {
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// PermissionRequestDecision represents the decision object for PermissionRequest hooks.
// NOTE: This type does not have a corresponding definition in the TypeScript Agent SDK type definitions.
// It is derived from the Claude Code hooks reference:
// https://code.claude.com/docs/en/hooks#permissionrequest-decision-control
type PermissionRequestDecision struct {
	Behavior           string          `json:"behavior"`                     // "allow" or "deny"
	UpdatedInput       json.RawMessage `json:"updatedInput,omitempty"`       // for "allow": modifies tool input
	UpdatedPermissions json.RawMessage `json:"updatedPermissions,omitempty"` // for "allow": applies permission rules
	Message            *string         `json:"message,omitempty"`            // for "deny": tells Claude why
	Interrupt          *bool           `json:"interrupt,omitempty"`          // for "deny": if true, stops Claude
}

// HookSpecificOutput is a flattened representation of all hook-specific output fields.
// In protobuf this is a oneof; here all fields coexist and only the relevant ones
// are populated based on the hook type.
type HookSpecificOutput struct {
	// hookEventName discriminator (required for PermissionRequest, PreToolUse, etc.)
	HookEventName *string `json:"hookEventName,omitempty"`
	// PreToolUse fields
	PermissionDecision       *string         `json:"permissionDecision,omitempty"`
	PermissionDecisionReason *string         `json:"permissionDecisionReason,omitempty"`
	UpdatedInput             json.RawMessage `json:"updatedInput,omitempty"`
	// Shared field for UserPromptSubmit, SessionStart, PostToolUse
	AdditionalContext *string `json:"additionalContext,omitempty"`
	// PermissionRequest fields
	Decision *PermissionRequestDecision `json:"decision,omitempty"`
}

// SyncHookJSONOutput represents the output of a synchronous hook.
type SyncHookJSONOutput struct {
	Continue           *bool               `json:"continue,omitempty"`
	SuppressOutput     *bool               `json:"suppressOutput,omitempty"`
	StopReason         *string             `json:"stopReason,omitempty"`
	Decision           *string             `json:"decision,omitempty"`
	SystemMessage      *string             `json:"systemMessage,omitempty"`
	Reason             *string             `json:"reason,omitempty"`
	HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}
