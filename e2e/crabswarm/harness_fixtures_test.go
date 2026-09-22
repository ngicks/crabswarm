package crabswarm_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// The files under testdata/harness are recordings of what the real harnesses
// did on a developer host: the MCP handshake each one opens with, the screens
// Claude Code paints in each state, the app-server traffic Codex speaks, and
// the post a fakechat channel takes together with the frame it answers with.
// Nothing regenerates them, so a truncated or unparsable copy has to fail loud
// rather than let a later test quietly assert against nothing.
// testdata/harness/README.md records how each one was captured.

const harnessFixtureDir = "testdata/harness"

// fakechatUploadID is the id the recorded upload carried. The channel frame the
// plugin server answered with repeats it, which is what pairs the two fixtures.
const fakechatUploadID = "crabswarm-1"

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

// harnessFixtureRequest additionally requires the whole file to be one HTTP
// request as it crossed the wire.
func harnessFixtureRequest(t *testing.T, name string) *http.Request {
	t.Helper()
	b := harnessFixtureBytes(t, name)
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(b)))
	if err != nil {
		t.Fatalf(
			"fixture %s is not one HTTP request: %v",
			filepath.Join(harnessFixtureDir, name),
			err,
		)
	}
	return req
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

func TestHarnessFixture_FakechatUpload(t *testing.T) {
	req := harnessFixtureRequest(t, "fakechat-upload.http")
	if req.Method != http.MethodPost {
		t.Errorf("method = %q, want %q", req.Method, http.MethodPost)
	}
	if req.URL.Path != "/upload" {
		t.Errorf("path = %q, want %q", req.URL.Path, "/upload")
	}
	if err := req.ParseMultipartForm(1 << 20); err != nil {
		t.Fatalf("parse multipart form: %v", err)
	}
	if got := req.FormValue("id"); got != fakechatUploadID {
		t.Errorf("field id = %q, want %q", got, fakechatUploadID)
	}
	if req.FormValue("text") == "" {
		t.Error("field text is empty")
	}
}

func TestHarnessFixture_FakechatChannelNotification(t *testing.T) {
	const name = "fakechat-channel-notification.jsonl"
	harnessFixtureNDJSON(t, name)

	// One upload produced this recording, so a second line would mean the
	// capture picked up traffic that does not belong to it.
	var frames [][]byte
	for line := range bytes.SplitSeq(harnessFixtureBytes(t, name), []byte("\n")) {
		if len(bytes.TrimSpace(line)) > 0 {
			frames = append(frames, line)
		}
	}
	if len(frames) != 1 {
		t.Fatalf(
			"fixture %s holds %d JSON lines, want 1",
			filepath.Join(harnessFixtureDir, name),
			len(frames),
		)
	}

	var frame struct {
		Method string `json:"method"`
		Params struct {
			Content string `json:"content"`
			Meta    struct {
				MessageID string `json:"message_id"`
			} `json:"meta"`
		} `json:"params"`
	}
	if err := json.Unmarshal(frames[0], &frame); err != nil {
		t.Fatalf("unmarshal frame: %v", err)
	}
	if frame.Method != "notifications/claude/channel" {
		t.Errorf("method = %q, want %q", frame.Method, "notifications/claude/channel")
	}
	if frame.Params.Content == "" {
		t.Error("params.content is empty")
	}
	if frame.Params.Meta.MessageID != fakechatUploadID {
		t.Errorf(
			"params.meta.message_id = %q, want the id the upload fixture sent (%q)",
			frame.Params.Meta.MessageID,
			fakechatUploadID,
		)
	}
}

func TestHarnessFixture_FakechatLaunch(t *testing.T) {
	harnessFixtureBytes(t, "fakechat-launch.md")
}

func TestHarnessFixture_README(t *testing.T) {
	harnessFixtureBytes(t, "README.md")
}
