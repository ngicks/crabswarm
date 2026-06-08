package exec

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	sdktypesv1 "github.com/ngicks/crabswarm/pkg/api/types/sdktypes/v1"
	"github.com/ngicks/crabswarm/pkg/filetype"
)

// parseInput reads a Claude / Codex hook envelope into [Data]: it parses
// the typed input, extracts the edited file(s), and detects filetype and
// module root from the first edited file.
func parseInput(raw []byte, ftCfg filetype.Config) (Data, error) {
	var envelope struct {
		sdktypesv1.BaseHookInput
		HookEventName sdktypesv1.HookEvent `json:"hook_event_name"`
		ToolName      string               `json:"tool_name"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Data{}, fmt.Errorf("parsing hook input: %w", err)
	}

	var input sdktypesv1.HookInput
	if err := input.UnmarshalJSON(raw); err != nil {
		return Data{}, fmt.Errorf("parsing hook input: %w", err)
	}

	files := editedFiles(toolInputOf(input), envelope.Cwd)
	var file string
	if len(files) > 0 {
		file = files[0]
	}
	d := Data{
		Input:     input.GetValue(),
		Event:     envelope.HookEventName,
		ToolName:  envelope.ToolName,
		SessionID: envelope.SessionID,
		Cwd:       envelope.Cwd,
		File:      file,
		Files:     files,
	}
	if d.File != "" {
		if name, ok := ftCfg.Detect(d.File); ok {
			d.Filetype = name
			if root, ok := ftCfg.FindRoot(name, d.File); ok {
				d.Root = root
			}
		}
	}
	return d, nil
}

// editedFiles returns the file paths an edit tool touched. Claude tools
// carry a single structured path (FileEditInput, FileWriteInput, ...);
// Codex's apply_patch embeds one or more paths in its patch text (decoded
// into CodexApplyPatchInput), which are resolved to absolute against cwd so
// downstream path handling matches Claude's absolute file_path. Returns nil
// for tools without a file.
func editedFiles(input sdktypesv1.ToolInputSchemas, cwd string) []string {
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
func toolInputOf(input sdktypesv1.HookInput) sdktypesv1.ToolInputSchemas {
	switch v := input.GetValue().(type) {
	case *sdktypesv1.PreToolUseHookInput:
		return v.ToolInput
	case *sdktypesv1.PostToolUseHookInput:
		return v.ToolInput
	case *sdktypesv1.PostToolUseFailureHookInput:
		return v.ToolInput
	case *sdktypesv1.PermissionRequestHookInput:
		return v.ToolInput
	}
	return sdktypesv1.ToolInputSchemas{}
}

// extractFilePath returns the file path for tool inputs that carry one,
// or "" for tool inputs that don't (Bash, Glob, Grep, ...).
func extractFilePath(input sdktypesv1.ToolInputSchemas) string {
	switch v := input.GetValue().(type) {
	case *sdktypesv1.FileEditInput:
		return v.FilePath
	case *sdktypesv1.FileReadInput:
		return v.FilePath
	case *sdktypesv1.FileWriteInput:
		return v.FilePath
	case *sdktypesv1.NotebookEditInput:
		return v.NotebookPath
	}
	return ""
}
