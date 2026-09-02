---
tags: apm-package codex hooks upstream
---

# apm path substitution corrupts `\n\n` in deployed codex hook templates (2026-09-02)

The deployed `~/.codex/hooks.json` (PostToolUse and Stop entries)
contains the literal
`<home>/.apm/apm_modules/ngicks/crabswarm/apm-package/crabswarm-chat/n/n`
where the source template
(`apm-package/crabswarm-chat/.apm/hooks/report-state.json`) has `\n\n`:
apm's path rewriting treats the backslash-n sequence as a relative
path and expands it. The rendered `context`/`blockDecision` text that
codex agents receive mid-turn therefore carries a bogus path instead of
a paragraph break. Unrelated to the missed nudge, but it is this
package's text that gets mangled. External tool bug (apm-cli 0.29.0).

Follow-up: report upstream; meanwhile avoid backslash escapes in the
hook templates that pass through apm (e.g. use a template function or
a literal newline inside the JSON string) and add a rendered-output
check to the package's e2e (`e2e/crabswarm/chat_mcp_package_test.go`).
