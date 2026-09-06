// Package httpapi composes the previewer's HTTP surface: the connect-go
// PreviewService and IssuesService handlers, a /healthz probe, raw file serving
// under /raw, the render stylesheets under /assets/css, and the single-page-app
// static files with an index.html fallback.
//
// It deliberately does not import the parent preview package. The preview
// service owns the domain logic (root registry, watchers, renderer) and passes
// what this layer needs in through the connect handler interfaces and the narrow
// [RawResolver] interface, so preview -> httpapi is a one-way dependency with no
// import cycle.
package httpapi

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/issues/v1/issuesv1connect"
	"github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/preview/v1/previewv1connect"
	"github.com/ngicks/crabswarm/crabswarm/preview/render"
	"github.com/ngicks/crabswarm/crabswarm/preview/render/alert"
)

// Stable stylesheet paths the frontend links to. They are served from
// [render.ChromaStylesheets] (class-based chroma CSS) and [alert.CSS], switched
// together with the daisyUI light/dark theme.
const (
	// ChromaLightCSSPath serves the "github" chroma style (light theme).
	ChromaLightCSSPath = "/assets/css/chroma-github.css"
	// ChromaDarkCSSPath serves the "github-dark" chroma style (dark theme).
	ChromaDarkCSSPath = "/assets/css/chroma-github-dark.css"
	// AlertCSSPath serves the GitHub alert-block stylesheet.
	AlertCSSPath = "/assets/css/alert.css"
)

// Config carries the collaborators [New] wires into the mux.
type Config struct {
	// Logger receives handler-side warnings (raw resolve failures, missing
	// stylesheets). A nil Logger falls back to [slog.Default].
	Logger *slog.Logger
	// Connect is the PreviewService implementation mounted at the connect
	// route.
	Connect previewv1connect.PreviewServiceHandler
	// Issues is the IssuesService implementation, mounted at its own connect
	// route beside Connect. A nil Issues leaves the route unmounted, which is
	// what a previewer built without an issue registry wants.
	Issues issuesv1connect.IssuesServiceHandler
	// Raw resolves /raw requests to absolute filesystem paths with path-escape
	// rejection.
	Raw RawResolver
	// Static is the SPA file system served with an index.html fallback. When
	// nil, a minimal built-in placeholder index.html is served so the server is
	// still usable before the frontend is embedded.
	Static fs.FS
	// ConnectOpts are extra handler options passed to the connect handler.
	ConnectOpts []connect.HandlerOption
}

// New builds the composed previewer HTTP handler from cfg.
func New(cfg Config) http.Handler {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	mux := http.NewServeMux()

	connectPath, connectHandler := previewv1connect.NewPreviewServiceHandler(
		cfg.Connect,
		cfg.ConnectOpts...)
	mux.Handle(connectPath, connectHandler)

	if cfg.Issues != nil {
		issuesPath, issuesHandler := issuesv1connect.NewIssuesServiceHandler(
			cfg.Issues,
			cfg.ConnectOpts...)
		mux.Handle(issuesPath, issuesHandler)
	}

	mux.HandleFunc("GET /healthz", healthz)

	mux.Handle("GET /raw/{rootId}/{path...}", rawHandler(logger, cfg.Raw))

	for path, css := range cssAssets(logger) {
		mux.Handle("GET "+path, cssHandler(css))
	}

	// SPA / static: the catch-all, least specific route.
	mux.Handle("/", newStaticHandler(logger, cfg.Static))

	return mux
}

// healthz answers the readiness probe EnsureDaemon polls with a plain JSON 200.
func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// cssAssets returns the served stylesheets keyed by their URL path: the two
// class-based chroma themes and the alert stylesheet. A failure to build the
// chroma CSS is logged and that theme is simply omitted rather than failing the
// whole handler.
func cssAssets(logger *slog.Logger) map[string]string {
	out := map[string]string{AlertCSSPath: alert.CSS}
	sheets, err := render.ChromaStylesheets()
	if err != nil {
		logger.Warn("preview httpapi: chroma stylesheets unavailable", "err", err)
		return out
	}
	if css, ok := sheets["github"]; ok {
		out[ChromaLightCSSPath] = css
	}
	if css, ok := sheets["github-dark"]; ok {
		out[ChromaDarkCSSPath] = css
	}
	return out
}

// cssHandler serves a fixed stylesheet body as text/css.
func cssHandler(css string) http.HandlerFunc {
	body := []byte(css)
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write(body)
	}
}
