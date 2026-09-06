package issues

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/sync/errgroup"

	issuesv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/issues/v1"
	"github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/issues/v1/issuesv1connect"
	previewv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/preview/v1"
	"github.com/ngicks/crabswarm/crabswarm/preview/render"
)

// allStatuses names every status bd stores. bd's own listing hides closed
// issues, so anything that has to see one — the poll diff, the child tally
// behind the epic progress affordance, an epic's closed children — asks for
// all of them explicitly.
var allStatuses = []Status{
	StatusOpen,
	StatusInProgress,
	StatusBlocked,
	StatusDeferred,
	StatusClosed,
}

// Renderer turns one markdown fragment into HTML and its heading table of
// contents. [render.Renderer] implements it; the service names only what it
// uses so a caller can render issue text its own way.
type Renderer interface {
	Render(src []byte) (render.Document, error)
}

// issueReader is the part of [Client] the service reads a source through.
type issueReader interface {
	IssueLister
	ListFull(ctx context.Context, f ListFilter) ([]Issue, error)
	Get(ctx context.Context, id string) (*Issue, error)
}

// Service implements [issuesv1connect.IssuesServiceHandler] over a
// [SourceStore], one bd reader per registered source, and a [Renderer] for
// every markdown field an issue carries. It runs one [Poller] per source and
// fans the diffs out to WatchIssues subscribers.
//
// Pollers start when a source is added and stop when it is removed, all
// under the errgroup created by [Service.Run]. Adding or removing a source
// also publishes a SourcesChanged event.
type Service struct {
	logger   *slog.Logger
	renderer Renderer
	store    *SourceStore
	interval time.Duration
	hub      *eventHub

	// newClient builds the bd reader for a source's directory. It is a field
	// rather than a direct [NewClient] call so a test can read a source
	// without a bd subprocess.
	newClient func(dir string) issueReader

	mu      sync.Mutex
	clients map[string]issueReader // sourceID -> reader
	group   *errgroup.Group        // poller supervisor; set while Run runs
	runCtx  context.Context        // parent of every poller context; nil until Run
	pollers map[string]*pollHandle // sourceID -> running poller
}

// pollHandle is the per-source poller registration. The pointer identity
// lets a poller goroutine deregister only its own entry, so a
// remove-then-re-add race never drops the newer poller.
type pollHandle struct {
	cancel context.CancelFunc
}

var _ issuesv1connect.IssuesServiceHandler = (*Service)(nil)

// ServiceOption configures a [Service] built by [NewService].
type ServiceOption func(*Service)

// WithPollInterval sets how often each source is listed for changes. The
// default is [defaultPollInterval]; a non-positive duration is ignored.
func WithPollInterval(d time.Duration) ServiceOption {
	return func(s *Service) {
		if d > 0 {
			s.interval = d
		}
	}
}

// NewService returns a service reading the sources registered in store. A
// nil logger discards logs, a nil renderer falls back to the previewer's
// default markdown pipeline, and a nil store starts an empty registry. The
// returned service serves RPCs immediately; call [Service.Run] to start the
// polls behind WatchIssues.
func NewService(
	logger *slog.Logger,
	renderer Renderer,
	store *SourceStore,
	opts ...ServiceOption,
) *Service {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	if renderer == nil {
		renderer = render.New(render.Options{})
	}
	if store == nil {
		store = NewSourceStore()
	}
	s := &Service{
		logger:   logger,
		renderer: renderer,
		store:    store,
		interval: defaultPollInterval,
		hub:      newEventHub(),
		clients:  make(map[string]issueReader),
		pollers:  make(map[string]*pollHandle),
	}
	s.newClient = func(dir string) issueReader {
		return NewClient(dir, WithLogger(logger))
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Run supervises one [Poller] per registered source until ctx is cancelled.
// It blocks for its whole lifetime and returns nil on a clean shutdown, so
// it composes as an errgroup.Group.Go body next to an HTTP server.
func (s *Service) Run(ctx context.Context) error {
	g, gctx := errgroup.WithContext(ctx)

	s.mu.Lock()
	s.group = g
	s.runCtx = gctx
	s.mu.Unlock()

	// Sources already registered before Run get their poller now; the common
	// path adds them later via RPC.
	for _, src := range s.store.List() {
		s.startPoller(src)
	}

	// Sources are registered at runtime, so the group must outlive an empty
	// registry instead of returning the moment it finds nothing to wait for.
	g.Go(func() error {
		<-gctx.Done()
		return nil
	})
	return g.Wait()
}

// startPoller locks s.mu and starts a poller for src.
func (s *Service) startPoller(src Source) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startPollerLocked(src)
}

// startPollerLocked launches a poller for src under the Run errgroup unless
// one is already running or the service is not running. The caller holds
// s.mu.
func (s *Service) startPollerLocked(src Source) {
	if s.group == nil || s.runCtx == nil || s.runCtx.Err() != nil {
		return
	}
	if _, ok := s.pollers[src.ID]; ok {
		return
	}
	poller := NewPoller(
		s.logger,
		src.ID,
		s.clientLocked(src),
		s.interval,
		func(sourceID string, issueIDs []string) {
			s.hub.Publish(event{
				kind:     issuesChanged,
				sourceID: sourceID,
				issueIDs: issueIDs,
			})
		},
	)

	pctx, cancel := context.WithCancel(s.runCtx)
	handle := &pollHandle{cancel: cancel}
	s.pollers[src.ID] = handle
	s.group.Go(func() error {
		err := poller.Run(pctx)
		s.mu.Lock()
		if s.pollers[src.ID] == handle {
			delete(s.pollers, src.ID)
		}
		s.mu.Unlock()
		return err
	})
}

// clientLocked returns the bd reader for src, building it on first use. The
// caller holds s.mu.
func (s *Service) clientLocked(src Source) issueReader {
	if c, ok := s.clients[src.ID]; ok {
		return c
	}
	c := s.newClient(src.Dir)
	s.clients[src.ID] = c
	return c
}

// source resolves a request's source id to its registration and bd reader,
// reporting a connect NotFound when nothing is registered under that id.
func (s *Service) source(id string) (Source, issueReader, error) {
	src, ok := s.store.Get(id)
	if !ok {
		return Source{}, nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("source %q not found", id))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return src, s.clientLocked(src), nil
}

// --- issuesv1connect.IssuesServiceHandler ---

// ListSources returns the registered issue sources.
func (s *Service) ListSources(
	_ context.Context,
	_ *connect.Request[issuesv1.ListSourcesRequest],
) (*connect.Response[issuesv1.ListSourcesResponse], error) {
	sources := s.store.List()
	out := make([]*issuesv1.Source, len(sources))
	for i, src := range sources {
		out[i] = sourceToProto(src)
	}
	return connect.NewResponse(&issuesv1.ListSourcesResponse{Sources: out}), nil
}

// AddSource resolves the beads database governing the request's directory
// and registers it. A directory resolving to an already registered database
// is idempotent, which is what makes every worktree of one repository the
// same source. A directory with no beads database is NotFound.
func (s *Service) AddSource(
	ctx context.Context,
	req *connect.Request[issuesv1.AddSourceRequest],
) (*connect.Response[issuesv1.AddSourceResponse], error) {
	// The store mutation and the poller start are one critical section so a
	// concurrent RemoveSource of the same (deterministic) source ID cannot
	// interleave between them and leave a registered source unpolled.
	s.mu.Lock()
	src, added, err := s.store.Add(ctx, req.Msg.GetDir())
	if err != nil {
		s.mu.Unlock()
		if errors.Is(err, ErrNoBeads) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		s.logger.Warn("issues: adding source failed",
			"dir", req.Msg.GetDir(), "err", err)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("adding source %q", req.Msg.GetDir()))
	}
	if added {
		s.startPollerLocked(src)
	}
	s.mu.Unlock()

	if added {
		s.hub.Publish(event{kind: sourcesChanged})
	}
	return connect.NewResponse(&issuesv1.AddSourceResponse{
		Source: sourceToProto(src),
	}), nil
}

// RemoveSource drops a registered source and stops its poller.
func (s *Service) RemoveSource(
	_ context.Context,
	req *connect.Request[issuesv1.RemoveSourceRequest],
) (*connect.Response[issuesv1.RemoveSourceResponse], error) {
	id := req.Msg.GetSourceId()

	s.mu.Lock()
	src, ok := s.store.Remove(id)
	if !ok {
		s.mu.Unlock()
		return nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("source %q not found", id))
	}
	if h, ok := s.pollers[src.ID]; ok {
		h.cancel()
		delete(s.pollers, src.ID)
	}
	delete(s.clients, src.ID)
	s.mu.Unlock()

	s.hub.Publish(event{kind: sourcesChanged})
	return connect.NewResponse(&issuesv1.RemoveSourceResponse{}), nil
}

// ListIssues lists a source's issues newest-updated first.
func (s *Service) ListIssues(
	ctx context.Context,
	req *connect.Request[issuesv1.ListIssuesRequest],
) (*connect.Response[issuesv1.ListIssuesResponse], error) {
	_, client, err := s.source(req.Msg.GetSourceId())
	if err != nil {
		return nil, err
	}

	// The whole record is listed rather than the summary: bd reports an
	// issue's metadata in its listing, and the list view draws chips from it.
	listed, err := client.ListFull(ctx, ListFilter{
		Statuses:      statusesFromProto(req.Msg.GetStatuses()),
		Labels:        req.Msg.GetLabels(),
		ParentID:      req.Msg.GetParentId(),
		SortByUpdated: true,
	})
	if err != nil {
		return nil, s.bdError("listing issues", err)
	}
	counts, err := s.childCounts(ctx, client)
	if err != nil {
		return nil, s.bdError("tallying children", err)
	}

	out := make([]*issuesv1.IssueSummary, len(listed))
	for i, issue := range listed {
		out[i] = summaryToProto(issue.Summary, issue.Metadata, counts[issue.ID])
	}
	return connect.NewResponse(&issuesv1.ListIssuesResponse{Issues: out}), nil
}

// childCounts tallies every issue's children by parent. bd's listing carries
// no child count, so one extra all-status listing pays for both the total
// and the closed count the epic progress affordance needs.
func (s *Service) childCounts(
	ctx context.Context,
	client IssueLister,
) (map[string]childCount, error) {
	all, err := client.List(ctx, ListFilter{Statuses: allStatuses})
	if err != nil {
		return nil, err
	}
	counts := make(map[string]childCount)
	for _, sum := range all {
		if sum.ParentID == "" {
			continue
		}
		c := counts[sum.ParentID]
		c.total++
		if sum.Status == StatusClosed {
			c.closed++
		}
		counts[sum.ParentID] = c
	}
	return counts, nil
}

// GetIssue returns one issue with every markdown field rendered to HTML, its
// comments, its children and its dependencies.
func (s *Service) GetIssue(
	ctx context.Context,
	req *connect.Request[issuesv1.GetIssueRequest],
) (*connect.Response[issuesv1.GetIssueResponse], error) {
	_, client, err := s.source(req.Msg.GetSourceId())
	if err != nil {
		return nil, err
	}
	id := req.Msg.GetIssueId()

	issue, err := client.Get(ctx, id)
	if err != nil {
		// bd reports a missing id and a broken invocation the same way, a
		// non-zero exit with a message, so the id the caller asked for is
		// the only thing worth telling them; the rest goes to the log.
		s.logger.Warn("issues: reading issue failed", "issue", id, "err", err)
		return nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("issue %q not found", id))
	}

	// Children are listed with every status so a closed child still shows in
	// an epic's progress. Their own child counts stay zero: tallying those
	// costs a listing per child, and the UI asks for a child when it opens
	// one.
	children, err := client.ListFull(ctx, ListFilter{
		ParentID: id,
		Statuses: allStatuses,
	})
	if err != nil {
		return nil, s.bdError("listing children", err)
	}

	pbIssue, err := s.issueToProto(issue, children)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&issuesv1.GetIssueResponse{Issue: pbIssue}), nil
}

// issueToProto renders one full issue, its comments and the summaries of its
// children into the API shape.
func (s *Service) issueToProto(issue *Issue, children []Issue) (*issuesv1.Issue, error) {
	pbChildren := make([]*issuesv1.IssueSummary, len(children))
	var closed int
	for i, child := range children {
		if child.Status == StatusClosed {
			closed++
		}
		pbChildren[i] = summaryToProto(child.Summary, child.Metadata, childCount{})
	}

	summary := summaryToProto(issue.Summary, issue.Metadata, childCount{
		total:  len(children),
		closed: closed,
	})

	fields := make([]*issuesv1.RenderedField, 0, 5)
	for _, src := range []string{
		issue.Description,
		issue.Design,
		issue.AcceptanceCriteria,
		issue.Notes,
		issue.CloseReason,
	} {
		field, err := s.renderField(src)
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}

	comments := make([]*issuesv1.IssueComment, len(issue.Comments))
	for i, c := range issue.Comments {
		text, err := s.renderField(c.Text)
		if err != nil {
			return nil, err
		}
		comments[i] = &issuesv1.IssueComment{
			Id:        c.ID,
			Author:    c.Author,
			Text:      text,
			CreatedAt: timestampOrNil(c.CreatedAt),
		}
	}

	deps := make([]*issuesv1.IssueDependency, 0, len(issue.Dependencies))
	for _, d := range issue.Dependencies {
		if d.DependencyType == parentChildDependency {
			continue
		}
		deps = append(deps, &issuesv1.IssueDependency{
			Id:    d.ID,
			Title: d.Title,
			Type:  d.DependencyType,
			// `bd show` reports the edges this issue is the from side of —
			// what it depends on, what it was discovered from. The incoming
			// ones come from ListDependencies.
			Outgoing: true,
		})
	}

	return &issuesv1.Issue{
		Summary:            summary,
		Description:        fields[0],
		Design:             fields[1],
		AcceptanceCriteria: fields[2],
		Notes:              fields[3],
		CloseReason:        fields[4],
		MetadataJson:       metadataJSON(issue.Metadata),
		Comments:           comments,
		Children:           pbChildren,
		Dependencies:       deps,
	}, nil
}

// renderField renders one markdown field. An empty field renders to an empty
// RenderedField rather than to a nil message, so a client can read html and
// toc without a presence check.
func (s *Service) renderField(src string) (*issuesv1.RenderedField, error) {
	if strings.TrimSpace(src) == "" {
		return &issuesv1.RenderedField{}, nil
	}
	doc, err := s.renderer.Render([]byte(src))
	if err != nil {
		s.logger.Warn("issues: render failed", "err", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("render failed"))
	}
	toc := make([]*previewv1.Heading, len(doc.TOC))
	for i, h := range doc.TOC {
		toc[i] = &previewv1.Heading{Level: int32(h.Level), Text: h.Text, Id: h.ID}
	}
	return &issuesv1.RenderedField{Html: string(doc.HTML), Toc: toc}, nil
}

// ListDependencies is not implemented yet: the dependency graph is a later
// step, and bd's edge listing has to be added to [Client] first.
func (s *Service) ListDependencies(
	_ context.Context,
	_ *connect.Request[issuesv1.ListDependenciesRequest],
) (*connect.Response[issuesv1.ListDependenciesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented,
		errors.New("ListDependencies is not implemented"))
}

// WatchIssues streams the change notifications the per-source polls produce.
// The request carries no filter, so every subscriber sees every source's
// events and filters by source id itself. The stream runs until the client
// disconnects or the server shuts down.
func (s *Service) WatchIssues(
	ctx context.Context,
	_ *connect.Request[issuesv1.WatchIssuesRequest],
	stream *connect.ServerStream[issuesv1.WatchIssuesResponse],
) error {
	sub, unsub := s.hub.Subscribe()
	defer unsub()
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-sub:
			if !ok {
				return nil
			}
			if err := stream.Send(eventToProto(ev)); err != nil {
				return err
			}
		}
	}
}

// bdError logs a failed bd read and reports it as an internal error. bd's
// message can carry absolute host paths, so it stays server-side.
func (s *Service) bdError(what string, err error) error {
	s.logger.Warn("issues: "+what+" failed", "err", err)
	return connect.NewError(connect.CodeInternal, errors.New(what+" failed"))
}
