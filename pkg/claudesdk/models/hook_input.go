package models

import "encoding/json"

// PreToolUseHookInput represents input for the pre-tool-use hook.
// ToolInput is kept as json.RawMessage because its schema depends on ToolName.
// Use ParseToolInput to decode it into a concrete type.
type PreToolUseHookInput struct {
	SessionID      string          `json:"session_id"`
	TranscriptPath string          `json:"transcript_path"`
	Cwd            string          `json:"cwd"`
	PermissionMode *string         `json:"permission_mode,omitempty"`
	ToolName       string          `json:"tool_name"`
	ToolInput      json.RawMessage `json:"tool_input"`
	HookEventName  string          `json:"hook_event_name"`
	ToolUseID      string          `json:"tool_use_id"`
}

// PostToolUseHookInput represents input for the post-tool-use hook.
type PostToolUseHookInput struct {
	SessionID      string          `json:"session_id"`
	TranscriptPath string          `json:"transcript_path"`
	Cwd            string          `json:"cwd"`
	PermissionMode *string         `json:"permission_mode,omitempty"`
	ToolName       string          `json:"tool_name"`
	ToolInput      json.RawMessage `json:"tool_input"`
	ToolResponse   json.RawMessage `json:"tool_response"`
	HookEventName  string          `json:"hook_event_name"`
	ToolUseID      string          `json:"tool_use_id"`
}

// PostToolUseFailureHookInput represents input for the post-tool-use-failure hook.
type PostToolUseFailureHookInput struct {
	SessionID      string          `json:"session_id"`
	TranscriptPath string          `json:"transcript_path"`
	Cwd            string          `json:"cwd"`
	PermissionMode *string         `json:"permission_mode,omitempty"`
	ToolName       string          `json:"tool_name"`
	ToolInput      json.RawMessage `json:"tool_input"`
	Error          string          `json:"error"`
	IsInterrupt    *bool           `json:"is_interrupt,omitempty"`
	HookEventName  string          `json:"hook_event_name"`
}

// NotificationHookInput represents input for the notification hook.
type NotificationHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
	Message        string  `json:"message"`
	Title          *string `json:"title,omitempty"`
	HookEventName  string  `json:"hook_event_name"`
}

// UserPromptSubmitHookInput represents input for the user-prompt-submit hook.
type UserPromptSubmitHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
	Prompt         string  `json:"prompt"`
	HookEventName  string  `json:"hook_event_name"`
}

// SessionStartHookInput represents input for the session-start hook.
type SessionStartHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
	Source         string  `json:"source"`
	HookEventName  string  `json:"hook_event_name"`
}

// SessionEndHookInput represents input for the session-end hook.
type SessionEndHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
	Reason         string  `json:"reason"`
	HookEventName  string  `json:"hook_event_name"`
}

// StopHookInput represents input for the stop hook.
type StopHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
	StopHookActive bool    `json:"stop_hook_active"`
	HookEventName  string  `json:"hook_event_name"`
}

// SubagentStartHookInput represents input for the subagent-start hook.
type SubagentStartHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
	AgentID        string  `json:"agent_id"`
	AgentType      string  `json:"agent_type"`
	HookEventName  string  `json:"hook_event_name"`
}

// SubagentStopHookInput represents input for the subagent-stop hook.
type SubagentStopHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
	StopHookActive bool    `json:"stop_hook_active"`
	HookEventName  string  `json:"hook_event_name"`
}

// PreCompactHookInput represents input for the pre-compact hook.
type PreCompactHookInput struct {
	SessionID          string  `json:"session_id"`
	TranscriptPath     string  `json:"transcript_path"`
	Cwd                string  `json:"cwd"`
	PermissionMode     *string `json:"permission_mode,omitempty"`
	Trigger            string  `json:"trigger"`
	CustomInstructions *string `json:"custom_instructions,omitempty"`
	HookEventName      string  `json:"hook_event_name"`
}

// PermissionRequestHookInput represents input for the permission-request hook.
type PermissionRequestHookInput struct {
	SessionID             string          `json:"session_id"`
	TranscriptPath        string          `json:"transcript_path"`
	Cwd                   string          `json:"cwd"`
	PermissionMode        *string         `json:"permission_mode,omitempty"`
	ToolName              string          `json:"tool_name"`
	ToolInput             json.RawMessage `json:"tool_input"`
	PermissionSuggestions json.RawMessage `json:"permission_suggestions,omitempty"`
	HookEventName         string          `json:"hook_event_name"`
}
