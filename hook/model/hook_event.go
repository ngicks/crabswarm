// Package model defines the Claude Code hook I/O types.
package model

// HookEventName represents the type of hook event from Claude Code.
type HookEventName string

// Hook event constants for all 12 Claude Code hook types.
const (
	// HookEventSessionStart fires when a session begins or resumes.
	// Matcher values: "startup", "resume", "clear", "compact"
	HookEventSessionStart HookEventName = "SessionStart"

	// HookEventUserPromptSubmit fires when a user submits a prompt.
	HookEventUserPromptSubmit HookEventName = "UserPromptSubmit"

	// HookEventPreToolUse fires before a tool is executed.
	// Matcher values: tool names (e.g., "Bash", "Read", "Write")
	HookEventPreToolUse HookEventName = "PreToolUse"

	// HookEventPermissionRequest fires when a permission dialog appears.
	// Matcher values: tool names
	HookEventPermissionRequest HookEventName = "PermissionRequest"

	// HookEventPostToolUse fires after a tool executes successfully.
	// Matcher values: tool names
	HookEventPostToolUse HookEventName = "PostToolUse"

	// HookEventPostToolUseFailure fires after a tool fails.
	// Matcher values: tool names
	HookEventPostToolUseFailure HookEventName = "PostToolUseFailure"

	// HookEventNotification fires when a notification is sent.
	// Matcher values: "permission_prompt", "idle_prompt", etc.
	HookEventNotification HookEventName = "Notification"

	// HookEventSubagentStart fires when a subagent is spawned.
	// Matcher values: agent types
	HookEventSubagentStart HookEventName = "SubagentStart"

	// HookEventSubagentStop fires when a subagent finishes.
	// Matcher values: agent types
	HookEventSubagentStop HookEventName = "SubagentStop"

	// HookEventStop fires when Claude finishes responding.
	HookEventStop HookEventName = "Stop"

	// HookEventPreCompact fires before context compaction.
	// Matcher values: "manual", "auto"
	HookEventPreCompact HookEventName = "PreCompact"

	// HookEventSessionEnd fires when a session terminates.
	// Matcher values: "clear", "logout", etc.
	HookEventSessionEnd HookEventName = "SessionEnd"
)

// IsToolRelated returns true if this hook event involves tool execution.
func (h HookEventName) IsToolRelated() bool {
	switch h {
	case HookEventPreToolUse, HookEventPermissionRequest, HookEventPostToolUse, HookEventPostToolUseFailure:
		return true
	default:
		return false
	}
}

// IsSubagentRelated returns true if this hook event involves subagents.
func (h HookEventName) IsSubagentRelated() bool {
	switch h {
	case HookEventSubagentStart, HookEventSubagentStop:
		return true
	default:
		return false
	}
}

// String returns the string representation of the hook event name.
func (h HookEventName) String() string {
	return string(h)
}

// SessionStartReason represents the reason for a SessionStart event.
type SessionStartReason string

const (
	SessionStartReasonStartup SessionStartReason = "startup"
	SessionStartReasonResume  SessionStartReason = "resume"
	SessionStartReasonClear   SessionStartReason = "clear"
	SessionStartReasonCompact SessionStartReason = "compact"
)

// NotificationType represents types of notifications.
type NotificationType string

const (
	NotificationTypePermissionPrompt NotificationType = "permission_prompt"
	NotificationTypeIdlePrompt       NotificationType = "idle_prompt"
)

// CompactTrigger represents what triggered context compaction.
type CompactTrigger string

const (
	CompactTriggerManual CompactTrigger = "manual"
	CompactTriggerAuto   CompactTrigger = "auto"
)

// SessionEndReason represents the reason for session termination.
type SessionEndReason string

const (
	SessionEndReasonClear  SessionEndReason = "clear"
	SessionEndReasonLogout SessionEndReason = "logout"
)
