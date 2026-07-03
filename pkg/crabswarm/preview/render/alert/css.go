package alert

import _ "embed"

// CSS is the GitHub-style alert stylesheet matching the markdown-alert-*
// classes emitted by [HTMLRenderer]. The HTTP layer serves it alongside the
// chroma and github-markdown stylesheets.
//
//go:embed alert.css
var CSS string
