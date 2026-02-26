# Plan Review: Fix Iteration Count in Plan Callback Hook

## Findings

1. **High: Path identity matching can fail without canonicalization**
   - The plan stores and compares `SourcePlanPath` as a plain string. In current code, `ResolveLastPlanPath` returns whatever `file_path` appears in transcript, which may differ across runs (relative vs absolute paths, symlinked paths).
   - This can cause `FindExistingPlanDir` to miss the correct directory and create a new one, reintroducing iteration reset behavior.
   - Recommendation: canonicalize before write and compare (at least `filepath.Abs`, ideally same symlink-aware strategy used by `PathWithinDir` / `resolvePartial`).

2. **Medium: Duplicate-skip behavior on callback failure is underspecified**
   - Current hook writes `NNN_PLAN.md` before running callback, and writes `NNN_REVIEW.md` even on callback failure. With content-based dedup, the next identical call would skip callback entirely because last snapshot equals current content.
   - This may suppress intended retries after transient callback failures.
   - Recommendation: specify expected behavior. If retry is desired, dedup should consider last callback status (success/failure) instead of only plan content.

## Summary

The proposal is directionally correct and addresses the root cause (new directory per invocation), but it should tighten path identity rules and lookup completeness to avoid reintroducing the same issue under common path/layout conditions.
