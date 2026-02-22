// Package models provides JSON-native Go types that match Claude Code's
// actual JSON wire format. These exist because protobuf oneof types
// (ToolInput, ToolOutput, HookSpecificOutput) produce nested JSON that
// doesn't match what Claude Code sends/expects.
package models

import (
	"encoding/json"
	"fmt"
)

// AgentInput is the input for the Task tool.
type AgentInput struct {
	Description  string `json:"description"`
	Prompt       string `json:"prompt"`
	SubagentType string `json:"subagent_type"`
}

// QuestionOption represents a single choice option within a question.
type QuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Question represents a single question to ask the user.
type Question struct {
	Question    string           `json:"question"`
	Header      string           `json:"header"`
	Options     []QuestionOption `json:"options"`
	MultiSelect bool             `json:"multiSelect"`
}

// AskUserQuestionInput asks the user clarifying questions during execution.
type AskUserQuestionInput struct {
	Questions []Question        `json:"questions"`
	Answers   map[string]string `json:"answers,omitempty"`
}

// BashInput executes bash commands in a persistent shell session.
type BashInput struct {
	Command         string  `json:"command"`
	Timeout         *int32  `json:"timeout,omitempty"`
	Description     *string `json:"description,omitempty"`
	RunInBackground *bool   `json:"run_in_background,omitempty"`
}

// BashOutputInput retrieves output from a running or completed background shell.
type BashOutputInput struct {
	BashID string  `json:"bash_id"`
	Filter *string `json:"filter,omitempty"`
}

// FileEditInput performs exact string replacements in files.
type FileEditInput struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll *bool  `json:"replace_all,omitempty"`
}

// FileReadInput reads files from the local filesystem.
type FileReadInput struct {
	FilePath string `json:"file_path"`
	Offset   *int32 `json:"offset,omitempty"`
	Limit    *int32 `json:"limit,omitempty"`
}

// FileWriteInput writes a file to the local filesystem.
type FileWriteInput struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// GlobInput performs fast file pattern matching.
type GlobInput struct {
	Pattern string  `json:"pattern"`
	Path    *string `json:"path,omitempty"`
}

// GrepInput searches file contents using ripgrep with regex support.
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

// KillShellInput kills a running background shell by its ID.
type KillShellInput struct {
	ShellID string `json:"shell_id"`
}

// NotebookEditInput edits cells in Jupyter notebook files.
type NotebookEditInput struct {
	NotebookPath string  `json:"notebook_path"`
	CellID       *string `json:"cell_id,omitempty"`
	NewSource    string  `json:"new_source"`
	CellType     string  `json:"cell_type"`
	EditMode     string  `json:"edit_mode"`
}

// WebFetchInput fetches content from a URL and processes it with an AI model.
type WebFetchInput struct {
	URL    string `json:"url"`
	Prompt string `json:"prompt"`
}

// WebSearchInput searches the web and returns formatted results.
type WebSearchInput struct {
	Query          string   `json:"query"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	BlockedDomains []string `json:"blocked_domains,omitempty"`
}

// TodoItem represents a single task in the todo list.
type TodoItem struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"activeForm"`
}

// TodoWriteInput creates and manages a structured task list.
type TodoWriteInput struct {
	Todos []TodoItem `json:"todos"`
}

// ExitPlanModeInput exits planning mode and prompts the user to approve the plan.
type ExitPlanModeInput struct {
	Plan string `json:"plan"`
}

// ListMcpResourcesInput lists available MCP resources from connected servers.
type ListMcpResourcesInput struct {
	Server *string `json:"server,omitempty"`
}

// ReadMcpResourceInput reads a specific MCP resource from a server.
type ReadMcpResourceInput struct {
	Server string `json:"server"`
	URI    string `json:"uri"`
}

// ParseToolInput unmarshals raw JSON tool_input into a concrete Go type
// based on the tool_name discriminator. Returns the typed struct for known
// tools, or map[string]any for unknown/MCP tools.
func ParseToolInput(toolName string, raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var target any
	switch toolName {
	case "Read":
		target = new(FileReadInput)
	case "Write":
		target = new(FileWriteInput)
	case "Edit":
		target = new(FileEditInput)
	case "Bash":
		target = new(BashInput)
	case "Glob":
		target = new(GlobInput)
	case "Grep":
		target = new(GrepInput)
	case "Task":
		target = new(AgentInput)
	case "WebFetch":
		target = new(WebFetchInput)
	case "WebSearch":
		target = new(WebSearchInput)
	case "NotebookEdit":
		target = new(NotebookEditInput)
	case "AskUserQuestion":
		target = new(AskUserQuestionInput)
	case "ExitPlanMode":
		target = new(ExitPlanModeInput)
	case "EnterPlanMode":
		// EnterPlanMode has no parameters.
		return nil, nil
	case "BashOutput", "TaskOutput":
		target = new(BashOutputInput)
	case "KillShell", "TaskStop":
		target = new(KillShellInput)
	case "TodoWrite", "TaskCreate", "TaskUpdate", "TaskList", "TaskGet":
		target = new(TodoWriteInput)
	case "ListMcpResources":
		target = new(ListMcpResourcesInput)
	case "ReadMcpResource":
		target = new(ReadMcpResourceInput)
	default:
		// Unknown or MCP tools: unmarshal into generic map.
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("unmarshal unknown tool %q input: %w", toolName, err)
		}
		return m, nil
	}

	if err := json.Unmarshal(raw, target); err != nil {
		return nil, fmt.Errorf("unmarshal %q tool input: %w", toolName, err)
	}
	return target, nil
}
