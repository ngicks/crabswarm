package preview

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/timestamppb"

	crabpreviewv1 "github.com/ngicks/crabswarm/api/gen/proto/go/crabpreview/v1"
	"github.com/ngicks/crabswarm/api/gen/proto/go/crabpreview/v1/crabpreviewv1connect"
	"github.com/ngicks/crabswarm/crabswarm/preview/httpapi"
	"github.com/ngicks/crabswarm/crabswarm/preview/render"
)

// shutdownTimeout bounds the graceful HTTP shutdown once the serve context is
// cancelled.
const shutdownTimeout = 5 * time.Second

// Service is the running previewer: it composes the in-memory [RootStore], a
// shared broadcast [Hub], the markdown [render.Renderer] and one [Watcher] per
// registered root, and serves them over HTTP (connect API + /healthz + /raw +
// SPA static). It implements [crabpreviewv1connect.PreviewServiceHandler] so the
// HTTP layer mounts it directly, and [httpapi.RawResolver] for /raw path safety.
//
// Watchers are started when a root is added and stopped when it is removed, all
// under the errgroup created by [Service.Serve] (per the repo's concurrency
// rules). Adding or removing a root also publishes a RootsChanged event through
// the shared Hub.
type Service struct {
	logger   *slog.Logger
	cfg      Config
	store    *RootStore
	hub      *Hub
	renderer *render.Renderer

	mu       sync.Mutex
	static   fs.FS                   // injected SPA file system (see SetStaticFS)
	group    *errgroup.Group         // watcher supervisor; set while Serve runs
	runCtx   context.Context         // parent of every watcher context; nil until Serve
	watchers map[string]*watchHandle // rootID -> running watcher
}

// watchHandle is the per-root watcher registration. The pointer identity lets a
// watcher goroutine deregister only its own entry, so a remove+re-add race never
// deletes the newer watcher.
type watchHandle struct {
	cancel context.CancelFunc
}

var (
	_ crabpreviewv1connect.PreviewServiceHandler = (*Service)(nil)
	_ httpapi.RawResolver                        = (*Service)(nil)
)

// New builds a previewer service from cfg. A nil logger discards logs. The
// returned service is not yet listening; call [Service.Serve].
func New(logger *slog.Logger, cfg Config) (*Service, error) {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Service{
		logger:   logger,
		cfg:      cfg,
		store:    NewRootStore(),
		hub:      NewHub(),
		renderer: render.New(render.Options{}),
		watchers: make(map[string]*watchHandle),
	}, nil
}

// SetStaticFS injects the SPA file system served with an index.html fallback.
// It is a no-op-safe setter meant to be called once, before [Service.Serve];
// the cobra `preview __serve` command wires the embedded `web` file system here.
// A nil fsys leaves the built-in placeholder index.html in place.
func (s *Service) SetStaticFS(fsys fs.FS) {
	s.mu.Lock()
	s.static = fsys
	s.mu.Unlock()
}

// Handler returns the composed HTTP handler (connect API, /healthz, /raw, the
// render stylesheets and the SPA static files). It is exposed so a caller can
// mount the previewer under its own server; [Service.Serve] uses it internally.
func (s *Service) Handler() http.Handler {
	s.mu.Lock()
	static := s.static
	s.mu.Unlock()
	return httpapi.New(httpapi.Config{
		Logger:  s.logger,
		Connect: s,
		Raw:     s,
		Static:  static,
	})
}

// Serve listens on cfg.Addr and serves the previewer until ctx is cancelled,
// then shuts the HTTP server down gracefully. It supervises the per-root
// watchers on the same errgroup, so a fatal watcher error (only a failed
// construction or initial walk) also stops the server. It is the entry point for
// `preview __serve`.
func (s *Service) Serve(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return fmt.Errorf("preview serve: listen %q: %w", s.cfg.Addr, err)
	}

	g, gctx := errgroup.WithContext(ctx)

	s.mu.Lock()
	s.group = g
	s.runCtx = gctx
	s.mu.Unlock()

	// Start watchers for any roots already registered (e.g. added
	// programmatically before Serve); the common path adds them later via RPC.
	for _, root := range s.store.List() {
		s.startWatcher(root)
	}

	srv := &http.Server{
		Handler:     s.Handler(),
		BaseContext: func(net.Listener) context.Context { return gctx },
	}

	g.Go(func() error {
		<-gctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	})
	g.Go(func() error {
		s.logger.Info("preview server listening", "addr", ln.Addr().String())
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	return g.Wait()
}

// addRoot registers path, starts its watcher, and publishes RootsChanged when
// the root is genuinely new. Re-adding an existing root is idempotent and
// publishes nothing.
func (s *Service) addRoot(ctx context.Context, path string) (Root, error) {
	// The store mutation and the watcher start are one critical section so a
	// concurrent removeRoot of the same (deterministic) root ID cannot interleave
	// between them and leave a registered root without a watcher.
	s.mu.Lock()
	root, err := s.store.Add(ctx, path)
	if err != nil {
		s.mu.Unlock()
		return Root{}, err
	}
	started := s.startWatcherLocked(root)
	s.mu.Unlock()
	if started {
		s.hub.Publish(Event{Kind: RootsChanged})
	}
	return root, nil
}

// removeRoot drops a root (by id or name), stops its watcher, and publishes
// RootsChanged. It reports whether a root was found. The store removal and the
// watcher teardown share the same critical section as [Service.addRoot] so add
// and remove of the same root ID are fully serialized.
func (s *Service) removeRoot(idOrName string) (Root, bool) {
	s.mu.Lock()
	root, ok := s.store.Remove(idOrName)
	if !ok {
		s.mu.Unlock()
		return Root{}, false
	}
	if h, ok := s.watchers[root.ID]; ok {
		h.cancel()
		delete(s.watchers, root.ID)
	}
	s.mu.Unlock()
	s.hub.Publish(Event{Kind: RootsChanged})
	return root, true
}

// startWatcher locks s.mu and starts a watcher for root; used during Serve
// startup for the roots already registered before the server accepts requests.
func (s *Service) startWatcher(root Root) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startWatcherLocked(root)
}

// startWatcherLocked launches a watcher for root under the serve errgroup unless
// one is already running, the service is not serving yet, or it is shutting
// down. The caller must hold s.mu. It reports whether a new watcher was started.
func (s *Service) startWatcherLocked(root Root) bool {
	if s.group == nil || s.runCtx == nil || s.runCtx.Err() != nil {
		return false
	}
	if _, ok := s.watchers[root.ID]; ok {
		return false
	}
	w, err := NewWatcher(s.logger, root.ID, root.Path, nil, s.hub)
	if err != nil {
		s.logger.Warn("preview: watcher not started", "root", root.Path, "err", err)
		return false
	}

	wctx, cancel := context.WithCancel(s.runCtx)
	handle := &watchHandle{cancel: cancel}
	s.watchers[root.ID] = handle
	s.group.Go(func() error {
		err := w.Run(wctx)
		s.mu.Lock()
		if s.watchers[root.ID] == handle {
			delete(s.watchers, root.ID)
		}
		s.mu.Unlock()
		return err
	})
	return true
}

// --- crabpreviewv1connect.PreviewServiceHandler ---

// ListRoots returns the currently registered roots.
func (s *Service) ListRoots(
	_ context.Context,
	_ *connect.Request[crabpreviewv1.ListRootsRequest],
) (*connect.Response[crabpreviewv1.ListRootsResponse], error) {
	roots := s.store.List()
	pbRoots := make([]*crabpreviewv1.Root, len(roots))
	for i, r := range roots {
		pbRoots[i] = rootToProto(r)
	}
	return connect.NewResponse(&crabpreviewv1.ListRootsResponse{Roots: pbRoots}), nil
}

// AddRoot registers a directory path as a preview root and returns it. Adding an
// already-registered path is idempotent.
func (s *Service) AddRoot(
	ctx context.Context,
	req *connect.Request[crabpreviewv1.AddRootRequest],
) (*connect.Response[crabpreviewv1.AddRootResponse], error) {
	root, err := s.addRoot(ctx, req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&crabpreviewv1.AddRootResponse{Root: rootToProto(root)}), nil
}

// RemoveRoot drops a registered root and its watcher.
func (s *Service) RemoveRoot(
	_ context.Context,
	req *connect.Request[crabpreviewv1.RemoveRootRequest],
) (*connect.Response[crabpreviewv1.RemoveRootResponse], error) {
	if _, ok := s.removeRoot(req.Msg.GetRootId()); !ok {
		return nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("root %q not found", req.Msg.GetRootId()))
	}
	return connect.NewResponse(&crabpreviewv1.RemoveRootResponse{}), nil
}

// GetTree lists one directory level under a root.
func (s *Service) GetTree(
	_ context.Context,
	req *connect.Request[crabpreviewv1.GetTreeRequest],
) (*connect.Response[crabpreviewv1.GetTreeResponse], error) {
	root, ok := s.store.Get(req.Msg.GetRootId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("root %q not found", req.Msg.GetRootId()))
	}
	entries, err := Tree(root.Path, req.Msg.GetPath())
	if err != nil {
		return nil, s.fsErrToConnect(err)
	}
	pbEntries := make([]*crabpreviewv1.TreeEntry, len(entries))
	for i, e := range entries {
		pbEntries[i] = &crabpreviewv1.TreeEntry{
			Name:       e.Name,
			Type:       entryTypeToProto(e.Type),
			IsMarkdown: e.IsMarkdown,
		}
	}
	return connect.NewResponse(&crabpreviewv1.GetTreeResponse{Entries: pbEntries}), nil
}

// GetDocument renders a markdown file under a root to HTML with its title, TOC
// and modification time. The title falls back to the file name.
func (s *Service) GetDocument(
	_ context.Context,
	req *connect.Request[crabpreviewv1.GetDocumentRequest],
) (*connect.Response[crabpreviewv1.GetDocumentResponse], error) {
	root, ok := s.store.Get(req.Msg.GetRootId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("root %q not found", req.Msg.GetRootId()))
	}
	abs, err := SafeJoin(root.Path, req.Msg.GetPath())
	if err != nil {
		return nil, s.fsErrToConnect(err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, s.fsErrToConnect(err)
	}
	src, err := os.ReadFile(abs)
	if err != nil {
		return nil, s.fsErrToConnect(err)
	}
	doc, err := s.renderer.Render(src)
	if err != nil {
		s.logger.Warn("preview: render failed", "err", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("render failed"))
	}

	title := doc.Title
	if title == "" {
		title = documentTitleFallback(req.Msg.GetPath(), abs)
	}
	toc := make([]*crabpreviewv1.Heading, len(doc.TOC))
	for i, h := range doc.TOC {
		toc[i] = &crabpreviewv1.Heading{Level: int32(h.Level), Text: h.Text, Id: h.ID}
	}
	return connect.NewResponse(&crabpreviewv1.GetDocumentResponse{
		Html:  string(doc.HTML),
		Title: title,
		Toc:   toc,
		Mtime: timestamppb.New(info.ModTime()),
	}), nil
}

// WatchEvents streams live-reload events. The WatchEventsRequest carries no
// filter, so every subscriber receives every root's events; the client filters
// by root id itself. The stream runs until the client disconnects or the server
// shuts down (both cancel ctx, which derives from the serve context).
func (s *Service) WatchEvents(
	ctx context.Context,
	_ *connect.Request[crabpreviewv1.WatchEventsRequest],
	stream *connect.ServerStream[crabpreviewv1.WatchEventsResponse],
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
			msg := watchEventToProto(ev)
			if msg == nil {
				continue
			}
			if err := stream.Send(msg); err != nil {
				return err
			}
		}
	}
}

// ResolveRaw implements [httpapi.RawResolver]: it resolves a root-relative path
// under the named root to an absolute filesystem path, translating the preview
// package's path-safety errors into httpapi's sentinels so the HTTP layer stays
// independent of this package.
func (s *Service) ResolveRaw(rootID, relPath string) (string, error) {
	root, ok := s.store.Get(rootID)
	if !ok {
		return "", httpapi.ErrRootNotFound
	}
	abs, err := SafeJoin(root.Path, relPath)
	if err != nil {
		if errors.Is(err, ErrPathEscape) {
			return "", httpapi.ErrPathEscape
		}
		return "", err // includes fs.ErrNotExist for a missing target
	}
	return abs, nil
}

// fsErrToConnect maps a filesystem/path error to the matching connect code with
// a generic, non-leaky message: a path escape is a bad request, a missing path
// is not-found, anything else is internal. The raw error — which can embed
// symlink-resolved absolute host paths — is logged server-side only, never sent
// to the client.
func (s *Service) fsErrToConnect(err error) error {
	switch {
	case errors.Is(err, ErrPathEscape):
		return connect.NewError(connect.CodeInvalidArgument, errors.New("path escapes the root"))
	case errors.Is(err, fs.ErrNotExist):
		return connect.NewError(connect.CodeNotFound, errors.New("not found"))
	default:
		s.logger.Warn("preview: filesystem error", "err", err)
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}

// documentTitleFallback derives a title from the request path, falling back to
// the resolved absolute path when the request path has no usable base name.
func documentTitleFallback(reqPath, abs string) string {
	base := filepath.Base(reqPath)
	if base == "." || base == string(filepath.Separator) || base == "" {
		base = filepath.Base(abs)
	}
	return base
}

func rootToProto(r Root) *crabpreviewv1.Root {
	return &crabpreviewv1.Root{Id: r.ID, Path: r.Path, Name: r.Name}
}

func entryTypeToProto(t EntryType) crabpreviewv1.EntryType {
	switch t {
	case EntryDir:
		return crabpreviewv1.EntryType_ENTRY_TYPE_DIR
	case EntryFile:
		return crabpreviewv1.EntryType_ENTRY_TYPE_FILE
	default:
		return crabpreviewv1.EntryType_ENTRY_TYPE_UNSPECIFIED
	}
}

func watchEventToProto(ev Event) *crabpreviewv1.WatchEventsResponse {
	switch ev.Kind {
	case DocChanged:
		return &crabpreviewv1.WatchEventsResponse{
			Event: &crabpreviewv1.WatchEventsResponse_DocChanged{
				DocChanged: &crabpreviewv1.DocChanged{RootId: ev.RootID, Path: ev.Path},
			},
		}
	case TreeChanged:
		return &crabpreviewv1.WatchEventsResponse{
			Event: &crabpreviewv1.WatchEventsResponse_TreeChanged{
				TreeChanged: &crabpreviewv1.TreeChanged{RootId: ev.RootID, Dir: ev.Path},
			},
		}
	case RootsChanged:
		return &crabpreviewv1.WatchEventsResponse{
			Event: &crabpreviewv1.WatchEventsResponse_RootsChanged{
				RootsChanged: &crabpreviewv1.RootsChanged{},
			},
		}
	default:
		return nil
	}
}
