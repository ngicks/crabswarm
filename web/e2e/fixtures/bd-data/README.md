# bd-data

Recorded `bd` output the issues spec runs against. `playwright.config.ts` puts
`../bd-fake/` first on the preview daemon's PATH and points `FAKE_BD_TESTDATA`
at this directory, so the daemon reads its issues from these files and the
suite needs no beads database.

The bd that produced the recordings:

```
bd version 1.2.2 (6c124203e: HEAD@6c124203e771)
```

## Recorded from this repository's beads database

Recorded 2026-09-07 by running bd in the repository worktree. The three files
come from one snapshot: an issue's `show` record and its record in the listing
have to agree, or the page and the counts the spec computes disagree.

| File | Command |
| --- | --- |
| `list.json` | `bd list --json --status open,in_progress,blocked,deferred,closed --limit 0` |
| `show_crabswarm-3hp.json` | `bd show --id=crabswarm-3hp --json --include-comments` |
| `show_crabswarm-3hp.2.json` | `bd show --id=crabswarm-3hp.2 --json --include-comments` |

`crabswarm-3hp` is an open epic labelled `plan`, carrying every text field, a
mermaid diagram in its description, metadata, seven children, comments with the
`Decision:` prefix, and one dependency edge that only the issue at the other end
records. `crabswarm-3hp.2` is a closed child of it, with a close reason and
dependency edges running both ways.

The listing is the whole backlog rather than a hand-picked subset: the daemon
derives the board, the labels, the children, the child counts and every
dependency edge from that one listing, so anything trimmed out of it disappears
from the page. The spec reads its expected counts back out of `list.json` for
the same reason.

## Kept from earlier recordings

| File | Command it answers |
| --- | --- |
| `where.json` | `bd where --json` with `BD_JSON_ENVELOPE=1` |
| `show_not_found.json` | `bd show` for an id no issue matches |

`where.json` is edited to name `/fake/crabswarm/.beads`, a path no database sits
at. A daemon started without the fake bd on its PATH resolves this directory to
the repository's own beads database instead; the spec asserts the fake path
first, so that run fails on the reason rather than on every later assertion.

`show_not_found.json` is a failure. bd prints its error report on stdout and
exits non-zero, and the report on stdout is what makes a missing issue
machine-readable.
