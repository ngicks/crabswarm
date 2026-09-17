// OpenCode plugin: the crabswarm-mcp wiring for a harness that has no hook
// file. OpenCode loads JavaScript or TypeScript plugins and delivers session,
// permission and tool events to them, so this file is the counterpart of
// hooks/hooks.json: each handler maps one OpenCode event onto one
// `crabswarm chat` verb. The delivery wording is the same text the hook file
// carries.
//
// It also carries what no hook file needs: the relay the MCP server hands a
// mention to. The plugin is the only thing here that knows which session the
// person is driving, so a mention that arrives while the agent is idle comes
// back from the server over loopback and is prompted into that session.
//
// Nothing here writes to stdout: in TUI mode the plugin shares it with the
// screen, so what is worth recording goes through the SDK's own log. Every
// command runs quietly and a failure is ignored, because a chat nobody is
// hosting must never break a turn. Only `node:` builtins may be imported:
// OpenCode installs @opencode-ai/plugin beside its plugins, but loading must
// not wait for that install.

const DELIVERED_MID_TURN =
  "[crabswarm chat] Messages just arrived. Reply with the `chat_send` tool to any line marked [mentioned you]; otherwise carry on."
const DELIVERED_AT_IDLE =
  "[crabswarm chat] Messages arrived while you were working. Act on anything addressed to you, which is every line marked [mentioned you], reply with the `chat_send` tool, then finish."

// The variables the server cannot work without, forwarded to the MCP server
// the same way the Codex declaration's env_vars and the Claude Code plugin's
// env entries forward them.
const FORWARDED = ["CMDMAN_CMD_ID", "CRABSWARM_CHAT_TOKEN", "XDG_RUNTIME_DIR"]

// RELAY_ENV is where the MCP server reads the relay's address, and the whole
// of what makes an OpenCode member one the daemon stops typing at: a server
// started without it has no channel and attends as a terminal one.
const RELAY_ENV = "CRABSWARM_OPENCODE_RELAY"

type Shell = (strings: TemplateStringsArray, ...values: unknown[]) => {
  quiet(): { nothrow(): Promise<{ exitCode: number; stdout: { toString(): string } }> }
}

type Client = {
  app: {
    log(input: {
      body: { service: string; level: string; message: string }
    }): Promise<unknown>
  }
  session: {
    get(input: { path: { id: string } }): Promise<{ data?: { parentID?: string } }>
    prompt(input: { path: { id: string }; body: { parts: { type: "text"; text: string }[] } }): Promise<unknown>
  }
}

// What this file uses of Bun's own globals, declared rather than imported:
// the plugin loads before anything is installed beside it.
declare const Bun: {
  serve(options: {
    hostname: string
    port: number
    fetch: (request: Request) => Promise<Response> | Response
  }): { port: number }
}

export const CrabswarmChat = async ({ client, $ }: { client: Client; $: Shell }) => {
  const token = process.env.CRABSWARM_CHAT_TOKEN || process.env.CMDMAN_CMD_ID

  const log = (level: string, message: string) =>
    client.app.log({ body: { service: "crabswarm-mcp", level, message } }).catch(() => {})

  // chat runs one `crabswarm chat` verb as this session's member and returns
  // its stdout, or nothing when the verb failed or the session has no token.
  const chat = async (...args: string[]): Promise<string> => {
    if (!token) return ""
    const result = await $`crabswarm chat ${args}`.quiet().nothrow()
    if (result.exitCode !== 0) return ""
    return result.stdout.toString()
  }
  const report = (state: string) => chat("report-state", state)

  // Only the session a person drives reports state. A subagent's session
  // carries a parentID, and its events would flip the display of a member
  // that is still working.
  const topLevel = new Map<string, Promise<boolean>>()
  const isTopLevel = (sessionID: string) => {
    let known = topLevel.get(sessionID)
    if (!known) {
      known = client.session
        .get({ path: { id: sessionID } })
        .then((r) => !r.data?.parentID)
        .catch(() => false)
      topLevel.set(sessionID, known)
    }
    return known
  }

  // The session the notices go to: the last top-level one the person wrote in.
  // There is no other way to know it — a session is not "current" to anything
  // the SDK exposes — and the person may drive several over one server.
  let current: string | undefined

  // deliver hands one notice to that session as a prompt of its own.
  //
  // Not through the TUI's own composer: appending to it and submitting would
  // submit whatever the person had half-written along with the notice. And not
  // awaited, like the idle delivery below — the call returns after the whole
  // reply, and the server waiting on the other end of the relay must not be
  // held for a turn.
  const deliver = (sessionID: string, text: string) => {
    void client.session
      .prompt({ path: { id: sessionID }, body: { parts: [{ type: "text", text }] } })
      .catch(() => {})
  }

  // The relay: one loopback listener the MCP server posts a mention to. The
  // server delivers nothing itself, so a plugin that cannot open this listener
  // simply declares no relay, and the member stays one the daemon types at.
  let relay: string | undefined
  try {
    const server = Bun.serve({
      hostname: "127.0.0.1",
      port: 0,
      fetch: async (request: Request) => {
        const url = new URL(request.url)
        if (request.method !== "POST" || url.pathname !== "/nudge") {
          return new Response("not found", { status: 404 })
        }
        let notice: { content?: unknown; from?: unknown }
        try {
          notice = await request.json()
        } catch {
          return new Response("the notice is not JSON", { status: 400 })
        }
        const text = typeof notice?.content === "string" ? notice.content : ""
        if (!text) return new Response("the notice carries no content", { status: 400 })
        const from = typeof notice?.from === "string" ? notice.from : ""
        const sessionID = current
        if (!sessionID) {
          // Refused rather than queued: the server leaves an undelivered
          // mention waiting and comes back to it when the agent next reports a
          // turn over, by which time there is a session to deliver it to.
          void log("warn", `a chat notice${from ? ` from ${from}` : ""} arrived ` +
            "before any session took a turn")
          return new Response("no session has taken a turn yet", { status: 409 })
        }
        deliver(sessionID, text)
        return new Response(null, { status: 202 })
      },
    })
    relay = `http://127.0.0.1:${server.port}/nudge`
  } catch (err) {
    void log("error", `the crabswarm chat relay could not listen: ${err}`)
  }

  return {
    // The MCP server is what attends the room; declaring it here is what makes
    // the plugin the whole install. A server the operator declared by hand
    // wins over this one — and is then one the daemon types at, since the
    // relay's address only reaches the server declared here.
    config: async (cfg: { mcp?: Record<string, unknown> }) => {
      cfg.mcp ??= {}
      if (cfg.mcp["crabswarm-mcp"]) return
      const environment: Record<string, string> = {}
      for (const name of FORWARDED) {
        const value = process.env[name]
        if (value) environment[name] = value
      }
      if (relay) environment[RELAY_ENV] = relay
      cfg.mcp["crabswarm-mcp"] = {
        type: "local",
        command: ["crabswarm", "mcp"],
        enabled: true,
        environment,
      }
    },

    "chat.message": async (input: { sessionID: string }) => {
      if (!(await isTopLevel(input.sessionID))) return
      current = input.sessionID
      await report("working")
    },

    "permission.ask": async (permission: { sessionID: string }) => {
      if (await isTopLevel(permission.sessionID)) await report("waiting")
    },

    // Mid-turn delivery: the messages a read handed over ride on the tool's
    // result, which is the closest OpenCode has to a hook's additionalContext.
    // A tool call completing is also the signal that a dialog resolved.
    "tool.execute.after": async (input: { sessionID: string }, output: { output: string }) => {
      if (!(await isTopLevel(input.sessionID))) return
      const messages = await chat("read", "--quiet")
      if (messages) output.output += `\n\n${DELIVERED_MID_TURN}\n\n${messages}`
      await report("working")
    },

    event: async ({ event }: { event: { type: string; properties: { sessionID: string } } }) => {
      switch (event.type) {
        case "permission.replied":
          if (await isTopLevel(event.properties.sessionID)) await report("working")
          return
        case "session.idle": {
          const sessionID = event.properties.sessionID
          if (!(await isTopLevel(sessionID))) return
          // The read reports the member done exactly when it hands nothing
          // over. What it found at idle becomes the next prompt, which is what
          // a blocked Stop is on the other harnesses.
          const messages = await chat("read", "--quiet", "--done-when-empty")
          if (!messages) return
          deliver(sessionID, `${DELIVERED_AT_IDLE}\n\n${messages}`)
        }
      }
    },
  }
}
