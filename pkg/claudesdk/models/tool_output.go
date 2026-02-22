package models

// UsageInfo contains token usage statistics.
type UsageInfo struct {
	InputTokens              int32  `json:"input_tokens"`
	OutputTokens             int32  `json:"output_tokens"`
	CacheCreationInputTokens *int32 `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     *int32 `json:"cache_read_input_tokens,omitempty"`
}

// TaskOutput returns the final result from a subagent.
type TaskOutput struct {
	Result       string     `json:"result"`
	Usage        *UsageInfo `json:"usage,omitempty"`
	TotalCostUSD *float64   `json:"total_cost_usd,omitempty"`
	DurationMs   *int64     `json:"duration_ms,omitempty"`
}

// BashOutput returns command output with exit status.
type BashOutput struct {
	Output   string  `json:"output"`
	ExitCode int32   `json:"exit_code"`
	Killed   *bool   `json:"killed,omitempty"`
	ShellID  *string `json:"shell_id,omitempty"`
}

// BashOutputToolOutput returns incremental output from background shells.
type BashOutputToolOutput struct {
	Output   string `json:"output"`
	Status   string `json:"status"`
	ExitCode *int32 `json:"exit_code,omitempty"`
}

// EditOutput returns confirmation of successful edits.
type EditOutput struct {
	Message      string `json:"message"`
	Replacements int32  `json:"replacements"`
	FilePath     string `json:"file_path"`
}

// TextFileOutput contains text file contents.
type TextFileOutput struct {
	Content       string `json:"content"`
	TotalLines    int32  `json:"total_lines"`
	LinesReturned int32  `json:"lines_returned"`
}

// WriteOutput returns confirmation after writing a file.
type WriteOutput struct {
	Message      string `json:"message"`
	BytesWritten int64  `json:"bytes_written"`
	FilePath     string `json:"file_path"`
}

// GlobOutput returns file paths matching the glob pattern.
type GlobOutput struct {
	Matches    []string `json:"matches"`
	Count      int32    `json:"count"`
	SearchPath string   `json:"search_path"`
}

// GrepMatch represents a single grep match with context.
type GrepMatch struct {
	File          string   `json:"file"`
	LineNumber    *int32   `json:"line_number,omitempty"`
	Line          string   `json:"line"`
	BeforeContext []string `json:"before_context,omitempty"`
	AfterContext  []string `json:"after_context,omitempty"`
}

// GrepContentOutput contains matching lines with context.
type GrepContentOutput struct {
	Matches      []GrepMatch `json:"matches"`
	TotalMatches int32       `json:"total_matches"`
}

// GrepFilesOutput contains files with matches.
type GrepFilesOutput struct {
	Files []string `json:"files"`
	Count int32    `json:"count"`
}

// KillBashOutput returns confirmation after terminating a background shell.
type KillBashOutput struct {
	Message string `json:"message"`
	ShellID string `json:"shell_id"`
}

// NotebookEditOutput returns confirmation after modifying a Jupyter notebook.
type NotebookEditOutput struct {
	Message    string  `json:"message"`
	EditType   string  `json:"edit_type"`
	CellID     *string `json:"cell_id,omitempty"`
	TotalCells int32   `json:"total_cells"`
}

// WebFetchOutput returns the AI's analysis of fetched web content.
type WebFetchOutput struct {
	Response   string  `json:"response"`
	URL        string  `json:"url"`
	FinalURL   *string `json:"final_url,omitempty"`
	StatusCode *int32  `json:"status_code,omitempty"`
}

// WebSearchResult represents a single web search result.
type WebSearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// WebSearchOutput returns formatted search results from the web.
type WebSearchOutput struct {
	Results      []WebSearchResult `json:"results"`
	TotalResults int32             `json:"total_results"`
	Query        string            `json:"query"`
}

// TodoStats contains current task statistics.
type TodoStats struct {
	Total      int32 `json:"total"`
	Pending    int32 `json:"pending"`
	InProgress int32 `json:"in_progress"`
	Completed  int32 `json:"completed"`
}

// TodoWriteOutput returns confirmation with current task statistics.
type TodoWriteOutput struct {
	Message string     `json:"message"`
	Stats   *TodoStats `json:"stats,omitempty"`
}

// ExitPlanModeOutput returns confirmation after exiting plan mode.
type ExitPlanModeOutput struct {
	Message  string `json:"message"`
	Approved *bool  `json:"approved,omitempty"`
}
