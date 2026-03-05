package models

import (
	"encoding/json"
	"fmt"
	"maps"
)

// As per https://platform.claude.com/docs/en/agent-sdk/typescript#hook-input
/*
type HookInput =
  | PreToolUseHookInput
  | PostToolUseHookInput
  | PostToolUseFailureHookInput
  | NotificationHookInput
  | UserPromptSubmitHookInput
  | SessionStartHookInput
  | SessionEndHookInput
  | StopHookInput
  | SubagentStartHookInput
  | SubagentStopHookInput
  | PreCompactHookInput
  | PermissionRequestHookInput
  | SetupHookInput
  | TeammateIdleHookInput
  | TaskCompletedHookInput
  | ConfigChangeHookInput
  | WorktreeCreateHookInput
  | WorktreeRemoveHookInput;
*/

type HookInput interface {
	hookInput()
}

// fallback target
func (UnknownHookInput) hookInput() {}

func (PreToolUseHookInput) hookInput()         {}
func (PostToolUseHookInput) hookInput()        {}
func (PostToolUseFailureHookInput) hookInput() {}
func (NotificationHookInput) hookInput()       {}
func (UserPromptSubmitHookInput) hookInput()   {}
func (SessionStartHookInput) hookInput()       {}
func (SessionEndHookInput) hookInput()         {}
func (StopHookInput) hookInput()               {}
func (SubagentStartHookInput) hookInput()      {}
func (SubagentStopHookInput) hookInput()       {}
func (PreCompactHookInput) hookInput()         {}
func (PermissionRequestHookInput) hookInput()  {}
func (SetupHookInput) hookInput()              {}
func (TeammateIdleHookInput) hookInput()       {}
func (TaskCompletedHookInput) hookInput()      {}
func (ConfigChangeHookInput) hookInput()       {}
func (WorktreeCreateHookInput) hookInput()     {}
func (WorktreeRemoveHookInput) hookInput()     {}

func unmarshalHookInput(data []byte) (_ HookInput, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("HookInput: %w", err)
		}
	}()

	type hookEventName struct {
		HookEventName string `json:"hook_event_name"`
	}

	var h hookEventName
	err = json.Unmarshal(data, &h)
	if err != nil {
		return nil, err
	}

	switch h.HookEventName {
	default:
		var v UnknownHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "":
		return nil, fmt.Errorf("empty hook_event_name")
	case "PreToolUse":
		var v PreToolUseHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "PostToolUse":
		var v PostToolUseHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "PostToolUseFailure":
		var v PostToolUseFailureHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "Notification":
		var v NotificationHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "UserPromptSubmit":
		var v UserPromptSubmitHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "SessionStart":
		var v SessionStartHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "SessionEnd":
		var v SessionEndHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "Stop":
		var v StopHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "SubagentStart":
		var v SubagentStartHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "SubagentStop":
		var v SubagentStopHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "PreCompact":
		var v PreCompactHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "PermissionRequest":
		var v PermissionRequestHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "Setup":
		var v SetupHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "TeammateIdle":
		var v TeammateIdleHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "TaskCompleted":
		var v TaskCompletedHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "ConfigChange":
		var v ConfigChangeHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "WorktreeCreate":
		var v WorktreeCreateHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	case "WorktreeRemove":
		var v WorktreeRemoveHookInput
		err := json.Unmarshal(data, &v)
		return v, err
	}
}

// BaseHookInput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#base-hook-input
type BaseHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
}

type PreToolUseHookInput struct {
	BaseHookInput
	HookEventName string           `json:"hook_event_name"`
	ToolName      string           `json:"tool_name"`
	ToolInput     ToolInputSchemas `json:"tool_input"`
	ToolUseID     string           `json:"tool_use_id"`
}

type PostToolUseHookInput struct {
	BaseHookInput
	HookEventName string           `json:"hook_event_name"`
	ToolName      string           `json:"tool_name"`
	ToolInput     ToolInputSchemas `json:"tool_input"`
	// response from tool
	ToolResponse json.RawMessage `json:"tool_response"`
	ToolUseID    string          `json:"tool_use_id"`
}

type PostToolUseFailureHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"`
	ToolName      string `json:"tool_name"`
	ToolInput     any    `json:"tool_input"`
	ToolUseID     string `json:"tool_use_id"`
	Error         string `json:"error"`
	IsInterrupt   *bool  `json:"is_interrupt,omitempty"`
}

type NotificationHookInput struct {
	BaseHookInput
	HookEventName    string  `json:"hook_event_name"`
	Message          string  `json:"message"`
	Title            *string `json:"title,omitempty"`
	NotificationType string  `json:"notification_type"`
}

type UserPromptSubmitHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"`
	Prompt        string `json:"prompt"`
}

type SessionStartHookInput struct {
	BaseHookInput
	HookEventName string  `json:"hook_event_name"`
	Source        string  `json:"source"`
	AgentType     *string `json:"agent_type,omitempty"`
	Model         *string `json:"model,omitempty"`
}

type ExitReason string

type SessionEndHookInput struct {
	BaseHookInput
	HookEventName string     `json:"hook_event_name"`
	Reason        ExitReason `json:"reason"`
}

type StopHookInput struct {
	BaseHookInput
	HookEventName        string  `json:"hook_event_name"`
	StopHookActive       bool    `json:"stop_hook_active"`
	LastAssistantMessage *string `json:"last_assistant_message,omitempty"`
}

type SubagentStartHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"`
	AgentID       string `json:"agent_id"`
	AgentType     string `json:"agent_type"`
}

type SubagentStopHookInput struct {
	BaseHookInput
	HookEventName        string  `json:"hook_event_name"`
	StopHookActive       bool    `json:"stop_hook_active"`
	AgentID              string  `json:"agent_id"`
	AgentTranscriptPath  string  `json:"agent_transcript_path"`
	AgentType            string  `json:"agent_type"`
	LastAssistantMessage *string `json:"last_assistant_message,omitempty"`
}

type PreCompactHookInput struct {
	BaseHookInput
	HookEventName      string  `json:"hook_event_name"`
	Trigger            string  `json:"trigger"`
	CustomInstructions *string `json:"custom_instructions"`
}

type PermissionRequestHookInput struct {
	BaseHookInput
	HookEventName         string             `json:"hook_event_name"`
	ToolName              string             `json:"tool_name"`
	ToolInput             any                `json:"tool_input"`
	PermissionSuggestions []PermissionUpdate `json:"permission_suggestions,omitempty"`
}

type SetupHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"`
	Trigger       string `json:"trigger"`
}

type TeammateIdleHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"`
	TeammateName  string `json:"teammate_name"`
	TeamName      string `json:"team_name"`
}

type TaskCompletedHookInput struct {
	BaseHookInput
	HookEventName   string  `json:"hook_event_name"`
	TaskID          string  `json:"task_id"`
	TaskSubject     string  `json:"task_subject"`
	TaskDescription *string `json:"task_description,omitempty"`
	TeammateName    *string `json:"teammate_name,omitempty"`
	TeamName        *string `json:"team_name,omitempty"`
}

type ConfigChangeHookInput struct {
	BaseHookInput
	HookEventName string  `json:"hook_event_name"`
	Source        string  `json:"source"`
	FilePath      *string `json:"file_path,omitempty"`
}

type WorktreeCreateHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"`
	Name          string `json:"name"`
}

type WorktreeRemoveHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"`
	WorktreePath  string `json:"worktree_path"`
}

type UnknownHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"`
	Unknown       map[string]json.RawMessage
}

func (o UnknownHookInput) MarshalJSON() ([]byte, error) {
	m := maps.Clone(o.Unknown)
	if m == nil {
		// panic instead?
		m = map[string]json.RawMessage{}
	}

	marshal := func(v any) json.RawMessage {
		raw, _ := json.Marshal(v)
		return json.RawMessage(raw)
	}

	m["session_id"] = marshal(o.SessionID)
	m["transcript_path"] = marshal(o.TranscriptPath)
	m["cwd"] = marshal(o.Cwd)
	if o.PermissionMode != nil {
		m["permission_mode"] = marshal(o.PermissionMode)
	}
	m["tool_use_id"] = marshal(o.ToolUseID)

	m["hook_event_name"] = marshal(o.HookEventName)

	return json.Marshal(m)
}

func (o *UnknownHookInput) UnmarshalJSON(data []byte) error {
	unknown := map[string]json.RawMessage{}
	err := json.Unmarshal(data, &unknown)
	if err != nil {
		return fmt.Errorf("HookSpecificOutputUnknown: %w", err)
	}

	type pair struct {
		k   string
		tgt any
	}
	for _, kv := range []pair{
		{"session_id", &o.SessionID},
		{"transcript_path", &o.TranscriptPath},
		{"cwd", &o.Cwd},
		{"permission_mode", &o.PermissionMode},
		{"tool_use_id", &o.ToolUseID},
		{"hook_event_name", &o.HookEventName},
	} {
		if _, ok := unknown[kv.k]; !ok {
			continue
		}
		if err := json.Unmarshal(unknown[kv.k], kv.tgt); err != nil {
			return fmt.Errorf("HookSpecificOutputUnknown: %w", err)
		}
	}
	if o.HookEventName == "" {
		return fmt.Errorf("HookSpecificOutputUnknown: empty hookEventName")
	}
	o.Unknown = unknown
	return nil
}
