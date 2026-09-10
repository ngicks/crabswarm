// OpenCode plugin: the crabswarm-chat wiring for a harness that has no hook
// file. OpenCode loads JavaScript or TypeScript plugins and delivers session,
// permission and tool events to them, so this file is the counterpart of
// hooks/hooks.json: every handler maps one OpenCode event onto one
// `crabswarm chat` verb and does nothing else. The delivery wording is the
// same text the hook file carries.
//
// Nothing here writes to stdout: in TUI mode the plugin shares it with the
// screen. Every command runs quietly and a failure is ignored, because a chat
// nobody is hosting must never break a turn. Only `node:` builtins may be
// imported: OpenCode installs @opencode-ai/plugin beside its plugins, but
// loading must not wait for that install.

const DELIVERED_MID_TURN =
  "[crabswarm chat] Messages just arrived. Reply with `crabswarm chat send <name|team/name> <text>` if any is addressed to you; otherwise carry on."
const DELIVERED_AT_IDLE =
  "[crabswarm chat] Messages arrived while you were working. Act on anything addressed to you, reply with `crabswarm chat send <name|team/name> <text>`, then finish."

// The variables the bridge cannot work without, forwarded to the MCP server
// the same way the Codex declaration's env_vars and the Claude Code plugin's
// env entries forward them.
const FORWARDED = ["CMDMAN_CMD_ID", "CRABSWARM_CHAT_TOKEN", "XDG_RUNTIME_DIR"]

type Shell = (strings: TemplateStringsArray, ...values: unknown[]) => {
  quiet(): { nothrow(): Promise<{ exitCode: number; stdout: { toString(): string } }> }
}

type Client = {
  session: {
    get(input: { path: { id: string } }): Promise<{ data?: { parentID?: string } }>
    prompt(input: { path: { id: string }; body: { parts: { type: "text"; text: string }[] } }): Promise<unknown>
  }
}

export const CrabswarmChat = async ({ client, $ }: { client: Client; $: Shell }) => {
  const token = process.env.CRABSWARM_CHAT_TOKEN || process.env.CMDMAN_CMD_ID

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

  return {
    // The bridge is what attends the room; declaring it here is what makes
    // the plugin the whole install. A server the operator declared by hand
    // wins over this one.
    config: async (cfg: { mcp?: Record<string, unknown> }) => {
      cfg.mcp ??= {}
      if (cfg.mcp["crabswarm-chat"]) return
      const environment: Record<string, string> = {}
      for (const name of FORWARDED) {
        const value = process.env[name]
        if (value) environment[name] = value
      }
      cfg.mcp["crabswarm-chat"] = {
        type: "local",
        command: ["crabswarm", "chat", "mcp"],
        enabled: true,
        environment,
      }
    },

    "chat.message": async (input: { sessionID: string }) => {
      if (await isTopLevel(input.sessionID)) await report("working")
    },

    "permission.ask": async (permission: { sessionID: string }) => {
      if (await isTopLevel(permission.sessionID)) await report("waiting")
    },

    // Mid-turn delivery: the messages a read drained ride on the tool's
    // result, which is the closest OpenCode has to a hook's additionalContext.
    // A tool call completing is also the signal that a dialog resolved.
    "tool.execute.after": async (input: { sessionID: string }, output: { output: string }) => {
      if (!(await isTopLevel(input.sessionID))) return
      const mail = await chat("read", "--quiet")
      if (mail) output.output += `\n\n${DELIVERED_MID_TURN}\n\n${mail}`
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
          // over. Mail found at idle becomes the next prompt, which is what a
          // blocked Stop is on the other harnesses. The prompt is not awaited:
          // the call returns after the whole reply, and the event bus must not
          // wait for a turn.
          const mail = await chat("read", "--quiet", "--done-when-empty")
          if (!mail) return
          void client.session
            .prompt({
              path: { id: sessionID },
              body: { parts: [{ type: "text", text: `${DELIVERED_AT_IDLE}\n\n${mail}` }] },
            })
            .catch(() => {})
        }
      }
    },
  }
}
