-- SeedReadPosition creates the role's position at the room's current last_seq,
-- which is what leaves a role attending for the first time with nothing unread.
-- An existing position is left exactly where it is: attendance comes and goes
-- with a stream, the position is the durable half of a role.
-- name: SeedReadPosition :exec
INSERT INTO read_positions (room, team, name, last_seq)
SELECT r.name, sqlc.arg(team), sqlc.arg(name), r.last_seq FROM rooms r
WHERE r.name = sqlc.arg(room)
ON CONFLICT (room, team, name) DO NOTHING;

-- name: ReadPosition :one
SELECT last_seq FROM read_positions WHERE room = ? AND team = ? AND name = ?;

-- AdvanceReadPosition never moves a position backwards: reading old history
-- shows what the role has already seen, and must not make newer messages unread
-- again.
-- name: AdvanceReadPosition :exec
UPDATE read_positions SET last_seq = sqlc.arg(last_seq)
WHERE room = sqlc.arg(room)
	AND team = sqlc.arg(team)
	AND name = sqlc.arg(name)
	AND last_seq < sqlc.arg(last_seq);

-- ListRoomRoles lists every role of the room: everyone that has ever attended
-- it. A target names one of these or it names nobody.
-- name: ListRoomRoles :many
SELECT team, name FROM read_positions WHERE room = ? ORDER BY team, name;
