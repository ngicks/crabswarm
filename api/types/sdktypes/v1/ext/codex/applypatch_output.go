package codex

import (
	"encoding/json"
	"strconv"
	"strings"
)

// SummaryEntry is one file line from an apply_patch success summary
// ("A path", "M path", or "D path").
//
// Source: codex-rs/apply-patch/src/lib.rs (print_summary)
type SummaryEntry struct {
	// Kind is the change kind: add (A), update (M), or delete (D).
	Kind PatchKind `json:"kind"`
	// Path is the file path as printed in the summary.
	Path string `json:"path"`
}

// ApplyPatchOutput is a decoded Codex apply_patch tool_response. On the wire it
// is a bare JSON string produced by Codex's exec-output formatter:
//
//	Exit code: 0
//	Wall time: 0.6 seconds
//	Output:
//	Success. Updated the following files:
//	M path/to/file
//
// The decoded fields expose that structure; the original string round-trips
// byte-for-byte via the preserved raw bytes.
//
// Source: codex-rs/core/src/tools/mod.rs (format_exec_output_for_model),
// codex-rs/apply-patch/src/lib.rs (print_summary)
type ApplyPatchOutput struct {
	// Text is the decoded tool_response string.
	Text string `json:"-"`
	// ExitCode is the patch process exit code, nil when no "Exit code:" header
	// was present.
	ExitCode *int64 `json:"-"`
	// WallTimeSeconds is the reported wall time, nil when absent.
	WallTimeSeconds *float64 `json:"-"`
	// TotalOutputLines is the pre-truncation line count Codex reports only when
	// it truncated output, nil otherwise.
	TotalOutputLines *int64 `json:"-"`
	// Output is the body after the "Output:" header.
	Output string `json:"-"`
	// Summary holds the parsed "Success. Updated the following files:" entries,
	// nil when the body carried no summary.
	Summary []SummaryEntry `json:"-"`

	// raw preserves the original tool_response bytes for a byte-exact re-marshal.
	raw json.RawMessage
}

// RawJSON returns a copy of the original tool_response bytes, or nil for a
// value that wasn't decoded from JSON.
func (o ApplyPatchOutput) RawJSON() json.RawMessage { return cloneRaw(o.raw) }

// MarshalJSON re-emits the original tool_response bytes when present, otherwise
// encodes Text as a JSON string.
func (o ApplyPatchOutput) MarshalJSON() ([]byte, error) {
	if len(o.raw) > 0 {
		return cloneRaw(o.raw), nil
	}
	return json.Marshal(o.Text)
}

// UnmarshalJSON preserves the raw bytes and, when the tool_response is a JSON
// string, decodes and parses it. A non-string value is tolerated (raw is still
// preserved) so a malformed payload never aborts the surrounding hook parse.
func (o *ApplyPatchOutput) UnmarshalJSON(data []byte) error {
	o.raw = cloneRaw(data)
	var s string
	if json.Unmarshal(data, &s) != nil {
		return nil
	}
	o.Text = s
	o.ExitCode, o.WallTimeSeconds, o.TotalOutputLines, o.Output = parseExecFormatted(s)
	o.Summary = parseApplyPatchSummary(o.Output)
	return nil
}

// parseExecFormatted splits Codex's "Exit code: / Wall time: / [Total output
// lines:] / Output:" header block from the output body. Parsing is best-effort:
// missing headers leave their fields nil and the remainder becomes the body.
//
// Source: codex-rs/core/src/tools/mod.rs (format_exec_output_for_model)
func parseExecFormatted(
	text string,
) (exit *int64, wall *float64, totalLines *int64, output string) {
	lines := strings.Split(text, "\n")
	for i := range len(lines) {
		line := lines[i]
		switch {
		case strings.HasPrefix(line, "Exit code: "):
			if n, err := strconv.ParseInt(
				strings.TrimSpace(line[len("Exit code: "):]),
				10,
				64,
			); err == nil {
				exit = &n
			}
		case strings.HasPrefix(line, "Wall time: ") && strings.HasSuffix(line, " seconds"):
			body := strings.TrimSuffix(strings.TrimPrefix(line, "Wall time: "), " seconds")
			if f, err := strconv.ParseFloat(strings.TrimSpace(body), 64); err == nil {
				wall = &f
			}
		case strings.HasPrefix(line, "Total output lines: "):
			if n, err := strconv.ParseInt(
				strings.TrimSpace(line[len("Total output lines: "):]),
				10,
				64,
			); err == nil {
				totalLines = &n
			}
		case line == "Output:":
			output = strings.Join(lines[i+1:], "\n")
			return exit, wall, totalLines, output
		default:
			// A line that isn't a recognized header ends the header block; the
			// remainder (including this line) is the body.
			output = strings.Join(lines[i:], "\n")
			return exit, wall, totalLines, output
		}
	}
	return exit, wall, totalLines, output
}

// parseApplyPatchSummary extracts the "A/M/D path" entries that follow the
// "Success. Updated the following files:" marker, returning nil when the body
// carries no such summary.
//
// Source: codex-rs/apply-patch/src/lib.rs (print_summary)
func parseApplyPatchSummary(output string) []SummaryEntry {
	const marker = "Success. Updated the following files:"
	_, after, found := strings.Cut(output, marker)
	if !found {
		return nil
	}
	rest := strings.TrimPrefix(after, "\n")
	var entries []SummaryEntry
	for line := range strings.Lines(rest) {
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}
		if len(line) < 3 || line[1] != ' ' {
			continue
		}
		var kind PatchKind
		switch line[0] {
		case 'A':
			kind = PatchKindAdd
		case 'M':
			kind = PatchKindUpdate
		case 'D':
			kind = PatchKindDelete
		default:
			continue
		}
		entries = append(entries, SummaryEntry{Kind: kind, Path: line[2:]})
	}
	return entries
}
