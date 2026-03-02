# Review Result: Changes Requested

## Findings

1. High: Backward compatibility strategy preserves the current mis-association behavior for existing plan directories.
- Reference: `001_PLAN.md:20`, `001_PLAN.md:30-31`.
- Risk: Existing `state.json` files without `done` deserialize to `Done=false`, so previously completed directories are still treated as active. The first reused-path event after rollout can still append to an old directory.
- Required fix: Define an explicit migration/transition rule for legacy states (for example: one-time migration marker, or a compatibility policy that can distinguish legacy completed directories from active ones).

2. Medium: `plan-done` lookup is not guaranteed to target the exact directory created during PreToolUse.
- Reference: `001_PLAN.md:44-46`.
- Risk: `FindExistingPlanDir(..., includeDone=true)` is path-based and heuristic (recent-dir scan in current implementation). If multiple directories share the same source path, PostToolUse can mark a different directory (or none), leaving the active one as `done:false`.
- Required fix: Add a stable identifier from PreToolUse to PostToolUse for exact matching (for example: persist the selected plan dir in hook state keyed by session/tool-use, then mark that exact dir done).

## Additional Gap

1. Tests do not cover the full cross-conversation lifecycle that motivates this change.
- Reference: `001_PLAN.md:82`, `001_PLAN.md:87`.
- Suggestion: Add an integration-style test sequence: PreToolUse creates dir A, PostToolUse marks A done, then next PreToolUse with same source path creates dir B (not A).

## Verdict

Direction is good, but the above correctness gaps should be resolved before implementation.
