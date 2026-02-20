# Add `json_name` Annotations & `hook_event_name` Fields to Proto Files

## Context

Proto message definitions in `pkg/api/schema/proto/sdk_types/v1/` were created with all field names converted to snake_case (proto convention). However, the TypeScript Claude Code SDK (the reference source at https://platform.claude.com/docs/en/agent-sdk/typescript) uses a **mix of camelCase and snake_case** for JSON field names. Protobuf's default JSON serialization converts snake_case to lowerCamelCase, which produces incorrect JSON for fields where the TS SDK expects snake_case.

**Goal**: 100% JSON field-name match with the TS SDK reference, with one known divergence: GrepInput flag fields (`-i`, `-n`, `-B`, `-A`, `-C`) use semantic names in proto (`case_insensitive`, `show_line_numbers`, etc.) since these are unusual JSON keys. Best-effort via `json_name` will be attempted; if `protojson` doesn't support `-` prefixed keys, the semantic names are kept and documented.

**Scope of changes**:
- `json_name` annotations: affects only JSON serialization (protojson), not binary wire format
- New `hook_event_name` fields: adds new fields to the protobuf schema/API surface (backward-compatible additive change, changes wire payload possibilities)

**Important context**: This Go code is a **consumer** of hook inputs (reads JSON from stdin via `protojson.Unmarshal`), not a producer. Claude Code (TS SDK) produces the JSON with `hook_event_name` already set. No Go construction/emission sites need updating — the new `hook_event_name` field is for deserialization only. The only producer code is in `pkg/claudehook/handler/handler.go` which outputs hook **responses** (not inputs) using plain `encoding/json` with its own Go structs — that code is unaffected by proto changes.

## Files Requiring Changes

### 1. `hook.proto` — Hook Input Messages

All 12 hook input messages share base fields that are snake_case in TS. Hook **output** messages use camelCase in TS → proto default matches → no changes to outputs.

#### A. Add `hook_event_name` field to each hook input message

The TS SDK includes a `hook_event_name` discriminator field in each hook input type. Add a `string hook_event_name` field (with `[json_name = "hook_event_name"]`) to each message, using the next available field number. The value is a fixed string per message type — producers must set it when constructing these messages.

| Message | Field Number | Required Value |
|---------|-------------|----------------|
| `PreToolUseHookInput` | 7 | `"PreToolUse"` |
| `PostToolUseHookInput` | 8 | `"PostToolUse"` |
| `PostToolUseFailureHookInput` | 9 | `"PostToolUseFailure"` |
| `NotificationHookInput` | 7 | `"Notification"` |
| `UserPromptSubmitHookInput` | 6 | `"UserPromptSubmit"` |
| `SessionStartHookInput` | 6 | `"SessionStart"` |
| `SessionEndHookInput` | 6 | `"SessionEnd"` |
| `StopHookInput` | 6 | `"Stop"` |
| `SubagentStartHookInput` | 7 | `"SubagentStart"` |
| `SubagentStopHookInput` | 6 | `"SubagentStop"` |
| `PreCompactHookInput` | 7 | `"PreCompact"` |
| `PermissionRequestHookInput` | 8 | `"PermissionRequest"` |

Example: `string hook_event_name = 7 [json_name = "hook_event_name"];`

**Value population**: Producers (code that constructs these messages for JSON output) must set `hook_event_name` to the correct literal value. Add proto comments documenting the expected value for each message type.

#### B. Add `json_name` annotations to existing fields

**Shared base fields** (in all 12 input messages):
- `session_id` → `[json_name = "session_id"]`
- `transcript_path` → `[json_name = "transcript_path"]`
- `permission_mode` → `[json_name = "permission_mode"]`

**Per-message additional fields:**
- `PreToolUseHookInput`: `tool_name`, `tool_input`
- `PostToolUseHookInput`: `tool_name`, `tool_input`, `tool_response`
- `PostToolUseFailureHookInput`: `tool_name`, `tool_input`, `is_interrupt`
- `StopHookInput`: `stop_hook_active`
- `SubagentStartHookInput`: `agent_id`, `agent_type`
- `SubagentStopHookInput`: `stop_hook_active`
- `PreCompactHookInput`: `custom_instructions`
- `PermissionRequestHookInput`: `tool_name`, `tool_input`, `permission_suggestions`
- `NotificationHookInput`, `UserPromptSubmitHookInput`, `SessionStartHookInput`, `SessionEndHookInput`: base fields only

### 2. `tool_input.proto` — Tool Input Messages

| Message | Fields needing `json_name` |
|---------|---------------------------|
| `AgentInput` | `subagent_type` |
| `BashInput` | `run_in_background` |
| `BashOutputInput` | `bash_id` |
| `FileEditInput` | `file_path`, `old_string`, `new_string`, `replace_all` |
| `FileReadInput` | `file_path` |
| `FileWriteInput` | `file_path` |
| `GrepInput` | `output_mode`, `head_limit`, plus flag fields (see below) |
| `NotebookEditInput` | `notebook_path`, `cell_id`, `new_source`, `cell_type`, `edit_mode` |
| `WebSearchInput` | `allowed_domains`, `blocked_domains` |

**No changes**: `Question` (`multiSelect` matches), `TodoItem` (`activeForm` matches), single-word fields.

**GrepInput flag fields**: TS uses `-i`, `-n`, `-B`, `-A`, `-C` as JSON keys. Proto field names can't start with `-`, but `json_name` can be any JSON key. Map them:

| Proto field | `json_name` |
|------------|-------------|
| `case_insensitive` | `[json_name = "-i"]` |
| `show_line_numbers` | `[json_name = "-n"]` |
| `before_context` | `[json_name = "-B"]` |
| `after_context` | `[json_name = "-A"]` |
| `context` | `[json_name = "-C"]` |

### 3. `tool_output.proto` — Tool Output Messages

| Message | Fields needing `json_name` |
|---------|---------------------------|
| `UsageInfo` | `input_tokens`, `output_tokens`, `cache_creation_input_tokens`, `cache_read_input_tokens` |
| `TaskOutput` | `total_cost_usd`, `duration_ms` |
| `EditOutput` | `file_path` |
| `TextFileOutput` | `total_lines`, `lines_returned` |
| `ImageFileOutput` | `mime_type`, `file_size` |
| `PDFPageImage` | `mime_type` |
| `PDFPage` | `page_number` |
| `PDFFileOutput` | `total_pages` |
| `NotebookCell` | `cell_type`, `execution_count` |
| `WriteOutput` | `bytes_written`, `file_path` |
| `GlobOutput` | `search_path` |
| `GrepMatch` | `line_number`, `before_context`, `after_context` |
| `GrepContentOutput` | `total_matches` |
| `KillBashOutput` | `shell_id` |
| `NotebookEditOutput` | `edit_type`, `cell_id`, `total_cells` |
| `WebFetchOutput` | `final_url`, `status_code` |
| `WebSearchOutput` | `total_results` |
| `TodoStats` | `in_progress` |

**No changes**: `BashOutput` (`exitCode`/`shellId` camelCase in TS), `BashOutputToolOutput` (`exitCode`), `McpResource`/`McpResourceContent` (`mimeType` camelCase).

### 4. `other.proto` (types) — Usage & Model Types

| Message | Fields needing `json_name` |
|---------|---------------------------|
| `ModelUsage` | `cost_usd` → **special**: `[json_name = "costUSD"]` (TS uses `costUSD` not `costUsd`) |
| `NonNullableUsage` | `input_tokens`, `output_tokens`, `cache_creation_input_tokens`, `cache_read_input_tokens` |
| `Usage` | `input_tokens`, `output_tokens`, `cache_creation_input_tokens`, `cache_read_input_tokens` |

**No changes**: `ModelUsage` token fields (TS uses camelCase `inputTokens` etc. → matches proto default), `SlashCommand`, `ModelInfo`, `McpServerStatus`, `AccountInfo`, `CallToolResult`.

### 5. `message.proto` — SDK Message Types

| Message | Fields needing `json_name` |
|---------|---------------------------|
| `SDKAssistantMessage` | `session_id`, `parent_tool_use_id` |
| `SDKUserMessage` | `session_id`, `parent_tool_use_id` |
| `SDKUserMessageReplay` | `session_id`, `parent_tool_use_id` |
| `SDKPermissionDenial` | `tool_name`, `tool_use_id`, `tool_input` |
| `SDKResultMessage` | `session_id`, `duration_ms`, `duration_api_ms`, `is_error`, `num_turns`, `total_cost_usd`, `permission_denials`, `structured_output` |
| `SDKSystemMessage` | `session_id`, `mcp_servers`, `slash_commands`, `output_style` |
| `SDKPartialAssistantMessage` | `session_id`, `parent_tool_use_id` |
| `CompactMetadata` | `pre_tokens` |
| `SDKCompactBoundaryMessage` | `session_id`, `compact_metadata` |

**No changes in SDKResultMessage**: `model_usage` (TS uses camelCase `modelUsage` → matches proto default).
**No changes in SDKSystemMessage**: `api_key_source` (TS `apiKeySource`), `permission_mode` (TS `permissionMode`) → both camelCase, match proto default.

### Files with NO Changes

- `permission.proto` — all TS fields use camelCase
- `sandbox.proto` — all TS fields use camelCase

## Verification

### 1. Build checks
- `cd pkg/api && buf lint` — STANDARD ruleset allows `json_name`
- `cd pkg/api && buf generate` — regenerate Go code
- `go build ./...` — ensure generated code compiles

### 2. Exhaustive round-trip JSON compatibility tests

Create a Go test file (near generated code) using `google.golang.org/protobuf/encoding/protojson` with **table-driven tests covering every changed message type** (not just one per category).

#### Test structure

For each proto message that had `json_name` annotations added, create a table entry with:
- A fully-populated proto message instance
- The expected JSON string (canonical fixture from TS SDK docs)
- Marshal the proto → JSON, assert all field names match expected keys
- Unmarshal the expected JSON → proto, assert all fields are populated correctly

#### Exhaustive coverage requirements

**All 12 hook input messages** — each must verify:
- All `json_name`-annotated fields serialize to snake_case
- `hook_event_name` field is present with correct literal value after marshal
- Unmarshal of JSON with `hook_event_name` populates the field correctly

**All tool input messages** (AgentInput, BashInput, BashOutputInput, FileEditInput, FileReadInput, FileWriteInput, GrepInput, NotebookEditInput, WebSearchInput):
- Each annotated field serializes to correct JSON key
- **GrepInput specifically**: verify `-i`, `-n`, `-B`, `-A`, `-C` work as JSON keys with `protojson` (both marshal and unmarshal) — this is a smoke test for an assumption that custom `json_name` keys starting with `-` are supported

**All tool output messages** with annotations (UsageInfo, TaskOutput, EditOutput, TextFileOutput, ImageFileOutput, PDFPageImage, PDFPage, PDFFileOutput, NotebookCell, WriteOutput, GlobOutput, GrepMatch, GrepContentOutput, KillBashOutput, NotebookEditOutput, WebFetchOutput, WebSearchOutput, TodoStats)

**Usage types** (ModelUsage with `costUSD`, NonNullableUsage, Usage)

**All SDK message types** with annotations (SDKAssistantMessage, SDKUserMessage, SDKUserMessageReplay, SDKPermissionDenial, SDKResultMessage, SDKSystemMessage, SDKPartialAssistantMessage, CompactMetadata, SDKCompactBoundaryMessage)

#### GrepInput `-i`/`-B` etc. (known divergence)

Attempt `json_name = "-i"` etc. in proto and verify with a test. If `protojson` does not support these keys, keep the semantic proto names (`case_insensitive`, `before_context`, etc.) and document the divergence. This is the one accepted deviation from 100% match.

## Approximate Scale

~120 `json_name` annotations + 12 new `hook_event_name` fields across 6 proto files, plus 1 exhaustive test file.
