package models

import (
	"encoding/json"
	"fmt"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

var unmarshaler = protojson.UnmarshalOptions{DiscardUnknown: true}

// ToProto converts the PreToolUseHookInput to its proto equivalent.
func (m *PreToolUseHookInput) ToProto() (*pb.PreToolUseHookInput, error) {
	ti, err := toolInputToProto(m.ToolName, m.ToolInput)
	if err != nil {
		return nil, fmt.Errorf("converting tool_input: %w", err)
	}

	p := &pb.PreToolUseHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: m.PermissionMode,
		ToolName:       m.ToolName,
		ToolInput:      ti,
		HookEventName:  m.HookEventName,
		ToolUseId:      m.ToolUseID,
	}
	return p, nil
}

// FromProto populates the receiver's fields from a proto PreToolUseHookInput.
func (m *PreToolUseHookInput) FromProto(p *pb.PreToolUseHookInput) error {
	raw, err := toolInputFromProto(p.GetToolInput())
	if err != nil {
		return fmt.Errorf("converting tool_input: %w", err)
	}

	m.SessionID = p.GetSessionId()
	m.TranscriptPath = p.GetTranscriptPath()
	m.Cwd = p.GetCwd()
	m.PermissionMode = p.PermissionMode
	m.ToolName = p.GetToolName()
	m.ToolInput = raw
	m.HookEventName = p.GetHookEventName()
	m.ToolUseID = p.GetToolUseId()
	return nil
}

// toolInputToProto converts raw JSON tool input + tool name to a proto ToolInput.
// It uses protojson.Unmarshal on the specific proto message type, which works because
// the individual proto messages accept both camelCase and proto field names.
func toolInputToProto(toolName string, raw json.RawMessage) (*pb.ToolInput, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	msg, wrap := newProtoToolInput(toolName)
	if msg == nil {
		// Unknown/MCP tool: parse as Struct.
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("unmarshal MCP tool input: %w", err)
		}
		s, err := structpb.NewStruct(m)
		if err != nil {
			return nil, fmt.Errorf("creating Struct for MCP tool: %w", err)
		}
		return &pb.ToolInput{
			Input: &pb.ToolInput_McpTool{McpTool: s},
		}, nil
	}

	if err := unmarshaler.Unmarshal(raw, msg); err != nil {
		return nil, fmt.Errorf("unmarshal %q into proto: %w", toolName, err)
	}

	ti := &pb.ToolInput{}
	wrap(ti)
	return ti, nil
}

// toolInputFromProto converts a proto ToolInput to raw JSON.
func toolInputFromProto(ti *pb.ToolInput) (json.RawMessage, error) {
	if ti == nil {
		return nil, nil
	}

	var msg proto.Message
	switch v := ti.GetInput().(type) {
	case *pb.ToolInput_Agent:
		msg = v.Agent
	case *pb.ToolInput_AskUserQuestion:
		msg = v.AskUserQuestion
	case *pb.ToolInput_Bash:
		msg = v.Bash
	case *pb.ToolInput_BashOutput:
		msg = v.BashOutput
	case *pb.ToolInput_FileEdit:
		msg = v.FileEdit
	case *pb.ToolInput_FileRead:
		msg = v.FileRead
	case *pb.ToolInput_FileWrite:
		msg = v.FileWrite
	case *pb.ToolInput_Glob:
		msg = v.Glob
	case *pb.ToolInput_Grep:
		msg = v.Grep
	case *pb.ToolInput_KillShell:
		msg = v.KillShell
	case *pb.ToolInput_NotebookEdit:
		msg = v.NotebookEdit
	case *pb.ToolInput_WebFetch:
		msg = v.WebFetch
	case *pb.ToolInput_WebSearch:
		msg = v.WebSearch
	case *pb.ToolInput_TodoWrite:
		msg = v.TodoWrite
	case *pb.ToolInput_ExitPlanMode:
		msg = v.ExitPlanMode
	case *pb.ToolInput_ListMcpResources:
		msg = v.ListMcpResources
	case *pb.ToolInput_ReadMcpResource:
		msg = v.ReadMcpResource
	case *pb.ToolInput_McpTool:
		data, err := protojson.Marshal(v.McpTool)
		if err != nil {
			return nil, fmt.Errorf("marshal MCP tool Struct: %w", err)
		}
		return json.RawMessage(data), nil
	default:
		return nil, nil
	}

	data, err := protojson.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal proto tool input: %w", err)
	}
	return json.RawMessage(data), nil
}

// newProtoToolInput returns the proto message for the given tool name and
// a function that wraps it into a ToolInput oneof. Returns nil, nil for
// unknown/MCP tools.
func newProtoToolInput(toolName string) (proto.Message, func(*pb.ToolInput)) {
	switch toolName {
	case "Read":
		m := &pb.FileReadInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_FileRead{FileRead: m} }
	case "Write":
		m := &pb.FileWriteInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_FileWrite{FileWrite: m} }
	case "Edit":
		m := &pb.FileEditInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_FileEdit{FileEdit: m} }
	case "Bash":
		m := &pb.BashInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_Bash{Bash: m} }
	case "Glob":
		m := &pb.GlobInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_Glob{Glob: m} }
	case "Grep":
		m := &pb.GrepInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_Grep{Grep: m} }
	case "Task":
		m := &pb.AgentInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_Agent{Agent: m} }
	case "WebFetch":
		m := &pb.WebFetchInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_WebFetch{WebFetch: m} }
	case "WebSearch":
		m := &pb.WebSearchInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_WebSearch{WebSearch: m} }
	case "NotebookEdit":
		m := &pb.NotebookEditInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_NotebookEdit{NotebookEdit: m} }
	case "AskUserQuestion":
		m := &pb.AskUserQuestionInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_AskUserQuestion{AskUserQuestion: m} }
	case "ExitPlanMode":
		m := &pb.ExitPlanModeInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_ExitPlanMode{ExitPlanMode: m} }
	case "EnterPlanMode":
		// EnterPlanMode has no input parameters.
		return nil, nil
	case "BashOutput", "TaskOutput":
		m := &pb.BashOutputInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_BashOutput{BashOutput: m} }
	case "KillShell", "TaskStop":
		m := &pb.KillShellInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_KillShell{KillShell: m} }
	case "TodoWrite", "TaskCreate", "TaskUpdate", "TaskList", "TaskGet":
		m := &pb.TodoWriteInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_TodoWrite{TodoWrite: m} }
	case "ListMcpResources":
		m := &pb.ListMcpResourcesInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_ListMcpResources{ListMcpResources: m} }
	case "ReadMcpResource":
		m := &pb.ReadMcpResourceInput{}
		return m, func(ti *pb.ToolInput) { ti.Input = &pb.ToolInput_ReadMcpResource{ReadMcpResource: m} }
	default:
		return nil, nil
	}
}
