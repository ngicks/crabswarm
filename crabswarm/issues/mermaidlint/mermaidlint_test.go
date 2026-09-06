package mermaidlint

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/crabswarm/issues"
)

// The diagrams the tests are written around: one the mermaid parser
// accepts and three it refuses, each in a different way.
const (
	goodFence     = "```mermaid\nflowchart TD\n  A --> B\n```\n"
	badFence      = "```mermaid\nflowchart TD\n  A -->\n```\n"
	unclosedFence = "```mermaid\nflowchart TD\n  A --> B\n"
	unknownFence  = "  ```mermaid\n  notadiagram foo\n  ```\n"
)

// linted are the issues testdata/report.json was recorded from — two
// carrying diagrams and one carrying none. The report is what the real
// mermaid-lint printed for exactly these texts, so the positions the test
// expects are the ones the tool reports for them.
func linted() []issues.Issue {
	return []issues.Issue{
		{
			Summary: issues.Summary{
				ID:          "plan-aaa",
				Title:       "two diagrams in the description",
				Description: "how it works\n\n" + goodFence + "\nand the other half\n\n" + badFence,
				// A field without a fence is never written out.
				Notes: "nothing to draw here",
			},
			Comments: []issues.Comment{
				{Text: "plain prose, no diagram"},
				{Text: "revised:\n" + unclosedFence},
			},
		},
		{
			Summary: issues.Summary{
				ID:                 "plan-bbb",
				Title:              "a diagram in the design",
				Design:             goodFence,
				AcceptanceCriteria: "the diagram parses",
			},
			Comments: []issues.Comment{
				{Text: "still wrong:\n\n- see:\n\n" + unknownFence},
			},
		},
		{
			Summary: issues.Summary{
				ID:          "plan-ccc",
				Title:       "no diagrams at all",
				Description: "prose only",
			},
			Comments: []issues.Comment{{Text: "prose only"}},
		},
	}
}

func TestLint(t *testing.T) {
	invocations := installFakeMermaidLint(t, "report.json")

	got, err := Lint(t.Context(), linted())
	assert.NilError(t, err)

	inv := invocations()
	assert.Equal(t, len(inv), 1)
	// One run covers every text: the files go in named relatively, in the
	// order the issues, their fields and their comments were passed.
	assert.Equal(
		t,
		inv[0].args,
		"--format json --quiet plan-aaa.description.md plan-aaa.comment-2.md"+
			" plan-bbb.design.md plan-bbb.comment-1.md",
	)
	// Text with no fence never reaches the disk, so neither the issue
	// without diagrams nor the fenceless field and comment of the others
	// have a file.
	assert.Equal(
		t,
		inv[0].files,
		"plan-aaa.comment-2.md plan-aaa.description.md"+
			" plan-bbb.comment-1.md plan-bbb.design.md",
	)
	// The directory holding them is gone once Lint returns.
	_, statErr := os.Stat(inv[0].dir)
	assert.Assert(t, errors.Is(statErr, fs.ErrNotExist), "temp dir left behind: %v", statErr)

	assert.DeepEqual(t, got, lintedFindings)
}

// lintedFindings is what [linted] amounts to: the parse failure at the
// second diagram of a description, an unclosed fence in a comment and a
// diagram type mermaid cannot read in another. The failure is placed where
// mermaid-lint puts it rather than at the fence — the last one opens at
// column 3 and is reported at column 1.
var lintedFindings = []Finding{
	{
		IssueID: "plan-aaa",
		Field:   FieldDescription,
		Line:    12,
		Col:     4,
		Type:    "flowchart",
		Message: "expected `&`, `:`, `|`, `v`, `default`, or 8 more," +
			" found the end of the diagram",
	},
	{
		IssueID: "plan-aaa",
		Field:   FieldComment,
		Comment: 2,
		Line:    2,
		Col:     1,
		Type:    "unknown",
		Message: "unclosed ```mermaid fence (no closing ``` found)",
	},
	{
		IssueID: "plan-bbb",
		Field:   FieldComment,
		Comment: 1,
		Line:    5,
		Col:     1,
		Type:    "notadiagram",
		Message: "unknown diagram type `notadiagram`",
	},
}

func TestLintNoFences(t *testing.T) {
	invocations := installFakeMermaidLint(t, "report.json")

	// The binary is never resolved when nothing has a diagram, so a name
	// that cannot be found on PATH is no obstacle.
	got, err := Lint(
		t.Context(),
		[]issues.Issue{{
			Summary:  issues.Summary{ID: "plan-ccc", Description: "prose only"},
			Comments: []issues.Comment{{Text: "prose only"}},
		}},
		WithBinary("crabswarm-no-such-mermaid-lint"),
	)
	assert.NilError(t, err)
	assert.Equal(t, len(got), 0)
	assert.Equal(t, len(invocations()), 0)
}

func TestLintWithBinary(t *testing.T) {
	installFakeMermaidLint(t, "report.json")

	// Reach the fake by path rather than by PATH lookup, under a name
	// mermaid-lint would never be found as.
	dir := t.TempDir()
	bin := filepath.Join(dir, "not-mermaid-lint")
	assert.NilError(t, os.WriteFile(bin, []byte(fakeMermaidLintScript), 0o755))

	got, err := Lint(t.Context(), linted(), WithBinary(bin))
	assert.NilError(t, err)
	assert.Equal(t, len(got), 3)
}

func TestLintBinaryNotFound(t *testing.T) {
	for _, tc := range []struct {
		name string
		bin  string
		is   error
	}{
		// A bare name fails the PATH lookup itself, a path fails on the
		// file: only the package's own sentinel covers both.
		{name: "on PATH", bin: "crabswarm-no-such-mermaid-lint", is: exec.ErrNotFound},
		{name: "by path", bin: "/nonexistent/crabswarm-no-such-mermaid-lint", is: os.ErrNotExist},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Lint(t.Context(), linted(), WithBinary(tc.bin))
			assert.ErrorIs(t, err, ErrBinaryNotFound)
			assert.ErrorIs(t, err, tc.is)
		})
	}
}

func TestLintOutputNotAReport(t *testing.T) {
	installFakeMermaidLint(t, "")
	t.Setenv("FAKE_MERMAID_LINT_MESSAGE", "no files matched the given paths")
	t.Setenv("FAKE_MERMAID_LINT_EXIT", "2")

	_, err := Lint(t.Context(), linted())
	assert.Assert(t, err != nil)
	// The message the binary printed in place of a report reaches the
	// caller; mermaid-lint puts it on stdout, not on stderr.
	assert.Assert(
		t,
		strings.Contains(err.Error(), "no files matched the given paths"),
		"got %v",
		err,
	)
}

func TestLintReportsUnknownFile(t *testing.T) {
	installFakeMermaidLint(t, "report_unknown_file.json")

	_, err := Lint(t.Context(), linted())
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "somewhere-else.md"), "got %v", err)
}

func TestLintRejectsUnusableIDs(t *testing.T) {
	invocations := installFakeMermaidLint(t, "report.json")

	for _, tc := range []struct {
		name string
		list []issues.Issue
		want string
	}{
		{
			name: "empty",
			list: []issues.Issue{{Summary: issues.Summary{Description: badFence}}},
			want: `issue ID "" cannot name a file`,
		},
		{
			name: "path",
			list: []issues.Issue{{
				Summary: issues.Summary{ID: "../plan-aaa", Description: badFence},
			}},
			want: `issue ID "../plan-aaa" cannot name a file`,
		},
		{
			name: "repeated",
			list: []issues.Issue{
				{Summary: issues.Summary{ID: "plan-aaa", Description: badFence}},
				{Summary: issues.Summary{ID: "plan-aaa", Notes: badFence}},
			},
			want: "issue plan-aaa given twice",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Lint(t.Context(), tc.list)
			assert.ErrorContains(t, err, tc.want)
		})
	}
	// Nothing ran: the names are rejected before any file is written.
	assert.Equal(t, len(invocations()), 0)
}

func TestLintRealBinary(t *testing.T) {
	if _, err := exec.LookPath(defaultBinary); err != nil {
		t.Skipf("%s not on PATH: %v", defaultBinary, err)
	}

	got, err := Lint(t.Context(), linted())
	assert.NilError(t, err)

	// The same issues the report in testdata was recorded from, judged by
	// the installed mermaid-lint: what the fake replays is what the tool
	// says, positions and messages included.
	assert.DeepEqual(t, got, lintedFindings)
}

func TestHasFence(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		want bool
	}{
		{name: "empty"},
		{name: "prose", text: "the mermaid diagram is described in words"},
		{name: "another language", text: "```go\nfunc main() {}\n```\n"},
		{name: "bare fence", text: "```\nplain text\n```\n"},
		{name: "mid line", text: "the opener is written ```mermaid on its own line\n"},
		{name: "longer word", text: "```mermaidish\nnot a diagram\n```\n"},

		{name: "alone", text: "```mermaid\nflowchart TD\n  A --> B\n```", want: true},
		{name: "after prose", text: "how it works\n\n" + goodFence, want: true},
		{name: "tildes", text: "~~~mermaid\nflowchart TD\n  A --> B\n~~~\n", want: true},
		{name: "indented", text: "- item:\n\n  ```mermaid\n  flowchart TD\n  ```\n", want: true},
		{name: "more backticks", text: "````mermaid\nflowchart TD\n````\n", want: true},
		{name: "info string", text: "```mermaid title=\"x\"\nflowchart TD\n```\n", want: true},
		// An unclosed fence counts: mermaid-lint reports the missing
		// terminator itself.
		{name: "unclosed", text: "```mermaid\nflowchart TD\n  A --> B\n", want: true},
		// mermaid-lint reads neither of these as a diagram. The filter
		// still lets them through, because writing a file that turns out
		// to hold nothing is cheaper than missing a broken diagram.
		{name: "uppercase", text: "```MERMAID\nflowchart TD\n```\n", want: true},
		{name: "spaced", text: "``` mermaid\nflowchart TD\n```\n", want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, HasFence(tc.text), tc.want)
		})
	}
}
