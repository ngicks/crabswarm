package chat

import "time"

// TargetKind is who a message was written to.
type TargetKind string

const (
	// TargetNone is a board post: it is in the room, addressed to nobody, and
	// is nobody's unread.
	TargetNone TargetKind = "none"
	// TargetEveryone addresses the whole room. It names no role, so it is
	// unread for every role of the room past its position, the sender excepted,
	// including roles that first attend afterwards.
	TargetEveryone TargetKind = "everyone"
	// TargetRoles addresses the listed roles and nobody else.
	TargetRoles TargetKind = "roles"
)

// Target is who a message is for.
type Target struct {
	Kind TargetKind
	// Roles are the addressed roles, resolved to team and name. It is empty
	// for every kind but [TargetRoles].
	Roles []Sender
}

// Message is one row of a room's conversation.
type Message struct {
	// Id is a UUID v7, unique across every room and every database.
	Id string
	// Room is the room the message was said in.
	Room string
	// Seq is the message's place in its room: dense, increasing, per room.
	// Read positions and the Since/Until bounds are seqs.
	Seq int64
	// From is who said it, as resolved at send time.
	From Sender
	// Target is who it was written to.
	Target Target
	// Text is the message body.
	Text string
	// SentAt is when the daemon accepted the message, in UTC.
	SentAt time.Time
	// MentionedYou reports that this message is one the reading role had not
	// been shown: it targets the role or everyone, the role did not send it,
	// and it lay past the role's position before this read. It is always false
	// on a room read, which reads on nobody's behalf.
	MentionedYou bool
}

// Sent is what one [Store.Send] produced.
type Sent struct {
	// Message is the message as it was recorded.
	Message Message
	// Mentioned is every role the target resolved to, empty for a board post
	// and for everyone, which names no role in particular. A role somebody
	// attends under carries that member's live attendance; a role nobody
	// attends under carries only its room, team and name.
	Mentioned []Member
	// Absent is the subset of Mentioned nobody is attending under. The mention
	// waits for them at their read position.
	Absent []Member
}

// ReadCursor is where a read starts.
type ReadCursor string

const (
	// CursorUnread starts past the reading role's position and keeps only what
	// the role has not been shown. Forward only, and for a role read only.
	CursorUnread ReadCursor = "unread"
	// CursorHead starts at the room's oldest retained message.
	CursorHead ReadCursor = "head"
	// CursorTail starts at the room's newest message and counts backward.
	CursorTail ReadCursor = "tail"
)

// defaultReadRange is how many messages a read takes when it asks for no
// particular number: a screenful, in the cursor's own direction.
const defaultReadRange = 10

// ReadFilter is the one read shape, shared by [Store.Read] and
// [Store.ReadRoom].
type ReadFilter struct {
	// Cursor is where the read starts. Empty means [CursorUnread] for a role
	// read and [CursorTail] for a room read, which has no position to be
	// unread of.
	Cursor ReadCursor
	// Range is how many messages to take from the cursor, forward when
	// positive and backward when negative. Zero means ten in the cursor's own
	// direction. A range running against the cursor is [ErrInvalidArgument]:
	// nothing lies before the head or after the tail, and unread counts
	// forward only.
	Range int
	// To narrows the set to one written target. Nil keeps every target, board
	// posts included.
	To *Target
	// Since and Until bound the set by seq within the room, both exclusive.
	// Zero means unbounded on that side.
	Since, Until int64
}
