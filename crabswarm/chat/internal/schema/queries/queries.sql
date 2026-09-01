-- name: MemberByToken :one
SELECT token, name, team, room, kind, state, state_reported_at FROM members WHERE token = ?;

-- name: MemberByName :one
SELECT token, name, team, room, kind, state, state_reported_at FROM members
WHERE room = ? AND team = ? AND name = ?;

-- name: InsertMember :exec
INSERT INTO members (token, name, team, room, kind, state, state_reported_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: SetMemberState :execrows
UPDATE members SET state = ?, state_reported_at = ? WHERE token = ?;

-- name: SetMemberTeam :exec
UPDATE members SET team = ? WHERE token = ?;

-- name: DeleteMember :exec
DELETE FROM members WHERE token = ?;

-- name: ListRoomMembers :many
SELECT token, name, team, room, kind, state, state_reported_at FROM members
WHERE room = ? ORDER BY team, name;

-- name: ListAllMembers :many
SELECT token, name, team, room, kind, state, state_reported_at FROM members
ORDER BY room, team, name;

-- Bare-name resolution falls back to this when the caller's own team has no
-- such member: every other team of the room is a candidate, and two or more
-- make the address ambiguous.
-- name: MembersByRoomAndName :many
SELECT token, name, team, room, kind, state, state_reported_at FROM members
WHERE room = ? AND name = ? ORDER BY team;

-- name: PendingMessages :many
SELECT from_name, from_team, from_room, text, sent_at FROM messages
WHERE recipient = ? ORDER BY id;

-- name: InsertMessage :exec
INSERT INTO messages (recipient, from_name, from_team, from_room, text, sent_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: DeleteMessages :exec
DELETE FROM messages WHERE recipient = ?;
