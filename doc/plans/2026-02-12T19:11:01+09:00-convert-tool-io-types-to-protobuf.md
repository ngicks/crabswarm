# Convert Claude Agent SDK Tool I/O TypeScript Interfaces to Protobuf

## Context

The Claude Agent SDK TypeScript reference at `https://platform.claude.com/docs/en/agent-sdk/typescript` defines `Tool Input Types` and `Tool Output Types` as TypeScript interfaces. We need to convert these into protobuf3 message definitions for use in the `crabhook` module, enabling typed gRPC communication for tool inputs/outputs.

Target files (both exist but are empty skeletons):
- `crabhook/api/schema/proto/tool-io/v1/input.proto`
- `crabhook/api/schema/proto/tool-io/v1/output.proto`

## Pre-requisite Fixes

1. **Fix `buf.yaml` path**: `crabhook/buf.yaml` references `api/scheme/proto` but directory is `api/schema/proto`
2. **Fix package name**: Both proto files use `package tool-io.v1;` — hyphens are invalid in protobuf; change to `package tool_io.v1;`

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| `Record<string, string>` | `map<string, string>` | Native protobuf map type |
| `Record<string, any>` | `google.protobuf.Struct` | Well-known type for arbitrary JSON |
| `unknown`/`any` | `google.protobuf.Value` / `google.protobuf.Struct` | Context-dependent |
| Optional primitives | `optional` keyword | Proto3 presence tracking |
| String literal unions | Enums with `UNSPECIFIED = 0` prefix | Matches project convention |
| Union types | `oneof` in wrapper message | Standard protobuf pattern |

## Implementation Steps

### Step 1: Fix `crabhook/buf.yaml`
Change `api/scheme/proto` → `api/schema/proto`

### Step 2: Write `input.proto`
17 tool input message types plus enums, wrapped in a `ToolInput` oneof union:

- `AgentInput` (Task), `AskUserQuestionInput`, `BashInput`, `BashOutputInput`
- `FileEditInput`, `FileReadInput`, `FileWriteInput`
- `GlobInput`, `GrepInput` (with `GrepOutputMode` enum)
- `KillShellInput`, `NotebookEditInput` (with `NotebookCellType`, `NotebookEditMode` enums)
- `WebFetchInput`, `WebSearchInput`
- `TodoWriteInput` (with `TodoStatus` enum, `TodoItem` message)
- `ExitPlanModeInput`, `ListMcpResourcesInput`, `ReadMcpResourceInput`

Shared sub-messages: `Question`, `QuestionOption`

### Step 3: Write `output.proto`
17 tool output message types wrapped in a `ToolOutput` oneof union:

- `TaskOutput` (with `UsageInfo`), `AskUserQuestionOutput` (reuses `Question` from input.proto)
- `BashOutput`, `BashOutputToolOutput` (with `BashOutputStatus` enum)
- `EditOutput`, `ReadOutput` (oneof: `TextFileOutput`, `ImageFileOutput`, `PDFFileOutput`, `NotebookFileOutput`)
- `WriteOutput`, `GlobOutput`
- `GrepOutput` (oneof: `GrepContentOutput`, `GrepFilesOutput`, `GrepCountOutput`)
- `KillBashOutput`, `NotebookEditOutput` (with `NotebookEditType` enum)
- `WebFetchOutput`, `WebSearchOutput` (with `WebSearchResult`)
- `TodoWriteOutput` (with `TodoStats`), `ExitPlanModeOutput`
- `ListMcpResourcesOutput` (with `McpResource`), `ReadMcpResourceOutput` (with `McpResourceContent`)

Cross-file reference: `output.proto` imports `input.proto` for `Question`/`QuestionOption`.

### Step 4: Validate
```bash
cd crabhook && buf lint
cd crabhook && buf generate
go build ./...
```

## Critical Files

- `crabhook/api/schema/proto/tool-io/v1/input.proto` — all tool input messages
- `crabhook/api/schema/proto/tool-io/v1/output.proto` — all tool output messages
- `crabhook/buf.yaml` — needs path fix
- `crabhook/buf.gen.yaml` — code gen config (no changes needed)
- `hook/api/scheme/proto/permission/v1/permission.proto` — reference for conventions
