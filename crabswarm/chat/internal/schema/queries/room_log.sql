-- name: InsertRoomLog :exec
INSERT INTO room_log (room, from_name, from_team, to_name, to_team, text, sent_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- The tail is read newest first and reversed by the caller: taking the last N
-- rows the other way round would mean scanning the whole room.
-- name: RoomLogTail :many
SELECT from_name, from_team, to_name, to_team, text, sent_at FROM room_log
WHERE room = ? ORDER BY id DESC LIMIT ?;

-- name: PruneRoomLog :exec
DELETE FROM room_log
WHERE room_log.room = sqlc.arg(room)
	AND room_log.id NOT IN (
		SELECT recent.id FROM room_log AS recent
		WHERE recent.room = sqlc.arg(room)
		ORDER BY recent.id DESC LIMIT sqlc.arg(keep)
	);
