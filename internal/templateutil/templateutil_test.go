package templateutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"gotest.tools/v3/assert"
)

func TestFuncMap_ExposesExpectedFuncs(t *testing.T) {
	fm := FuncMap()
	names := []string{"env", "basename", "dirname", "ext", "trim", "quote", "quoteJoin", "which"}
	for _, name := range names {
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
		`|{{ trim .Spaced }}|{{ env "CRABSWARM_TEMPLATEUTIL_TEST" }}|{{ quote .Quoted }}` +
		`|{{ quoteJoin .Args }}`
	tmpl := template.Must(template.New("t").Funcs(FuncMap()).Parse(src))
	var buf strings.Builder
	assert.NilError(t, tmpl.Execute(&buf, map[string]any{
		"Path":   "/a/b/c.go",
		"Spaced": "  pad  ",
		"Quoted": "it's",
		"Args":   []string{"one", "two words"},
	}))
	assert.Equal(t, buf.String(), `c.go|/a/b|.go|pad|xyz|'it'\''s'|"one" "two words"`)
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

func TestFuncDocs_MatchesFuncMap(t *testing.T) {
	fm := FuncMap()
	docs := FuncDocs()

	if len(docs) != len(fm) {
		t.Fatalf("FuncDocs has %d entries, FuncMap has %d", len(docs), len(fm))
	}
	seen := map[string]bool{}
	for _, d := range docs {
		if _, ok := fm[d.Name]; !ok {
			t.Errorf("FuncDocs documents %q which is not in FuncMap", d.Name)
		}
		if seen[d.Name] {
			t.Errorf("FuncDocs has duplicate entry for %q", d.Name)
		}
		seen[d.Name] = true
		if d.Usage == "" || d.Desc == "" {
			t.Errorf("FuncDocs entry %q has empty Usage or Desc", d.Name)
		}
	}
	for name := range fm {
		if !seen[name] {
			t.Errorf("FuncMap has %q with no FuncDocs entry", name)
		}
	}
}

func TestFuncHelp_RendersEveryFunc(t *testing.T) {
	help := FuncHelp()
	if !strings.HasSuffix(help, "\n") {
		t.Error("FuncHelp should end with a trailing newline")
	}
	for _, d := range FuncDocs() {
		if !strings.Contains(help, d.Usage) {
			t.Errorf("FuncHelp missing usage %q", d.Usage)
		}
		if !strings.Contains(help, d.Desc) {
			t.Errorf("FuncHelp missing desc for %q", d.Name)
		}
	}
}

func TestQuoteJoin(t *testing.T) {
	for _, tc := range []struct {
		in   []string
		want string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"a"}, `"a"`},
		{[]string{"a", "b", "c"}, `"a" "b" "c"`},
		{[]string{"with space", "two words"}, `"with space" "two words"`},
		{[]string{`say "hi"`}, `"say \"hi\""`},
		{[]string{`back\slash`}, `"back\\slash"`},
		{[]string{""}, `""`},
	} {
		assert.Equal(t, QuoteJoin(tc.in), tc.want)
	}
}

func TestWhich(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "crabswarm-which-tool")
	assert.NilError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755))
	t.Setenv("PATH", dir)

	got, err := Which("crabswarm-which-tool")
	assert.NilError(t, err)
	assert.Equal(t, got, bin)
	assert.Assert(t, filepath.IsAbs(got))
}

func TestWhich_NotFoundErrors(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := Which("crabswarm-no-such-command")
	assert.Assert(t, err != nil)
}
