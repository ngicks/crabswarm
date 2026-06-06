package templateutil

import (
	"strings"
	"testing"
	"text/template"

	"gotest.tools/v3/assert"
)

func TestFuncMap_ExposesExpectedFuncs(t *testing.T) {
	fm := FuncMap()
	for _, name := range []string{"env", "basename", "dirname", "ext", "trim", "quote"} {
		if _, ok := fm[name]; !ok {
			t.Errorf("FuncMap missing %q", name)
		}
	}
}

func TestFuncMap_FreshMapPerCall(t *testing.T) {
	a := FuncMap()
	a["extra"] = strings.ToUpper
	b := FuncMap()
	if _, ok := b["extra"]; ok {
		t.Fatal("FuncMap returned a shared map; mutation leaked across calls")
	}
}

func TestFuncMap_RendersThroughTemplate(t *testing.T) {
	t.Setenv("CRABSWARM_TEMPLATEUTIL_TEST", "xyz")
	src := `{{ basename .Path }}|{{ dirname .Path }}|{{ ext .Path }}` +
		`|{{ trim .Spaced }}|{{ env "CRABSWARM_TEMPLATEUTIL_TEST" }}|{{ quote .Quoted }}`
	tmpl := template.Must(template.New("t").Funcs(FuncMap()).Parse(src))
	var buf strings.Builder
	assert.NilError(t, tmpl.Execute(&buf, map[string]string{
		"Path":   "/a/b/c.go",
		"Spaced": "  pad  ",
		"Quoted": "it's",
	}))
	assert.Equal(t, buf.String(), `c.go|/a/b|.go|pad|xyz|'it'\''s'`)
}

func TestShellQuote(t *testing.T) {
	for _, tc := range []struct {
		in, want string
	}{
		{"plain", `'plain'`},
		{"with space", `'with space'`},
		{"it's", `'it'\''s'`},
		{"", `''`},
	} {
		assert.Equal(t, ShellQuote(tc.in), tc.want)
	}
}
