---
tags: chat daemon test
---

# Review-noted chat daemon test gaps (2026-09-02)

Deferred, not defects, from the MCP-server run's review: nothing pins
the magnitude of the 10-minute staleness threshold (fixtures are
relative to the const); the daemon's `ChainStreamInterceptor` wiring in
`crabswarm/server/server.go` is never exercised by a test; the bounded
`GracefulStop` path has no test; no test asserts the timestamp
`ReportState` writes (a zero-time regression would make every busy
member instantly stale); the hook e2e exhaustiveness guard keys on
event name only, so re-adding a matcher-less catch-all `Notification`
group would fail nothing.

Follow-up: pin whichever of these bite first.
