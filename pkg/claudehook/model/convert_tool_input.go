package model

import (
	sdktypes "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
)

func ToolInputFromProto(pb *sdktypes.ToolInput) *ToolInput {
	if pb == nil {
		return nil
	}
	m := &ToolInput{}
	switch v := pb.GetInput().(type) {
	case *sdktypes.ToolInput_Agent:
		m.Agent = AgentInputFromProto(v.Agent)
	case *sdktypes.ToolInput_AskUserQuestion:
		m.AskUserQuestion = AskUserQuestionInputFromProto(v.AskUserQuestion)
	case *sdktypes.ToolInput_Bash:
		m.Bash = BashInputFromProto(v.Bash)
	case *sdktypes.ToolInput_BashOutput:
		m.BashOutput = BashOutputInputFromProto(v.BashOutput)
	case *sdktypes.ToolInput_FileEdit:
		m.FileEdit = FileEditInputFromProto(v.FileEdit)
	case *sdktypes.ToolInput_FileRead:
		m.FileRead = FileReadInputFromProto(v.FileRead)
	case *sdktypes.ToolInput_FileWrite:
		m.FileWrite = FileWriteInputFromProto(v.FileWrite)
	case *sdktypes.ToolInput_Glob:
		m.Glob = GlobInputFromProto(v.Glob)
	case *sdktypes.ToolInput_Grep:
		m.Grep = GrepInputFromProto(v.Grep)
	case *sdktypes.ToolInput_KillShell:
		m.KillShell = KillShellInputFromProto(v.KillShell)
	case *sdktypes.ToolInput_NotebookEdit:
		m.NotebookEdit = NotebookEditInputFromProto(v.NotebookEdit)
	case *sdktypes.ToolInput_WebFetch:
		m.WebFetch = WebFetchInputFromProto(v.WebFetch)
	case *sdktypes.ToolInput_WebSearch:
		m.WebSearch = WebSearchInputFromProto(v.WebSearch)
	case *sdktypes.ToolInput_TodoWrite:
		m.TodoWrite = TodoWriteInputFromProto(v.TodoWrite)
	case *sdktypes.ToolInput_ExitPlanMode:
		m.ExitPlanMode = ExitPlanModeInputFromProto(v.ExitPlanMode)
	case *sdktypes.ToolInput_ListMcpResources:
		m.ListMcpResources = ListMcpResourcesInputFromProto(v.ListMcpResources)
	case *sdktypes.ToolInput_ReadMcpResource:
		m.ReadMcpResource = ReadMcpResourceInputFromProto(v.ReadMcpResource)
	case *sdktypes.ToolInput_McpTool:
		m.McpTool = structFromProto(v.McpTool)
	}
	return m
}

func (m *ToolInput) ToProto() *sdktypes.ToolInput {
	if m == nil {
		return nil
	}
	pb := &sdktypes.ToolInput{}
	switch {
	case m.Agent != nil:
		pb.Input = &sdktypes.ToolInput_Agent{Agent: m.Agent.ToProto()}
	case m.AskUserQuestion != nil:
		pb.Input = &sdktypes.ToolInput_AskUserQuestion{AskUserQuestion: m.AskUserQuestion.ToProto()}
	case m.Bash != nil:
		pb.Input = &sdktypes.ToolInput_Bash{Bash: m.Bash.ToProto()}
	case m.BashOutput != nil:
		pb.Input = &sdktypes.ToolInput_BashOutput{BashOutput: m.BashOutput.ToProto()}
	case m.FileEdit != nil:
		pb.Input = &sdktypes.ToolInput_FileEdit{FileEdit: m.FileEdit.ToProto()}
	case m.FileRead != nil:
		pb.Input = &sdktypes.ToolInput_FileRead{FileRead: m.FileRead.ToProto()}
	case m.FileWrite != nil:
		pb.Input = &sdktypes.ToolInput_FileWrite{FileWrite: m.FileWrite.ToProto()}
	case m.Glob != nil:
		pb.Input = &sdktypes.ToolInput_Glob{Glob: m.Glob.ToProto()}
	case m.Grep != nil:
		pb.Input = &sdktypes.ToolInput_Grep{Grep: m.Grep.ToProto()}
	case m.KillShell != nil:
		pb.Input = &sdktypes.ToolInput_KillShell{KillShell: m.KillShell.ToProto()}
	case m.NotebookEdit != nil:
		pb.Input = &sdktypes.ToolInput_NotebookEdit{NotebookEdit: m.NotebookEdit.ToProto()}
	case m.WebFetch != nil:
		pb.Input = &sdktypes.ToolInput_WebFetch{WebFetch: m.WebFetch.ToProto()}
	case m.WebSearch != nil:
		pb.Input = &sdktypes.ToolInput_WebSearch{WebSearch: m.WebSearch.ToProto()}
	case m.TodoWrite != nil:
		pb.Input = &sdktypes.ToolInput_TodoWrite{TodoWrite: m.TodoWrite.ToProto()}
	case m.ExitPlanMode != nil:
		pb.Input = &sdktypes.ToolInput_ExitPlanMode{ExitPlanMode: m.ExitPlanMode.ToProto()}
	case m.ListMcpResources != nil:
		pb.Input = &sdktypes.ToolInput_ListMcpResources{ListMcpResources: m.ListMcpResources.ToProto()}
	case m.ReadMcpResource != nil:
		pb.Input = &sdktypes.ToolInput_ReadMcpResource{ReadMcpResource: m.ReadMcpResource.ToProto()}
	case m.McpTool != nil:
		pb.Input = &sdktypes.ToolInput_McpTool{McpTool: structToProto(m.McpTool)}
	}
	return pb
}

func AgentInputFromProto(pb *sdktypes.AgentInput) *AgentInput {
	if pb == nil {
		return nil
	}
	return &AgentInput{
		Description:  pb.GetDescription(),
		Prompt:       pb.GetPrompt(),
		SubagentType: pb.GetSubagentType(),
	}
}

func (m *AgentInput) ToProto() *sdktypes.AgentInput {
	if m == nil {
		return nil
	}
	return &sdktypes.AgentInput{
		Description:  m.Description,
		Prompt:       m.Prompt,
		SubagentType: m.SubagentType,
	}
}

func QuestionOptionFromProto(pb *sdktypes.QuestionOption) *QuestionOption {
	if pb == nil {
		return nil
	}
	return &QuestionOption{
		Label:       pb.GetLabel(),
		Description: pb.GetDescription(),
	}
}

func (m *QuestionOption) ToProto() *sdktypes.QuestionOption {
	if m == nil {
		return nil
	}
	return &sdktypes.QuestionOption{
		Label:       m.Label,
		Description: m.Description,
	}
}

func questionOptionsFromProto(pbs []*sdktypes.QuestionOption) []*QuestionOption {
	if pbs == nil {
		return nil
	}
	result := make([]*QuestionOption, len(pbs))
	for i, pb := range pbs {
		result[i] = QuestionOptionFromProto(pb)
	}
	return result
}

func questionOptionsToProto(ms []*QuestionOption) []*sdktypes.QuestionOption {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.QuestionOption, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func QuestionFromProto(pb *sdktypes.Question) *Question {
	if pb == nil {
		return nil
	}
	return &Question{
		Question:    pb.GetQuestion(),
		Header:      pb.GetHeader(),
		Options:     questionOptionsFromProto(pb.GetOptions()),
		MultiSelect: pb.GetMultiSelect(),
	}
}

func (m *Question) ToProto() *sdktypes.Question {
	if m == nil {
		return nil
	}
	return &sdktypes.Question{
		Question:    m.Question,
		Header:      m.Header,
		Options:     questionOptionsToProto(m.Options),
		MultiSelect: m.MultiSelect,
	}
}

func questionsFromProto(pbs []*sdktypes.Question) []*Question {
	if pbs == nil {
		return nil
	}
	result := make([]*Question, len(pbs))
	for i, pb := range pbs {
		result[i] = QuestionFromProto(pb)
	}
	return result
}

func questionsToProto(ms []*Question) []*sdktypes.Question {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.Question, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func AskUserQuestionInputFromProto(pb *sdktypes.AskUserQuestionInput) *AskUserQuestionInput {
	if pb == nil {
		return nil
	}
	return &AskUserQuestionInput{
		Questions: questionsFromProto(pb.GetQuestions()),
		Answers:   pb.GetAnswers(),
	}
}

func (m *AskUserQuestionInput) ToProto() *sdktypes.AskUserQuestionInput {
	if m == nil {
		return nil
	}
	return &sdktypes.AskUserQuestionInput{
		Questions: questionsToProto(m.Questions),
		Answers:   m.Answers,
	}
}

func BashInputFromProto(pb *sdktypes.BashInput) *BashInput {
	if pb == nil {
		return nil
	}
	return &BashInput{
		Command:         pb.GetCommand(),
		Timeout:         optionalInt32FromProto(pb.Timeout),
		Description:     optionalStringFromProto(pb.Description),
		RunInBackground: optionalBoolFromProto(pb.RunInBackground),
	}
}

func (m *BashInput) ToProto() *sdktypes.BashInput {
	if m == nil {
		return nil
	}
	return &sdktypes.BashInput{
		Command:         m.Command,
		Timeout:         optionalInt32ToProto(m.Timeout),
		Description:     optionalStringToProto(m.Description),
		RunInBackground: optionalBoolToProto(m.RunInBackground),
	}
}

func BashOutputInputFromProto(pb *sdktypes.BashOutputInput) *BashOutputInput {
	if pb == nil {
		return nil
	}
	return &BashOutputInput{
		BashID: pb.GetBashId(),
		Filter: optionalStringFromProto(pb.Filter),
	}
}

func (m *BashOutputInput) ToProto() *sdktypes.BashOutputInput {
	if m == nil {
		return nil
	}
	return &sdktypes.BashOutputInput{
		BashId: m.BashID,
		Filter: optionalStringToProto(m.Filter),
	}
}

func FileEditInputFromProto(pb *sdktypes.FileEditInput) *FileEditInput {
	if pb == nil {
		return nil
	}
	return &FileEditInput{
		FilePath:   pb.GetFilePath(),
		OldString:  pb.GetOldString(),
		NewString:  pb.GetNewString(),
		ReplaceAll: optionalBoolFromProto(pb.ReplaceAll),
	}
}

func (m *FileEditInput) ToProto() *sdktypes.FileEditInput {
	if m == nil {
		return nil
	}
	return &sdktypes.FileEditInput{
		FilePath:   m.FilePath,
		OldString:  m.OldString,
		NewString:  m.NewString,
		ReplaceAll: optionalBoolToProto(m.ReplaceAll),
	}
}

func FileReadInputFromProto(pb *sdktypes.FileReadInput) *FileReadInput {
	if pb == nil {
		return nil
	}
	return &FileReadInput{
		FilePath: pb.GetFilePath(),
		Offset:   optionalInt32FromProto(pb.Offset),
		Limit:    optionalInt32FromProto(pb.Limit),
	}
}

func (m *FileReadInput) ToProto() *sdktypes.FileReadInput {
	if m == nil {
		return nil
	}
	return &sdktypes.FileReadInput{
		FilePath: m.FilePath,
		Offset:   optionalInt32ToProto(m.Offset),
		Limit:    optionalInt32ToProto(m.Limit),
	}
}

func FileWriteInputFromProto(pb *sdktypes.FileWriteInput) *FileWriteInput {
	if pb == nil {
		return nil
	}
	return &FileWriteInput{
		FilePath: pb.GetFilePath(),
		Content:  pb.GetContent(),
	}
}

func (m *FileWriteInput) ToProto() *sdktypes.FileWriteInput {
	if m == nil {
		return nil
	}
	return &sdktypes.FileWriteInput{
		FilePath: m.FilePath,
		Content:  m.Content,
	}
}

func GlobInputFromProto(pb *sdktypes.GlobInput) *GlobInput {
	if pb == nil {
		return nil
	}
	return &GlobInput{
		Pattern: pb.GetPattern(),
		Path:    optionalStringFromProto(pb.Path),
	}
}

func (m *GlobInput) ToProto() *sdktypes.GlobInput {
	if m == nil {
		return nil
	}
	return &sdktypes.GlobInput{
		Pattern: m.Pattern,
		Path:    optionalStringToProto(m.Path),
	}
}

func GrepInputFromProto(pb *sdktypes.GrepInput) *GrepInput {
	if pb == nil {
		return nil
	}
	return &GrepInput{
		Pattern:         pb.GetPattern(),
		Path:            optionalStringFromProto(pb.Path),
		Glob:            optionalStringFromProto(pb.Glob),
		Type:            optionalStringFromProto(pb.Type),
		OutputMode:      GrepOutputModeFromProto(pb.GetOutputMode()),
		CaseInsensitive: optionalBoolFromProto(pb.CaseInsensitive),
		ShowLineNumbers: optionalBoolFromProto(pb.ShowLineNumbers),
		BeforeContext:   optionalInt32FromProto(pb.BeforeContext),
		AfterContext:    optionalInt32FromProto(pb.AfterContext),
		Context:         optionalInt32FromProto(pb.Context),
		HeadLimit:       optionalInt32FromProto(pb.HeadLimit),
		Multiline:       optionalBoolFromProto(pb.Multiline),
	}
}

func (m *GrepInput) ToProto() *sdktypes.GrepInput {
	if m == nil {
		return nil
	}
	return &sdktypes.GrepInput{
		Pattern:         m.Pattern,
		Path:            optionalStringToProto(m.Path),
		Glob:            optionalStringToProto(m.Glob),
		Type:            optionalStringToProto(m.Type),
		OutputMode:      m.OutputMode.ToProto(),
		CaseInsensitive: optionalBoolToProto(m.CaseInsensitive),
		ShowLineNumbers: optionalBoolToProto(m.ShowLineNumbers),
		BeforeContext:   optionalInt32ToProto(m.BeforeContext),
		AfterContext:    optionalInt32ToProto(m.AfterContext),
		Context:         optionalInt32ToProto(m.Context),
		HeadLimit:       optionalInt32ToProto(m.HeadLimit),
		Multiline:       optionalBoolToProto(m.Multiline),
	}
}

func KillShellInputFromProto(pb *sdktypes.KillShellInput) *KillShellInput {
	if pb == nil {
		return nil
	}
	return &KillShellInput{
		ShellID: pb.GetShellId(),
	}
}

func (m *KillShellInput) ToProto() *sdktypes.KillShellInput {
	if m == nil {
		return nil
	}
	return &sdktypes.KillShellInput{
		ShellId: m.ShellID,
	}
}

func NotebookEditInputFromProto(pb *sdktypes.NotebookEditInput) *NotebookEditInput {
	if pb == nil {
		return nil
	}
	return &NotebookEditInput{
		NotebookPath: pb.GetNotebookPath(),
		CellID:       optionalStringFromProto(pb.CellId),
		NewSource:    pb.GetNewSource(),
		CellType:     NotebookCellTypeFromProto(pb.GetCellType()),
		EditMode:     NotebookEditModeFromProto(pb.GetEditMode()),
	}
}

func (m *NotebookEditInput) ToProto() *sdktypes.NotebookEditInput {
	if m == nil {
		return nil
	}
	return &sdktypes.NotebookEditInput{
		NotebookPath: m.NotebookPath,
		CellId:       optionalStringToProto(m.CellID),
		NewSource:    m.NewSource,
		CellType:     m.CellType.ToProto(),
		EditMode:     m.EditMode.ToProto(),
	}
}

func WebFetchInputFromProto(pb *sdktypes.WebFetchInput) *WebFetchInput {
	if pb == nil {
		return nil
	}
	return &WebFetchInput{
		URL:    pb.GetUrl(),
		Prompt: pb.GetPrompt(),
	}
}

func (m *WebFetchInput) ToProto() *sdktypes.WebFetchInput {
	if m == nil {
		return nil
	}
	return &sdktypes.WebFetchInput{
		Url:    m.URL,
		Prompt: m.Prompt,
	}
}

func WebSearchInputFromProto(pb *sdktypes.WebSearchInput) *WebSearchInput {
	if pb == nil {
		return nil
	}
	return &WebSearchInput{
		Query:          pb.GetQuery(),
		AllowedDomains: pb.GetAllowedDomains(),
		BlockedDomains: pb.GetBlockedDomains(),
	}
}

func (m *WebSearchInput) ToProto() *sdktypes.WebSearchInput {
	if m == nil {
		return nil
	}
	return &sdktypes.WebSearchInput{
		Query:          m.Query,
		AllowedDomains: m.AllowedDomains,
		BlockedDomains: m.BlockedDomains,
	}
}

func TodoItemFromProto(pb *sdktypes.TodoItem) *TodoItem {
	if pb == nil {
		return nil
	}
	return &TodoItem{
		Content:    pb.GetContent(),
		Status:     TodoStatusFromProto(pb.GetStatus()),
		ActiveForm: pb.GetActiveForm(),
	}
}

func (m *TodoItem) ToProto() *sdktypes.TodoItem {
	if m == nil {
		return nil
	}
	return &sdktypes.TodoItem{
		Content:    m.Content,
		Status:     m.Status.ToProto(),
		ActiveForm: m.ActiveForm,
	}
}

func todoItemsFromProto(pbs []*sdktypes.TodoItem) []*TodoItem {
	if pbs == nil {
		return nil
	}
	result := make([]*TodoItem, len(pbs))
	for i, pb := range pbs {
		result[i] = TodoItemFromProto(pb)
	}
	return result
}

func todoItemsToProto(ms []*TodoItem) []*sdktypes.TodoItem {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.TodoItem, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func TodoWriteInputFromProto(pb *sdktypes.TodoWriteInput) *TodoWriteInput {
	if pb == nil {
		return nil
	}
	return &TodoWriteInput{
		Todos: todoItemsFromProto(pb.GetTodos()),
	}
}

func (m *TodoWriteInput) ToProto() *sdktypes.TodoWriteInput {
	if m == nil {
		return nil
	}
	return &sdktypes.TodoWriteInput{
		Todos: todoItemsToProto(m.Todos),
	}
}

func ExitPlanModeInputFromProto(pb *sdktypes.ExitPlanModeInput) *ExitPlanModeInput {
	if pb == nil {
		return nil
	}
	return &ExitPlanModeInput{
		Plan: pb.GetPlan(),
	}
}

func (m *ExitPlanModeInput) ToProto() *sdktypes.ExitPlanModeInput {
	if m == nil {
		return nil
	}
	return &sdktypes.ExitPlanModeInput{
		Plan: m.Plan,
	}
}

func ListMcpResourcesInputFromProto(pb *sdktypes.ListMcpResourcesInput) *ListMcpResourcesInput {
	if pb == nil {
		return nil
	}
	return &ListMcpResourcesInput{
		Server: optionalStringFromProto(pb.Server),
	}
}

func (m *ListMcpResourcesInput) ToProto() *sdktypes.ListMcpResourcesInput {
	if m == nil {
		return nil
	}
	return &sdktypes.ListMcpResourcesInput{
		Server: optionalStringToProto(m.Server),
	}
}

func ReadMcpResourceInputFromProto(pb *sdktypes.ReadMcpResourceInput) *ReadMcpResourceInput {
	if pb == nil {
		return nil
	}
	return &ReadMcpResourceInput{
		Server: pb.GetServer(),
		URI:    pb.GetUri(),
	}
}

func (m *ReadMcpResourceInput) ToProto() *sdktypes.ReadMcpResourceInput {
	if m == nil {
		return nil
	}
	return &sdktypes.ReadMcpResourceInput{
		Server: m.Server,
		Uri:    m.URI,
	}
}
