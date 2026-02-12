// Package model defines the Claude Code hook I/O types.
package model

import (
	"encoding/json"
	"fmt"
)

// HookInput represents the JSON input received from Claude Code via stdin.
// This is the common structure for all hook types.
// See https://platform.claude.com/docs/en/agent-sdk/hooks for the official protocol.
type HookInput struct {
	// HookEventName is the name of the hook being invoked (e.g., "PreToolUse", "PostToolUse").
	HookEventName HookEventName `json:"hook_event_name"`

	// SessionID is the unique identifier for the Claude Code session.
	SessionID string `json:"session_id"`

	// TranscriptPath is the path to the conversation transcript.
	TranscriptPath string `json:"transcript_path,omitempty"`

	// Cwd is the current working directory.
	Cwd string `json:"cwd,omitempty"`

	// MessageID is the unique identifier for the current message.
	MessageID string `json:"message_id,omitempty"`

	// ToolName is the name of the tool being used (only for tool-related hooks).
	ToolName ToolName `json:"tool_name,omitempty"`

	// ToolInput contains the tool's input parameters as raw JSON.
	ToolInput json.RawMessage `json:"tool_input,omitempty"`

	// ToolResponse contains the tool's response (only for PostToolUse).
	ToolResponse json.RawMessage `json:"tool_response,omitempty"`

	// Error contains the error message (only for PostToolUseFailure).
	Error string `json:"error,omitempty"`

	// IsInterrupt indicates whether the failure was caused by an interrupt (PostToolUseFailure).
	IsInterrupt bool `json:"is_interrupt,omitempty"`

	// Source is the reason for SessionStart (startup, resume, clear, compact).
	Source SessionStartReason `json:"source,omitempty"`

	// Reason is the reason for SessionEnd (clear, logout, etc.).
	Reason SessionEndReason `json:"reason,omitempty"`

	// NotificationType is the type of notification (for Notification hook).
	NotificationType NotificationType `json:"notification_type,omitempty"`

	// Message is the status message from the agent (for Notification hook).
	Message string `json:"message,omitempty"`

	// Title is an optional title set by the agent (for Notification hook).
	Title string `json:"title,omitempty"`

	// Trigger indicates what triggered compaction (for PreCompact hook).
	Trigger CompactTrigger `json:"trigger,omitempty"`

	// CustomInstructions contains custom instructions provided for compaction (for PreCompact hook).
	CustomInstructions string `json:"custom_instructions,omitempty"`

	// AgentType is the type of subagent (for SubagentStart/SubagentStop).
	AgentType string `json:"agent_type,omitempty"`

	// AgentID is the ID of the subagent (for SubagentStart/SubagentStop).
	AgentID string `json:"agent_id,omitempty"`

	// AgentTranscriptPath is the path to the subagent's conversation transcript (for SubagentStop).
	AgentTranscriptPath string `json:"agent_transcript_path,omitempty"`

	// StopHookActive indicates whether a stop hook is currently processing (for Stop/SubagentStop).
	StopHookActive bool `json:"stop_hook_active,omitempty"`

	// Prompt is the user's prompt text (for UserPromptSubmit).
	Prompt string `json:"prompt,omitempty"`

	// PermissionSuggestions contains suggested permission updates (for PermissionRequest).
	PermissionSuggestions json.RawMessage `json:"permission_suggestions,omitempty"`

	// StopReason indicates why Claude stopped (for Stop hook).
	StopReason string `json:"stop_reason,omitempty"`
}

// Category returns the category of the tool for this hook input.
// Returns CategoryOther if no tool is associated with this hook.
func (h *HookInput) Category() ToolCategory {
	if h.ToolName == "" {
		return CategoryOther
	}
	return h.ToolName.Category()
}

// ParseToolInput parses the raw ToolInput JSON into a typed struct.
// Returns nil if ToolInput is empty or parsing fails.
// The returned type depends on the ToolName.
func (h *HookInput) ParseToolInput() (any, error) {
	if len(h.ToolInput) == 0 {
		return nil, nil
	}

	var target any

	switch h.ToolName {
	case ToolNameBash:
		target = &BashInput{}
	case ToolNameRead:
		target = &ReadInput{}
	case ToolNameWrite:
		target = &WriteInput{}
	case ToolNameEdit:
		target = &EditInput{}
	case ToolNameGlob:
		target = &GlobInput{}
	case ToolNameGrep:
		target = &GrepInput{}
	case ToolNameWebFetch:
		target = &WebFetchInput{}
	case ToolNameWebSearch:
		target = &WebSearchInput{}
	case ToolNameTask:
		target = &TaskInput{}
	case ToolNameTaskOutput:
		target = &TaskOutputInput{}
	case ToolNameTaskStop:
		target = &TaskStopInput{}
	case ToolNameAskUserQuestion:
		target = &AskUserQuestionInput{}
	case ToolNameNotebookEdit:
		target = &NotebookEditInput{}
	case ToolNameSkill:
		target = &SkillInput{}
	case ToolNameTaskCreate:
		target = &TaskCreateInput{}
	case ToolNameTaskUpdate:
		target = &TaskUpdateInput{}
	case ToolNameTaskGet:
		target = &TaskGetInput{}
	case ToolNameTaskList:
		target = &TaskListInput{}
	case ToolNameEnterPlanMode:
		target = &EnterPlanModeInput{}
	case ToolNameExitPlanMode:
		target = &ExitPlanModeInput{}
	default:
		// Check if it's an MCP tool
		if h.ToolName.IsMCP() {
			mcpInfo := h.ToolName.ParseMCP()
			var params map[string]any
			if err := json.Unmarshal(h.ToolInput, &params); err != nil {
				return nil, fmt.Errorf("failed to parse MCP tool input: %w", err)
			}
			return &MCPToolInput{
				Server:     mcpInfo.Server,
				Tool:       mcpInfo.Tool,
				Parameters: params,
			}, nil
		}
		// Unknown tool - return raw map
		var raw map[string]any
		if err := json.Unmarshal(h.ToolInput, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse unknown tool input: %w", err)
		}
		return raw, nil
	}

	if err := json.Unmarshal(h.ToolInput, target); err != nil {
		return nil, fmt.Errorf("failed to parse %s input: %w", h.ToolName, err)
	}
	return target, nil
}

// MustParseToolInput parses the raw ToolInput JSON into a typed struct.
// Panics if parsing fails.
func (h *HookInput) MustParseToolInput() any {
	result, err := h.ParseToolInput()
	if err != nil {
		panic(err)
	}
	return result
}

// ParseBashInput parses the ToolInput as BashInput.
// Returns nil and error if the tool is not Bash or parsing fails.
func (h *HookInput) ParseBashInput() (*BashInput, error) {
	if h.ToolName != ToolNameBash {
		return nil, fmt.Errorf("tool is %s, not Bash", h.ToolName)
	}
	result, err := h.ParseToolInput()
	if err != nil {
		return nil, err
	}
	return result.(*BashInput), nil
}

// ParseReadInput parses the ToolInput as ReadInput.
func (h *HookInput) ParseReadInput() (*ReadInput, error) {
	if h.ToolName != ToolNameRead {
		return nil, fmt.Errorf("tool is %s, not Read", h.ToolName)
	}
	result, err := h.ParseToolInput()
	if err != nil {
		return nil, err
	}
	return result.(*ReadInput), nil
}

// ParseWriteInput parses the ToolInput as WriteInput.
func (h *HookInput) ParseWriteInput() (*WriteInput, error) {
	if h.ToolName != ToolNameWrite {
		return nil, fmt.Errorf("tool is %s, not Write", h.ToolName)
	}
	result, err := h.ParseToolInput()
	if err != nil {
		return nil, err
	}
	return result.(*WriteInput), nil
}

// ParseEditInput parses the ToolInput as EditInput.
func (h *HookInput) ParseEditInput() (*EditInput, error) {
	if h.ToolName != ToolNameEdit {
		return nil, fmt.Errorf("tool is %s, not Edit", h.ToolName)
	}
	result, err := h.ParseToolInput()
	if err != nil {
		return nil, err
	}
	return result.(*EditInput), nil
}

// ParseGlobInput parses the ToolInput as GlobInput.
func (h *HookInput) ParseGlobInput() (*GlobInput, error) {
	if h.ToolName != ToolNameGlob {
		return nil, fmt.Errorf("tool is %s, not Glob", h.ToolName)
	}
	result, err := h.ParseToolInput()
	if err != nil {
		return nil, err
	}
	return result.(*GlobInput), nil
}

// ParseGrepInput parses the ToolInput as GrepInput.
func (h *HookInput) ParseGrepInput() (*GrepInput, error) {
	if h.ToolName != ToolNameGrep {
		return nil, fmt.Errorf("tool is %s, not Grep", h.ToolName)
	}
	result, err := h.ParseToolInput()
	if err != nil {
		return nil, err
	}
	return result.(*GrepInput), nil
}

// ParseWebFetchInput parses the ToolInput as WebFetchInput.
func (h *HookInput) ParseWebFetchInput() (*WebFetchInput, error) {
	if h.ToolName != ToolNameWebFetch {
		return nil, fmt.Errorf("tool is %s, not WebFetch", h.ToolName)
	}
	result, err := h.ParseToolInput()
	if err != nil {
		return nil, err
	}
	return result.(*WebFetchInput), nil
}

// ParseWebSearchInput parses the ToolInput as WebSearchInput.
func (h *HookInput) ParseWebSearchInput() (*WebSearchInput, error) {
	if h.ToolName != ToolNameWebSearch {
		return nil, fmt.Errorf("tool is %s, not WebSearch", h.ToolName)
	}
	result, err := h.ParseToolInput()
	if err != nil {
		return nil, err
	}
	return result.(*WebSearchInput), nil
}

// ParseTaskInput parses the ToolInput as TaskInput.
func (h *HookInput) ParseTaskInput() (*TaskInput, error) {
	if h.ToolName != ToolNameTask {
		return nil, fmt.Errorf("tool is %s, not Task", h.ToolName)
	}
	result, err := h.ParseToolInput()
	if err != nil {
		return nil, err
	}
	return result.(*TaskInput), nil
}

// ParseAskUserQuestionInput parses the ToolInput as AskUserQuestionInput.
func (h *HookInput) ParseAskUserQuestionInput() (*AskUserQuestionInput, error) {
	if h.ToolName != ToolNameAskUserQuestion {
		return nil, fmt.Errorf("tool is %s, not AskUserQuestion", h.ToolName)
	}
	result, err := h.ParseToolInput()
	if err != nil {
		return nil, err
	}
	return result.(*AskUserQuestionInput), nil
}

// ParseMCPInput parses the ToolInput as MCPToolInput.
// Returns error if this is not an MCP tool.
func (h *HookInput) ParseMCPInput() (*MCPToolInput, error) {
	if !h.ToolName.IsMCP() {
		return nil, fmt.Errorf("tool %s is not an MCP tool", h.ToolName)
	}
	result, err := h.ParseToolInput()
	if err != nil {
		return nil, err
	}
	return result.(*MCPToolInput), nil
}
