---
tags: logging daemon mcp
---

# Logging is discarded by default, so every chat failure above is invisible (2026-09-02)

`loggerfactory.BuildLogger` (`internal/loggerfactory/loggerfactory.go`)
returns `slog.DiscardHandler` unless `--log`/`--log-level` or
`CRABSWARM_LOG_FORMAT`/`CRABSWARM_LOG_LEVEL` is set, and there is no
log file. The bridge's "giving up on attending the chat room", the
watch loop's "room event feed ended", the daemon's nudge declines and
the codex startup error all vanish in a normal launch, which is why
each of the four reports above arrived as an observation of silence
rather than a message. Codex additionally hides MCP stderr.

Follow-up: default the daemon (`crabswarm serve`) and the bridge to a
warn-level stderr logger, and consider a file sink under
`$XDG_STATE_HOME/crabswarm/` for the bridge, whose stderr no harness
shows. Keep `--log-level debug` as the opt-in for the chatty lines.
