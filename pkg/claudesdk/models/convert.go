package models

import (
	"encoding/json"
	"fmt"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// convertModelToolInputToProto converts model tool input + tool name to proto ToolInput.
func convertModelToolInputToProto(toolName string, input any) (*pb.ToolInput, error) {
	if input == nil {
		return nil, nil
	}

	switch toolName {
	case "Task":
		v, ok := asAgentInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected AgentInput, got %T", toolName, input)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_Agent{Agent: &pb.AgentInput{
			Description:  v.Description,
			Prompt:       v.Prompt,
			SubagentType: v.SubagentType,
		}}}, nil
	case "AskUserQuestion":
		v, ok := asAskUserQuestionInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected AskUserQuestionInput, got %T", toolName, input)
		}
		questions := make([]*pb.Question, 0, len(v.Questions))
		for _, q := range v.Questions {
			options := make([]*pb.QuestionOption, 0, len(q.Options))
			for _, o := range q.Options {
				options = append(options, &pb.QuestionOption{
					Label:       o.Label,
					Description: o.Description,
				})
			}
			questions = append(questions, &pb.Question{
				Question:    q.Question,
				Header:      q.Header,
				Options:     options,
				MultiSelect: q.MultiSelect,
			})
		}
		return &pb.ToolInput{Input: &pb.ToolInput_AskUserQuestion{AskUserQuestion: &pb.AskUserQuestionInput{
			Questions: questions,
			Answers:   cloneStringMap(v.Answers),
		}}}, nil
	case "Bash":
		v, ok := asBashInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected BashInput, got %T", toolName, input)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_Bash{Bash: &pb.BashInput{
			Command:         v.Command,
			Timeout:         cloneInt32Ptr(v.Timeout),
			Description:     cloneStringPtr(v.Description),
			RunInBackground: cloneBoolPtr(v.RunInBackground),
		}}}, nil
	case "BashOutput":
		v, ok := asBashOutputInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected BashOutputInput, got %T", toolName, input)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_BashOutput{BashOutput: &pb.BashOutputInput{
			BashId: v.BashID,
			Filter: cloneStringPtr(v.Filter),
		}}}, nil
	case "TaskOutput":
		v, ok := asTaskOutputInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected TaskOutputInput, got %T", toolName, input)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_BashOutput{BashOutput: &pb.BashOutputInput{
			BashId: v.TaskId,
		}}}, nil
	case "Edit":
		v, ok := asFileEditInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected FileEditInput, got %T", toolName, input)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_FileEdit{FileEdit: &pb.FileEditInput{
			FilePath:   v.FilePath,
			OldString:  v.OldString,
			NewString:  v.NewString,
			ReplaceAll: cloneBoolPtr(v.ReplaceAll),
		}}}, nil
	case "Read":
		v, ok := asFileReadInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected FileReadInput, got %T", toolName, input)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_FileRead{FileRead: &pb.FileReadInput{
			FilePath: v.FilePath,
			Offset:   cloneInt32Ptr(v.Offset),
			Limit:    cloneInt32Ptr(v.Limit),
		}}}, nil
	case "Write":
		v, ok := asFileWriteInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected FileWriteInput, got %T", toolName, input)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_FileWrite{FileWrite: &pb.FileWriteInput{
			FilePath: v.FilePath,
			Content:  v.Content,
		}}}, nil
	case "Glob":
		v, ok := asGlobInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected GlobInput, got %T", toolName, input)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_Glob{Glob: &pb.GlobInput{
			Pattern: v.Pattern,
			Path:    cloneStringPtr(v.Path),
		}}}, nil
	case "Grep":
		v, ok := asGrepInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected GrepInput, got %T", toolName, input)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_Grep{Grep: &pb.GrepInput{
			Pattern:         v.Pattern,
			Path:            cloneStringPtr(v.Path),
			Glob:            cloneStringPtr(v.Glob),
			Type:            cloneStringPtr(v.Type),
			OutputMode:      v.OutputMode,
			CaseInsensitive: cloneBoolPtr(v.CaseInsensitive),
			ShowLineNumbers: cloneBoolPtr(v.ShowLineNumbers),
			BeforeContext:   cloneInt32Ptr(v.BeforeContext),
			AfterContext:    cloneInt32Ptr(v.AfterContext),
			Context:         cloneInt32Ptr(v.Context),
			HeadLimit:       cloneInt32Ptr(v.HeadLimit),
			Multiline:       cloneBoolPtr(v.Multiline),
		}}}, nil
	case "KillShell":
		v, ok := asKillShellInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected KillShellInput, got %T", toolName, input)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_KillShell{KillShell: &pb.KillShellInput{
			ShellId: v.ShellID,
		}}}, nil
	case "TaskStop":
		v, ok := asTaskStopInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected TaskStopInput, got %T", toolName, input)
		}
		shellID := ""
		switch {
		case v.ShellId != nil:
			shellID = *v.ShellId
		case v.TaskId != nil:
			shellID = *v.TaskId
		}
		return &pb.ToolInput{Input: &pb.ToolInput_KillShell{KillShell: &pb.KillShellInput{
			ShellId: shellID,
		}}}, nil
	case "NotebookEdit":
		v, ok := asNotebookEditInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected NotebookEditInput, got %T", toolName, input)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_NotebookEdit{NotebookEdit: &pb.NotebookEditInput{
			NotebookPath: v.NotebookPath,
			CellId:       cloneStringPtr(v.CellId),
			NewSource:    v.NewSource,
			CellType:     v.CellType,
			EditMode:     v.EditMode,
		}}}, nil
	case "WebFetch":
		v, ok := asWebFetchInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected WebFetchInput, got %T", toolName, input)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_WebFetch{WebFetch: &pb.WebFetchInput{
			Url:    v.Url,
			Prompt: v.Prompt,
		}}}, nil
	case "WebSearch":
		v, ok := asWebSearchInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected WebSearchInput, got %T", toolName, input)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_WebSearch{WebSearch: &pb.WebSearchInput{
			Query:          v.Query,
			AllowedDomains: cloneStringSlice(v.AllowedDomains),
			BlockedDomains: cloneStringSlice(v.BlockedDomains),
		}}}, nil
	case "TodoWrite", "TaskCreate", "TaskUpdate", "TaskList", "TaskGet":
		v, ok := asTodoWriteInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected TodoWriteInput, got %T", toolName, input)
		}
		todos := make([]*pb.TodoItem, 0, len(v.Todos))
		for _, t := range v.Todos {
			todos = append(todos, &pb.TodoItem{
				Content:    t.Content,
				Status:     t.Status,
				ActiveForm: t.ActiveForm,
			})
		}
		return &pb.ToolInput{Input: &pb.ToolInput_TodoWrite{TodoWrite: &pb.TodoWriteInput{
			Todos: todos,
		}}}, nil
	case "ExitPlanMode":
		v, ok := asExitPlanModeInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected ExitPlanModeInput, got %T", toolName, input)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_ExitPlanMode{ExitPlanMode: &pb.ExitPlanModeInput{
			Plan: v.Plan,
		}}}, nil
	case "EnterPlanMode":
		return nil, nil
	case "ListMcpResources":
		v, ok := asListMcpResourcesInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected ListMcpResourcesInput, got %T", toolName, input)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_ListMcpResources{ListMcpResources: &pb.ListMcpResourcesInput{
			Server: cloneStringPtr(v.Server),
		}}}, nil
	case "ReadMcpResource":
		v, ok := asReadMcpResourceInput(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected ReadMcpResourceInput, got %T", toolName, input)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_ReadMcpResource{ReadMcpResource: &pb.ReadMcpResourceInput{
			Server: v.Server,
			Uri:    v.Uri,
		}}}, nil
	default:
		m, ok := asMap(input)
		if !ok {
			return nil, fmt.Errorf("tool %q: expected map[string]any for MCP/unknown tool, got %T", toolName, input)
		}
		s, err := structpb.NewStruct(m)
		if err != nil {
			return nil, fmt.Errorf("creating Struct for MCP tool: %w", err)
		}
		return &pb.ToolInput{Input: &pb.ToolInput_McpTool{McpTool: s}}, nil
	}
}

// convertProtoToolInputToModel converts a proto ToolInput to model tool input.
func convertProtoToolInputToModel(toolName string, ti *pb.ToolInput) (any, error) {
	if ti == nil {
		return nil, nil
	}

	switch v := ti.GetInput().(type) {
	case *pb.ToolInput_Agent:
		return AgentInput{
			Description:  v.Agent.GetDescription(),
			Prompt:       v.Agent.GetPrompt(),
			SubagentType: v.Agent.GetSubagentType(),
		}, nil
	case *pb.ToolInput_AskUserQuestion:
		questions := make([]Question, 0, len(v.AskUserQuestion.GetQuestions()))
		for _, q := range v.AskUserQuestion.GetQuestions() {
			options := make([]QuestionOption, 0, len(q.GetOptions()))
			for _, o := range q.GetOptions() {
				options = append(options, QuestionOption{
					Label:       o.GetLabel(),
					Description: o.GetDescription(),
				})
			}
			questions = append(questions, Question{
				Question:    q.GetQuestion(),
				Header:      q.GetHeader(),
				Options:     options,
				MultiSelect: q.GetMultiSelect(),
			})
		}
		return AskUserQuestionInput{
			Questions: questions,
			Answers:   cloneStringMap(v.AskUserQuestion.GetAnswers()),
		}, nil
	case *pb.ToolInput_Bash:
		return BashInput{
			Command:         v.Bash.GetCommand(),
			Timeout:         cloneInt32Ptr(v.Bash.Timeout),
			Description:     cloneStringPtr(v.Bash.Description),
			RunInBackground: cloneBoolPtr(v.Bash.RunInBackground),
		}, nil
	case *pb.ToolInput_BashOutput:
		if toolName == "TaskOutput" {
			return TaskOutputInput{
				TaskId: v.BashOutput.GetBashId(),
			}, nil
		}
		return BashOutputInput{
			BashID: v.BashOutput.GetBashId(),
			Filter: cloneStringPtr(v.BashOutput.Filter),
		}, nil
	case *pb.ToolInput_FileEdit:
		return FileEditInput{
			FilePath:   v.FileEdit.GetFilePath(),
			OldString:  v.FileEdit.GetOldString(),
			NewString:  v.FileEdit.GetNewString(),
			ReplaceAll: cloneBoolPtr(v.FileEdit.ReplaceAll),
		}, nil
	case *pb.ToolInput_FileRead:
		return FileReadInput{
			FilePath: v.FileRead.GetFilePath(),
			Offset:   cloneInt32Ptr(v.FileRead.Offset),
			Limit:    cloneInt32Ptr(v.FileRead.Limit),
		}, nil
	case *pb.ToolInput_FileWrite:
		return FileWriteInput{
			FilePath: v.FileWrite.GetFilePath(),
			Content:  v.FileWrite.GetContent(),
		}, nil
	case *pb.ToolInput_Glob:
		return GlobInput{
			Pattern: v.Glob.GetPattern(),
			Path:    cloneStringPtr(v.Glob.Path),
		}, nil
	case *pb.ToolInput_Grep:
		return GrepInput{
			Pattern:         v.Grep.GetPattern(),
			Path:            cloneStringPtr(v.Grep.Path),
			Glob:            cloneStringPtr(v.Grep.Glob),
			Type:            cloneStringPtr(v.Grep.Type),
			OutputMode:      v.Grep.GetOutputMode(),
			CaseInsensitive: cloneBoolPtr(v.Grep.CaseInsensitive),
			ShowLineNumbers: cloneBoolPtr(v.Grep.ShowLineNumbers),
			BeforeContext:   cloneInt32Ptr(v.Grep.BeforeContext),
			AfterContext:    cloneInt32Ptr(v.Grep.AfterContext),
			Context:         cloneInt32Ptr(v.Grep.Context),
			HeadLimit:       cloneInt32Ptr(v.Grep.HeadLimit),
			Multiline:       cloneBoolPtr(v.Grep.Multiline),
		}, nil
	case *pb.ToolInput_KillShell:
		if toolName == "TaskStop" {
			shellID := v.KillShell.GetShellId()
			return TaskStopInput{
				ShellId: &shellID,
			}, nil
		}
		return KillShellInput{
			ShellID: v.KillShell.GetShellId(),
		}, nil
	case *pb.ToolInput_NotebookEdit:
		return NotebookEditInput{
			NotebookPath: v.NotebookEdit.GetNotebookPath(),
			CellId:       cloneStringPtr(v.NotebookEdit.CellId),
			NewSource:    v.NotebookEdit.GetNewSource(),
			CellType:     v.NotebookEdit.GetCellType(),
			EditMode:     v.NotebookEdit.GetEditMode(),
		}, nil
	case *pb.ToolInput_WebFetch:
		return WebFetchInput{
			Url:    v.WebFetch.GetUrl(),
			Prompt: v.WebFetch.GetPrompt(),
		}, nil
	case *pb.ToolInput_WebSearch:
		return WebSearchInput{
			Query:          v.WebSearch.GetQuery(),
			AllowedDomains: cloneStringSlice(v.WebSearch.GetAllowedDomains()),
			BlockedDomains: cloneStringSlice(v.WebSearch.GetBlockedDomains()),
		}, nil
	case *pb.ToolInput_TodoWrite:
		todos := make([]TodoItem, 0, len(v.TodoWrite.GetTodos()))
		for _, t := range v.TodoWrite.GetTodos() {
			todos = append(todos, TodoItem{
				Content:    t.GetContent(),
				Status:     t.GetStatus(),
				ActiveForm: t.GetActiveForm(),
			})
		}
		return TodoWriteInput{Todos: todos}, nil
	case *pb.ToolInput_ExitPlanMode:
		return ExitPlanModeInput{
			Plan: v.ExitPlanMode.GetPlan(),
		}, nil
	case *pb.ToolInput_ListMcpResources:
		return ListMcpResourcesInput{
			Server: cloneStringPtr(v.ListMcpResources.Server),
		}, nil
	case *pb.ToolInput_ReadMcpResource:
		return ReadMcpResourceInput{
			Server: v.ReadMcpResource.GetServer(),
			Uri:    v.ReadMcpResource.GetUri(),
		}, nil
	case *pb.ToolInput_McpTool:
		return v.McpTool.AsMap(), nil
	default:
		return nil, nil
	}
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
			updatedInput, err := convertProtoToolInputToModel("", v.PreToolUse.UpdatedInput)
			if err != nil {
				return nil, fmt.Errorf("converting updated_input: %w", err)
			}
			raw, err := json.Marshal(updatedInput)
			if err != nil {
				return nil, fmt.Errorf("marshal updated_input: %w", err)
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
			updatedInput, err := convertProtoToolInputToModel("", v.Allow.UpdatedInput)
			if err != nil {
				return nil, fmt.Errorf("converting updated_input: %w", err)
			}
			raw, err := json.Marshal(updatedInput)
			if err != nil {
				return nil, fmt.Errorf("marshal updated_input: %w", err)
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

func cloneBoolPtr(v *bool) *bool {
	if v == nil {
		return nil
	}
	x := *v
	return &x
}

func cloneInt32Ptr(v *int32) *int32 {
	if v == nil {
		return nil
	}
	x := *v
	return &x
}

func cloneStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	x := *v
	return &x
}

func cloneStringSlice(v []string) []string {
	if len(v) == 0 {
		return nil
	}
	out := make([]string, len(v))
	copy(out, v)
	return out
}

func cloneStringMap(v map[string]string) map[string]string {
	if len(v) == 0 {
		return nil
	}
	out := make(map[string]string, len(v))
	for k, val := range v {
		out[k] = val
	}
	return out
}

func asAgentInput(v any) (AgentInput, bool) {
	switch x := v.(type) {
	case AgentInput:
		return x, true
	case *AgentInput:
		if x != nil {
			return *x, true
		}
	}
	return AgentInput{}, false
}

func asAskUserQuestionInput(v any) (AskUserQuestionInput, bool) {
	switch x := v.(type) {
	case AskUserQuestionInput:
		return x, true
	case *AskUserQuestionInput:
		if x != nil {
			return *x, true
		}
	}
	return AskUserQuestionInput{}, false
}

func asBashInput(v any) (BashInput, bool) {
	switch x := v.(type) {
	case BashInput:
		return x, true
	case *BashInput:
		if x != nil {
			return *x, true
		}
	}
	return BashInput{}, false
}

func asBashOutputInput(v any) (BashOutputInput, bool) {
	switch x := v.(type) {
	case BashOutputInput:
		return x, true
	case *BashOutputInput:
		if x != nil {
			return *x, true
		}
	}
	return BashOutputInput{}, false
}

func asTaskOutputInput(v any) (TaskOutputInput, bool) {
	switch x := v.(type) {
	case TaskOutputInput:
		return x, true
	case *TaskOutputInput:
		if x != nil {
			return *x, true
		}
	}
	return TaskOutputInput{}, false
}

func asFileEditInput(v any) (FileEditInput, bool) {
	switch x := v.(type) {
	case FileEditInput:
		return x, true
	case *FileEditInput:
		if x != nil {
			return *x, true
		}
	}
	return FileEditInput{}, false
}

func asFileReadInput(v any) (FileReadInput, bool) {
	switch x := v.(type) {
	case FileReadInput:
		return x, true
	case *FileReadInput:
		if x != nil {
			return *x, true
		}
	}
	return FileReadInput{}, false
}

func asFileWriteInput(v any) (FileWriteInput, bool) {
	switch x := v.(type) {
	case FileWriteInput:
		return x, true
	case *FileWriteInput:
		if x != nil {
			return *x, true
		}
	}
	return FileWriteInput{}, false
}

func asGlobInput(v any) (GlobInput, bool) {
	switch x := v.(type) {
	case GlobInput:
		return x, true
	case *GlobInput:
		if x != nil {
			return *x, true
		}
	}
	return GlobInput{}, false
}

func asGrepInput(v any) (GrepInput, bool) {
	switch x := v.(type) {
	case GrepInput:
		return x, true
	case *GrepInput:
		if x != nil {
			return *x, true
		}
	}
	return GrepInput{}, false
}

func asKillShellInput(v any) (KillShellInput, bool) {
	switch x := v.(type) {
	case KillShellInput:
		return x, true
	case *KillShellInput:
		if x != nil {
			return *x, true
		}
	}
	return KillShellInput{}, false
}

func asTaskStopInput(v any) (TaskStopInput, bool) {
	switch x := v.(type) {
	case TaskStopInput:
		return x, true
	case *TaskStopInput:
		if x != nil {
			return *x, true
		}
	}
	return TaskStopInput{}, false
}

func asNotebookEditInput(v any) (NotebookEditInput, bool) {
	switch x := v.(type) {
	case NotebookEditInput:
		return x, true
	case *NotebookEditInput:
		if x != nil {
			return *x, true
		}
	}
	return NotebookEditInput{}, false
}

func asWebFetchInput(v any) (WebFetchInput, bool) {
	switch x := v.(type) {
	case WebFetchInput:
		return x, true
	case *WebFetchInput:
		if x != nil {
			return *x, true
		}
	}
	return WebFetchInput{}, false
}

func asWebSearchInput(v any) (WebSearchInput, bool) {
	switch x := v.(type) {
	case WebSearchInput:
		return x, true
	case *WebSearchInput:
		if x != nil {
			return *x, true
		}
	}
	return WebSearchInput{}, false
}

func asTodoWriteInput(v any) (TodoWriteInput, bool) {
	switch x := v.(type) {
	case TodoWriteInput:
		return x, true
	case *TodoWriteInput:
		if x != nil {
			return *x, true
		}
	}
	return TodoWriteInput{}, false
}

func asExitPlanModeInput(v any) (ExitPlanModeInput, bool) {
	switch x := v.(type) {
	case ExitPlanModeInput:
		return x, true
	case *ExitPlanModeInput:
		if x != nil {
			return *x, true
		}
	}
	return ExitPlanModeInput{}, false
}

func asListMcpResourcesInput(v any) (ListMcpResourcesInput, bool) {
	switch x := v.(type) {
	case ListMcpResourcesInput:
		return x, true
	case *ListMcpResourcesInput:
		if x != nil {
			return *x, true
		}
	}
	return ListMcpResourcesInput{}, false
}

func asReadMcpResourceInput(v any) (ReadMcpResourceInput, bool) {
	switch x := v.(type) {
	case ReadMcpResourceInput:
		return x, true
	case *ReadMcpResourceInput:
		if x != nil {
			return *x, true
		}
	}
	return ReadMcpResourceInput{}, false
}

func asMap(v any) (map[string]any, bool) {
	switch x := v.(type) {
	case map[string]any:
		return x, true
	case *map[string]any:
		if x != nil {
			return *x, true
		}
	}
	return nil, false
}
