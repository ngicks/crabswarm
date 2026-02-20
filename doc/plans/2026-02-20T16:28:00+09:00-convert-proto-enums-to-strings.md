# Convert Proto Enums to String Fields

## Context

The proto files under `pkg/api/schema/proto/sdk_types/v1/` use protobuf `enum` types to represent string-like values (hook events, permission modes, etc.). These enums are modeling what are actually plain strings in the Claude Code SDK's TypeScript types. The enum approach creates unnecessary conversion boilerplate between proto generated code and the Go model layer. Converting to `string` fields with comments documenting valid values simplifies the schema and eliminates the need for the `convert_enum.go` and all `enum_*.go` files.

## Enums to Convert (20 total)

### hook.proto (3 enums)
- `HookEvent`: PreToolUse, PostToolUse, PostToolUseFailure, Notification, UserPromptSubmit, SessionStart, SessionEnd, Stop, SubagentStart, SubagentStop, PreCompact, PermissionRequest
- `SessionStartSource`: startup, resume, clear, compact
- `PreCompactTrigger`: manual, auto

### tool_input.proto (4 enums)
- `GrepOutputMode`: content, files_with_matches, count
- `NotebookCellType`: code, markdown
- `NotebookEditMode`: replace, insert, delete
- `TodoStatus`: pending, in_progress, completed

### tool_output.proto (2 enums)
- `BashOutputStatus`: running, completed, failed
- `NotebookEditType`: replaced, inserted, deleted

### permission.proto (3 enums)
- `PermissionMode`: default, acceptEdits, bypassPermissions, plan
- `PermissionBehavior`: allow, deny, ask
- `PermissionUpdateDestination`: userSettings, projectSettings, localSettings, session

### other.proto (5 enums)
- `ApiKeySource`: user, project, org, temporary
- `SdkBeta`: context-1m-2025-08-07
- `ConfigScope`: local, user, project
- `McpServerConnectionStatus`: connected, failed, needs-auth, pending
- `CallToolResultContentType`: text, image, resource

### types.proto (2 enums)
- `SettingSource`: user, project, local
- `AgentModel`: sonnet, opus, haiku, inherit

### message.proto (1 enum)
- `SDKResultSubtype`: success, error_max_turns, error_during_execution, error_max_budget_usd, error_max_structured_output_retries

## Changes

### 1. Proto files — replace enums with string fields + comments

For each enum, remove the `enum` block and replace field references with `string`, adding a comment listing valid values **and a link to the SDK docs** where the original type is defined.

**Example** (hook.proto `SessionStartSource`):
```proto
// Before:
enum SessionStartSource {
  SESSION_START_SOURCE_UNSPECIFIED = 0;
  SESSION_START_SOURCE_STARTUP = 1;
  ...
}
message SessionStartHookInput {
  SessionStartSource source = 5;
}

// After:
message SessionStartHookInput {
  // Possible values: "startup", "resume", "clear", "compact"
  // See https://platform.claude.com/docs/en/agent-sdk/typescript#session-start-hook-input
  string source = 5;
}
```

**Doc links per enum** (use as comment on the field or the message that uses it):
| Enum | Doc Link |
|---|---|
| `HookEvent` | https://platform.claude.com/docs/en/agent-sdk/typescript#hook-event |
| `SessionStartSource` | https://platform.claude.com/docs/en/agent-sdk/typescript#session-start-hook-input |
| `PreCompactTrigger` | https://platform.claude.com/docs/en/agent-sdk/typescript#pre-compact-hook-input |
| `GrepOutputMode` | https://platform.claude.com/docs/en/agent-sdk/typescript#grep-input |
| `NotebookCellType` | https://platform.claude.com/docs/en/agent-sdk/typescript#notebook-edit-input |
| `NotebookEditMode` | https://platform.claude.com/docs/en/agent-sdk/typescript#notebook-edit-input |
| `TodoStatus` | https://platform.claude.com/docs/en/agent-sdk/typescript#todo-write-input |
| `BashOutputStatus` | https://platform.claude.com/docs/en/agent-sdk/typescript#bash-output-tool-output |
| `NotebookEditType` | https://platform.claude.com/docs/en/agent-sdk/typescript#notebook-edit-output |
| `PermissionMode` | https://platform.claude.com/docs/en/agent-sdk/typescript#permission-mode |
| `PermissionBehavior` | https://platform.claude.com/docs/en/agent-sdk/typescript#permission-types |
| `PermissionUpdateDestination` | https://platform.claude.com/docs/en/agent-sdk/typescript#permission-types |
| `ApiKeySource` | https://platform.claude.com/docs/en/agent-sdk/typescript#api-key-source |
| `SdkBeta` | https://platform.claude.com/docs/en/agent-sdk/typescript#sdk-beta |
| `ConfigScope` | https://platform.claude.com/docs/en/agent-sdk/typescript#config-scope |
| `McpServerConnectionStatus` | https://platform.claude.com/docs/en/agent-sdk/typescript#mcp-server-status |
| `CallToolResultContentType` | https://platform.claude.com/docs/en/agent-sdk/typescript#call-tool-result |
| `SettingSource` | https://platform.claude.com/docs/en/agent-sdk/typescript#setting-source |
| `AgentModel` | https://platform.claude.com/docs/en/agent-sdk/typescript#agent-definition |
| `SDKResultSubtype` | https://platform.claude.com/docs/en/agent-sdk/typescript#sdk-result-message |

**Files:**
- `pkg/api/schema/proto/sdk_types/v1/hook.proto` — remove `HookEvent`, `SessionStartSource`, `PreCompactTrigger`
- `pkg/api/schema/proto/sdk_types/v1/tool_input.proto` — remove `GrepOutputMode`, `NotebookCellType`, `NotebookEditMode`, `TodoStatus`
- `pkg/api/schema/proto/sdk_types/v1/tool_output.proto` — remove `BashOutputStatus`, `NotebookEditType`
- `pkg/api/schema/proto/sdk_types/v1/permission.proto` — remove `PermissionMode`, `PermissionBehavior`, `PermissionUpdateDestination`
- `pkg/api/schema/proto/sdk_types/v1/other.proto` — remove `ApiKeySource`, `SdkBeta`, `ConfigScope`, `McpServerConnectionStatus`, `CallToolResultContentType`
- `pkg/api/schema/proto/sdk_types/v1/types.proto` — remove `SettingSource`, `AgentModel`
- `pkg/api/schema/proto/sdk_types/v1/message.proto` — remove `SDKResultSubtype`

**Special cases:**
- `HookEvent` is only used as a standalone enum (not referenced as a message field) — just delete it entirely
- `repeated SdkBeta betas` in `Options` → `repeated string betas`
- `repeated SettingSource setting_sources` in `Options` → `repeated string setting_sources`

### 2. Regenerate protobuf Go code

```sh
cd pkg/api && buf generate
```

### 3. Delete Go enum definition files

Delete all `enum_*.go` files:
- `pkg/claudehook/model/enum_hook.go`
- `pkg/claudehook/model/enum_tool_input.go`
- `pkg/claudehook/model/enum_tool_output.go`
- `pkg/claudehook/model/enum_permission.go`
- `pkg/claudehook/model/enum_other.go`
- `pkg/claudehook/model/enum_types.go`
- `pkg/claudehook/model/enum_message.go`

### 4. Delete Go enum converter file

Delete `pkg/claudehook/model/convert_enum.go` entirely.

### 5. Update Go model structs — change typed enum fields to `string`

In Go model files, change fields from custom type (e.g., `SessionStartSource`) to `string`:

- `pkg/claudehook/model/hook.go`:
  - `SessionStartHookInput.Source`: `SessionStartSource` → `string`
  - `PreCompactHookInput.Trigger`: `PreCompactTrigger` → `string`

- `pkg/claudehook/model/tool_input.go`:
  - `GrepInput.OutputMode`: `GrepOutputMode` → `string`
  - `NotebookEditInput.CellType`: `NotebookCellType` → `string`
  - `NotebookEditInput.EditMode`: `NotebookEditMode` → `string`
  - `TodoItem.Status`: `TodoStatus` → `string`

- `pkg/claudehook/model/tool_output.go`:
  - `BashOutputToolOutput.Status`: `BashOutputStatus` → `string`
  - `NotebookEditOutput.EditType`: `NotebookEditType` → `string`

- `pkg/claudehook/model/permission.go`:
  - All `AddRulesUpdate`, `ReplaceRulesUpdate`, `RemoveRulesUpdate`: `Behavior` → `string`, `Destination` → `string`
  - `SetModeUpdate.Mode` → `string`, `Destination` → `string`
  - `AddDirectoriesUpdate.Destination` → `string`
  - `RemoveDirectoriesUpdate.Destination` → `string`

- `pkg/claudehook/model/other.go`:
  - `McpServerStatus.Status` → `string`
  - `CallToolResultContent.Type` → `string`

- `pkg/claudehook/model/types.go`:
  - `AgentDefinition.Model` → `string`
  - `Options.Betas` → `[]string`
  - `Options.PermissionMode` → `string`
  - `Options.SettingSources` → `[]string`

- `pkg/claudehook/model/message.go`:
  - `SDKResultMessage.Subtype` → `string`
  - `SDKSystemMessage.ApiKeySource` → `string`
  - `SDKSystemMessage.PermissionMode` → `string`

### 6. Update converter files — remove enum conversion calls

Replace all `EnumFromProto(pb.GetField())` with `pb.GetField()` and all `m.Field.ToProto()` with `m.Field`.

Files affected:
- `pkg/claudehook/model/convert_hook.go`
- `pkg/claudehook/model/convert_tool_input.go`
- `pkg/claudehook/model/convert_tool_output.go`
- `pkg/claudehook/model/convert_permission.go`
- `pkg/claudehook/model/convert_message.go`
- `pkg/claudehook/model/convert_other.go`
- `pkg/claudehook/model/convert_types.go` (also delete `sdkBetasFromProto`/`sdkBetasToProto`, `settingSourcesFromProto`/`settingSourcesToProto`)

### 7. Update tests

`pkg/claudehook/model/convert_test.go`:
- Delete `TestEnumRoundTrips`, `TestJSONMarshalProducesSDKNames`, `TestJSONMarshalHyphenEnum`
- Update `TestHookInputOneofConversion`: enum refs → string values
- Update `TestPermissionUpdateConversion`: enum refs → string values

### 8. Run `go mod tidy`

## Verification

1. `cd pkg/api && buf generate` — proto generates without errors
2. `go build ./...` — all packages compile
3. `go test ./pkg/claudehook/model/...` — tests pass
4. `go vet ./...` — no issues
