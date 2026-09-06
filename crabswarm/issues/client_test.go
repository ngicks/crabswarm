package issues

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestClientList(t *testing.T) {
	invocations := installFakeBd(t)
	dir := t.TempDir()

	got, err := NewClient(dir).List(t.Context(), ListFilter{})
	assert.NilError(t, err)
	assert.Equal(t, len(got), 3)

	// A bare filter still pins the limit: bd's own default would cap the
	// result at 50.
	inv := invocations()
	assert.Equal(t, len(inv), 1)
	assert.Equal(t, inv[0].args, "list --json --limit 0")
	assert.Equal(t, inv[0].dir, dir)
	// Only `bd where` runs with the envelope.
	assert.Equal(t, inv[0].envelope, "")

	open := got[0]
	assert.Equal(t, open.ID, "crabswarm-no2")
	assert.Equal(t, open.Title, "sample-title")
	assert.Equal(t, open.Status, StatusOpen)
	assert.Equal(t, open.Type, "task")
	assert.Equal(t, open.Assignee, "me")
	assert.Equal(t, open.CommentCount, 1)
	assert.Equal(t, open.CreatedAt.Format("2006-01-02T15:04:05Z"), "2026-09-04T12:20:33Z")
	// bd list carries the long text, so a listing is enough to see what an
	// issue says.
	assert.Equal(t, open.Description, "woo\n\n## Context\nwhoooaaa")
	assert.Equal(t, open.Design, "realy nice")
	assert.Equal(t, open.AcceptanceCriteria, "weeee")
	// Omitted fields decode as empty.
	assert.Equal(t, open.Notes, "")
	assert.Equal(t, len(open.Labels), 0)
	assert.Assert(t, open.ClosedAt.IsZero())
	assert.Equal(t, open.ParentID, "")

	assert.DeepEqual(t, got[1].Labels, []string{"admin", "chat", "proto", "tui"})

	closed := got[2]
	assert.Equal(t, closed.ID, "crabswarm-125")
	assert.Equal(t, closed.Status, StatusClosed)
	assert.Assert(t, !closed.ClosedAt.IsZero())
}

func TestClientListFilters(t *testing.T) {
	invocations := installFakeBd(t)

	_, err := NewClient(t.TempDir()).List(t.Context(), ListFilter{
		Statuses:      []Status{StatusOpen, StatusInProgress},
		Labels:        []string{"chat", "tui"},
		ParentID:      "crabswarm-125",
		Limit:         5,
		SortByUpdated: true,
	})
	assert.NilError(t, err)

	inv := invocations()
	assert.Equal(t, len(inv), 1)
	// Statuses go in comma-separated: repeating bd's --status overwrites
	// the previous value, while --label accumulates.
	assert.Equal(
		t,
		inv[0].args,
		"list --json --status open,in_progress --label chat --label tui"+
			" --parent crabswarm-125 --limit 5 --sort updated",
	)
}

func TestClientChildren(t *testing.T) {
	invocations := installFakeBd(t)

	got, err := NewClient(t.TempDir()).Children(t.Context(), "scratch-2o5")
	assert.NilError(t, err)

	inv := invocations()
	assert.Equal(t, len(inv), 1)
	assert.Equal(t, inv[0].args, "list --json --parent scratch-2o5 --limit 0")

	assert.Equal(t, len(got), 1)
	assert.Equal(t, got[0].ID, "scratch-uoj")
	assert.Equal(t, got[0].ParentID, "scratch-2o5")
	assert.DeepEqual(t, got[0].Labels, []string{"alpha", "beta"})
}

func TestClientGet(t *testing.T) {
	invocations := installFakeBd(t)

	issue, err := NewClient(t.TempDir()).Get(t.Context(), "scratch-uoj")
	assert.NilError(t, err)

	inv := invocations()
	assert.Equal(t, len(inv), 1)
	// The ID goes in as --id= so that one starting with "-" cannot be read
	// as a bd flag.
	assert.Equal(t, inv[0].args, "show --id=scratch-uoj --json --include-comments")

	assert.Equal(t, issue.ID, "scratch-uoj")
	assert.Equal(t, issue.Title, "child task")
	assert.Equal(t, issue.Description, "the description")
	assert.Equal(t, issue.Design, "a design")
	assert.Equal(t, issue.AcceptanceCriteria, "some criteria")
	assert.Equal(t, issue.Notes, "a note")
	assert.Equal(t, issue.CloseReason, "")
	assert.Equal(t, issue.ParentID, "scratch-2o5")
	assert.DeepEqual(t, issue.Labels, []string{"alpha", "beta"})

	// Metadata stays raw JSON; its keys are a caller convention.
	var meta map[string]any
	assert.NilError(t, json.Unmarshal(issue.Metadata, &meta))
	assert.Equal(t, meta["plan"], "doc/plan/x")

	assert.Equal(t, len(issue.Comments), 1)
	assert.Equal(t, issue.Comments[0].Author, "ngicks")
	assert.Equal(t, issue.Comments[0].Text, "first comment")
	assert.Equal(t, issue.Comments[0].IssueID, "scratch-uoj")
	assert.Assert(t, !issue.Comments[0].CreatedAt.IsZero())

	// A dependency carries the whole target issue plus the edge kind.
	assert.Equal(t, len(issue.Dependencies), 1)
	dep := issue.Dependencies[0]
	assert.Equal(t, dep.ID, "scratch-2o5")
	assert.Equal(t, dep.Title, "parent epic")
	assert.Equal(t, dep.Type, "epic")
	assert.Equal(t, dep.DependencyType, "parent-child")
}

func TestClientGetMissing(t *testing.T) {
	installFakeBd(t)

	_, err := NewClient(t.TempDir()).Get(t.Context(), "scratch-nope")
	assert.Assert(t, err != nil)
	// Both bd's JSON report and its stderr line reach the caller.
	assert.Assert(
		t,
		strings.Contains(err.Error(), "no issues found matching the provided IDs"),
		"got %v",
		err,
	)
	assert.Assert(
		t,
		strings.Contains(err.Error(), `no issue found matching "scratch-nope"`),
		"got %v",
		err,
	)
}

func TestClientGetFlagLikeID(t *testing.T) {
	invocations := installFakeBd(t)

	// bd's own --directory shorthand: passed positionally it would change
	// bd's working directory instead of naming an issue.
	_, err := NewClient(t.TempDir()).Get(t.Context(), "-C")
	assert.Assert(t, err != nil)

	inv := invocations()
	assert.Equal(t, len(inv), 1)
	assert.Equal(t, inv[0].args, "show --id=-C --json --include-comments")
}

func TestClientWithEnvAndBinary(t *testing.T) {
	invocations := installFakeBd(t)

	// Reach the fake by path rather than by PATH lookup, under a name bd
	// would never be found as.
	dir := t.TempDir()
	bin := filepath.Join(dir, "not-bd")
	assert.NilError(t, os.WriteFile(bin, []byte(fakeBdScript), 0o755))

	_, err := NewClient(t.TempDir(), WithBinary(bin), WithEnv("FAKE_BD_EXTRA=carried")).
		List(t.Context(), ListFilter{})
	assert.NilError(t, err)

	inv := invocations()
	assert.Equal(t, len(inv), 1)
	assert.Equal(t, inv[0].extra, "carried")
}
