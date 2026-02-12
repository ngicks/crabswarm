package model

// BashInput represents the input for the Bash tool.
type BashInput struct {
	// Command is the bash command to execute.
	Command string `json:"command"`
	// Description is a clear, concise description of what this command does.
	Description string `json:"description,omitempty"`
	// Timeout is an optional timeout in milliseconds (max 600000).
	Timeout int `json:"timeout,omitempty"`
	// RunInBackground runs the command in background if true.
	RunInBackground bool `json:"run_in_background,omitempty"`
	// DangerouslyDisableSandbox disables sandbox mode if true.
	DangerouslyDisableSandbox bool `json:"dangerouslyDisableSandbox,omitempty"`
}

// ReadInput represents the input for the Read tool.
type ReadInput struct {
	// FilePath is the absolute path to the file to read.
	FilePath string `json:"file_path"`
	// Offset is the line number to start reading from.
	Offset int `json:"offset,omitempty"`
	// Limit is the number of lines to read.
	Limit int `json:"limit,omitempty"`
}

// WriteInput represents the input for the Write tool.
type WriteInput struct {
	// FilePath is the absolute path to the file to write.
	FilePath string `json:"file_path"`
	// Content is the content to write to the file.
	Content string `json:"content"`
}

// EditInput represents the input for the Edit tool.
type EditInput struct {
	// FilePath is the absolute path to the file to modify.
	FilePath string `json:"file_path"`
	// OldString is the text to replace.
	OldString string `json:"old_string"`
	// NewString is the text to replace it with.
	NewString string `json:"new_string"`
	// ReplaceAll replaces all occurrences if true.
	ReplaceAll bool `json:"replace_all,omitempty"`
}

// GlobInput represents the input for the Glob tool.
type GlobInput struct {
	// Pattern is the glob pattern to match files against.
	Pattern string `json:"pattern"`
	// Path is the directory to search in.
	Path string `json:"path,omitempty"`
}

// GrepInput represents the input for the Grep tool.
type GrepInput struct {
	// Pattern is the regular expression pattern to search for.
	Pattern string `json:"pattern"`
	// Path is the file or directory to search in.
	Path string `json:"path,omitempty"`
	// Glob is a glob pattern to filter files.
	Glob string `json:"glob,omitempty"`
	// Type is the file type to search.
	Type string `json:"type,omitempty"`
	// OutputMode is "content", "files_with_matches", or "count".
	OutputMode string `json:"output_mode,omitempty"`
	// Context is the number of lines to show before and after each match.
	Context int `json:"context,omitempty"`
	// ContextBefore (-B) is the number of lines to show before each match.
	ContextBefore int `json:"-B,omitempty"`
	// ContextAfter (-A) is the number of lines to show after each match.
	ContextAfter int `json:"-A,omitempty"`
	// ContextC (-C) is alias for context.
	ContextC int `json:"-C,omitempty"`
	// CaseInsensitive (-i) enables case insensitive search.
	CaseInsensitive bool `json:"-i,omitempty"`
	// ShowLineNumbers (-n) shows line numbers in output.
	ShowLineNumbers bool `json:"-n,omitempty"`
	// Multiline enables multiline mode.
	Multiline bool `json:"multiline,omitempty"`
	// HeadLimit limits output to first N lines/entries.
	HeadLimit int `json:"head_limit,omitempty"`
	// Offset skips first N lines/entries.
	Offset int `json:"offset,omitempty"`
}

// WebFetchInput represents the input for the WebFetch tool.
type WebFetchInput struct {
	// URL is the URL to fetch content from.
	URL string `json:"url"`
	// Prompt describes what information to extract from the page.
	Prompt string `json:"prompt"`
}

// WebSearchInput represents the input for the WebSearch tool.
type WebSearchInput struct {
	// Query is the search query to use.
	Query string `json:"query"`
	// AllowedDomains limits results to these domains.
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	// BlockedDomains excludes results from these domains.
	BlockedDomains []string `json:"blocked_domains,omitempty"`
}

// TaskInput represents the input for the Task tool.
type TaskInput struct {
	// Description is a short (3-5 word) description of the task.
	Description string `json:"description"`
	// Prompt is the task for the agent to perform.
	Prompt string `json:"prompt"`
	// SubagentType is the type of specialized agent to use.
	SubagentType string `json:"subagent_type"`
	// Model is an optional model override.
	Model string `json:"model,omitempty"`
	// MaxTurns is the maximum number of agentic turns.
	MaxTurns int `json:"max_turns,omitempty"`
	// Resume is an optional agent ID to resume from.
	Resume string `json:"resume,omitempty"`
	// RunInBackground runs the agent in background if true.
	RunInBackground bool `json:"run_in_background,omitempty"`
}

// TaskOutputInput represents the input for the TaskOutput tool.
type TaskOutputInput struct {
	// TaskID is the task ID to get output from.
	TaskID string `json:"task_id"`
	// Block waits for completion if true.
	Block bool `json:"block"`
	// Timeout is the max wait time in ms.
	Timeout int `json:"timeout"`
}

// TaskStopInput represents the input for the TaskStop tool.
type TaskStopInput struct {
	// TaskID is the ID of the background task to stop.
	TaskID string `json:"task_id,omitempty"`
	// ShellID is deprecated, use TaskID instead.
	ShellID string `json:"shell_id,omitempty"`
}

// AskUserQuestionInput represents the input for the AskUserQuestion tool.
type AskUserQuestionInput struct {
	// Questions is the list of questions to ask the user.
	Questions []Question `json:"questions"`
	// Answers contains user answers collected by the permission component.
	Answers map[string]string `json:"answers,omitempty"`
	// Metadata is optional metadata for tracking and analytics.
	Metadata *QuestionMetadata `json:"metadata,omitempty"`
}

// Question represents a single question in AskUserQuestion.
type Question struct {
	// Question is the complete question to ask the user.
	Question string `json:"question"`
	// Header is a very short label (max 12 chars).
	Header string `json:"header"`
	// Options are the available choices.
	Options []QuestionOption `json:"options"`
	// MultiSelect allows multiple selections if true.
	MultiSelect bool `json:"multiSelect"`
}

// QuestionOption represents an option for a question.
type QuestionOption struct {
	// Label is the display text for this option.
	Label string `json:"label"`
	// Description explains what this option means.
	Description string `json:"description,omitempty"`
}

// QuestionMetadata contains optional metadata for questions.
type QuestionMetadata struct {
	// Source identifies the source of this question.
	Source string `json:"source,omitempty"`
}

// NotebookEditInput represents the input for the NotebookEdit tool.
type NotebookEditInput struct {
	// NotebookPath is the absolute path to the Jupyter notebook.
	NotebookPath string `json:"notebook_path"`
	// NewSource is the new source for the cell.
	NewSource string `json:"new_source"`
	// CellID is the ID of the cell to edit.
	CellID string `json:"cell_id,omitempty"`
	// CellType is "code" or "markdown".
	CellType string `json:"cell_type,omitempty"`
	// EditMode is "replace", "insert", or "delete".
	EditMode string `json:"edit_mode,omitempty"`
}

// SkillInput represents the input for the Skill tool.
type SkillInput struct {
	// Skill is the skill name.
	Skill string `json:"skill"`
	// Args are optional arguments for the skill.
	Args string `json:"args,omitempty"`
}

// TaskCreateInput represents the input for the TaskCreate tool.
type TaskCreateInput struct {
	// Subject is a brief title for the task.
	Subject string `json:"subject"`
	// Description is a detailed description of what needs to be done.
	Description string `json:"description"`
	// ActiveForm is present continuous form shown in spinner.
	ActiveForm string `json:"activeForm,omitempty"`
	// Metadata is arbitrary metadata to attach.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// TaskUpdateInput represents the input for the TaskUpdate tool.
type TaskUpdateInput struct {
	// TaskID is the ID of the task to update.
	TaskID string `json:"taskId"`
	// Status is the new status.
	Status string `json:"status,omitempty"`
	// Subject is the new subject.
	Subject string `json:"subject,omitempty"`
	// Description is the new description.
	Description string `json:"description,omitempty"`
	// ActiveForm is the new active form.
	ActiveForm string `json:"activeForm,omitempty"`
	// Owner is the new owner.
	Owner string `json:"owner,omitempty"`
	// Metadata is metadata keys to merge.
	Metadata map[string]any `json:"metadata,omitempty"`
	// AddBlocks are task IDs that this task blocks.
	AddBlocks []string `json:"addBlocks,omitempty"`
	// AddBlockedBy are task IDs that block this task.
	AddBlockedBy []string `json:"addBlockedBy,omitempty"`
}

// TaskGetInput represents the input for the TaskGet tool.
type TaskGetInput struct {
	// TaskID is the ID of the task to retrieve.
	TaskID string `json:"taskId"`
}

// EnterPlanModeInput represents the input for the EnterPlanMode tool.
// It has no parameters.
type EnterPlanModeInput struct{}

// ExitPlanModeInput represents the input for the ExitPlanMode tool.
type ExitPlanModeInput struct {
	// AllowedPrompts are prompt-based permissions needed to implement the plan.
	AllowedPrompts []AllowedPrompt `json:"allowedPrompts,omitempty"`
	// PushToRemote indicates whether to push the plan to a remote session.
	PushToRemote bool `json:"pushToRemote,omitempty"`
	// RemoteSessionID is the remote session ID if pushed.
	RemoteSessionID string `json:"remoteSessionId,omitempty"`
	// RemoteSessionTitle is the remote session title if pushed.
	RemoteSessionTitle string `json:"remoteSessionTitle,omitempty"`
	// RemoteSessionURL is the remote session URL if pushed.
	RemoteSessionURL string `json:"remoteSessionUrl,omitempty"`
}

// AllowedPrompt represents a prompt-based permission.
type AllowedPrompt struct {
	// Tool is the tool this prompt applies to.
	Tool string `json:"tool"`
	// Prompt is a semantic description of the action.
	Prompt string `json:"prompt"`
}

// TaskListInput represents the input for the TaskList tool.
// It has no parameters.
type TaskListInput struct{}

// MCPToolInput represents a generic input for MCP tools.
// Since MCP tools have variable schemas, this stores the raw parameters.
type MCPToolInput struct {
	// Server is the MCP server name (parsed from tool name).
	Server string `json:"-"`
	// Tool is the MCP tool name (parsed from tool name).
	Tool string `json:"-"`
	// Parameters contains the raw MCP tool parameters.
	Parameters map[string]any `json:"-"`
}
