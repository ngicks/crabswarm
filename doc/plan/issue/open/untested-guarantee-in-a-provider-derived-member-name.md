---
tags: chat join test
---

# Untested guarantee: `/` in a provider-derived member name (2026-08-31)

The comment in `crabswarm/chat/resolver/cmdman.go` (label values used
verbatim) leans on join-time rejection of `/` in names, but only
`req.Name` is test-driven through that path
(`crabswarm/chat/service_member_test.go`); no test sends a
`resolver.TeamInfo.Name` containing `/` through `Service.Join`. A future
"cmdman is trusted, skip validation" refactor could silently break the
documented guarantee.

Follow-up: add a service test driving a `/`-carrying provider-derived
name through `Service.Join` and asserting the `InvalidArgument`
rejection.
