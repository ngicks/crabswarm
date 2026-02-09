package model

import "strings"

// ToolName represents the name of a Claude Code tool.
type ToolName string

// Core tool constants.
const (
	// Command execution
	ToolNameBash ToolName = "Bash"

	// File operations
	ToolNameRead  ToolName = "Read"
	ToolNameWrite ToolName = "Write"
	ToolNameEdit  ToolName = "Edit"
	ToolNameGlob  ToolName = "Glob"
	ToolNameGrep  ToolName = "Grep"

	// Web operations
	ToolNameWebFetch  ToolName = "WebFetch"
	ToolNameWebSearch ToolName = "WebSearch"

	// Agent/task operations
	ToolNameTask       ToolName = "Task"
	ToolNameTaskCreate ToolName = "TaskCreate"
	ToolNameTaskUpdate ToolName = "TaskUpdate"
	ToolNameTaskGet    ToolName = "TaskGet"
	ToolNameTaskList   ToolName = "TaskList"
	ToolNameTaskOutput ToolName = "TaskOutput"
	ToolNameTaskStop   ToolName = "TaskStop"

	// User interaction
	ToolNameAskUserQuestion ToolName = "AskUserQuestion"

	// Plan mode
	ToolNameEnterPlanMode ToolName = "EnterPlanMode"
	ToolNameExitPlanMode  ToolName = "ExitPlanMode"

	// Notebook operations
	ToolNameNotebookEdit ToolName = "NotebookEdit"

	// Todo operations
	ToolNameTodoRead  ToolName = "TodoRead"
	ToolNameTodoWrite ToolName = "TodoWrite"

	// Skills
	ToolNameSkill ToolName = "Skill"
)

// ToolCategory represents a category of tools for matching purposes.
type ToolCategory string

const (
	// CategoryCommand represents command execution tools (Bash).
	CategoryCommand ToolCategory = "command"

	// CategoryFilePath represents file operation tools (Read, Write, Edit, Glob, Grep).
	CategoryFilePath ToolCategory = "file_path"

	// CategoryWeb represents web operation tools (WebFetch, WebSearch).
	CategoryWeb ToolCategory = "web"

	// CategoryTask represents agent/task tools (Task, TaskCreate, etc.).
	CategoryTask ToolCategory = "task"

	// CategoryUserInteraction represents user interaction tools (AskUserQuestion).
	CategoryUserInteraction ToolCategory = "user_interaction"

	// CategoryMCP represents MCP server tools (mcp__*).
	CategoryMCP ToolCategory = "mcp"

	// CategoryOther represents all other tools.
	CategoryOther ToolCategory = "other"
)

// Category returns the category of this tool for matching purposes.
func (t ToolName) Category() ToolCategory {
	switch t {
	case ToolNameBash:
		return CategoryCommand
	case ToolNameRead, ToolNameWrite, ToolNameEdit, ToolNameGlob, ToolNameGrep:
		return CategoryFilePath
	case ToolNameWebFetch, ToolNameWebSearch:
		return CategoryWeb
	case ToolNameTask, ToolNameTaskCreate, ToolNameTaskUpdate, ToolNameTaskGet, ToolNameTaskList, ToolNameTaskOutput, ToolNameTaskStop:
		return CategoryTask
	case ToolNameAskUserQuestion:
		return CategoryUserInteraction
	default:
		if t.IsMCP() {
			return CategoryMCP
		}
		return CategoryOther
	}
}

// IsMCP returns true if this is an MCP tool (format: mcp__<server>__<tool>).
func (t ToolName) IsMCP() bool {
	return strings.HasPrefix(string(t), "mcp__")
}

// MCPInfo contains parsed information from an MCP tool name.
type MCPInfo struct {
	// Server is the MCP server name.
	Server string
	// Tool is the MCP tool name.
	Tool string
}

// ParseMCP parses an MCP tool name and returns server and tool info.
// Returns nil if this is not an MCP tool.
func (t ToolName) ParseMCP() *MCPInfo {
	if !t.IsMCP() {
		return nil
	}
	// Format: mcp__<server>__<tool>
	parts := strings.SplitN(string(t), "__", 3)
	if len(parts) != 3 {
		return nil
	}
	return &MCPInfo{
		Server: parts[1],
		Tool:   parts[2],
	}
}

// String returns the string representation of the tool name.
func (t ToolName) String() string {
	return string(t)
}
