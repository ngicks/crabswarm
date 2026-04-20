# AGENTS.md

This file provides guidance to LLM cli agents when working with code in this repository.

## Important

- Use context7 for tool specific knowledge.
- For requests to update or reimplement interfaces defined under `pkg/api/gen/proto/go/sdktypes/`, consult `./doc/rules/claude_sdk_interface_conversion.md` first. If the rule is incomplete or ambiguous, ask the user clarifying questions one at a time and update that document as decisions are made.
- If you are not `codex`:
  - In difficult reserach, complex planning, ask `codex` for help using `codex exec` tool.
- You might be in a restricted enviroment: some commands may fail and some special files may not be present (e.g. `/dev/kvm`).
- If you are `claude code`: `codex` will review your output
- If you are `codex`: `claude code` will review your output

## Tech stack

- Go
  - `github.com/spf13/cobra` for subcommands
  - Using gRPC and protobuf for communications

## Structure overview

```
.
├── AGENTS.md
├── bin             git-ignore'd bin dir.
├── cmd             Entry point. cobra subcommand structure.
│   ├── cmdman
│   └── crabswarm
├── doc
│   └── rules       Some rule guidance sits here.
├── e2e             e2e test
│   ├── cmdman
│   └── crabswarm
├── pkg
│   ├── api         API definition and generated code / related type definitions. proto schema basically sit here.
│   ├── claudehook  helper for claude code hook. Same code can be resued for codex.
│   ├── cmdman      cmdman implementation: a simple command daemon. It's like Podman without pods, tmux without terminals.
│   ├── crabswarm   crabswarm implementation: hooks and server impl for claude / codex
│   └── mux         terminal multiplexer helper implementation.
└── plugin          claude code defintion. Basic part can be reused for codex.
```

## Implementing functionality

- Edit most relavant files to implement said functionality.
- You may ask back the user to resolve unclear corners.
- Implement e2e tests if any existing test is not covering the case.
