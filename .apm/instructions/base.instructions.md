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

### Implementing functionality

- Implement e2e tests if any existing test is not covering the case.
- Don't think too much about backward compatibility, since the app was never actually deployed

```
.
├── AGENTS.md
├── bin             git-ignore'd bin dir.
├── cmd             Entry point. cobra subcommand structure.
│   └── crabswarm
├── doc
│   └── rules       Some rule guidance sits here.
├── e2e             e2e test
│   └── crabswarm
├── pkg
│   ├── api         API definition and generated code / related type definitions. proto schema basically sit here.
│   ├── claudehook  helper for claude code hook. Same code can be resued for codex.
│   └── crabswarm   crabswarm implementation: hooks and server impl for claude / codex
└── plugin          claude code defintion. Basic part can be reused for codex.
```

## Implementing functionality

- Edit most relavant files to implement said functionality.
- You may ask back the user to resolve unclear corners.
- Implement e2e tests if any existing test is not covering the case.
