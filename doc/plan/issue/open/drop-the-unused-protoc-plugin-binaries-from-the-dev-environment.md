---
tags: devenv buf codegen
---

# Drop the unused protoc plugin binaries from the dev environment (2026-09-02)

The repo no longer invokes any PATH-installed protoc plugin:
`api/buf.gen.yaml` runs the Go plugins through `go tool` (pinned by
`go.mod` tool directives) and `protoc-gen-es` through
`web/node_modules/.bin/protoc-gen-es` (pinned by `web/package.json`).
The nix profile still provides `protoc-gen-go`, `protoc-gen-go-grpc`,
`protoc-gen-connect-go`, and `protoc-gen-es` on PATH — and the PATH
`protoc-gen-es` is 2.12.0 while the pinned one is 2.12.1, a real drift
trap if anything falls back to PATH. The dev-environment definition
lives outside this repository.

Follow-up: remove the four plugin packages from the dev-environment
definition.
