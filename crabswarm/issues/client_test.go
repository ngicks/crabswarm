package issues

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"
)

// findSummary returns the listed issue with the given id, failing the test
// when the recording does not carry it.
func findSummary(t *testing.T, listed []Summary, id string) Summary {
	t.Helper()
	for _, sum := range listed {
		if sum.ID == id {
			return sum
		}
	}
	t.Fatalf("no issue %q in the listing", id)
	return Summary{}
}

func TestClientList(t *testing.T) {
	invocations := installFakeBd(t)
	dir := t.TempDir()

	got, err := NewClient(dir).List(t.Context(), ListFilter{})
	assert.NilError(t, err)
	assert.Equal(t, len(got), 81)

	// A bare filter still pins the limit: bd's own default would cap the
	// result at 50.
	inv := invocations()
	assert.Equal(t, len(inv), 1)
	assert.Equal(t, inv[0].args, "list --json --limit 0")
	assert.Equal(t, inv[0].dir, dir)
	// Only `bd where` runs with the envelope.
	assert.Equal(t, inv[0].envelope, "")

	closed := got[0]
	assert.Equal(t, closed.ID, "crabswarm-lpq.9")
	assert.Equal(t, closed.Title, "Step 9 — Dogfood")
	assert.Equal(t, closed.Status, StatusClosed)
	assert.Equal(t, closed.Type, "task")
	assert.Equal(t, closed.ParentID, "crabswarm-lpq")
	assert.DeepEqual(t, closed.Labels, []string{"step"})
	assert.Equal(t, closed.CreatedAt.Format("2006-01-02T15:04:05Z"), "2026-09-06T13:19:54Z")
	assert.Assert(t, !closed.ClosedAt.IsZero())
	// bd list carries the long text, so a listing is enough to see what an
	// issue says.
	assert.Assert(t, closed.Description != "")
	// Omitted fields decode as empty.
	assert.Equal(t, closed.Design, "")
	assert.Equal(t, closed.Notes, "")
	assert.Equal(t, closed.Assignee, "")
	assert.Equal(t, closed.CommentCount, 0)

	// An epic carries the long fields and the metadata object its children
	// leave empty.
	epic := findSummary(t, got, "crabswarm-lpq")
	assert.Equal(t, epic.Type, "epic")
	assert.Equal(t, epic.ParentID, "")
	assert.DeepEqual(t, epic.Labels, []string{"plan"})
	assert.Assert(t, epic.Design != "")
	assert.Assert(t, epic.AcceptanceCriteria != "")
	assert.Assert(t, epic.Notes != "")
	assert.Equal(t, epic.CommentCount, 36)
	// Metadata stays raw JSON; its keys are a caller convention.
	var meta map[string]any
	assert.NilError(t, json.Unmarshal(epic.Metadata, &meta))
	assert.Equal(t, meta["idea_gate_passed"], "2026-09-04")

	open := findSummary(t, got, "crabswarm-jp7")
	assert.Equal(t, open.Status, StatusOpen)
	assert.Assert(t, open.ClosedAt.IsZero())
	assert.DeepEqual(t, open.Labels, []string{"admin", "chat", "proto", "tui"})
}

func TestClientListCarriesDependencies(t *testing.T) {
	installFakeBd(t)

	got, err := NewClient(t.TempDir()).List(t.Context(), ListFilter{})
	assert.NilError(t, err)

	// Every issue reports its own outgoing edges, so flattening a listing
	// yields the whole graph of a backlog without a second bd call.
	var edges []Edge
	var carrying int
	for _, sum := range got {
		if len(sum.Dependencies) > 0 {
			carrying++
		}
		edges = append(edges, sum.Dependencies...)
	}
	assert.Equal(t, carrying, 32)
	assert.Equal(t, len(edges), 54)

	// An issue's edges always name it as the from side, and one issue reports
	// links of several kinds.
	assert.DeepEqual(t, findSummary(t, got, "crabswarm-lpq.9").Dependencies, []Edge{
		{FromID: "crabswarm-lpq.9", ToID: "crabswarm-lpq", Type: "parent-child"},
		{FromID: "crabswarm-lpq.9", ToID: "crabswarm-lpq.8", Type: "blocks"},
	})

	// An issue with no links reports none rather than an empty record.
	assert.Equal(t, len(findSummary(t, got, "crabswarm-jp7").Dependencies), 0)
}

func TestClientListFilters(t *testing.T) {
	invocations := installFakeBd(t)

	_, err := NewClient(t.TempDir()).List(t.Context(), ListFilter{
		Statuses:      []Status{StatusOpen, StatusInProgress},
		Labels:        []string{"chat", "tui"},
		ParentID:      "crabswarm-lpq",
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
			" --parent crabswarm-lpq --limit 5 --sort updated",
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

func TestClientRunSharesOneRun(t *testing.T) {
	invocations := installFakeBd(t)
	requireExclusiveFakeBd(t)
	client := NewClient(t.TempDir())

	// Released together, so every caller asks while the first run is still
	// in flight.
	start := make(chan struct{})
	results := make([][]Summary, 8)
	var g errgroup.Group
	for i := range results {
		g.Go(func() error {
			<-start
			got, err := client.List(t.Context(), ListFilter{})
			results[i] = got
			return err
		})
	}
	close(start)
	assert.NilError(t, g.Wait())

	// One bd read the database; every caller decoded that one output.
	inv := invocations()
	assert.Equal(t, len(inv), 1)
	assert.Equal(t, inv[0].args, "list --json --limit 0")
	for _, got := range results {
		assert.Equal(t, len(got), 81)
		assert.Equal(t, got[0].ID, results[0][0].ID)
	}
}

func TestClientRunQueuesDifferentArguments(t *testing.T) {
	invocations := installFakeBd(t)
	requireExclusiveFakeBd(t)
	client := NewClient(t.TempDir())

	var g errgroup.Group
	g.Go(func() error {
		_, err := client.List(t.Context(), ListFilter{})
		return err
	})
	g.Go(func() error {
		_, err := client.Get(t.Context(), "scratch-uoj")
		return err
	})
	// Two different argument lists share no run, so they can only both
	// succeed by taking the slot one after the other: the fake refuses a bd
	// started beside another one.
	assert.NilError(t, g.Wait())

	assert.Equal(t, len(invocations()), 2)
}

func TestClientRunCancelWhileQueued(t *testing.T) {
	invocations := installFakeBd(t)
	requireExclusiveFakeBd(t)
	client := NewClient(t.TempDir())

	running := make(chan error, 1)
	go func() {
		_, err := client.List(t.Context(), ListFilter{})
		running <- err
	}()
	// The fake records itself before it sleeps, so the slot is taken by the
	// time the invocation shows up.
	waitInvocations(t, invocations, 1)

	ctx, cancel := context.WithCancel(t.Context())
	queued := make(chan error, 1)
	go func() {
		_, err := client.Get(ctx, "scratch-uoj")
		queued <- err
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	assert.ErrorIs(t, <-queued, context.Canceled)

	select {
	case <-running:
		t.Fatal("the cancelled caller waited for the running bd to finish")
	default:
	}
	assert.NilError(t, <-running)

	// Nobody was left wanting the queued run, so it never spawned a bd.
	inv := invocations()
	assert.Equal(t, len(inv), 1)
	assert.Equal(t, inv[0].args, "list --json --limit 0")
}

func TestClientRunOutlivesOneCallerLeaving(t *testing.T) {
	invocations := installFakeBd(t)
	requireExclusiveFakeBd(t)
	client := NewClient(t.TempDir())

	var got []Summary
	staying := make(chan error, 1)
	go func() {
		var err error
		got, err = client.List(t.Context(), ListFilter{})
		staying <- err
	}()
	waitInvocations(t, invocations, 1)

	ctx, cancel := context.WithCancel(t.Context())
	leaving := make(chan error, 1)
	go func() {
		_, err := client.List(ctx, ListFilter{})
		leaving <- err
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	assert.ErrorIs(t, <-leaving, context.Canceled)

	// The run belongs to neither caller: the one that stayed still reads
	// the output of the single bd both had joined.
	assert.NilError(t, <-staying)
	assert.Equal(t, len(got), 81)
	assert.Equal(t, len(invocations()), 1)
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
