# Plan Review: Fix Iteration Count in Plan Callback Hook

## Findings

### High

1. **Matching only the latest 5 plan directories can reintroduce iteration reset**  
   - Reference: `PLAN.md:30-31`, `PLAN.md:78`  
   - `FindExistingPlanDir` is capped to 5 recent directories. If a target plan directory is older than the newest 5 entries in `outputDir`, lookup returns no match, a new directory is created, and iteration restarts from `001`. This directly conflicts with the core goal of stable per-plan iteration accumulation.
   - Recommended change: search all candidate plan directories (or maintain an index map from canonical source path to plan dir) instead of truncating to 5.

2. **Retry behavior section is internally inconsistent and leaves implementation ambiguity**  
   - Reference: `PLAN.md:92-95`  
   - The test description contains mutually exclusive expectations (`retry happened` vs `dedup still triggers`) before a final “Decision” line. This is likely to cause divergent implementation/tests depending on which bullet is followed.
   - Recommended change: remove the obsolete branch entirely and rewrite this section as a single normative requirement.

### Medium

1. **Using `*_REVIEW.md` existence as success state introduces behavior regression risk**  
   - Reference: `PLAN.md:57-58`, `PLAN.md:95`, `PLAN.md:103`  
   - The plan changes behavior from “always persist review/stderr artifact” to “no review file on callback failure” to drive retry logic. This may reduce debuggability and breaks current expectations where failure stderr is preserved in `001_REVIEW.md`.
   - Recommended change: keep failure artifacts (e.g., `NNN_REVIEW.stderr.md` or `state.json` success flag) and decouple retry gating from presence of review output.

2. **Dedup semantics when callback is disabled are unspecified**  
   - Reference: `PLAN.md:56-58`  
   - Skip condition depends on `LastReviewExists`. When `CallbackCmd` is empty, no review is produced, so identical plans will never dedup and will continue creating snapshots.
   - Recommended change: define explicit policy for `CallbackCmd == ""` (either always snapshot, or dedup by content only).

## Overall

The direction is good and addresses the root cause (directory identity based on source plan path), but the lookup cap and retry/success signaling need tightening before implementation to avoid regressions.
