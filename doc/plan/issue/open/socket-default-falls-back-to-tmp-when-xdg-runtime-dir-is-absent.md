---
tags: config env socket
---

# Socket default falls back to /tmp when XDG_RUNTIME_DIR is absent (2026-09-02)

`defaultSockPath` (`crabswarm/config.go`) derives the socket from
`$XDG_RUNTIME_DIR` and falls back to `/tmp/crabswarm/default.sock`.
Any client launched with a stripped env (codex MCP servers, see the
entry above) therefore dials `/tmp/...` while the daemon listens on
`/run/user/<uid>/crabswarm/default.sock`. Because `cli.Dial` is lazy
(`grpc.NewClient`), the bridge handshakes fine and then every tool
fails on join, indefinitely and quietly. Verified: `env -i HOME=$HOME
PATH=$PATH crabswarm config` prints `"sock": "/tmp/crabswarm/default.sock"`.

Follow-up: derive the default without depending on the env — e.g.
`/run/user/<os.Getuid()>` when it exists, then `$XDG_RUNTIME_DIR`,
then `/tmp` — and/or have the apm package forward `XDG_RUNTIME_DIR`
(see above). A pinned `sock` in `~/.config/crabswarm/config.json` is
the operator workaround today.
