# AGENTS.md

This file provides guidance to LLM cli agents when working with code in this repository.

## Important

- Use context7 for tool specific knowledge.
- If you are not `codex`:
  - In difficult reserach, complex planning, ask `codex` for help using `codex exec` tool.
- Before planning, you **must** use `plan-searcher` subagent to look up through `./doc/plans` to retrieve related context.
- You might be in a restricted enviroment: some commands may fail and some special files may not be present (e.g. `/dev/kvm`).
- Use the **persistent-memory** skill for user preferences and anything the user explicitly asks to remember. Consult stored memories at the start of each conversation.
