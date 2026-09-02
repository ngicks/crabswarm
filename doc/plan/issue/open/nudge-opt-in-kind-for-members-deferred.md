---
tags: chat notify cmdman
---

# Nudge opt-in kind for members (deferred) (2026-08-31)

Every cmdman-resolved member is unconditionally `KindAgent`
(`crabswarm/chat/service_member.go`), so a human joining chat from a
plain cmdman-tracked shell gets the keystroke nudge typed into the
shell and executed as a command line — the injection guards
(`crabswarm/chat/internal/cmdman/cmdman.go`,
`crabswarm/chat/notify/notify.go`) cannot tell a harness from a shell.
Deferred for now because the admin plane (see the `chat admin` entry)
covers the human case: admin never joins as a member.

Follow-up (if plain-shell membership returns): make nudging opt-in at
registration/join — e.g. an `--agent` flag meaning "notify by
keystroke injection"; without it, the member is inbox-only and the
notify path never types at its terminal.
