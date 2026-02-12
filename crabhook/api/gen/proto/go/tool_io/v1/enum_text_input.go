package tool_iov1

import "fmt"

func (x GrepOutputMode) MarshalText() ([]byte, error) {
	switch x {
	case GrepOutputMode_GREP_OUTPUT_MODE_CONTENT:
		return []byte("content"), nil
	case GrepOutputMode_GREP_OUTPUT_MODE_FILES_WITH_MATCHES:
		return []byte("files_with_matches"), nil
	case GrepOutputMode_GREP_OUTPUT_MODE_COUNT:
		return []byte("count"), nil
	default:
		return nil, nil
	}
}

func (x *GrepOutputMode) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = GrepOutputMode_GREP_OUTPUT_MODE_UNSPECIFIED
	case "content":
		*x = GrepOutputMode_GREP_OUTPUT_MODE_CONTENT
	case "files_with_matches":
		*x = GrepOutputMode_GREP_OUTPUT_MODE_FILES_WITH_MATCHES
	case "count":
		*x = GrepOutputMode_GREP_OUTPUT_MODE_COUNT
	default:
		return fmt.Errorf("unknown GrepOutputMode text: %q", string(b))
	}
	return nil
}

func (x NotebookCellType) MarshalText() ([]byte, error) {
	switch x {
	case NotebookCellType_NOTEBOOK_CELL_TYPE_CODE:
		return []byte("code"), nil
	case NotebookCellType_NOTEBOOK_CELL_TYPE_MARKDOWN:
		return []byte("markdown"), nil
	default:
		return nil, nil
	}
}

func (x *NotebookCellType) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = NotebookCellType_NOTEBOOK_CELL_TYPE_UNSPECIFIED
	case "code":
		*x = NotebookCellType_NOTEBOOK_CELL_TYPE_CODE
	case "markdown":
		*x = NotebookCellType_NOTEBOOK_CELL_TYPE_MARKDOWN
	default:
		return fmt.Errorf("unknown NotebookCellType text: %q", string(b))
	}
	return nil
}

func (x NotebookEditMode) MarshalText() ([]byte, error) {
	switch x {
	case NotebookEditMode_NOTEBOOK_EDIT_MODE_REPLACE:
		return []byte("replace"), nil
	case NotebookEditMode_NOTEBOOK_EDIT_MODE_INSERT:
		return []byte("insert"), nil
	case NotebookEditMode_NOTEBOOK_EDIT_MODE_DELETE:
		return []byte("delete"), nil
	default:
		return nil, nil
	}
}

func (x *NotebookEditMode) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = NotebookEditMode_NOTEBOOK_EDIT_MODE_UNSPECIFIED
	case "replace":
		*x = NotebookEditMode_NOTEBOOK_EDIT_MODE_REPLACE
	case "insert":
		*x = NotebookEditMode_NOTEBOOK_EDIT_MODE_INSERT
	case "delete":
		*x = NotebookEditMode_NOTEBOOK_EDIT_MODE_DELETE
	default:
		return fmt.Errorf("unknown NotebookEditMode text: %q", string(b))
	}
	return nil
}

func (x TodoStatus) MarshalText() ([]byte, error) {
	switch x {
	case TodoStatus_TODO_STATUS_PENDING:
		return []byte("pending"), nil
	case TodoStatus_TODO_STATUS_IN_PROGRESS:
		return []byte("in_progress"), nil
	case TodoStatus_TODO_STATUS_COMPLETED:
		return []byte("completed"), nil
	default:
		return nil, nil
	}
}

func (x *TodoStatus) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = TodoStatus_TODO_STATUS_UNSPECIFIED
	case "pending":
		*x = TodoStatus_TODO_STATUS_PENDING
	case "in_progress":
		*x = TodoStatus_TODO_STATUS_IN_PROGRESS
	case "completed":
		*x = TodoStatus_TODO_STATUS_COMPLETED
	default:
		return fmt.Errorf("unknown TodoStatus text: %q", string(b))
	}
	return nil
}
