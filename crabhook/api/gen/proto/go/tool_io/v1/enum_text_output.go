package tool_iov1

import "fmt"

func (x BashOutputStatus) MarshalText() ([]byte, error) {
	switch x {
	case BashOutputStatus_BASH_OUTPUT_STATUS_RUNNING:
		return []byte("running"), nil
	case BashOutputStatus_BASH_OUTPUT_STATUS_COMPLETED:
		return []byte("completed"), nil
	case BashOutputStatus_BASH_OUTPUT_STATUS_FAILED:
		return []byte("failed"), nil
	default:
		return nil, nil
	}
}

func (x *BashOutputStatus) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = BashOutputStatus_BASH_OUTPUT_STATUS_UNSPECIFIED
	case "running":
		*x = BashOutputStatus_BASH_OUTPUT_STATUS_RUNNING
	case "completed":
		*x = BashOutputStatus_BASH_OUTPUT_STATUS_COMPLETED
	case "failed":
		*x = BashOutputStatus_BASH_OUTPUT_STATUS_FAILED
	default:
		return fmt.Errorf("unknown BashOutputStatus text: %q", string(b))
	}
	return nil
}

func (x NotebookEditType) MarshalText() ([]byte, error) {
	switch x {
	case NotebookEditType_NOTEBOOK_EDIT_TYPE_REPLACED:
		return []byte("replaced"), nil
	case NotebookEditType_NOTEBOOK_EDIT_TYPE_INSERTED:
		return []byte("inserted"), nil
	case NotebookEditType_NOTEBOOK_EDIT_TYPE_DELETED:
		return []byte("deleted"), nil
	default:
		return nil, nil
	}
}

func (x *NotebookEditType) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = NotebookEditType_NOTEBOOK_EDIT_TYPE_UNSPECIFIED
	case "replaced":
		*x = NotebookEditType_NOTEBOOK_EDIT_TYPE_REPLACED
	case "inserted":
		*x = NotebookEditType_NOTEBOOK_EDIT_TYPE_INSERTED
	case "deleted":
		*x = NotebookEditType_NOTEBOOK_EDIT_TYPE_DELETED
	default:
		return fmt.Errorf("unknown NotebookEditType text: %q", string(b))
	}
	return nil
}
