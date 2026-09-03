---
tags: buf codegen web devenv docs
---

# Fresh worktrees need `pnpm install` before `buf generate` (2026-09-03)

`api/buf.gen.yaml` runs `protoc-gen-es` from
`../web/node_modules/.bin/protoc-gen-es`, and `web/node_modules` is
git-ignored, so in a freshly added worktree `go generate ./api/...`
fails until `pnpm install --frozen-lockfile` has been run in `web/`.
Nothing in the repo says so; the first proto change in a new worktree
finds out the hard way. Related: "Drop the unused protoc plugin binaries
from the dev environment" covers the PATH-installed copies of the same
plugins.

Follow-up: document the prerequisite next to the generate directive
(`api/generate.go`) and in the repo instructions, or have the generate
step run the install itself when `web/node_modules` is missing.
