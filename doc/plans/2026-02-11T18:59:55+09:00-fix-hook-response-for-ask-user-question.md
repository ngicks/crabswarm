# Fix: crabswarm-hook I/O Does Not Match Official Protocol

**Source**: https://platform.claude.com/docs/en/agent-sdk/hooks

## Context

`crabswarm-hook` (`crabhook`) cannot correctly respond to hook events because both its input and output models are misaligned with the official Claude Agent SDK hooks protocol. The `AskUserQuestion` case is the most visible symptom, but the issue is systemic.

## Problems Found

### 1. Input field names are wrong

**File**: `hook/model/types.go`

| Current field (json tag) | Official SDK field | Notes |
|---|---|---|
| `hook_name` | `hook_event_name` | Wrong name |
| `tool_output` | `tool_response` | Wrong name |
| `tool_error` | `error` | Wrong name |
| _(missing)_ | `transcript_path` | Not modeled |
| _(missing)_ | `cwd` | Not modeled |
| _(missing)_ | `stop_hook_active` | Not modeled (Stop, SubagentStop) |
| _(missing)_ | `agent_transcript_path` | Not modeled (SubagentStop) |
| `session_start_reason` | `source` | Wrong name |
| `session_end_reason` | `reason` | Wrong name |
| _(missing)_ | `message` | Not modeled (Notification) |
| `user_prompt` | `prompt` | Wrong name |
| _(missing)_ | `permission_suggestions` | Not modeled (PermissionRequest) |
| `compact_trigger` | `trigger` | Wrong name |
| `subagent_type` | `agent_type` | Wrong name |
| `subagent_id` | `agent_id` | Wrong name |
| _(missing)_ | `is_interrupt` | Not modeled (PostToolUseFailure) |
| _(missing)_ | `custom_instructions` | Not modeled (PreCompact) |
| _(missing)_ | `title` | Not modeled (Notification) |

### 2. Output format is completely different

**File**: `hook/model/output.go`

**Current `HookOutput`** (wrong):
```json
{
  "decision": "allow|block|allow_always",
  "reason": "",
  "output_to_model": "",
  "suppress_output": false,
  "modified_input": {}
}
```

**Official SDK protocol**:
```json
{
  "continue": true,
  "stopReason": "",
  "suppressOutput": false,
  "systemMessage": "",
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow|deny|ask",
    "permissionDecisionReason": "",
    "updatedInput": {},
    "additionalContext": ""
  }
}
```

Key differences:
- Decision values: `block` → `deny`, `allow_always` → `ask` (different semantics)
- No `hookSpecificOutput` nesting
- `modified_input` → `updatedInput` (inside `hookSpecificOutput`)
- `output_to_model` → `systemMessage` (top-level) + `additionalContext` (hook-specific)
- Missing `continue`/`stopReason` for session control
- `suppress_output` → `suppressOutput` (camelCase)

### 3. gRPC PermissionResponse is too narrow

**File**: `hook/api/scheme/proto/permission/v1/permission.proto`

Only carries `decision` + `reason`. Cannot represent the full output format.

### 4. AskUserQuestion cannot be answered

No way to return `updatedInput` with `answers` populated. The prompter shows generic allow/block for all tools.

## Proposed Fix

### Step 1: Fix HookInput field names to match official protocol

**File**: `hook/model/types.go`

Update JSON tags and add missing fields:
- `hook_name` → `hook_event_name`
- `tool_output` → `tool_response`
- `tool_error` → `error`
- `session_start_reason` → `source`
- `session_end_reason` → `reason`
- `user_prompt` → `prompt`
- `compact_trigger` → `trigger`
- `subagent_type` → `agent_type`
- `subagent_id` → `agent_id`
- Add `transcript_path`, `cwd`, `stop_hook_active`, `agent_transcript_path`, `message`, `title`, `permission_suggestions`, `is_interrupt`, `custom_instructions`

### Step 2: Rewrite HookOutput to match official protocol

**File**: `hook/model/output.go`

Replace with the correct structure:
```go
type HookOutput struct {
    Continue        *bool              `json:"continue,omitempty"`
    StopReason      string             `json:"stopReason,omitempty"`
    SuppressOutput  bool               `json:"suppressOutput,omitempty"`
    SystemMessage   string             `json:"systemMessage,omitempty"`
    HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

type HookSpecificOutput struct {
    HookEventName            string         `json:"hookEventName"`
    PermissionDecision       string         `json:"permissionDecision,omitempty"`  // "allow", "deny", "ask"
    PermissionDecisionReason string         `json:"permissionDecisionReason,omitempty"`
    UpdatedInput             json.RawMessage `json:"updatedInput,omitempty"`
    AdditionalContext        string         `json:"additionalContext,omitempty"`
}
```

Update all helper functions (`Allow()`, `Block()`, etc.) to produce the new format.

### Step 3: Extend the protobuf schema

**File**: `hook/api/scheme/proto/permission/v1/permission.proto`

Update `PermissionResponse` to carry all output fields:
```protobuf
message PermissionResponse {
  string permission_decision = 1;          // "allow", "deny", "ask"
  string permission_decision_reason = 2;
  bool continue = 3;
  string stop_reason = 4;
  bool suppress_output = 5;
  string system_message = 6;
  string updated_input_json = 7;
  string additional_context = 8;
  string hook_event_name = 9;
}
```

Regenerate with `buf generate`.

**Compatibility note**: Both server and client must be updated together.

### Step 4: Update the client to produce correct output

**File**: `hook/cmd/crabhook/internal/root.go`

Map all gRPC response fields into the new `HookOutput` format.

### Step 5: Update the gRPC service implementation

**File**: `hook/api/impl/go/permission/v1/service.go`

Pass all fields from the prompter through to the gRPC response.

### Step 6: Refactor Prompter to return a richer response type

**File**: `hook/internal/server/prompt.go`

Change `Prompt()` to return a struct carrying all `HookOutput`-equivalent fields.

### Step 7: Add AskUserQuestion-aware prompting

**File**: `hook/internal/server/prompt.go`

When `tool_name == "AskUserQuestion"`:
- Parse `tool_input_json` as `AskUserQuestionInput`
- Display each question with numbered options
- Let operator select answers per question (or press Enter to skip)
- **Merge rule**: Parse original input, inject `answers` map, preserve all other fields
- **Skip semantics**: If operator allows without answering, do NOT set `updatedInput`, leaving the original input unchanged
- If parsing fails, show error and fall back to generic allow/deny flow

### Step 8 (lower priority): Generalize for other tools

Not in scope for this change, but the protocol extension enables it.

## Critical Files

| File | Change |
|------|--------|
| `hook/model/types.go` | Fix all JSON tags to match official protocol, add missing fields |
| `hook/model/output.go` | Rewrite output format with `hookSpecificOutput` nesting |
| `hook/model/types_test.go` | Update tests for new field names |
| `hook/model/output_test.go` | Update tests for new output format |
| `hook/sdk/matcher.go` | Update if it references old field names |
| `hook/sdk/matcher_test.go` | Update tests |
| `hook/api/scheme/proto/permission/v1/permission.proto` | Extend PermissionResponse |
| `hook/api/gen/go/permission/v1/` | Regenerate with `buf generate` |
| `hook/api/impl/go/permission/v1/service.go` | Pass through all fields |
| `hook/cmd/crabhook/internal/root.go` | Produce correct output format |
| `hook/internal/server/prompt.go` | Richer return type + AskUserQuestion UI |

## Verification

1. `buf generate` — regenerate protobuf
2. `go build ./...` — compiles
3. `go test ./...` — all tests pass (after updating)
4. **New tests**: AskUserQuestion prompting (answer merge, skip, partial answers, parse error fallback)
5. **Manual test**: Pipe AskUserQuestion JSON input to `crabhook`, verify operator can answer and output matches official format
6. **Manual test**: Verify other hook types produce correct output format
7. **Integration test**: Run `crabhook` as a Claude Code hook and verify it correctly allows/denies/answers

## Codex Review Feedback (from initial review, still applicable)

- Compatibility: Both server and client must update together
- Merge rule for `updatedInput`: Always merge into original, never drop fields
- Skip/unset semantics: If not answering, omit `updatedInput`
- Error handling: If parsing fails, fall back to generic flow
- Add automated tests for prompt logic and JSON merge correctness
