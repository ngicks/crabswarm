package models

import (
	"encoding/json"
	"fmt"
	"reflect"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

type hookInputBase struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
	HookEventName  string  `json:"hook_event_name"`
	ToolUseID      string  `json:"tool_use_id,omitempty"`
}

// PreToolUseHookInput represents input for the pre-tool-use hook.
type PreToolUseHookInput struct {
	hookInputBase
	hookInputToolInput
}

func (i *PreToolUseHookInput) UnmarshalJSON(data []byte) error {
	var base hookInputBase
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}
	var withTool hookInputToolInput
	if err := json.Unmarshal(data, &withTool); err != nil {
		return err
	}
	i.hookInputBase = base
	i.hookInputToolInput = withTool
	return nil
}

type hookInputToolInput struct {
	ToolName  string `json:"tool_name"`
	ToolInput any    `json:"tool_input"`
}

func (i *hookInputToolInput) UnmarshalJSON(data []byte) error {
	type toolName struct {
		ToolName  string          `json:"tool_name"`
		ToolInput json.RawMessage `json:"tool_input"`
	}
	var t toolName
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	input, err := unmarshalModelToolInput(t.ToolName, t.ToolInput)
	if err != nil {
		return err
	}
	i.ToolName = t.ToolName
	i.ToolInput = input
	return nil
}

type hookInputToolInputResponse struct {
	hookInputToolInput
	ToolResponse any `json:"tool_response"`
}

func (i *hookInputToolInputResponse) UnmarshalJSON(data []byte) error {
	type withResponse struct {
		ToolName     string          `json:"tool_name"`
		ToolInput    json.RawMessage `json:"tool_input"`
		ToolResponse json.RawMessage `json:"tool_response"`
	}
	var w withResponse
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	input, err := unmarshalModelToolInput(w.ToolName, w.ToolInput)
	if err != nil {
		return err
	}
	response, err := unmarshalModelToolOutput(w.ToolName, w.ToolResponse)
	if err != nil {
		return err
	}
	i.ToolName = w.ToolName
	i.ToolInput = input
	i.ToolResponse = response
	return nil
}

// PostToolUseHookInput represents input for the post-tool-use hook.
type PostToolUseHookInput struct {
	hookInputBase
	hookInputToolInputResponse
}

func (i *PostToolUseHookInput) UnmarshalJSON(data []byte) error {
	var base hookInputBase
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}
	var withToolResponse hookInputToolInputResponse
	if err := json.Unmarshal(data, &withToolResponse); err != nil {
		return err
	}
	i.hookInputBase = base
	i.hookInputToolInputResponse = withToolResponse
	return nil
}

// PostToolUseFailureHookInput represents input for the post-tool-use-failure hook.
type PostToolUseFailureHookInput struct {
	hookInputBase
	hookInputToolInput
	Error       string `json:"error"`
	IsInterrupt *bool  `json:"is_interrupt,omitempty"`
}

func (i *PostToolUseFailureHookInput) UnmarshalJSON(data []byte) error {
	var base hookInputBase
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}
	var withTool hookInputToolInput
	if err := json.Unmarshal(data, &withTool); err != nil {
		return err
	}
	var detail struct {
		Error       string `json:"error"`
		IsInterrupt *bool  `json:"is_interrupt,omitempty"`
	}
	if err := json.Unmarshal(data, &detail); err != nil {
		return err
	}
	i.hookInputBase = base
	i.hookInputToolInput = withTool
	i.Error = detail.Error
	i.IsInterrupt = detail.IsInterrupt
	return nil
}

// NotificationHookInput represents input for the notification hook.
type NotificationHookInput struct {
	hookInputBase
	Message string  `json:"message"`
	Title   *string `json:"title,omitempty"`
}

// UserPromptSubmitHookInput represents input for the user-prompt-submit hook.
type UserPromptSubmitHookInput struct {
	hookInputBase
	Prompt string `json:"prompt"`
}

// SessionStartHookInput represents input for the session-start hook.
type SessionStartHookInput struct {
	hookInputBase
	Source string `json:"source"`
}

// SessionEndHookInput represents input for the session-end hook.
type SessionEndHookInput struct {
	hookInputBase
	Reason string `json:"reason"`
}

// StopHookInput represents input for the stop hook.
type StopHookInput struct {
	hookInputBase
	StopHookActive bool `json:"stop_hook_active"`
}

// SubagentStartHookInput represents input for the subagent-start hook.
type SubagentStartHookInput struct {
	hookInputBase
	AgentID   string `json:"agent_id"`
	AgentType string `json:"agent_type"`
}

// SubagentStopHookInput represents input for the subagent-stop hook.
type SubagentStopHookInput struct {
	hookInputBase
	StopHookActive bool `json:"stop_hook_active"`
}

// PreCompactHookInput represents input for the pre-compact hook.
type PreCompactHookInput struct {
	hookInputBase
	Trigger            string  `json:"trigger"`
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// PermissionRequestHookInput represents input for the permission-request hook.
type PermissionRequestHookInput struct {
	hookInputBase
	hookInputToolInput
	PermissionSuggestions []PermissionUpdate `json:"permission_suggestions,omitempty"`
}

func (i *PermissionRequestHookInput) UnmarshalJSON(data []byte) error {
	var base hookInputBase
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}
	var withTool hookInputToolInput
	if err := json.Unmarshal(data, &withTool); err != nil {
		return err
	}
	var detail struct {
		PermissionSuggestions []PermissionUpdate `json:"permission_suggestions,omitempty"`
	}
	if err := json.Unmarshal(data, &detail); err != nil {
		return err
	}
	i.hookInputBase = base
	i.hookInputToolInput = withTool
	i.PermissionSuggestions = detail.PermissionSuggestions
	return nil
}

// ToProto converts the PreToolUseHookInput to its proto equivalent.
func (m *PreToolUseHookInput) ToProto() (*pb.PreToolUseHookInput, error) {
	ti, err := modelToolInputToProto(m.ToolName, m.ToolInput)
	if err != nil {
		return nil, fmt.Errorf("converting tool_input: %w", err)
	}
	return &pb.PreToolUseHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: m.PermissionMode,
		ToolName:       m.ToolName,
		ToolInput:      ti,
		HookEventName:  m.HookEventName,
		ToolUseId:      m.ToolUseID,
	}, nil
}

// FromProto populates the receiver's fields from a proto PreToolUseHookInput.
func (m *PreToolUseHookInput) FromProto(p *pb.PreToolUseHookInput) error {
	input, err := modelToolInputFromProto(p.GetToolName(), p.GetToolInput())
	if err != nil {
		return fmt.Errorf("converting tool_input: %w", err)
	}
	m.SessionID = p.GetSessionId()
	m.TranscriptPath = p.GetTranscriptPath()
	m.Cwd = p.GetCwd()
	m.PermissionMode = p.PermissionMode
	m.ToolName = p.GetToolName()
	m.ToolInput = input
	m.HookEventName = p.GetHookEventName()
	m.ToolUseID = p.GetToolUseId()
	return nil
}

func (m *PostToolUseHookInput) ToProto() (*pb.PostToolUseHookInput, error) {
	ti, err := modelToolInputToProto(m.ToolName, m.ToolInput)
	if err != nil {
		return nil, fmt.Errorf("converting tool_input: %w", err)
	}
	to, err := modelToolOutputToProto(m.ToolName, m.ToolResponse)
	if err != nil {
		return nil, fmt.Errorf("converting tool_response: %w", err)
	}
	return &pb.PostToolUseHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: m.PermissionMode,
		ToolName:       m.ToolName,
		ToolInput:      ti,
		ToolResponse:   to,
		HookEventName:  m.HookEventName,
		ToolUseId:      m.ToolUseID,
	}, nil
}

func (m *PostToolUseHookInput) FromProto(p *pb.PostToolUseHookInput) error {
	input, err := modelToolInputFromProto(p.GetToolName(), p.GetToolInput())
	if err != nil {
		return fmt.Errorf("converting tool_input: %w", err)
	}
	output, err := modelToolOutputFromProto(p.GetToolName(), p.GetToolResponse())
	if err != nil {
		return fmt.Errorf("converting tool_response: %w", err)
	}
	m.SessionID = p.GetSessionId()
	m.TranscriptPath = p.GetTranscriptPath()
	m.Cwd = p.GetCwd()
	m.PermissionMode = p.PermissionMode
	m.ToolName = p.GetToolName()
	m.ToolInput = input
	m.ToolResponse = output
	m.HookEventName = p.GetHookEventName()
	m.ToolUseID = p.GetToolUseId()
	return nil
}

func (m *PostToolUseFailureHookInput) ToProto() (*pb.PostToolUseFailureHookInput, error) {
	ti, err := modelToolInputToProto(m.ToolName, m.ToolInput)
	if err != nil {
		return nil, fmt.Errorf("converting tool_input: %w", err)
	}
	return &pb.PostToolUseFailureHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: m.PermissionMode,
		ToolName:       m.ToolName,
		ToolInput:      ti,
		Error:          m.Error,
		IsInterrupt:    m.IsInterrupt,
		HookEventName:  m.HookEventName,
	}, nil
}

func (m *PostToolUseFailureHookInput) FromProto(p *pb.PostToolUseFailureHookInput) error {
	input, err := modelToolInputFromProto(p.GetToolName(), p.GetToolInput())
	if err != nil {
		return fmt.Errorf("converting tool_input: %w", err)
	}
	m.SessionID = p.GetSessionId()
	m.TranscriptPath = p.GetTranscriptPath()
	m.Cwd = p.GetCwd()
	m.PermissionMode = p.PermissionMode
	m.ToolName = p.GetToolName()
	m.ToolInput = input
	m.Error = p.GetError()
	m.IsInterrupt = p.IsInterrupt
	m.HookEventName = p.GetHookEventName()
	return nil
}

func (m *NotificationHookInput) ToProto() (*pb.NotificationHookInput, error) {
	return &pb.NotificationHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: m.PermissionMode,
		Message:        m.Message,
		Title:          m.Title,
		HookEventName:  m.HookEventName,
	}, nil
}

func (m *NotificationHookInput) FromProto(p *pb.NotificationHookInput) error {
	m.SessionID = p.GetSessionId()
	m.TranscriptPath = p.GetTranscriptPath()
	m.Cwd = p.GetCwd()
	m.PermissionMode = p.PermissionMode
	m.Message = p.GetMessage()
	m.Title = p.Title
	m.HookEventName = p.GetHookEventName()
	return nil
}

func (m *UserPromptSubmitHookInput) ToProto() (*pb.UserPromptSubmitHookInput, error) {
	return &pb.UserPromptSubmitHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: m.PermissionMode,
		Prompt:         m.Prompt,
		HookEventName:  m.HookEventName,
	}, nil
}

func (m *UserPromptSubmitHookInput) FromProto(p *pb.UserPromptSubmitHookInput) error {
	m.SessionID = p.GetSessionId()
	m.TranscriptPath = p.GetTranscriptPath()
	m.Cwd = p.GetCwd()
	m.PermissionMode = p.PermissionMode
	m.Prompt = p.GetPrompt()
	m.HookEventName = p.GetHookEventName()
	return nil
}

func (m *SessionStartHookInput) ToProto() (*pb.SessionStartHookInput, error) {
	return &pb.SessionStartHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: m.PermissionMode,
		Source:         m.Source,
		HookEventName:  m.HookEventName,
	}, nil
}

func (m *SessionStartHookInput) FromProto(p *pb.SessionStartHookInput) error {
	m.SessionID = p.GetSessionId()
	m.TranscriptPath = p.GetTranscriptPath()
	m.Cwd = p.GetCwd()
	m.PermissionMode = p.PermissionMode
	m.Source = p.GetSource()
	m.HookEventName = p.GetHookEventName()
	return nil
}

func (m *SessionEndHookInput) ToProto() (*pb.SessionEndHookInput, error) {
	return &pb.SessionEndHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: m.PermissionMode,
		Reason:         m.Reason,
		HookEventName:  m.HookEventName,
	}, nil
}

func (m *SessionEndHookInput) FromProto(p *pb.SessionEndHookInput) error {
	m.SessionID = p.GetSessionId()
	m.TranscriptPath = p.GetTranscriptPath()
	m.Cwd = p.GetCwd()
	m.PermissionMode = p.PermissionMode
	m.Reason = p.GetReason()
	m.HookEventName = p.GetHookEventName()
	return nil
}

func (m *StopHookInput) ToProto() (*pb.StopHookInput, error) {
	return &pb.StopHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: m.PermissionMode,
		StopHookActive: m.StopHookActive,
		HookEventName:  m.HookEventName,
	}, nil
}

func (m *StopHookInput) FromProto(p *pb.StopHookInput) error {
	m.SessionID = p.GetSessionId()
	m.TranscriptPath = p.GetTranscriptPath()
	m.Cwd = p.GetCwd()
	m.PermissionMode = p.PermissionMode
	m.StopHookActive = p.GetStopHookActive()
	m.HookEventName = p.GetHookEventName()
	return nil
}

func (m *SubagentStartHookInput) ToProto() (*pb.SubagentStartHookInput, error) {
	return &pb.SubagentStartHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: m.PermissionMode,
		AgentId:        m.AgentID,
		AgentType:      m.AgentType,
		HookEventName:  m.HookEventName,
	}, nil
}

func (m *SubagentStartHookInput) FromProto(p *pb.SubagentStartHookInput) error {
	m.SessionID = p.GetSessionId()
	m.TranscriptPath = p.GetTranscriptPath()
	m.Cwd = p.GetCwd()
	m.PermissionMode = p.PermissionMode
	m.AgentID = p.GetAgentId()
	m.AgentType = p.GetAgentType()
	m.HookEventName = p.GetHookEventName()
	return nil
}

func (m *SubagentStopHookInput) ToProto() (*pb.SubagentStopHookInput, error) {
	return &pb.SubagentStopHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: m.PermissionMode,
		StopHookActive: m.StopHookActive,
		HookEventName:  m.HookEventName,
	}, nil
}

func (m *SubagentStopHookInput) FromProto(p *pb.SubagentStopHookInput) error {
	m.SessionID = p.GetSessionId()
	m.TranscriptPath = p.GetTranscriptPath()
	m.Cwd = p.GetCwd()
	m.PermissionMode = p.PermissionMode
	m.StopHookActive = p.GetStopHookActive()
	m.HookEventName = p.GetHookEventName()
	return nil
}

func (m *PreCompactHookInput) ToProto() (*pb.PreCompactHookInput, error) {
	return &pb.PreCompactHookInput{
		SessionId:          m.SessionID,
		TranscriptPath:     m.TranscriptPath,
		Cwd:                m.Cwd,
		PermissionMode:     m.PermissionMode,
		Trigger:            m.Trigger,
		CustomInstructions: m.CustomInstructions,
		HookEventName:      m.HookEventName,
	}, nil
}

func (m *PreCompactHookInput) FromProto(p *pb.PreCompactHookInput) error {
	m.SessionID = p.GetSessionId()
	m.TranscriptPath = p.GetTranscriptPath()
	m.Cwd = p.GetCwd()
	m.PermissionMode = p.PermissionMode
	m.Trigger = p.GetTrigger()
	m.CustomInstructions = p.CustomInstructions
	m.HookEventName = p.GetHookEventName()
	return nil
}

func (m *PermissionRequestHookInput) ToProto() (*pb.PermissionRequestHookInput, error) {
	ti, err := modelToolInputToProto(m.ToolName, m.ToolInput)
	if err != nil {
		return nil, fmt.Errorf("converting tool_input: %w", err)
	}
	suggestions, err := permissionUpdatesToProto(m.PermissionSuggestions)
	if err != nil {
		return nil, fmt.Errorf("converting permission_suggestions: %w", err)
	}
	return &pb.PermissionRequestHookInput{
		SessionId:             m.SessionID,
		TranscriptPath:        m.TranscriptPath,
		Cwd:                   m.Cwd,
		PermissionMode:        m.PermissionMode,
		ToolName:              m.ToolName,
		ToolInput:             ti,
		PermissionSuggestions: suggestions,
		HookEventName:         m.HookEventName,
	}, nil
}

func (m *PermissionRequestHookInput) FromProto(p *pb.PermissionRequestHookInput) error {
	input, err := modelToolInputFromProto(p.GetToolName(), p.GetToolInput())
	if err != nil {
		return fmt.Errorf("converting tool_input: %w", err)
	}
	suggestions, err := permissionUpdatesFromProto(p.GetPermissionSuggestions())
	if err != nil {
		return fmt.Errorf("converting permission_suggestions: %w", err)
	}
	m.SessionID = p.GetSessionId()
	m.TranscriptPath = p.GetTranscriptPath()
	m.Cwd = p.GetCwd()
	m.PermissionMode = p.PermissionMode
	m.ToolName = p.GetToolName()
	m.ToolInput = input
	m.PermissionSuggestions = suggestions
	m.HookEventName = p.GetHookEventName()
	return nil
}

func modelToolInputToProto(toolName string, input any) (*pb.ToolInput, error) {
	if input == nil {
		return nil, nil
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal model tool_input: %w", err)
	}
	return toolInputToProto(toolName, raw)
}

func modelToolInputFromProto(toolName string, ti *pb.ToolInput) (any, error) {
	raw, err := toolInputFromProto(ti)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	return unmarshalModelToolInput(toolName, raw)
}

func unmarshalModelToolInput(toolName string, raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	tgt := newModelToolInput(toolName)
	decoded, err := decodeJSONToModel(tgt, raw)
	if err != nil {
		return nil, fmt.Errorf("unmarshal tool_input for %q: %w", toolName, err)
	}
	return decoded, nil
}

func newModelToolInput(toolName string) any {
	switch toolName {
	case "Task":
		return AgentInput{}
	case "AskUserQuestion":
		return AskUserQuestionInput{}
	case "Bash":
		return BashInput{}
	case "TaskOutput":
		return TaskOutputInput{}
	case "Edit":
		return FileEditInput{}
	case "Read":
		return FileReadInput{}
	case "Write":
		return FileWriteInput{}
	case "Glob":
		return GlobInput{}
	case "Grep":
		return GrepInput{}
	case "TaskStop":
		return TaskStopInput{}
	case "NotebookEdit":
		return NotebookEditInput{}
	case "WebFetch":
		return WebFetchInput{}
	case "WebSearch":
		return WebSearchInput{}
	case "TodoWrite":
		return TodoWriteInput{}
	case "ExitPlanMode":
		return ExitPlanModeInput{}
	case "ListMcpResources":
		return ListMcpResourcesInput{}
	case "ReadMcpResource":
		return ReadMcpResourceInput{}
	case "Config":
		return ConfigInput{}
	case "EnterWorktree":
		return EnterWorktreeInput{}
	default:
		return map[string]any{}
	}
}

func modelToolOutputToProto(toolName string, output any) (*pb.ToolOutput, error) {
	if output == nil {
		return nil, nil
	}
	raw, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("marshal model tool_response: %w", err)
	}
	msg, wrap := newProtoToolOutput(toolName)
	if msg == nil {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("unmarshal MCP tool output: %w", err)
		}
		s, err := structpb.NewStruct(m)
		if err != nil {
			return nil, fmt.Errorf("creating Struct for MCP tool output: %w", err)
		}
		return &pb.ToolOutput{Output: &pb.ToolOutput_McpTool{McpTool: s}}, nil
	}
	if err := unmarshaler.Unmarshal(raw, msg); err != nil {
		return nil, fmt.Errorf("unmarshal %q tool_output into proto: %w", toolName, err)
	}
	to := &pb.ToolOutput{}
	wrap(to)
	return to, nil
}

func modelToolOutputFromProto(_ string, to *pb.ToolOutput) (any, error) {
	if to == nil {
		return nil, nil
	}
	var (
		msg proto.Message
		tgt any
	)
	switch v := to.GetOutput().(type) {
	case *pb.ToolOutput_Task:
		msg = v.Task
		tgt = TaskOutput{}
	case *pb.ToolOutput_AskUserQuestion:
		msg = v.AskUserQuestion
		tgt = AskUserQuestionOutput{}
	case *pb.ToolOutput_Bash:
		msg = v.Bash
		tgt = BashOutput{}
	case *pb.ToolOutput_BashOutput:
		msg = v.BashOutput
		tgt = BashOutputToolOutput{}
	case *pb.ToolOutput_Edit:
		msg = v.Edit
		tgt = EditOutput{}
	case *pb.ToolOutput_Read:
		msg = v.Read
		tgt = ReadOutput{}
	case *pb.ToolOutput_Write:
		msg = v.Write
		tgt = WriteOutput{}
	case *pb.ToolOutput_Glob:
		msg = v.Glob
		tgt = GlobOutput{}
	case *pb.ToolOutput_Grep:
		msg = v.Grep
		tgt = GrepOutput{}
	case *pb.ToolOutput_KillBash:
		msg = v.KillBash
		tgt = KillBashOutput{}
	case *pb.ToolOutput_NotebookEdit:
		msg = v.NotebookEdit
		tgt = NotebookEditOutput{}
	case *pb.ToolOutput_WebFetch:
		msg = v.WebFetch
		tgt = WebFetchOutput{}
	case *pb.ToolOutput_WebSearch:
		msg = v.WebSearch
		tgt = WebSearchOutput{}
	case *pb.ToolOutput_TodoWrite:
		msg = v.TodoWrite
		tgt = TodoWriteOutput{}
	case *pb.ToolOutput_ExitPlanMode:
		msg = v.ExitPlanMode
		tgt = ExitPlanModeOutput{}
	case *pb.ToolOutput_ListMcpResources:
		msg = v.ListMcpResources
		tgt = ListMcpResourcesOutput{}
	case *pb.ToolOutput_ReadMcpResource:
		msg = v.ReadMcpResource
		tgt = ReadMcpResourceOutput{}
	case *pb.ToolOutput_McpTool:
		return v.McpTool.AsMap(), nil
	default:
		return nil, nil
	}
	raw, err := protojson.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal proto tool_output: %w", err)
	}
	decoded, err := decodeJSONToModel(tgt, raw)
	if err != nil {
		return nil, fmt.Errorf("unmarshal model tool_output: %w", err)
	}
	return decoded, nil
}

func unmarshalModelToolOutput(toolName string, raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	tgt := newModelToolOutput(toolName)
	decoded, err := decodeJSONToModel(tgt, raw)
	if err != nil {
		return nil, fmt.Errorf("unmarshal tool_response for %q: %w", toolName, err)
	}
	return decoded, nil
}

func newModelToolOutput(toolName string) any {
	switch toolName {
	case "Task":
		return TaskOutput{}
	case "AskUserQuestion":
		return AskUserQuestionOutput{}
	case "Bash":
		return BashOutput{}
	case "BashOutput", "TaskOutput":
		return BashOutputToolOutput{}
	case "Edit":
		return EditOutput{}
	case "Read":
		return ReadOutput{}
	case "Write":
		return WriteOutput{}
	case "Glob":
		return GlobOutput{}
	case "Grep":
		return GrepOutput{}
	case "KillShell", "TaskStop":
		return KillBashOutput{}
	case "NotebookEdit":
		return NotebookEditOutput{}
	case "WebFetch":
		return WebFetchOutput{}
	case "WebSearch":
		return WebSearchOutput{}
	case "TodoWrite", "TaskCreate", "TaskUpdate", "TaskList", "TaskGet":
		return TodoWriteOutput{}
	case "ExitPlanMode":
		return ExitPlanModeOutput{}
	case "ListMcpResources":
		return ListMcpResourcesOutput{}
	case "ReadMcpResource":
		return ReadMcpResourceOutput{}
	default:
		return map[string]any{}
	}
}

func newProtoToolOutput(toolName string) (proto.Message, func(*pb.ToolOutput)) {
	switch toolName {
	case "Task":
		m := &pb.TaskOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_Task{Task: m} }
	case "AskUserQuestion":
		m := &pb.AskUserQuestionOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_AskUserQuestion{AskUserQuestion: m} }
	case "Bash":
		m := &pb.BashOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_Bash{Bash: m} }
	case "BashOutput", "TaskOutput":
		m := &pb.BashOutputToolOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_BashOutput{BashOutput: m} }
	case "Edit":
		m := &pb.EditOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_Edit{Edit: m} }
	case "Read":
		m := &pb.ReadOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_Read{Read: m} }
	case "Write":
		m := &pb.WriteOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_Write{Write: m} }
	case "Glob":
		m := &pb.GlobOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_Glob{Glob: m} }
	case "Grep":
		m := &pb.GrepOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_Grep{Grep: m} }
	case "KillShell", "TaskStop":
		m := &pb.KillBashOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_KillBash{KillBash: m} }
	case "NotebookEdit":
		m := &pb.NotebookEditOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_NotebookEdit{NotebookEdit: m} }
	case "WebFetch":
		m := &pb.WebFetchOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_WebFetch{WebFetch: m} }
	case "WebSearch":
		m := &pb.WebSearchOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_WebSearch{WebSearch: m} }
	case "TodoWrite", "TaskCreate", "TaskUpdate", "TaskList", "TaskGet":
		m := &pb.TodoWriteOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_TodoWrite{TodoWrite: m} }
	case "ExitPlanMode":
		m := &pb.ExitPlanModeOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_ExitPlanMode{ExitPlanMode: m} }
	case "ListMcpResources":
		m := &pb.ListMcpResourcesOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_ListMcpResources{ListMcpResources: m} }
	case "ReadMcpResource":
		m := &pb.ReadMcpResourceOutput{}
		return m, func(to *pb.ToolOutput) { to.Output = &pb.ToolOutput_ReadMcpResource{ReadMcpResource: m} }
	default:
		return nil, nil
	}
}

func permissionUpdatesToProto(updates []PermissionUpdate) ([]*pb.PermissionUpdate, error) {
	if len(updates) == 0 {
		return nil, nil
	}
	res := make([]*pb.PermissionUpdate, 0, len(updates))
	for _, u := range updates {
		raw, err := json.Marshal(u)
		if err != nil {
			return nil, fmt.Errorf("marshal permission update: %w", err)
		}
		p := &pb.PermissionUpdate{}
		if err := unmarshaler.Unmarshal(raw, p); err != nil {
			return nil, fmt.Errorf("unmarshal permission update into proto: %w", err)
		}
		res = append(res, p)
	}
	return res, nil
}

func permissionUpdatesFromProto(updates []*pb.PermissionUpdate) ([]PermissionUpdate, error) {
	if len(updates) == 0 {
		return nil, nil
	}
	res := make([]PermissionUpdate, 0, len(updates))
	for _, p := range updates {
		if p == nil {
			continue
		}
		raw, err := protojson.Marshal(p)
		if err != nil {
			return nil, fmt.Errorf("marshal permission update from proto: %w", err)
		}
		var u PermissionUpdate
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil, fmt.Errorf("unmarshal permission update to model: %w", err)
		}
		res = append(res, u)
	}
	return res, nil
}

func decodeJSONToModel(tgt any, raw []byte) (any, error) {
	t := reflect.TypeOf(tgt)
	if t == nil {
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, err
		}
		return v, nil
	}
	ptr := reflect.New(t)
	if err := json.Unmarshal(raw, ptr.Interface()); err != nil {
		return nil, err
	}
	return ptr.Elem().Interface(), nil
}
