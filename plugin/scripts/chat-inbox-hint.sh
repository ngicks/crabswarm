#!/usr/bin/env sh
# PostToolUse inbox delivery: hand over anything that arrived mid-turn, right
# after a tool call, so a teammate's message does not wait for the turn to end.
#
# It injects the messages themselves rather than a "you have mail" hint,
# because there is no peek: `crabswarm chat read` drains. Injecting what it
# drained is the delivery — the text reaches the agent as additionalContext.
# A bare hint would cost the same round trip and leave the agent to fetch what
# this script already holds.
#
# Shared verbatim by Claude Code and Codex: both feed the same snake_case
# envelope on stdin and both read a camelCase hookSpecificOutput back.
set -u

# The harness writes the event to stdin whether or not this script wants it;
# draining the pipe keeps the harness from writing into a closed one.
cat >/dev/null 2>&1

# Without jq the messages could not be encoded back to the harness, so they
# are left in the inbox for the Stop-hook drain.
command -v jq >/dev/null 2>&1 || exit 0

# Silence on every failure: this runs after every tool call, and a daemon that
# is not there is not news the agent needs mid-task.
messages=$(crabswarm chat read 2>/dev/null) || exit 0
[ -n "$messages" ] && [ "$messages" != "no pending messages" ] || exit 0

# additionalContext only. Codex rejects an output object carrying fields its
# schema does not know, and a rejected output after a drain loses the message.
printf '%s' "$messages" | jq -Rs '{
  hookSpecificOutput: {
    hookEventName: "PostToolUse",
    additionalContext: (
      "[crabswarm chat] Messages just arrived. Reply with `crabswarm chat send <name|team/name> <text>` if any is addressed to you; otherwise carry on.\n\n" + .
    )
  }
}'
