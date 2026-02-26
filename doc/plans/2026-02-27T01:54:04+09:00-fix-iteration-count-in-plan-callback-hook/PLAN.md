# Fix Iteration Count in Plan Callback Hook

## Context

The `HookPlanCallback` function creates a new timestamped directory on every `ExitPlanMode` invocation because it uses `time.Now()` to generate the directory name (`pkg/crabswarm/hook_plan_callback.go:77`). This means `CountIterations` always sees an empty `_intermediate/` directory and returns 0, so the iteration number is always 1. The iteration count never accumulates across multiple ExitPlanMode calls for the same plan.

Additionally, there's no deduplication — if the plan content hasn't changed between ExitPlanMode calls, it still creates a new snapshot and runs the callback unnecessarily.

## Changes

### 1. Add `state.json` support to `pkg/crabswarm/planreview/paths.go`

Add a `PlanDirState` struct and functions for reading/writing `state.json`:

```go
type PlanDirState struct {
    SourcePlanPath string `json:"source_plan_path"` // Canonicalized (absolute, symlink-resolved)
}

func WritePlanDirState(planDir string, state PlanDirState) error
func ReadPlanDirState(planDir string) (PlanDirState, error)
```

### 2. Add `FindExistingPlanDir` to `pkg/crabswarm/planreview/paths.go`

```go
// FindExistingPlanDir searches outputDir for an existing plan directory
// whose state.json records sourcePlanPath as the original plan source.
// sourcePlanPath must already be canonicalized.
// Searches at most 5 most recent directories (sorted reverse-lexicographic
// by RFC3339 prefix). Returns the matching directory path, or "" if none found.
func FindExistingPlanDir(outputDir, sourcePlanPath string) (string, error)
```

### 3. Add `LastSnapshotContent` and `LastReviewExists` to `pkg/crabswarm/planreview/paths.go`

```go
// LastSnapshotContent reads the content of the highest-numbered *_PLAN.md
// file in intermediateDir. Returns nil if no snapshots exist.
func LastSnapshotContent(intermediateDir string) ([]byte, error)

// LastReviewExists checks if a *_REVIEW.md exists for the highest-numbered iteration.
// Returns false if no snapshots exist or no corresponding review exists.
func LastReviewExists(intermediateDir string) (bool, error)
```

### 4. Update `HookPlanCallback` in `pkg/crabswarm/hook_plan_callback.go`

Replace the `time.Now()` directory creation logic:

1. After resolving `planPath`, **canonicalize** it using `filepath.Abs` + `resolvePartial` (same strategy as `PathWithinDir`) before any comparisons or storage.
2. Call `FindExistingPlanDir(cfg.OutputDir, canonicalPlanPath)` to find existing dir by source plan path identity.
3. If found, reuse it. If not, create new with `PlanDirName(time.Now(), planName)` and write `state.json` with canonicalized source path.
4. After reading plan content, apply **dedup check**:
   - Call `LastSnapshotContent(intermediateDir)` and compare with current content using `bytes.Equal`
   - If content is identical, also check `LastReviewExists(intermediateDir)`
   - **Skip only if** content matches AND a review file exists for the last iteration (meaning the callback completed successfully for this content)
   - If content matches but no review exists (callback failed previously), **do not skip** — re-run the callback as a retry

### 5. Export `resolvePartial` as `ResolvePartial` in `pkg/crabswarm/planreview/paths.go`

Currently unexported. Export it so `HookPlanCallback` can use it for path canonicalization. (It's in the same package `planreview`, so technically accessible, but making it exported clarifies the API surface.)

Actually — `HookPlanCallback` is in package `crabswarm`, not `planreview`. So it needs to call `planreview.ResolvePartial`. **Export** the function by renaming `resolvePartial` → `ResolvePartial`.

Alternatively, add a convenience `CanonicalizePath(path string) (string, error)` that does `filepath.Abs` + `ResolvePartial`:

```go
// CanonicalizePath returns an absolute, symlink-resolved path.
// Uses resolvePartial to handle paths where the target file may not exist.
func CanonicalizePath(path string) (string, error)
```

### 6. Add tests

**`pkg/crabswarm/planreview/paths_test.go`:**
- `TestPlanDirState_WriteRead`: round-trip
- `TestFindExistingPlanDir`: empty dir → "", match by canonical source path, no match → "", multiple dirs only one matches, stops after 5 dirs
- `TestLastSnapshotContent`: empty → nil, single, multiple picks highest
- `TestLastReviewExists`: no snapshots → false, snapshot with review → true, snapshot without review → false
- `TestCanonicalizePath`: absolute path, relative path, symlinked path

**`pkg/crabswarm/hook_plan_callback_test.go`:**
- `TestHookPlanCallback_ReusesExistingDir`: call twice with updated plan content, verify:
  - Only 1 plan directory in outputDir
  - `_intermediate/` has `001_PLAN.md` and `002_PLAN.md`
  - `state.json` exists with correct canonicalized source path
- `TestHookPlanCallback_SkipsDuplicateContent`: call twice with identical content, verify:
  - Only `001_PLAN.md` (no `002_PLAN.md`)
  - Only `001_REVIEW.md` (no `002_REVIEW.md`)
  - Callback side-effect file written only once
- `TestHookPlanCallback_RetriesOnCallbackFailure`: first call with failing callback, second call with same content, verify:
  - `001_PLAN.md` and `002_PLAN.md` both exist (retry happened because no review from first call's success)
  - OR more precisely: since callback failure writes stderr to review, check that `001_REVIEW.md` exists from failed attempt, and the dedup still triggers since review exists. **Revisit**: current code writes review file even on failure. So dedup would skip. Need to decide: either don't write review on failure, or track success in state.json.
  - **Decision**: Don't write `NNN_REVIEW.md` on callback failure. Only write it on success. This way, `LastReviewExists` correctly indicates whether the last callback succeeded, enabling retry on failure.

## Files to Modify

| File | Action |
|------|--------|
| `pkg/crabswarm/planreview/paths.go` | Add `PlanDirState`, `WritePlanDirState`, `ReadPlanDirState`, `FindExistingPlanDir`, `LastSnapshotContent`, `LastReviewExists`, `CanonicalizePath`. Export `resolvePartial` → `ResolvePartial`. |
| `pkg/crabswarm/planreview/paths_test.go` | Add tests for new functions |
| `pkg/crabswarm/hook_plan_callback.go` | Use `FindExistingPlanDir` + canonicalized paths + dedup check. Don't write review on callback failure. |
| `pkg/crabswarm/hook_plan_callback_test.go` | Add iteration reuse + dedup + retry tests |

## Verification

1. `go test ./pkg/crabswarm/planreview/...` — new path tests pass
2. `go test ./pkg/crabswarm/...` — all hook callback tests pass (existing + new)
3. `go build ./cmd/crabswarm/` — compiles
