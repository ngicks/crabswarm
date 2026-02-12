# Redirect ExitPlanMode "Would you like to proceed?" Prompt

## Context

The hook system already intercepts `AskUserQuestion` tool calls and redirects them to a dedicated prompt (both plain text and TUI). The "Would you like to proceed?" prompt shown at the end of plan mode is triggered by the `ExitPlanMode` tool call. Currently, `ExitPlanMode` falls into `CategoryOther` and shows a generic Allow/Deny/Ask permission prompt with no plan-specific context. We want a dedicated prompt that displays the plan's `allowedPrompts` (permissions the plan will request) so the operator can make an informed approval decision.

## Plan

### 1. Add `ParseExitPlanModeInput` to `prompt_logic.go`

**File:** `hook/internal/server/prompt_logic.go`

Add structs and parser following the `ParseAskUserInput` pattern:

```go
type ExitPlanModeInput struct {
    AllowedPrompts []AllowedPrompt `json:"allowedPrompts,omitempty"`
    PushToRemote   bool            `json:"pushToRemote,omitempty"`
}

type AllowedPrompt struct {
    Tool   string `json:"tool"`
    Prompt string `json:"prompt"`
}

func ParseExitPlanModeInput(toolInputJSON string) (ExitPlanModeInput, error) { ... }
```

### 2. Add `promptExitPlanMode` to PlainPrompter

**File:** `hook/internal/server/prompt.go`

- In `Prompt()`, add check after AskUserQuestion routing (line 54):
  ```go
  if req.ToolName == "ExitPlanMode" && req.ToolInputJson != "" {
      return p.promptExitPlanMode(ctx, req)
  }
  ```
- Add `promptExitPlanMode()` method that:
  - Parses input with `ParseExitPlanModeInput`
  - Falls back to `promptStandard` on parse failure
  - Displays "Plan Approval" header
  - Lists each `AllowedPrompt` with `[Tool] Prompt` format
  - Shows Allow/Deny/Ask choices (same input handling as generic prompt)
  - Returns `BuildPermissionResponse` (no updatedInput needed)

### 3. Create TUI `exitPlanModel`

**File (new):** `hook/internal/tui/exitplan.go`

Follow `permissionModel` pattern from `permission.go`:
- Same Allow/Deny/Ask choice selection with j/k, a/d shortcuts
- Same deny-reason text input flow
- **View** differs: shows "Plan Approval" header and lists `AllowedPrompts` instead of raw JSON

### 4. Wire into TUI `rootModel`

**File:** `hook/internal/tui/tui.go`

- Add `stateExitPlan` to state enum
- Add `exitModel exitPlanModel` field to `rootModel`
- In `Update()`: delegate key events to `exitModel` when in `stateExitPlan`
- In `activateRequest()`: detect `ExitPlanMode` tool, parse input, activate `exitPlanModel`
- In `View()`: render `exitModel.View()` when in `stateExitPlan`

### 5. Tests

**File:** `hook/internal/server/prompt_logic_test.go`
- `TestParseExitPlanModeInput` - valid JSON with allowedPrompts
- `TestParseExitPlanModeInput_Empty` - empty/no allowedPrompts
- `TestParseExitPlanModeInput_Invalid` - malformed JSON

**File:** `hook/internal/server/prompt_test.go`
- `TestPromptExitPlanMode_Allow` - verify ALLOW decision
- `TestPromptExitPlanMode_Deny` - verify DENY with reason
- `TestPromptExitPlanMode_Ask` - verify ASK passthrough
- `TestPromptExitPlanMode_ParseFailFallback` - verify fallback to standard prompt

### 6. (Optional) Add `CategoryPlanMode` to tool taxonomy

**File:** `hook/model/tools.go`

```go
CategoryPlanMode ToolCategory = "plan_mode"
```
Map `ToolNameEnterPlanMode` and `ToolNameExitPlanMode` to this category. Add `OnPlanMode` handler to `Matcher` in `hook/sdk/matcher.go`.

## Files to modify

| File | Action |
|------|--------|
| `hook/internal/server/prompt_logic.go` | Add types + parser |
| `hook/internal/server/prompt.go` | Add routing + handler |
| `hook/internal/tui/exitplan.go` | New file - TUI model |
| `hook/internal/tui/tui.go` | Wire new state + model |
| `hook/internal/server/prompt_logic_test.go` | Add parser tests |
| `hook/internal/server/prompt_test.go` | Add prompt tests |
| `hook/model/tools.go` | (Optional) Add CategoryPlanMode |
| `hook/sdk/matcher.go` | (Optional) Add OnPlanMode handler |

## Verification

1. `go build ./...` - ensure compilation
2. `go test ./hook/internal/server/...` - run prompt logic and prompt tests
3. Manual test: configure hook for ExitPlanMode PreToolUse, trigger plan mode in Claude Code, verify dedicated prompt appears with allowedPrompts listed
