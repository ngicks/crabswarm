package models

import (
	"encoding/json"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
)

type hookInputBase struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
	HookEventName  string  `json:"hook_event_name"`
	ToolUseID      string  `json:"tool_use_id"`
}

// PreToolUseHookInput represents input for the pre-tool-use hook.
// ToolInput is kept as json.RawMessage because its schema depends on ToolName.
type PreToolUseHookInput struct {
	hookInputBase
	preToolUseHookInputToolInput
}

func (i *PreToolUseHookInput) UnmarshalJSON(data []byte) error {
	var base hookInputBase
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}
	var preToolBase preToolUseHookInputToolInput
	if err := json.Unmarshal(data, &preToolBase); err != nil {
		return err
	}
	i.hookInputBase = base
	i.preToolUseHookInputToolInput = preToolBase
	return nil
}

type preToolUseHookInputToolInput struct {
	ToolName  string `json:"tool_name"`
	ToolInput any    `json:"tool_input"`
}

func (i *preToolUseHookInputToolInput) UnmarshalJSON(data []byte) error {
	type toolName struct {
		ToolName  string          `json:"tool_name"`
		ToolInput json.RawMessage `json:"tool_input"`
	}
	var t toolName
	err := json.Unmarshal(data, &t)
	if err != nil {
		return err
	}
	var tgt any
	switch t.ToolName {
	case "Task":
		tgt = AgentInput{}
	case "AskUserQuestion":
		tgt = AskUserQuestionInput{}
	case "Bash":
		tgt = BashInput{}
	case "TaskOutput":
		tgt = TaskOutputInput{}
	case "Edit":
		tgt = FileEditInput{}
	case "Read":
		tgt = FileReadInput{}
	case "Write":
		tgt = FileWriteInput{}
	case "Glob":
		tgt = GlobInput{}
	case "Grep":
		tgt = GrepInput{}
	case "TaskStop":
		tgt = TaskStopInput{}
	case "NotebookEdit":
		tgt = NotebookEditInput{}
	case "WebFetch":
		tgt = WebFetchInput{}
	case "WebSearch":
		tgt = WebSearchInput{}
	case "TodoWrite":
		tgt = TodoWriteInput{}
	case "ExitPlanMode":
		tgt = ExitPlanModeInput{}
	case "ListMcpResources":
		tgt = ListMcpResourcesInput{}
	case "ReadMcpResource":
		tgt = ReadMcpResourceInput{}
	case "Config":
		tgt = ConfigInput{}
	case "EnterWorktree":
		tgt = EnterWorktreeInput{}
	default:
		tgt = map[string]any{}
	}

	if err := json.Unmarshal(t.ToolInput, &tgt); err != nil {
		return err
	}

	i.ToolName = t.ToolName
	i.ToolInput = tgt

	return nil
}

func (i PreToolUseHookInput) ToProto() *pb.PreToolUseHookInput {
	ti := &pb.ToolInput{}
	switch x := i.preToolUseHookInputToolInput.ToolInput.(type) {
	case AskUserQuestionInput:
	case BashInput:
	case TaskOutputInput:
	case FileEditInput:
	case FileReadInput:
	case FileWriteInput:
	case GlobInput:
	case GrepInput:
	case TaskStopInput:
	case NotebookEditInput:
	case WebFetchInput:
	case WebSearchInput:
	case TodoWriteInput:
	case ExitPlanModeInput:
	case ListMcpResourcesInput:
	case ReadMcpResourceInput:
	case ConfigInput:
	case EnterWorktreeInput:
	default:
	}
	return &pb.PreToolUseHookInput{
		SessionId:      i.SessionID,
		TranscriptPath: i.TranscriptPath,
		Cwd:            i.Cwd,
		PermissionMode: i.PermissionMode,
		ToolName:       i.ToolName,
		ToolInput:      ti,
		HookEventName:  i.HookEventName,
		ToolUseId:      i.ToolUseID,
	}
}

func (i *PreToolUseHookInput) FromProto(p *pb.PreToolUseHookInput) {
	var ti any
	switch v := ti.GetInput().(type) {
	case *pb.ToolInput_Agent:
		v.Agent
	case *pb.ToolInput_AskUserQuestion:
		v.AskUserQuestion
	case *pb.ToolInput_Bash:
		v.Bash
	case *pb.ToolInput_BashOutput:
		v.BashOutput
	case *pb.ToolInput_FileEdit:
		v.FileEdit
	case *pb.ToolInput_FileRead:
		v.FileRead
	case *pb.ToolInput_FileWrite:
		v.FileWrite
	case *pb.ToolInput_Glob:
		v.Glob
	case *pb.ToolInput_Grep:
		v.Grep
	case *pb.ToolInput_KillShell:
		v.KillShell
	case *pb.ToolInput_NotebookEdit:
		v.NotebookEdit
	case *pb.ToolInput_WebFetch:
		v.WebFetch
	case *pb.ToolInput_WebSearch:
		v.WebSearch
	case *pb.ToolInput_TodoWrite:
		v.TodoWrite
	case *pb.ToolInput_ExitPlanMode:
		v.ExitPlanMode
	case *pb.ToolInput_ListMcpResources:
		v.ListMcpResources
	case *pb.ToolInput_ReadMcpResource:
		v.ReadMcpResource
	case *pb.ToolInput_McpTool:
		ti = v.McpTool.AsMap()
	}
	i.SessionID = p.GetSessionId()
	i.TranscriptPath = p.GetTranscriptPath()
	i.Cwd = p.GetCwd()
	i.PermissionMode = p.PermissionMode
	i.ToolName = p.GetToolName()
	i.ToolInput = ti
	i.HookEventName = p.GetHookEventName()
	i.ToolUseID = p.GetToolUseId()
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
