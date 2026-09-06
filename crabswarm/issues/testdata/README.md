# testdata

Recorded `bd` output the fake bd in `fakebd_test.go` replays. Every fixture is
verbatim bd output unless a note below says otherwise.

The bd that produced the re-recorded fixtures:

```
bd version 1.2.2 (6c124203e: HEAD@6c124203e771)
```

## Re-recorded from this repository's beads database

Recorded 2026-09-07 by running bd in the repository worktree.

| File | Command |
| --- | --- |
| `list.json` | `bd list --json --status open,in_progress,blocked,deferred,closed --limit 0` |
| `list_changed.json` | a copy of `list.json`, then edited by hand |

`list_changed.json` is what a poller sees after the backlog moved. Exactly one
issue differs: its `updated_at` was moved forward and its title changed, so the
diff between the two listings names that one issue and nothing else.

## Kept from earlier recordings

These were not re-recorded, so the commands below are the ones each fixture
answers rather than commands run against the database this repository carries.

`show.json` and `list_children.json` carry issues under the `scratch` prefix.
This repository's backlog never held them. They record a child issue with
metadata, notes, labels and one comment, plus the epic it hangs under.

| File | Command it answers |
| --- | --- |
| `show.json` | `bd show --id=scratch-uoj --json --include-comments` |
| `list_children.json` | `bd list --json --status open,in_progress,blocked,deferred,closed --parent scratch-2o5 --limit 0` |
| `show_not_found.json` | `bd show` for an id no issue matches |
| `where.json` | `bd where --json` with `BD_JSON_ENVELOPE=1` |
| `where_no_beads.json` | the same `bd where` in a directory no beads workspace governs |

`show_not_found.json` and `where_no_beads.json` are failures. bd prints its
error report on stdout and exits non-zero. The report on stdout is what makes a
missing issue or a missing workspace machine-readable.
