// Package web bundles the compiled Preact single-page app for
// `crabswarm preview` and exposes it as an fs.FS for the HTTP layer to serve.
//
// The SPA is built by `pnpm build` in web/ (buf generate + vite build) into
// dist/, which is *not* committed; `pnpm build` finishes by packing dist/ into
// dist.tar.zst (scripts/packdist), and that single seekable-zstd-compressed tar
// is what git tracks. go:embed only includes files present in the module zip
// derived from the git tag, so the committed blob is what makes
// `go install github.com/ngicks/crabswarm/cmd/crabswarm@<tag>` work without a
// node toolchain — one compressed file instead of a hundred build outputs whose
// diffs churn on every rebuild.
//
// Serving decompresses on demand: the seekable format indexes the zstd stream
// by decompressed offset, so tarfs reads a file by decoding only the frames it
// spans rather than expanding the whole archive up front.
package web

//go:generate pnpm build

import (
	"bytes"
	_ "embed"
	"io/fs"
	"os"
	"sync"

	seekable "github.com/SaveTheRbtz/zstd-seekable-format-go/pkg"
	"github.com/klauspost/compress/zstd"
	"github.com/ngicks/go-fsys-helper/tarfs"
)

// DevFSEnv is the environment variable that, when set to a non-empty directory
// path, makes [FS] serve the SPA from that directory on disk instead of the
// embedded build, so frontend changes are picked up without rebuilding the Go
// binary (DECISION.md D7/D7a dev path). Point it at a freshly `vite build`-ed
// web/dist, or run `vite dev` separately and proxy to the daemon (see
// web/vite.config.ts).
const DevFSEnv = "CRABSWARM_PREVIEW_DEV_FS"

//go:embed dist.tar.zst
var distTarZst []byte

var distFS = sync.OnceValue(func() fs.FS {
	dec, err := zstd.NewReader(nil)
	if err != nil {
		// The blob is produced by scripts/packdist at build time; anything
		// failing here is a build defect, not a runtime condition.
		panic("web: embedded dist archive is unusable: " + err.Error())
	}
	// bytes.Reader is an io.ReaderAt, so frame reads never move a shared
	// offset and concurrent ReadAt stays safe.
	sr, err := seekable.NewReader(bytes.NewReader(distTarZst), dec)
	if err != nil {
		panic("web: embedded dist archive is unusable: " + err.Error())
	}
	fsys, err := tarfs.New(sr, nil)
	if err != nil {
		panic("web: embedded dist archive is unusable: " + err.Error())
	}
	return fsys
})

// DistFS returns the embedded production build (web/dist) rooted at its top
// level, so "index.html" and "assets/..." are at the root of the returned FS.
func DistFS() fs.FS {
	return distFS()
}

// FS returns the file system the HTTP layer should serve the SPA from: the
// embedded production build by default, or the directory named by [DevFSEnv]
// when that environment variable is set.
func FS() fs.FS {
	if dir := os.Getenv(DevFSEnv); dir != "" {
		return os.DirFS(dir)
	}
	return DistFS()
}
