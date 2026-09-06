package issues

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"

	issuesv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/issues/v1"
	"github.com/ngicks/crabswarm/crabswarm/preview/render"
)

// newTestService returns a service rendering with the previewer's real
// markdown pipeline, so the rendered fields a test asserts on are the ones a
// client would receive.
func newTestService(t *testing.T, opts ...ServiceOption) *Service {
	t.Helper()
	return NewService(nil, render.New(render.Options{}), NewSourceStore(), opts...)
}

// addSource registers dir and returns the source id.
func addSource(t *testing.T, s *Service, dir string) string {
	t.Helper()
	res, err := s.AddSource(t.Context(), connect.NewRequest(&issuesv1.AddSourceRequest{
		Dir: dir,
	}))
	assert.NilError(t, err)
	return res.Msg.GetSource().GetId()
}

func TestServiceAddSourceNoBeads(t *testing.T) {
	installFakeBd(t)
	t.Setenv("FAKE_BD_NO_BEADS", "1")

	_, err := newTestService(t).AddSource(t.Context(),
		connect.NewRequest(&issuesv1.AddSourceRequest{Dir: t.TempDir()}))
	assert.Assert(t, err != nil)
	assert.Equal(t, connect.CodeOf(err), connect.CodeNotFound)
}

func TestServiceAddSourceSameDatabaseOnce(t *testing.T) {
	installFakeBd(t)
	svc := newTestService(t)

	first := addSource(t, svc, t.TempDir())
	// Another worktree of the same repository: bd reports the same .beads
	// directory, so the registry keeps one source.
	second := addSource(t, svc, t.TempDir())
	assert.Equal(t, second, first)

	res, err := svc.ListSources(t.Context(),
		connect.NewRequest(&issuesv1.ListSourcesRequest{}))
	assert.NilError(t, err)
	assert.Equal(t, len(res.Msg.GetSources()), 1)
	assert.Equal(t, res.Msg.GetSources()[0].GetPrefix(), "crabswarm")
	assert.Equal(t, res.Msg.GetSources()[0].GetBeadsPath(), fixtureBeadsPath)
}

func TestServiceSourceRegistryEvents(t *testing.T) {
	installFakeBd(t)
	svc := newTestService(t)

	sub, unsub := svc.hub.Subscribe()
	defer unsub()

	id := addSource(t, svc, t.TempDir())
	assert.Equal(t, receive(t, sub).kind, sourcesChanged)

	_, err := svc.RemoveSource(t.Context(),
		connect.NewRequest(&issuesv1.RemoveSourceRequest{SourceId: id}))
	assert.NilError(t, err)
	assert.Equal(t, receive(t, sub).kind, sourcesChanged)

	// The message a subscriber actually receives carries the registry event.
	assert.Assert(t, eventToProto(event{kind: sourcesChanged}).GetSourcesChanged() != nil)

	_, err = svc.RemoveSource(t.Context(),
		connect.NewRequest(&issuesv1.RemoveSourceRequest{SourceId: id}))
	assert.Equal(t, connect.CodeOf(err), connect.CodeNotFound)
}

// receive takes the next event, failing the test rather than hanging when the
// service publishes none.
func receive(t *testing.T, sub <-chan event) event {
	t.Helper()
	select {
	case ev := <-sub:
		return ev
	case <-time.After(5 * time.Second):
		t.Fatal("no event published")
		return event{}
	}
}

// findProtoIssue returns the listed issue with the given id, failing the test
// when the recording does not carry it.
func findProtoIssue(
	t *testing.T,
	listed []*issuesv1.IssueSummary,
	id string,
) *issuesv1.IssueSummary {
	t.Helper()
	for _, issue := range listed {
		if issue.GetId() == id {
			return issue
		}
	}
	t.Fatalf("no issue %q in the listing", id)
	return nil
}

func TestServiceListIssuesFromBd(t *testing.T) {
	invocations := installFakeBd(t)
	svc := newTestService(t)
	id := addSource(t, svc, t.TempDir())

	res, err := svc.ListIssues(t.Context(),
		connect.NewRequest(&issuesv1.ListIssuesRequest{SourceId: id}))
	assert.NilError(t, err)

	// 17 of the 81 recorded issues are closed, and a request naming no status
	// keeps bd's own default of hiding them.
	got := res.Msg.GetIssues()
	assert.Equal(t, len(got), 64)
	assert.Assert(t, !slices.ContainsFunc(got, func(issue *issuesv1.IssueSummary) bool {
		return issue.GetStatus() == issuesv1.IssueStatus_ISSUE_STATUS_CLOSED
	}), "a closed issue reached the default listing")

	// Newest-updated first.
	assert.Equal(t, got[0].GetId(), "crabswarm-3hp.1")
	assert.Equal(t, got[0].GetTitle(), "Decode bd list edges and re-record the Go fixtures")
	assert.Equal(t, got[0].GetIssueType(), "task")
	assert.Equal(t, got[0].GetStatus(), issuesv1.IssueStatus_ISSUE_STATUS_IN_PROGRESS)
	assert.Equal(t, got[0].GetCommentCount(), int32(0))
	assert.Assert(t, got[0].GetCreatedAt() != nil)
	// An issue recording no metadata still carries a parsable object.
	assert.Equal(t, got[0].GetMetadataJson(), "{}")
	assert.DeepEqual(t, got[0].GetLabels(), []string{"step"})

	open := findProtoIssue(t, got, "crabswarm-jp7")
	assert.Equal(t, open.GetStatus(), issuesv1.IssueStatus_ISSUE_STATUS_OPEN)
	assert.DeepEqual(t, open.GetLabels(), []string{"admin", "chat", "proto", "tui"})

	// One listing answers the whole request: the status filter, the ordering
	// and the child tally are all applied to it here rather than asked of bd.
	inv := invocations()
	assert.Equal(t, len(inv), 2) // where + list
	assert.Equal(t, inv[1].args,
		"list --json --status open,in_progress,blocked,deferred,closed --limit 0")
}

func TestServiceListIssuesCountsChildrenFromTheListing(t *testing.T) {
	installFakeBd(t)
	svc := newTestService(t)
	id := addSource(t, svc, t.TempDir())

	res, err := svc.ListIssues(t.Context(),
		connect.NewRequest(&issuesv1.ListIssuesRequest{SourceId: id}))
	assert.NilError(t, err)

	// Hand-counted in testdata/list.json: crabswarm-ylc has eight children,
	// two of them closed. The closed ones are counted but not listed, which
	// is the point of tallying the whole listing rather than the result.
	got := res.Msg.GetIssues()
	epic := findProtoIssue(t, got, "crabswarm-ylc")
	assert.Equal(t, epic.GetChildCount(), int32(8))
	assert.Equal(t, epic.GetChildClosedCount(), int32(2))
	assert.Assert(t, !slices.ContainsFunc(got, func(issue *issuesv1.IssueSummary) bool {
		return issue.GetId() == "crabswarm-ylc.6"
	}), "a closed child was listed; it should only be counted")
}

func TestServiceListIssuesUnknownSource(t *testing.T) {
	installFakeBd(t)

	_, err := newTestService(t).ListIssues(t.Context(),
		connect.NewRequest(&issuesv1.ListIssuesRequest{SourceId: "nope"}))
	assert.Equal(t, connect.CodeOf(err), connect.CodeNotFound)
}

func TestServiceGetIssueFromBd(t *testing.T) {
	invocations := installFakeBd(t)
	svc := newTestService(t)
	id := addSource(t, svc, t.TempDir())

	res, err := svc.GetIssue(t.Context(), connect.NewRequest(&issuesv1.GetIssueRequest{
		SourceId: id,
		IssueId:  "scratch-uoj",
	}))
	assert.NilError(t, err)

	issue := res.Msg.GetIssue()
	assert.Equal(t, issue.GetSummary().GetId(), "scratch-uoj")
	assert.Equal(t, issue.GetSummary().GetParentId(), "scratch-2o5")
	assert.Equal(t, issue.GetMetadataJson(), `{"plan":"doc/plan/x","rank":3}`)

	// Every text field is rendered; an empty one renders to an empty field
	// rather than a missing message.
	assert.Assert(t, strings.Contains(issue.GetDescription().GetHtml(), "the description"))
	assert.Assert(t, strings.Contains(issue.GetDesign().GetHtml(), "a design"))
	assert.Assert(t, strings.Contains(issue.GetAcceptanceCriteria().GetHtml(), "some criteria"))
	assert.Assert(t, strings.Contains(issue.GetNotes().GetHtml(), "a note"))
	assert.Equal(t, issue.GetCloseReason().GetHtml(), "")

	assert.Equal(t, len(issue.GetComments()), 1)
	assert.Equal(t, issue.GetComments()[0].GetAuthor(), "ngicks")
	assert.Assert(t, strings.Contains(issue.GetComments()[0].GetText().GetHtml(), "first comment"))

	// Children come from the source's listing, and the recorded listing holds
	// no issue under the recorded one.
	assert.Equal(t, len(issue.GetChildren()), 0)
	assert.Equal(t, issue.GetSummary().GetChildCount(), int32(0))

	// The one dependency the fixture carries is the parent link, which
	// parent_id and children already report.
	assert.Equal(t, len(issue.GetDependencies()), 0)

	// The listing runs first so a detail page firing this call beside a board
	// and a graph joins one listing instead of starting a second, and it is
	// the whole-source listing: nothing asks bd for one issue's children.
	inv := invocations()
	assert.Equal(t, len(inv), 3) // where + list + show
	assert.Equal(t, inv[1].args,
		"list --json --status open,in_progress,blocked,deferred,closed --limit 0")
	assert.Equal(t, inv[2].args, "show --id=scratch-uoj --json --include-comments")
	assert.Assert(t, !slices.ContainsFunc(inv, func(i invocation) bool {
		return strings.Contains(i.args, "--parent")
	}), "bd was asked for one issue's children")
}

func TestServiceGetIssueChildrenComeFromTheListing(t *testing.T) {
	reader := &stubReader{issues: []Issue{
		{Summary: Summary{ID: "epic", Status: StatusOpen}},
		{Summary: Summary{ID: "c1", ParentID: "epic", Status: StatusClosed}},
		{Summary: Summary{ID: "c2", ParentID: "epic", Status: StatusOpen}},
		{Summary: Summary{ID: "other", ParentID: "elsewhere", Status: StatusOpen}},
	}}
	svc, id := withStub(t, reader)

	res, err := svc.GetIssue(t.Context(), connect.NewRequest(&issuesv1.GetIssueRequest{
		SourceId: id,
		IssueId:  "epic",
	}))
	assert.NilError(t, err)

	issue := res.Msg.GetIssue()
	// A closed child is still a child: an epic's progress is what the count
	// is for.
	assert.DeepEqual(t, []string{
		issue.GetChildren()[0].GetId(),
		issue.GetChildren()[1].GetId(),
	}, []string{"c1", "c2"})
	assert.Equal(t, issue.GetSummary().GetChildCount(), int32(2))
	assert.Equal(t, issue.GetSummary().GetChildClosedCount(), int32(1))

	// The source was listed once, whole and unnarrowed. Asking bd for one
	// issue's children would be a second listing of the same database.
	assert.DeepEqual(t, reader.filters, []ListFilter{{Statuses: allStatuses}})
}

func TestServiceGetIssueMissing(t *testing.T) {
	installFakeBd(t)
	svc := newTestService(t)
	id := addSource(t, svc, t.TempDir())

	_, err := svc.GetIssue(t.Context(), connect.NewRequest(&issuesv1.GetIssueRequest{
		SourceId: id,
		IssueId:  "scratch-nope",
	}))
	assert.Equal(t, connect.CodeOf(err), connect.CodeNotFound)
}

func TestServiceListDependenciesUnknownSource(t *testing.T) {
	_, err := newTestService(t).ListDependencies(t.Context(),
		connect.NewRequest(&issuesv1.ListDependenciesRequest{SourceId: "nope"}))
	assert.Equal(t, connect.CodeOf(err), connect.CodeNotFound)
}

func TestServiceListDependenciesWholeSource(t *testing.T) {
	invocations := installFakeBd(t)
	svc := newTestService(t)
	id := addSource(t, svc, t.TempDir())

	res, err := svc.ListDependencies(t.Context(),
		connect.NewRequest(&issuesv1.ListDependenciesRequest{SourceId: id}))
	assert.NilError(t, err)

	// The listing already carries every issue's outgoing edges, so the whole
	// graph costs one bd call.
	inv := invocations()
	assert.Equal(t, len(inv), 2) // where + list
	assert.Equal(t, inv[1].args,
		"list --json --status open,in_progress,blocked,deferred,closed --limit 0")

	got := res.Msg.GetEdges()
	assert.Equal(t, len(got), 54)
	assert.Equal(t, got[0].GetFromId(), "crabswarm-lpq.9")
	assert.Equal(t, got[0].GetToId(), "crabswarm-lpq")
	// The parent link is an edge here, unlike in GetIssue's dependencies.
	assert.Equal(t, got[0].GetType(), "parent-child")
	assert.Equal(t, got[1].GetFromId(), "crabswarm-lpq.9")
	assert.Equal(t, got[1].GetToId(), "crabswarm-lpq.8")
	assert.Equal(t, got[1].GetType(), "blocks")
}

func TestServiceListDependenciesWithinRequestedIssues(t *testing.T) {
	svc, id := withStub(t, &stubReader{
		issues: []Issue{
			{Summary: Summary{ID: "epic", Status: StatusOpen}},
			{Summary: Summary{
				ID:       "c1",
				ParentID: "epic",
				Status:   StatusOpen,
				Dependencies: []Edge{
					{FromID: "c1", ToID: "epic", Type: "parent-child"},
					{FromID: "c1", ToID: "outside", Type: "blocks"},
				},
			}},
			{Summary: Summary{ID: "outside", Status: StatusOpen}},
		},
	})

	res, err := svc.ListDependencies(t.Context(),
		connect.NewRequest(&issuesv1.ListDependenciesRequest{
			SourceId: id,
			IssueIds: []string{"epic", "c1"},
		}))
	assert.NilError(t, err)

	// The caller asked for a set of issues to draw, so the edge leaving it is
	// dropped: it points at a node the caller does not hold.
	got := res.Msg.GetEdges()
	assert.Equal(t, len(got), 1)
	assert.Equal(t, got[0].GetFromId(), "c1")
	assert.Equal(t, got[0].GetToId(), "epic")
	assert.Equal(t, got[0].GetType(), "parent-child")
}

// stubReader is an in-memory [issueReader]: the service reads it instead of
// running bd, so a test can pin the exact records a source reports. It applies
// no filter of its own — the service narrows a listing in Go — and records
// every filter it was asked for, so a test can see what reached bd.
type stubReader struct {
	issues  []Issue
	filters []ListFilter
}

func (r *stubReader) List(_ context.Context, f ListFilter) ([]Summary, error) {
	r.filters = append(r.filters, f)
	out := make([]Summary, len(r.issues))
	for i, issue := range r.issues {
		out[i] = issue.Summary
	}
	return out, nil
}

func (r *stubReader) Get(_ context.Context, id string) (*Issue, error) {
	for i := range r.issues {
		if r.issues[i].ID == id {
			return &r.issues[i], nil
		}
	}
	return nil, ErrNoBeads
}

// withStub registers one source read through reader instead of bd.
func withStub(t *testing.T, reader *stubReader) (*Service, string) {
	t.Helper()
	svc := newTestService(t)
	svc.newClient = func(string) issueReader { return reader }
	src := Source{ID: "src-1", BeadsPath: "/repo/.beads", Prefix: "scratch", Dir: "/repo"}
	svc.store.sources[src.ID] = src
	return svc, src.ID
}

func TestServiceListIssuesCountsChildren(t *testing.T) {
	svc, id := withStub(t, &stubReader{issues: []Issue{
		{Summary: Summary{ID: "epic", Title: "an epic", Status: StatusOpen}},
		{Summary: Summary{ID: "c1", ParentID: "epic", Status: StatusClosed}},
		{Summary: Summary{ID: "c2", ParentID: "epic", Status: StatusOpen}},
		{Summary: Summary{
			ID:       "meta",
			Status:   StatusOpen,
			Metadata: json.RawMessage("{\n  \"plan\": \"doc/plan/x\"\n}"),
		}},
	}})

	res, err := svc.ListIssues(t.Context(),
		connect.NewRequest(&issuesv1.ListIssuesRequest{SourceId: id}))
	assert.NilError(t, err)

	// The closed child is counted but not listed: a request naming no status
	// keeps bd's own default of hiding closed issues.
	got := res.Msg.GetIssues()
	assert.Equal(t, len(got), 3)

	// bd reports no child count, so it is tallied from the listing: the epic
	// has two children, one of them closed.
	epic := findProtoIssue(t, got, "epic")
	assert.Equal(t, epic.GetChildCount(), int32(2))
	assert.Equal(t, epic.GetChildClosedCount(), int32(1))
	assert.Equal(t, findProtoIssue(t, got, "c2").GetChildCount(), int32(0))
	// Metadata rides along compacted, verbatim keys and all.
	assert.Equal(t, findProtoIssue(t, got, "meta").GetMetadataJson(),
		`{"plan":"doc/plan/x"}`)
}

// TestServiceDetailPageBurstReadsBdOnce fires the three calls a detail page
// makes at once. The source is listed once for all of them and read once for
// the issue itself, and the fake bd refuses to run beside another one, so an
// extra invocation would fail the burst rather than merely be counted.
func TestServiceDetailPageBurstReadsBdOnce(t *testing.T) {
	invocations := installFakeBd(t)
	svc := newTestService(t)
	// Registering runs bd too; the exclusive check is for the burst below.
	id := addSource(t, svc, t.TempDir())
	requireExclusiveFakeBd(t)

	// Released together, so every call asks while the first bd is still in
	// flight.
	start := make(chan struct{})
	var g errgroup.Group
	g.Go(func() error {
		<-start
		_, err := svc.ListIssues(t.Context(),
			connect.NewRequest(&issuesv1.ListIssuesRequest{SourceId: id}))
		return err
	})
	g.Go(func() error {
		<-start
		_, err := svc.GetIssue(t.Context(), connect.NewRequest(&issuesv1.GetIssueRequest{
			SourceId: id,
			IssueId:  "scratch-uoj",
		}))
		return err
	})
	g.Go(func() error {
		<-start
		_, err := svc.ListDependencies(t.Context(),
			connect.NewRequest(&issuesv1.ListDependenciesRequest{SourceId: id}))
		return err
	})
	close(start)
	assert.NilError(t, g.Wait())

	var lists, shows int
	for _, inv := range invocations() {
		switch {
		case strings.HasPrefix(inv.args, "list "):
			lists++
		case strings.HasPrefix(inv.args, "show "):
			shows++
		}
	}
	assert.Equal(t, lists, 1)
	assert.Equal(t, shows, 1)
}

func TestServiceGetIssueRendersMarkdown(t *testing.T) {
	svc, id := withStub(t, &stubReader{issues: []Issue{{
		Summary: Summary{
			ID:          "scratch-1",
			Title:       "rendered",
			Status:      StatusOpen,
			Description: "intro\n\n## Context\n\nbody text\n",
		},
		CloseReason: "### Conclusion\n\ndone\n",
		Comments: []Comment{{
			ID:     "c-1",
			Author: "ngicks",
			Text:   "## Discussion\n\nsomething",
		}},
		Dependencies: []Dependency{
			{Summary: Summary{ID: "dep-1", Title: "blocker"}, DependencyType: "blocks"},
			{Summary: Summary{ID: "scratch-0", Title: "parent"}, DependencyType: "parent-child"},
		},
	}}})

	res, err := svc.GetIssue(t.Context(), connect.NewRequest(&issuesv1.GetIssueRequest{
		SourceId: id,
		IssueId:  "scratch-1",
	}))
	assert.NilError(t, err)
	issue := res.Msg.GetIssue()

	desc := issue.GetDescription()
	assert.Assert(t, strings.Contains(desc.GetHtml(), "<h2"))
	assert.Equal(t, len(desc.GetToc()), 1)
	assert.Equal(t, desc.GetToc()[0].GetText(), "Context")
	assert.Equal(t, desc.GetToc()[0].GetLevel(), int32(2))
	// The heading anchor matches the id emitted in the body.
	assert.Assert(t, strings.Contains(desc.GetHtml(), `id="`+desc.GetToc()[0].GetId()+`"`))

	// A close reason is markdown too, and so is a comment.
	assert.Assert(t, strings.Contains(issue.GetCloseReason().GetHtml(), "<h3"))
	comment := issue.GetComments()[0].GetText()
	assert.Assert(t, strings.Contains(comment.GetHtml(), "<h2"))
	assert.Equal(t, len(comment.GetToc()), 1)

	// An empty field renders to an empty fragment with no headings.
	assert.Equal(t, issue.GetDesign().GetHtml(), "")
	assert.Equal(t, len(issue.GetDesign().GetToc()), 0)

	// The parent link is dropped; what is left points away from this issue.
	assert.Equal(t, len(issue.GetDependencies()), 1)
	assert.Equal(t, issue.GetDependencies()[0].GetId(), "dep-1")
	assert.Equal(t, issue.GetDependencies()[0].GetType(), "blocks")
	assert.Assert(t, issue.GetDependencies()[0].GetOutgoing())
}

func TestServiceRunStopsOnCancel(t *testing.T) {
	installFakeBd(t)
	svc := newTestService(t, WithPollInterval(10*time.Millisecond))

	sub, unsub := svc.hub.Subscribe()
	defer unsub()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- svc.Run(ctx) }()

	addSource(t, svc, t.TempDir())
	assert.Equal(t, receive(t, sub).kind, sourcesChanged)

	cancel()
	select {
	case err := <-done:
		assert.NilError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
}
