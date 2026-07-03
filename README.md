# crabswarm

Swarm Claude Code using tmux sessions, so you can chat with Claude while lying in bed.

## Markdown previewer

`crabswarm preview` serves a browser-based, GitHub-flavored markdown previewer
with live reload, multi-root support, a file tree, and a toggleable table of
contents.

```console
# Add the current directory (default) as a preview root and print its URL.
crabswarm preview

# Add a specific directory.
crabswarm preview ./docs

# List the roots the running daemon knows about.
crabswarm preview list

# Drop a root by its NAME or ID (see `preview list`).
crabswarm preview remove docs
```

The first `crabswarm preview` starts the preview server as a background daemon
under [cmdman](https://github.com/ngicks/cmdman) (expected on `PATH`);
subsequent invocations just add another root to the running daemon. `preview`,
`preview list`, and `preview remove` all talk to that daemon, so they require
cmdman for the daemonized path.

Stopping is cmdman's job:

```console
cmdman stop crabswarm-preview
```

If you don't want cmdman, run the server in the foreground yourself — the same
process cmdman would daemonize:

```console
crabswarm preview __serve --addr 127.0.0.1:6419
```

`preview __serve` needs no cmdman; add roots against it with `crabswarm preview
list`/`remove` or the ConnectRPC API directly.

### Configuration

The `preview` block of the crabswarm config (defaults < file < flags; these keys
are file-only and are not settable via environment variables):

| Key                   | Default            | Meaning                                        |
| --------------------- | ------------------ | ---------------------------------------------- |
| `preview.addr`        | `0.0.0.0:6419`     | TCP listen address of the preview HTTP server. |
| `preview.daemon_name` | `crabswarm-preview`| cmdman command name the previewer runs under.  |

`preview.addr` can also be overridden per-invocation with `--addr`.

### Security warning

The default `preview.addr` is `0.0.0.0:6419` so phones and tablets on the
tailnet (Tailscale MagicDNS) can reach it. There is no authentication: the
server exposes registered roots' file contents to anyone who can reach the
port. On untrusted networks, set `preview.addr` to `127.0.0.1:6419` or firewall
the port accordingly.

## License

[Unlicense](LICENSE) - Public Domain
