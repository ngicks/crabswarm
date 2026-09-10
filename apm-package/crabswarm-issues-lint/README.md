# crabswarm-issues-lint

Guards the mermaid diagrams written into a beads backlog, packaged for
[apm](https://github.com/microsoft/apm): a `Stop` hook runs
`crabswarm hook exec 'crabswarm issues lint'`, so a turn that left a broken
mermaid fence in an open issue is blocked until the issue text is fixed.
The [`crabswarm-issues-lint`](.apm/skills/crabswarm-issues-lint/SKILL.md)
skill tells the agent how to read a finding and repair the issue.

Everything here assumes `crabswarm`, `bd` and `mermaid-lint` are on `PATH`.

## Install

```yaml
dependencies:
  apm:
    - git: github.com/ngicks/crabswarm
      path: apm-package/crabswarm-issues-lint
```

then `apm install`, or `apm install -g` for every session on the host.

## Layout

```
apm-package/crabswarm-issues-lint/
├── apm.yml
└── .apm/
    ├── hooks/codex-hooks.json                Codex: merged into hooks.json by apm
    └── skills/crabswarm-issues-lint/         Claude Code: a skills-directory plugin
        ├── SKILL.md
        ├── .claude-plugin/plugin.json
        └── hooks/hooks.json
```

The two hook files declare the same `Stop` entry. `e2e/crabswarm` keeps them
equal.

### Claude Code loads the skill directory as a plugin

apm copies a skill directory to `~/.claude/skills/<name>/` (project scope:
`.claude/skills/<name>/`) with every file it holds. Claude Code treats a skill
directory that carries `.claude-plugin/plugin.json` as a plugin named
`<name>@skills-dir`: it reads `hooks/hooks.json` from that directory on every
session and merges nothing into `settings.json`. So on Claude Code apm only
ever adds, replaces and removes files, and a hook this package stops
declaring disappears with the file. A project-scope copy loads only after the
workspace is trusted; a user-scope copy loads in every session.

apm needs a `SKILL.md` at the directory root to deploy the directory at all,
which is why a hook-only package carries a skill.

### Codex still gets a merged hooks file

Codex plugins carry no hooks, so `.apm/hooks/codex-hooks.json` stays. Its
`codex-` stem routes it to Codex alone. apm calls stem routing deprecated in
favor of `targets:` on the consuming dependency, but that setting would
restrict the skill too, so the stem is the right tool here. If a future apm
stops honoring the stem, Claude Code runs the hook twice: once from the plugin
and once from `settings.json`.

apm's merge never removes an entry for an event a package stopped declaring,
so an upgrade from a version that merged this hook into `settings.json` leaves
that entry in place beside the plugin's copy. Delete every entry running
`crabswarm issues lint` from `.claude/settings.json`, `~/.claude/settings.json`
and the `apm-hooks.json` beside them once after upgrading.

## What the hook does

`crabswarm issues lint` runs `bd` in the working directory and hands every
mermaid fence in the open issues to mermaid-lint. It prints one line per refused
diagram and exits 1 when anything was refused. `crabswarm hook exec` with no
output template turns a non-zero exit into a blocked turn whose reason is the
captured output, so the agent sees the findings and can fix the issue text.
