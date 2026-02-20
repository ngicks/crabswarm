package model

// GrepOutputMode represents the output mode for the grep tool.
type GrepOutputMode string

const (
	GrepOutputModeContent          GrepOutputMode = "content"
	GrepOutputModeFilesWithMatches GrepOutputMode = "files_with_matches"
	GrepOutputModeCount            GrepOutputMode = "count"
)

// NotebookCellType represents the type of a notebook cell.
type NotebookCellType string

const (
	NotebookCellTypeCode     NotebookCellType = "code"
	NotebookCellTypeMarkdown NotebookCellType = "markdown"
)

// NotebookEditMode represents the edit mode for notebook editing.
type NotebookEditMode string

const (
	NotebookEditModeReplace NotebookEditMode = "replace"
	NotebookEditModeInsert  NotebookEditMode = "insert"
	NotebookEditModeDelete  NotebookEditMode = "delete"
)

// TodoStatus represents the status of a todo item.
type TodoStatus string

const (
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusCompleted  TodoStatus = "completed"
)
