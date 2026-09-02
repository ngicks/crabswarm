---
tags: chat store release
---

# Release note: old dev chat DBs need recreating (2026-09-02)

Existing dev chat databases lack the `members.state_reported_at` NOT
NULL column (no migration by design; the repo has no deployment
back-compat obligation). They must be deleted/recreated.

Follow-up: a release-note line when anything ships.
