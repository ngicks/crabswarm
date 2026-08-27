CREATE TABLE IF NOT EXISTS members (
	token TEXT PRIMARY KEY,
	name  TEXT NOT NULL,
	team  TEXT NOT NULL,
	room  TEXT NOT NULL,
	kind  TEXT NOT NULL,
	state TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS members_room_team_name
	ON members (room, team, name);
CREATE INDEX IF NOT EXISTS members_room ON members (room);

CREATE TABLE IF NOT EXISTS messages (
	id        INTEGER PRIMARY KEY AUTOINCREMENT,
	recipient TEXT NOT NULL REFERENCES members (token) ON DELETE CASCADE,
	from_name TEXT NOT NULL,
	from_team TEXT NOT NULL,
	from_room TEXT NOT NULL,
	text      TEXT NOT NULL,
	sent_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS messages_recipient ON messages (recipient, id);
