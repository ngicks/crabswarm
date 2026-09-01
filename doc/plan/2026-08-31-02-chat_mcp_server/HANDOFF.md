# Handoff — deferred / out-of-scope discoveries

- e2e coverage for the `crabswarm://chat/members` resource subscription
  through real processes (read + `notifications/resources/updated`);
  unit-covered in `crabswarm/chat/mcpserver/resources_test.go` only.
- Nudge-after-staleness e2e leg: the 10-minute threshold in
  `crabswarm/chat/notify` is an unexported const, so the delivered-nudge
  path after staleness cannot be exercised end to end without a test
  hook or option on `NewSendKeys`.
- protoc-gen-es version skew: `web/package.json` pins
  `@bufbuild/protoc-gen-es` at 2.12.1, but the plugin `go generate ./api`
  actually runs is v2.12.0, so regeneration rewrites banners *downwards*.
  The committed `chat_service_pb.ts` now carries the v2.12.0 banner while
  its four siblings (`audit_service_pb.ts`, `preview_service_pb.ts`,
  `codex_pb.ts`, `types_pb.ts`) still carry v2.12.1, so every regen
  re-dirties those four and they have to be reverted by hand. Install the
  pinned version locally, or move the pin to what is installed.
- apm packed-bundle install (apm 0.28.0) lands `hooks.json` at
  `.github/hooks.json` and skills under `.agents/skills/` only, with no
  `.claude/settings.json`; the README claim that a bundle carries the
  same wiring as an install from source looks stale (pre-existing).
- Codex runtime unverified: apm writes `[mcp_servers.crabswarm-chat]`
  into `.codex/config.toml`, but nobody has confirmed codex actually
  starts the bridge.
- Existing dev chat DBs lack the new NOT NULL `members.state_reported_at`
  column (no migration by design); they must be deleted/recreated —
  worth a release-note line.
- godoc nit: the `[nudgeable]` doc link in the exported `SendKeys`
  comment points at an unexported func and renders as plain text.
- Review-noted test gaps (deferred, not defects): nothing pins the
  magnitude of the 10-minute staleness threshold (fixtures are relative
  to the const); the daemon's `ChainStreamInterceptor` wiring in
  `crabswarm/server/server.go` is never exercised by a test (deleting
  it keeps the suite green); the bounded `GracefulStop` path has no
  test; no test asserts the timestamp `ReportState` writes (a zero-time
  regression would make every busy member instantly stale); the hook
  e2e exhaustiveness guard keys on event name only, so re-adding a
  matcher-less catch-all `Notification` group would fail nothing.
- `apm-package` SKILL.md still teaches only the CLI verbs — it should
  mention the four MCP tools and the `crabswarm://chat/members`
  resource now that the package ships the bridge.
- README's "every other notification type" list enumerates three
  values; the official hooks docs list ~11 (e.g. `agent_needs_input`,
  `agent_completed`).
