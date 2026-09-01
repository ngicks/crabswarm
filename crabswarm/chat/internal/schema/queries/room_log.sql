-- name: InsertRoomLog :exec
INSERT INTO room_log (room, from_name, from_team, to_name, to_team, text, sent_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- The tail is read newest first and reversed by the caller: taking the last N
-- rows the other way round would mean scanning the whole room.
-- name: RoomLogTail :many
SELECT id, from_name, from_team, to_name, to_team, text, sent_at FROM room_log
WHERE room = ? ORDER BY id DESC LIMIT ?;

-- Reading forward from a cursor needs no reversal: the rows wanted are the
-- oldest of what comes after it, which is the end the index already starts at.
-- name: RoomLogSince :many
SELECT id, from_name, from_team, to_name, to_team, text, sent_at FROM room_log
WHERE room = ? AND id > ? ORDER BY id ASC LIMIT ?;

-- name: PruneRoomLog :exec
DELETE FROM room_log
WHERE room_log.room = sqlc.arg(room)
	AND room_log.id NOT IN (
		SELECT recent.id FROM room_log AS recent
		WHERE recent.room = sqlc.arg(room)
		ORDER BY recent.id DESC LIMIT sqlc.arg(keep)
	);
