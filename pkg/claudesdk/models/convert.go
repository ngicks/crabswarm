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

// ToProto converts the SyncHookJSONOutput to its proto equivalent.
func (m *SyncHookJSONOutput) ToProto() (*pb.SyncHookJSONOutput, error) {
	p := &pb.SyncHookJSONOutput{
		Continue:       m.Continue,
		SuppressOutput: m.SuppressOutput,
		StopReason:     m.StopReason,
		Decision:       m.Decision,
		SystemMessage:  m.SystemMessage,
		Reason:         m.Reason,
	}

	if m.HookSpecificOutput != nil {
		hso, err := hookSpecificOutputToProto(m.HookSpecificOutput)
		if err != nil {
			return nil, fmt.Errorf("converting hook_specific_output: %w", err)
		}
		p.HookSpecificOutput = hso
	}

	return p, nil
}

// FromProto populates the receiver's fields from a proto SyncHookJSONOutput.
func (m *SyncHookJSONOutput) FromProto(p *pb.SyncHookJSONOutput) error {
	m.Continue = p.Continue
	m.SuppressOutput = p.SuppressOutput
	m.StopReason = p.StopReason
	m.Decision = p.Decision
	m.SystemMessage = p.SystemMessage
	m.Reason = p.Reason

	if p.HookSpecificOutput != nil {
		hso, err := hookSpecificOutputFromProto(p.HookSpecificOutput)
		if err != nil {
			return fmt.Errorf("converting hook_specific_output: %w", err)
		}
		m.HookSpecificOutput = hso
	}

	return nil
}

func hookSpecificOutputToProto(h *HookSpecificOutput) (*pb.HookSpecificOutput, error) {
	p := &pb.HookSpecificOutput{}

	// Determine which oneof variant to use based on populated fields.
	if h.Decision != nil {
		// PermissionRequest
		pr := &pb.PermissionRequestHookSpecificOutput{}
		result, err := permissionRequestDecisionToProto(h.Decision)
		if err != nil {
			return nil, err
		}
		pr.Decision = result
		p.Output = &pb.HookSpecificOutput_PermissionRequest{PermissionRequest: pr}
	} else if h.PermissionDecision != nil || h.PermissionDecisionReason != nil || len(h.UpdatedInput) > 0 {
		// PreToolUse
		ptuo := &pb.PreToolUseHookSpecificOutput{
			PermissionDecision:       h.PermissionDecision,
			PermissionDecisionReason: h.PermissionDecisionReason,
		}
		if len(h.UpdatedInput) > 0 {
			// We can't convert UpdatedInput without knowing the tool name,
			// so we skip it here. It remains as raw JSON.
		}
		p.Output = &pb.HookSpecificOutput_PreToolUse{PreToolUse: ptuo}
	} else if h.AdditionalContext != nil {
		// Could be UserPromptSubmit, SessionStart, or PostToolUse.
		// Default to PostToolUse since they all have the same field.
		ptuo := &pb.PostToolUseHookSpecificOutput{
			AdditionalContext: h.AdditionalContext,
		}
		p.Output = &pb.HookSpecificOutput_PostToolUse{PostToolUse: ptuo}
	}

	return p, nil
}

func hookSpecificOutputFromProto(p *pb.HookSpecificOutput) (*HookSpecificOutput, error) {
	h := &HookSpecificOutput{}

	switch v := p.GetOutput().(type) {
	case *pb.HookSpecificOutput_PreToolUse:
		h.PermissionDecision = v.PreToolUse.PermissionDecision
		h.PermissionDecisionReason = v.PreToolUse.PermissionDecisionReason
		if v.PreToolUse.UpdatedInput != nil {
			raw, err := toolInputFromProto(v.PreToolUse.UpdatedInput)
			if err != nil {
				return nil, fmt.Errorf("converting updated_input: %w", err)
			}
			h.UpdatedInput = raw
		}
	case *pb.HookSpecificOutput_UserPromptSubmit:
		h.AdditionalContext = v.UserPromptSubmit.AdditionalContext
	case *pb.HookSpecificOutput_SessionStart:
		h.AdditionalContext = v.SessionStart.AdditionalContext
	case *pb.HookSpecificOutput_PostToolUse:
		h.AdditionalContext = v.PostToolUse.AdditionalContext
	case *pb.HookSpecificOutput_PermissionRequest:
		if v.PermissionRequest.Decision != nil {
			dec, err := permissionRequestDecisionFromProto(v.PermissionRequest.Decision)
			if err != nil {
				return nil, err
			}
			h.Decision = dec
		}
	}

	return h, nil
}

func permissionRequestDecisionToProto(d *PermissionRequestDecision) (*pb.PermissionResult, error) {
	result := &pb.PermissionResult{}

	switch d.Behavior {
	case "allow":
		allow := &pb.AllowResult{}
		if len(d.UpdatedInput) > 0 {
			// Parse as generic Struct since we don't know the tool name
			var m map[string]any
			if err := json.Unmarshal(d.UpdatedInput, &m); err != nil {
				return nil, fmt.Errorf("unmarshal updatedInput: %w", err)
			}
			s, err := structpb.NewStruct(m)
			if err != nil {
				return nil, fmt.Errorf("creating Struct for updatedInput: %w", err)
			}
			allow.UpdatedInput = &pb.ToolInput{
				Input: &pb.ToolInput_McpTool{McpTool: s},
			}
		}
		// updatedPermissions conversion would require full PermissionUpdate parsing;
		// omitted for now as it's not needed by the auto-approve hook.
		result.Result = &pb.PermissionResult_Allow{Allow: allow}
	case "deny":
		deny := &pb.DenyResult{
			Interrupt: d.Interrupt,
		}
		if d.Message != nil {
			deny.Message = *d.Message
		}
		result.Result = &pb.PermissionResult_Deny{Deny: deny}
	default:
		return nil, fmt.Errorf("unknown behavior: %q", d.Behavior)
	}

	return result, nil
}

func permissionRequestDecisionFromProto(p *pb.PermissionResult) (*PermissionRequestDecision, error) {
	d := &PermissionRequestDecision{}

	switch v := p.GetResult().(type) {
	case *pb.PermissionResult_Allow:
		d.Behavior = "allow"
		if v.Allow.UpdatedInput != nil {
			raw, err := toolInputFromProto(v.Allow.UpdatedInput)
			if err != nil {
				return nil, fmt.Errorf("converting updated_input: %w", err)
			}
			d.UpdatedInput = raw
		}
		// updatedPermissions conversion omitted for now.
	case *pb.PermissionResult_Deny:
		d.Behavior = "deny"
		msg := v.Deny.Message
		if msg != "" {
			d.Message = &msg
		}
		d.Interrupt = v.Deny.Interrupt
	}

	return d, nil
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
