package planreview

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// transcriptEntry represents a single line in the JSONL transcript.
// We only parse the fields we need to find Write events.
type transcriptEntry struct {
	Type      string          `json:"type"`
	ToolName  string          `json:"tool_name,omitempty"`
	ToolInput json.RawMessage `json:"tool_input,omitempty"`
	// tool_use nested structure
	Message *transcriptMessage `json:"message,omitempty"`
}

type transcriptMessage struct {
	Content []transcriptContent `json:"content,omitempty"`
}

type transcriptContent struct {
	Type  string          `json:"type"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type writeInput struct {
	FilePath string `json:"file_path"`
}

// ResolveLastPlanPath parses the JSONL transcript file to find the last Write event
// whose file_path is under plansDir. Returns the resolved plan file path or empty string
// if none found. Malformed JSONL lines are skipped with a warning to stderr.
func ResolveLastPlanPath(transcriptPath, plansDir string) (string, error) {
	f, err := os.Open(transcriptPath)
	if err != nil {
		return "", fmt.Errorf("open transcript: %w", err)
	}
	defer f.Close()

	var lastPlanPath string

	scanner := bufio.NewScanner(f)
	// Increase scanner buffer for potentially large lines.
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Try to extract Write tool_input.file_path from various transcript formats.
		path := extractWritePath(line)
		if path == "" {
			continue
		}

		// Check if the path is under plansDir.
		under, err := PathWithinDir(path, plansDir)
		if err != nil {
			log.Printf("warning: line %d: checking path containment: %v", lineNum, err)
			continue
		}
		if under && strings.HasSuffix(path, ".md") {
			lastPlanPath = path
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scanning transcript: %w", err)
	}

	return lastPlanPath, nil
}

// extractWritePath tries to extract a file_path from a Write tool invocation
// in a transcript JSONL line. Returns empty string if not a Write event.
func extractWritePath(line string) string {
	// Fast pre-check: the line must contain "Write" somewhere.
	if !strings.Contains(line, "Write") {
		return ""
	}

	var entry transcriptEntry
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		log.Printf("warning: skipping malformed JSONL line: %v", err)
		return ""
	}

	// Format 1: Direct tool_name + tool_input at top level.
	if entry.ToolName == "Write" && len(entry.ToolInput) > 0 {
		var wi writeInput
		if err := json.Unmarshal(entry.ToolInput, &wi); err == nil && wi.FilePath != "" {
			return wi.FilePath
		}
	}

	// Format 2: Nested in message.content[].
	if entry.Message != nil {
		for _, c := range entry.Message.Content {
			if c.Type == "tool_use" && c.Name == "Write" && len(c.Input) > 0 {
				var wi writeInput
				if err := json.Unmarshal(c.Input, &wi); err == nil && wi.FilePath != "" {
					return wi.FilePath
				}
			}
		}
	}

	return ""
}
