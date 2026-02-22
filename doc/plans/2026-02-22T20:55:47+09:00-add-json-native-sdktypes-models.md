# Plan: Add JSON-native model types for sdk_types/v1

## Context

`protojson.Marshal` of `PreToolUseHookInput` produces `"tool_input": {"fileRead": {"file_path": "..."}}` because `ToolInput` is a protobuf `oneof`. Claude Code's actual JSON is flat: `"tool_input": {"file_path": "..."}` with `tool_name` as external discriminator. Both marshal and unmarshal are broken — unmarshal fails with `unknown field "file_path"`.

## Approach

Create Go structs under `pkg/claudesdk/model/` that use `encoding/json` tags and match Claude Code's JSON shape exactly. These are 1:1 with the proto definitions in `pkg/api/schema/proto/sdk_types/v1/` but flatten oneofs that protobuf can't express correctly.

`ToolInput` is modeled as a tagged union: a struct carrying a `Tag` (the `tool_name` string) and typed fields for each known tool. Custom `MarshalJSON`/`UnmarshalJSON` on the parent hook input types handle the flat JSON ↔ tagged union conversion.

## Files to Create

### 1. `pkg/claudesdk/model/tool_input.go` — ToolInput tagged union + all concrete types

**Tagged union struct:**

```go
type ToolInputTag string

const (
    ToolInputTagRead            ToolInputTag = "Read"
    ToolInputTagWrite           ToolInputTag = "Write"
    ToolInputTagEdit            ToolInputTag = "Edit"
    ToolInputTagBash            ToolInputTag = "Bash"
    ToolInputTagBashOutput      ToolInputTag = "BashOutput"
    ToolInputTagGlob            ToolInputTag = "Glob"
    ToolInputTagGrep            ToolInputTag = "Grep"
    ToolInputTagTask            ToolInputTag = "Task"
    ToolInputTagWebFetch        ToolInputTag = "WebFetch"
    ToolInputTagWebSearch       ToolInputTag = "WebSearch"
    ToolInputTagNotebookEdit    ToolInputTag = "NotebookEdit"
    ToolInputTagAskUserQuestion ToolInputTag = "AskUserQuestion"
    ToolInputTagExitPlanMode    ToolInputTag = "ExitPlanMode"
    ToolInputTagKillShell       ToolInputTag = "KillShell"
    ToolInputTagTodoWrite       ToolInputTag = "TodoWrite"
    ToolInputTagListMcpResources ToolInputTag = "ListMcpResources"
    ToolInputTagReadMcpResource  ToolInputTag = "ReadMcpResource"
    ToolInputTagMcp             ToolInputTag = "mcp"
    ToolInputTagUnknown         ToolInputTag = "unknown"
)

type ToolInput struct {
    Tag              ToolInputTag
    Read             *FileReadInput
    Write            *FileWriteInput
    Edit             *FileEditInput
    Bash             *BashInput
    BashOutput       *BashOutputInput
    Glob             *GlobInput
    Grep             *GrepInput
    Task             *AgentInput
    WebFetch         *WebFetchInput
    WebSearch        *WebSearchInput
    NotebookEdit     *NotebookEditInput
    AskUserQuestion  *AskUserQuestionInput
    ExitPlanMode     *ExitPlanModeInput
    KillShell        *KillShellInput
    TodoWrite        *TodoWriteInput
    ListMcpResources *ListMcpResourcesInput
    ReadMcpResource  *ReadMcpResourceInput
    Mcp              map[string]any // tool_name prefixed with "mcp__"
    Unknown          map[string]any // anything else
}
```

`MarshalJSON` on `ToolInput`: switch on `Tag`, marshal the corresponding non-nil field directly (flat).

`UnmarshalToolInput(tag ToolInputTag, data []byte) (ToolInput, error)`: factory function that creates a `ToolInput` with the right field populated by unmarshaling `data` into the concrete type. Determines tag from `tool_name`:
- Known tool names → corresponding tag/field
- `mcp__*` prefix → `ToolInputTagMcp`, unmarshal into `map[string]any`
- Anything else → `ToolInputTagUnknown`, unmarshal into `map[string]any`

`TagFromToolName(toolName string) ToolInputTag`: resolves `tool_name` string to tag. Handles aliases like `"TaskOutput"` → `ToolInputTagBashOutput`, `"TaskStop"` → `ToolInputTagKillShell`.

**Concrete input structs** (plain structs, matching JSON keys exactly):

```go
type FileReadInput struct {
    FilePath string `json:"file_path"`
    Offset   *int32 `json:"offset,omitempty"`
    Limit    *int32 `json:"limit,omitempty"`
}

type BashInput struct {
    Command         string `json:"command"`
    Timeout         *int32 `json:"timeout,omitempty"`
    Description     string `json:"description,omitempty"`
    RunInBackground *bool  `json:"run_in_background,omitempty"`
}
// ... all 18 types from tool_input.proto
```

### 2. `pkg/claudesdk/model/hook_input.go` — Hook input types

Hook inputs that contain `ToolInput` get custom `UnmarshalJSON`:

```go
type PreToolUseHookInput struct {
    SessionID      string    `json:"session_id"`
    TranscriptPath string    `json:"transcript_path"`
    Cwd            string    `json:"cwd"`
    PermissionMode string    `json:"permission_mode,omitempty"`
    ToolName       string    `json:"tool_name"`
    ToolInput      ToolInput `json:"-"` // custom handled
    HookEventName  string    `json:"hook_event_name"`
    ToolUseID      string    `json:"tool_use_id"`
}

func (p *PreToolUseHookInput) UnmarshalJSON(data []byte) error {
    type alias PreToolUseHookInput
    var raw struct {
        alias
        RawToolInput json.RawMessage `json:"tool_input"`
    }
    if err := json.Unmarshal(data, &raw); err != nil {
        return err
    }
    *p = PreToolUseHookInput(raw.alias)
    tag := TagFromToolName(p.ToolName)
    ti, err := UnmarshalToolInput(tag, raw.RawToolInput)
    if err != nil {
        return err
    }
    p.ToolInput = ti
    return nil
}

func (p PreToolUseHookInput) MarshalJSON() ([]byte, error) {
    type alias PreToolUseHookInput
    // Marshal ToolInput flat, merge with other fields
    tiBytes, _ := json.Marshal(p.ToolInput)
    var raw map[string]json.RawMessage
    // marshal alias fields, then set "tool_input" = tiBytes
    ...
}
```

Same pattern for: `PostToolUseHookInput`, `PostToolUseFailureHookInput`, `PermissionRequestHookInput`.

All other hook input types (NotificationHookInput, SessionStartHookInput, etc.) are plain structs — no oneof issue.

### 3. `pkg/claudesdk/model/hook_output.go` — Hook output types

`SyncHookJSONOutput` and hook-specific outputs flattened. `HookSpecificOutput` oneof → tagged union or flat struct with optional fields.

### 4. `pkg/claudesdk/model/tool_output.go` — Tool output types

Mirror `tool_output.proto` if it has similar oneof issues.

### 5. `pkg/claudesdk/model/convert.go` — Proto ↔ model conversion

```go
func PreToolUseHookInputToProto(m *PreToolUseHookInput) (*pb.PreToolUseHookInput, error)
func PreToolUseHookInputFromProto(p *pb.PreToolUseHookInput) (*PreToolUseHookInput, error)
```

## Files to Modify

- `pkg/crabswarm/hook.go` — Use `json.Unmarshal` into model type, then convert to proto for gRPC
- `pkg/crabswarm/hook_test.go` — Update to use model types for JSON, fix assertions

## Verification

1. `go build ./pkg/claudesdk/model/...`
2. `go test ./pkg/claudesdk/model/...` — round-trip tests: JSON → model → JSON matches, JSON → model → proto → model → JSON matches
3. `go test ./pkg/crabswarm/...` — existing tests pass with model-based unmarshal
4. Verify `json.Marshal` of a PreToolUseHookInput produces the flat format matching `hook_input.jsonl`
