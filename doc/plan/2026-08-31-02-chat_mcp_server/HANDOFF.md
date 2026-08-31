# Handoff — deferred / out-of-scope discoveries

- e2e coverage for the `crabswarm://chat/members` resource subscription
  through real processes (read + `notifications/resources/updated`);
  unit-covered in `crabswarm/chat/mcpserver/resources_test.go` only.
- Nudge-after-staleness e2e leg: the 10-minute threshold in
  `crabswarm/chat/notify` is an unexported const, so the delivered-nudge
  path after staleness cannot be exercised end to end without a test
  hook or option on `NewSendKeys`.
- protoc-gen-es version skew: local plugin v2.12.0 vs. committed files
  generated with v2.12.1 — every `go generate ./api` re-dirties four
  unrelated `*_pb.ts` files with a banner downgrade. Pin the plugin or
  upgrade the local toolchain.
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
