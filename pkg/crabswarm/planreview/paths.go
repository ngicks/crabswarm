// Package planreview provides plan iteration review functionality for Claude Code hooks.
package planreview

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// PlanDirName returns the directory name for a plan iteration directory.
// Format: {RFC3339}-{planName}
func PlanDirName(t time.Time, planName string) string {
	return t.Format(time.RFC3339) + "-" + planName
}

// IntermediateFileName returns the intermediate file name for a plan iteration step.
// Format: {NNN}_{SS}_{suffix}.md
func IntermediateFileName(iteration, step int, suffix string) string {
	return fmt.Sprintf("%03d_%02d_%s.md", iteration, step, suffix)
}

var whitespaceRe = regexp.MustCompile(`\s+`)

// DerivePlanName reads the plan file and derives a kebab-case name from its
// first non-empty line. The line is lowercased, leading '#' markers are stripped,
// and whitespace runs are replaced with '-'.
// Returns an error if the file cannot be read or contains no non-empty lines.
func DerivePlanName(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open plan file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Strip leading markdown heading markers.
		line = strings.TrimLeft(line, "#")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name := strings.ToLower(line)
		name = whitespaceRe.ReplaceAllString(name, "-")
		return name, nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading plan file: %w", err)
	}
	return "", fmt.Errorf("plan file has no non-empty lines: %s", filePath)
}

// PathWithinDir checks whether filePath is within dirPath after canonicalization.
// It handles the case where filePath may not exist yet (PreToolUse scenario)
// by resolving the longest existing ancestor and appending the remaining suffix.
func PathWithinDir(filePath, dirPath string) (bool, error) {
	absFile, err := filepath.Abs(filePath)
	if err != nil {
		return false, fmt.Errorf("abs file path: %w", err)
	}
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return false, fmt.Errorf("abs dir path: %w", err)
	}

	// Resolve symlinks on dirPath (must exist).
	realDir, err := filepath.EvalSymlinks(absDir)
	if err != nil {
		return false, fmt.Errorf("eval symlinks on dir: %w", err)
	}

	// For filePath, resolve the longest existing ancestor then append the rest.
	realFile, err := resolvePartial(absFile)
	if err != nil {
		return false, fmt.Errorf("resolving file path: %w", err)
	}

	// Check containment: filePath == dirPath or filePath starts with dirPath + separator.
	if realFile == realDir {
		return true, nil
	}
	return strings.HasPrefix(realFile, realDir+string(filepath.Separator)), nil
}

// resolvePartial resolves symlinks on the longest existing ancestor of path,
// then appends the remaining suffix. This handles files that don't exist yet.
func resolvePartial(path string) (string, error) {
	// Try to resolve the full path first.
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, nil
	}

	// Walk up to find the longest existing ancestor.
	remaining := ""
	current := path
	for {
		parent := filepath.Dir(current)
		if parent == current {
			// Reached root without finding existing path.
			return path, nil
		}
		base := filepath.Base(current)
		if remaining == "" {
			remaining = base
		} else {
			remaining = base + string(filepath.Separator) + remaining
		}
		current = parent

		resolved, err = filepath.EvalSymlinks(current)
		if err == nil {
			return filepath.Join(resolved, remaining), nil
		}
	}
}

// CountIterations counts the number of plan iterations in an intermediate directory
// by globbing for *_00_PLAN.md files.
func CountIterations(intermediateDir string) (int, error) {
	matches, err := filepath.Glob(filepath.Join(intermediateDir, "*_00_PLAN.md"))
	if err != nil {
		return 0, fmt.Errorf("glob intermediate dir: %w", err)
	}
	return len(matches), nil
}

// EnsureDir creates the directory and all parents if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
