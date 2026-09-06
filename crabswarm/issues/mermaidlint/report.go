package mermaidlint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// report is the document `mermaid-lint --format json` prints. Only what
// the mapping needs is decoded.
type report struct {
	Files []reportFile `json:"files"`
}

type reportFile struct {
	Path     string    `json:"path"`
	Diagrams []diagram `json:"diagrams"`
}

type diagram struct {
	Type string `json:"type"`
	// Error is absent for a diagram that parsed. The report also carries
	// the position of the fence and any semantic warnings; a finding needs
	// neither.
	Error *diagramError `json:"error"`
}

type diagramError struct {
	Message string `json:"message"`
	// Line and Col are counted in the file, not in the diagram.
	Line int `json:"line"`
	Col  int `json:"col"`
}

// run lints names inside dir and decodes the report.
func run(ctx context.Context, bin, dir string, names []string) (report, error) {
	// The files are named relatively with the working directory set to
	// their own, so the report echoes back exactly the names that were
	// written and no configuration around the caller's directory applies.
	cmd := exec.CommandContext(
		ctx,
		bin,
		append([]string{"--format", "json", "--quiet"}, names...)...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	// mermaid-lint exits 1 whenever it refused something, so the exit
	// status says nothing about whether the run itself worked. Output that
	// does not decode is the signal that it did not; a usage or IO failure
	// prints its message on stdout in place of the report.
	var rep report
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if runErr != nil {
			if msg != "" {
				return report{}, fmt.Errorf("running mermaid-lint: %w: %s", runErr, msg)
			}
			return report{}, fmt.Errorf("running mermaid-lint: %w", runErr)
		}
		return report{}, fmt.Errorf("decoding mermaid-lint report: %w: %s", err, msg)
	}
	return rep, nil
}

// findings maps the report back onto the text each file was written from.
func findings(rep report, units []unit) ([]Finding, error) {
	byName := make(map[string][]diagram, len(rep.Files))
	known := make(map[string]struct{}, len(units))
	for _, u := range units {
		known[u.name] = struct{}{}
	}
	for _, f := range rep.Files {
		name := filepath.Base(f.Path)
		if _, ok := known[name]; !ok {
			return nil, fmt.Errorf(
				"mermaid-lint reported %q, which is not issue text it was given",
				f.Path,
			)
		}
		byName[name] = f.Diagrams
	}

	var out []Finding
	for _, u := range units {
		for _, d := range byName[u.name] {
			if d.Error == nil {
				continue
			}
			out = append(out, Finding{
				IssueID: u.issueID,
				Field:   u.field,
				Comment: u.comment,
				// mermaid-lint counts the failure in the file, so the
				// position is already the one inside the text the file was
				// written from.
				Line:    d.Error.Line,
				Col:     d.Error.Col,
				Type:    d.Type,
				Message: d.Error.Message,
			})
		}
	}
	return out, nil
}
