# Review: 001_PLAN.md

## Findings

1. **High** — Plan identity is ambiguous and can merge unrelated plans.
   - Reference: `001_PLAN.md:13-25`, `001_PLAN.md:43-44`
   - `FindExistingPlanDir(outputDir, planName)` keyed only by `planName` can reuse a directory from a different source plan that happens to share the same heading-derived name. That would corrupt iteration history across independent plans.
   - Fix: add new file, state.json per plan dir. In it, record original plan path. Copy source is always tracked.

2. **High** — Suffix glob `*-{planName}` can match the wrong directory.
   - Reference: `001_PLAN.md:13`, `001_PLAN.md:22`
   - Pattern matching by suffix allows false matches when one plan name is a suffix of another (example: searching for `bug` also matches `fix-bug`). The “latest” chosen dir may be incorrect.
   - Fix: read plan dir from latest to oldest. Read state.json in each plan dir. So you can find a match. You can cut off searching at, like, 5 plan dirs so you don't block unnecessary long time. There shouldn't be too many simultaneous planning client.

3. **Medium** — Dedup test scope is underspecified for callback side effects.
   - Reference: `001_PLAN.md:56`
   - “no-op” should explicitly assert that no new `*_PLAN.md` and no new `*_REVIEW.md` are written, and callback is not executed again. Without these assertions, regressions can slip through.
   - Suggested fix: in `TestHookPlanCallback_SkipsDuplicateContent`, assert file counts remain unchanged and callback output/side effect appears only once.

## Summary

The core direction is correct, but directory selection needs a stronger identity strategy and stricter matching/parsing to avoid cross-plan contamination and false matches.
