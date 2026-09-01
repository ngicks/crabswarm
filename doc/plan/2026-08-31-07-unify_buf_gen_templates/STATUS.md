# Status

State: implemented 2026-09-01 — all four steps landed on
`worktree-unify-buf-gen-templates`; both entry points reproduce the
committed generated trees byte-for-byte.

Origin: `doc/plan/issue/issue.md` entry "Unify the two buf generate
templates (2026-08-29)".

## Checklist

- [x] Step 1 — D1: "Go protoc plugins become `go.mod` `tool`
      dependencies" (`go get -tool` × 3, `go mod tidy`)
- [x] Step 2 — D1: "one template, toolchain-relative plugins" —
      rewrite `api/buf.gen.yaml` plugin entries, fold in the surviving
      ts.yaml comment; regenerate, diff clean
- [x] Step 3 — D1: "Delete `api/buf.gen.ts.yaml`" + point
      `web/package.json` `gen` at the default template; `pnpm -C web run
      gen` clean
- [x] Step 4 — IDEA.md "prerequisite belongs in a comment at the point
      of use": update `api/generate.go` comment (pnpm install
      prerequisite); `go generate ./api` clean
- [x] Verification — PLAN.md "Testing and verification": both entry
      points reproduce committed output; `pnpm -C web run build` and
      `dist.tar.zst` diff clean

## Next action

None — review the branch and merge.
