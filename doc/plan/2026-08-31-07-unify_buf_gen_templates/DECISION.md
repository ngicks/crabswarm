# Decisions

## D1 — Go toolchain owns generation; one template, toolchain-relative plugins (automatic decision)

Choice: keep only `api/buf.gen.yaml`; both `go generate ./api` and
`web/package.json`'s `gen` script run it unmodified. The Go protoc
plugins become `go.mod` `tool` dependencies invoked as
`local: ["go", "run", "<pkg>"]`; `protoc-gen-es` is invoked by explicit
path `../web/node_modules/.bin/protoc-gen-es`. Delete
`api/buf.gen.ts.yaml`.

Rationale: the TS-only template existed so the web build would need
neither Go protoc plugins nor a Go toolchain — but `pnpm build` already
runs `go run ./scripts/packdist`, so the Go toolchain is a web-build
prerequisite regardless; only pre-installed plugin binaries were a real
cost, and buf v2's argv-array `local:` form plus `tool` directives
(already repo practice via sqlc) removes it. Pinning `protoc-gen-es` by
node_modules path likewise removes the PATH reliance in the reverse
direction, so a single template runs identically from either entry
point.

Rejected:

- Generate TS solely from `go generate ./api` and drop the web `gen`
  build step — `pnpm build` would silently use stale committed TS after
  a proto edit; the current self-refreshing build is worth keeping.
- Keep both templates and add a CI check that their shared blocks match —
  preserves the duplication the issue backlog calls meaningless
  over-engineering and adds machinery instead of removing it.

Tagged automatic per user directive (2026-08-31): skip questions, use
the recommendation.
