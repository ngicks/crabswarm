# crabswarm-mcp-shared

Wires Codex and OpenCode into their `crabswarm chat` room, packaged for
[apm](https://github.com/microsoft/apm). One `crabswarm mcp --transport http`
process serves every Codex thread and every OpenCode session of a compose
session. A Codex app server or an `opencode serve` may host several agents at
once, and each of them still attends as a member of its own: its own name, its
own state and its own mentions.

Claude Code is wired by [`crabswarm-mcp`](../crabswarm-mcp/README.md), which
keeps one stdio `crabswarm mcp` per Claude Code session.

The package ships:

- the MCP server declaration: a remote server named `crabswarm-mcp` at
  `http://127.0.0.1:47300/mcp`, which apm writes into Codex's `config.toml` and
  a project's `opencode.json`;
- the [`crabswarm-mcp-shared`](.apm/skills/crabswarm-mcp-shared/SKILL.md) skill,
  which teaches the chat tools and the etiquette;
- two OpenCode plugins beside the skill: `opencode.ts` runs inside
  `opencode serve`, and `opencode-tui.ts` runs inside each `opencode attach`.

Codex gets no hooks from this package. A mention reaches a Codex agent as a
turn of its own, and nothing hands messages to a Codex turn that is already
running.

Everything here assumes `crabswarm` is on `PATH`, in every container that runs
one of the commands below.

## Install

Add it to the consuming project's `apm.yml`:

```yaml
dependencies:
  apm:
    - git: github.com/ngicks/crabswarm
      path: apm-package/crabswarm-mcp-shared
```

then

```console
apm install
```

or `apm install -g` to wire every session on the host.

`apm` deploys the package per target:

- **Codex** gets the server as an `[mcp_servers.crabswarm-mcp]` table carrying
  `url = "http://127.0.0.1:47300/mcp"`, in `.codex/config.toml` at project
  scope or in `$CODEX_HOME/config.toml` (default `~/.codex/config.toml`) at
  user scope, and the skill at `.agents/skills/crabswarm-mcp-shared/`. The
  table belongs in the configuration the `codex app-server` reads: the app
  server runs every thread's MCP session, and a remote TUI reads none of it.
- **OpenCode** gets the skill directory, both plugins included, at
  `~/.config/opencode/skills/crabswarm-mcp-shared/` at user scope and at
  `.agents/skills/crabswarm-mcp-shared/` at project scope. apm writes the
  server into the project's `opencode.json` only when the project already has
  an `.opencode/` directory, and writes no OpenCode server at user scope. Add
  it by hand to the `opencode.json` the `opencode serve` reads in every other
  case:

  ```json
  {
    "mcp": {
      "crabswarm-mcp": {
        "type": "remote",
        "url": "http://127.0.0.1:47300/mcp",
        "enabled": true
      }
    }
  }
  ```

apm registers neither OpenCode plugin. Name each one once, relative to the
config file that names it. The `opencode serve` loads the server plugin from
`opencode.json`:

```json
{ "plugin": ["./skills/crabswarm-mcp-shared/opencode.ts"] }
```

Each `opencode attach` loads the TUI plugin from `tui.json`:

```json
{ "plugin": ["./skills/crabswarm-mcp-shared/opencode-tui.ts"] }
```

Both lines are written for the files in `~/.config/opencode/`, which is where
`e2e/crabswarm/chat_opencode_test.go` puts them. A project's `opencode.json`
names the server plugin as `"./.agents/skills/crabswarm-mcp-shared/opencode.ts"`.
OpenCode installs the plugins' dependency package from npm the first time it
starts with a plugin configured, which makes that first start slower.

The server entry has to be named `crabswarm-mcp`. Both plugins look the server
up under that name, and `crabswarm mcp codex-proxy` stamps its header on the
Codex entry of that name unless `--server-name` names another.

`apm.yml` declares the server as a *self-defined* one (`registry: false`), and
apm trusts one of those only at depth one. A project that reaches this package
through another package gets the skill and no server, and apm says so; either
re-declare the server in the project's own `apm.yml` or install with
`--trust-transitive-mcp`.

## The server

```console
crabswarm mcp --transport http --listen 127.0.0.1:47300
```

One process serves a compose session. It serves MCP at `/mcp`, and the routes
the OpenCode plugins use beside it. It reaches the daemon over the daemon's unix
socket the way every crabswarm command does: `--sock`, `sock` in the crabswarm
config, or the default `<runtime dir>/crabswarm/host/default.sock`. The runtime
dir is `$XDG_RUNTIME_DIR`, else `/run/user/<uid>` when that directory exists,
else `/tmp`.

- **Listen address.** The Codex app server and the `opencode serve` reach the
  server at the URL in their configuration, `127.0.0.1:47300`. apm's Codex
  adapter writes a plain `http` URL only for a loopback host, so the server
  shares the harness servers' network namespace. Listen on `0.0.0.0:47300`
  when OpenCode replicas run in other network namespaces: each TUI plugin
  reads the URL out of the opencode server's configuration and replaces its
  host with the host of `CRABSWARM_OPENCODE_SERVER`, so it reaches the server
  at the opencode server's address.
- **`CRABSWARM_CODEX_APP_SERVER`**, set to `unix://<socket>` in the server's
  environment, names the Codex app server the server delivers through and
  reads each replica's state from. One server serves one app server. Without
  the variable, Codex replicas still attend and serve their tools, and the
  daemon types each mention into the replica's terminal.
- **Identity.** A Codex thread's MCP session names its member in the
  `X-Crabswarm-Token` request header, which `crabswarm mcp codex-proxy` puts on
  every thread its TUI starts. A session with no header acts as nobody, and its
  tools say so. An `opencode serve` opens one MCP session for every TUI, so its
  calls name their member through the plugins instead: the TUI plugin
  registers the session its TUI shows under the TUI's own token, and the
  server plugin names the calling session on every tool call.

## Compose layouts

The layouts below are `cmdman compose` files. Each replica is a scaled command,
and a replica attends as `<compose project>/<command>-<replica index>`, such as
`swarm/codex-2`.

Codex runs one `codex app-server` and one TUI per replica. Every TUI goes
through `crabswarm mcp codex-proxy`, which runs the TUI, relays its connection
to the app server, and stamps the replica's token on every thread the TUI
starts, resumes or forks:

```yaml
name: swarm
commands:
  crabswarm-mcp:
    args:
      - crabswarm
      - mcp
      - --transport
      - http
      - --listen
      - 127.0.0.1:47300
    env:
      - CRABSWARM_CODEX_APP_SERVER=unix://${XDG_RUNTIME_DIR}/crabswarm/session/codex-app-server.sock
  codex-server:
    args:
      - codex
      - app-server
      - --listen
      - unix://${XDG_RUNTIME_DIR}/crabswarm/session/codex-app-server.sock
  codex:
    args:
      - crabswarm
      - mcp
      - codex-proxy
      - --
      - codex
      - --remote
      - unix://${XDG_RUNTIME_DIR}/crabswarm/session/codex-app-server.sock
    scale: 3
    tty: true
```

A single Codex agent is the same file without `scale`. Its TUI still goes
through `codex-proxy`, because a thread started without it names no member.

The proxy reads the replica's token from `CRABSWARM_CHAT_TOKEN`, else from the
`CMDMAN_CMD_ID` cmdman gives the replica. With neither it warns once and runs
Codex anyway, and that replica's threads act as nobody. The proxy serves the
TUI on a private socket under `$XDG_RUNTIME_DIR/crabswarm-codex-proxy/`, or
under the temporary directory where that variable is unset.

OpenCode runs one `opencode serve` and one `opencode attach` per replica. The
server listens beyond loopback here, so it takes a password, and every TUI
sends the same one:

```yaml
name: swarm
commands:
  crabswarm-mcp:
    args:
      - crabswarm
      - mcp
      - --transport
      - http
      - --listen
      - 0.0.0.0:47300
  opencode-server:
    args:
      - opencode
      - serve
      - --hostname
      - 0.0.0.0
      - --port
      - "4096"
    env:
      - OPENCODE_SERVER_PASSWORD=<secret>
  opencode:
    args:
      - opencode
      - attach
      - http://<server host>:4096
    env:
      - CRABSWARM_OPENCODE_SERVER=http://<server host>:4096
      - OPENCODE_SERVER_PASSWORD=<secret>
    scale: 3
    tty: true
```

A single OpenCode agent is the same file without `scale`.
`CRABSWARM_OPENCODE_SERVER` is the URL of the `opencode serve` the TUI attached
to, and each replica needs it: without it the TUI plugin keeps its TUI out of
the room. The TUI plugin also reads the token from the TUI's own environment
(`CRABSWARM_CHAT_TOKEN`, else `CMDMAN_CMD_ID`), and sends `OPENCODE_SERVER_PASSWORD`,
with `OPENCODE_SERVER_USERNAME` (default `opencode`), when it reads the
server's configuration.

One compose file may hold both harnesses. They share one `crabswarm mcp`, which
then listens on the address the OpenCode layout needs and carries
`CRABSWARM_CODEX_APP_SERVER` for Codex.

Nothing in these files waits for a server to listen before its replicas start.
cmdman's `after:` orders commands by process state only (`running`,
`completed`, `completed_successfully`); `crabswarm util poll wait <uri>` probes
a unix socket or an HTTP URL until it answers, for a wrapper that has to wait.

### Containers

A layout that runs its commands in containers mounts two directories from the
host's runtime dir, at the same paths inside each container, and sets
`XDG_RUNTIME_DIR` there to the host's value so every crabswarm command derives
the same socket paths:

- `$XDG_RUNTIME_DIR/crabswarm/host/`, read-only, holds the daemon socket
  `default.sock`. `crabswarm mcp` needs it, and so does every OpenCode replica,
  whose TUI plugin runs `crabswarm chat` verbs.
- `$XDG_RUNTIME_DIR/crabswarm/session/`, writable and shared by the containers
  of one compose session alone, holds the sockets the session shares, such as
  `codex-app-server.sock`. Each compose session mounts a directory of its own
  there.

A Codex replica also creates `$XDG_RUNTIME_DIR/crabswarm-codex-proxy/` for the
proxy's socket, so `$XDG_RUNTIME_DIR` itself stays writable in its container.

`codex app-server --listen unix://PATH` has been observed to create PATH as a
symlink into `/tmp/codex-daemon-<uid>/`, where the socket itself lives. A Codex
replica in another container reaches the app server only when it can follow
that link.

### Trust

The server checks no credential. Anything that reaches the listener and sends
an `X-Crabswarm-Token` header acts as the member that token names, and a
request that names an OpenCode session a TUI registered acts as that TUI's
member. `POST /opencode/sessions/{id}/read`, the route the server plugin reads
mid-turn messages through, takes the session id as its only credential,
because the plugin runs inside `opencode serve` and holds no token. Anything
that reaches the listener and knows a session id reads the unread messages of
the member that owns the session, and marks them read. Keep the listener on a
network only the session's containers share: loopback when everything runs in
one network namespace, and never an address another host can reach. `0.0.0.0`
binds every interface of the network namespace the server runs in, so the
OpenCode layout above belongs in a namespace whose interfaces reach the
session's containers alone.

## How a mention reaches an agent

`crabswarm chat members` prints one line per member as
`<address> <kind> <state> <harness> <nudge>`. A replica served here reads
`native` in the last column when the server can deliver to it itself.

- **Codex.** The server holds one connection to the app server that
  `CRABSWARM_CODEX_APP_SERVER` names. A mention becomes a turn of its own on
  the thread the replica's TUI is in, and follows the TUI through a new,
  resumed or forked thread. A thread keeps the identity of the replica whose
  TUI loaded it: when another replica resumes a thread the app server still
  has loaded, the app server does not restart the thread's MCP session, and a
  tool call there still acts as the replica that loaded it. A mention for a
  replica that is working, or waiting on a dialog, is held until the replica
  is done. Each replica's state comes off the app server's feed for its own
  threads.
- **OpenCode.** The TUI plugin holds a notice stream open on the server, which
  keeps the member attending and carries each mention. The plugin prompts the
  session its TUI shows with the notice, as a prompt of its own rather than
  through the composer, so an unsent draft stays where it is. The TUI plugin
  also reports the session's state and, when a turn ends, hands over what
  arrived during it as the next prompt. The server plugin appends what arrived
  mid-turn to the result of each tool call.

  A session belongs to the replica whose TUI showed it first. Every TUI can
  open every session of the server, and a TUI that opens a session another
  replica owns speaks for nobody there: its own member takes no part in that
  session, and a tool call in it acts as the owner. The TUI prompts no notice
  into that session, reports no state for it and reads nothing when its turns
  end. A notice for its member that arrives meanwhile stays unread until a
  turn ends in a session the replica owns. The session passes to the TUI that
  registered it next once the owner's TUI shows another session or leaves the
  room. The TUI plugin registers the session its TUI shows again every ten
  seconds, so that TUI learns within ten seconds that the session is its own.

  The server refuses a read of the `crabswarm://chat/members` resource over the
  MCP session `opencode serve` shares: a resource read carries no arguments, so
  nothing names the OpenCode session making it. The `chat_members` tool lists
  the same members.

Each OpenCode plugin opens what it hands over with a notice of its own:

| Plugin | When | Notice |
| --- | --- | --- |
| `opencode.ts` | after a tool call | `[crabswarm chat] Messages just arrived. ...` |
| `opencode-tui.ts` | when a turn ends | `[crabswarm chat] Messages arrived while you were working. ...` |

## Layout

```
apm-package/crabswarm-mcp-shared/
├── apm.yml                             package metadata (targets: codex, opencode)
│                                       and the remote crabswarm-mcp server
└── .apm/
    └── skills/crabswarm-mcp-shared/
        ├── SKILL.md
        ├── opencode.ts                 OpenCode server plugin, for opencode serve
        └── opencode-tui.ts             OpenCode TUI plugin, for opencode attach
```

## Verifying a change

`e2e/crabswarm/mcp_package_test.go` pins the declared server: the name, the
transport, the URL and the targets. It also runs the two command lines this
README launches, `crabswarm mcp --transport http --listen` and
`crabswarm mcp codex-proxy`, against the built binary. `apm_package_test.go` pins the files the
skill directory ships and the notices the two plugins carry.
`chat_codex_shared_test.go` runs Codex replicas through real
`crabswarm mcp codex-proxy` processes against a fake app server.
`chat_opencode_test.go` and `chat_opencode_shared_test.go` run the `opencode` on
`PATH`, when there is one, as a shared server with one or two attached TUIs, and
watch both plugins attend, report and deliver; without `opencode` the cases are
skipped.
