# Markdown Previewer (`crabswarm preview`)

One-line summary: add a browser-based GitHub-flavored markdown previewer to
crabswarm — a cmdman-daemonized Go HTTP server (connect-go API) with live
reload, multi-root support, a file-tree left drawer, a toggleable TOC right
panel, and a responsive Preact frontend.

## Goal / success criteria

- `crabswarm preview [ROOT]` adds ROOT (default `.`) to the previewer's root
  list, starting the preview daemon via cmdman if it is not already running;
  it prints the URL to open.
- Opening the URL in a browser (desktop, tablet, or phone in portrait) shows:
  - a left drawer with a root switcher and a file tree (nvim-tree / VSCode
    style, lazily expanded, all files listed),
  - the rendered markdown in the center, styled like GitHub (go-grip look),
    including math,
  - a toggleable TOC on the right.
- Saving a previewed file updates the rendered view without a manual refresh
  (live reload). Creating/deleting files updates the tree.
- Images and other assets referenced relatively from markdown resolve.
- Reachable from phones/tablets over the tailnet (Tailscale MagicDNS): the
  server listens on `0.0.0.0:<port>` by default.
- Works with the existing config layering (defaults < file < env < flags)
  and the repo's cobra conventions (thin `./cmd`, logic in `pkg/`).

## Scope / non-goals

In scope:
- New `preview` cobra command tree (`preview [ROOT]`, hidden
  `preview __serve`, `preview list`, `preview remove`).
- Proto definition `crabpreview/v1` + connect-go server / connect-web client.
- New `pkg/crabswarm/preview` service package (renderer, watcher, root
  registry, HTTP composition).
- New `web/` frontend (Preact + preact-iso + @preact/signals + Ark UI +
  daisyUI + @tanstack/preact-query), embedded via `go:embed` with a dev-FS
  mode.
- Config additions (`preview` section).

Non-goals:
- Editing files from the browser (read-only previewer).
- Rendering non-markdown files in the previewer UI — clicking one opens it
  raw with the browser's default behavior after a confirmation dialog.
- Authentication — the trust model is "tailnet or trusted LAN"; no token
  auth in v1 (documented warning for untrusted networks).
- Root list persistence across daemon restarts (in-memory only; decided).
- Building/forking cmdman itself — external binary we shell out to, same
  stance as the agent_swarming plan.

## Context (current code)

- CLI root: `cmd/crabswarm/commands/root.go` — `rootCmd()` wires
  subcommands; `previewCmd(cmd, &flagConfig)` follows the same pattern.
  Persistent flags: `--sock`, `--config`.
- Config: `pkg/crabswarm/config.go` — `Config` / `PartialConfig` layered
  merge; nested sub-configs follow the `exec.Config` / `exec.PartialConfig`
  pattern (`pkg/crabswarm/hook/exec`). A `Preview preview.Config` field
  follows the `HookExec` precedent.
- Proto/codegen: `pkg/api/buf.gen.yaml` (buf v2, managed mode,
  `protoc-gen-go` + `protoc-gen-go-grpc`, output
  `pkg/api/gen/proto/go`); schema under `pkg/api/schema/proto/crabhook/v1`.
  The preview API adds `crabpreview/v1` beside it and a
  `protoc-gen-connect-go` plugin entry; TS codegen for the frontend uses
  `protoc-gen-es` v2 (connect-es v2 needs no separate connect plugin).
- Existing server: `pkg/crabswarm/server/server.go` — gRPC over unix socket.
  The previewer is a separate process (browsers need TCP) with its own
  lifecycle, daemonized under cmdman.
- cmdman: external daemon manager (`github.com/ngicks/cmdman`, on PATH via
  mise). Relevant subcommands: `run` (create+start named command), `ls`,
  `inspect` (definition + runtime state), `wait`, `stop`, `rm`,
  `send-keys`. crabswarm shells out to it; same integration stance as
  `doc/plan/2026-06-27-01-agent_swarming`.
- CLI-presentation code convention: `pkg/crabswarm/cli/` (per
  `.claude/rules/go-design-preference.md`).
- e2e tests live under `e2e/crabswarm`.
- go-grip reference (verified from upstream `chrishrb/go-grip@main`):
  goldmark v1.7.16, chroma v2 (`github`/`github-dark`, CSS classes),
  goldmark-emoji, client-side mermaid + MathJax, in-repo extensions for
  alerts/tasklist/footnote/details/ghissue, vendored
  `github-markdown-{light,dark}.css`, WebSocket whole-page reload via
  `aarol/reload`.

## Approach

### Process model (decided — cmdman daemonization)

`crabswarm preview [ROOT]`:
1. Resolve ROOT to an absolute path (default `.`; must be an existing dir).
2. Check daemon presence: `cmdman inspect <daemon-name>` (name from config
   `preview.daemon_name`, default `crabswarm-preview`) — running?
3. If not running: `cmdman run --name <daemon-name> -- crabswarm preview
   __serve [--addr ...] [--config ...]`. `preview __serve` is a hidden
   cobra command (`Hidden: true`) that runs the server in the foreground;
   cmdman owns daemonization, logs, and restart.
4. Wait until the server listens (poll `GET /healthz` with timeout).
5. Call `AddRoot` over ConnectRPC; print the URL
   (`http://<host>:<port>/r/<rootId>/`).

`preview list` / `preview remove NAME|ID` are thin ConnectRPC clients
against the running daemon (error out with a hint if it is not running).
Stopping is cmdman's job (`cmdman stop crabswarm-preview`) — no wrapper in
v1.

### Package layout

```
cmd/crabswarm/commands/
  preview.go              # `preview [ROOT]`: cmdman ensure + AddRoot RPC
  preview_serve.go        # hidden `preview __serve`: runs preview.Service
  preview_list.go         # `preview list`
  preview_remove.go       # `preview remove`

pkg/api/schema/proto/crabpreview/v1/
  preview_service.proto   # PreviewService (see API below)

pkg/api/gen/proto/go/crabpreview/v1/   # buf-generated (go + connect-go)

pkg/crabswarm/preview/
  config.go               # preview.Config / PartialConfig
  service.go              # Service: store+watcher+renderer+http composition
  daemon.go               # cmdman shell-out: EnsureDaemon (inspect/run/wait)
  client.go               # connect client helpers for the CLI side
  roots.go                # in-memory RootStore (add/list/remove, stable IDs)
  tree.go                 # one-level directory listing (all files)
  watch.go                # fsnotify recursive watcher -> debounced event bus
  render/
    render.go             # goldmark pipeline: markdown -> HTML + TOC + title
    alert/                # GitHub alerts ext (vendored from go-grip, MIT)
    render_test.go        # golden tests
  httpapi/
    handler.go            # mux: connect handlers + /healthz + /raw + SPA
    raw.go                # asset serving with path-escape rejection
    static.go             # embedded SPA / dev-FS fallback
    handler_test.go

web/                      # Preact SPA (see Frontend below)
  embed.go                # //go:embed all:dist  (+ dev-FS switch)
  package.json, vite.config.ts, buf codegen for TS, src/...
```

### Public API (Go)

```go
package preview // pkg/crabswarm/preview

type Config struct {
    // Addr is the TCP listen address of the preview HTTP server.
    // Default "0.0.0.0:6419" — reachable over the tailnet by design.
    Addr string `json:"addr" yaml:"addr"`
    // DaemonName is the cmdman command name the previewer runs under.
    DaemonName string `json:"daemon_name" yaml:"daemon_name"` // default "crabswarm-preview"
}
type PartialConfig struct{ ... } // sparse-overlay pattern as exec.PartialConfig

type Root struct {
    ID   string // stable hash of the absolute path
    Path string // absolute
    Name string // display name (base name, deduped)
}

func New(logger *slog.Logger, cfg Config) (*Service, error)
func (s *Service) Serve(ctx context.Context) error // used by `preview __serve`

// CLI-side helpers (used by `preview`, `preview list`, `preview remove`):
func EnsureDaemon(ctx context.Context, logger *slog.Logger, cfg Config, configPath string) error
    // cmdman inspect -> cmdman run `crabswarm preview __serve` -> poll /healthz
func NewClient(addr string) crabpreviewv1connect.PreviewServiceClient
```

```go
package render // pkg/crabswarm/preview/render

type Document struct {
    HTML  template.HTML
    Title string    // first h1, falls back to file name
    TOC   []Heading // levels 1-4, IDs from the same AutoHeadingID pass
}
type Heading struct{ Level int; Text, ID string }

func New(opts Options) *Renderer
func (r *Renderer) Render(src []byte) (Document, error)
```

### API (decided — connect-go, proto-first)

`pkg/api/schema/proto/crabpreview/v1/preview_service.proto`:

```proto
service PreviewService {
  rpc ListRoots(ListRootsRequest) returns (ListRootsResponse);
  rpc AddRoot(AddRootRequest) returns (AddRootResponse);       // path -> Root
  rpc RemoveRoot(RemoveRootRequest) returns (RemoveRootResponse);
  rpc GetTree(GetTreeRequest) returns (GetTreeResponse);
  // root_id + dir path -> one level: [{name, type: DIR|FILE, is_markdown}]
  rpc GetDocument(GetDocumentRequest) returns (GetDocumentResponse);
  // root_id + file path -> {html, title, toc: [{level,text,id}], mtime}
  rpc WatchEvents(WatchEventsRequest) returns (stream Event);
  // server-streaming live-reload feed; Event is oneof:
  //   DocChanged{root_id, path} | TreeChanged{root_id, dir} | RootsChanged{}
}
```

Served with connect-go mounted on the same `http.ServeMux` as:
- `GET /healthz` — plain JSON, used by `EnsureDaemon` polling.
- `GET /raw/{rootId}/{path...}` — raw file bytes (images/assets from
  markdown, and non-markdown tree entries opened by the browser).
- `GET /`, `/r/{rootId}/...` — SPA index.html (embedded or dev FS).

Frontend consumes it with connect-es v2 (`@connectrpc/connect-web` +
`protoc-gen-es`-generated types) — generated TS client, no hand-mirrored
types. `WatchEvents` server-streaming works in browsers over fetch; the
client wraps it with reconnect-on-drop (retry with backoff), which replaces
SSE's native auto-reconnect.

Path safety: every `path` is cleaned and confirmed to stay under the root
(`filepath.IsLocal` + `filepath.Rel` check); symlinks escaping the root are
rejected. Applies to GetTree, GetDocument, and `/raw`.

### Markdown rendering (decided — goldmark, go-grip parity, math in v1)

- `github.com/yuin/goldmark` + `extension.GFM` (tables, strikethrough,
  linkify, task lists) + `extension.Footnote` + `parser.WithAutoHeadingID()`
- `github.com/yuin/goldmark-emoji`
- `github.com/yuin/goldmark-highlighting/v2` (chroma v2,
  `WithClasses(true)`; serve generated `github`/`github-dark` chroma CSS
  like go-grip)
- `go.abhg.dev/goldmark/mermaid` (`RenderModeClient`, mermaid.min.js bundled
  in the frontend)
- **Math (in v1)**: goldmark math extension emitting MathJax-compatible
  markup + client-side MathJax (`tex-mml-chtml.js`) — same split as
  go-grip's `pkg/mathjax`; vendor that extension if no maintained equivalent
  fits.
- GitHub alerts (`> [!NOTE]` family): vendor go-grip's MIT `pkg/alert` into
  `pkg/crabswarm/preview/render/alert` (proven output parity).
- `github-markdown-css` light/dark, switched together with the daisyUI
  theme.
- TOC extracted by walking the goldmark AST (levels 1–4); anchors come from
  the same `AutoHeadingID` pass so TOC links always match.

### Live reload

- `fsnotify` per root. fsnotify is non-recursive, so `watch.go` walks the
  root, watches each directory, and adds watches for newly created dirs
  (skipping `.git`, `node_modules`, and `GitListIgnorePatterns`-style
  ignores — reuse the pattern-match approach from `crabswarm git list`).
- Events debounced (~100ms per path) and fanned out to `WatchEvents`
  subscribers through a broadcast hub in `service.go` (errgroup-managed per
  `.claude/rules/go-basics.md`).
- Frontend: one `WatchEvents` stream; handlers call
  `queryClient.invalidateQueries(...)` — @tanstack/preact-query refetches,
  the view updates in place. No page reload; scroll and tree state
  preserved.

### Frontend

Stack (decided): Preact, preact-iso (router), @preact/signals (UI state),
Ark UI (headless behavior: TreeView, Dialog, Splitter), daisyUI on Tailwind
(theme + skin), **@tanstack/preact-query** (official Preact adapter —
https://tanstack.com/query/latest/docs/framework/preact/overview),
@connectrpc/connect-web + generated TS from `crabpreview/v1`.

```
web/src/
  main.tsx                # LocationProvider + QueryClientProvider + App
  routes.tsx              # "/" (root picker) and "/r/:rootId/*" (doc view)
  api/client.ts           # connect-web transport + PreviewService client
  api/events.ts           # WatchEvents stream w/ reconnect -> invalidation
  gen/                    # protoc-gen-es output (git-ignored, buf-generated)
  components/
    Layout.tsx            # responsive shell (drawer / content / toc)
    RootSwitcher.tsx      # root list at top of left drawer
    FileTree.tsx          # Ark TreeView, lazy GetTree per level, all files
    OpenRawDialog.tsx     # confirm dialog for non-markdown entries
    DocView.tsx           # rendered HTML, mermaid/MathJax init, anchor sync
    Toc.tsx               # right panel, active-heading highlight
    ThemeToggle.tsx       # daisyUI light/dark (syncs github-markdown css)
  signals/ui.ts           # drawerOpen, tocOpen, theme signals
```

File tree behavior (decided): list **all files** (dirs first). Markdown
files navigate in the SPA. Clicking a non-markdown file opens a
confirmation dialog (Ark Dialog); on confirm, open `/raw/{rootId}/{path}`
letting the browser do its default thing (render image/PDF, download,
etc.).

Responsive behavior:
- `lg` and up: persistent left sidebar (~280px tree), content, right TOC
  (~240px) toggleable; Ark Splitter optional.
- Below `lg` (phone/tablet portrait): daisyUI `drawer` — tree slides over
  from the left via hamburger; TOC becomes a right-side overlay drawer via a
  toolbar button. Content is full-width.

### Build / embed (decided — committed dist + dev FS; revised, see D7)

- Release path: vite build → `web/dist` **committed to git**, `web/embed.go`
  embeds it (`//go:embed all:dist`), `httpapi/static.go` serves it with SPA
  fallback. Committing is required, not optional: `go install
  github.com/ngicks/crabswarm/cmd/crabswarm@<tag>` builds from the module
  zip (derived from the git tag), so a git-ignored `dist` would make the
  embed match nothing and break `go install` entirely.
- A build script (sibling to `build-plugin-crabswarm.sh`) rebuilds
  `web/dist`; CI verifies freshness (rebuild + `git diff --exit-code
  web/dist`) so a stale committed dist can't ship. Diff noise is bounded:
  vite emits a handful of hashed bundles for a small SPA.
- Dev path: a `dev` build tag (or `CRABSWARM_PREVIEW_DEV_FS` env) makes
  `static.go` serve from the `web/` source tree / proxy to `vite dev`
  instead of the embed, so frontend iteration doesn't require Go rebuilds.
- `buf generate` covers both Go (connect-go) and TS (protoc-gen-es)
  outputs; the TS output under `web/src/gen` is committed for the same
  reason (`dist` build must be reproducible from a checkout without buf).

### Network binding (decided — tailnet-first)

Default `preview.addr` = `0.0.0.0:6419` so phones/tablets reach it via
Tailscale MagicDNS (`http://<host>.<tailnet>.ts.net:6419`). Documented
warning: the server exposes registered roots' file contents to anyone who
can reach the port — on untrusted LANs, set `preview.addr` to
`127.0.0.1:6419` or firewall accordingly. No auth in v1.

### Root registry (decided — in-memory)

`RootStore` is in-process state only. Each `crabswarm preview ROOT`
invocation (re-)adds; restarting the daemon starts empty. `RemoveRoot`
drops a root and its watches. Duplicate adds are idempotent (same ID).

### Rejected alternatives

- **Plain JSON REST**: less machinery, but the cmdman flow already needs an
  RPC client in the CLI, and the repo convention is proto-first; connect-go
  gives generated TS types for free. (User decision.)
- **SSE for live reload**: native auto-reconnect is nice, but it would be a
  second API surface beside connect; `WatchEvents` server-streaming keeps
  everything proto-defined, with reconnect handled in `api/events.ts`.
- **WebSocket whole-page reload** (go-grip's `aarol/reload`): loses scroll
  and tree state; query invalidation updates in place.
- **Client-side markdown rendering**: duplicate renderer in JS, bundle
  bloat, weaker highlighting; server-side goldmark keeps one canonical
  renderer.
- **Foreground probe-or-serve / extending `crabswarm serve`**: cmdman
  already solves daemon lifecycle (logs, restart, stop) and is the house
  tool; folding into `serve` couples hook IPC with an interactive UI.
- **`@tanstack/react-query` via preact/compat**: unnecessary — official
  `@tanstack/preact-query` adapter exists.
- **Git-ignored `web/dist` + build-before-build script**: breaks
  `go install` from the module proxy (embed matches nothing in the module
  zip); only works for repo clones. Committed dist with a CI freshness
  check instead.
- **Persisted root list**: stale-state complexity for little gain; cmdman
  keeps the daemon alive long-term anyway, and re-adding is one command.

## Implementation steps

Each step is independently buildable/testable.

1. **Proto + codegen** — add
   `pkg/api/schema/proto/crabpreview/v1/preview_service.proto`; extend
   `pkg/api/buf.gen.yaml` with `protoc-gen-connect-go` (and a TS
   `protoc-gen-es` target consumed by `web/`); generate; commit generated Go
   per existing repo practice.
2. **`pkg/crabswarm/preview/config.go`** — `Config`/`PartialConfig`
   (`Addr`, `DaemonName`), defaults; wire `Preview` field into
   `crabswarm.Config` / `crabswarm.PartialConfig`
   (`pkg/crabswarm/config.go`) following the `HookExec` pattern. Unit tests
   beside existing `config_test.go`.
3. **`pkg/crabswarm/preview/render/`** — goldmark pipeline producing
   `Document{HTML, Title, TOC}`, including vendored alert extension and math
   markup. Golden tests for GFM, alerts, code highlighting, math blocks,
   heading anchors, TOC extraction.
4. **`roots.go` + `tree.go`** — in-memory `RootStore` (add/list/remove,
   stable IDs, idempotent add) and safe one-level all-files directory
   listing with path-escape rejection tests.
5. **`watch.go`** — recursive fsnotify watcher with ignore patterns,
   debounce, and a broadcast hub; test with a temp dir.
6. **`httpapi/` + `service.go`** — connect handlers implementing
   `PreviewService` (including `WatchEvents` streaming), `/healthz`,
   `/raw/...`, static serving behind a placeholder `fs.FS`. httptest
   coverage including path-traversal attempts and a streaming-event test.
7. **`daemon.go` + `client.go`** — `EnsureDaemon` (cmdman inspect/run +
   healthz poll; unit-test with a fake `cmdman` on PATH) and connect client
   helpers.
8. **`web/`** — scaffold Vite+Preact app with the decided stack; connect-web
   client from generated TS; Layout, RootSwitcher, FileTree (+ raw-open
   dialog), DocView (mermaid/MathJax init), Toc, WatchEvents invalidation;
   `web/embed.go` with dev-FS switch; build script; commit `web/dist` +
   `web/src/gen` and add the CI freshness check (rebuild +
   `git diff --exit-code`). Verify responsive behavior at phone width and
   that a clean-checkout `go install ./cmd/crabswarm` works with no node
   toolchain.
9. **`cmd/crabswarm/commands/preview*.go`** — cobra wiring (use the
   `go-edit-cobra` skill): `preview [ROOT]` (+`--addr`), hidden
   `preview __serve`, `preview list`, `preview remove`.
10. **Polish** — README section, `e2e/crabswarm` test: start `preview
    __serve` directly (no cmdman dependency in CI), AddRoot, GetDocument,
    touch the file, observe a `WatchEvents` event; separate opt-in e2e for
    the cmdman path when `cmdman` is on PATH.

## Testing / verification

- Go unit tests per step (renderer goldens, path-safety, watcher, connect
  handlers, EnsureDaemon with fake cmdman).
- e2e tests under `e2e/crabswarm` as in step 10.
- Manual: `crabswarm preview .` on this repo; view README.md and
  `doc/plan/*` on desktop + a phone over the tailnet; edit a file and watch
  it live-update; toggle TOC; switch roots; click a non-markdown file and
  confirm the raw-open dialog.

## Risks

- **fsnotify recursion**: many-directory roots mean many watches (inotify
  limits). Mitigation: default ignore patterns (`.git`, `node_modules`),
  clear error when the watch limit is hit.
- **connect-web streaming reconnects**: `WatchEvents` needs hand-rolled
  reconnect/backoff (unlike SSE). Keep `api/events.ts` small and tested;
  events are invalidation hints only, so a missed event during reconnect is
  healed by a blanket invalidate-on-reconnect.
- **`@tanstack/preact-query` maturity**: the Preact adapter is newer than
  the React one; pin the version, and the compat-alias route remains a
  drop-in fallback.
- **Ark UI + daisyUI overlap**: Ark for behavior/a11y, daisyUI classes for
  skin; if a pairing fights, drop Ark per-component (e.g. hand-rolled tree).
- **cmdman coupling**: `preview` (root add) requires cmdman on PATH; keep
  `preview __serve` runnable directly so tests/CI and cmdman-less users have
  a path (documented).
- **0.0.0.0 default**: file contents exposed to whatever can reach the
  port; acceptable on a tailnet, documented for everything else.
- **Committed build artifacts**: `web/dist` (and `web/src/gen`) can drift
  from source. Mitigation: CI rebuilds and `git diff --exit-code`s them;
  the build script is the single entry point for regenerating both.

## Open questions

(none — all resolved 2026-07-02; see DECISION.md)
