// OpenCode server plugin: the half of the crabswarm-mcp wiring that runs inside
// a shared `opencode serve`. The server hosts every TUI attached to it and opens
// one session on the crabswarm MCP server for all of them, so a tool call there
// names no member of its own. This plugin names the OpenCode session making
// each call, and the MCP server acts as the member whose TUI shows that session.
// opencode-tui.ts, running in each attached TUI with that TUI's own token, is
// what says which member shows which session.
//
// The plugin runs with the server's environment, which carries a token of its
// own when the server itself runs under cmdman. Acting with that token would
// make every TUI one member, so nothing here reads a token or runs a
// `crabswarm chat` verb: attendance, state and delivery at idle are the TUI
// plugin's. What is left is the mid-turn delivery, which only the server sees:
// the result of a tool call is the closest OpenCode has to a hook's
// additionalContext, and the MCP server reads for the member showing the
// calling session. The delivery wording is the same text the Codex hook file
// carries.
//
// The crabswarm MCP server is a remote entry of the OpenCode config, which is
// where this plugin finds its address. Only `node:` builtins may be imported:
// OpenCode installs @opencode-ai/plugin beside its plugins, but loading must not
// wait for that install. Nothing writes to stdout, and a failure is logged and
// ignored, because a chat nobody is hosting must never break a turn.

const DELIVERED_MID_TURN =
  "[crabswarm chat] Messages just arrived. Reply with the `chat_send` tool to any line marked [mentioned you]; otherwise carry on."

// MCP_ENTRY is the name the crabswarm MCP server is configured under. OpenCode
// names that server's tools `<entry>_<tool>`, replacing every character outside
// [a-zA-Z0-9_-] with "_", which leaves this name as it is.
const MCP_ENTRY = "crabswarm-mcp"
const TOOL_PREFIX = `${MCP_ENTRY}_`

// SESSION_ARG is the argument the MCP server takes the calling session from. It
// takes it off the call before any tool sees it.
const SESSION_ARG = "_crabswarm_session"

// READ_TIMEOUT bounds the read a tool result waits on, so an MCP server that
// stopped answering holds no turn up for long.
const READ_TIMEOUT = 10_000

type Client = {
  app: {
    log(input: {
      body: { service: string; level: string; message: string }
    }): Promise<unknown>
  }
  session: {
    get(input: { path: { id: string } }): Promise<{ data?: { parentID?: string } }>
  }
}

// ToolOutput is what `tool.execute.after` hands over. A built-in tool's result
// is `output`; an MCP tool's is the MCP result itself, whose text parts OpenCode
// joins with newlines afterwards; a subagent's task that failed has neither.
type ToolOutput = {
  output?: unknown
  content?: { type: string; text?: string }[]
}

export const CrabswarmChatShared = async ({ client }: { client: Client }) => {
  const log = (level: string, message: string) =>
    client.app.log({ body: { service: "crabswarm-mcp", level, message } }).catch(() => {})

  // The directory the MCP server's routes sit in, beside its /mcp endpoint.
  let base: URL | undefined

  // Only a session a person drives has a member to read for. A subagent's
  // session carries a parentID, and no TUI ever registers one.
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

  // read returns what the member showing sessionID has unread, marked read, or
  // nothing. A session no TUI shows is not an error: a session started over the
  // HTTP API, or from a TUI without the plugin, has no member.
  const read = async (sessionID: string): Promise<string> => {
    if (!base) return ""
    try {
      const response = await fetch(
        new URL(`opencode/sessions/${encodeURIComponent(sessionID)}/read`, base),
        { method: "POST", signal: AbortSignal.timeout(READ_TIMEOUT) },
      )
      if (response.status === 200) return await response.text()
      if (response.status !== 204 && response.status !== 404) {
        void log("warn", `the crabswarm chat read for ${sessionID} answered ` +
          `${response.status}: ${await response.text()}`)
      }
    } catch (err) {
      void log("warn", `the crabswarm chat read for ${sessionID} failed: ${err}`)
    }
    return ""
  }

  return {
    config: async (cfg: { mcp?: Record<string, { url?: unknown } | undefined> }) => {
      const url = cfg.mcp?.[MCP_ENTRY]?.url
      if (typeof url !== "string") {
        void log("warn", `no remote MCP server named ${MCP_ENTRY} is configured; ` +
          "chat messages are not delivered mid-turn")
        return
      }
      try {
        base = new URL(".", url)
      } catch (err) {
        void log("error", `the ${MCP_ENTRY} url ${url} is not a URL: ${err}`)
      }
    },

    // The argument is set on the object OpenCode goes on to call the tool with,
    // so it is added in place rather than by replacing args.
    "tool.execute.before": async (
      input: { tool: string; sessionID: string },
      output: { args?: Record<string, unknown> },
    ) => {
      if (!input.tool.startsWith(TOOL_PREFIX) || !output.args) return
      output.args[SESSION_ARG] = input.sessionID
    },

    // The place to append is settled before the read: the read marks what it
    // returns as read, and a result with nowhere to carry it would lose it.
    "tool.execute.after": async (input: { sessionID: string }, output: ToolOutput | undefined) => {
      const append = appender(output)
      if (!append || !(await isTopLevel(input.sessionID))) return
      const messages = await read(input.sessionID)
      if (messages) append(`${DELIVERED_MID_TURN}\n\n${messages}`)
    },
  }
}

// appender returns how to add text to a tool's result, or nothing for a result
// that carries no text. An MCP tool's text is carried in content: OpenCode reads
// its result from there after this hook, and ignores an output field on it.
const appender = (output: ToolOutput | undefined): ((text: string) => void) | undefined => {
  if (!output) return undefined
  if (typeof output.output === "string") {
    const result = output as { output: string }
    return (text) => {
      result.output += `\n\n${text}`
    }
  }
  const content = output.content
  if (Array.isArray(content)) {
    return (text) => {
      content.push({ type: "text", text: `\n${text}` })
    }
  }
  return undefined
}
