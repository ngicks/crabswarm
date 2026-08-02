package httpapi

import (
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
)

// placeholderIndexHTML is served when no SPA file system is injected (the
// frontend is built and embedded in a later step). It keeps the server usable —
// /healthz, the connect API, /raw and the stylesheets all work — and tells a
// browser visitor why the UI is absent.
const placeholderIndexHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>crabswarm preview</title>
</head>
<body>
<main>
<h1>crabswarm preview</h1>
<p>The preview web UI is not embedded in this build. The API is available under
<code>/crabpreview.v1.PreviewService/</code>.</p>
</main>
</body>
</html>
`

// staticHandler serves an injected SPA file system with an index.html fallback:
// an existing file is served as-is; anything else (unknown route, directory)
// falls back to index.html so client-side routing works. A nil file system
// serves the built-in placeholder.
type staticHandler struct {
	logger *slog.Logger
	fsys   fs.FS // may be nil
	files  http.Handler
}

func newStaticHandler(logger *slog.Logger, fsys fs.FS) *staticHandler {
	h := &staticHandler{logger: logger, fsys: fsys}
	if fsys != nil {
		h.files = http.FileServerFS(fsys)
	}
	return h
}

func (h *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.fsys != nil {
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name != "" {
			if info, err := fs.Stat(h.fsys, name); err == nil && !info.IsDir() {
				h.files.ServeHTTP(w, r)
				return
			}
		}
		if b, err := fs.ReadFile(h.fsys, "index.html"); err == nil {
			writeHTML(w, b)
			return
		}
	}
	writeHTML(w, []byte(placeholderIndexHTML))
}

func writeHTML(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(body)
}
