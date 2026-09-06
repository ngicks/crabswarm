package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/ngicks/crabswarm/crabswarm/issues/mermaidlint"
)

// RenderFindings writes one line per refused mermaid diagram to w:
//
//	<issue-id> <field>[#<comment-n>]:<line>:<col>: <message>
//
// The position after the field is the one inside that text, so an editor
// opening the issue lands on the diagram. Nothing is written when there is
// nothing to report.
func RenderFindings(w io.Writer, findings []mermaidlint.Finding) error {
	for _, f := range findings {
		_, err := fmt.Fprintf(
			w, "%s %s:%d:%d: %s\n",
			f.IssueID, findingField(f), f.Line, f.Col, f.Message,
		)
		if err != nil {
			return fmt.Errorf("writing lint findings: %w", err)
		}
	}
	return nil
}

// findingField names the text the diagram sits in: the field, plus the
// position of the comment when the text is one.
func findingField(f mermaidlint.Finding) string {
	if f.Field == mermaidlint.FieldComment {
		return f.Field + "#" + strconv.Itoa(f.Comment)
	}
	return f.Field
}

// RenderFindingsJSON writes the findings to w as a JSON array, one object
// per finding. A sweep that found nothing writes an empty array, not null,
// so a consumer can decode the output the same way either way.
func RenderFindingsJSON(w io.Writer, findings []mermaidlint.Finding) error {
	if findings == nil {
		findings = []mermaidlint.Finding{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	// A parser message quotes the diagram's own syntax, where &, < and >
	// are everyday characters. The output is read as JSON, never embedded
	// in a page, so escaping them only makes the message harder to read.
	enc.SetEscapeHTML(false)
	if err := enc.Encode(findings); err != nil {
		return fmt.Errorf("encoding lint findings: %w", err)
	}
	return nil
}
