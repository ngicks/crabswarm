package models

import (
	"encoding/json"
	"fmt"
)

type ToolOutputSchemas interface {
	toolOutputSchemas()
	json.Marshaler
	json.Unmarshaler
}

// As per https://platform.claude.com/docs/en/agent-sdk/typescript#tool-output-schemas
/*
type ToolOutputSchemas =
  | AgentOutput
  | AskUserQuestionOutput
  | BashOutput
  | ConfigOutput
  | EnterWorktreeOutput
  | ExitPlanModeOutput
  | FileEditOutput
  | FileReadOutput
  | FileWriteOutput
  | GlobOutput
  | GrepOutput
  | ListMcpResourcesOutput
  | NotebookEditOutput
  | ReadMcpResourceOutput
  | TaskStopOutput
  | TodoWriteOutput
  | WebFetchOutput
  | WebSearchOutput;
*/

// fallback target
func (*UnknownOutput) toolOutputSchemas() {}

func (*AgentOutputCompleted) toolOutputSchemas()        {}
func (*AgentOutputAsyncLaunched) toolOutputSchemas()    {}
func (*AgentOutputSubAgentEntered) toolOutputSchemas()  {}
func (*AgentOutputUnknown) toolOutputSchemas()          {}
func (*AskUserQuestionOutput) toolOutputSchemas()       {}
func (*BashOutput) toolOutputSchemas()                  {}
func (*ConfigOutput) toolOutputSchemas()                {}
func (*EnterWorktreeOutput) toolOutputSchemas()         {}
func (*ExitPlanModeOutput) toolOutputSchemas()          {}
func (*FileEditOutput) toolOutputSchemas()              {}
func (*FileReadOutputText) toolOutputSchemas()          {}
func (*FileReadOutputImage) toolOutputSchemas()         {}
func (*FileReadOutputNotebook) toolOutputSchemas()      {}
func (*FileReadOutputPdf) toolOutputSchemas()           {}
func (*FileReadOutputParts) toolOutputSchemas()         {}
func (*FileReadOutputUnknown) toolOutputSchemas()       {}
func (*FileWriteOutput) toolOutputSchemas()             {}
func (*GlobOutput) toolOutputSchemas()                  {}
func (*GrepOutput) toolOutputSchemas()                  {}
func (*ListMcpResourcesOutput) toolOutputSchemas()      {}
func (*NotebookEditOutput) toolOutputSchemas()          {}
func (*ReadMcpResourceOutput) toolOutputSchemas()       {}
func (*TaskStopOutput) toolOutputSchemas()              {}
func (*TodoWriteOutput) toolOutputSchemas()             {}
func (*WebFetchOutput) toolOutputSchemas()              {}
func (*WebSearchOutput) toolOutputSchemas()             {}

func unmarshalToolOutputSchemas(toolName string, data []byte) (_ ToolOutputSchemas, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("ToolOutputSchemas: %w", err)
		}
	}()

	switch toolName {
	default:
		var v UnknownOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	case "":
		return nil, fmt.Errorf("empty tool_name")
	case "Task":
		v, err := unmarshalAgentOutput(data)
		if err != nil {
			return nil, err
		}
		return v, nil
	case "AskUserQuestion":
		var v AskUserQuestionOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	case "Bash":
		var v BashOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	case "Edit":
		var v FileEditOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	case "Read":
		v, err := unmarshalFileReadOutput(data)
		if err != nil {
			return nil, err
		}
		return v, nil
	case "Write":
		var v FileWriteOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	case "Glob":
		var v GlobOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	case "Grep":
		var v GrepOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	case "TaskStop":
		var v TaskStopOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	case "NotebookEdit":
		var v NotebookEditOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	case "WebFetch":
		var v WebFetchOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	case "WebSearch":
		var v WebSearchOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	case "TodoWrite":
		var v TodoWriteOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	case "ExitPlanMode":
		var v ExitPlanModeOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	case "ListMcpResources":
		var v ListMcpResourcesOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	case "ReadMcpResource":
		var v ReadMcpResourceOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	case "Config":
		var v ConfigOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	case "EnterWorktree":
		var v EnterWorktreeOutput
		err := json.Unmarshal(data, &v)
		return &v, err
	}
}

// UnknownOutput is a fallback type for unknown tool outputs.
type UnknownOutput map[string]json.RawMessage

func (o UnknownOutput) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]json.RawMessage(o))
}

func (o *UnknownOutput) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*map[string]json.RawMessage)(o))
}

// AgentOutputUnknown is a fallback type for unknown AgentOutput variants.
type AgentOutputUnknown map[string]json.RawMessage

func (o AgentOutputUnknown) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]json.RawMessage(o))
}

func (o *AgentOutputUnknown) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*map[string]json.RawMessage)(o))
}

// FileReadOutputUnknown is a fallback type for unknown FileReadOutput variants.
type FileReadOutputUnknown map[string]json.RawMessage

func (o FileReadOutputUnknown) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]json.RawMessage(o))
}

func (o *FileReadOutputUnknown) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*map[string]json.RawMessage)(o))
}

// AgentOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#task-2
// Tool name: "Task".
type AgentOutput interface {
	agentOutput()
	ToolOutputSchemas
}

// fallback target
func (*AgentOutputUnknown) agentOutput() {}

func (*AgentOutputCompleted) agentOutput()       {}
func (*AgentOutputAsyncLaunched) agentOutput()   {}
func (*AgentOutputSubAgentEntered) agentOutput() {}

func unmarshalAgentOutput(data []byte) (_ AgentOutput, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("AgentOutput: %w", err)
		}
	}()

	var d struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	switch d.Status {
	default:
		var v AgentOutputUnknown
		err := json.Unmarshal(data, &v)
		return &v, err
	case "":
		return nil, fmt.Errorf("empty status")
	case "completed":
		var v AgentOutputCompleted
		err := json.Unmarshal(data, &v)
		return &v, err
	case "async_launched":
		var v AgentOutputAsyncLaunched
		err := json.Unmarshal(data, &v)
		return &v, err
	case "sub_agent_entered":
		var v AgentOutputSubAgentEntered
		err := json.Unmarshal(data, &v)
		return &v, err
	}
}

// AgentOutputContentBlock represents a content block in AgentOutputCompleted.
type AgentOutputContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AgentOutputServerToolUse represents server tool usage stats.
type AgentOutputServerToolUse struct {
	WebSearchRequests json.Number `json:"web_search_requests"`
	WebFetchRequests  json.Number `json:"web_fetch_requests"`
}

// AgentOutputCacheCreation represents cache creation stats.
type AgentOutputCacheCreation struct {
	Ephemeral1hInputTokens json.Number `json:"ephemeral_1h_input_tokens"`
	Ephemeral5mInputTokens json.Number `json:"ephemeral_5m_input_tokens"`
}

// AgentOutputUsage represents usage statistics for an agent run.
type AgentOutputUsage struct {
	InputTokens                json.Number               `json:"input_tokens"`
	OutputTokens               json.Number               `json:"output_tokens"`
	CacheCreationInputTokens   *json.Number              `json:"cache_creation_input_tokens"`
	CacheReadInputTokens       *json.Number              `json:"cache_read_input_tokens"`
	ServerToolUse              *AgentOutputServerToolUse  `json:"server_tool_use"`
	ServiceTier                *string                    `json:"service_tier"`
	CacheCreation              *AgentOutputCacheCreation  `json:"cache_creation"`
}

// AgentOutputCompleted is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#task-2
type AgentOutputCompleted struct {
	Status            string                    `json:"status"`
	AgentID           string                    `json:"agentId"`
	Content           []AgentOutputContentBlock `json:"content"`
	TotalToolUseCount json.Number               `json:"totalToolUseCount"`
	TotalDurationMs   json.Number               `json:"totalDurationMs"`
	TotalTokens       json.Number               `json:"totalTokens"`
	Usage             AgentOutputUsage          `json:"usage"`
	Prompt            string                    `json:"prompt"`
}

func (o AgentOutputCompleted) MarshalJSON() ([]byte, error) {
	type plain AgentOutputCompleted
	o.Status = "completed"
	return json.Marshal(plain(o))
}

func (o *AgentOutputCompleted) UnmarshalJSON(data []byte) error {
	type plain AgentOutputCompleted
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = AgentOutputCompleted(p)
	o.Status = "completed"
	return nil
}

// AgentOutputAsyncLaunched is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#task-2
type AgentOutputAsyncLaunched struct {
	Status             string `json:"status"`
	AgentID            string `json:"agentId"`
	Description        string `json:"description"`
	Prompt             string `json:"prompt"`
	OutputFile         string `json:"outputFile"`
	CanReadOutputFile  *bool  `json:"canReadOutputFile,omitempty"`
}

func (o AgentOutputAsyncLaunched) MarshalJSON() ([]byte, error) {
	type plain AgentOutputAsyncLaunched
	o.Status = "async_launched"
	return json.Marshal(plain(o))
}

func (o *AgentOutputAsyncLaunched) UnmarshalJSON(data []byte) error {
	type plain AgentOutputAsyncLaunched
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = AgentOutputAsyncLaunched(p)
	o.Status = "async_launched"
	return nil
}

// AgentOutputSubAgentEntered is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#task-2
type AgentOutputSubAgentEntered struct {
	Status      string `json:"status"`
	Description string `json:"description"`
	Message     string `json:"message"`
}

func (o AgentOutputSubAgentEntered) MarshalJSON() ([]byte, error) {
	type plain AgentOutputSubAgentEntered
	o.Status = "sub_agent_entered"
	return json.Marshal(plain(o))
}

func (o *AgentOutputSubAgentEntered) UnmarshalJSON(data []byte) error {
	type plain AgentOutputSubAgentEntered
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = AgentOutputSubAgentEntered(p)
	o.Status = "sub_agent_entered"
	return nil
}

// AskUserQuestionOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#ask-user-question-2
// Tool name: AskUserQuestion
type AskUserQuestionOutput struct {
	Questions []AskUserQuestionOutputQuestion `json:"questions"`
	Answers   map[string]string               `json:"answers"`
}

func (o AskUserQuestionOutput) MarshalJSON() ([]byte, error) {
	type plain AskUserQuestionOutput
	return json.Marshal(plain(o))
}

func (o *AskUserQuestionOutput) UnmarshalJSON(data []byte) error {
	type plain AskUserQuestionOutput
	return json.Unmarshal(data, (*plain)(o))
}

// AskUserQuestionOutputQuestion represents a single question in AskUserQuestionOutput.
type AskUserQuestionOutputQuestion struct {
	Question    string                              `json:"question"`
	Header      string                              `json:"header"`
	Options     []AskUserQuestionOutputQuestionOption `json:"options"`
	MultiSelect bool                                `json:"multiSelect"`
}

// AskUserQuestionOutputQuestionOption represents an option for a question.
type AskUserQuestionOutputQuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// BashOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#bash-2
// Tool name: Bash
type BashOutput struct {
	Stdout                  string             `json:"stdout"`
	Stderr                  string             `json:"stderr"`
	RawOutputPath           *string            `json:"rawOutputPath,omitempty"`
	Interrupted             bool               `json:"interrupted"`
	IsImage                 *bool              `json:"isImage,omitempty"`
	BackgroundTaskID        *string            `json:"backgroundTaskId,omitempty"`
	BackgroundedByUser      *bool              `json:"backgroundedByUser,omitempty"`
	DangerouslyDisableSandbox *bool            `json:"dangerouslyDisableSandbox,omitempty"`
	ReturnCodeInterpretation *string           `json:"returnCodeInterpretation,omitempty"`
	StructuredContent       []json.RawMessage  `json:"structuredContent,omitempty"`
	PersistedOutputPath     *string            `json:"persistedOutputPath,omitempty"`
	PersistedOutputSize     *json.Number       `json:"persistedOutputSize,omitempty"`
}

func (o BashOutput) MarshalJSON() ([]byte, error) {
	type plain BashOutput
	return json.Marshal(plain(o))
}

func (o *BashOutput) UnmarshalJSON(data []byte) error {
	type plain BashOutput
	return json.Unmarshal(data, (*plain)(o))
}

// StructuredPatchHunk represents a hunk in a structured patch.
type StructuredPatchHunk struct {
	OldStart json.Number `json:"oldStart"`
	OldLines json.Number `json:"oldLines"`
	NewStart json.Number `json:"newStart"`
	NewLines json.Number `json:"newLines"`
	Lines    []string    `json:"lines"`
}

// GitDiff represents a git diff for a file.
type GitDiff struct {
	Filename  string      `json:"filename"`
	Status    string      `json:"status"`
	Additions json.Number `json:"additions"`
	Deletions json.Number `json:"deletions"`
	Changes   json.Number `json:"changes"`
	Patch     string      `json:"patch"`
}

// FileEditOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#edit-2
// Tool name: Edit
type FileEditOutput struct {
	FilePath        string                `json:"filePath"`
	OldString       string                `json:"oldString"`
	NewString       string                `json:"newString"`
	OriginalFile    string                `json:"originalFile"`
	StructuredPatch []StructuredPatchHunk `json:"structuredPatch"`
	UserModified    bool                  `json:"userModified"`
	ReplaceAll      bool                  `json:"replaceAll"`
	GitDiff         *GitDiff              `json:"gitDiff,omitempty"`
}

func (o FileEditOutput) MarshalJSON() ([]byte, error) {
	type plain FileEditOutput
	return json.Marshal(plain(o))
}

func (o *FileEditOutput) UnmarshalJSON(data []byte) error {
	type plain FileEditOutput
	return json.Unmarshal(data, (*plain)(o))
}

// FileReadOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#read-2
// Tool name: Read
type FileReadOutput interface {
	fileReadOutput()
	ToolOutputSchemas
}

// fallback target
func (*FileReadOutputUnknown) fileReadOutput() {}

func (*FileReadOutputText) fileReadOutput()     {}
func (*FileReadOutputImage) fileReadOutput()    {}
func (*FileReadOutputNotebook) fileReadOutput() {}
func (*FileReadOutputPdf) fileReadOutput()      {}
func (*FileReadOutputParts) fileReadOutput()    {}

func unmarshalFileReadOutput(data []byte) (_ FileReadOutput, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("FileReadOutput: %w", err)
		}
	}()

	var d struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	switch d.Type {
	default:
		var v FileReadOutputUnknown
		err := json.Unmarshal(data, &v)
		return &v, err
	case "":
		return nil, fmt.Errorf("empty type")
	case "text":
		var v FileReadOutputText
		err := json.Unmarshal(data, &v)
		return &v, err
	case "image":
		var v FileReadOutputImage
		err := json.Unmarshal(data, &v)
		return &v, err
	case "notebook":
		var v FileReadOutputNotebook
		err := json.Unmarshal(data, &v)
		return &v, err
	case "pdf":
		var v FileReadOutputPdf
		err := json.Unmarshal(data, &v)
		return &v, err
	case "parts":
		var v FileReadOutputParts
		err := json.Unmarshal(data, &v)
		return &v, err
	}
}

// FileReadOutputTextFile represents the file data in FileReadOutputText.
type FileReadOutputTextFile struct {
	FilePath   string      `json:"filePath"`
	Content    string      `json:"content"`
	NumLines   json.Number `json:"numLines"`
	StartLine  json.Number `json:"startLine"`
	TotalLines json.Number `json:"totalLines"`
}

// FileReadOutputText is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputText struct {
	Type string                 `json:"type"`
	File FileReadOutputTextFile `json:"file"`
}

func (o FileReadOutputText) MarshalJSON() ([]byte, error) {
	type plain FileReadOutputText
	o.Type = "text"
	return json.Marshal(plain(o))
}

func (o *FileReadOutputText) UnmarshalJSON(data []byte) error {
	type plain FileReadOutputText
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = FileReadOutputText(p)
	o.Type = "text"
	return nil
}

// FileReadOutputImageDimensions represents image dimension information.
type FileReadOutputImageDimensions struct {
	OriginalWidth  *json.Number `json:"originalWidth,omitempty"`
	OriginalHeight *json.Number `json:"originalHeight,omitempty"`
	DisplayWidth   *json.Number `json:"displayWidth,omitempty"`
	DisplayHeight  *json.Number `json:"displayHeight,omitempty"`
}

// FileReadOutputImageFile represents the file data in FileReadOutputImage.
type FileReadOutputImageFile struct {
	Base64       string                         `json:"base64"`
	Type         string                         `json:"type"`
	OriginalSize json.Number                    `json:"originalSize"`
	Dimensions   *FileReadOutputImageDimensions `json:"dimensions,omitempty"`
}

// FileReadOutputImage is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputImage struct {
	Type string                  `json:"type"`
	File FileReadOutputImageFile `json:"file"`
}

func (o FileReadOutputImage) MarshalJSON() ([]byte, error) {
	type plain FileReadOutputImage
	o.Type = "image"
	return json.Marshal(plain(o))
}

func (o *FileReadOutputImage) UnmarshalJSON(data []byte) error {
	type plain FileReadOutputImage
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = FileReadOutputImage(p)
	o.Type = "image"
	return nil
}

// FileReadOutputNotebookFile represents the file data in FileReadOutputNotebook.
type FileReadOutputNotebookFile struct {
	FilePath string            `json:"filePath"`
	Cells    []json.RawMessage `json:"cells"`
}

// FileReadOutputNotebook is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputNotebook struct {
	Type string                     `json:"type"`
	File FileReadOutputNotebookFile `json:"file"`
}

func (o FileReadOutputNotebook) MarshalJSON() ([]byte, error) {
	type plain FileReadOutputNotebook
	o.Type = "notebook"
	return json.Marshal(plain(o))
}

func (o *FileReadOutputNotebook) UnmarshalJSON(data []byte) error {
	type plain FileReadOutputNotebook
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = FileReadOutputNotebook(p)
	o.Type = "notebook"
	return nil
}

// FileReadOutputPdfFile represents the file data in FileReadOutputPdf.
type FileReadOutputPdfFile struct {
	FilePath     string      `json:"filePath"`
	Base64       string      `json:"base64"`
	OriginalSize json.Number `json:"originalSize"`
}

// FileReadOutputPdf is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputPdf struct {
	Type string                `json:"type"`
	File FileReadOutputPdfFile `json:"file"`
}

func (o FileReadOutputPdf) MarshalJSON() ([]byte, error) {
	type plain FileReadOutputPdf
	o.Type = "pdf"
	return json.Marshal(plain(o))
}

func (o *FileReadOutputPdf) UnmarshalJSON(data []byte) error {
	type plain FileReadOutputPdf
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = FileReadOutputPdf(p)
	o.Type = "pdf"
	return nil
}

// FileReadOutputPartsFile represents the file data in FileReadOutputParts.
type FileReadOutputPartsFile struct {
	FilePath     string      `json:"filePath"`
	OriginalSize json.Number `json:"originalSize"`
	Count        json.Number `json:"count"`
	OutputDir    string      `json:"outputDir"`
}

// FileReadOutputParts is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputParts struct {
	Type string                  `json:"type"`
	File FileReadOutputPartsFile `json:"file"`
}

func (o FileReadOutputParts) MarshalJSON() ([]byte, error) {
	type plain FileReadOutputParts
	o.Type = "parts"
	return json.Marshal(plain(o))
}

func (o *FileReadOutputParts) UnmarshalJSON(data []byte) error {
	type plain FileReadOutputParts
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = FileReadOutputParts(p)
	o.Type = "parts"
	return nil
}

// FileWriteOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#write-2
// Tool name: Write
type FileWriteOutput struct {
	Type            string                `json:"type"`
	FilePath        string                `json:"filePath"`
	Content         string                `json:"content"`
	StructuredPatch []StructuredPatchHunk `json:"structuredPatch"`
	OriginalFile    *string               `json:"originalFile"`
	GitDiff         *GitDiff              `json:"gitDiff,omitempty"`
}

func (o FileWriteOutput) MarshalJSON() ([]byte, error) {
	type plain FileWriteOutput
	return json.Marshal(plain(o))
}

func (o *FileWriteOutput) UnmarshalJSON(data []byte) error {
	type plain FileWriteOutput
	return json.Unmarshal(data, (*plain)(o))
}

// GlobOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#glob-2
// Tool name: Glob
type GlobOutput struct {
	DurationMs json.Number `json:"durationMs"`
	NumFiles   json.Number `json:"numFiles"`
	Filenames  []string    `json:"filenames"`
	Truncated  bool        `json:"truncated"`
}

func (o GlobOutput) MarshalJSON() ([]byte, error) {
	type plain GlobOutput
	return json.Marshal(plain(o))
}

func (o *GlobOutput) UnmarshalJSON(data []byte) error {
	type plain GlobOutput
	return json.Unmarshal(data, (*plain)(o))
}

// GrepOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#grep-2
// Tool name: Grep
type GrepOutput struct {
	Mode          *string     `json:"mode,omitempty"`
	NumFiles      json.Number `json:"numFiles"`
	Filenames     []string    `json:"filenames"`
	Content       *string     `json:"content,omitempty"`
	NumLines      *json.Number `json:"numLines,omitempty"`
	NumMatches    *json.Number `json:"numMatches,omitempty"`
	AppliedLimit  *json.Number `json:"appliedLimit,omitempty"`
	AppliedOffset *json.Number `json:"appliedOffset,omitempty"`
}

func (o GrepOutput) MarshalJSON() ([]byte, error) {
	type plain GrepOutput
	return json.Marshal(plain(o))
}

func (o *GrepOutput) UnmarshalJSON(data []byte) error {
	type plain GrepOutput
	return json.Unmarshal(data, (*plain)(o))
}

// TaskStopOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#task-stop-2
// Tool name: TaskStop
type TaskStopOutput struct {
	Message  string  `json:"message"`
	TaskID   string  `json:"task_id"`
	TaskType string  `json:"task_type"`
	Command  *string `json:"command,omitempty"`
}

func (o TaskStopOutput) MarshalJSON() ([]byte, error) {
	type plain TaskStopOutput
	return json.Marshal(plain(o))
}

func (o *TaskStopOutput) UnmarshalJSON(data []byte) error {
	type plain TaskStopOutput
	return json.Unmarshal(data, (*plain)(o))
}

// NotebookEditOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#notebook-edit-2
// Tool name: NotebookEdit
type NotebookEditOutput struct {
	NewSource    string  `json:"new_source"`
	CellID       *string `json:"cell_id,omitempty"`
	CellType     string  `json:"cell_type"`
	Language     string  `json:"language"`
	EditMode     string  `json:"edit_mode"`
	Error        *string `json:"error,omitempty"`
	NotebookPath string  `json:"notebook_path"`
	OriginalFile string  `json:"original_file"`
	UpdatedFile  string  `json:"updated_file"`
}

func (o NotebookEditOutput) MarshalJSON() ([]byte, error) {
	type plain NotebookEditOutput
	return json.Marshal(plain(o))
}

func (o *NotebookEditOutput) UnmarshalJSON(data []byte) error {
	type plain NotebookEditOutput
	return json.Unmarshal(data, (*plain)(o))
}

// WebFetchOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#web-fetch-2
// Tool name: WebFetch
type WebFetchOutput struct {
	Bytes      json.Number `json:"bytes"`
	Code       json.Number `json:"code"`
	CodeText   string      `json:"codeText"`
	Result     string      `json:"result"`
	DurationMs json.Number `json:"durationMs"`
	URL        string      `json:"url"`
}

func (o WebFetchOutput) MarshalJSON() ([]byte, error) {
	type plain WebFetchOutput
	return json.Marshal(plain(o))
}

func (o *WebFetchOutput) UnmarshalJSON(data []byte) error {
	type plain WebFetchOutput
	return json.Unmarshal(data, (*plain)(o))
}

// WebSearchOutputResultEntry represents a single entry in a search result's content.
type WebSearchOutputResultEntry struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// WebSearchOutputResultToolUse represents a tool-use based search result.
type WebSearchOutputResultToolUse struct {
	ToolUseID string                       `json:"tool_use_id"`
	Content   []WebSearchOutputResultEntry `json:"content"`
}

// WebSearchOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#web-search-2
// Tool name: WebSearch
type WebSearchOutput struct {
	Query           string            `json:"query"`
	Results         []json.RawMessage `json:"results"`
	DurationSeconds json.Number       `json:"durationSeconds"`
}

func (o WebSearchOutput) MarshalJSON() ([]byte, error) {
	type plain WebSearchOutput
	return json.Marshal(plain(o))
}

func (o *WebSearchOutput) UnmarshalJSON(data []byte) error {
	type plain WebSearchOutput
	return json.Unmarshal(data, (*plain)(o))
}

// TodoWriteOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#todo-write-2
// Tool name: TodoWrite
type TodoWriteOutput struct {
	OldTodos []TodoItem `json:"oldTodos"`
	NewTodos []TodoItem `json:"newTodos"`
}

func (o TodoWriteOutput) MarshalJSON() ([]byte, error) {
	type plain TodoWriteOutput
	return json.Marshal(plain(o))
}

func (o *TodoWriteOutput) UnmarshalJSON(data []byte) error {
	type plain TodoWriteOutput
	return json.Unmarshal(data, (*plain)(o))
}

// ExitPlanModeOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#exit-plan-mode-2
// Tool name: ExitPlanMode
type ExitPlanModeOutput struct {
	Plan                    *string `json:"plan"`
	IsAgent                 bool    `json:"isAgent"`
	FilePath                *string `json:"filePath,omitempty"`
	HasTaskTool             *bool   `json:"hasTaskTool,omitempty"`
	AwaitingLeaderApproval  *bool   `json:"awaitingLeaderApproval,omitempty"`
	RequestID               *string `json:"requestId,omitempty"`
}

func (o ExitPlanModeOutput) MarshalJSON() ([]byte, error) {
	type plain ExitPlanModeOutput
	return json.Marshal(plain(o))
}

func (o *ExitPlanModeOutput) UnmarshalJSON(data []byte) error {
	type plain ExitPlanModeOutput
	return json.Unmarshal(data, (*plain)(o))
}

// ListMcpResourcesOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#list-mcp-resources-2
// Tool name: ListMcpResources
type ListMcpResourcesOutput []ListMcpResourcesOutputSingle

func (o ListMcpResourcesOutput) MarshalJSON() ([]byte, error) {
	return json.Marshal([]ListMcpResourcesOutputSingle(o))
}

func (o *ListMcpResourcesOutput) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*[]ListMcpResourcesOutputSingle)(o))
}

// ListMcpResourcesOutputSingle represents a single MCP resource.
type ListMcpResourcesOutputSingle struct {
	URI         string  `json:"uri"`
	Name        string  `json:"name"`
	MimeType    *string `json:"mimeType,omitempty"`
	Description *string `json:"description,omitempty"`
	Server      string  `json:"server"`
}

// ReadMcpResourceOutputContent represents a single content entry in ReadMcpResourceOutput.
type ReadMcpResourceOutputContent struct {
	URI      string  `json:"uri"`
	MimeType *string `json:"mimeType,omitempty"`
	Text     *string `json:"text,omitempty"`
}

// ReadMcpResourceOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#read-mcp-resource-2
// Tool name: ReadMcpResource
type ReadMcpResourceOutput struct {
	Contents []ReadMcpResourceOutputContent `json:"contents"`
}

func (o ReadMcpResourceOutput) MarshalJSON() ([]byte, error) {
	type plain ReadMcpResourceOutput
	return json.Marshal(plain(o))
}

func (o *ReadMcpResourceOutput) UnmarshalJSON(data []byte) error {
	type plain ReadMcpResourceOutput
	return json.Unmarshal(data, (*plain)(o))
}

// ConfigOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#config-2
// Tool name: Config
type ConfigOutput struct {
	Success       bool            `json:"success"`
	Operation     *string         `json:"operation,omitempty"`
	Setting       *string         `json:"setting,omitempty"`
	Value         json.RawMessage `json:"value,omitempty"`
	PreviousValue json.RawMessage `json:"previousValue,omitempty"`
	NewValue      json.RawMessage `json:"newValue,omitempty"`
	Error         *string         `json:"error,omitempty"`
}

func (o ConfigOutput) MarshalJSON() ([]byte, error) {
	type plain ConfigOutput
	return json.Marshal(plain(o))
}

func (o *ConfigOutput) UnmarshalJSON(data []byte) error {
	type plain ConfigOutput
	return json.Unmarshal(data, (*plain)(o))
}

// EnterWorktreeOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#enter-worktree-2
// Tool name: EnterWorktree
type EnterWorktreeOutput struct {
	WorktreePath   string  `json:"worktreePath"`
	WorktreeBranch *string `json:"worktreeBranch,omitempty"`
	Message        string  `json:"message"`
}

func (o EnterWorktreeOutput) MarshalJSON() ([]byte, error) {
	type plain EnterWorktreeOutput
	return json.Marshal(plain(o))
}

func (o *EnterWorktreeOutput) UnmarshalJSON(data []byte) error {
	type plain EnterWorktreeOutput
	return json.Unmarshal(data, (*plain)(o))
}
