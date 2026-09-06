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
// issues, so the shared listing behind every read asks for all of them
// explicitly: a closed issue still has to reach the poll diff, an epic's
// children and a dependency graph. Narrowing to what a request asked for
// happens afterwards, in [filterSummaries].
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

// issueReader is the part of [Client] a source is read through: the listing
// its [Poller] runs, and the `bd show` behind GetIssue.
type issueReader interface {
	IssueLister
	Get(ctx context.Context, id string) (*Issue, error)
}

// Service implements [issuesv1connect.IssuesServiceHandler] over a
// [SourceStore], one bd reader and [Poller] per registered source, and a
// [Renderer] for every markdown field an issue carries.
//
// A source's poller owns the one listing every read of that source derives
// from, so a board, an issue's children and a dependency graph share a single
// `bd list`. It exists as soon as the source is first read, whether or not
// [Service.Run] is running; Run only adds the ticker that polls it on a
// schedule and fans the diffs out to WatchIssues subscribers.
//
// Tickers start when a source is added and stop when it is removed, all under
// the errgroup created by Run. Adding or removing a source also publishes a
// SourcesChanged event.
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
	states  map[string]*sourceState // sourceID -> reader and poller
	group   *errgroup.Group         // ticker supervisor; set while Run runs
	runCtx  context.Context         // parent of every ticker context; nil until Run
	pollers map[string]*pollHandle  // sourceID -> running ticker
}

// sourceState is what a source is read through: the bd reader and the poller
// holding that source's one shared listing. Both are built on the first read
// of the source and dropped together when it is removed.
type sourceState struct {
	reader issueReader
	poller *Poller
}

// pollHandle is the per-source ticker registration. The pointer identity
// lets a ticker goroutine deregister only its own entry, so a
// remove-then-re-add race never drops the newer ticker.
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
		states:   make(map[string]*sourceState),
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

// Run polls every registered source on the service's interval until ctx is
// cancelled. It blocks for its whole lifetime and returns nil on a clean
// shutdown, so it composes as an errgroup.Group.Go body next to an HTTP
// server. Reads do not wait for it: a source is read through its [Poller]
// whether or not Run ever started the ticker driving that poller.
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

// startPoller locks s.mu and starts src's poll ticker.
func (s *Service) startPoller(src Source) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startPollerLocked(src)
}

// startPollerLocked runs src's poller on its interval under the Run errgroup,
// unless the ticker is already running or the service is not. It does not
// create the poller: a source has one from its first read, so an RPC arriving
// before Run — or with no Run at all — still reads through it. The caller
// holds s.mu.
func (s *Service) startPollerLocked(src Source) {
	if s.group == nil || s.runCtx == nil || s.runCtx.Err() != nil {
		return
	}
	if _, ok := s.pollers[src.ID]; ok {
		return
	}
	poller := s.stateLocked(src).poller

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

// stateLocked returns the reader and poller of src, building both on first
// use. The caller holds s.mu.
func (s *Service) stateLocked(src Source) *sourceState {
	if st, ok := s.states[src.ID]; ok {
		return st
	}
	reader := s.newClient(src.Dir)
	st := &sourceState{
		reader: reader,
		poller: NewPoller(
			s.logger,
			src.ID,
			reader,
			s.interval,
			func(sourceID string, issueIDs []string) {
				s.hub.Publish(event{
					kind:     issuesChanged,
					sourceID: sourceID,
					issueIDs: issueIDs,
				})
			},
		),
	}
	s.states[src.ID] = st
	return st
}

// state resolves a request's source id to what the source is read through,
// reporting a connect NotFound when nothing is registered under that id.
//
// The registry is read under s.mu, the same lock RemoveSource drops a source
// and its state under, so a removal landing beside a request either happens
// first — nothing is built — or after, and takes the state built here with
// it. Reading the registry outside the lock would leave a rebuilt reader and
// poller behind for a source nobody can reach.
func (s *Service) state(id string) (*sourceState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src, ok := s.store.Get(id)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("source %q not found", id))
	}
	return s.stateLocked(src), nil
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
	// Registering runs `bd where` in a subprocess, so it stays outside s.mu:
	// holding the service lock across it would stall every other RPC for as
	// long as bd takes.
	src, added, err := s.store.Add(ctx, req.Msg.GetDir())
	if err != nil {
		if errors.Is(err, ErrNoBeads) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		s.logger.Warn("issues: adding source failed",
			"dir", req.Msg.GetDir(), "err", err)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("adding source %q", req.Msg.GetDir()))
	}

	if added {
		// A RemoveSource of the same (deterministic) source ID can land
		// between the registration above and this lock, so the registry is
		// read again under s.mu rather than trusted. RemoveSource drops the
		// source, its reader and its poller, and cancels its ticker in one
		// section under the same lock, which leaves two orderings and no
		// third: it ran first, the source is gone and no ticker starts; it
		// runs after, and it takes the ticker started here with it. Either
		// way a registered source is polled and an unregistered one is not.
		s.mu.Lock()
		if _, stillRegistered := s.store.Get(src.ID); stillRegistered {
			s.startPollerLocked(src)
		}
		s.mu.Unlock()

		s.hub.Publish(event{kind: sourcesChanged})
	}
	return connect.NewResponse(&issuesv1.AddSourceResponse{
		Source: sourceToProto(src),
	}), nil
}

// RemoveSource drops a registered source, stops its poll ticker and forgets
// the reader and poller it was read through, so re-adding it starts fresh.
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
	delete(s.states, src.ID)
	s.mu.Unlock()

	s.hub.Publish(event{kind: sourcesChanged})
	return connect.NewResponse(&issuesv1.RemoveSourceResponse{}), nil
}

// ListIssues lists a source's issues newest-updated first. The request's
// filter is applied to the source's shared listing rather than handed to bd,
// so a board costs the one listing every other read of the source shares.
func (s *Service) ListIssues(
	ctx context.Context,
	req *connect.Request[issuesv1.ListIssuesRequest],
) (*connect.Response[issuesv1.ListIssuesResponse], error) {
	st, err := s.state(req.Msg.GetSourceId())
	if err != nil {
		return nil, err
	}

	listed, err := st.poller.Refresh(ctx)
	if err != nil {
		return nil, s.bdError("listing issues", err)
	}

	// The tally reads the whole listing, not the filtered result: a child
	// hidden by the request's own filter still counts towards its parent.
	counts := childCounts(listed)
	matched := filterSummaries(listed, ListFilter{
		Statuses:      statusesFromProto(req.Msg.GetStatuses()),
		Labels:        req.Msg.GetLabels(),
		ParentID:      req.Msg.GetParentId(),
		SortByUpdated: true,
	})

	out := make([]*issuesv1.IssueSummary, len(matched))
	for i, sum := range matched {
		out[i] = summaryToProto(sum, counts[sum.ID])
	}
	return connect.NewResponse(&issuesv1.ListIssuesResponse{Issues: out}), nil
}

// GetIssue returns one issue with every markdown field rendered to HTML, its
// comments, its children and its dependencies.
func (s *Service) GetIssue(
	ctx context.Context,
	req *connect.Request[issuesv1.GetIssueRequest],
) (*connect.Response[issuesv1.GetIssueResponse], error) {
	st, err := s.state(req.Msg.GetSourceId())
	if err != nil {
		return nil, err
	}
	id := req.Msg.GetIssueId()

	// The listing is asked for before `bd show`, not after it: opening a
	// detail page fires this call beside ListIssues and ListDependencies, and
	// joining the listing first is what collapses all three onto one. Reading
	// the issue first would let the show finish and then start a second
	// listing of its own. Nothing is lost by the order — one bd runs at a
	// time per source either way.
	listed, err := st.poller.Refresh(ctx)
	if err != nil {
		return nil, s.bdError("listing issues", err)
	}

	issue, err := st.reader.Get(ctx, id)
	if err != nil {
		// bd reports a missing id and a broken invocation the same way, a
		// non-zero exit with a message, so the id the caller asked for is
		// the only thing worth telling them; the rest goes to the log.
		s.logger.Warn("issues: reading issue failed", "issue", id, "err", err)
		return nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("issue %q not found", id))
	}

	// Children come from the listing with every status, so a closed child
	// still shows in an epic's progress. Their own child counts stay zero:
	// the UI asks for a child when it opens one.
	pbIssue, err := s.issueToProto(issue, childrenOf(listed, id))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&issuesv1.GetIssueResponse{Issue: pbIssue}), nil
}

// issueToProto renders one full issue, its comments and the summaries of its
// children into the API shape.
func (s *Service) issueToProto(issue *Issue, children []Summary) (*issuesv1.Issue, error) {
	pbChildren := make([]*issuesv1.IssueSummary, len(children))
	var closed int
	for i, child := range children {
		if child.Status == StatusClosed {
			closed++
		}
		pbChildren[i] = summaryToProto(child, childCount{})
	}

	summary := summaryToProto(issue.Summary, childCount{
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

// ListDependencies returns the edges running between the request's issues,
// or every edge of the source when issue_ids is empty. The shared listing
// already carries each issue's outgoing edges, so a graph costs no bd call of
// its own however many nodes it draws.
func (s *Service) ListDependencies(
	ctx context.Context,
	req *connect.Request[issuesv1.ListDependenciesRequest],
) (*connect.Response[issuesv1.ListDependenciesResponse], error) {
	st, err := s.state(req.Msg.GetSourceId())
	if err != nil {
		return nil, err
	}

	// keep stays nil for the whole-source request: every id is in the set by
	// construction, so nothing needs filtering.
	var keep map[string]struct{}
	if ids := req.Msg.GetIssueIds(); len(ids) > 0 {
		keep = make(map[string]struct{}, len(ids))
		for _, id := range ids {
			keep[id] = struct{}{}
		}
	}

	listed, err := st.poller.Refresh(ctx)
	if err != nil {
		return nil, s.bdError("listing issues", err)
	}

	edges := listingEdges(listed, keep)
	out := make([]*issuesv1.IssueEdge, len(edges))
	for i, e := range edges {
		out[i] = edgeToProto(e)
	}
	return connect.NewResponse(&issuesv1.ListDependenciesResponse{Edges: out}), nil
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
