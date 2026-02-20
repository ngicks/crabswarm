package model

// PreToolUseHookInput represents input for the pre-tool-use hook.
type PreToolUseHookInput struct {
	SessionID      string     `json:"session_id"`
	TranscriptPath string     `json:"transcript_path"`
	Cwd            string     `json:"cwd"`
	PermissionMode *string    `json:"permission_mode,omitempty"`
	ToolName       string     `json:"tool_name"`
	ToolInput      *ToolInput `json:"tool_input,omitempty"`
}

// PostToolUseHookInput represents input for the post-tool-use hook.
type PostToolUseHookInput struct {
	SessionID      string      `json:"session_id"`
	TranscriptPath string      `json:"transcript_path"`
	Cwd            string      `json:"cwd"`
	PermissionMode *string     `json:"permission_mode,omitempty"`
	ToolName       string      `json:"tool_name"`
	ToolInput      *ToolInput  `json:"tool_input,omitempty"`
	ToolResponse   *ToolOutput `json:"tool_response,omitempty"`
}

// PostToolUseFailureHookInput represents input for the post-tool-use-failure hook.
type PostToolUseFailureHookInput struct {
	SessionID      string     `json:"session_id"`
	TranscriptPath string     `json:"transcript_path"`
	Cwd            string     `json:"cwd"`
	PermissionMode *string    `json:"permission_mode,omitempty"`
	ToolName       string     `json:"tool_name"`
	ToolInput      *ToolInput `json:"tool_input,omitempty"`
	Error          string     `json:"error"`
	IsInterrupt    *bool      `json:"is_interrupt,omitempty"`
}

// NotificationHookInput represents input for the notification hook.
type NotificationHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
	Message        string  `json:"message"`
	Title          *string `json:"title,omitempty"`
}

// UserPromptSubmitHookInput represents input for the user-prompt-submit hook.
type UserPromptSubmitHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
	Prompt         string  `json:"prompt"`
}

// SessionStartHookInput represents input for the session-start hook.
type SessionStartHookInput struct {
	SessionID      string             `json:"session_id"`
	TranscriptPath string             `json:"transcript_path"`
	Cwd            string             `json:"cwd"`
	PermissionMode *string            `json:"permission_mode,omitempty"`
	Source         string             `json:"source"`
}

// SessionEndHookInput represents input for the session-end hook.
type SessionEndHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
	Reason         string  `json:"reason"`
}

// StopHookInput represents input for the stop hook.
type StopHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
	StopHookActive bool    `json:"stop_hook_active"`
}

// SubagentStartHookInput represents input for the subagent-start hook.
type SubagentStartHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
	AgentID        string  `json:"agent_id"`
	AgentType      string  `json:"agent_type"`
}

// SubagentStopHookInput represents input for the subagent-stop hook.
type SubagentStopHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
	StopHookActive bool    `json:"stop_hook_active"`
}

// PreCompactHookInput represents input for the pre-compact hook.
type PreCompactHookInput struct {
	SessionID          string            `json:"session_id"`
	TranscriptPath     string            `json:"transcript_path"`
	Cwd                string            `json:"cwd"`
	PermissionMode     *string           `json:"permission_mode,omitempty"`
	Trigger            string            `json:"trigger"`
	CustomInstructions *string           `json:"custom_instructions,omitempty"`
}

// PermissionRequestHookInput represents input for the permission-request hook.
type PermissionRequestHookInput struct {
	SessionID             string              `json:"session_id"`
	TranscriptPath        string              `json:"transcript_path"`
	Cwd                   string              `json:"cwd"`
	PermissionMode        *string             `json:"permission_mode,omitempty"`
	ToolName              string              `json:"tool_name"`
	ToolInput             *ToolInput          `json:"tool_input,omitempty"`
	PermissionSuggestions []*PermissionUpdate `json:"permission_suggestions,omitempty"`
}

// HookInput is a union type for all hook inputs.
type HookInput struct {
	PreToolUse         *PreToolUseHookInput         `json:"pre_tool_use,omitempty"`
	PostToolUse        *PostToolUseHookInput        `json:"post_tool_use,omitempty"`
	PostToolUseFailure *PostToolUseFailureHookInput `json:"post_tool_use_failure,omitempty"`
	Notification       *NotificationHookInput       `json:"notification,omitempty"`
	UserPromptSubmit   *UserPromptSubmitHookInput   `json:"user_prompt_submit,omitempty"`
	SessionStart       *SessionStartHookInput       `json:"session_start,omitempty"`
	SessionEnd         *SessionEndHookInput         `json:"session_end,omitempty"`
	Stop               *StopHookInput               `json:"stop,omitempty"`
	SubagentStart      *SubagentStartHookInput      `json:"subagent_start,omitempty"`
	SubagentStop       *SubagentStopHookInput       `json:"subagent_stop,omitempty"`
	PreCompact         *PreCompactHookInput         `json:"pre_compact,omitempty"`
	PermissionRequest  *PermissionRequestHookInput  `json:"permission_request,omitempty"`
}

// AsyncHookJSONOutput represents output for async hooks.
type AsyncHookJSONOutput struct {
	AsyncTimeout *int32 `json:"async_timeout,omitempty"`
}

// PreToolUseHookSpecificOutput represents hook-specific output for pre-tool-use.
type PreToolUseHookSpecificOutput struct {
	PermissionDecision       *string    `json:"permission_decision,omitempty"`
	PermissionDecisionReason *string    `json:"permission_decision_reason,omitempty"`
	UpdatedInput             *ToolInput `json:"updated_input,omitempty"`
}

// UserPromptSubmitHookSpecificOutput represents hook-specific output for user-prompt-submit.
type UserPromptSubmitHookSpecificOutput struct {
	AdditionalContext *string `json:"additional_context,omitempty"`
}

// SessionStartHookSpecificOutput represents hook-specific output for session-start.
type SessionStartHookSpecificOutput struct {
	AdditionalContext *string `json:"additional_context,omitempty"`
}

// PostToolUseHookSpecificOutput represents hook-specific output for post-tool-use.
type PostToolUseHookSpecificOutput struct {
	AdditionalContext *string `json:"additional_context,omitempty"`
}

// HookSpecificOutput is a union type for hook-specific outputs.
type HookSpecificOutput struct {
	PreToolUse       *PreToolUseHookSpecificOutput       `json:"pre_tool_use,omitempty"`
	UserPromptSubmit *UserPromptSubmitHookSpecificOutput `json:"user_prompt_submit,omitempty"`
	SessionStart     *SessionStartHookSpecificOutput     `json:"session_start,omitempty"`
	PostToolUse      *PostToolUseHookSpecificOutput      `json:"post_tool_use,omitempty"`
}

// SyncHookJSONOutput represents output for sync hooks.
type SyncHookJSONOutput struct {
	Continue           *bool               `json:"continue,omitempty"`
	SuppressOutput     *bool               `json:"suppress_output,omitempty"`
	StopReason         *string             `json:"stop_reason,omitempty"`
	Decision           *string             `json:"decision,omitempty"`
	SystemMessage      *string             `json:"system_message,omitempty"`
	Reason             *string             `json:"reason,omitempty"`
	HookSpecificOutput *HookSpecificOutput `json:"hook_specific_output,omitempty"`
}

// HookJSONOutput is a union type for hook outputs.
type HookJSONOutput struct {
	AsyncOutput *AsyncHookJSONOutput `json:"async_output,omitempty"`
	SyncOutput  *SyncHookJSONOutput  `json:"sync_output,omitempty"`
}
