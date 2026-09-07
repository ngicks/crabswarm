package issues

import (
	"encoding/json"
	"time"
)

// Status is an issue's stored status.
type Status string

// The statuses bd stores. Anything bd adds later decodes as itself.
const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusBlocked    Status = "blocked"
	StatusDeferred   Status = "deferred"
	StatusClosed     Status = "closed"
)

// Summary is one record of `bd list --json`: an issue without its comments.
// bd list reports the long text fields, the metadata object and the issue's
// outgoing edges too, so a sweep over the text and the graph of a backlog
// runs off a listing and falls back to [Client.Get] only for comments. bd
// omits empty fields from its JSON, so every field here decodes as its zero
// value when the issue does not carry it.
type Summary struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   Status `json:"status"`
	Priority int    `json:"priority"`
	// Type is bd's issue_type: task, epic, bug and the rest.
	Type     string `json:"issue_type"`
	Assignee string `json:"assignee"`
	Owner    string `json:"owner"`
	// ParentID is the issue this one is a child of, empty at top level.
	ParentID string   `json:"parent"`
	Labels   []string `json:"labels"`
	// Metadata is bd's free-form per-issue JSON object, left undecoded
	// because its shape is a convention between whoever writes it and
	// whoever reads it, not something bd defines.
	Metadata json.RawMessage `json:"metadata"`

	Description        string `json:"description"`
	Design             string `json:"design"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	Notes              string `json:"notes"`

	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
	UpdatedAt time.Time `json:"updated_at"`
	ClosedAt  time.Time `json:"closed_at"`

	DependencyCount int `json:"dependency_count"`
	DependentCount  int `json:"dependent_count"`
	CommentCount    int `json:"comment_count"`

	// Dependencies is the outgoing edges bd list reports: what the issue
	// depends on, whose child it is, what it was discovered from.
	Dependencies []Edge `json:"dependencies"`
}

// Issue is a full issue as `bd show --include-comments` reports it: a
// [Summary] plus what only bd show returns.
type Issue struct {
	Summary

	// CloseReason is the conclusion recorded when the issue was closed.
	CloseReason string `json:"close_reason"`

	Comments []Comment `json:"comments"`
	// Dependencies shadows [Summary.Dependencies]: bd show answers the same
	// JSON name with whole issue records rather than the edge records bd list
	// reports, and the shallower field is the one encoding/json fills, so an
	// Issue carries the show shape and the embedded Summary's slice stays nil.
	Dependencies []Dependency `json:"dependencies"`
}

// Comment is one comment on an issue.
type Comment struct {
	ID        string    `json:"id"`
	IssueID   string    `json:"issue_id"`
	Author    string    `json:"author"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// Dependency is an issue another issue depends on. bd reports it as a whole
// issue record carrying the kind of the edge, so the summary of the target
// comes along with the link.
type Dependency struct {
	Summary

	// DependencyType is the edge kind: blocks, parent-child,
	// discovered-from, related.
	DependencyType string `json:"dependency_type"`
}

// Edge is one dependency link between two issues, the shape bd list reports
// under an issue's dependencies. bd stores every link in one direction, so
// From is always the side that carries the link — the dependent, the child,
// the discoverer — and To the side it points at.
type Edge struct {
	// FromID is the issue the link belongs to: the one that depends on
	// ToID, is a child of it, or was discovered from it.
	FromID string `json:"issue_id"`
	// ToID is the blocker, the parent, or the origin.
	ToID string `json:"depends_on_id"`
	// Type is the edge kind: blocks, parent-child, discovered-from,
	// related.
	Type string `json:"type"`
}
