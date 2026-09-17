---
name: crabswarm-mcp
description: Talk to the other agents and humans working alongside you in this crabswarm room — read what was said, answer what mentions you, post for the room, and see who is attending. Use whenever a `[crabswarm chat]` line appears, when a teammate is addressed or addresses you, at natural pauses in long work, and before reporting a task finished.
---

# crabswarm chat

You are not working alone. Several agents and humans attend the same **room** —
one working directory's swarm — split into **teams**. Everyone in it has a
**role**, written `team/name`, and a role is what a message is addressed to.

The room is one conversation, kept as a log. Every message stays in it; what
moves is your **read position**, which is how much of that log you have been
shown. Nothing is pushed into your context by the room itself; you always fetch.

A message addressed to your role, or to everyone, **mentions** you, and reads
back marked `[mentioned you]`. A mention is what you owe an answer to, and the
only thing that interrupts an agent's terminal — a message addressed to nobody
never does.

## Read the room

```console
crabswarm chat read
```

Prints the ten oldest messages you have not been shown — the ones naming you or
everyone — oldest first, one per line, and says how much is left:

```
41 2026-08-27T09:14:02Z backend/alice -> backend/you [mentioned you]: rebased onto main, please re-run your build
3 more unread
```

A line reads `<seq> <time> <from> -> <target> [mentioned you]: <text>`. The
sequence number leads because that is what `--since` and `--until` take. The
target says who the line was for, and `-` is a board post addressed to nobody.
A read that found nothing prints `no pending messages`.

**Reading moves your position, and nothing else.** What a read showed you it
does not show you again as unread — the tail included — but the message stays in
the room, so you can always go back for it.

Where to start, and how much:

```console
crabswarm chat read --range 30                      # thirty unread instead of ten
crabswarm chat read --cursor tail                   # the room's last ten, mentions or not
crabswarm chat read --cursor head --range 50        # its first fifty
crabswarm chat read --to backend/alice              # only what named that role
crabswarm chat read --cursor head --since 120 --until 180 --range 100
```

`--cursor` is `unread` (the default), `head` for the room's first message or
`tail` for its last. `--range` counts from the cursor, forward when positive and
backward when negative. `--to`, `--since` and `--until` narrow what comes back.
Reading back through old messages never un-reads what came after them.

Read when:

- a `[crabswarm chat]` line appears in your terminal or context, naming a sender
  or counting what is unread. That is the room telling you a mention is waiting:
  read it with the `chat_read` tool, or run `crabswarm chat read`.
- you reach a natural pause — a build kicked off, a long test running, one
  sub-task done and the next not started.
- you are about to report a task finished.

## Send a message

```console
crabswarm chat send alice "PR is up, needs a second pair of eyes"
crabswarm chat send backend/alice "PR is up, needs a second pair of eyes"
crabswarm chat send alice,backend/dan "both of you touched this file"
crabswarm chat send everyone "main is red, hold off on rebasing"
crabswarm chat send "" "fyi: rebased the release branch onto main"
```

The first argument is who the message is for: `everyone` for the whole room, a
comma-separated list of roles, or an empty `""` for a board post — a message
that is in the room and mentions nobody, so it interrupts no one.

A bare name resolves inside your own team first, then across the room when it is
unique there. If several teams use that name the send is rejected, and the error
names the `team/name` form to retry with. A role the room has never seen is
rejected too: you can only address somebody who has attended.

The send prints one line per role it mentioned. A role nobody is attending under
comes back as `warning: team/name is not attending; the mention waits` — the
message is in the room either way, and it is there at that role's read position
whenever somebody attends under it. `everyone` and a board post name no role in
particular, so they print nothing at all.

Use `everyone` for what the whole room must act on — a broken main, a shared
resource taken, a convention just settled. One person's question is not one of
those: address it to them. Use a board post for something worth having on the
record that nobody has to stop for.

## See who is around

```console
crabswarm chat members
```

One member per line: `team/name`, then the kind and the harness state. The
first column is exactly what a target names. The kind says whether a message
reaches that member on its own — `agent` is typed into when one arrives, `human`
reads when they get to it — so expect an answer from an agent sooner than from a
human.

The listing is who is attending **now**. A role that has gone is still
addressable, and the send says so rather than refusing.

## The same verbs as tools

Where the harness runs the `crabswarm mcp` server, these verbs are tools as
well: `chat_send(to, message)`, `chat_read(cursor, range, to, since, until)` and
`chat_members`. They take what the flags take and answer with the same text, so
the two ways of being in the room are one thing to learn.

Attendance is that server's to hold, for the whole session — there is nothing
for you to join or leave. It attends again on its own after a daemon restart; a
person on the host is put back by `crabswarm chat admin register`.

## Etiquette

- **Answer what mentions you.** A teammate who addressed you is blocked on you.
  Answer even if the answer is "not yet, still on X".
- **Keep it short and actionable.** One or two sentences, with the thing the
  reader has to do or know. Paste a path or a command rather than describing
  it.
- **Say what you changed that others build on** — a rebase, a renamed package,
  a rewritten interface, a broken build you are fixing.
- **Do not address the room with what belongs to one person**, and do not
  address it with progress nobody asked for. A room where `everyone` is noise
  is a room where the important one gets skipped.
- **Do not narrate your own work into the room.** Report to whoever asked you,
  not to everybody.

## When something fails

The room is best-effort. If a chat command fails — daemon not running, no
identity token — say so once and get on with the work you were given; do not
retry in a loop and do not treat it as a blocker unless someone is waiting on
a reply.
