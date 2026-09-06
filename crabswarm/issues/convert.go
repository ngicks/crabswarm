package issues

import (
	"bytes"
	"encoding/json"
	"slices"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	issuesv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/issues/v1"
)

// emptyMetadata is what the API carries for an issue that records none, so a
// client can parse metadata_json unconditionally.
const emptyMetadata = "{}"

// parentChildDependency is bd's edge kind for the parent link. The API omits
// those rows from an issue's dependencies: parent_id and children carry the
// same information in a shape the UI can draw.
const parentChildDependency = "parent-child"

// childCount is one issue's children as a listing tallies them.
type childCount struct {
	total  int
	closed int
}

func sourceToProto(s Source) *issuesv1.Source {
	return &issuesv1.Source{
		Id:        s.ID,
		BeadsPath: s.BeadsPath,
		Prefix:    s.Prefix,
		Dir:       s.Dir,
	}
}

// summaryToProto renders one listing record. Neither the metadata nor the
// child counts come from the record itself — bd's listing carries no child
// count and [Summary] holds no metadata — so both are passed in by the
// caller that gathered them.
func summaryToProto(
	s Summary,
	metadata json.RawMessage,
	children childCount,
) *issuesv1.IssueSummary {
	return &issuesv1.IssueSummary{
		Id:               s.ID,
		Title:            s.Title,
		IssueType:        s.Type,
		Status:           statusToProto(s.Status),
		Priority:         int32(s.Priority),
		Labels:           slices.Clone(s.Labels),
		ParentId:         s.ParentID,
		CommentCount:     int32(s.CommentCount),
		ChildCount:       int32(children.total),
		ChildClosedCount: int32(children.closed),
		CreatedAt:        timestampOrNil(s.CreatedAt),
		UpdatedAt:        timestampOrNil(s.UpdatedAt),
		MetadataJson:     metadataJSON(metadata),
	}
}

// statusToProto maps a stored status onto the API enum. bd's "deferred" has
// no API counterpart and lands on unspecified, as does any status a later bd
// adds.
func statusToProto(s Status) issuesv1.IssueStatus {
	switch s {
	case StatusOpen:
		return issuesv1.IssueStatus_ISSUE_STATUS_OPEN
	case StatusInProgress:
		return issuesv1.IssueStatus_ISSUE_STATUS_IN_PROGRESS
	case StatusBlocked:
		return issuesv1.IssueStatus_ISSUE_STATUS_BLOCKED
	case StatusClosed:
		return issuesv1.IssueStatus_ISSUE_STATUS_CLOSED
	default:
		return issuesv1.IssueStatus_ISSUE_STATUS_UNSPECIFIED
	}
}

// statusesFromProto maps a request's status filter onto stored statuses.
// Unspecified entries are dropped, so a filter of nothing but unspecified
// leaves bd's own default in place.
func statusesFromProto(in []issuesv1.IssueStatus) []Status {
	var out []Status
	for _, s := range in {
		switch s {
		case issuesv1.IssueStatus_ISSUE_STATUS_OPEN:
			out = append(out, StatusOpen)
		case issuesv1.IssueStatus_ISSUE_STATUS_IN_PROGRESS:
			out = append(out, StatusInProgress)
		case issuesv1.IssueStatus_ISSUE_STATUS_BLOCKED:
			out = append(out, StatusBlocked)
		case issuesv1.IssueStatus_ISSUE_STATUS_CLOSED:
			out = append(out, StatusClosed)
		case issuesv1.IssueStatus_ISSUE_STATUS_UNSPECIFIED:
		}
	}
	return out
}

// metadataJSON renders bd's free-form metadata as the compact JSON object
// the API carries. Anything absent, null, or not an object becomes "{}": the
// field is a convention between whoever writes it and whoever reads it, and
// a client that cannot even parse it is worse off than one seeing nothing.
func metadataJSON(raw json.RawMessage) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return emptyMetadata
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, trimmed); err != nil {
		return emptyMetadata
	}
	if buf.Len() == 0 || buf.Bytes()[0] != '{' {
		return emptyMetadata
	}
	return buf.String()
}

// timestampOrNil leaves a field bd omitted — an unclosed issue's close time,
// say — absent instead of reporting the zero instant as a real date.
func timestampOrNil(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}
