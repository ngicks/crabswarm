-- room_log is the room's conversation as it was said: one row per send or
-- broadcast, appended beside the inbox rows that deliver it. A broadcast leaves
-- to_name and to_team empty, since it addresses the room rather than a member.
--
-- No foreign key to members, unlike messages: history has to outlive the people
-- who wrote it, and an ON DELETE CASCADE would erase a room's transcript the
-- moment its author left.
CREATE TABLE IF NOT EXISTS room_log (
	id        INTEGER PRIMARY KEY AUTOINCREMENT,
	room      TEXT NOT NULL,
	from_name TEXT NOT NULL,
	from_team TEXT NOT NULL,
	to_name   TEXT NOT NULL DEFAULT '',
	to_team   TEXT NOT NULL DEFAULT '',
	text      TEXT NOT NULL,
	-- RFC3339Nano in UTC like messages.sent_at. Display only: rows are ordered
	-- by id, which no clock skew can reorder.
	sent_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS room_log_room ON room_log (room, id);
