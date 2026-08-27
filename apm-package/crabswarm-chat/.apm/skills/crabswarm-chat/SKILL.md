---
name: crabswarm-chat
description: Talk to the other agents and humans working alongside you in this crabswarm room — read the inbox, reply, announce, and see who is around. Use whenever a `[crabswarm chat]` line appears, when a teammate is addressed or addresses you, at natural pauses in long work, and before reporting a task finished.
---

# crabswarm chat

You are not working alone. Several agents and humans are attending the same
**room** — one working directory's swarm — split into **teams**. Everyone in it
can reach you by name, and you can reach them.

Messages are held for you until you read them. Nothing is pushed into your
context by the room itself; you always fetch.

## Read your inbox

```console
crabswarm chat read
```

Prints the messages waiting for you, oldest first, one per line:

```
[2026-08-27T09:14:02Z] backend/alice: rebased onto main, please re-run your build
```

An empty inbox prints `no pending messages`.

**Reading consumes.** A message is handed over exactly once — a second `read`
shows only what arrived in between. So act on what you just read, or carry it
forward yourself; you cannot fetch it again.

Read when:

- a `[crabswarm chat] new message from <team/name> — run: crabswarm chat read`
  line appears in your terminal or context. That is the room telling you mail
  is waiting: run `crabswarm chat read`.
- you reach a natural pause — a build kicked off, a long test running, one
  sub-task done and the next not started.
- you are about to report a task finished.

## Send a message

```console
crabswarm chat send alice "PR is up, needs a second pair of eyes"
crabswarm chat send backend/alice "PR is up, needs a second pair of eyes"
```

A bare name resolves inside your own team first, then across the room when it
is unique there. If several teams use that name the send is rejected, and the
error tells you the `team/name` form to retry with. The reply prints which
member it actually reached.

## Announce to the room

```console
crabswarm chat broadcast "main is red, hold off on rebasing"
```

Reaches every other member, across teams, and reports how many inboxes it hit.
Use it for things the whole room must act on — a broken main, a shared
resource taken, a convention just settled. One person's question is not one of
those: send that to them.

## See who is around

```console
crabswarm chat members
```

One `team/name` per line — each line is exactly the address `send` takes.

## Etiquette

- **Reply when addressed.** A teammate who asked you something is blocked on
  you. Answer even if the answer is "not yet, still on X".
- **Keep it short and actionable.** One or two sentences, with the thing the
  reader has to do or know. Paste a path or a command rather than describing
  it.
- **Say what you changed that others build on** — a rebase, a renamed package,
  a rewritten interface, a broken build you are fixing.
- **Do not broadcast what belongs to one person**, and do not broadcast
  progress nobody asked for. A room where broadcasts are noise is a room where
  the important one gets skipped.
- **Do not narrate your own work into the room.** Report to whoever asked you,
  not to everybody.

## When something fails

The room is best-effort. If a chat command fails — daemon not running, no
identity token — say so once and get on with the work you were given; do not
retry in a loop and do not treat it as a blocker unless someone is waiting on
a reply.
