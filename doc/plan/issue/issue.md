# Issue backlog

Durable follow-ups that outlive their originating plan directories.
Append only; never rewrite or reorder existing entries.

## Unify the two buf generate templates (2026-08-29)

`api/buf.gen.yaml` (full: Go + TS plugins, run by `go generate` in
`api/`) and `api/buf.gen.ts.yaml` (TS-only subset, run by the
`web/package.json` "gen" script) duplicate the `managed` block and the
`protoc-gen-es` plugin block, with a comment obliging humans to keep
them in sync by hand. The duplication is meaningless over-engineering —
both sides must be in sync anyway, so a single file should cover both.

Follow-up: collapse to one template covering both toolchains — e.g.
keep only `buf.gen.yaml` and have `web`'s "gen" script invoke it (Go
plugins must then be present for the web build), or generate the TS
solely from the Go-side `go generate` and drop the pnpm-side gen step.
Decide which toolchain owns generation, then delete `buf.gen.ts.yaml`.
