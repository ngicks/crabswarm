// Package codexhook interprets Codex-specific hook payloads that don't map
// onto the Claude Agent SDK shapes.
//
// Codex edits files through a single apply_patch tool whose tool_input
// carries the whole change as patch text in a "command" field, rather than
// Claude's per-file Edit/Write tools with a structured file_path. This
// package parses that tool input into the affected file paths so the rest
// of crabswarm can treat a Codex edit like a Claude one.
package codexhook

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ApplyPatchToolName is the tool_name Codex reports for file edits applied
// via the apply_patch envelope.
const ApplyPatchToolName = "apply_patch"

// ApplyPatchInput is a decoded Codex apply_patch tool input.
type ApplyPatchInput struct {
	// Command is the raw apply_patch envelope text — the tool_input's
	// "command" field.
	Command string
	// Files holds the paths of files that exist after the patch is applied:
	// added files, updated files, and rename (*** Move to:) destinations,
	// in the order they appear. Deleted files are excluded — there's
	// nothing left to format or lint. Paths are exactly as written in the
	// patch, so relative paths stay relative to the patch's cwd.
	Files []string
}

// ParseApplyPatchInput decodes a Codex apply_patch tool_input — the JSON
// object that carries the change as patch text in its "command" field —
// and extracts the affected files. It errors only when toolInput isn't
// such an object; a well-formed input that touches no surviving files
// (deletes only, or a command that isn't an apply_patch envelope) yields
// an empty Files and no error.
func ParseApplyPatchInput(toolInput json.RawMessage) (ApplyPatchInput, error) {
	var v struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(toolInput, &v); err != nil {
		return ApplyPatchInput{}, fmt.Errorf("codexhook: decoding apply_patch tool input: %w", err)
	}
	return ApplyPatchInput{Command: v.Command, Files: editedFiles(v.Command)}, nil
}

// apply_patch envelope markers. Real content and context lines are always
// prefixed with '+', '-', or ' ', so a bare "*** " line is unambiguously a
// structural marker and can't be confused with patched file contents.
const (
	prefixAddFile    = "*** Add File: "
	prefixUpdateFile = "*** Update File: "
	prefixDeleteFile = "*** Delete File: "
	prefixMoveTo     = "*** Move to: "
)

// editedFiles scans apply_patch envelope text for the files that exist
// after it's applied — see [ApplyPatchInput.Files].
func editedFiles(patch string) []string {
	var files []string
	// Index in files of the most recent *** Update File:, so a following
	// *** Move to: can rewrite it to the rename destination. -1 means the
	// previous marker wasn't an update, so a stray Move to: is ignored.
	lastUpdate := -1
	for line := range strings.Lines(patch) {
		line = strings.TrimRight(line, "\r\n")
		switch {
		case strings.HasPrefix(line, prefixAddFile):
			files = append(files, strings.TrimSpace(line[len(prefixAddFile):]))
			lastUpdate = -1
		case strings.HasPrefix(line, prefixUpdateFile):
			files = append(files, strings.TrimSpace(line[len(prefixUpdateFile):]))
			lastUpdate = len(files) - 1
		case strings.HasPrefix(line, prefixMoveTo):
			if lastUpdate >= 0 {
				files[lastUpdate] = strings.TrimSpace(line[len(prefixMoveTo):])
			}
		case strings.HasPrefix(line, prefixDeleteFile):
			lastUpdate = -1
		}
	}
	return files
}
