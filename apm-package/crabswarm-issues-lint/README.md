# crabswarm-issues-lint

Guards the mermaid diagrams written into a beads backlog, packaged for
[apm](https://github.com/microsoft/apm): an [hk](https://hk.jdx.dev)-managed
git pre-commit hook runs `crabswarm issues lint`, so a commit made while an
open issue holds a broken mermaid fence is refused until the issue text is
fixed. The [`crabswarm-issues-lint`](.apm/skills/crabswarm-issues-lint/SKILL.md)
skill scaffolds the hook and tells the agent how to read a finding and repair
the issue.

Everything here assumes `hk` 2.x, `crabswarm`, `bd` and `mermaid-lint` are on
`PATH`.

## Install

```yaml
dependencies:
  apm:
    - git: github.com/ngicks/crabswarm
      path: apm-package/crabswarm-issues-lint
```

then `apm install`, or `apm install -g` for every session on the host.

Installing the package deploys the skill only. Run the skill's
`scripts/hk-init.sh` in each repository that wants the hook, and commit
`hk.pkl` and `.hk/crabswarm-issues-lint.pkl`.

## Layout

```
apm-package/crabswarm-issues-lint/
├── apm.yml
└── .apm/
    └── skills/crabswarm-issues-lint/
        ├── SKILL.md
        ├── reference/crabswarm-issues-lint.hk.pkl   the pre-commit step
        └── scripts/hk-init.sh                       copies the step into a repository
```

The package wires no harness hook and ships no Claude Code plugin. Git runs
the check, so it holds for every harness and for a person at a plain shell.

## What the hook does

`scripts/hk-init.sh` copies the fragment to `.hk/crabswarm-issues-lint.pkl`
and pins it to the hk version the repository already uses. `hk.pkl` imports
the copy and spreads its `steps` into the pre-commit hook:

```pkl
import ".hk/crabswarm-issues-lint.pkl" as crabswarm_issues_lint

hooks {
  ["pre-commit"] {
    steps {
      ...crabswarm_issues_lint.steps
    }
  }
}
```

The step runs

```
crabswarm issues lint --changed
```

from the worktree root. It declares no `glob`: the issue text lives in the
beads database and no staged file names it, so the step runs on every commit.

`crabswarm issues lint` runs `bd` in that directory and hands every mermaid
fence in the open issues to mermaid-lint. It prints one line per refused
diagram and exits 1 when anything was refused, which fails the commit and
shows the findings.

`--changed` skips the lint when the backlog has not changed since the last
clean lint. bd commits every backlog write under its default
`dolt.auto-commit` policy. The command reads the beads commit with
`bd vc status` and compares it with a record of the last clean lint.

The record lives in `$XDG_STATE_HOME/crabswarm/issues-lint/<source id>`.
`$XDG_STATE_HOME` defaults to `~/.local/state`. The source id is a hash of the
`.beads` path, and every worktree of a repository therefore shares one record.
When the beads commit and the scope (`--all`, `--limit`) match the record, the
command exits 0 and prints nothing. A clean lint records the beads commit. A
lint with findings records nothing. The next commit then lints again and
shows the findings again.

The record covers the backlog only. A change to the mermaid-lint configuration
alone does not trigger a re-lint. Run plain `crabswarm issues lint` after
changing it. The plain command always lints.

Under the `off` or `batch` `dolt.auto-commit` policy, bd leaves a write
uncommitted. The beads commit then stays put. `--changed` skips the lint until
bd commits the write.

A repository no beads database governs has no backlog to lint. There the
command exits 0 without running mermaid-lint.

## Upgrading from the Stop hook

Versions before 0.3.0 wired the lint as a `Stop` hook for Claude Code and
Codex. apm removes the Claude Code plugin files on upgrade. apm's merge never
removes an entry for an event a package stopped declaring, so delete every
entry running `crabswarm issues lint` from Codex's `hooks.json` and the
`apm-hooks.json` beside it once after upgrading.
