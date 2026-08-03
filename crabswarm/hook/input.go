// Package hook holds crabswarm's Claude Code / Codex hook implementations
// and the helpers they share. [Parse] decodes a hook envelope into the
// fields the hook subcommands care about; the exec and path subpackages
// build on it.
package hook

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/ngicks/crabswarm/pkg/claudehook/types"
)

// Parsed is the decoded hook envelope shared across hook subcommands.
type Parsed struct {
	// Input is the parsed Claude / Codex hook envelope variant. Concrete
	// type depends on the event, e.g. [*types.PreToolUseHookInput].
	Input types.HookInput_Value
	// Event mirrors hook_event_name from the envelope.
	Event types.HookEvent
	// ToolName mirrors tool_name for tool events; empty otherwise.
	ToolName string
	// SessionID mirrors session_id; empty for events without one.
	SessionID string
	// Cwd mirrors the envelope's cwd; empty for events without one.
	Cwd string
	// Files holds every edited file path, resolved to absolute against Cwd.
	// A single-file Claude edit yields a one-element slice; a Codex
	// apply_patch may yield several. Empty when the event touches no file.
	Files []string
}

// Parse reads a Claude / Codex hook envelope into [Parsed]: it parses the
// typed input, extracts the envelope metadata, and resolves the edited
// file paths from the tool input. The exec and path subpackages build on
// it so envelope handling lives in one place.
func Parse(raw []byte) (Parsed, error) {
	var envelope struct {
		types.BaseHookInput
		HookEventName types.HookEvent `json:"hook_event_name"`
		ToolName      string          `json:"tool_name"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Parsed{}, fmt.Errorf("parsing hook input: %w", err)
	}

	var input types.HookInput
	if err := input.UnmarshalJSON(raw); err != nil {
		return Parsed{}, fmt.Errorf("parsing hook input: %w", err)
	}

	return Parsed{
		Input:     input.GetValue(),
		Event:     envelope.HookEventName,
		ToolName:  envelope.ToolName,
		SessionID: envelope.SessionID,
		Cwd:       envelope.Cwd,
		Files:     editedFiles(toolInputOf(input), envelope.Cwd),
	}, nil
}

// editedFiles returns the file paths an edit tool touched. Claude tools
// carry a single structured path (FileEditInput, FileWriteInput, ...);
// Codex's apply_patch embeds one or more paths in its patch text (decoded
// into CodexApplyPatchInput), which are resolved to absolute against cwd so
// downstream path handling matches Claude's absolute file_path. Returns nil
// for tools without a file.
func editedFiles(input types.ToolInputSchemas, cwd string) []string {
	if p := extractFilePath(input); p != "" {
		return []string{p}
	}
	patch, ok := input.GetCodexApplyPatchInput()
	if !ok {
		return nil
	}
	parsed := patch.Patch.EditedFiles()
	if len(parsed) == 0 {
		return nil
	}
	files := make([]string, len(parsed))
	for i, p := range parsed {
		files[i] = resolveAgainst(cwd, p)
	}
	return files
}

// resolveAgainst makes a patch-relative path absolute against cwd so
// filetype/root detection and rendered commands see the same absolute
// paths Claude already provides. Absolute paths and (as a fallback) paths
// with no cwd are returned unchanged.
func resolveAgainst(cwd, p string) string {
	if p == "" || cwd == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(cwd, p)
}

// toolInputOf returns the parsed ToolInputSchemas from a hook input,
// or nil when the event doesn't carry one.
func toolInputOf(input types.HookInput) types.ToolInputSchemas {
	switch v := input.GetValue().(type) {
	case *types.PreToolUseHookInput:
		return v.ToolInput
	case *types.PostToolUseHookInput:
		return v.ToolInput
	case *types.PostToolUseFailureHookInput:
		return v.ToolInput
	case *types.PermissionRequestHookInput:
		return v.ToolInput
	}
	return types.ToolInputSchemas{}
}

// extractFilePath returns the file path for tool inputs that carry one,
// or "" for tool inputs that don't (Bash, Glob, Grep, ...).
func extractFilePath(input types.ToolInputSchemas) string {
	switch v := input.GetValue().(type) {
	case *types.FileEditInput:
		return v.FilePath
	case *types.FileReadInput:
		return v.FilePath
	case *types.FileWriteInput:
		return v.FilePath
	case *types.NotebookEditInput:
		return v.NotebookPath
	}
	return ""
}
