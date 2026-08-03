// Package codex defines the Codex (OpenAI codex-rs) hook tool input and
// output shapes that diverge from the Claude Agent SDK tool schemas.
//
// Codex emits hook payloads in the same envelope Claude Code uses
// (session_id, hook_event_name, tool_name, tool_input, tool_response, ...),
// but a handful of tools carry shapes the Claude SDK never produces:
//
//   - apply_patch — Codex edits files through a single freeform tool whose
//     tool_input is {"command": "<patch text>"} and whose tool_response is a
//     bare JSON string ("Exit code: 0\nWall time: ...\nOutput:\nSuccess.
//     Updated the following files:\nM path\n"). Claude has no equivalent.
//   - Bash — Codex's shell_command / exec_command tools both report the hook
//     tool_name "Bash". The tool_input ({"command": "..."}) matches Claude's
//     BashInput and is decoded with that type, but the tool_response is a bare
//     JSON string of raw output rather than Claude's BashOutput object.
//   - update_plan — Codex's plan tool, with no Claude equivalent.
//   - MCP tools (tool_name "mcp__<server>__<tool>") — tool_response is an MCP
//     CallToolResult object.
//
// Each type here preserves the exact original JSON bytes so a parsed value
// re-marshals byte-for-byte, while also exposing the decoded structure for
// callers that want more than the raw text. Tool names whose Claude SDK shape
// already matches Codex (Bash input, MCP input) are intentionally not
// redefined here; the Claude types decode them unchanged.
//
// Wire shapes are derived from openai/codex (codex-rs):
//   - codex-rs/core/src/tools/hook_names.rs (canonical hook tool names)
//   - codex-rs/core/src/tools/handlers/apply_patch.rs
//   - codex-rs/core/src/tools/handlers/shell/shell_command.rs
//   - codex-rs/core/src/tools/handlers/mcp.rs
//   - codex-rs/apply-patch/src/lib.rs (apply_patch summary format)
//   - codex-rs/core/src/tools/handlers/plan_spec.rs (update_plan schema)
package codex

// Canonical hook tool_name values Codex writes into the hook envelope's
// "tool_name" field. These are the serialized names, not the matcher aliases
// (apply_patch additionally matches "Write"/"Edit", spawn_agent matches
// "Agent"); matcher aliases are never serialized.
//
// Source: codex-rs/core/src/tools/hook_names.rs (HookToolName)
const (
	// ToolNameApplyPatch is the hook tool_name for file edits applied through
	// Codex's apply_patch freeform tool.
	ToolNameApplyPatch = "apply_patch"
	// ToolNameBash is the hook tool_name shared by Codex's shell_command and
	// exec_command tools.
	ToolNameBash = "Bash"
	// ToolNameUpdatePlan is the hook tool_name for Codex's plan tool.
	ToolNameUpdatePlan = "update_plan"
	// ToolNameSpawnAgent is the hook tool_name for Codex sub-agent spawning.
	ToolNameSpawnAgent = "spawn_agent"
	// ToolNameViewImage is the hook tool_name for Codex's image viewer.
	ToolNameViewImage = "view_image"
)

// McpToolNamePrefix marks Codex MCP tools, e.g. "mcp__server__tool". It is the
// same prefix Claude uses, so MCP tool inputs decode with the Claude McpInput
// type; only the MCP tool_response (a CallToolResult) is Codex-shaped here.
//
// Source: codex-rs/core/src/tools/handlers/mcp.rs (LEGACY_MCP_TOOL_NAME_PREFIX)
const McpToolNamePrefix = "mcp__"
