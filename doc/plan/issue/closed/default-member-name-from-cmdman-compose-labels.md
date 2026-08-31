# Default member name from cmdman compose labels (2026-08-31)

Member names must be aliased from cmdman-compose's command name, not
from the token. Today the default is `agent-<first-8-hex-of-token>`
(`defaultName` in `crabswarm/chat/service.go`), and the resolver
decodes only `dir` and the labels map, using just
`cmdman.compose.project` (`crabswarm/chat/resolver/cmdman.go`).

Follow-up: read `cmdman.compose.command` and
`cmdman.compose.scale-index` from the already-decoded labels (i.e. what
`cmdman inspect $ID --format '{{index .Config.Labels
"cmdman.compose.command"}}'` returns), carry a `Name` on
`resolver.TeamInfo` (`crabswarm/chat/resolver/resolver.go`), and have
`Service.Join` prefer it (e.g. `<command>-<scale-index>`) over the
token-derived fallback when `--name` is absent.
