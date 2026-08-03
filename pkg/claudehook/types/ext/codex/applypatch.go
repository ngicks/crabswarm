package codex

import (
	"encoding/json"
	"fmt"
	"strings"
)

// cloneRaw returns an independent copy of b, or nil when b is empty, so a
// preserved raw payload cannot be aliased to a caller's buffer. Decoded Codex
// values store the original bytes and re-emit them verbatim, preserving key
// order, whitespace, and number formatting. (Go's json.Marshal still applies
// its usual HTML escaping of '<', '>', and '&' to any marshaler output, so
// those bytes may differ; the JSON remains semantically identical.)
func cloneRaw(b json.RawMessage) json.RawMessage {
	if len(b) == 0 {
		return nil
	}
	out := make(json.RawMessage, len(b))
	copy(out, b)
	return out
}

// PatchKind is the kind of change a single apply_patch operation makes.
//
// Source: codex-rs/apply-patch/src/lib.rs (apply_patch envelope markers)
type PatchKind string

const (
	// PatchKindAdd creates a new file (*** Add File:).
	PatchKindAdd PatchKind = "add"
	// PatchKindUpdate edits an existing file in place (*** Update File:),
	// optionally renaming it (*** Move to:).
	PatchKindUpdate PatchKind = "update"
	// PatchKindDelete removes a file (*** Delete File:).
	PatchKindDelete PatchKind = "delete"
)

// PatchOperation is one file change inside an apply_patch envelope.
//
// Source: codex-rs/apply-patch/src/lib.rs (apply_patch envelope markers)
type PatchOperation struct {
	// Kind is the change kind: add, update, or delete.
	Kind PatchKind `json:"kind"`
	// Path is the file path exactly as written in the patch, so relative paths
	// stay relative to the patch's cwd.
	Path string `json:"path"`
	// MoveTo is the rename destination for an update operation, empty for adds,
	// deletes, and updates that don't rename.
	MoveTo string `json:"move_to,omitzero"`
}

// Patch is the parsed structure of an apply_patch envelope's operations, in the
// order they appear in the patch text.
//
// Source: codex-rs/apply-patch/src/parser.rs (ApplyPatchArgs),
// codex-rs/apply-patch/src/streaming_parser.rs (ENVIRONMENT_ID_MARKER)
type Patch struct {
	// EnvironmentId selects the target environment when the patch includes an
	// optional *** Environment ID: preamble, nil when absent.
	EnvironmentId *string          `json:"environment_id,omitzero"`
	Operations    []PatchOperation `json:"operations"`
}

// EditedFiles returns the paths of files that exist after the patch is applied:
// added files, updated files, and rename (*** Move to:) destinations, in order.
// Deleted files are excluded — there's nothing left to format or lint.
func (p Patch) EditedFiles() []string {
	var files []string
	for _, op := range p.Operations {
		switch op.Kind {
		case PatchKindAdd:
			files = append(files, op.Path)
		case PatchKindUpdate:
			if op.MoveTo != "" {
				files = append(files, op.MoveTo)
			} else {
				files = append(files, op.Path)
			}
		case PatchKindDelete:
			// Nothing remains to operate on.
		}
	}
	return files
}

// apply_patch envelope markers. Real content and context lines are always
// prefixed with '+', '-', or ' ', so a bare "*** " line is unambiguously a
// structural marker and can't be confused with patched file contents.
const (
	prefixAddFile       = "*** Add File: "
	prefixUpdateFile    = "*** Update File: "
	prefixDeleteFile    = "*** Delete File: "
	prefixMoveTo        = "*** Move to: "
	prefixEnvironmentId = "*** Environment ID:"
)

// ParsePatch decodes apply_patch envelope text into its ordered operations. A
// command that isn't an apply_patch envelope (or touches no files) yields a
// Patch with no operations.
//
// Source: codex-rs/apply-patch/src/parser.rs,
// codex-rs/apply-patch/src/streaming_parser.rs
func ParsePatch(command string) Patch {
	var ops []PatchOperation
	var environmentId *string
	// Index of the most recent update op so a following *** Move to: can set
	// its rename destination. -1 means the previous marker wasn't an update.
	lastUpdate := -1
	for line := range strings.Lines(command) {
		line = strings.TrimRight(line, "\r\n")
		switch {
		case strings.HasPrefix(line, prefixEnvironmentId):
			id := strings.TrimSpace(line[len(prefixEnvironmentId):])
			if id != "" {
				environmentId = new(id)
			}
		case strings.HasPrefix(line, prefixAddFile):
			ops = append(
				ops,
				PatchOperation{
					Kind: PatchKindAdd,
					Path: strings.TrimSpace(line[len(prefixAddFile):]),
				},
			)
			lastUpdate = -1
		case strings.HasPrefix(line, prefixUpdateFile):
			ops = append(
				ops,
				PatchOperation{
					Kind: PatchKindUpdate,
					Path: strings.TrimSpace(line[len(prefixUpdateFile):]),
				},
			)
			lastUpdate = len(ops) - 1
		case strings.HasPrefix(line, prefixMoveTo):
			if lastUpdate >= 0 {
				ops[lastUpdate].MoveTo = strings.TrimSpace(line[len(prefixMoveTo):])
			}
		case strings.HasPrefix(line, prefixDeleteFile):
			ops = append(
				ops,
				PatchOperation{
					Kind: PatchKindDelete,
					Path: strings.TrimSpace(line[len(prefixDeleteFile):]),
				},
			)
			lastUpdate = -1
		}
	}
	return Patch{EnvironmentId: environmentId, Operations: ops}
}

// ApplyPatchInput is a decoded Codex apply_patch tool_input: the JSON object
// {"command": "<patch text>"} that carries a file change as patch text.
//
// Source: codex-rs/core/src/tools/handlers/apply_patch.rs (apply_patch_payload_command)
type ApplyPatchInput struct {
	// Command is the raw apply_patch envelope text — the tool_input's
	// "command" field.
	Command string `json:"-"`
	// Patch is the parsed structure of Command.
	Patch Patch `json:"-"`

	// raw preserves the original tool_input bytes for a byte-exact re-marshal.
	raw json.RawMessage
}

// RawJSON returns a copy of the original tool_input bytes, or nil for a value
// that wasn't decoded from JSON.
func (i ApplyPatchInput) RawJSON() json.RawMessage { return cloneRaw(i.raw) }

// MarshalJSON re-emits the original tool_input bytes when present, otherwise
// builds the {"command": ...} object from Command.
func (i ApplyPatchInput) MarshalJSON() ([]byte, error) {
	if len(i.raw) > 0 {
		return cloneRaw(i.raw), nil
	}
	return json.Marshal(struct {
		Command string `json:"command"`
	}{Command: i.Command})
}

// UnmarshalJSON decodes the {"command": ...} object, preserves the raw bytes,
// and parses the patch text into Patch.
func (i *ApplyPatchInput) UnmarshalJSON(data []byte) error {
	var v struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("codex: decoding apply_patch tool input: %w", err)
	}
	i.raw = cloneRaw(data)
	i.Command = v.Command
	i.Patch = ParsePatch(v.Command)
	return nil
}
