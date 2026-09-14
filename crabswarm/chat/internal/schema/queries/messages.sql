-- name: InsertMessage :exec
INSERT INTO messages (id, room, seq, from_name, from_team, target_kind, text, sent_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertMessageMention :exec
INSERT INTO message_mention (message_id, team, name) VALUES (?, ?, ?);

-- PruneRoomMessages drops everything at or below a seq; the mentions of those
-- messages go with them through ON DELETE CASCADE. Pruning on send keeps the
-- table bounded without a background job.
-- name: PruneRoomMessages :exec
DELETE FROM messages WHERE room = ? AND seq <= ?;

-- MessagesAsc is the read window taken forward from its lower bound, which is
-- how the head and unread cursors read.
--
-- Every narrowing but the seq bounds is optional, switched off by the empty
-- string: a team and a name are never empty, so no real role can be mistaken
-- for "unfiltered". role_team and role_name switch on the unread predicate,
-- which keeps what the role has not been shown: not sent by it, and either
-- addressed to everyone or naming it. to_team and to_name keep only messages
-- naming one particular role; several roles are read one query per role and
-- merged, since the first n of the union are among the first n of one of them.
-- name: MessagesAsc :many
SELECT id, seq, from_name, from_team, target_kind, text, sent_at FROM messages m
WHERE m.room = sqlc.arg(room)
	AND m.seq > sqlc.arg(after)
	AND (m.seq < sqlc.arg(before) OR sqlc.arg(before) = 0)
	AND (m.target_kind = sqlc.arg(kind) OR sqlc.arg(kind) = '')
	AND (
		EXISTS (
			SELECT 1 FROM message_mention mm
			WHERE mm.message_id = m.id
				AND mm.team = sqlc.arg(to_team)
				AND mm.name = sqlc.arg(to_name)
		)
		OR sqlc.arg(to_team) = ''
	)
	AND (
		(
			(m.from_team <> sqlc.arg(role_team) OR m.from_name <> sqlc.arg(role_name))
			AND (
				m.target_kind = 'everyone'
				OR EXISTS (
					SELECT 1 FROM message_mention mine
					WHERE mine.message_id = m.id
						AND mine.team = sqlc.arg(role_team)
						AND mine.name = sqlc.arg(role_name)
				)
			)
		)
		OR sqlc.arg(role_team) = ''
	)
ORDER BY m.seq ASC
LIMIT sqlc.arg(lim);

-- MessagesDesc is MessagesAsc taken backward from its upper bound, which is how
-- the tail cursor reads; the caller reverses the rows, since a read always hands
-- back the conversation in the order it happened.
-- name: MessagesDesc :many
SELECT id, seq, from_name, from_team, target_kind, text, sent_at FROM messages m
WHERE m.room = sqlc.arg(room)
	AND m.seq > sqlc.arg(after)
	AND (m.seq < sqlc.arg(before) OR sqlc.arg(before) = 0)
	AND (m.target_kind = sqlc.arg(kind) OR sqlc.arg(kind) = '')
	AND (
		EXISTS (
			SELECT 1 FROM message_mention mm
			WHERE mm.message_id = m.id
				AND mm.team = sqlc.arg(to_team)
				AND mm.name = sqlc.arg(to_name)
		)
		OR sqlc.arg(to_team) = ''
	)
	AND (
		(
			(m.from_team <> sqlc.arg(role_team) OR m.from_name <> sqlc.arg(role_name))
			AND (
				m.target_kind = 'everyone'
				OR EXISTS (
					SELECT 1 FROM message_mention mine
					WHERE mine.message_id = m.id
						AND mine.team = sqlc.arg(role_team)
						AND mine.name = sqlc.arg(role_name)
				)
			)
		)
		OR sqlc.arg(role_team) = ''
	)
ORDER BY m.seq DESC
LIMIT sqlc.arg(lim);

-- CountUnreadMentions is how much a role still has waiting past a position: the
-- same predicate the unread cursor reads by, with no window and no filter.
-- name: CountUnreadMentions :one
SELECT COUNT(*) FROM messages m
WHERE m.room = sqlc.arg(room)
	AND m.seq > sqlc.arg(after)
	AND (m.from_team <> sqlc.arg(role_team) OR m.from_name <> sqlc.arg(role_name))
	AND (
		m.target_kind = 'everyone'
		OR EXISTS (
			SELECT 1 FROM message_mention mine
			WHERE mine.message_id = m.id
				AND mine.team = sqlc.arg(role_team)
				AND mine.name = sqlc.arg(role_name)
		)
	);

-- MentionsInSeqRange reads back the roles the messages of one window name, so a
-- read can hand every message the target it was written with. It is bounded by
-- the window's own seqs rather than by a list of ids.
-- name: MentionsInSeqRange :many
SELECT mm.message_id, mm.team, mm.name FROM message_mention mm
JOIN messages m ON m.id = mm.message_id
WHERE m.room = sqlc.arg(room)
	AND m.seq >= sqlc.arg(from_seq)
	AND m.seq <= sqlc.arg(to_seq)
ORDER BY mm.team, mm.name;
