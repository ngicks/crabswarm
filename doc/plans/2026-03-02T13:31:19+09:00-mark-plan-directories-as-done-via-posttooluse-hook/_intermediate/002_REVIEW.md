# Plan Review: Mark plan directories as done via PostToolUse hook

## Findings

1. **High** - `PlanDirStateStore.Update` bootstrap path is underspecified and currently incompatible with the planned PreToolUse call site
   - Reference: `002_PLAN.md:37-39`, `002_PLAN.md:55-56`
   - The plan says `Update(fn)` should "load if not cached" and then write through, and also says PreToolUse should replace `WritePlanDirState` with `NewPlanDirStateStore(planDir).Update(...)`.
   - For a newly created plan directory, `state.json` does not exist yet. With the described `Update` semantics, initial `Load()` fails before `fn` runs, so first-time state initialization in PreToolUse will fail.
   - Suggested fix: explicitly define first-write behavior for `Update` (for example, treat missing `state.json` as zero-value state and continue), and add a test that initializes state via `Update` when `state.json` is absent.

## Summary

The overall direction is sound, but the state-store API contract needs one explicit bootstrap rule; without that, the proposed call-site migration in PreToolUse can fail on new directories.
