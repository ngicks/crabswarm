# Unify the two buf generate templates — how it should be

Gate: not confirmed (automatic decisions, pending user review)

## The should-be

There is exactly one description of how protobuf code is generated:
`api/buf.gen.yaml`. Every entry point — `go generate ./api` and the web
build's `gen` script — runs that one template and produces byte-identical
committed output (`api/gen/proto/go`, `web/src/gen`). No human ever keeps
two `managed` blocks or two `protoc-gen-es` stanzas in sync by hand.

## Use cases

### Backend developer changes a proto

- Actor: a developer editing `api/schema/**.proto`.
- Walkthrough: edit the proto, run `go generate ./api`, commit the
  regenerated Go and TS. One command, one template, done. No knowledge of
  a second TS-only template is needed, because it does not exist.

### Frontend build regenerates types

- Actor: `pnpm build` in `web/` (locally or CI).
- Walkthrough: the `gen` script runs the same single template. The build
  must not require the developer to pre-install Go protoc plugin binaries
  on PATH — the toolchains the build already needs (Go, since `pack` runs
  `go run ./scripts/packdist`; node_modules, since vite needs them) are
  sufficient.

### CI rebuild-and-diff check

- Actor: CI verifying committed generated code is fresh.
- Walkthrough: run generation once, `git diff --exit-code`. Because there
  is one template, there is no scenario where Go-path output and TS-path
  output disagree on the embedded FileDescriptor.

## Usability requirements

- Deleting `api/buf.gen.ts.yaml` must not leave a broken `pnpm build` on a
  machine with only the repo's already-documented prerequisites.
- The failure mode when a prerequisite is missing (e.g. `pnpm install` not
  run before `go generate ./api`) must be an obvious error naming the
  missing piece, and the prerequisite belongs in a comment at the point of
  use (`api/generate.go`).
