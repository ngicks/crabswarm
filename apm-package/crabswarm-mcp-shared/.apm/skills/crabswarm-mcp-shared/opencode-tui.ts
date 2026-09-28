// OpenCode TUI plugin of crabswarm-mcp-shared: the half that runs in each TUI
// attached to a shared `opencode serve` (`opencode attach`). It runs in the
// TUI's own process, with the TUI's own environment and so with the chat token
// of the agent the TUI is, which the server and its plugin never have.
//
// It makes the TUI a member of the room. It holds a notice stream open on the
// crabswarm MCP server beside `opencode serve`, which keeps the member attending
// and is the channel a mention reaches it through, and it tells that server
// which session the TUI shows, so a tool call from that session acts as this
// member. It also reports the session's state and hands over what arrived by
// the time a turn ends. The delivery line opens with the `[crabswarm chat]`
// marker every notice from the room opens with, and it names the `chat_send`
// tool those notices name. It carries the messages themselves because the read
// that found them has already marked them read.
//
// CRABSWARM_OPENCODE_SERVER is the URL of the `opencode serve` the TUI attached
// to. The MCP server's address is read off that server's config, where the
// crabswarm-mcp entry is a remote one.
//
// Nothing here writes to stdout: the TUI draws on it, so what is worth
// recording goes through the server's log. Every failure is logged and
// ignored, because a chat nobody is hosting must never break the TUI. Only
// `node:` builtins may be imported: loading must not wait for OpenCode to
// install @opencode-ai/plugin beside the plugin.

import { execFile } from "node:child_process"

const DELIVERED_AT_IDLE =
  "[crabswarm chat] Messages arrived while you were working. Act on anything addressed to you, which is every line marked [mentioned you], reply with the `chat_send` tool, then finish."

const SERVER_ENV = "CRABSWARM_OPENCODE_SERVER"
// MCP_ENTRY is the name the crabswarm MCP server is configured under.
const MCP_ENTRY = "crabswarm-mcp"
const TOKEN_HEADER = "X-Crabswarm-Token"

// The TUI announces no route change, so the route is read this often.
const ROUTE_POLL = 300
// A dropped notice stream is opened again after RETRY_MIN, doubling up to
// RETRY_MAX while it keeps failing.
const RETRY_MIN = 1_000
const RETRY_MAX = 30_000
// COMMAND_TIMEOUT bounds one `crabswarm chat` verb.
const COMMAND_TIMEOUT = 30_000

type Route = { name: string; params?: { sessionID?: unknown } }

type Event = {
  properties: { sessionID?: string; status?: { type?: string } }
}

type McpConfig = { mcp?: Record<string, { url?: unknown } | undefined> }

// What this file uses of the API a TUI plugin is handed.
type Api = {
  route: { readonly current: Route }
  event: { on(type: string, handler: (event: Event) => void): () => void }
  client: {
    app: {
      log(input: { service: string; level: string; message: string }): Promise<unknown>
    }
    session: {
      get(input: { sessionID: string }): Promise<{ data?: { parentID?: string } }>
      promptAsync(input: {
        sessionID: string
        parts: { type: "text"; text: string }[]
      }): Promise<{ error?: unknown }>
    }
  }
  lifecycle: { signal: AbortSignal }
}

const tui = async (api: Api) => {
  const signal = api.lifecycle.signal
  const log = (level: string, message: string) => {
    api.client.app.log({ service: "crabswarm-mcp", level, message }).catch(() => {})
  }

  const token = process.env.CRABSWARM_CHAT_TOKEN || process.env.CMDMAN_CMD_ID
  const server = process.env[SERVER_ENV]
  if (!token) {
    log("info", "this TUI has no chat token (CRABSWARM_CHAT_TOKEN or CMDMAN_CMD_ID); " +
      "it stays out of the crabswarm room")
    return
  }
  if (!server) {
    log("warn", `${SERVER_ENV} is not set to the opencode server this TUI attached to; ` +
      "it stays out of the crabswarm room")
    return
  }

  // chat runs one `crabswarm chat` verb as this TUI's member, which the verb
  // resolves from the same environment, and returns its stdout, or nothing when
  // the verb failed.
  const chat = (...args: string[]) =>
    new Promise<string | undefined>((resolve) => {
      execFile("crabswarm", ["chat", ...args], { timeout: COMMAND_TIMEOUT }, (err, stdout) => {
        if (err) {
          log("warn", `crabswarm chat ${args[0]} failed: ${err.message}`)
          resolve(undefined)
          return
        }
        resolve(stdout)
      })
    })

  // The verbs run one after another, in the order the events arrived: a read
  // at the end of one turn reports done, and a report of the next turn's work
  // must not land before it.
  let commands: Promise<void> = Promise.resolve()
  const queue = (run: () => Promise<void>) => {
    commands = commands.then(run).catch(() => {})
  }

  // reported is the state this plugin last reported, so a turn that says it is
  // busy at every step is reported working once.
  let reported: string | undefined
  const report = (state: string) =>
    queue(async () => {
      if (reported === state) return
      if ((await chat("report-state", state)) !== undefined) reported = state
    })

  // shown is the session this TUI shows and its notices go to: the last
  // top-level session its route showed. A subagent's session is one the person
  // looks into; the session they drive is still its parent.
  let shown: string | undefined

  // prompt hands text to a session as a prompt of its own. Not through the
  // TUI's composer: submitting it would submit whatever the person had
  // half-written along with the text. promptAsync returns once the prompt is
  // queued rather than after the reply.
  const prompt = (sessionID: string, text: string) => {
    api.client.session
      .promptAsync({ sessionID, parts: [{ type: "text", text }] })
      .then((r) => {
        if (r?.error) log("warn", `prompting ${sessionID} failed: ${JSON.stringify(r.error)}`)
      })
      .catch((err) => log("warn", `prompting ${sessionID} failed: ${err}`))
  }

  // atIdle is the end of a turn. The read reports the member done exactly when
  // it hands nothing over, and what it found becomes the next prompt.
  const atIdle = (sessionID: string) =>
    queue(async () => {
      const messages = await chat("read", "--quiet", "--done-when-empty")
      reported = messages === "" ? "done" : undefined
      if (messages) prompt(sessionID, `${DELIVERED_AT_IDLE}\n\n${messages}`)
    })

  // Every attached TUI hears every session's events, so only the session this
  // one shows speaks for its member.
  api.event.on("session.status", (event) => {
    const { sessionID, status } = event.properties
    if (!sessionID || sessionID !== shown) return
    if (status?.type === "idle") atIdle(sessionID)
    else report("working")
  })
  api.event.on("permission.asked", (event) => {
    if (event.properties.sessionID && event.properties.sessionID === shown) report("waiting")
  })
  api.event.on("permission.replied", (event) => {
    if (event.properties.sessionID && event.properties.sessionID === shown) report("working")
  })

  // base is the directory the MCP server's routes sit in, beside its /mcp
  // endpoint, once it is known.
  let base: URL | undefined
  // streaming says a notice stream is open, which registering needs.
  let streaming = false

  // locate reads the MCP server's address off the opencode server's config.
  // The server names the MCP server as it reaches it itself: the MCP server
  // runs beside it, so a loopback host there is the opencode server's host
  // here, which may be across a network namespace.
  const locate = async (): Promise<URL> => {
    const serverURL = new URL(server.endsWith("/") ? server : `${server}/`)
    const headers: Record<string, string> = {}
    const password = process.env.OPENCODE_SERVER_PASSWORD
    if (password) {
      const username = process.env.OPENCODE_SERVER_USERNAME || "opencode"
      headers.Authorization = `Basic ${btoa(`${username}:${password}`)}`
    }
    const response = await fetch(new URL("config", serverURL), { headers, signal })
    if (response.status !== 200) {
      throw new Error(`GET ${serverURL}config answered ${response.status}`)
    }
    const config = (await response.json()) as McpConfig
    const url = config.mcp?.[MCP_ENTRY]?.url
    if (typeof url !== "string") {
      throw new Error(`the opencode server configures no remote MCP server named ${MCP_ENTRY}`)
    }
    const mcp = new URL(url)
    mcp.hostname = serverURL.hostname
    return new URL(".", mcp)
  }

  // Registrations are sent one at a time, so the last session shown is the
  // last one the server hears.
  let registration: Promise<void> = Promise.resolve()
  const register = () => {
    registration = registration.then(async () => {
      const sessionID = shown
      if (!sessionID || !base || !streaming) return
      try {
        const response = await fetch(new URL("opencode/session", base), {
          method: "PUT",
          headers: { [TOKEN_HEADER]: token, "Content-Type": "application/json" },
          body: JSON.stringify({ sessionID }),
          signal,
        })
        if (response.status !== 204) {
          log("warn", `registering ${sessionID} with crabswarm answered ` +
            `${response.status}: ${await response.text()}`)
        }
      } catch (err) {
        if (!signal.aborted) log("warn", `registering ${sessionID} with crabswarm failed: ${err}`)
      }
    })
  }

  // deliver prompts one notice into the session shown. With none shown yet it
  // is dropped: the mention stays unread, and the read that ends the person's
  // first turn hands it over.
  const deliver = (line: string) => {
    let notice: { content?: unknown; from?: unknown }
    try {
      notice = JSON.parse(line)
    } catch {
      log("warn", `a crabswarm notice is not JSON: ${line}`)
      return
    }
    const text = typeof notice?.content === "string" ? notice.content : ""
    if (!text) return
    const sessionID = shown
    if (!sessionID) {
      const from = typeof notice.from === "string" && notice.from ? ` from ${notice.from}` : ""
      log("warn", `a chat notice${from} arrived before this TUI showed a session`)
      return
    }
    prompt(sessionID, text)
  }

  const sleep = (ms: number) =>
    new Promise<void>((resolve) => {
      const done = () => {
        clearTimeout(timer)
        signal.removeEventListener("abort", done)
        resolve()
      }
      const timer = setTimeout(done, ms)
      signal.addEventListener("abort", done)
    })

  // listen holds the notice stream open for as long as the TUI runs. The
  // server forgets the session a member showed when its last stream closes, so
  // every stream that opens registers it again.
  const listen = async () => {
    let delay = RETRY_MIN
    while (!signal.aborted) {
      try {
        base ??= await locate()
        const response = await fetch(new URL("opencode/notices", base), {
          headers: { [TOKEN_HEADER]: token },
          signal,
        })
        if (response.status !== 200 || !response.body) {
          throw new Error(`GET /opencode/notices answered ${response.status}: ` +
            `${await response.text()}`)
        }
        delay = RETRY_MIN
        streaming = true
        register()
        await readLines(response.body, deliver)
      } catch (err) {
        if (signal.aborted) return
        // The address is read again: the MCP server may have moved.
        base = undefined
        log("warn", `the crabswarm notice stream failed: ${err}`)
      } finally {
        streaming = false
      }
      await sleep(delay)
      delay = Math.min(delay * 2, RETRY_MAX)
    }
  }

  // parentOf is the parent of a session, null for a top-level one, and
  // undefined when the server could not say.
  const parentOf = async (sessionID: string): Promise<string | null | undefined> => {
    const r = await api.client.session.get({ sessionID }).catch(() => undefined)
    if (!r?.data) return undefined
    return r.data.parentID ?? null
  }

  // follow registers the session the route shows once it changes. A session
  // the server could not describe is asked about again on the next poll, and
  // logged once.
  let looked: string | undefined
  let unknown: string | undefined
  let following = false
  const follow = async () => {
    const route = api.route.current
    const sessionID =
      route?.name === "session" && typeof route.params?.sessionID === "string"
        ? route.params.sessionID
        : undefined
    if (!sessionID || sessionID === looked || following) return
    following = true
    try {
      const parent = await parentOf(sessionID)
      if (parent === undefined) {
        if (unknown !== sessionID) log("warn", `the opencode server could not describe ${sessionID}`)
        unknown = sessionID
        return
      }
      looked = sessionID
      if (parent !== null) return
      shown = sessionID
      register()
    } finally {
      following = false
    }
  }

  const timer = setInterval(() => void follow(), ROUTE_POLL)
  signal.addEventListener("abort", () => clearInterval(timer))
  void follow()
  void listen()
}

// readLines calls each with every non-empty line of body until it ends.
const readLines = async (body: ReadableStream<Uint8Array>, each: (line: string) => void) => {
  const reader = body.getReader()
  const decoder = new TextDecoder()
  let pending = ""
  for (;;) {
    const { done, value } = await reader.read()
    if (done) return
    pending += decoder.decode(value, { stream: true })
    let end = pending.indexOf("\n")
    while (end >= 0) {
      const line = pending.slice(0, end).trim()
      pending = pending.slice(end + 1)
      if (line) each(line)
      end = pending.indexOf("\n")
    }
  }
}

export default { id: "crabswarm-mcp-shared", tui }
