# Handoff — unify buf generate templates

Deferred/out-of-scope discoveries from the implementation run
(2026-09-01). Entries are candidates for `doc/plan/issue/issue.md`.

# Drop the now-unused protoc plugin binaries from the dev environment

The repo no longer invokes any PATH-installed protoc plugin:
`api/buf.gen.yaml` runs the Go plugins through `go tool` (pinned by
`go.mod` tool directives) and `protoc-gen-es` through
`web/node_modules/.bin/protoc-gen-es` (pinned by `web/package.json`).
The nix profile still provides `protoc-gen-go`, `protoc-gen-go-grpc`,
`protoc-gen-connect-go`, and `protoc-gen-es` on PATH — and the PATH
`protoc-gen-es` is 2.12.0 while the pinned one is 2.12.1, a real drift
trap if anything ever falls back to PATH. The dev-environment definition
lives outside this repository, so it could not be fixed here; remove the
four plugin packages from it.
