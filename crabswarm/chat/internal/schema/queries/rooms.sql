-- EnsureRoom creates the room's counter row the first time anything happens in
-- it: an attend or a send, whichever comes first. That row is what makes the
-- room exist, and nothing removes it but an explicit delete.
-- name: EnsureRoom :exec
INSERT INTO rooms (name, last_seq) VALUES (?, 0) ON CONFLICT (name) DO NOTHING;

-- NextRoomSeq claims the room's next seq. The counter lives in the row rather
-- than being derived from MAX(seq) so that pruning the oldest messages away
-- cannot hand the same seq out a second time.
-- name: NextRoomSeq :one
UPDATE rooms SET last_seq = last_seq + 1 WHERE name = ? RETURNING last_seq;

-- name: RoomLastSeq :one
SELECT last_seq FROM rooms WHERE name = ?;

-- name: ListRoomNames :many
SELECT name FROM rooms ORDER BY name;

-- name: CountRoomMessages :one
SELECT COUNT(*) FROM messages WHERE room = ?;

-- DeleteRoom drops the counter row; the room's messages, their mentions and its
-- read positions follow through ON DELETE CASCADE. No row affected means there
-- was no such room.
-- name: DeleteRoom :execrows
DELETE FROM rooms WHERE name = ?;
