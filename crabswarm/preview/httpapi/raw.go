package httpapi

import (
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

// Sentinel errors a [RawResolver] returns so the raw handler can map them to
// the right HTTP status. The preview service translates its own path-safety
// errors into these before they cross the package boundary, keeping httpapi
// independent of the preview package.
var (
	// ErrRootNotFound reports that the referenced root id is not registered.
	ErrRootNotFound = errors.New("root not found")
	// ErrPathEscape reports that the requested path is absolute, climbs above
	// its root, or resolves through a symlink to a target outside the root.
	ErrPathEscape = errors.New("path escapes the root")
)

// RawResolver maps a root id and a root-relative path to an absolute filesystem
// path, enforcing path safety. It returns [ErrRootNotFound] for an unknown root
// and [ErrPathEscape] for a traversal attempt; a missing target surfaces as an
// error wrapping [fs.ErrNotExist].
type RawResolver interface {
	ResolveRaw(rootID, relPath string) (string, error)
}

// rawHandler serves the raw bytes of a file under a registered root. Path safety
// is enforced by resolver (which uses the shared SafeJoin gate); content type is
// detected by [http.ServeContent] from the extension, falling back to sniffing.
func rawHandler(logger *slog.Logger, resolver RawResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rootID := r.PathValue("rootId")
		relPath := r.PathValue("path")

		abs, err := resolver.ResolveRaw(rootID, relPath)
		if err != nil {
			switch {
			case errors.Is(err, ErrRootNotFound):
				http.Error(w, "root not found", http.StatusNotFound)
			case errors.Is(err, ErrPathEscape):
				http.Error(w, "forbidden path", http.StatusBadRequest)
			case errors.Is(err, fs.ErrNotExist):
				http.Error(w, "not found", http.StatusNotFound)
			default:
				logger.Warn("preview raw: resolve failed",
					"root", rootID, "path", relPath, "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		f, err := os.Open(abs)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				http.NotFound(w, r)
				return
			}
			logger.Warn("preview raw: open failed", "path", abs, "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer func() { _ = f.Close() }()

		info, err := f.Stat()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if info.IsDir() {
			// Directories are listed through GetTree, never served raw.
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, filepath.Base(abs), info.ModTime(), f)
	}
}
