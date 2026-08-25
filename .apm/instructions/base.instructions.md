---
description: "Basic instructions for the project"
applyTo: "*"
---

### General

Tools to swarm claude(, codex and others!)

### Tech stack

- Go
  - `github.com/spf13/cobra` for subcommands
  - Using gRPC and protobuf for communications
- TypeScript (`web/` frontend: Preact SPA for `crabswarm preview`)
  - connect-web + buf-generated types from the proto schema

### Implementing functionality

- Implement e2e tests if any existing test is not covering the case.
- Don't think too much about backward compatibility, since the app was never actually deployed

```
.
├── AGENTS.md
├── api             API definition and generated code / related type definitions. proto schema basically sit here.
├── bin             git-ignore'd bin dir.
├── cmd             Entry point. cobra subcommand structure.
│   └── crabswarm
├── crabswarm       crabswarm implementation: hooks and server impl for claude / codex
├── doc
│   └── rules       Some rule guidance sits here.
├── e2e             e2e test
│   └── crabswarm
├── internal        internal helper packages (libver, loggerfactory, templateutil, ...).
├── pkg
│   └── claudehook  helper for claude code hook. Same code can be resued for codex.
├── plugin          claude code defintion. Basic part can be reused for codex.
└── web             Preact SPA for `crabswarm preview`, embedded via go:embed as seekable-zstd tar (dist.tar.zst committed).
```

## Implementing functionality

- Edit most relavant files to implement said functionality.
- You may ask back the user to resolve unclear corners.
- Implement e2e tests if any existing test is not covering the case.
