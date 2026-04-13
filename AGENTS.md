# AGENTS.md

This file provides guidance to LLM cli agents when working with code in this repository.

## Important

- Use context7 for tool specific knowledge.
- For requests to update or reimplement interfaces, consult `./doc/rules/claude_sdk_interface_conversion.md` first. If the rule is incomplete or ambiguous, ask the user clarifying questions one at a time and update that document as decisions are made.
- If you are not `codex`:
  - In difficult reserach, complex planning, ask `codex` for help using `codex exec` tool.
- You might be in a restricted enviroment: some commands may fail and some special files may not be present (e.g. `/dev/kvm`).
- Use the **persistent-memory** skill for user preferences and anything the user explicitly asks to remember. Consult stored memories at the start of each conversation.
