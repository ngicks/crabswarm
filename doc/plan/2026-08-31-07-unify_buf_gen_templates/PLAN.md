# Unify the two buf generate templates

Collapse `api/buf.gen.yaml` and `api/buf.gen.ts.yaml` into the single
`api/buf.gen.yaml`, invoked identically by `go generate ./api` and the web
`gen` script.

## Goal / success criteria

- `api/buf.gen.ts.yaml` is deleted; `api/buf.gen.yaml` is the only
  template.
- `go generate ./api` and `pnpm -C web run gen` both run the full template
  and leave `git status` clean against the committed `api/gen/proto/go`
  and `web/src/gen`.
- Neither entry point requires protoc plugin binaries pre-installed on
  PATH: Go plugins run through the Go toolchain (`go run` + `tool`
  directives), `protoc-gen-es` resolves through `web/node_modules`.

## Non-goals

- Changing what is generated (plugins, options, output layout).
- Touching the proto schema or any consumer of the generated code.
- CI workflow changes beyond what keeps existing checks passing.

## Context

- `api/buf.gen.yaml` — full template: `protoc-gen-go`,
  `protoc-gen-go-grpc`, `protoc-gen-connect-go` (all `local:` bare names,
  i.e. must be on PATH) into `gen/proto/go`, plus `protoc-gen-es` into
  `../web/src/gen`. Run by `//go:generate buf generate` in
  `api/generate.go`.
- `api/buf.gen.ts.yaml` — TS-only duplicate of the `managed` block and
  the `protoc-gen-es` stanza, run by `web/package.json`'s
  `"gen": "cd ../api && buf generate --template buf.gen.ts.yaml"`. Its
  header comment obliges humans to keep the `managed` blocks in sync so
  the embedded FileDescriptor stays byte-identical.
- Its stated reason to exist — the frontend build "needs neither the Go
  protoc plugins nor a Go toolchain" — is already half false: `pnpm build`
  runs `"pack": "go run ./scripts/packdist"`, so the web build requires a
  Go toolchain today. Only the *plugin binaries on PATH* remain a real
  concern, and buf v2's array form for `local:` plugins removes it.
- `go.mod` is `go 1.26.0` and already uses a `tool` directive
  (`tool github.com/sqlc-dev/sqlc/cmd/sqlc`), so pinning the Go protoc
  plugins as tool dependencies follows existing repo practice.
- Both generated trees are committed (`git ls-files` confirms
  `api/gen/proto/go/**` and `web/src/gen/**`).

## Approach

Keep one template and make every plugin toolchain-relative, so the
template runs anywhere the repo's existing prerequisites (Go toolchain +
`pnpm install`) are met:

- The three Go plugins become `go.mod` `tool` dependencies and are invoked
  as `local: ["go", "run", "<pkg>"]` — version-pinned by `go.mod`, no
  PATH installation.
- `protoc-gen-es` is invoked by explicit path
  `../web/node_modules/.bin/protoc-gen-es` — version-pinned by
  `web/package.json` (`@bufbuild/protoc-gen-es` devDependency), no PATH
  reliance, and works both from `go generate` (no pnpm PATH injection)
  and from the web script.
- `web`'s `gen` script drops `--template buf.gen.ts.yaml` and runs the
  default template.

Rejected alternatives (see DECISION.md):

- TS generated solely from `go generate`, web `gen` step dropped — loses
  the self-refreshing `pnpm build` and silently builds against stale
  committed TS after a proto edit.
- Keeping two templates with a lint/CI check that they stay in sync —
  keeps the duplication the issue calls meaningless and adds machinery.

## Public surface delta

Build-tooling surface only; no exported Go/TS symbol changes.

```
removed: api/buf.gen.ts.yaml
```

```yaml
# api/buf.gen.yaml — plugins section becomes
plugins:
  - local: ["go", "tool", "protoc-gen-go"]
    out: gen/proto/go
    opt: paths=source_relative
  - local: ["go", "tool", "protoc-gen-go-grpc"]
    out: gen/proto/go
    opt: paths=source_relative
  - local: ["go", "tool", "protoc-gen-connect-go"]
    out: gen/proto/go
    opt: paths=source_relative
  - local: ../web/node_modules/.bin/protoc-gen-es
    out: ../web/src/gen
    opt:
      - target=ts
      - import_extension=js
```

```
# go.mod — added tool directives (versions land in require blocks via `go get -tool`)
tool connectrpc.com/connect/cmd/protoc-gen-connect-go
tool google.golang.org/grpc/cmd/protoc-gen-go-grpc
tool google.golang.org/protobuf/cmd/protoc-gen-go
```

```json
// web/package.json — "gen" script becomes
"gen": "cd ../api && buf generate"
```

## Implementation steps

1. `go get -tool google.golang.org/protobuf/cmd/protoc-gen-go
   google.golang.org/grpc/cmd/protoc-gen-go-grpc
   connectrpc.com/connect/cmd/protoc-gen-connect-go` — adds the `tool`
   directives; run `go mod tidy`. Verify: `go tool` (or
   `go run <pkg> --version`) resolves each plugin without PATH installs.
2. Rewrite the `plugins:` entries of `api/buf.gen.yaml` to the array/path
   forms in the Public surface delta, and fold the surviving parts of
   `api/buf.gen.ts.yaml`'s header comment (why `protoc-gen-es` comes from
   `web/node_modules`, the `pnpm install` prerequisite) into it. Verify:
   `cd api && buf generate` regenerates both trees; `git diff` shows no
   content change under `api/gen/proto/go` or `web/src/gen`.
3. Delete `api/buf.gen.ts.yaml` and change `web/package.json`'s `gen`
   script to `cd ../api && buf generate`. Verify: `pnpm -C web run gen`
   succeeds and `git status` stays clean.
4. Update the comment in `api/generate.go` to state the one remaining
   cross-toolchain prerequisite: `pnpm install` must have populated
   `web/node_modules` before `go generate ./api`. Verify: `go generate
   ./api` from a tree where that holds leaves `git status` clean.

## Testing and verification

- `go generate ./api && git diff --exit-code api/gen web/src/gen` — the
  unified template reproduces the committed output byte-for-byte.
- `pnpm -C web run gen && git diff --exit-code web/src/gen api/gen` —
  the web entry point produces the identical result.
- `pnpm -C web run build` — full frontend build (gen → vendor → vite →
  packdist) still passes; `git diff --exit-code web/dist.tar.zst` guards
  the reproducible-archive property `web/scripts/packdist` documents.
- `go build ./...` and existing tests stay green (generated Go unchanged).

## Risks

- `go run` per plugin invocation adds a small build-cache warm-up on first
  run; negligible afterward.
- `local:` with an argv array requires buf v2 template semantics — the
  templates are already `version: v2`; if the installed buf rejects the
  array form, pin/raise the buf version rather than reverting to PATH
  binaries.
- A stale `web/node_modules` (protoc-gen-es version drift vs. committed
  TS) would show up as a diff in step-2/3 verification, not silently.

## Open questions

None — resolved automatically per user directive; see DECISION.md.
