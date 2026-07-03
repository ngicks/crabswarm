// Package web bundles the compiled Preact single-page app for
// `crabswarm preview` and exposes it as an fs.FS for the HTTP layer to serve.
//
// The SPA under dist/ is built by `pnpm build` in web/ (buf generate + vite
// build) and committed to git: go:embed only includes files present in the
// module zip derived from the git tag, so a committed dist is what makes
// `go install github.com/ngicks/crabswarm/cmd/crabswarm@<tag>` work without a
// node toolchain (see doc/plan/.../DECISION.md D7a).
package web

//go:generate pnpm build

import (
	"embed"
	"io/fs"
	"os"
)

// DevFSEnv is the environment variable that, when set to a non-empty directory
// path, makes [FS] serve the SPA from that directory on disk instead of the
// embedded build, so frontend changes are picked up without rebuilding the Go
// binary (DECISION.md D7/D7a dev path). Point it at a freshly `vite build`-ed
// web/dist, or run `vite dev` separately and proxy to the daemon (see
// web/vite.config.ts).
const DevFSEnv = "CRABSWARM_PREVIEW_DEV_FS"

//go:embed all:dist
var distFS embed.FS

// DistFS returns the embedded production build (web/dist) rooted at its top
// level, so "index.html" and "assets/..." are at the root of the returned FS.
func DistFS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// dist is embedded at compile time; a failure here is a build defect.
		panic("web: embedded dist tree is missing: " + err.Error())
	}
	return sub
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
