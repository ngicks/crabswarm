package cli

import (
	"bytes"
	"errors"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/crabswarm/issues/mermaidlint"
)

// findings covers the two shapes of location a finding takes: a text field
// of the issue, and one of its comments.
var findings = []mermaidlint.Finding{
	{
		IssueID: "plan-aaa",
		Field:   mermaidlint.FieldDescription,
		Line:    12,
		Col:     4,
		Type:    "flowchart",
		Message: "expected `&`, found the end of the diagram",
	},
	{
		IssueID: "plan-bbb",
		Field:   mermaidlint.FieldComment,
		Comment: 2,
		Line:    2,
		Col:     1,
		Type:    "unknown",
		Message: "unclosed ```mermaid fence (no closing ``` found)",
	},
}

func TestRenderFindings(t *testing.T) {
	var buf bytes.Buffer
	assert.NilError(t, RenderFindings(&buf, findings))
	assert.Equal(t, buf.String(),
		"plan-aaa description:12:4: expected `&`, found the end of the diagram\n"+
			"plan-bbb comment#2:2:1: unclosed ```mermaid fence (no closing ``` found)\n")
}

func TestRenderFindingsNone(t *testing.T) {
	var buf bytes.Buffer
	assert.NilError(t, RenderFindings(&buf, nil))
	assert.Equal(t, buf.String(), "")
}

func TestRenderFindingsJSON(t *testing.T) {
	var buf bytes.Buffer
	assert.NilError(t, RenderFindingsJSON(&buf, findings))
	assert.Equal(t, buf.String(), `[
  {
    "issue_id": "plan-aaa",
    "field": "description",
    "line": 12,
    "col": 4,
    "type": "flowchart",
    "message": "expected `+"`&`"+`, found the end of the diagram"
  },
  {
    "issue_id": "plan-bbb",
    "field": "comment",
    "comment": 2,
    "line": 2,
    "col": 1,
    "type": "unknown",
    "message": "unclosed `+"```"+`mermaid fence (no closing `+"```"+` found)"
  }
]
`)
}

func TestRenderFindingsJSONNone(t *testing.T) {
	var buf bytes.Buffer
	// A nil slice would encode as null; a consumer decoding an array must
	// not have to special-case the empty sweep.
	assert.NilError(t, RenderFindingsJSON(&buf, nil))
	assert.Equal(t, buf.String(), "[]\n")
}

// failingWriter stands for a closed pipe: the writer the CLI renders into
// can fail, and a renderer that swallowed that would report a clean sweep.
type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestRenderFindingsWriteError(t *testing.T) {
	boom := errors.New("pipe closed")
	assert.ErrorIs(t, RenderFindings(failingWriter{err: boom}, findings), boom)
	assert.ErrorIs(t, RenderFindingsJSON(failingWriter{err: boom}, findings), boom)
}
