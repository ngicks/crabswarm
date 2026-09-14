-- rooms is one row per room the log knows, carrying the room's sequence
-- counter. seq is per room: two rooms share no numbering.
CREATE TABLE IF NOT EXISTS rooms (
	name     TEXT PRIMARY KEY,
	last_seq INTEGER NOT NULL DEFAULT 0
);

-- messages is the room's conversation as it was said, one row per send.
-- id is a UUID v7, unique across every room and every database; seq is the
-- message's place in its room, dense and increasing, and what read positions
-- and since/until are measured in.
CREATE TABLE IF NOT EXISTS messages (
	id          TEXT PRIMARY KEY,
	room        TEXT NOT NULL REFERENCES rooms (name) ON DELETE CASCADE,
	seq         INTEGER NOT NULL,
	from_name   TEXT NOT NULL,
	from_team   TEXT NOT NULL,
	-- target_kind is one of 'none', 'everyone', 'roles'. The roles are the
	-- message_mention rows.
	target_kind TEXT NOT NULL,
	text        TEXT NOT NULL,
	-- RFC3339Nano in UTC. Display only: rows are ordered by seq.
	sent_at     TEXT NOT NULL,
	UNIQUE (room, seq)
);

-- message_mention is one row per role a 'roles' message names. Pruning a
-- message prunes its mentions.
CREATE TABLE IF NOT EXISTS message_mention (
	message_id TEXT NOT NULL REFERENCES messages (id) ON DELETE CASCADE,
	team       TEXT NOT NULL,
	name       TEXT NOT NULL,
	PRIMARY KEY (message_id, team, name)
);
CREATE INDEX IF NOT EXISTS message_mention_role ON message_mention (team, name, message_id);

-- read_positions is where each role has read up to in its room: the seq of
-- the newest message a read of that role has shown. The row is created when
-- the role first attends, at the room's last_seq of that moment, so "unread"
-- is defined from the first attendance on.
CREATE TABLE IF NOT EXISTS read_positions (
	room     TEXT NOT NULL REFERENCES rooms (name) ON DELETE CASCADE,
	team     TEXT NOT NULL,
	name     TEXT NOT NULL,
	last_seq INTEGER NOT NULL,
	PRIMARY KEY (room, team, name)
);
