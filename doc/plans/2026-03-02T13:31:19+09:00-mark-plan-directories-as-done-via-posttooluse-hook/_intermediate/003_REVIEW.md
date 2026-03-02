# Plan Review: Mark plan directories as done via PostToolUse hook

## Findings

1. **High** - `plan-done` directory selection can mark the wrong directory when multiple plan dirs share the same `SourcePlanPath`
   - Reference: `003_PLAN.md:51`, `003_PLAN.md:65`, `003_PLAN.md:102`, `003_PLAN.md:108-110`
   - The plan sets PostToolUse lookup to `FindExistingPlanDir(..., includeDone=true)` and defines matching as path-based with done filtering disabled in that mode.
   - If multiple directories exist for the same source path (for example due to earlier hook failure, retries, or manual recovery), this lookup is not constrained to the currently active `done:false` directory and can return an already-`done:true` directory. In that case PostToolUse becomes a no-op on the wrong directory and leaves the active directory unclosed.
   - Required fix: define deterministic selection for PostToolUse that prioritizes the active directory (for example: prefer `Done==false` match first, then fallback), or pass/persist a stable directory identifier from PreToolUse and mark that exact directory.

## Additional Gap

1. Missing test for ambiguous multi-directory selection in PostToolUse
   - Reference: `003_PLAN.md:105-110`
   - Suggestion: add a test with two directories sharing one source path (`old done:true`, `active done:false`) and verify `HookPlanDone` marks the `done:false` one.

## Verdict

Changes requested before implementation.
