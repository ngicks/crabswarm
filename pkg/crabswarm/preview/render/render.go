// Package render turns markdown source into GitHub-flavored HTML with a table
// of contents and a document title. Its goldmark pipeline mirrors go-grip's
// output: GFM (tables, task lists, strikethrough, linkify), footnotes, emoji,
// chroma syntax highlighting (class-based), client-side mermaid, GitHub alert
// blocks, and MathJax-compatible math markup. Heading anchors come from a
// single AutoHeadingID pass so the TOC links always match the body HTML.
package render

import (
	"bytes"
	"fmt"
	"html/template"
	"maps"
	"strings"
	"sync"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/mermaid"

	"github.com/ngicks/crabswarm/pkg/crabswarm/preview/render/alert"
	"github.com/ngicks/crabswarm/pkg/crabswarm/preview/render/mathjax"
)

// DefaultHighlightStyle is the chroma style used when [Options.HighlightStyle]
// is empty. It resolves the token classes for code blocks; the matching CSS is
// served separately via [ChromaCSS].
const DefaultHighlightStyle = "github"

// ChromaStyleNames are the chroma styles the previewer ships CSS for: the light
// "github" and dark "github-dark" themes, matching go-grip.
var ChromaStyleNames = []string{"github", "github-dark"}

// Document is the result of rendering a single markdown file.
type Document struct {
	// HTML is the rendered body markup. It is trusted output produced by the
	// goldmark pipeline, not caller-supplied HTML.
	HTML template.HTML
	// Title is the text of the first level-1 heading, or "" if there is none
	// (the caller falls back to the file name).
	Title string
	// TOC holds the headings of levels 1-4 in document order. Each ID matches
	// the anchor emitted in HTML by the same AutoHeadingID pass.
	TOC []Heading
}

// Heading is a single table-of-contents entry.
type Heading struct {
	Level int
	Text  string
	ID    string
}

// Options configures a [Renderer].
type Options struct {
	// HighlightStyle is the chroma style goldmark-highlighting resolves token
	// classes against. Because highlighting runs with WithClasses(true), the
	// style only affects class names, not inline colors; the matching CSS is
	// served separately via [ChromaCSS]. Defaults to [DefaultHighlightStyle].
	HighlightStyle string
}

// Renderer renders markdown to [Document]s. It is safe for concurrent use: the
// underlying goldmark.Markdown is stateless across Render calls.
type Renderer struct {
	md goldmark.Markdown
}

// New builds a Renderer with the GitHub-flavored goldmark pipeline.
func New(opts Options) *Renderer {
	if opts.HighlightStyle == "" {
		opts.HighlightStyle = DefaultHighlightStyle
	}
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			emoji.Emoji,
			highlighting.NewHighlighting(
				highlighting.WithStyle(opts.HighlightStyle),
				highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
			),
			// NoScript: the SPA bundles mermaid itself, so the renderer must not
			// emit a CDN <script src=".../mermaid.min.js"> into the HTML.
			&mermaid.Extender{RenderMode: mermaid.RenderModeClient, NoScript: true},
			mathjax.NewMathJax(),
			alert.New(),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)
	return &Renderer{md: md}
}

// Render parses and renders src into a [Document]. The AST is parsed once so
// the heading IDs walked for the TOC are identical to the anchors emitted in
// the HTML body.
func (r *Renderer) Render(src []byte) (Document, error) {
	node := r.md.Parser().Parse(text.NewReader(src))

	var buf bytes.Buffer
	if err := r.md.Renderer().Render(&buf, src, node); err != nil {
		return Document{}, fmt.Errorf("render markdown: %w", err)
	}

	title, toc := extractHeadings(node, src)
	return Document{
		HTML:  template.HTML(buf.String()),
		Title: title,
		TOC:   toc,
	}, nil
}

// extractHeadings walks doc for headings, returning the first level-1 heading's
// text as the title and the levels 1-4 headings as the TOC.
func extractHeadings(doc ast.Node, source []byte) (string, []Heading) {
	var title string
	var toc []Heading
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		txt := headingText(h, source)
		if h.Level == 1 && title == "" {
			title = txt
		}
		if h.Level >= 1 && h.Level <= 4 {
			toc = append(toc, Heading{Level: h.Level, Text: txt, ID: headingID(h)})
		}
		// Headings cannot nest, so there is nothing to descend into.
		return ast.WalkSkipChildren, nil
	})
	return title, toc
}

// headingText concatenates the plain-text content of a heading node.
func headingText(n ast.Node, source []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := c.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(source))
		case *ast.String:
			b.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

// headingID returns the AutoHeadingID-assigned anchor id of a heading node.
func headingID(n ast.Node) string {
	v, ok := n.AttributeString("id")
	if !ok {
		return ""
	}
	switch id := v.(type) {
	case []byte:
		return string(id)
	case string:
		return id
	default:
		return ""
	}
}

// ChromaCSS returns the class-based (WithClasses(true)) chroma highlighting CSS
// for the named style, suitable for serving as a static stylesheet alongside
// the rendered HTML. It returns an error for an unknown style.
func ChromaCSS(style string) (string, error) {
	s, ok := styles.Registry[style]
	if !ok {
		return "", fmt.Errorf("unknown chroma style %q", style)
	}
	var buf bytes.Buffer
	formatter := chromahtml.New(chromahtml.WithClasses(true))
	if err := formatter.WriteCSS(&buf, s); err != nil {
		return "", fmt.Errorf("write chroma css for %q: %w", style, err)
	}
	return buf.String(), nil
}

var chromaStylesheets = sync.OnceValues(func() (map[string]string, error) {
	out := make(map[string]string, len(ChromaStyleNames))
	for _, name := range ChromaStyleNames {
		css, err := ChromaCSS(name)
		if err != nil {
			return nil, err
		}
		out[name] = css
	}
	return out, nil
})

// ChromaStylesheets returns the class-based chroma CSS for every style in
// [ChromaStyleNames], keyed by style name. The CSS is generated once on the
// first call and cached; the HTTP layer serves it as static stylesheets. The
// returned map is a fresh copy the caller may retain but must not expect to
// reflect later changes.
func ChromaStylesheets() (map[string]string, error) {
	cached, err := chromaStylesheets()
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(cached))
	maps.Copy(out, cached)
	return out, nil
}
