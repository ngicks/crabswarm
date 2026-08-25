---
description: "Instructions for the web/ frontend"
applyTo: "*.ts"
---

### TypeScript instructions

#### Package management

- Use `pnpm` (not npm/yarn) for the `web/` frontend.
- Pin exact dependency versions in package.json (no `^`/`~` ranges).

#### Build

- `pnpm build` (from `web/`) is the single build entry point: it runs the
  buf TS codegen (`api/buf.gen.ts.yaml`, TS-only template), vendors
  MathJax, `vite build`, then packs `web/dist` into `web/dist.tar.zst`
  (`go run ./scripts/packdist`, seekable zstd). No separate shell scripts.
- `web/dist.tar.zst` and `web/src/gen` are committed (go:embed needs them
  in the module zip); `web/dist` itself is git-ignored. Rebuild with
  `pnpm build` after changing `web/src` or the proto schema. CI verifies
  freshness via rebuild + `git diff --exit-code`.
