package model

// ToolInput is a union type for all tool inputs.
type ToolInput struct {
	Agent            *AgentInput            `json:"agent,omitempty"`
	AskUserQuestion  *AskUserQuestionInput  `json:"ask_user_question,omitempty"`
	Bash             *BashInput             `json:"bash,omitempty"`
	BashOutput       *BashOutputInput       `json:"bash_output,omitempty"`
	FileEdit         *FileEditInput         `json:"file_edit,omitempty"`
	FileRead         *FileReadInput         `json:"file_read,omitempty"`
	FileWrite        *FileWriteInput        `json:"file_write,omitempty"`
	Glob             *GlobInput             `json:"glob,omitempty"`
	Grep             *GrepInput             `json:"grep,omitempty"`
	KillShell        *KillShellInput        `json:"kill_shell,omitempty"`
	NotebookEdit     *NotebookEditInput     `json:"notebook_edit,omitempty"`
	WebFetch         *WebFetchInput         `json:"web_fetch,omitempty"`
	WebSearch        *WebSearchInput        `json:"web_search,omitempty"`
	TodoWrite        *TodoWriteInput        `json:"todo_write,omitempty"`
	ExitPlanMode     *ExitPlanModeInput     `json:"exit_plan_mode,omitempty"`
	ListMcpResources *ListMcpResourcesInput `json:"list_mcp_resources,omitempty"`
	ReadMcpResource  *ReadMcpResourceInput  `json:"read_mcp_resource,omitempty"`
	McpTool          map[string]any         `json:"mcp_tool,omitempty"`
}

// AgentInput represents input for the agent tool.
type AgentInput struct {
	Description  string `json:"description"`
	Prompt       string `json:"prompt"`
	SubagentType string `json:"subagent_type"`
}

// QuestionOption represents an option for a question.
type QuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Question represents a question to ask the user.
type Question struct {
	Question    string            `json:"question"`
	Header      string            `json:"header"`
	Options     []*QuestionOption `json:"options,omitempty"`
	MultiSelect bool              `json:"multi_select"`
}

// AskUserQuestionInput represents input for the ask-user-question tool.
type AskUserQuestionInput struct {
	Questions []*Question       `json:"questions,omitempty"`
	Answers   map[string]string `json:"answers,omitempty"`
}

// BashInput represents input for the bash tool.
type BashInput struct {
	Command         string  `json:"command"`
	Timeout         *int32  `json:"timeout,omitempty"`
	Description     *string `json:"description,omitempty"`
	RunInBackground *bool   `json:"run_in_background,omitempty"`
}

// BashOutputInput represents input for the bash-output tool.
type BashOutputInput struct {
	BashID string  `json:"bash_id"`
	Filter *string `json:"filter,omitempty"`
}

// FileEditInput represents input for the file-edit tool.
type FileEditInput struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll *bool  `json:"replace_all,omitempty"`
}

// FileReadInput represents input for the file-read tool.
type FileReadInput struct {
	FilePath string `json:"file_path"`
	Offset   *int32 `json:"offset,omitempty"`
	Limit    *int32 `json:"limit,omitempty"`
}

// FileWriteInput represents input for the file-write tool.
type FileWriteInput struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// GlobInput represents input for the glob tool.
type GlobInput struct {
	Pattern string  `json:"pattern"`
	Path    *string `json:"path,omitempty"`
}

// GrepInput represents input for the grep tool.
type GrepInput struct {
	Pattern         string         `json:"pattern"`
	Path            *string        `json:"path,omitempty"`
	Glob            *string        `json:"glob,omitempty"`
	Type            *string        `json:"type,omitempty"`
	OutputMode      GrepOutputMode `json:"output_mode"`
	CaseInsensitive *bool          `json:"case_insensitive,omitempty"`
	ShowLineNumbers *bool          `json:"show_line_numbers,omitempty"`
	BeforeContext   *int32         `json:"before_context,omitempty"`
	AfterContext    *int32         `json:"after_context,omitempty"`
	Context         *int32         `json:"context,omitempty"`
	HeadLimit       *int32         `json:"head_limit,omitempty"`
	Multiline       *bool          `json:"multiline,omitempty"`
}

// KillShellInput represents input for the kill-shell tool.
type KillShellInput struct {
	ShellID string `json:"shell_id"`
}

// NotebookEditInput represents input for the notebook-edit tool.
type NotebookEditInput struct {
	NotebookPath string           `json:"notebook_path"`
	CellID       *string          `json:"cell_id,omitempty"`
	NewSource    string           `json:"new_source"`
	CellType     NotebookCellType `json:"cell_type"`
	EditMode     NotebookEditMode `json:"edit_mode"`
}

// WebFetchInput represents input for the web-fetch tool.
type WebFetchInput struct {
	URL    string `json:"url"`
	Prompt string `json:"prompt"`
}

// WebSearchInput represents input for the web-search tool.
type WebSearchInput struct {
	Query          string   `json:"query"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	BlockedDomains []string `json:"blocked_domains,omitempty"`
}

// TodoItem represents a todo item.
type TodoItem struct {
	Content    string     `json:"content"`
	Status     TodoStatus `json:"status"`
	ActiveForm string     `json:"active_form"`
}

// TodoWriteInput represents input for the todo-write tool.
type TodoWriteInput struct {
	Todos []*TodoItem `json:"todos,omitempty"`
}

// ExitPlanModeInput represents input for the exit-plan-mode tool.
type ExitPlanModeInput struct {
	Plan string `json:"plan"`
}

// ListMcpResourcesInput represents input for the list-mcp-resources tool.
type ListMcpResourcesInput struct {
	Server *string `json:"server,omitempty"`
}

// ReadMcpResourceInput represents input for the read-mcp-resource tool.
type ReadMcpResourceInput struct {
	Server string `json:"server"`
	URI    string `json:"uri"`
}
