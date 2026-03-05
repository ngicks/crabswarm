package models

import (
	"encoding/json"
	"strings"
)

// As per https://platform.claude.com/docs/en/agent-sdk/typescript#tool-input-types
/*
type ToolInputSchemas =
  | AgentInput
  | AskUserQuestionInput
  | BashInput
  | TaskOutputInput
  | ConfigInput
  | EnterWorktreeInput
  | ExitPlanModeInput
  | FileEditInput
  | FileReadInput
  | FileWriteInput
  | GlobInput
  | GrepInput
  | ListMcpResourcesInput
  | McpInput
  | NotebookEditInput
  | ReadMcpResourceInput
  | SubscribeMcpResourceInput
  | SubscribePollingInput
  | TaskStopInput
  | TodoWriteInput
  | UnsubscribeMcpResourceInput
  | UnsubscribePollingInput
  | WebFetchInput
  | WebSearchInput;
*/

// ToolInputSchemas is union type as per As per https://platform.claude.com/docs/en/agent-sdk/typescript#tool-input-types
//
// Note that some types are not listed but actually exist.
// When the parser encounters unknown union variant then it
type ToolInputSchemas interface {
	toolInputSchemas()
	json.Marshaler
	json.Unmarshaler
}

// fallback target
func (UnknownInput) toolInputSchemas() {}

func (AgentInput) toolInputSchemas()                  {}
func (AskUserQuestionInput) toolInputSchemas()        {}
func (BashInput) toolInputSchemas()                   {}
func (TaskOutputInput) toolInputSchemas()             {}
func (ConfigInput) toolInputSchemas()                 {}
func (EnterWorktreeInput) toolInputSchemas()          {}
func (ExitPlanModeInput) toolInputSchemas()           {}
func (FileEditInput) toolInputSchemas()               {}
func (FileReadInput) toolInputSchemas()               {}
func (FileWriteInput) toolInputSchemas()              {}
func (GlobInput) toolInputSchemas()                   {}
func (GrepInput) toolInputSchemas()                   {}
func (ListMcpResourcesInput) toolInputSchemas()       {}
func (McpInput) toolInputSchemas()                    {}
func (NotebookEditInput) toolInputSchemas()           {}
func (ReadMcpResourceInput) toolInputSchemas()        {}
func (SubscribeMcpResourceInput) toolInputSchemas()   {}
func (SubscribePollingInput) toolInputSchemas()       {}
func (TaskStopInput) toolInputSchemas()               {}
func (TodoWriteInput) toolInputSchemas()              {}
func (UnsubscribeMcpResourceInput) toolInputSchemas() {}
func (UnsubscribePollingInput) toolInputSchemas()     {}
func (WebFetchInput) toolInputSchemas()               {}
func (WebSearchInput) toolInputSchemas()              {}

// -- not listed --

func (BashOutputInput) toolInputSchemas() {}
func (KillShellInput) toolInputSchemas()  {}

func toolNameToInputType(toolName string) ToolInputSchemas {
	switch toolName {
	case "Task":
		return &AgentInput{}
	case "AskUserQuestion":
		return &AskUserQuestionInput{}
	case "Bash":
		return &BashInput{}
	case "TaskOutput":
		return &TaskOutputInput{}
	case "Edit":
		return &FileEditInput{}
	case "Read":
		return &FileReadInput{}
	case "Write":
		return &FileWriteInput{}
	case "Glob":
		return &GlobInput{}
	case "Grep":
		return &GrepInput{}
	case "TaskStop":
		return &TaskStopInput{}
	case "NotebookEdit":
		return &NotebookEditInput{}
	case "WebFetch":
		return &WebFetchInput{}
	case "WebSearch":
		return &WebSearchInput{}
	case "TodoWrite":
		//		, "TaskCreate", "TaskUpdate", "TaskList", "TaskGet":
		return &TodoWriteInput{}
	case "ExitPlanMode":
		return &ExitPlanModeInput{}
	case "ListMcpResources":
		return &ListMcpResourcesInput{}
	case "ReadMcpResource":
		return &ReadMcpResourceInput{}
	case "Config":
		return &ConfigInput{}
	case "EnterWorktree":
		return &EnterWorktreeInput{}

		// actually tool names for these tools are not listed in SDK doc;
		// so we'll sample from actual softwares and update accordingly
	case "SubscribeMcpResource":
		return &SubscribeMcpResourceInput{}
	case "SubscribePolling":
		return &SubscribePollingInput{}
	case "UnsubscribeMcpResource":
		return &UnsubscribeMcpResourceInput{}
	case "UnsubscribePolling":
		return &UnsubscribePollingInput{}
	case "BashOutput":
		return &BashOutputInput{}
	case "KillShell":
		return &KillShellInput{}
	}

	if strings.HasPrefix(toolName, "mcp__") {
		return &McpInput{}
	}

	return &UnknownInput{}
}

func unmarshalToolInputSchemas(data []byte) (_ ToolInputSchemas, err error) {}

// UnknownInput is fallback target:
// If parser hits unlisted tool_name, it falls back to this value
type UnknownInput map[string]any

// AgentInput is the input for the Task tool.
type AgentInput struct {
	Description  string `json:"description"`
	Prompt       string `json:"prompt"`
	SubagentType string `json:"subagent_type"`
}

// AskUserQuestionInput asks the user clarifying questions during execution.
type AskUserQuestionInput struct {
	Questions []AskUserQuestionInputQuestion `json:"questions"`
	Answers   map[string]string              `json:"answers,omitempty"`
}

// Question represents a single question to ask the user.
type AskUserQuestionInputQuestion struct {
	Question    string                               `json:"question"`
	Header      string                               `json:"header"`
	Options     []AskUserQuestionInputQuestionOption `json:"options"`
	MultiSelect bool                                 `json:"multiSelect"`
}

// QuestionOption represents a single choice option within a question.
type AskUserQuestionInputQuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// BashInput executes bash commands in a persistent shell session.
type BashInput struct {
	Command                   string  `json:"command"`
	Timeout                   *int32  `json:"timeout,omitempty"`
	Description               *string `json:"description,omitempty"`
	RunInBackground           *bool   `json:"run_in_background,omitempty"`
	DangerouslyDisableSandbox *bool   `json:"dangerouslyDisableSandbox,omitempty"`
}

type TaskOutputInput struct {
	TaskId  string `json:"task_id"`
	Block   bool   `json:"block"`
	Timeout int32  `json:"timeout"`
}

type ConfigInput struct {
	Settings string `json:"setting"`
	Value    *any   `json:"value,omitempty"` // string | boolean | number;
}

type EnterWorktreeInput struct {
	Name *string `json:"name,omitempty"`
}

type ExitPlanModeInput struct {
	Plan           string          `json:"plan"`
	AllowedPrompts []AllowedPrompt `json:"allowedPrompts,omitempty"`
}

type AllowedPrompt struct {
	Tool   string `json:"tool"`
	Prompt string `json:"prompt"`
}

type FileEditInput struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll *bool  `json:"replace_all"`
}

type FileReadInput struct {
	FilePath string `json:"file_path"`
	Offset   *int32 `json:"offset,omitempty"`
	Limit    *int32 `json:"limit,omitempty"`
}

type FileWriteInput struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

type GlobInput struct {
	Pattern string  `json:"pattern"`
	Path    *string `json:"path,omitempty"`
}

type GrepInput struct {
	Pattern         string  `json:"pattern"`
	Path            *string `json:"path,omitempty"`
	Glob            *string `json:"glob,omitempty"`
	Type            *string `json:"type,omitempty"`
	OutputMode      string  `json:"output_mode"`
	CaseInsensitive *bool   `json:"-i,omitempty"`
	ShowLineNumbers *bool   `json:"-n,omitempty"`
	BeforeContext   *int32  `json:"-B,omitempty"`
	AfterContext    *int32  `json:"-A,omitempty"`
	Context         *int32  `json:"-C,omitempty"`
	HeadLimit       *int32  `json:"head_limit,omitempty"`
	Multiline       *bool   `json:"multiline,omitempty"`
}

type ListMcpResourcesInput struct {
	Server *string `json:"server,omitempty"`
}

// Not defined in SDK doc.

type McpInput map[string]any

type NotebookEditInput struct {
	NotebookPath string  `json:"notebook_path"`
	CellId       *string `json:"cell_id,omitempty"`
	NewSource    string  `json:"new_source"`
	CellType     string  `json:"cell_type"`
	EditMode     string  `json:"edit_mode"`
}

type ReadMcpResourceInput struct {
	Server string `json:"server"`
	Uri    string `json:"uri"`
}

// Not defined in SDK doc.

type SubscribeMcpResourceInput map[string]any

// Not defined in SDK doc.

type SubscribePollingInput map[string]any

type TaskStopInput struct {
	TaskId  *string `json:"task_id,omitempty"`
	ShellId *string `json:"shell_id"`
}

type TodoWriteInput struct {
	Todos []TodoItem `json:"todos"`
}

type TodoItem struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"activeForm"`
}

type UnsubscribeMcpResourceInput map[string]any

type UnsubscribePollingInput map[string]any

type WebFetchInput struct {
	Url    string `json:"url"`
	Prompt string `json:"prompt"`
}

type WebSearchInput struct {
	Query          string   `json:"query"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	BlockedDomains []string `json:"blocked_domains,omitempty"`
}

// -- Below here, types not listed on https://platform.claude.com/docs/en/agent-sdk/typescript#tool-input-types

type BashOutputInput struct {
	BashID string  `json:"bash_id"`
	Filter *string `json:"filter,omitempty"`
}

// KillShellInput kills a running background shell by its ID.
type KillShellInput struct {
	ShellID string `json:"shell_id"`
}
