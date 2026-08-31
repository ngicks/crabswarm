# Status

State: not started — plan drafted 2026-08-31 with automatic decisions
(idea gate not user-confirmed; see IDEA.md `Gate:` line).

Origin: `doc/plan/issue/issue.md` entry "Unify the two buf generate
templates (2026-08-29)".

## Checklist

- [ ] Step 1 — D1: "Go protoc plugins become `go.mod` `tool`
      dependencies" (`go get -tool` × 3, `go mod tidy`)
- [ ] Step 2 — D1: "one template, toolchain-relative plugins" —
      rewrite `api/buf.gen.yaml` plugin entries, fold in the surviving
      ts.yaml comment; regenerate, diff clean
- [ ] Step 3 — D1: "Delete `api/buf.gen.ts.yaml`" + point
      `web/package.json` `gen` at the default template; `pnpm -C web run
      gen` clean
- [ ] Step 4 — IDEA.md "prerequisite belongs in a comment at the point
      of use": update `api/generate.go` comment (pnpm install
      prerequisite); `go generate ./api` clean
- [ ] Verification — PLAN.md "Testing and verification": both entry
      points reproduce committed output; `pnpm -C web run build` and
      `dist.tar.zst` diff clean

## Next action

User reviews the automatic decision D1 and the IDEA.md gate, then step 1.
