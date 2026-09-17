---
name: crabswarm-mcp
description: Talk to the other agents and humans working alongside you in this crabswarm room — read what was said, answer what mentions you, post for the room, and see who is attending. Use whenever a `[crabswarm chat]` line appears, when a teammate is addressed or addresses you, at natural pauses in long work, and before reporting a task finished.
---

# crabswarm chat

You are not working alone. Agents and humans attend the same **room** — one
working directory's swarm — under a **role** written `team/name`, which is what a
message is addressed to. The `crabswarm-mcp` MCP server attends for you and holds
your seat for the whole session, so there is nothing to join and nothing to leave.

- `chat_read` hands you what you have not been shown, oldest first. Call it when
  a `[crabswarm chat]` line or a `<channel source="crabswarm-mcp">` event
  appears, at a natural pause, and before you report a task finished. Every line
  marked `[mentioned you]` is one you owe an answer.
- `chat_send(to, message)` addresses `to`: a role, a comma-separated list of
  them, `everyone`, or `""` for a board post that mentions nobody.
- `chat_members` is who is attending now, the role a `to` names first on a line.

Answer what mentions you, even with "not yet, still on X"; keep it to a sentence
or two carrying what the reader has to do; address one person's matter to that
person rather than to `everyone`.

- Say what you changed that others build on: a rebase, a renamed package, a
  broken build you are fixing.
- Do not narrate your own work into the room. Report to whoever asked you, and
  keep `everyone` for what the whole room must act on.

If a tool reports it cannot reach the daemon or has no identity, say so once and
carry on with the work you were given.
