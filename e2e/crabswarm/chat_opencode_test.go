package crabswarm_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// The OpenCode half of the crabswarm-mcp package is a plugin file rather
// than a hook file, and OpenCode runs it inside its own process, so the only
// way to see what it does is to run OpenCode. These cases do, against the
// `opencode` on PATH, with a mock model behind it: OpenCode needs a provider to
// take a turn at all, and a turn is what fires the events the plugin listens
// to. Without `opencode` on PATH the cases are skipped, not failed, because the
// binary is a host tool the suite does not build.
//
// OpenCode installs the plugin's dependency package from npm the first time a
// config directory is used, so a case's first wait is long.

// opencodePluginPath is where the package keeps the plugin: inside the skill
// directory, so apm carries it to every harness beside the Claude Code plugin
// files, and the operator points OpenCode at the deployed copy once.
var opencodePluginPath = []string{".apm", "skills", "crabswarm-mcp", "opencode.ts"}

// mockProvider is an OpenAI-compatible chat completions endpoint that answers
// every request with one short assistant turn and keeps the request bodies, so
// a case can read back what OpenCode sent the model: the prompt the plugin
// injects on idle arrives there and nowhere else.
type mockProvider struct {
	server *httptest.Server
	mu     sync.Mutex
	bodies []string
}

func newMockProvider(t *testing.T) *mockProvider {
	t.Helper()
	m := &mockProvider{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/completions", m.complete)
	m.server = httptest.NewServer(mux)
	t.Cleanup(m.server.Close)
	return m
}

func (m *mockProvider) complete(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	m.mu.Lock()
	m.bodies = append(m.bodies, string(body))
	m.mu.Unlock()

	var req struct {
		Stream bool `json:"stream"`
	}
	_ = json.Unmarshal(body, &req)

	if !req.Stream {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(
			w,
			`{"id":"chatcmpl-mock","object":"chat.completion","created":0,"model":"mock",`+
				`"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},`+
				`"finish_reason":"stop"}],`+
				`"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`,
		)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	for _, chunk := range []string{
		`{"id":"chatcmpl-mock","object":"chat.completion.chunk","created":0,"model":"mock",` +
			`"choices":[{"index":0,"delta":{"role":"assistant","content":"ok"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-mock","object":"chat.completion.chunk","created":0,"model":"mock",` +
			`"choices":[{"index":0,"delta":{},"finish_reason":"stop"}],` +
			`"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`,
	} {
		fmt.Fprintf(w, "data: %s\n\n", chunk)
	}
	fmt.Fprint(w, "data: [DONE]\n\n")
}

// requests returns the bodies received so far, oldest first.
func (m *mockProvider) requests() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.bodies)
}

// opencodeConfig is the global config the cases hand OpenCode: the plugin
// named the way the package README tells an operator to name it, relative to
// the config file, and the mock provider as the only model.
func opencodeConfig(baseURL string) string {
	return fmt.Sprintf(`{
  "$schema": "https://opencode.ai/config.json",
  "plugin": ["./skills/crabswarm-mcp/opencode.ts"],
  "model": "mock/mock",
  "provider": {
    "mock": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "mock",
      "options": {"baseURL": %q, "apiKey": "mock"},
      "models": {"mock": {"name": "mock", "limit": {"context": 128000, "output": 4096}}}
    }
  }
}
`, baseURL)
}

// opencodeHost is one OpenCode headless server started with the package's
// plugin, wired to a chat daemon and a mock model.
type opencodeHost struct {
	baseURL string
	mock    *mockProvider
	output  *syncBuffer
}

// syncBuffer collects a subprocess's output, which exec writes from its own
// goroutine, so a case can read it while the process still runs.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// startOpenCode starts `opencode serve` under a home of its own whose global
// config loads the plugin, with the environment an agent under cmdman has: the
// identity token and the runtime dir the daemon socket derives from. The
// daemon must already listen at the socket that runtime dir implies, since the
// bridge the plugin declares finds it that way and nothing else.
func startOpenCode(t *testing.T, opencode, cfgPath, runtimeDir, token string) *opencodeHost {
	t.Helper()

	mock := newMockProvider(t)

	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	pluginDir := filepath.Join(configDir, "skills", "crabswarm-mcp")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("make plugin dir: %v", err)
	}
	plugin, err := os.ReadFile(filepath.Join(append(
		[]string{repoRoot(), "apm-package", "crabswarm-mcp"}, opencodePluginPath...)...))
	if err != nil {
		t.Fatalf("read the shipped plugin: %v", err)
	}
	writeFile(t, filepath.Join(pluginDir, "opencode.ts"), string(plugin))
	writeFile(t, filepath.Join(configDir, "opencode.json"), opencodeConfig(mock.server.URL+"/v1"))

	addr := freeAddr(t)
	_, port, _ := strings.Cut(addr, ":")
	project := t.TempDir()

	output := &syncBuffer{}
	cmd := exec.Command(
		opencode,
		"serve",
		"--hostname",
		"127.0.0.1",
		"--port",
		port,
		"--print-logs",
	)
	cmd.Dir = project
	cmd.Env = append(chatEnviron(),
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_DATA_HOME="+filepath.Join(home, ".local", "share"),
		"XDG_STATE_HOME="+filepath.Join(home, ".local", "state"),
		"XDG_CACHE_HOME="+filepath.Join(home, ".cache"),
		"XDG_RUNTIME_DIR="+runtimeDir,
		"CRABSWARM_CONF="+cfgPath,
		chatTokenEnvVar+"="+token,
		"PATH="+filepath.Dir(crabswarmBin)+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Start(); err != nil {
		t.Fatalf("start opencode serve: %v", err)
	}
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("opencode output:\n%s", output.String())
		}
		stopProcess(t, cmd)
	})

	host := &opencodeHost{baseURL: "http://" + addr, mock: mock, output: output}
	host.waitReady(t, 240*time.Second)
	return host
}

// waitReady blocks until the server lists sessions, which it does only after
// its config, its plugins and their dependencies have loaded.
func (h *opencodeHost) waitReady(t *testing.T, timeout time.Duration) {
	t.Helper()
	// A request of its own deadline: the server accepts connections before its
	// bootstrap has finished, and a request that arrives then can sit unanswered.
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(h.baseURL + "/session")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("opencode serve did not answer within %s; output:\n%s", timeout, h.output.String())
}

// get fetches path and returns the body, failing on anything but 200.
func (h *opencodeHost) get(t *testing.T, path string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.baseURL+path, nil)
	if err != nil {
		t.Fatalf("build request %s: %v", path, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status %d\n%s", path, resp.StatusCode, b)
	}
	return string(b)
}

// relayAddress is the loopback address of a plugin's relay, as the plugin
// writes it into the environment of the MCP server it declares.
var relayAddress = regexp.MustCompile(`^http://127\.0\.0\.1:\d+/nudge$`)

// declaredRelay reads that address back off the running server's own config,
// which is where the config hook's work becomes visible.
func (h *opencodeHost) declaredRelay(t *testing.T) string {
	t.Helper()
	var config struct {
		MCP map[string]struct {
			Environment map[string]string `json:"environment"`
		} `json:"mcp"`
	}
	body := h.get(t, "/config")
	if err := json.Unmarshal([]byte(body), &config); err != nil {
		t.Fatalf("decode the opencode config %q: %v", body, err)
	}
	server, ok := config.MCP["crabswarm-mcp"]
	if !ok {
		t.Fatalf("opencode declares no crabswarm-mcp server: %v", config.MCP)
	}
	relay := server.Environment[harnessctl.OpenCodeRelayEnv]
	if !relayAddress.MatchString(relay) {
		t.Fatalf("the declared %s = %q, want the address of the plugin's own listener",
			harnessctl.OpenCodeRelayEnv, relay)
	}
	return relay
}

// post sends body as JSON to path and decodes the JSON answer into out.
func (h *opencodeHost) post(t *testing.T, path string, body string, out any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		h.baseURL+path,
		strings.NewReader(body),
	)
	if err != nil {
		t.Fatalf("build request %s: %v", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST %s: status %d\n%s", path, resp.StatusCode, b)
	}
	if out != nil {
		if err := json.Unmarshal(b, out); err != nil {
			t.Fatalf("decode POST %s answer %q: %v", path, b, err)
		}
	}
}

// takeTurn creates a session and sends one user message, returning once the
// model's reply has landed. The events the plugin acts on trail the reply, so
// callers poll for their effects.
func (h *opencodeHost) takeTurn(t *testing.T, prompt string) string {
	t.Helper()
	var session struct {
		ID string `json:"id"`
	}
	h.post(t, "/session", `{}`, &session)
	if session.ID == "" {
		t.Fatal("POST /session returned no id")
	}
	message, err := json.Marshal(map[string]any{
		"model": map[string]string{"providerID": "mock", "modelID": "mock"},
		"parts": []map[string]string{{"type": "text", "text": prompt}},
	})
	if err != nil {
		t.Fatal(err)
	}
	h.post(t, "/session/"+session.ID+"/message", string(message), nil)
	return session.ID
}

// waitFor polls cond until it holds or timeout passes, then fails with why.
func waitFor(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("%s did not happen within %s", what, timeout)
}

// The whole OpenCode story in one session. The plugin's config hook declares
// the bridge, so the member attends before any turn. A turn reports working
// when the message lands and done when the session goes idle with nothing
// unread. A mention waiting at idle is not left there: the read that finds it
// hands it to the session as the next prompt, so the model sees it, and the
// report of done comes only after that second read finds nothing.
func TestChatOpenCode_PluginAttendsReportsAndDelivers(t *testing.T) {
	opencode, err := exec.LookPath("opencode")
	if err != nil {
		t.Skip("opencode is not on PATH")
	}

	// A short runtime dir: the socket path derived from it is bounded at a
	// little over a hundred bytes.
	runtimeDir, err := os.MkdirTemp("", "crabswarm-rt")
	if err != nil {
		t.Fatalf("make a runtime dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(runtimeDir) })
	sock := filepath.Join(runtimeDir, "crabswarm", "default.sock")

	identity, recipient := newChatIdentityFile(t)
	cfg := writeChatConfigOn(t, t.TempDir(), sock, 0, []stubCommand{
		{token: "tok-oc", dir: chatRoom, project: "alpha", command: "opencode", scaleIndex: "1"},
	}, recipient)
	startChatServeOn(t, cfg, sock)

	// The teammate writing into the room is a person on the host, so cmdman
	// vouches for nothing: registration is what puts them in attendance, and
	// the token it prints is what they act with.
	bob := registerChatHuman(t, cfg, identity, chatRoom, "alpha", "bob")

	host := startOpenCode(t, opencode, cfg, runtimeDir, "tok-oc")

	// OpenCode starts its MCP servers when something first asks for them; the
	// TUI does at startup, and a headless server does on this request.
	host.get(t, "/mcp")

	// The bridge the plugin declared attends on its own, named after the
	// command and scale index the stub cmdman reports for the token.
	waitChatRosterHas(t, cfg, bob, "alpha/opencode-1", 120*time.Second)

	// The plugin opened the relay before it declared that bridge, so the bridge
	// started with the relay's address and attends as a member the daemon never
	// types at.
	relay := host.declaredRelay(t)
	roster := lines(runChat(t, cfg, bob, "members"))
	index := slices.IndexFunc(roster, func(line string) bool {
		return strings.HasPrefix(line, "alpha/opencode-1 ")
	})
	if index < 0 {
		t.Fatalf("members = %q, want a line for alpha/opencode-1", roster)
	}
	if !strings.HasSuffix(roster[index], "opencode  native") {
		t.Errorf("roster line = %q, want the member to deliver its own mentions "+
			"through the relay at %s", roster[index], relay)
	}

	// A bare name, resolved inside the sender's own team: the mention is what
	// makes the plugin's read hand the message to the session.
	runChat(t, cfg, bob, "send", "opencode-1", chatSentText)

	host.takeTurn(t, "say hello")

	waitFor(t, 60*time.Second, "the model receiving the delivered messages", func() bool {
		for _, body := range host.mock.requests() {
			if strings.Contains(body, "[crabswarm chat] Messages arrived while you were working") &&
				strings.Contains(body, `say \"hi\"`) {
				return true
			}
		}
		return false
	})

	waitFor(t, 60*time.Second, "the member reporting done", func() bool {
		log := stubStatus(t, cfg)
		return len(log) > 0 && log[len(log)-1] == "set done tok-oc --detail crabswarm chat"
	})

	log := stubStatus(t, cfg)
	if !slices.Contains(log, "set working tok-oc --detail crabswarm chat") {
		t.Errorf("cmdman status invocations = %q, want a working report before the done", log)
	}
	if got := runChat(t, cfg, "tok-oc", "read"); got != "no pending messages\n" {
		t.Errorf("read after the turn = %q, want the delivery to have moved the read position", got)
	}

	// Mentioned with the agent idle, which is the half the relay exists for: the
	// bridge posts the notice to the plugin rather than waiting for a hook, and
	// the plugin prompts it into the session it last saw a message in. The model
	// reads it as that session's next user message.
	//
	// What the person had half-written in the composer is left alone by
	// construction — the notice becomes a prompt of its own rather than text
	// appended to the composer — and a headless server has no composer to read
	// back, so there is nothing here to assert it against.
	runChat(t, cfg, bob, "send", "opencode-1", "the relay should wake you")

	waitFor(t, 120*time.Second, "the model reading the notice the relay delivered", func() bool {
		for _, body := range host.mock.requests() {
			if strings.Contains(body, "new message from alpha/bob") {
				return true
			}
		}
		return false
	})
}
