package model

// BashOutputStatus represents the status of a bash command output.
type BashOutputStatus string

const (
	BashOutputStatusRunning   BashOutputStatus = "running"
	BashOutputStatusCompleted BashOutputStatus = "completed"
	BashOutputStatusFailed    BashOutputStatus = "failed"
)

// NotebookEditType represents the result type of a notebook edit operation.
type NotebookEditType string

const (
	NotebookEditTypeReplaced NotebookEditType = "replaced"
	NotebookEditTypeInserted NotebookEditType = "inserted"
	NotebookEditTypeDeleted  NotebookEditType = "deleted"
)
