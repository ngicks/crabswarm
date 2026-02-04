# AGENTS.md

This file provides guidance to LLM cli agents when working with code in this repository.

## Important

- Use serena where possible.
- Use context7 for tool specific knowledge.
- If you are not `codex`:
  - ALWAYS output plans to under `./doc/plans`, following file format `${RFC3339-DATETIME-FORMAT}-${name-on-plan}.md`
  - ALWAYS ask `codex` to review your plan, using `codex review`.
- In difficult reserach, complex planning, ask `codex` for help using `codex mcp` tool.
- Remenber to use the agent-memory skill the moment user's preference become prominent.
