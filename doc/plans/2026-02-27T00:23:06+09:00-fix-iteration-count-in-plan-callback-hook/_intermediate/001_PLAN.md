# Fix Iteration Count in Plan Callback Hook

## Context

The `HookPlanCallback` function creates a new timestamped directory on every `ExitPlanMode` invocation because it uses `time.Now()` to generate the directory name (`pkg/crabswarm/hook_plan_callback.go:77`). This means `CountIterations` always sees an empty `_intermediate/` directory and returns 0, so the iteration number is always 1. The iteration count never accumulates across multiple ExitPlanMode calls for the same plan.

Additionally, there's no deduplication — if the plan content hasn't changed between ExitPlanMode calls, it still creates a new snapshot and runs the callback unnecessarily.

## Changes

### 1. Add `FindExistingPlanDir` to `pkg/crabswarm/planreview/paths.go`

Add a function that searches `outputDir` for an existing directory matching the pattern `*-{planName}`:

```go
// FindExistingPlanDir searches outputDir for an existing plan directory
// whose name ends with "-{planName}". Returns the most recent match
// (lexicographic sort on RFC3339 prefix), or empty string if none found.
func FindExistingPlanDir(outputDir, planName string) (string, error)
```

- Glob for `*-{planName}` in `outputDir`
- Filter to directories only
- Sort matches (lexicographic = chronological due to RFC3339 prefix)
- Return the last (most recent) match, or "" if none

### 2. Add `LastSnapshotContent` to `pkg/crabswarm/planreview/paths.go`

Add a function to read the latest plan snapshot for deduplication:

```go
// LastSnapshotContent reads the content of the highest-numbered *_PLAN.md
// file in intermediateDir. Returns nil if no snapshots exist.
func LastSnapshotContent(intermediateDir string) ([]byte, error)
```

- Glob `*_PLAN.md`, sort, read the last one

### 3. Update `HookPlanCallback` in `pkg/crabswarm/hook_plan_callback.go`

Replace the `time.Now()` directory creation with:

1. Call `FindExistingPlanDir(cfg.OutputDir, planName)` to find existing dir
2. If found, reuse it. If not, create new with `PlanDirName(time.Now(), planName)`
3. After reading plan content, call `LastSnapshotContent(intermediateDir)` and compare with current plan content using `bytes.Equal`
4. If content is identical to last snapshot, skip processing (return pass-through `HandlerError`)

### 4. Add tests

**`pkg/crabswarm/planreview/paths_test.go`:**
- `TestFindExistingPlanDir`: empty dir, single match, multiple matches (picks latest), no match
- `TestLastSnapshotContent`: empty dir, single snapshot, multiple snapshots (picks latest)

**`pkg/crabswarm/hook_plan_callback_test.go`:**
- `TestHookPlanCallback_ReusesExistingDir`: call twice with same plan, verify iteration increments to 2 in the same directory
- `TestHookPlanCallback_SkipsDuplicateContent`: call twice with same content, verify second call is a no-op

## Files to Modify

| File | Action |
|------|--------|
| `pkg/crabswarm/planreview/paths.go` | Add `FindExistingPlanDir`, `LastSnapshotContent` |
| `pkg/crabswarm/planreview/paths_test.go` | Add tests for new functions |
| `pkg/crabswarm/hook_plan_callback.go` | Use `FindExistingPlanDir` + dedup check |
| `pkg/crabswarm/hook_plan_callback_test.go` | Add iteration reuse + dedup tests |

## Verification

1. `go test ./pkg/crabswarm/planreview/...` — new path tests pass
2. `go test ./pkg/crabswarm/...` — all hook callback tests pass (existing + new)
3. `go build ./cmd/crabswarm/` — compiles
