package model

// ToolOutput is a union type for all tool outputs.
type ToolOutput struct {
	Task             *TaskOutput             `json:"task,omitempty"`
	AskUserQuestion  *AskUserQuestionOutput  `json:"ask_user_question,omitempty"`
	Bash             *BashOutput             `json:"bash,omitempty"`
	BashOutput       *BashOutputToolOutput   `json:"bash_output,omitempty"`
	Edit             *EditOutput             `json:"edit,omitempty"`
	Read             *ReadOutput             `json:"read,omitempty"`
	Write            *WriteOutput            `json:"write,omitempty"`
	Glob             *GlobOutput             `json:"glob,omitempty"`
	Grep             *GrepOutput             `json:"grep,omitempty"`
	KillBash         *KillBashOutput         `json:"kill_bash,omitempty"`
	NotebookEdit     *NotebookEditOutput     `json:"notebook_edit,omitempty"`
	WebFetch         *WebFetchOutput         `json:"web_fetch,omitempty"`
	WebSearch        *WebSearchOutput        `json:"web_search,omitempty"`
	TodoWrite        *TodoWriteOutput        `json:"todo_write,omitempty"`
	ExitPlanMode     *ExitPlanModeOutput     `json:"exit_plan_mode,omitempty"`
	ListMcpResources *ListMcpResourcesOutput `json:"list_mcp_resources,omitempty"`
	ReadMcpResource  *ReadMcpResourceOutput  `json:"read_mcp_resource,omitempty"`
	McpTool          map[string]any          `json:"mcp_tool,omitempty"`
}

// UsageInfo represents token usage information.
type UsageInfo struct {
	InputTokens               int32  `json:"input_tokens"`
	OutputTokens              int32  `json:"output_tokens"`
	CacheCreationInputTokens  *int32 `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens      *int32 `json:"cache_read_input_tokens,omitempty"`
}

// TaskOutput represents output from a task/agent tool.
type TaskOutput struct {
	Result       string     `json:"result"`
	Usage        *UsageInfo `json:"usage,omitempty"`
	TotalCostUsd *float64   `json:"total_cost_usd,omitempty"`
	DurationMs   *int64     `json:"duration_ms,omitempty"`
}

// AskUserQuestionOutput represents output from the ask-user-question tool.
type AskUserQuestionOutput struct {
	Questions []*Question       `json:"questions,omitempty"`
	Answers   map[string]string `json:"answers,omitempty"`
}

// BashOutput represents output from the bash tool.
type BashOutput struct {
	Output   string  `json:"output"`
	ExitCode int32   `json:"exit_code"`
	Killed   *bool   `json:"killed,omitempty"`
	ShellID  *string `json:"shell_id,omitempty"`
}

// BashOutputToolOutput represents output from the bash-output tool.
type BashOutputToolOutput struct {
	Output   string           `json:"output"`
	Status   BashOutputStatus `json:"status"`
	ExitCode *int32           `json:"exit_code,omitempty"`
}

// EditOutput represents output from the edit tool.
type EditOutput struct {
	Message      string `json:"message"`
	Replacements int32  `json:"replacements"`
	FilePath     string `json:"file_path"`
}

// TextFileOutput represents text file content from a read operation.
type TextFileOutput struct {
	Content       string `json:"content"`
	TotalLines    int32  `json:"total_lines"`
	LinesReturned int32  `json:"lines_returned"`
}

// ImageFileOutput represents image file content from a read operation.
type ImageFileOutput struct {
	Image    string `json:"image"`
	MimeType string `json:"mime_type"`
	FileSize int64  `json:"file_size"`
}

// PDFPageImage represents an image within a PDF page.
type PDFPageImage struct {
	Image    string `json:"image"`
	MimeType string `json:"mime_type"`
}

// PDFPage represents a single page of a PDF.
type PDFPage struct {
	PageNumber int32           `json:"page_number"`
	Text       *string         `json:"text,omitempty"`
	Images     []*PDFPageImage `json:"images,omitempty"`
}

// PDFFileOutput represents PDF file content from a read operation.
type PDFFileOutput struct {
	Pages      []*PDFPage `json:"pages,omitempty"`
	TotalPages int32      `json:"total_pages"`
}

// NotebookCell represents a cell in a notebook.
type NotebookCell struct {
	CellType       string `json:"cell_type"`
	Source         string `json:"source"`
	Outputs        []any  `json:"outputs,omitempty"`
	ExecutionCount *int32 `json:"execution_count,omitempty"`
}

// NotebookFileOutput represents notebook file content from a read operation.
type NotebookFileOutput struct {
	Cells    []*NotebookCell `json:"cells,omitempty"`
	Metadata map[string]any  `json:"metadata,omitempty"`
}

// ReadOutput is a union type for read operation results.
type ReadOutput struct {
	TextFile     *TextFileOutput     `json:"text_file,omitempty"`
	ImageFile    *ImageFileOutput    `json:"image_file,omitempty"`
	PdfFile      *PDFFileOutput      `json:"pdf_file,omitempty"`
	NotebookFile *NotebookFileOutput `json:"notebook_file,omitempty"`
}

// WriteOutput represents output from the write tool.
type WriteOutput struct {
	Message      string `json:"message"`
	BytesWritten int64  `json:"bytes_written"`
	FilePath     string `json:"file_path"`
}

// GlobOutput represents output from the glob tool.
type GlobOutput struct {
	Matches    []string `json:"matches,omitempty"`
	Count      int32    `json:"count"`
	SearchPath string   `json:"search_path"`
}

// GrepMatch represents a single grep match.
type GrepMatch struct {
	File          string   `json:"file"`
	LineNumber    *int32   `json:"line_number,omitempty"`
	Line          string   `json:"line"`
	BeforeContext []string `json:"before_context,omitempty"`
	AfterContext  []string `json:"after_context,omitempty"`
}

// GrepContentOutput represents grep output in content mode.
type GrepContentOutput struct {
	Matches      []*GrepMatch `json:"matches,omitempty"`
	TotalMatches int32        `json:"total_matches"`
}

// GrepFilesOutput represents grep output in files mode.
type GrepFilesOutput struct {
	Files []string `json:"files,omitempty"`
	Count int32    `json:"count"`
}

// GrepFileCount represents a file match count.
type GrepFileCount struct {
	File  string `json:"file"`
	Count int32  `json:"count"`
}

// GrepCountOutput represents grep output in count mode.
type GrepCountOutput struct {
	Counts []*GrepFileCount `json:"counts,omitempty"`
	Total  int32            `json:"total"`
}

// GrepOutput is a union type for grep results.
type GrepOutput struct {
	Content *GrepContentOutput `json:"content,omitempty"`
	Files   *GrepFilesOutput   `json:"files,omitempty"`
	Count   *GrepCountOutput   `json:"count,omitempty"`
}

// KillBashOutput represents output from the kill-bash tool.
type KillBashOutput struct {
	Message string `json:"message"`
	ShellID string `json:"shell_id"`
}

// NotebookEditOutput represents output from the notebook-edit tool.
type NotebookEditOutput struct {
	Message    string           `json:"message"`
	EditType   NotebookEditType `json:"edit_type"`
	CellID     *string          `json:"cell_id,omitempty"`
	TotalCells int32            `json:"total_cells"`
}

// WebFetchOutput represents output from the web-fetch tool.
type WebFetchOutput struct {
	Response   string  `json:"response"`
	URL        string  `json:"url"`
	FinalURL   *string `json:"final_url,omitempty"`
	StatusCode *int32  `json:"status_code,omitempty"`
}

// WebSearchResult represents a single web search result.
type WebSearchResult struct {
	Title    string         `json:"title"`
	URL      string         `json:"url"`
	Snippet  string         `json:"snippet"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// WebSearchOutput represents output from the web-search tool.
type WebSearchOutput struct {
	Results      []*WebSearchResult `json:"results,omitempty"`
	TotalResults int32              `json:"total_results"`
	Query        string             `json:"query"`
}

// TodoStats represents statistics about todo items.
type TodoStats struct {
	Total      int32 `json:"total"`
	Pending    int32 `json:"pending"`
	InProgress int32 `json:"in_progress"`
	Completed  int32 `json:"completed"`
}

// TodoWriteOutput represents output from the todo-write tool.
type TodoWriteOutput struct {
	Message string     `json:"message"`
	Stats   *TodoStats `json:"stats,omitempty"`
}

// ExitPlanModeOutput represents output from the exit-plan-mode tool.
type ExitPlanModeOutput struct {
	Message  string `json:"message"`
	Approved *bool  `json:"approved,omitempty"`
}

// McpResource represents an MCP resource.
type McpResource struct {
	URI         string  `json:"uri"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	MimeType    *string `json:"mime_type,omitempty"`
	Server      string  `json:"server"`
}

// ListMcpResourcesOutput represents output from the list-mcp-resources tool.
type ListMcpResourcesOutput struct {
	Resources []*McpResource `json:"resources,omitempty"`
	Total     int32          `json:"total"`
}

// McpResourceContent represents content of an MCP resource.
type McpResourceContent struct {
	URI      string  `json:"uri"`
	MimeType *string `json:"mime_type,omitempty"`
	Text     *string `json:"text,omitempty"`
	Blob     *string `json:"blob,omitempty"`
}

// ReadMcpResourceOutput represents output from the read-mcp-resource tool.
type ReadMcpResourceOutput struct {
	Contents []*McpResourceContent `json:"contents,omitempty"`
	Server   string                `json:"server"`
}
