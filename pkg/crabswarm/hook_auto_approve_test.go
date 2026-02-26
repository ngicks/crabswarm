package crabswarm

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
	"gotest.tools/v3/assert"
)

func makePermissionRequestInput(toolName string, toolInput map[string]any) string {
	input := map[string]any{
		"session_id":      "sess-1",
		"transcript_path": "/tmp/transcript.jsonl",
		"cwd":             "/test",
		"hook_event_name": "PermissionRequest",
		"tool_name":       toolName,
		"tool_input":      toolInput,
	}
	data, _ := json.Marshal(input)
	return string(data)
}

func isPassThrough(t *testing.T, err error) {
	t.Helper()
	var he *handler.HandlerError
	if !errors.As(err, &he) {
		t.Fatalf("expected HandlerError, got %T: %v", err, err)
	}
	if he.Output != nil {
		t.Fatal("expected nil output (pass-through), got non-nil")
	}
}

func isApproval(t *testing.T, err error) {
	t.Helper()
	var he *handler.HandlerError
	if !errors.As(err, &he) {
		t.Fatalf("expected HandlerError, got %T: %v", err, err)
	}
	if he.Output == nil {
		t.Fatal("expected non-nil output (approval), got nil")
	}
	if he.Output.HookSpecificOutput == nil {
		t.Fatal("expected hookSpecificOutput, got nil")
	}
	if he.Output.HookSpecificOutput.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
	assert.Equal(t, he.Output.HookSpecificOutput.Decision.Behavior, "allow")
}

func TestHookAutoApprove_MatchingToolAndDir(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	input := makePermissionRequestInput("Write", map[string]any{
		"file_path": filepath.Join(plansDir, "test.md"),
		"content":   "test",
	})

	err := HookAutoApprove(context.Background(), strings.NewReader(input), AutoApproveConfig{
		ToolPatterns: []string{`^(Read|Write|Edit)$`},
		UnderDirs:    []string{plansDir},
	})

	isApproval(t, err)
}

func TestHookAutoApprove_NonMatchingTool(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	input := makePermissionRequestInput("Bash", map[string]any{
		"command": "ls",
	})

	err := HookAutoApprove(context.Background(), strings.NewReader(input), AutoApproveConfig{
		ToolPatterns: []string{`^(Read|Write|Edit)$`},
		UnderDirs:    []string{plansDir},
	})

	isPassThrough(t, err)
}

func TestHookAutoApprove_FileOutsideDir(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	otherDir := filepath.Join(tmpDir, "other")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))
	assert.NilError(t, os.MkdirAll(otherDir, 0o755))

	input := makePermissionRequestInput("Write", map[string]any{
		"file_path": filepath.Join(otherDir, "code.go"),
		"content":   "package main",
	})

	err := HookAutoApprove(context.Background(), strings.NewReader(input), AutoApproveConfig{
		ToolPatterns: []string{`^(Read|Write|Edit)$`},
		UnderDirs:    []string{plansDir},
	})

	isPassThrough(t, err)
}

func TestHookAutoApprove_NoToolPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	// With no tool patterns, any tool matches.
	input := makePermissionRequestInput("Bash", map[string]any{
		"file_path": filepath.Join(plansDir, "test.md"),
	})

	err := HookAutoApprove(context.Background(), strings.NewReader(input), AutoApproveConfig{
		UnderDirs: []string{plansDir},
	})

	isApproval(t, err)
}

func TestHookAutoApprove_NoUnderDirs(t *testing.T) {
	// With no under dirs, only tool pattern is checked.
	input := makePermissionRequestInput("Write", map[string]any{
		"file_path": "/anywhere/file.md",
		"content":   "x",
	})

	err := HookAutoApprove(context.Background(), strings.NewReader(input), AutoApproveConfig{
		ToolPatterns: []string{`^Write$`},
	})

	isApproval(t, err)
}

func TestHookAutoApprove_NoConfig(t *testing.T) {
	// With no config at all, everything is approved.
	input := makePermissionRequestInput("Write", map[string]any{
		"file_path": "/anywhere/file.md",
	})

	err := HookAutoApprove(context.Background(), strings.NewReader(input), AutoApproveConfig{})

	isApproval(t, err)
}

func TestHookAutoApprove_NoFilePathInToolInput(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	// Bash tool has no file_path.
	input := makePermissionRequestInput("Bash", map[string]any{
		"command": "ls",
	})

	err := HookAutoApprove(context.Background(), strings.NewReader(input), AutoApproveConfig{
		ToolPatterns: []string{`.*`},
		UnderDirs:    []string{plansDir},
	})

	isPassThrough(t, err)
}

func TestHookAutoApprove_InvalidJSON(t *testing.T) {
	err := HookAutoApprove(context.Background(), strings.NewReader("not json"), AutoApproveConfig{})

	var he *handler.HandlerError
	assert.Assert(t, !errors.As(err, &he))
	assert.Assert(t, err != nil)
}

func TestHookAutoApprove_InvalidRegex(t *testing.T) {
	input := makePermissionRequestInput("Write", map[string]any{
		"file_path": "/foo/bar.md",
	})

	err := HookAutoApprove(context.Background(), strings.NewReader(input), AutoApproveConfig{
		ToolPatterns: []string{`[invalid`},
	})

	var he *handler.HandlerError
	assert.Assert(t, !errors.As(err, &he))
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "compiling tool pattern"))
}

func TestHookAutoApprove_MultipleToolPatterns(t *testing.T) {
	input := makePermissionRequestInput("Edit", map[string]any{
		"file_path": "/foo/bar.md",
	})

	// First pattern doesn't match, second does.
	err := HookAutoApprove(context.Background(), strings.NewReader(input), AutoApproveConfig{
		ToolPatterns: []string{`^Read$`, `^Edit$`},
	})

	isApproval(t, err)
}

func TestHookAutoApprove_MultipleUnderDirs(t *testing.T) {
	tmpDir := t.TempDir()
	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir2")
	assert.NilError(t, os.MkdirAll(dir1, 0o755))
	assert.NilError(t, os.MkdirAll(dir2, 0o755))

	// File is under dir2 (not dir1).
	input := makePermissionRequestInput("Write", map[string]any{
		"file_path": filepath.Join(dir2, "file.md"),
		"content":   "x",
	})

	err := HookAutoApprove(context.Background(), strings.NewReader(input), AutoApproveConfig{
		ToolPatterns: []string{`^Write$`},
		UnderDirs:    []string{dir1, dir2},
	})

	isApproval(t, err)
}

func TestExtractFilePath(t *testing.T) {
	tests := []struct {
		name  string
		input json.RawMessage
		want  string
	}{
		{"with file_path", json.RawMessage(`{"file_path":"/foo/bar.md","content":"x"}`), "/foo/bar.md"},
		{"without file_path", json.RawMessage(`{"command":"ls"}`), ""},
		{"empty input", json.RawMessage(``), ""},
		{"null input", nil, ""},
		{"invalid json", json.RawMessage(`not json`), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFilePath(tt.input)
			assert.Equal(t, got, tt.want)
		})
	}
}
