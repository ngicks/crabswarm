# issues-mermaid-lint

A Stop hook that refuses a turn while an open beads issue holds a mermaid fence
`mermaid-lint` rejects, packaged for
[apm](https://github.com/microsoft/apm).

The hook runs `crabswarm hook exec 'crabswarm issues lint'`. `issues lint`
sweeps the beads database's open issues and validates every `` ```mermaid ``
fence in description, design, acceptance criteria, notes and comments, printing
one line per refused diagram —
`<issue-id> <field>[#<comment-n>]:<line>:<col>: <message>` — and exiting 1.
`hook exec` gets no output template here, which selects its built-in behavior:
a non-zero exit blocks the event with the captured output as the reason. So the
findings reach the agent as the block reason, and the turn ends once every fence
in the open issues lints clean.

The package is one JSON file — no shell scripts, no `jq`, nothing to copy
alongside the wiring. It assumes `crabswarm`, `bd` and `mermaid-lint` are on
`PATH`. Both `bd` and `mermaid-lint` run in the hook's working directory, so the
repository's own mermaid-lint configuration governs its issue text.

## Install

Add it to the consuming project's `apm.yml`:

```yaml
dependencies:
  apm:
    - git: github.com/ngicks/crabswarm
      path: apm-package/issues-mermaid-lint
```

then

```console
apm install
```

`apm` compiles the package per target: the hook merges into
`.claude/settings.json` (Claude Code) and `.codex/hooks.json` (Codex) with the
command string copied through byte for byte. Installing twice changes nothing.

## Layout

```
apm-package/issues-mermaid-lint/
├── apm.yml                  package metadata (targets: claude, codex)
└── .apm/hooks/hook.json     the Stop hook entry, one file for every target
```

The file's stem carries no target token, so `apm` hands the same event to both
harnesses. `Stop` is common ground: each announces it, each feeds the hook the
same snake_case envelope on stdin and reads the same `decision`/`reason` back,
so the command is shared verbatim. The entry carries a 120 second timeout,
which covers a sweep over a backlog whose issues mostly hold no fence.

## Known limitation

The command template does not read `stop_hook_active`, so a diagram the agent
cannot repair re-blocks every turn rather than blocking once.
