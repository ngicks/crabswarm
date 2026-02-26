# Add Plan Iteration Hook

## Context

When Claude Code exits plan mode via `ExitPlanMode`, there is no automated review step. We want each plan version to be archived and sent to an external reviewer (e.g. `codex exec`). The hook saves the review output alongside the plan snapshot for the user's reference. The user then decides whether to iterate based on the review results — the hook itself does not block or deny the ExitPlanMode action.

## Architecture: Two Independent Hooks (No Daemon)

All hooks are stateless — no gRPC service needed. The last plan file path is resolved by parsing the transcript JSONL file (`transcript_path` from hook input).

### Hook 1: `hook plan-callback` (PreToolUse on ExitPlanMode)
- Parses `transcript_path` JSONL to find the last Write to `${CLAUDE_CONFIG_DIR:-$HOME/.claude}/plans/*.md`
- Creates/updates structured plan directory with iteration snapshots
- Runs review callback with env vars (file paths)
- Exit 0, no output — the user decides whether to iterate based on callback results
- Callback stdout/stderr are logged; iteration snapshots are saved for user reference

### Hook 2: `hook auto-approve` (PermissionRequest)
- Generic auto-approval hook for PermissionRequest events
- Flags: `--tool` (tool name regex to match), `--under` (directory — approve if `tool_input.file_path` is under this dir)
- Both flags can be repeated; approval requires all conditions to match
- For plan iteration: configured with `--tool "^(Read|Write|Edit)$" --under "${CLAUDE_PROJECT_DIR}/${CRABSWARM_PLAN_OUTPUT_DIR}"`
- `CRABSWARM_PLAN_OUTPUT_DIR` **must be a relative path** — it is concatenated with `CLAUDE_PROJECT_DIR`. An absolute path would produce an unwanted long path like `/project/root//absolute/path`
- If `CRABSWARM_PLAN_OUTPUT_DIR` is not set, defaults to `doc/plans`
- Ensures the iteration loop isn't blocked by permission prompts when Claude reads/writes/edits plan files

## Implementation Steps

### 1. Proto: Add PermissionRequest output to `pkg/api/schema/proto/sdk_types/v1/hook.proto`

Move `PermissionResult`, `AllowResult`, and `DenyResult` from `types.proto` to `hook.proto` (near `HookSpecificOutput`). These already model the same structure as the PermissionRequest hook decision:
- `AllowResult` has `updated_input: ToolInput` and `updated_permissions: repeated PermissionUpdate` — matches `decision.behavior = "allow"` with `updatedInput` and `updatedPermissions`
- `DenyResult` has `message: string` and `optional bool interrupt` — matches `decision.behavior = "deny"` with `message` and `interrupt`

Add a new `PermissionRequestHookSpecificOutput` message that uses `PermissionResult` as its decision field. This does NOT have a corresponding definition in the TypeScript SDK type definitions — it is derived from the [Claude Code hooks reference](https://code.claude.com/docs/en/hooks#permissionrequest-decision-control). Add a doc comment noting this.

```protobuf
// --- Moved from types.proto ---

// PermissionResult represents the result of a permission check.
// Copied from https://platform.claude.com/docs/en/agent-sdk/typescript#permission-result
message PermissionResult {
  oneof result {
    AllowResult allow = 1;
    DenyResult deny = 2;
  }
}

// AllowResult represents an allowed permission result.
message AllowResult {
  ToolInput updated_input = 1;
  repeated PermissionUpdate updated_permissions = 2;
}

// DenyResult represents a denied permission result.
message DenyResult {
  string message = 1;
  optional bool interrupt = 2;
}

// --- New ---

// PermissionRequestHookSpecificOutput represents hook-specific output for permission-request hooks.
// NOTE: This message does not have a corresponding definition in the TypeScript Agent SDK type definitions.
// It is derived from the Claude Code hooks reference:
// https://code.claude.com/docs/en/hooks#permissionrequest-decision-control
message PermissionRequestHookSpecificOutput {
  optional PermissionResult decision = 1;
}
```

Add `PermissionRequestHookSpecificOutput` as a new variant in the `HookSpecificOutput` oneof:

```protobuf
message HookSpecificOutput {
  oneof output {
    PreToolUseHookSpecificOutput pre_tool_use = 1;
    UserPromptSubmitHookSpecificOutput user_prompt_submit = 2;
    SessionStartHookSpecificOutput session_start = 3;
    PostToolUseHookSpecificOutput post_tool_use = 4;
    PermissionRequestHookSpecificOutput permission_request = 5;
  }
}
```

Remove `PermissionResult`, `AllowResult`, and `DenyResult` from `types.proto` (they are now in `hook.proto`). `hook.proto` already imports `permission.proto` and `tool_input.proto` which provide `PermissionUpdate` and `ToolInput`.

Then regenerate Go code.

### 1b. JSON models: Add PermissionRequest output types to `pkg/claudesdk/models/hook_output.go`

Add pure JSON model types matching the proto definitions:

```go
// PermissionRequestDecision represents the decision object for PermissionRequest hooks.
// NOTE: This type does not have a corresponding definition in the TypeScript Agent SDK type definitions.
// It is derived from the Claude Code hooks reference:
// https://code.claude.com/docs/en/hooks#permissionrequest-decision-control
type PermissionRequestDecision struct {
    Behavior           string          `json:"behavior"`                      // "allow" or "deny"
    UpdatedInput       json.RawMessage `json:"updatedInput,omitempty"`        // for "allow": modifies tool input
    UpdatedPermissions json.RawMessage `json:"updatedPermissions,omitempty"`  // for "allow": applies permission rules
    Message            *string         `json:"message,omitempty"`             // for "deny": tells Claude why
    Interrupt          *bool           `json:"interrupt,omitempty"`           // for "deny": if true, stops Claude
}

// PermissionRequestHookSpecificOutput represents hook-specific output for permission-request hooks.
// NOTE: This type does not have a corresponding definition in the TypeScript Agent SDK type definitions.
// It is derived from the Claude Code hooks reference:
// https://code.claude.com/docs/en/hooks#permissionrequest-decision-control
type PermissionRequestHookSpecificOutput struct {
    Decision *PermissionRequestDecision `json:"decision,omitempty"`
}
```

Add PermissionRequest fields to the flat `HookSpecificOutput`:

```go
type HookSpecificOutput struct {
    // hookEventName discriminator (required for PermissionRequest, PreToolUse, etc.)
    HookEventName *string `json:"hookEventName,omitempty"`
    // PreToolUse fields
    PermissionDecision       *string         `json:"permissionDecision,omitempty"`
    PermissionDecisionReason *string         `json:"permissionDecisionReason,omitempty"`
    UpdatedInput             json.RawMessage `json:"updatedInput,omitempty"`
    // Shared field for UserPromptSubmit, SessionStart, PostToolUse
    AdditionalContext *string `json:"additionalContext,omitempty"`
    // PermissionRequest fields
    Decision *PermissionRequestDecision `json:"decision,omitempty"`
}
```

### 2. Bi-di conversion: Add `SyncHookJSONOutput` conversion to `pkg/claudesdk/models/convert.go`

Add `(m *SyncHookJSONOutput) ToProto() (*pb.SyncHookJSONOutput, error)` and `(m *SyncHookJSONOutput) FromProto(p *pb.SyncHookJSONOutput) error` methods. The proto and JSON representations are slightly different (proto uses oneof for `HookSpecificOutput`, JSON is flat) but can be converted bi-directionally.

For PermissionRequest, the conversion maps between the flat JSON `PermissionRequestDecision` (with `behavior` discriminator) and the proto `PermissionResult` oneof (`AllowResult`/`DenyResult`):
- JSON `behavior: "allow"` + `updatedInput` + `updatedPermissions` ↔ proto `PermissionResult.AllowResult`
- JSON `behavior: "deny"` + `message` + `interrupt` ↔ proto `PermissionResult.DenyResult`
- `AllowResult.updated_input` is `ToolInput` in proto, `json.RawMessage` in JSON (reuse existing `toolInputToProto`/`toolInputFromProto`)
- `AllowResult.updated_permissions` is `repeated PermissionUpdate` in proto, `json.RawMessage` in JSON

### 3. Handler: Update `pkg/claudehook/handler/handler.go`

Change `HandlerError.Output` from `*sdk_typesv1.SyncHookJSONOutput` (proto) to `*models.SyncHookJSONOutput` (JSON model). Update `Handle()` to use `encoding/json.Marshal` instead of `protojson.Marshal`. This allows the handler to work with the richer JSON model that includes PermissionRequest fields.

Update existing code that constructs `HandlerError` values (e.g. `HookAudit` in `pkg/crabswarm/hook.go` returns `&handler.HandlerError{}` with nil Output — this still works since nil Output means exit 0).

For the `Handle()` method, update `GetDecision()` and `GetReason()` proto accessor calls to direct field access (`Output.Decision`, `Output.Reason`). The top-level `Decision` field is a `*string` for block decisions (used by UserPromptSubmit, PostToolUse, etc.), distinct from `HookSpecificOutput.Decision` which is a `*PermissionRequestDecision`.

Add a constructor for convenience:

```go
// NewPermissionRequestAllowError creates a HandlerError that outputs a PermissionRequest
// approval response (hookSpecificOutput with decision.behavior = "allow").
func NewPermissionRequestAllowError() *HandlerError {
    hookEventName := "PermissionRequest"
    return &HandlerError{
        Output: &models.SyncHookJSONOutput{
            HookSpecificOutput: &models.HookSpecificOutput{
                HookEventName: &hookEventName,
                Decision: &models.PermissionRequestDecision{
                    Behavior: string(PermissionRequestBehaviorAllow),
                },
            },
        },
    }
}
```

### 5. New package: `pkg/crabswarm/planreview/paths.go`

Pure functions:
- `PlanDirName(t, planName) string` — `{RFC3339}-{planName}`
- `IntermediateFileName(iteration, step, suffix) string` — `{NNN}_{SS}_{suffix}.md`
- `DerivePlanName(filePath) string` — base name minus `.md`
- `PathWithinDir(filePath, dirPath) (bool, error)` — canonicalizes both paths and checks containment:
  1. `filepath.Abs` both paths
  2. `filepath.EvalSymlinks` on `dirPath` (must exist)
  3. For `filePath`: `filepath.EvalSymlinks` on the longest existing ancestor, then append the remaining suffix. This handles PreToolUse where the target file may not exist yet
  4. Check `filePath == dirPath || strings.HasPrefix(filePath, dirPath + string(filepath.Separator))` — the separator suffix prevents `/tmp/plans2` matching `/tmp/plans`
- `CountIterations(intermediateDir) (int, error)` — glob `*_00_PLAN.md`

### 6. New: `pkg/crabswarm/planreview/transcript.go`

`ResolveLastPlanPath(transcriptPath, plansDir string) (string, error)` — parses the JSONL transcript file to find the last Write event whose `file_path` is under `plansDir`. Returns the resolved plan file path or empty string if none found.

### 7. New: `pkg/crabswarm/planreview/callback.go`

`RunCallback(ctx, CallbackConfig) (stdout, stderr string, err error)` using `exec.CommandContext`. Sets env vars:

| Env Var             | Value                                       |
|---------------------|---------------------------------------------|
| `PLAN_WORKING_FILE` | Abs path to `PLAN.md` in iteration dir      |
| `PLAN_SOURCE_FILE`  | Abs path to original plan in Claude plans dir |
| `PLAN_DIR`          | Abs path to iteration plan directory         |
| `PLAN_NAME`         | Plan name string                             |
| `ITERATION`         | Current iteration number                     |
| `INTERMEDIATE_DIR`  | Abs path to `_intermediate/`                 |

### 8. New: `pkg/crabswarm/planreview/hook.go`

**`HookPlanCallback(ctx, r, cfg HookCallbackConfig)`** (no gRPC):
1. Parse `PreToolUseHookInput`, check `ToolName == "ExitPlanMode"`
2. If `SessionID` empty, exit 0 (pass through)
3. `ResolveLastPlanPath(input.TranscriptPath, cfg.PlansDir)` — error or no result → exit 0 (nothing to review)
4. Read plan file content
5. Manage iteration dir (`${cfg.OutputDir}/{RFC3339}-{plan-name}/`), count iterations, write snapshot to `_intermediate/`
6. Run callback, save review output to intermediate dir
7. Exit 0 — user decides whether to continue iterating based on saved review

### 9. New: `pkg/crabswarm/hook_auto_approve.go`

**`HookAutoApprove(ctx, r, cfg AutoApproveConfig)`** (no gRPC):
1. Parse `PermissionRequestHookInput` from reader
2. If `cfg.ToolPatterns` is non-empty, check `ToolName` matches at least one regex — if not, pass through
3. If `cfg.UnderDirs` is non-empty, extract `file_path` from `ToolInput` (parse as generic map — works for Read, Write, and Edit which all have `file_path`), check `PathWithinDir(filePath, dir)` for each dir — if not under any, pass through
4. All conditions matched → return `handler.NewPermissionRequestAllowError()`
5. Otherwise pass through (exit 0, no output)

```go
type AutoApproveConfig struct {
    ToolPatterns []string // regex patterns to match tool_name
    UnderDirs    []string // directories — approve if file_path is under any
}
```

### 10. CLI: `cmd/crabswarm/commands/hook_plan_callback.go`

Cobra subcommand `hook plan-callback`. Flags:
- `--plans-dir` (default: `$CLAUDE_CONFIG_DIR/plans/` or `$HOME/.claude/plans/`) — where Claude writes plan files
- `--output-dir` (env fallback: `$CRABSWARM_PLAN_OUTPUT_DIR`, default: `doc/plans`). **Must be a relative path** — concatenated with `$CLAUDE_PROJECT_DIR` at runtime. An absolute path would produce an unwanted long joined path. Flag description and doc comments must state this.
- `--callback-cmd` (required, env fallback: `$CRABSWARM_PLAN_CALLBACK_CMD`)
- `--callback-arg` (repeatable string slice)
- `--callback-timeout` (default: 5m)
- `--max-iterations` (default: 0 = unlimited)
- `--approve-token` (default: "APPROVE")

No gRPC connection needed. Follows `hook_audit.go` pattern for stdin reading: `stdiopipe.Stdin`, assemble `HookCallbackConfig`, call `HookPlanCallback`.

### 11. CLI: `cmd/crabswarm/commands/hook_auto_approve.go`

Cobra subcommand `hook auto-approve`. Flags:
- `--tool` (repeatable string — regex patterns to match tool name)
- `--under` (repeatable string — directories to check file_path containment)

No gRPC connection needed. Follows `hook_audit.go` pattern for stdin reading: `stdiopipe.Stdin`, assemble `AutoApproveConfig`, call `HookAutoApprove`.

### 12. hooks.json: `plugin/crabswarm/hooks/hooks.json`

Add matchers for `ExitPlanMode` and `PermissionRequest`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/bin/crabswarm.sh hook audit",
            "timeout": 300,
            "statusMessage": "Waiting for permission..."
          },
          {
            "type": "command",
            "command": "tee >> hook_input.jsonl && echo \"\" >> hook_input.jsonl",
            "statusMessage": "copying input"
          }
        ]
      },
      {
        "matcher": "ExitPlanMode",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/bin/crabswarm.sh hook plan-callback --output-dir \"${CRABSWARM_PLAN_OUTPUT_DIR:-doc/plans}\" --callback-cmd \"${CRABSWARM_PLAN_CALLBACK_CMD}\"",
            "timeout": 600,
            "statusMessage": "Reviewing plan..."
          }
        ]
      }
    ],
    "PermissionRequest": [
      {
        "matcher": "^(Read|Write|Edit)$",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/bin/crabswarm.sh hook auto-approve --tool \"^(Read|Write|Edit)$\" --under \"${CLAUDE_PROJECT_DIR}/${CRABSWARM_PLAN_OUTPUT_DIR:-doc/plans}\"",
            "timeout": 10,
            "statusMessage": "Checking auto-approval..."
          }
        ]
      }
    ]
  }
}
```

## Directory Structure Created by plan-callback

```
{output-dir}/
  {RFC3339-datetime}-{plan-name}/
    PLAN.md                        # latest plan (always overwritten)
    _intermediate/
      001_00_PLAN.md               # iteration 1: plan snapshot
      001_01_REVIEW.md             # iteration 1: review
      002_00_PLAN.md               # iteration 2: updated plan
      002_01_REVIEW.md             # iteration 2: review
      ...
```

Iteration count derived by counting `*_00_PLAN.md` files — no state file needed.

## Error Handling

| Scenario                    | Behavior                                              |
|-----------------------------|-------------------------------------------------------|
| Malformed JSONL line        | Skip gracefully — log warning to stderr, continue parsing remaining lines |
| No plan file in transcript  | Exit 0 — nothing to review                            |
| Plan file missing/empty     | Exit 1 — log error to stderr, non-blocking            |
| Callback timeout            | Exit 1 — log timeout to stderr, non-blocking          |
| Callback non-zero exit      | Exit 1 — log callback stderr, non-blocking            |
| Empty review from callback  | Exit 0 — save empty review file, proceed              |
| session_id empty            | Exit 0, no output — pass through                      |
| Non-matching tool name      | Exit 0, no output — pass through                      |

## Existing Code to Reuse

- `pkg/claudehook/handler.HandlerError` — Output field changes from proto to JSON model (`handler.go`)
- `pkg/claudehook/handler.PermissionRequestBehaviorAllow` constant (`handler.go`)
- `pkg/claudesdk/models.PreToolUseHookInput`, `.PermissionRequestHookInput` (`hook_input.go`)
- `pkg/claudesdk/models.SyncHookJSONOutput`, `.HookSpecificOutput` (`hook_output.go`)
- `pkg/claudesdk/models.FileWriteInput`, `.ExitPlanModeInput`, `.FileReadInput` (`tool_input.go`)
- `cmd/internal/stdiopipe.Stdin` (`stdiopipe.go`)
- `planreview.PathWithinDir` — reused by `HookAutoApprove`
- Pattern from `pkg/crabswarm/hook.go` (`HookAudit`)
- Pattern from `cmd/crabswarm/commands/hook_audit.go`
- Existing bi-di conversion pattern in `pkg/claudesdk/models/convert.go`

## New/Modified Files

| File | Action |
|------|--------|
| `pkg/api/schema/proto/sdk_types/v1/hook.proto` | Modified (move PermissionResult/AllowResult/DenyResult here, add PermissionRequestHookSpecificOutput, update HookSpecificOutput oneof) |
| `pkg/api/schema/proto/sdk_types/v1/types.proto` | Modified (remove PermissionResult/AllowResult/DenyResult — moved to hook.proto) |
| `pkg/claudesdk/models/hook_output.go` | Modified (add PermissionRequest JSON models) |
| `pkg/claudesdk/models/convert.go` | Modified (add SyncHookJSONOutput bi-di conversion) |
| `pkg/claudehook/handler/handler.go` | Modified (switch Output to JSON model, add NewPermissionRequestAllowError) |
| `pkg/crabswarm/planreview/paths.go` | New |
| `pkg/crabswarm/planreview/transcript.go` | New |
| `pkg/crabswarm/planreview/callback.go` | New |
| `pkg/crabswarm/planreview/hook.go` | New |
| `pkg/crabswarm/hook_auto_approve.go` | New |
| `cmd/crabswarm/commands/hook_plan_callback.go` | New |
| `cmd/crabswarm/commands/hook_auto_approve.go` | New |
| `plugin/crabswarm/hooks/hooks.json` | Modified |

## Verification

### Build
1. `go build ./cmd/crabswarm/` — compiles
2. `go test ./pkg/...` — all tests pass
3. `crabswarm hook --help` — shows `plan-callback` and `auto-approve` subcommands

### Unit Tests Required

**`pkg/crabswarm/planreview/paths_test.go`:**
- `PathWithinDir`: exact match, nested path, sibling dir with shared prefix (e.g. `/tmp/plans2` vs `/tmp/plans`), symlinks, non-existent target file (PreToolUse scenario), `..` in remaining suffix
- `DerivePlanName`, `PlanDirName`, `IntermediateFileName`: basic correctness
- `CountIterations`: empty dir, one iteration, multiple iterations

**`pkg/crabswarm/planreview/transcript_test.go`:**
- No Write events → empty string
- Write outside plansDir → ignored
- Multiple Writes → last one wins
- Malformed JSONL lines → skip gracefully, no panic
- Empty transcript file → empty string

**`pkg/claudesdk/models/convert_test.go` (or `models_test.go`):**
- `SyncHookJSONOutput` ToProto/FromProto round-trip for PreToolUse fields
- PermissionRequest allow → proto AllowResult → JSON allow round-trip
- PermissionRequest deny → proto DenyResult → JSON deny round-trip
- PermissionRequest-specific fields survive round-trip

**`pkg/crabswarm/hook_auto_approve_test.go`:**
- Tool matches anchored regex `^(Read|Write|Edit)$`, file under dir → approve
- `NotebookEdit` does NOT match `^(Read|Write|Edit)$` → pass through
- Tool matches, file outside dir → pass through
- Tool does not match → pass through
- No ToolPatterns (match all), file under dir → approve
- No UnderDirs, tool matches → approve
- Tool with no `file_path` in input (e.g. Bash) + UnderDirs set → pass through

### Manual Integration Tests
4. Pipe ExitPlanMode hook JSON (with transcript_path pointing to a real transcript) to `crabswarm hook plan-callback`, verify iteration dir created with plan snapshot
5. Pipe PermissionRequest hook JSON (Read tool, file under target dir) to `crabswarm hook auto-approve --tool "^(Read|Write|Edit)$" --under /tmp/plans`, verify auto-approval JSON output
6. Pipe PermissionRequest hook JSON (Bash tool) to same command, verify pass-through (no output)
7. Pipe PermissionRequest hook JSON (Write tool, file outside target dir) to same command, verify pass-through
8. Pipe PermissionRequest hook JSON (NotebookEdit tool) to same command, verify pass-through (regex boundary check)
