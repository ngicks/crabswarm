# Fix Iteration Count in Plan Callback Hook

## Context

The `HookPlanCallback` function creates a new timestamped directory on every `ExitPlanMode` invocation because it uses `time.Now()` to generate the directory name (`pkg/crabswarm/hook_plan_callback.go:77`). This means `CountIterations` always sees an empty `_intermediate/` directory and returns 0, so the iteration number is always 1. The iteration count never accumulates across multiple ExitPlanMode calls for the same plan.

Additionally, there's no deduplication — if the plan content hasn't changed between ExitPlanMode calls, it still creates a new snapshot and runs the callback unnecessarily.

## Changes

### 1. Add `state.json` support to `pkg/crabswarm/planreview/paths.go`

Add a `PlanDirState` struct and functions for reading/writing `state.json`:

```go
// PlanDirState is persisted as state.json in each plan directory.
// It records the original source plan path for identity matching.
type PlanDirState struct {
    SourcePlanPath string `json:"source_plan_path"`
}

// WritePlanDirState writes state.json into planDir.
func WritePlanDirState(planDir string, state PlanDirState) error

// ReadPlanDirState reads state.json from planDir.
func ReadPlanDirState(planDir string) (PlanDirState, error)
```

### 2. Add `FindExistingPlanDir` to `pkg/crabswarm/planreview/paths.go`

Search `outputDir` for an existing plan directory whose `state.json` records the same source plan path:

```go
// FindExistingPlanDir searches outputDir for an existing plan directory
// whose state.json records sourcePlanPath as the original plan source.
// Searches at most 5 most recent directories (sorted reverse-lexicographic
// by RFC3339 prefix) to avoid blocking on large output dirs.
// Returns the matching directory path, or "" if none found.
func FindExistingPlanDir(outputDir, sourcePlanPath string) (string, error)
```

- List directories in `outputDir`, sort reverse-lexicographic (newest first)
- Check at most 5 directories: read `state.json`, compare `SourcePlanPath`
- Return the first match, or "" if none found

This avoids suffix-based glob matching which can produce false positives (e.g. `bug` matching `fix-bug`).

### 3. Add `LastSnapshotContent` to `pkg/crabswarm/planreview/paths.go`

```go
// LastSnapshotContent reads the content of the highest-numbered *_PLAN.md
// file in intermediateDir. Returns nil if no snapshots exist.
func LastSnapshotContent(intermediateDir string) ([]byte, error)
```

- Glob `*_PLAN.md`, sort, read the last one

### 4. Update `HookPlanCallback` in `pkg/crabswarm/hook_plan_callback.go`

Replace the `time.Now()` directory creation logic:

1. After deriving `planName` and resolving `planPath`, call `FindExistingPlanDir(cfg.OutputDir, planPath)` to find existing dir by source plan path identity
2. If found, reuse it. If not, create new with `PlanDirName(time.Now(), planName)` and write `state.json` with `SourcePlanPath: planPath`
3. After reading plan content, call `LastSnapshotContent(intermediateDir)` and compare with current plan content using `bytes.Equal`
4. If content is identical to last snapshot, skip processing (return pass-through `HandlerError`)

### 5. Add tests

**`pkg/crabswarm/planreview/paths_test.go`:**
- `TestPlanDirState_WriteRead`: round-trip write/read
- `TestFindExistingPlanDir`: empty dir → "", single match by source path, multiple dirs only matching one matches, no match returns "", stops after 5 dirs
- `TestLastSnapshotContent`: empty dir → nil, single snapshot, multiple snapshots picks highest-numbered

**`pkg/crabswarm/hook_plan_callback_test.go`:**
- `TestHookPlanCallback_ReusesExistingDir`: call twice with same plan, verify:
  - Only 1 plan directory exists in outputDir
  - `_intermediate/` contains `001_PLAN.md`, `002_PLAN.md` (iteration incremented)
  - `state.json` exists with correct source path
- `TestHookPlanCallback_SkipsDuplicateContent`: call twice with identical plan content, verify:
  - Only `001_PLAN.md` exists (no `002_PLAN.md` created)
  - Only `001_REVIEW.md` exists (no `002_REVIEW.md` created)
  - Callback was not executed a second time (use a side-effect file written by callback to verify)

## Files to Modify

| File | Action |
|------|--------|
| `pkg/crabswarm/planreview/paths.go` | Add `PlanDirState`, `WritePlanDirState`, `ReadPlanDirState`, `FindExistingPlanDir`, `LastSnapshotContent` |
| `pkg/crabswarm/planreview/paths_test.go` | Add tests for new functions |
| `pkg/crabswarm/hook_plan_callback.go` | Use `FindExistingPlanDir` + state.json + dedup check |
| `pkg/crabswarm/hook_plan_callback_test.go` | Add iteration reuse + dedup tests with explicit assertions |

## Verification

1. `go test ./pkg/crabswarm/planreview/...` — new path tests pass
2. `go test ./pkg/crabswarm/...` — all hook callback tests pass (existing + new)
3. `go build ./cmd/crabswarm/` — compiles
