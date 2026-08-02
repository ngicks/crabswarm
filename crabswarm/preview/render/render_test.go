package render_test

import (
	"strings"
	"testing"

	"github.com/ngicks/crabswarm/crabswarm/preview/render"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// renderDoc renders src with a default Renderer, failing the test on error.
func renderDoc(t *testing.T, src string) render.Document {
	t.Helper()
	doc, err := render.New(render.Options{}).Render([]byte(src))
	assert.NilError(t, err)
	return doc
}

func TestRenderGFMTable(t *testing.T) {
	html := string(renderDoc(t, "| a | b |\n| - | - |\n| 1 | 2 |\n").HTML)
	assert.Assert(t, cmp.Contains(html, "<table>"))
	assert.Assert(t, cmp.Contains(html, "<th>a</th>"))
	assert.Assert(t, cmp.Contains(html, "<td>1</td>"))
}

func TestRenderGFMTaskList(t *testing.T) {
	html := string(renderDoc(t, "- [x] done\n- [ ] todo\n").HTML)
	// GFM renders task list items as disabled checkboxes, checked for [x].
	assert.Assert(t, cmp.Contains(html, `<input checked="" disabled="" type="checkbox"> done`))
	assert.Assert(t, cmp.Contains(html, `<input disabled="" type="checkbox"> todo`))
}

func TestRenderAlert(t *testing.T) {
	html := string(renderDoc(t, "> [!NOTE]\n> hello note\n").HTML)
	assert.Assert(t, cmp.Contains(html, `class="markdown-alert markdown-alert-note"`))
	assert.Assert(t, cmp.Contains(html, `class="markdown-alert-title"`))
	assert.Assert(t, cmp.Contains(html, "octicon-info"))
	assert.Assert(t, cmp.Contains(html, "<p>hello note</p>"))
	// The [!NOTE] marker is consumed, not rendered as body text.
	assert.Assert(t, !strings.Contains(html, "[!NOTE]"), "marker leaked into output: %s", html)
	// The blockquote is replaced by the alert div, not kept.
	assert.Assert(t, !strings.Contains(html, "<blockquote"), "blockquote not replaced: %s", html)
}

func TestRenderMermaidNoScript(t *testing.T) {
	html := string(renderDoc(t, "```mermaid\ngraph TD; A-->B;\n```\n").HTML)
	// The mermaid block renders as a client-side container the SPA enriches.
	assert.Assert(t, cmp.Contains(html, `class="mermaid"`))
	assert.Assert(t, cmp.Contains(html, "graph TD; A--&gt;B;"))
	// NoScript: true suppresses the CDN <script src=".../mermaid.min.js"> the
	// extender would otherwise append, since the SPA bundles mermaid itself.
	assert.Assert(t, !strings.Contains(html, "<script"), "script leaked into output: %s", html)
}

func TestRenderCodeHighlightingClasses(t *testing.T) {
	html := string(renderDoc(t, "```go\nfunc main() {}\n```\n").HTML)
	// chroma with WithClasses(true): the block is class-based, not inline.
	assert.Assert(t, cmp.Contains(html, `<pre class="chroma">`))
	assert.Assert(t, cmp.Contains(html, `<span class="kd">func</span>`))
	assert.Assert(t, cmp.Contains(html, `<span class="nf">main</span>`))
	// No inline style attributes when highlighting by class.
	assert.Assert(t, !strings.Contains(html, "style="), "unexpected inline styles: %s", html)
}

func TestRenderMathBlock(t *testing.T) {
	html := string(renderDoc(t, "$$\na = b\n$$\n").HTML)
	// Block math emits MathJax \[ ... \] display markup.
	assert.Assert(t, cmp.Contains(html, `<span class="math display">\[`))
	assert.Assert(t, cmp.Contains(html, `\]</span>`))
	assert.Assert(t, cmp.Contains(html, "a = b"))
}

func TestRenderInlineMath(t *testing.T) {
	html := string(renderDoc(t, "inline $a+b$ math\n").HTML)
	// Inline math emits MathJax \( ... \) markup.
	assert.Assert(t, cmp.Contains(html, `<span class="math inline">\(a+b\)</span>`))
}

func TestRenderHeadingAnchors(t *testing.T) {
	html := string(renderDoc(t, "# Title Here\n\n## Sub Section\n").HTML)
	// AutoHeadingID assigns slug anchors that the TOC links point at.
	assert.Assert(t, cmp.Contains(html, `<h1 id="title-here">Title Here</h1>`))
	assert.Assert(t, cmp.Contains(html, `<h2 id="sub-section">Sub Section</h2>`))
}

func TestRenderTitleAndTOC(t *testing.T) {
	doc := renderDoc(t, "# Title Here\n\n## Sub Section\n### Deeper\n#### Deepest\n##### TooDeep\n")
	// Title is the first level-1 heading.
	assert.Equal(t, doc.Title, "Title Here")
	// TOC covers levels 1-4; the level-5 heading is excluded. IDs come from
	// the same AutoHeadingID pass as the body anchors.
	assert.DeepEqual(t, doc.TOC, []render.Heading{
		{Level: 1, Text: "Title Here", ID: "title-here"},
		{Level: 2, Text: "Sub Section", ID: "sub-section"},
		{Level: 3, Text: "Deeper", ID: "deeper"},
		{Level: 4, Text: "Deepest", ID: "deepest"},
	})
}

func TestRenderTitleFallbackEmpty(t *testing.T) {
	// A document with no level-1 heading yields an empty title so the caller
	// can fall back to the file name.
	doc := renderDoc(t, "## Only A Sub Heading\n\nbody\n")
	assert.Equal(t, doc.Title, "")
	assert.DeepEqual(t, doc.TOC, []render.Heading{
		{Level: 2, Text: "Only A Sub Heading", ID: "only-a-sub-heading"},
	})
}

func TestRenderEmoji(t *testing.T) {
	html := string(renderDoc(t, "hi :smile: there\n").HTML)
	// goldmark-emoji renders :smile: as the U+1F604 HTML entity.
	assert.Assert(t, cmp.Contains(html, "&#x1f604;"))
	assert.Assert(t, !strings.Contains(html, ":smile:"), "emoji shortcode not replaced: %s", html)
}

func TestChromaStylesheets(t *testing.T) {
	sheets, err := render.ChromaStylesheets()
	assert.NilError(t, err)
	for _, name := range render.ChromaStyleNames {
		css, ok := sheets[name]
		assert.Assert(t, ok, "missing stylesheet for %q", name)
		assert.Assert(t, cmp.Contains(css, ".chroma"))
	}
	// The returned map is a copy: mutating it must not affect later calls.
	delete(sheets, render.ChromaStyleNames[0])
	again, err := render.ChromaStylesheets()
	assert.NilError(t, err)
	_, ok := again[render.ChromaStyleNames[0]]
	assert.Assert(t, ok, "cached stylesheets were mutated by caller")
}

func TestChromaCSSUnknownStyle(t *testing.T) {
	_, err := render.ChromaCSS("no-such-style")
	assert.ErrorContains(t, err, "no-such-style")
}
