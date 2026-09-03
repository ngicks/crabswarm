---
tags: chat notify cmdman test flaky
---

# `crabswarm/chat/notify` tests time out under load (2026-09-03)

Running `go test ./...` while other test runs shared the machine failed
four notify tests (`TestSendKeys_NudgesDoneAgent`,
`TestSendKeys_NudgesMemberWedgedInABusyState`,
`TestSendKeys_SanitizesSenderAddress`,
`TestTerminal_SendCommandTypesThenSubmits` in
`crabswarm/chat/internal/cmdman`) with
`cmdman send-keys "…" "Enter": signal: killed` — a 3 s deadline on a
fake cmdman subprocess. The packages pass alone in well under a second.

Follow-up: make the deadline load-tolerant (scale it, or drive the fake
in-process instead of as a subprocess) so a busy CI box cannot fail
them.
