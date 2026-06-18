// Package path implements `crabswarm hook path`: hooks that validate the
// file path a tool is about to touch and block the tool call when the path
// breaks a rule. Each subcommand is one such rule. The windows rule is
// per-OS: it blocks paths that collide with a Windows reserved device name
// or otherwise violate Windows file-naming rules, so a repository edited on
// a POSIX host stays checkout-able on Windows. The rules follow Microsoft's
// "Naming Files, Paths, and Namespaces" reference:
// https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file
package path

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
	"github.com/ngicks/crabswarm/pkg/crabswarm/hook"
)

// reservedNames is the set of Windows reserved device names, lower-cased.
// A component collides when its stem (the part before the first dot)
// matches one of these case-insensitively, e.g. CON, con.txt and Aux.tar.gz
// are all reserved.
//
// The set is the canonical list from Microsoft's "Naming Files, Paths, and
// Namespaces" reference: CON, PRN, AUX, NUL, COM1-COM9, LPT1-LPT9, plus the
// ISO/IEC 8859-1 superscript-digit ports COM¹/COM²/COM³ and LPT¹/LPT²/LPT³,
// which Windows treats as digits and therefore reserves as well. See:
// https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file
var reservedNames = func() map[string]struct{} {
	names := []string{"con", "prn", "aux", "nul"}
	// COM#/LPT# ports: ASCII digits 1-9 and, per the reference, the
	// superscript digits ¹ ² ³ (U+00B9, U+00B2, U+00B3).
	for _, d := range []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "¹", "²", "³"} {
		names = append(names, "com"+d, "lpt"+d)
	}
	m := make(map[string]struct{}, len(names))
	for _, n := range names {
		m[n] = struct{}{}
	}
	return m
}()

// reservedChars are the characters Windows forbids in a file name. The
// reference forbids < > : " / \ | ? * (and the 0-31 control range, handled
// separately below). The path separators '/' and '\\' are omitted here:
// '/' never survives component splitting, and '\\' is itself a separator we
// reject structurally below. See:
// https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file
const reservedChars = `<>:"|?*`

// Violation describes one path component that fails a Windows naming rule.
type Violation struct {
	// Path is the full edited file path the component came from.
	Path string
	// Component is the offending path component.
	Component string
	// Reason is a human-readable explanation of the rule it broke.
	Reason string
}

// String renders the violation for the block message shown to the agent.
func (v Violation) String() string {
	return fmt.Sprintf("%s: component %q %s", v.Path, v.Component, v.Reason)
}

// Windows reads a Claude / Codex hook envelope from r and blocks the tool
// call when any edited file path collides with a Windows reserved name or
// violates Windows naming rules. It returns a block-decision
// [handler.HandlerError] listing the violations, or an allow
// [handler.HandlerError] when every path is clean.
func Windows(ctx context.Context, r io.Reader) error {
	raw, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}
	parsed, err := hook.Parse(raw)
	if err != nil {
		return err
	}

	violations := CheckWindows(parsed.Cwd, parsed.Files)
	if len(violations) == 0 {
		return &handler.HandlerError{}
	}
	return handler.Block(blockReason(violations))
}

// CheckWindows reports every Windows naming violation across files. Each
// path is checked component-by-component relative to cwd; components above
// cwd (the host prefix outside the repository) and unrelated paths fall
// back to checking just the base name, so a host directory like
// /home/aux/... is not flagged while an in-repo aux/ directory is.
func CheckWindows(cwd string, files []string) []Violation {
	var violations []Violation
	for _, f := range files {
		for _, component := range componentsToCheck(cwd, f) {
			if reason := checkComponent(component); reason != "" {
				violations = append(violations, Violation{
					Path:      f,
					Component: component,
					Reason:    reason,
				})
			}
		}
	}
	return violations
}

// componentsToCheck returns the path components of p that belong to the
// repository: the components of p relative to cwd when p is within cwd,
// otherwise just the base name.
func componentsToCheck(cwd, p string) []string {
	if cwd != "" {
		if rel, err := filepath.Rel(cwd, p); err == nil &&
			rel != "." && !strings.HasPrefix(rel, "..") {
			return strings.Split(rel, string(filepath.Separator))
		}
	}
	base := filepath.Base(p)
	if base == "." || base == string(filepath.Separator) {
		return nil
	}
	return []string{base}
}

// checkComponent returns a non-empty reason when component breaks a Windows
// file-naming rule, or "" when it is acceptable.
func checkComponent(component string) string {
	if component == "" || component == "." || component == ".." {
		return ""
	}

	stem := component
	if i := strings.IndexByte(stem, '.'); i >= 0 {
		stem = stem[:i]
	}
	if _, ok := reservedNames[strings.ToLower(stem)]; ok {
		return fmt.Sprintf(
			"collides with the Windows reserved device name %q",
			strings.ToUpper(stem),
		)
	}

	if i := strings.IndexAny(component, reservedChars); i >= 0 {
		return fmt.Sprintf("contains the Windows-reserved character %q", string(component[i]))
	}
	if strings.ContainsRune(component, '\\') {
		return `contains a backslash, which Windows treats as a path separator`
	}
	for _, r := range component {
		if unicode.IsControl(r) {
			return fmt.Sprintf("contains the control character U+%04X", r)
		}
	}

	if last := component[len(component)-1]; last == ' ' || last == '.' {
		return "ends with a space or period, which Windows silently trims"
	}

	return ""
}

// blockReason formats the violations as the hook's block reason. The agent
// reads this text to learn why the edit was rejected.
func blockReason(violations []Violation) string {
	var sb strings.Builder
	sb.WriteString("blocked: path is not valid on Windows\n")
	for _, v := range violations {
		fmt.Fprintf(&sb, "  %s\n", v)
	}
	return sb.String()
}
