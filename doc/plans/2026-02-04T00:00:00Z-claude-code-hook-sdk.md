# Claude Code Hook SDK Implementation

**Date**: 2026-02-04
**Status**: Planning

## Overview

Create a comprehensive Go SDK for Claude Code hooks with type-safe tagged unions for hook events and tool inputs.

## Research Summary

### Hook Event Types (12 total)

| Event | Description | Matcher Support |
|-------|-------------|-----------------|
| `SessionStart` | Session begins/resumes | `startup`, `resume`, `clear`, `compact` |
| `UserPromptSubmit` | User submits prompt | None |
| `PreToolUse` | Before tool execution | Tool names |
| `PermissionRequest` | Permission dialog appears | Tool names |
| `PostToolUse` | After tool succeeds | Tool names |
| `PostToolUseFailure` | After tool fails | Tool names |
| `Notification` | Notification sent | `permission_prompt`, `idle_prompt`, etc. |
| `SubagentStart` | Subagent spawned | Agent types |
| `SubagentStop` | Subagent finishes | Agent types |
| `Stop` | Claude finishes responding | None |
| `PreCompact` | Before context compaction | `manual`, `auto` |
| `SessionEnd` | Session terminates | `clear`, `logout`, etc. |

### Tool Names (for PreToolUse/PostToolUse/etc.)

- `Bash`, `Read`, `Write`, `Edit`, `Glob`, `Grep`
- `WebFetch`, `WebSearch`, `Task`
- `NotebookEdit`, `TodoRead`, `TodoWrite`
- `AskUserQuestion`, `EnterPlanMode`, `ExitPlanMode`
- `Skill`, `TaskCreate`, `TaskUpdate`, `TaskGet`, `TaskList`
- `TaskOutput`, `TaskStop`
- MCP tools: `mcp__<server>__<tool>`

### Common Input Fields (all hooks)

```json
{
  "session_id": "string",
  "transcript_path": "string",
  "cwd": "string",
  "permission_mode": "default|plan|acceptEdits|dontAsk|bypassPermissions",
  "hook_event_name": "string"
}
```

## Implementation Plan

### 1. Define Hook Event Types

File: `hook/model/hook_event.go`

```go
type HookEventName string

const (
    HookEventSessionStart       HookEventName = "SessionStart"
    HookEventUserPromptSubmit   HookEventName = "UserPromptSubmit"
    HookEventPreToolUse         HookEventName = "PreToolUse"
    HookEventPermissionRequest  HookEventName = "PermissionRequest"
    HookEventPostToolUse        HookEventName = "PostToolUse"
    HookEventPostToolUseFailure HookEventName = "PostToolUseFailure"
    HookEventNotification       HookEventName = "Notification"
    HookEventSubagentStart      HookEventName = "SubagentStart"
    HookEventSubagentStop       HookEventName = "SubagentStop"
    HookEventStop               HookEventName = "Stop"
    HookEventPreCompact         HookEventName = "PreCompact"
    HookEventSessionEnd         HookEventName = "SessionEnd"
)
```

### 2. Define Tool Names and Tool Inputs

File: `hook/model/tools.go`

Complete the ToolName enum and add typed tool input structs:

```go
type ToolName string

const (
    ToolNameBash            ToolName = "Bash"
    ToolNameRead            ToolName = "Read"
    ToolNameWrite           ToolName = "Write"
    ToolNameEdit            ToolName = "Edit"
    ToolNameGlob            ToolName = "Glob"
    ToolNameGrep            ToolName = "Grep"
    ToolNameWebFetch        ToolName = "WebFetch"
    ToolNameWebSearch       ToolName = "WebSearch"
    ToolNameTask            ToolName = "Task"
    ToolNameNotebookEdit    ToolName = "NotebookEdit"
    ToolNameAskUserQuestion ToolName = "AskUserQuestion"
    ToolNameSkill           ToolName = "Skill"
    // ... etc
)

// Tool input structs
type BashInput struct {
    Command         string `json:"command"`
    Description     string `json:"description,omitempty"`
    Timeout         int    `json:"timeout,omitempty"`
    RunInBackground bool   `json:"run_in_background,omitempty"`
}

type ReadInput struct {
    FilePath string `json:"file_path"`
    Offset   int    `json:"offset,omitempty"`
    Limit    int    `json:"limit,omitempty"`
}

// ... other tool inputs
```

### 3. Refactor HookInput as Tagged Union

File: `hook/model/types.go`

```go
// HookInput is the base structure with common fields
type HookInput struct {
    SessionID      string          `json:"session_id"`
    TranscriptPath string          `json:"transcript_path"`
    Cwd            string          `json:"cwd"`
    PermissionMode string          `json:"permission_mode"`
    HookEventName  HookEventName   `json:"hook_event_name"`

    // Tool-related fields (PreToolUse, PostToolUse, etc.)
    ToolName   ToolName        `json:"tool_name,omitempty"`
    ToolInput  json.RawMessage `json:"tool_input,omitempty"`
    ToolUseID  string          `json:"tool_use_id,omitempty"`

    // PostToolUse specific
    ToolResponse json.RawMessage `json:"tool_response,omitempty"`

    // PostToolUseFailure specific
    Error       string `json:"error,omitempty"`
    IsInterrupt bool   `json:"is_interrupt,omitempty"`

    // UserPromptSubmit specific
    Prompt string `json:"prompt,omitempty"`

    // SessionStart specific
    Source    string `json:"source,omitempty"`
    Model     string `json:"model,omitempty"`
    AgentType string `json:"agent_type,omitempty"`

    // Notification specific
    Message          string `json:"message,omitempty"`
    Title            string `json:"title,omitempty"`
    NotificationType string `json:"notification_type,omitempty"`

    // Subagent specific
    AgentID            string `json:"agent_id,omitempty"`
    AgentTranscriptPath string `json:"agent_transcript_path,omitempty"`

    // Stop/SubagentStop specific
    StopHookActive bool `json:"stop_hook_active,omitempty"`

    // PreCompact specific
    Trigger            string `json:"trigger,omitempty"`
    CustomInstructions string `json:"custom_instructions,omitempty"`

    // SessionEnd specific
    Reason string `json:"reason,omitempty"`
}

// ParseToolInput parses ToolInput into the appropriate typed struct
func (h *HookInput) ParseToolInput() (any, error) {
    switch h.ToolName {
    case ToolNameBash:
        var input BashInput
        if err := json.Unmarshal(h.ToolInput, &input); err != nil {
            return nil, err
        }
        return &input, nil
    case ToolNameRead:
        var input ReadInput
        // ...
    // etc.
    }
}
```

### 4. Define HookOutput Variants

File: `hook/model/output.go`

```go
type HookOutput struct {
    Decision    string `json:"decision,omitempty"`
    Reason      string `json:"reason,omitempty"`
    Continue    bool   `json:"continue,omitempty"`
    StopReason  string `json:"stopReason,omitempty"`

    HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

type HookSpecificOutput struct {
    HookEventName          string          `json:"hookEventName"`
    PermissionDecision     string          `json:"permissionDecision,omitempty"`
    PermissionDecisionReason string        `json:"permissionDecisionReason,omitempty"`
    UpdatedInput           json.RawMessage `json:"updatedInput,omitempty"`
    AdditionalContext      string          `json:"additionalContext,omitempty"`
}
```

### 5. SDK Handler with Matcher Pattern

File: `hook/sdk/handler.go`

Design a matcher-based handler that groups tools by input data form:

```go
// ToolInputCategory represents different input structures
type ToolInputCategory int

const (
    // CategoryCommand - tools with command/description (Bash)
    CategoryCommand ToolInputCategory = iota
    // CategoryFilePath - tools operating on files (Read, Write, Edit, Glob, Grep)
    CategoryFilePath
    // CategoryWeb - tools with URL/query (WebFetch, WebSearch)
    CategoryWeb
    // CategoryTask - agent/subagent tools (Task)
    CategoryTask
    // CategoryUserInteraction - special UI tools (AskUserQuestion)
    CategoryUserInteraction
    // CategoryMCP - MCP server tools (mcp__*)
    CategoryMCP
    // CategoryOther - fallback for unknown tools
    CategoryOther
)

// Matcher provides callback-based handling grouped by input category
type Matcher struct {
    OnCommand         func(ctx context.Context, input *HookInput, cmd *BashInput) (*HookOutput, error)
    OnFilePath        func(ctx context.Context, input *HookInput, file FilePathInput) (*HookOutput, error)
    OnWeb             func(ctx context.Context, input *HookInput, web WebInput) (*HookOutput, error)
    OnTask            func(ctx context.Context, input *HookInput, task *TaskInput) (*HookOutput, error)
    OnUserInteraction func(ctx context.Context, input *HookInput, ask *AskUserQuestionInput) (*HookOutput, error)
    OnMCP             func(ctx context.Context, input *HookInput, mcp *MCPInput) (*HookOutput, error)
    OnOther           func(ctx context.Context, input *HookInput) (*HookOutput, error)
}

// FilePathInput is a union type for file-based tools
type FilePathInput struct {
    ToolName ToolName
    Read     *ReadInput
    Write    *WriteInput
    Edit     *EditInput
    Glob     *GlobInput
    Grep     *GrepInput
}

// WebInput is a union type for web tools
type WebInput struct {
    ToolName  ToolName
    WebFetch  *WebFetchInput
    WebSearch *WebSearchInput
}

// MCPInput represents MCP tool calls
type MCPInput struct {
    Server   string          // extracted from mcp__<server>__<tool>
    Tool     string
    RawInput json.RawMessage
}

func (m *Matcher) Match(ctx context.Context, input *HookInput) (*HookOutput, error) {
    category := input.Category()
    switch category {
    case CategoryCommand:
        if m.OnCommand != nil {
            cmd, _ := input.ParseToolInput()
            return m.OnCommand(ctx, input, cmd.(*BashInput))
        }
    case CategoryUserInteraction:
        if m.OnUserInteraction != nil {
            ask, _ := input.ParseToolInput()
            return m.OnUserInteraction(ctx, input, ask.(*AskUserQuestionInput))
        }
    // ... etc
    }
    if m.OnOther != nil {
        return m.OnOther(ctx, input)
    }
    return &HookOutput{Decision: DecisionAllow}, nil
}
```

This design:
- Groups tools by input structure (command, file path, web, etc.)
- Treats MCP tools same as built-in (need approval)
- Special branch for `AskUserQuestion` (has questions/options structure)
- Extensible for future tool categories

### 6. AskUserQuestion Special Structure

`AskUserQuestion` is unique - it presents choices to users rather than requesting approval:

```go
type AskUserQuestionInput struct {
    Questions []Question `json:"questions"`
    Answers   map[string]string `json:"answers,omitempty"` // user responses
    Metadata  *QuestionMetadata `json:"metadata,omitempty"`
}

type Question struct {
    Question    string    `json:"question"`
    Header      string    `json:"header"`
    Options     []Option  `json:"options"`
    MultiSelect bool      `json:"multiSelect"`
}

type Option struct {
    Label       string `json:"label"`
    Description string `json:"description"`
}
```

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `hook/model/types.go` | Modify | Refactor HookInput with all event fields, add Category() method |
| `hook/model/hook_event.go` | Create | Hook event name constants |
| `hook/model/tools.go` | Create | Tool names and typed tool input structs |
| `hook/model/tools_input.go` | Create | All tool input structs (BashInput, ReadInput, etc.) |
| `hook/model/output.go` | Create | HookOutput and HookSpecificOutput types |
| `hook/sdk/matcher.go` | Create | Matcher with category-based callbacks |
| `hook/sdk/category.go` | Create | Tool category classification logic |

## Verification

1. Build the project: `go build ./...`
2. Run existing tests (if any): `go test ./...`
3. Test with crabhook binary that it correctly parses various hook inputs
4. Manual test: Run `./scripts/install-hook.sh` and verify hooks fire correctly

## Notes

- MCP tools (`mcp__<server>__<tool>`) treated same as built-in tools - need approval
- Tool categories based on input structure:
  - **Command**: `Bash` (command string)
  - **FilePath**: `Read`, `Write`, `Edit`, `Glob`, `Grep` (file_path based)
  - **Web**: `WebFetch`, `WebSearch` (URL/query based)
  - **Task**: `Task` (subagent spawning)
  - **UserInteraction**: `AskUserQuestion` (special - presents choices)
  - **MCP**: Any `mcp__*` tool
  - **Other**: Fallback for new/unknown tools
- Design is extensible - new tool categories can be added as Claude Code evolves

## Codex Review Feedback (2026-02-04)

### Issues to Address

1. **Tool coverage incomplete** - Many tools listed (`EnterPlanMode`, `ExitPlanMode`, `Skill`, `TodoRead`, `TodoWrite`, `NotebookEdit`, `TaskCreate/Update/Get/List/Output/Stop`) don't map to categories. Need to either:
   - Add categories for these (e.g., `CategoryPlanning`, `CategoryTodo`, `CategoryNotebook`)
   - Explicitly route to `CategoryOther` with safe fallback

2. **Unsafe type assertions** - `Matcher.Match` ignores `ParseToolInput` errors and does unsafe casts. Fix:
   ```go
   func (m *Matcher) Match(ctx context.Context, input *HookInput) (*HookOutput, error) {
       category := input.Category()
       switch category {
       case CategoryCommand:
           if m.OnCommand != nil {
               cmd, err := input.ParseToolInput()
               if err != nil {
                   return nil, fmt.Errorf("parse bash input: %w", err)
               }
               bashInput, ok := cmd.(*BashInput)
               if !ok {
                   return nil, fmt.Errorf("unexpected type for bash input")
               }
               return m.OnCommand(ctx, input, bashInput)
           }
       // ...
       }
   }
   ```

3. **JSON tag inconsistency** - Some fields use camelCase (`stopReason`, `hookSpecificOutput`) while others use snake_case. Need to verify Claude Code's actual wire format and be consistent.

4. **Mega-struct design** - Consider using discriminator-based `UnmarshalJSON` with a `HookEvent` interface instead of one large struct with many optional fields.

5. **MCP tool parsing ambiguity** - If `<tool>` contains `__`, parsing is ambiguous. Use `strings.SplitN(name, "__", 3)` to handle this safely.

### Recommended Approach

For v1, keep the simpler mega-struct approach but:
- Add proper error handling in Matcher
- Handle nil `ToolInput` gracefully
- Route unknown tools to `CategoryOther` with raw JSON access
- Document that v2 could use discriminator-based design for true type safety
