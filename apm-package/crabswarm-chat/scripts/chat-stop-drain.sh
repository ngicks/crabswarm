#!/usr/bin/env sh
# Stop-hook inbox drain: hand the agent what arrived during the turn instead
# of letting it go idle with the room still talking to it.
#
# `crabswarm chat read` consumes — a message it hands over exists nowhere
# else afterwards. Every branch below therefore either delivers what it
# drained or does not drain at all, so the worst outcome is a late read and
# never a lost message.
#
# Shared verbatim by Claude Code and Codex: both feed the same snake_case
# envelope on stdin and both read a camelCase decision back.
set -u

# Reporting idle is what re-arms the daemon's terminal nudge for this member,
# so it runs on every path that lets the stop through.
report_idle() {
	crabswarm chat report-state idle >/dev/null 2>&1 || true
}

input=$(cat)

# Without jq there is no way to encode a drained message back to the harness,
# so the inbox is left untouched and the messages wait for the next turn.
if ! command -v jq >/dev/null 2>&1; then
	report_idle
	exit 0
fi

# Only an explicit false drains. stop_hook_active means an earlier Stop hook
# already blocked this turn; reading again would either loop the agent or,
# once the harness stops honoring the block, swallow messages it never
# showed. Anything unreadable is treated the same way, since an inbox left
# full costs a late read while a drain nobody delivers costs the message.
active=$(printf '%s' "$input" | jq -r '.stop_hook_active // false' 2>/dev/null) || active=true
if [ "$active" != "false" ]; then
	report_idle
	exit 0
fi

# Daemon unreachable: nothing was handed over, so nothing is lost.
messages=$(crabswarm chat read 2>/dev/null) || {
	report_idle
	exit 0
}

if [ -z "$messages" ] || [ "$messages" = "no pending messages" ]; then
	report_idle
	exit 0
fi

# Messages in hand — block the stop so they reach the agent. No idle report
# here: the turn is about to continue, so this member is running.
#
# The output carries decision and reason only. Codex rejects an output object
# with fields its schema does not know, and a rejected output after a drain is
# exactly the lost message this script exists to prevent.
printf '%s' "$messages" | jq -Rs '{
  decision: "block",
  reason: (
    "[crabswarm chat] Messages arrived while you were working. Act on anything addressed to you, reply with `crabswarm chat send <name|team/name> <text>`, then finish.\n\n" + .
  )
}'
