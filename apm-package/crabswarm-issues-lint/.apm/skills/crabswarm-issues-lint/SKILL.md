---
name: crabswarm-issues-lint
description: Lint the mermaid diagrams in a beads backlog with `crabswarm issues lint`. Use when a commit is refused with a `<issue-id> <field>:<line>:<col>: <message>` line, before writing a mermaid fence into an issue's description, design, acceptance criteria, notes or comments, or to scaffold the hk-managed git pre-commit hook that runs the lint.
---

# crabswarm issues lint

Every mermaid fence written into the beads backlog is checked by mermaid-lint
before a commit. A refused diagram fails the commit until the issue text is
fixed.

## Read the finding

One line per refused diagram:

```
crabswarm-abc design:14:7: Parse error on line 3 ...
crabswarm-abc notes#2:3:1: ...
```

The first token is the issue id, the second the field the fence sits in.
`#<n>` after a field names the n-th comment. Line and column count from the
first line of that field's text, not from the fence.

## Fix it

1. Show the text: `bd show <issue-id>`.
2. Replace the field that holds the fence with the corrected text:
   `bd update <issue-id> --description ...`, `--design ...`,
   `--acceptance ...` or `--notes ...`, or `--body-file FILE` /
   `--design-file FILE` for long text. `bd` cannot edit or delete a comment,
   so a finding that names one stays until the issue is closed. Keep mermaid
   out of comments.
3. Re-check with plain `crabswarm issues lint` before committing again. The
   plain command takes no `--changed` and always lints:

```console
crabswarm issues lint
```

Exit 0 with no output means every open issue is clean.

## Options

- `--changed` skips the lint when the beads commit and the scope (`--all`,
  `--limit`) match the last lint that found nothing. A lint that finds nothing
  records the beads commit. A lint with findings records nothing.
- `--all` lints closed issues too.
- `--limit N` lints the N most recently updated issues in any status.
- `--json` prints the findings as a JSON array.
- `-C DIR` runs `bd` in DIR and applies that repository's mermaid-lint
  configuration.

## Scaffold the pre-commit hook

Wire the lint into a repository through [hk](https://hk.jdx.dev). The hook
runs one step, `crabswarm-issues-lint`, on every commit. The step takes no
staged files, because the issue text lives in the beads database. The step
runs `crabswarm issues lint --changed`. A commit made while the backlog stands
still skips the lint.

Requirements: `hk` 2.x, `crabswarm`, `bd` and `mermaid-lint` on `PATH`, and a
git repository.

1. Run `scripts/hk-init.sh` from this skill's directory, with the target
   repository as the working directory.
   - It copies `reference/crabswarm-issues-lint.hk.pkl` to
     `.hk/crabswarm-issues-lint.pkl`, pinned to the hk version the project
     already uses.
   - It creates `hk.pkl` when the repository has none.
   - It validates the config and runs `hk install`.
2. If the script exits 1 with wiring instructions, `hk.pkl` already exists.
   - Add the printed `import` line after its `amends` line.
   - Add the printed `...crabswarm_issues_lint.steps` line into the `steps` of
     `hooks["pre-commit"]`; create that hook when it is missing.
   - Rerun the script; it exits 0 once wired.
3. Commit `hk.pkl` and `.hk/crabswarm-issues-lint.pkl`.
   - A worktree without them runs no hooks.

Do not edit `.hk/crabswarm-issues-lint.pkl`; rerun the script to update it.
`hk run pre-commit` runs the hook without committing.
