package crabswarm_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The files under testdata/harness are recordings of what the real harnesses
// did on a developer host: the MCP handshake each one opens with, the screens
// Claude Code paints in each state, and the app-server traffic Codex speaks.
// Nothing regenerates them, so a truncated or unparsable copy has to fail loud
// rather than let a later test quietly assert against nothing.
// testdata/harness/README.md records how each one was captured.

const harnessFixtureDir = "testdata/harness"

// harnessFixtureBytes reads a fixture and fails when it is missing or empty.
func harnessFixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join(harnessFixtureDir, name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	if len(bytes.TrimSpace(b)) == 0 {
		t.Fatalf("fixture %s is empty", path)
	}
	return b
}

// harnessFixtureJSON additionally requires the whole file to be one JSON value.
func harnessFixtureJSON(t *testing.T, name string) {
	t.Helper()
	b := harnessFixtureBytes(t, name)
	if !json.Valid(b) {
		t.Fatalf("fixture %s is not valid JSON", filepath.Join(harnessFixtureDir, name))
	}
}

// harnessFixtureNDJSON additionally requires every non-blank line to be one
// JSON value, and requires at least one such line.
func harnessFixtureNDJSON(t *testing.T, name string) {
	t.Helper()
	path := filepath.Join(harnessFixtureDir, name)
	b := harnessFixtureBytes(t, name)
	var lines int
	for i, line := range bytes.Split(b, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if !json.Valid(line) {
			t.Fatalf("fixture %s line %d is not valid JSON: %s", path, i+1, line)
		}
		lines++
	}
	if lines == 0 {
		t.Fatalf("fixture %s holds no JSON line", path)
	}
}

func TestHarnessFixture_InitializeClaude(t *testing.T) {
	harnessFixtureJSON(t, "initialize-claude.json")
}

func TestHarnessFixture_InitializeCodex(t *testing.T) {
	harnessFixtureJSON(t, "initialize-codex.json")
}

func TestHarnessFixture_InitializeOpenCode(t *testing.T) {
	harnessFixtureJSON(t, "initialize-opencode.json")
}

func TestHarnessFixture_ClaudeChannelNotification(t *testing.T) {
	harnessFixtureNDJSON(t, "claude-channel-notification.jsonl")
}

func TestHarnessFixture_ClaudeChannelLaunch(t *testing.T) {
	harnessFixtureBytes(t, "claude-channel-launch.md")
}

func TestHarnessFixture_ClaudeScreenIdle(t *testing.T) {
	harnessFixtureBytes(t, "claude-screen-idle.txt")
}

func TestHarnessFixture_ClaudeScreenWorking(t *testing.T) {
	harnessFixtureBytes(t, "claude-screen-working.txt")
}

func TestHarnessFixture_ClaudeScreenDialog(t *testing.T) {
	harnessFixtureBytes(t, "claude-screen-dialog.txt")
}

func TestHarnessFixture_CodexAppServerClient(t *testing.T) {
	harnessFixtureNDJSON(t, "codex-app-server.client.ndjson")
}

func TestHarnessFixture_CodexAppServerServer(t *testing.T) {
	harnessFixtureNDJSON(t, "codex-app-server.server.ndjson")
}

func TestHarnessFixture_README(t *testing.T) {
	harnessFixtureBytes(t, "README.md")
}
