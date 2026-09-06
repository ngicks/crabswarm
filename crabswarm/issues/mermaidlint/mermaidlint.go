// Package mermaidlint validates the mermaid diagrams carried by issue text
// by running the mermaid-lint command line tool over it.
//
// Markdown files are guarded by running mermaid-lint on the file itself.
// Issue text lives in a beads database instead of in a file, so [Lint]
// writes each text field and each comment that holds a mermaid fence to a
// temp file, runs mermaid-lint once over the whole set and maps its report
// back to the issue, field, comment and line the text came from. One
// parser therefore judges diagrams in files and diagrams in issues alike.
package mermaidlint

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ngicks/crabswarm/crabswarm/issues"
)

// defaultBinary is the mermaid-lint executable looked up on PATH when the
// caller passes no [WithBinary].
const defaultBinary = "mermaid-lint"

// The values [Finding.Field] takes. The first four name a text field of an
// issue; FieldComment marks text that came from a comment, whose position
// in the issue is [Finding.Comment].
const (
	FieldDescription        = "description"
	FieldDesign             = "design"
	FieldAcceptanceCriteria = "acceptance_criteria"
	FieldNotes              = "notes"
	FieldComment            = "comment"
)

// ErrBinaryNotFound reports that the mermaid-lint executable could not be
// resolved. Callers match it with [errors.Is] to tell "the tool is not
// installed" apart from a diagram that failed to parse.
var ErrBinaryNotFound = errors.New("mermaid-lint not found")

// Finding is one mermaid diagram that mermaid-lint refused, located in the
// issue text it was written in.
type Finding struct {
	IssueID string
	// Field is the text field the diagram sits in: one of
	// [FieldDescription], [FieldDesign], [FieldAcceptanceCriteria],
	// [FieldNotes] or [FieldComment].
	Field string
	// Comment is the 1-based position of the comment among the issue's
	// comments, zero unless Field is [FieldComment].
	Comment int
	// Line and Col locate the failure inside that text, counted from its
	// first line.
	Line int
	Col  int
	// Type is the diagram type mermaid-lint detected, "unknown" when it
	// could not read one.
	Type string
	// Message is mermaid-lint's account of the failure.
	Message string
}

// linter holds what [Lint] resolves from its options.
type linter struct {
	bin string
}

// Option configures a [Lint] call.
type Option func(*linter)

// WithBinary overrides the mermaid-lint executable. The default,
// "mermaid-lint", is looked up on PATH.
func WithBinary(path string) Option {
	return func(l *linter) { l.bin = path }
}

// fenceOpener matches a line opening a mermaid fence: any indentation,
// three or more backticks or tildes, then the word mermaid. It is
// deliberately looser than mermaid-lint's own reader — case and spacing do
// not matter here — because it only decides whether text is worth writing
// out, and text mermaid-lint finds no diagram in costs one temp file.
var fenceOpener = regexp.MustCompile("(?im)^[ \t]*(?:`{3,}|~{3,})[ \t]*mermaid\\b")

// HasFence reports whether text opens a mermaid fence anywhere, so that a
// caller listing a backlog can drop the issues [Lint] has nothing to do
// for. A fence that is never closed still counts: mermaid-lint reports the
// missing terminator as a failure of its own.
func HasFence(text string) bool {
	return fenceOpener.MatchString(text)
}

// unit is one piece of issue text about to be written out for linting.
type unit struct {
	name    string // base name of the temp file holding text
	text    string
	issueID string
	field   string
	comment int
}

// Lint runs mermaid-lint over every mermaid diagram in the text of list
// and returns the diagrams it refused. Text carrying no mermaid fence is
// skipped before any file is written, so a set of issues without diagrams
// needs no mermaid-lint at all. Findings come back in the order the
// issues, their fields and their comments were passed.
//
// A missing executable is reported as an error wrapping
// [ErrBinaryNotFound]. Diagrams that fail to parse are not an error: they
// are the result.
func Lint(ctx context.Context, list []issues.Issue, opts ...Option) ([]Finding, error) {
	l := &linter{bin: defaultBinary}
	for _, o := range opts {
		o(l)
	}

	units, err := plan(list)
	if err != nil {
		return nil, err
	}
	if len(units) == 0 {
		return nil, nil
	}

	bin, err := exec.LookPath(l.bin)
	if err != nil {
		return nil, fmt.Errorf("%s: %w: %w", l.bin, ErrBinaryNotFound, err)
	}

	dir, err := os.MkdirTemp("", "crabswarm-mermaidlint-")
	if err != nil {
		return nil, fmt.Errorf("creating temp directory for issue text: %w", err)
	}
	defer os.RemoveAll(dir)

	names := make([]string, len(units))
	for i, u := range units {
		text := u.text
		// mermaid-lint reads the last line of a file whether or not it
		// ends in a newline; the newline only keeps the file well-formed.
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		if err := os.WriteFile(filepath.Join(dir, u.name), []byte(text), 0o600); err != nil {
			return nil, fmt.Errorf("writing issue text: %w", err)
		}
		names[i] = u.name
	}

	rep, err := run(ctx, bin, dir, names)
	if err != nil {
		return nil, err
	}
	return findings(rep, units)
}

// plan turns the issues into the files to write, one per text that holds a
// mermaid fence, in the order the report is expected to read.
func plan(list []issues.Issue) ([]unit, error) {
	var units []unit
	seen := make(map[string]struct{}, len(list))
	for _, iss := range list {
		// The ID names the file, so one that is not a bare name would
		// write outside the temp directory, and a repeated one would hide
		// the findings of the issue it collides with.
		if iss.ID == "" || iss.ID != filepath.Base(iss.ID) {
			return nil, fmt.Errorf("issue ID %q cannot name a file", iss.ID)
		}
		if _, dup := seen[iss.ID]; dup {
			return nil, fmt.Errorf("issue %s given twice", iss.ID)
		}
		seen[iss.ID] = struct{}{}

		for _, f := range []struct {
			name string
			text string
		}{
			{FieldDescription, iss.Description},
			{FieldDesign, iss.Design},
			{FieldAcceptanceCriteria, iss.AcceptanceCriteria},
			{FieldNotes, iss.Notes},
		} {
			if !HasFence(f.text) {
				continue
			}
			units = append(units, unit{
				name:    fmt.Sprintf("%s.%s.md", iss.ID, f.name),
				text:    f.text,
				issueID: iss.ID,
				field:   f.name,
			})
		}
		for i, c := range iss.Comments {
			if !HasFence(c.Text) {
				continue
			}
			units = append(units, unit{
				name:    fmt.Sprintf("%s.comment-%d.md", iss.ID, i+1),
				text:    c.Text,
				issueID: iss.ID,
				field:   FieldComment,
				comment: i + 1,
			})
		}
	}
	return units, nil
}
