---
tags: chat notify docs
---

# godoc nit: [nudgeable] link in SendKeys doc points at an unexported func (2026-09-02)

The `[nudgeable]` doc link in the exported `SendKeys` comment
(`crabswarm/chat/notify`) points at an unexported func and renders as
plain text.

Follow-up: reword or export what the link needs.
