package model

import (
	"encoding/json"
	"testing"
)

func TestHookInputParseBashInput(t *testing.T) {
	input := &HookInput{
		HookEventName:  HookEventPreToolUse,
		SessionID: "test-session",
		ToolName:  ToolNameBash,
		ToolInput: json.RawMessage(`{"command":"ls -la","description":"List files"}`),
	}

	parsed, err := input.ParseToolInput()
	if err != nil {
		t.Fatalf("ParseToolInput failed: %v", err)
	}

	bashInput, ok := parsed.(*BashInput)
	if !ok {
		t.Fatalf("Expected *BashInput, got %T", parsed)
	}

	if bashInput.Command != "ls -la" {
		t.Errorf("Expected command 'ls -la', got '%s'", bashInput.Command)
	}
	if bashInput.Description != "List files" {
		t.Errorf("Expected description 'List files', got '%s'", bashInput.Description)
	}
}

func TestHookInputParseReadInput(t *testing.T) {
	input := &HookInput{
		HookEventName:  HookEventPreToolUse,
		SessionID: "test-session",
		ToolName:  ToolNameRead,
		ToolInput: json.RawMessage(`{"file_path":"/tmp/test.txt","offset":10,"limit":100}`),
	}

	parsed, err := input.ParseReadInput()
	if err != nil {
		t.Fatalf("ParseReadInput failed: %v", err)
	}

	if parsed.FilePath != "/tmp/test.txt" {
		t.Errorf("Expected file_path '/tmp/test.txt', got '%s'", parsed.FilePath)
	}
	if parsed.Offset != 10 {
		t.Errorf("Expected offset 10, got %d", parsed.Offset)
	}
	if parsed.Limit != 100 {
		t.Errorf("Expected limit 100, got %d", parsed.Limit)
	}
}

func TestHookInputParseMCPInput(t *testing.T) {
	input := &HookInput{
		HookEventName:  HookEventPreToolUse,
		SessionID: "test-session",
		ToolName:  ToolName("mcp__serena__find_symbol"),
		ToolInput: json.RawMessage(`{"name_path":"Foo","relative_path":"src/main.go"}`),
	}

	parsed, err := input.ParseMCPInput()
	if err != nil {
		t.Fatalf("ParseMCPInput failed: %v", err)
	}

	if parsed.Server != "serena" {
		t.Errorf("Expected server 'serena', got '%s'", parsed.Server)
	}
	if parsed.Tool != "find_symbol" {
		t.Errorf("Expected tool 'find_symbol', got '%s'", parsed.Tool)
	}
	if parsed.Parameters["name_path"] != "Foo" {
		t.Errorf("Expected name_path 'Foo', got '%v'", parsed.Parameters["name_path"])
	}
}

func TestToolNameCategory(t *testing.T) {
	tests := []struct {
		name     ToolName
		expected ToolCategory
	}{
		{ToolNameBash, CategoryCommand},
		{ToolNameRead, CategoryFilePath},
		{ToolNameWrite, CategoryFilePath},
		{ToolNameEdit, CategoryFilePath},
		{ToolNameGlob, CategoryFilePath},
		{ToolNameGrep, CategoryFilePath},
		{ToolNameWebFetch, CategoryWeb},
		{ToolNameWebSearch, CategoryWeb},
		{ToolNameTask, CategoryTask},
		{ToolNameTaskCreate, CategoryTask},
		{ToolNameAskUserQuestion, CategoryUserInteraction},
		{ToolName("mcp__serena__find_symbol"), CategoryMCP},
		{ToolName("Unknown"), CategoryOther},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			got := tt.name.Category()
			if got != tt.expected {
				t.Errorf("Category() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestToolNameParseMCP(t *testing.T) {
	tests := []struct {
		name     ToolName
		wantNil  bool
		server   string
		tool     string
	}{
		{ToolName("mcp__serena__find_symbol"), false, "serena", "find_symbol"},
		{ToolName("mcp__context7__query-docs"), false, "context7", "query-docs"},
		{ToolNameBash, true, "", ""},
		{ToolName("mcp__invalid"), true, "", ""}, // Only 2 parts
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			got := tt.name.ParseMCP()
			if tt.wantNil {
				if got != nil {
					t.Errorf("ParseMCP() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("ParseMCP() = nil, want non-nil")
			}
			if got.Server != tt.server {
				t.Errorf("Server = %v, want %v", got.Server, tt.server)
			}
			if got.Tool != tt.tool {
				t.Errorf("Tool = %v, want %v", got.Tool, tt.tool)
			}
		})
	}
}

func TestHookInputCategory(t *testing.T) {
	tests := []struct {
		name     string
		input    *HookInput
		expected ToolCategory
	}{
		{
			name:     "Bash tool",
			input:    &HookInput{ToolName: ToolNameBash},
			expected: CategoryCommand,
		},
		{
			name:     "No tool",
			input:    &HookInput{},
			expected: CategoryOther,
		},
		{
			name:     "MCP tool",
			input:    &HookInput{ToolName: ToolName("mcp__test__tool")},
			expected: CategoryMCP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.Category()
			if got != tt.expected {
				t.Errorf("Category() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHookInputParseNilToolInput(t *testing.T) {
	input := &HookInput{
		HookEventName:  HookEventPreToolUse,
		SessionID: "test-session",
		ToolName:  ToolNameBash,
		ToolInput: nil,
	}

	parsed, err := input.ParseToolInput()
	if err != nil {
		t.Fatalf("ParseToolInput failed: %v", err)
	}
	if parsed != nil {
		t.Errorf("Expected nil, got %v", parsed)
	}
}
