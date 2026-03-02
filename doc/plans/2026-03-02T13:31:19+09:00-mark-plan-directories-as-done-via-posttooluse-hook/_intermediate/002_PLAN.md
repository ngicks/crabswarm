# Mark plan directories as done via PostToolUse hook

## Context

Claude Code reuses plan file paths (e.g., `~/.claude/plans/spicy-discovering-ocean.md`) for different plans across conversations. The current `FindExistingPlanDir` matches on `SourcePlanPath` in `state.json`, so when a path is reused for a completely different plan, iterations incorrectly continue in the old plan directory instead of creating a new one.

**Fix**: Add a `PostToolUse` hook on `ExitPlanMode`. PostToolUse only fires after ExitPlanMode actually executes (i.e., the user approved the plan). It marks `state.json` with `done: true`. `FindExistingPlanDir` skips done directories, so the next use of the same plan file path creates a fresh directory.

## Changes

### 1. Split state into model + store — `pkg/crabswarm/planreview/paths.go`

**Model type** (pure data, unchanged name):
```go
type PlanDirState struct {
    SourcePlanPath string `json:"source_plan_path"`
    Done           bool   `json:"done,omitempty"`
}
```

**Store type** (loader/saver with write-through cache):
```go
type PlanDirStateStore struct {
    path   string        // path to state.json
    cached *PlanDirState // nil until first load
}

func NewPlanDirStateStore(planDir string) *PlanDirStateStore

// Load reads state.json from disk and caches it. Returns cached value on subsequent calls.
func (s *PlanDirStateStore) Load() (PlanDirState, error)

// Update passes the cached state to fn, writes the returned state to disk, and updates the cache.
func (s *PlanDirStateStore) Update(fn func(PlanDirState) PlanDirState) error
```

- `Load()`: if `cached != nil`, return `*cached`. Otherwise read from disk, cache, return.
- `Update(fn)`: load if not cached, call `fn(cached)`, marshal+write to disk, update cache.
- Remove the standalone `WritePlanDirState` / `ReadPlanDirState` functions (replaced by store methods).
- `omitempty` on `Done` ensures backward compat: existing state.json without the field → `Done: false`.

### 2. Add `includeDone` param to `FindExistingPlanDir` — `pkg/crabswarm/planreview/paths.go`

```go
func FindExistingPlanDir(outputDir, sourcePlanPath string, includeDone bool) (string, error)
```

In the search loop, use `NewPlanDirStateStore(dirPath).Load()` instead of the old `ReadPlanDirState`. Skip entries where `state.Done && !includeDone`.

- PreToolUse calls with `includeDone=false` (skip done dirs → create new dir for reused paths)
- PostToolUse calls with `includeDone=true` (find the dir to mark it done)

### 3. Update PreToolUse caller — `pkg/crabswarm/hook_plan_callback.go`

- Pass `false` as the new `includeDone` parameter (line 85)
- Use `NewPlanDirStateStore(planDir).Update(...)` instead of `WritePlanDirState` for writing state (line 101-105)

### 4. New `HookPlanDone` function — `pkg/crabswarm/hook_plan_done.go` (new file)

Handles PostToolUse ExitPlanMode:
1. Parse `PostToolUseHookInput` from stdin
2. Guard: only process `ExitPlanMode`, bail on empty `SessionID`
3. Resolve last plan path from transcript (reuse `planreview.ResolveLastPlanPath`)
4. Canonicalize path
5. `FindExistingPlanDir(outputDir, canonicalPath, true)` — find the active dir
6. `NewPlanDirStateStore(planDir).Update(func(s PlanDirState) PlanDirState { s.Done = true; return s })`
7. Return `&handler.HandlerError{}` (exit 0)

Config struct: `HookDoneConfig{PlansDir, OutputDir}` (subset of `HookCallbackConfig`).

### 5. New CLI subcommand — `cmd/crabswarm/commands/hook_plan_done.go` (new file)

Follow the pattern from `hook_plan_callback.go`. Register `hookPlanDoneCmd` under `hookCmd` in `init()`. Same flag handling for `--plans-dir` and `--output-dir`. Reuse existing `resolvePlansDir()`.

### 6. Add PostToolUse entry — `plugin/crabswarm/hooks/hooks.json`

```json
"PostToolUse": [
  {
    "matcher": "ExitPlanMode",
    "hooks": [
      {
        "type": "command",
        "command": "${CLAUDE_PLUGIN_ROOT}/bin/crabswarm.sh hook plan-done --output-dir \"${CRABSWARM_PLAN_OUTPUT_DIR:-doc/plans}\"",
        "timeout": 10,
        "statusMessage": "Marking plan as done..."
      }
    ]
  }
]
```

### 7. Update tests

**`pkg/crabswarm/planreview/paths_test.go`**:
- Update all `FindExistingPlanDir` calls to pass `false` (preserves existing behavior)
- Replace `WritePlanDirState`/`ReadPlanDirState` usage with `NewPlanDirStateStore` + `Update`/`Load`
- Add `TestPlanDirStateStore_LoadCache` — second `Load()` returns cached value without re-reading disk
- Add `TestPlanDirStateStore_UpdateWritesThrough` — `Update` writes to disk and updates cache
- Add `TestFindExistingPlanDir_SkipsDone`: dir with `Done:true` is skipped by `includeDone=false`, found by `includeDone=true`
- Add `TestPlanDirState_DoneBackwardCompat`: state.json without `done` field → `Done==false`

**`pkg/crabswarm/hook_plan_done_test.go`** (new file):
- `TestHookPlanDone_NonExitPlanMode` — passthrough
- `TestHookPlanDone_EmptySessionID` — passthrough
- `TestHookPlanDone_MarksDone` — set up plan dir with state.json, call HookPlanDone, verify `done:true`
- `TestHookPlanDone_NoPlanDir` — no error when no dir exists
- `TestHookPlanDone_Idempotent` — calling twice doesn't error

**`pkg/crabswarm/hook_plan_callback_test.go`**:
- Add `TestHookPlanCallback_SkipsDonePlanDir` — create dir with `Done:true`, call HookPlanCallback with same source path, verify new directory is created

## Verification

1. `go build ./...` — compiles
2. `go test ./pkg/crabswarm/... ./pkg/crabswarm/planreview/...` — all tests pass
3. Manual end-to-end: enter plan mode, write plan, exit plan mode → verify state.json has `done:false` after PreToolUse, then `done:true` after PostToolUse
