---
name: crabswarm-issues-lint
description: Fix a mermaid diagram in a beads issue that `crabswarm issues lint` refused. Use when a Stop hook blocks with a `<issue-id> <field>:<line>:<col>: <message>` line, or before writing a mermaid fence into an issue's description, design, acceptance criteria, notes or comments.
---

# crabswarm issues lint

Every mermaid fence written into the beads backlog is checked by mermaid-lint
at the end of a turn. A refused diagram blocks the turn until the
issue text is fixed.

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
3. Re-check before ending the turn:

```console
crabswarm issues lint
```

Exit 0 with no output means every open issue is clean.

## Options

- `--all` lints closed issues too.
- `--limit N` lints the N most recently updated issues in any status.
- `--json` prints the findings as a JSON array.
- `-C DIR` runs `bd` in DIR and applies that repository's mermaid-lint
  configuration.
