package crabswarm_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// The OpenCode half of the crabswarm-mcp wiring is two plugins for the shared
// layout: one `opencode serve` hosts every agent, carrying the server plugin
// and a remote crabswarm-mcp entry pointing at `crabswarm mcp --transport http`
// beside it, and each agent is an `opencode attach` TUI carrying the TUI plugin
// and a token of its own. OpenCode runs the plugins inside its own processes,
// so the only way to see what they do is to run OpenCode. These cases do,
// against the `opencode` on PATH, with a mock model behind the server: OpenCode
// needs a provider to take a turn at all, and a turn is what fires the events
// the plugins listen to. Without `opencode` on PATH the cases are skipped, not
// failed, because the binary is a host tool the suite does not build.
//
// The TUI runs without a terminal, drawing into output nobody reads. What a
// person does in it is done over the server's API instead: a turn is a message
// posted to the session, and moving to another session is the server's
// select-session call, which every attached TUI follows.
//
// OpenCode installs the plugins' dependency package from npm the first time a
// config directory is used, so a case's first waits are long.

// opencodePluginDir is where the package keeps both plugins: inside the skill
// directory, so apm carries them beside the skill, and the operator points
// OpenCode at the deployed copies once.
var opencodePluginDir = []string{
	"apm-package", "crabswarm-mcp-shared", ".apm", "skills", "crabswarm-mcp-shared",
}

// opencodePassword guards the OpenCode server, as it does on a host where the
// server listens beyond loopback. The TUI and its plugin both read it from the
// environment.
const opencodePassword = "e2e-secret"

// opencodeSendTool is chat_send as OpenCode names it to the model: prefixed
// with the name the MCP server is configured under.
const opencodeSendTool = "crabswarm-mcp_chat_send"

// The cmdman status lines the replica's reports become.
const (
	opencodeWorking = "set working tok-oc --detail crabswarm chat"
	opencodeDone    = "set done tok-oc --detail crabswarm chat"
)

// mockRequest is what a case reads of one chat completions request.
type mockRequest struct {
	Stream   bool              `json:"stream"`
	Messages []json.RawMessage `json:"messages"`
	Tools    []mockTool        `json:"tools"`
}

// mockTool is one tool a request hands the model.
type mockTool struct {
	Function struct {
		Name string `json:"name"`
	} `json:"function"`
}

// last returns the role and the raw JSON of the request's last message.
func (r mockRequest) last() (role, raw string) {
	if len(r.Messages) == 0 {
		return "", ""
	}
	msg := r.Messages[len(r.Messages)-1]
	var head struct {
		Role string `json:"role"`
	}
	_ = json.Unmarshal(msg, &head)
	return head.Role, string(msg)
}

// offers reports whether the request hands the model the tool named name.
func (r mockRequest) offers(name string) bool {
	return slices.ContainsFunc(r.Tools, func(tool mockTool) bool {
		return tool.Function.Name == name
	})
}

// mockReply is one turn of the model: a call of tool with the JSON arguments
// args, or the text "ok" when tool is empty.
type mockReply struct {
	tool string
	args string
}

// mockProvider is an OpenAI-compatible chat completions endpoint that keeps
// the request bodies, so a case can read back what OpenCode sent the model: a
// prompt a plugin injects arrives there and nowhere else. answer picks the
// reply to each streamed request, and may hold it to let a case act while the
// turn is in flight; a request that is not streamed is answered "ok".
type mockProvider struct {
	server *httptest.Server
	answer func(ctx context.Context, req mockRequest) mockReply
	calls  atomic.Int64
	mu     sync.Mutex
	bodies []string
}

func newMockProvider(
	t *testing.T, answer func(ctx context.Context, req mockRequest) mockReply,
) *mockProvider {
	t.Helper()
	m := &mockProvider{answer: answer}
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

	var req mockRequest
	_ = json.Unmarshal(body, &req)
	usage := map[string]int{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}

	if !req.Stream {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-mock", "object": "chat.completion", "created": 0, "model": "mock",
			"choices": []map[string]any{{
				"index":         0,
				"message":       map[string]any{"role": "assistant", "content": "ok"},
				"finish_reason": "stop",
			}},
			"usage": usage,
		})
		return
	}

	reply := m.answer(r.Context(), req)
	delta := map[string]any{"role": "assistant", "content": "ok"}
	finish := "stop"
	if reply.tool != "" {
		delta = map[string]any{"role": "assistant", "tool_calls": []map[string]any{{
			"index": 0,
			"id":    fmt.Sprintf("call_mock_%d", m.calls.Add(1)),
			"type":  "function",
			"function": map[string]any{
				"name":      reply.tool,
				"arguments": reply.args,
			},
		}}}
		finish = "tool_calls"
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	for _, chunk := range []map[string]any{
		{"choices": []map[string]any{{"index": 0, "delta": delta, "finish_reason": nil}}},
		{"choices": []map[string]any{{"index": 0, "delta": map[string]any{}, "finish_reason": finish}},
			"usage": usage},
	} {
		chunk["id"], chunk["object"], chunk["created"], chunk["model"] =
			"chatcmpl-mock", "chat.completion.chunk", 0, "mock"
		b, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", b)
	}
	fmt.Fprint(w, "data: [DONE]\n\n")
}

// requests returns the bodies received so far, oldest first.
func (m *mockProvider) requests() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.bodies)
}

// sawRequest reports whether any request so far carried every one of parts.
func (m *mockProvider) sawRequest(parts ...string) bool {
	return slices.ContainsFunc(m.requests(), func(body string) bool {
		for _, part := range parts {
			if !strings.Contains(body, part) {
				return false
			}
		}
		return true
	})
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

// opencodeHome makes a home of its own for one OpenCode process, with the
// package's plugin file installed in its config directory the way the package
// README tells an operator to, and returns the home and that config directory.
func opencodeHome(t *testing.T, plugin string) (home, configDir string) {
	t.Helper()
	home = t.TempDir()
	configDir = filepath.Join(home, ".config", "opencode")
	pluginDir := filepath.Join(configDir, "skills", "crabswarm-mcp-shared")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("make plugin dir: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(append(
		append([]string{repoRoot()}, opencodePluginDir...), plugin)...))
	if err != nil {
		t.Fatalf("read the shipped plugin %s: %v", plugin, err)
	}
	writeFile(t, filepath.Join(pluginDir, plugin), string(b))
	return home, configDir
}

// opencodeEnviron is the environment an OpenCode process under home runs with:
// the suite's scrubbed one, the home, the config every crabswarm verb reads,
// the server password, and the built crabswarm first on PATH.
func opencodeEnviron(home, cfgPath string, extra ...string) []string {
	env := append(chatEnviron(),
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_DATA_HOME="+filepath.Join(home, ".local", "share"),
		"XDG_STATE_HOME="+filepath.Join(home, ".local", "state"),
		"XDG_CACHE_HOME="+filepath.Join(home, ".cache"),
		"CRABSWARM_CONF="+cfgPath,
		"OPENCODE_SERVER_PASSWORD="+opencodePassword,
		"PATH="+filepath.Dir(crabswarmBin)+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	return append(env, extra...)
}

// startOpenCodeMCP starts the `crabswarm mcp --transport http` a shared OpenCode
// server reaches, and returns the address it listens on.
func startOpenCodeMCP(t *testing.T, cfgPath string) string {
	t.Helper()
	addr := freeAddr(t)
	cmd := exec.Command(crabswarmBin, "mcp", "--config", cfgPath,
		"--transport", "http", "--listen", addr)
	cmd.Env = chatEnviron()
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start crabswarm mcp --transport http: %v", err)
	}
	t.Cleanup(func() { stopProcess(t, cmd) })
	waitFor(t, 30*time.Second, "crabswarm mcp listening on "+addr, func() bool {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	})
	return addr
}

// opencodeServer is one `opencode serve` started with the server plugin, the
// remote crabswarm-mcp entry and a mock model.
type opencodeServer struct {
	baseURL string
	// project is the directory the server runs in, which is the project every
	// TUI attached to it works in.
	project string
	output  *syncBuffer
}

// startOpenCodeServe starts `opencode serve` under a home of its own whose
// global config loads the server plugin and names the MCP server at mcpAddr as
// a remote one. The server runs under cmdman as tok-srv, a token cmdman
// vouches for, which the plugin must never act as.
func startOpenCodeServe(
	t *testing.T, opencode, cfgPath, mcpAddr string, mock *mockProvider,
) *opencodeServer {
	t.Helper()

	home, configDir := opencodeHome(t, "opencode.ts")
	writeFile(t, filepath.Join(configDir, "opencode.json"), fmt.Sprintf(`{
  "$schema": "https://opencode.ai/config.json",
  "plugin": ["./skills/crabswarm-mcp-shared/opencode.ts"],
  "mcp": {"crabswarm-mcp": {"type": "remote", "url": %q, "enabled": true}},
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
`, "http://"+mcpAddr+"/mcp", mock.server.URL+"/v1"))

	addr := freeAddr(t)
	_, port, _ := strings.Cut(addr, ":")
	s := &opencodeServer{baseURL: "http://" + addr, project: t.TempDir(), output: &syncBuffer{}}

	cmd := exec.Command(
		opencode,
		"serve",
		"--hostname",
		"127.0.0.1",
		"--port",
		port,
		"--print-logs",
	)
	cmd.Dir = s.project
	cmd.Env = opencodeEnviron(home, cfgPath, "CMDMAN_CMD_ID=tok-srv")
	cmd.Stdout = s.output
	cmd.Stderr = s.output
	if err := cmd.Start(); err != nil {
		t.Fatalf("start opencode serve: %v", err)
	}
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("opencode serve output:\n%s", s.output.String())
		}
		stopProcess(t, cmd)
	})

	s.waitReady(t, 240*time.Second)
	return s
}

// startOpenCodeTUI attaches a TUI to s, showing session, under a home of its
// own whose tui.json loads the TUI plugin. It runs as the agent token names,
// which is the only place the plugin finds its identity.
func startOpenCodeTUI(
	t *testing.T,
	opencode string,
	s *opencodeServer,
	cfgPath, token, session string,
) {
	t.Helper()

	home, configDir := opencodeHome(t, "opencode-tui.ts")
	writeFile(t, filepath.Join(configDir, "tui.json"),
		`{"plugin": ["./skills/crabswarm-mcp-shared/opencode-tui.ts"]}`+"\n")

	output := &syncBuffer{}
	cmd := exec.Command(opencode, "attach", s.baseURL, "--session", session, "--print-logs")
	cmd.Dir = s.project
	cmd.Env = opencodeEnviron(home, cfgPath,
		chatTokenEnvVar+"="+token,
		"CRABSWARM_OPENCODE_SERVER="+s.baseURL,
		"TERM=xterm-256color",
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = output
	if err := cmd.Start(); err != nil {
		t.Fatalf("start opencode attach: %v", err)
	}
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("opencode attach stderr:\n%s", output.String())
		}
		stopProcess(t, cmd)
	})
}

// waitReady blocks until the server lists sessions, which it does only after
// its config, its plugins and their dependencies have loaded.
func (s *opencodeServer) waitReady(t *testing.T, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// A request of its own deadline: the server accepts connections before
		// its bootstrap has finished, and a request that arrives then can sit
		// unanswered.
		if status, _ := s.try(
			t,
			http.MethodGet,
			"/session",
			"",
			5*time.Second,
		); status == http.StatusOK {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("opencode serve did not answer within %s; output:\n%s", timeout, s.output.String())
}

// try sends one request to the server with its password and returns the
// status and the body, or 0 when the request itself failed.
func (s *opencodeServer) try(
	t *testing.T, method, path, body string, timeout time.Duration,
) (int, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build request %s %s: %v", method, path, err)
	}
	req.SetBasicAuth("opencode", opencodePassword)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

// call sends one request and returns the body, failing on anything but 200.
func (s *opencodeServer) call(t *testing.T, method, path, body string) string {
	t.Helper()
	status, answer := s.try(t, method, path, body, 120*time.Second)
	if status != http.StatusOK {
		t.Fatalf("%s %s: status %d\n%s", method, path, status, answer)
	}
	return answer
}

// createSession creates a session and returns its id.
func (s *opencodeServer) createSession(t *testing.T) string {
	t.Helper()
	var session struct {
		ID string `json:"id"`
	}
	answer := s.call(t, http.MethodPost, "/session", `{}`)
	if err := json.Unmarshal([]byte(answer), &session); err != nil || session.ID == "" {
		t.Fatalf("POST /session answered %q, want a session id (%v)", answer, err)
	}
	return session.ID
}

// takeTurn sends one user message to session and returns once the model's
// reply has landed. The events the plugins act on trail the reply, so callers
// poll for their effects.
func (s *opencodeServer) takeTurn(t *testing.T, session, prompt string) {
	t.Helper()
	message, err := json.Marshal(map[string]any{
		"model": map[string]string{"providerID": "mock", "modelID": "mock"},
		"parts": []map[string]string{{"type": "text", "text": prompt}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s.call(t, http.MethodPost, "/session/"+session+"/message", string(message))
}

// openCodeRead is the server plugin's read of what the member whose TUI shows
// session has unread, answered with the status alone: 404 for a session no TUI
// shows. It reads as that member, so a case asks only while nothing is unread.
func openCodeRead(t *testing.T, mcpAddr, session string) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://"+mcpAddr+"/opencode/sessions/"+session+"/read", nil)
	if err != nil {
		t.Fatalf("build the read of %s: %v", session, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode
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

// waitLastStubStatus blocks until the newest `cmdman status` invocation is
// want, which is the state the member shows now.
func waitLastStubStatus(t *testing.T, cfgPath, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var published []string
	for time.Now().Before(deadline) {
		published = stubStatus(t, cfgPath)
		if len(published) > 0 && published[len(published)-1] == want {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("the newest cmdman status never became %q within %s; it carried %q",
		want, timeout, published)
}

// The shared layout in one story. The TUI's plugin makes the TUI a member of
// its own: it attends as a native OpenCode member, whose mentions the daemon
// never types, and registers the session the TUI shows, again whenever the TUI
// moves to another. A turn there is reported working and then done.
//
// A mention of the member is prompted into the session the TUI shows. The model
// answers it with chat_send, which the server's plugin names the session for,
// so the message goes out as this member. The send's result carries the
// mention's text, read mid-turn. A second mention arriving while the turn runs
// waits for it to end, and the read ending the turn hands it over as the next
// prompt, after which the member is done with nothing left unread.
//
// The server's own token never acts: the server plugin runs with it and reads
// none.
func TestChatOpenCode_ASharedServerCarriesEachTUIAsItsOwnMember(t *testing.T) {
	opencode, err := exec.LookPath("opencode")
	if err != nil {
		t.Skip("opencode is not on PATH")
	}

	identity, recipient := newChatIdentityFile(t)
	cfg := writeChatConfig(t, 0, []stubCommand{
		{token: "tok-oc", dir: chatRoom, project: "alpha", command: "opencode", scaleIndex: "1"},
		{token: "tok-srv", dir: chatRoom, project: "alpha", command: "opencode-serve"},
	}, recipient)
	startChatServe(t, cfg)
	// The teammate writing into the room is a person on the host, so cmdman
	// vouches for nothing: registration is what puts them in attendance.
	bob := registerChatHuman(t, cfg, identity, chatRoom, "alpha", "bob")

	mcpAddr := startOpenCodeMCP(t, cfg)

	// The model answers the notice of bob's mention with a chat_send to bob, and
	// holds the reply to that send's result until the case has sent a second
	// mention, so the second one lands while the turn runs.
	const notice = "new message from alpha/bob"
	midTurn := make(chan struct{})
	resume := make(chan struct{})
	release := sync.OnceFunc(func() { close(resume) })
	var held sync.Once
	mock := newMockProvider(t, func(ctx context.Context, req mockRequest) mockReply {
		role, last := req.last()
		switch {
		case role == "user" && strings.Contains(last, notice) && req.offers(opencodeSendTool):
			return mockReply{tool: opencodeSendTool, args: `{"to":"bob","message":"hello back"}`}
		case role == "tool":
			held.Do(func() {
				close(midTurn)
				select {
				case <-resume:
				case <-ctx.Done():
				}
			})
		}
		return mockReply{}
	})
	t.Cleanup(release)

	server := startOpenCodeServe(t, opencode, cfg, mcpAddr, mock)
	first := server.createSession(t)
	startOpenCodeTUI(t, opencode, server, cfg, "tok-oc", first)

	// The TUI attends under the name cmdman's labels give it, as a member whose
	// mentions its own channel carries.
	waitChatRosterHas(t, cfg, bob, "alpha/opencode-1", 120*time.Second)
	roster := lines(runChat(t, cfg, bob, "members"))
	index := slices.IndexFunc(roster, func(line string) bool {
		return strings.HasPrefix(line, "alpha/opencode-1 ")
	})
	if index < 0 {
		t.Fatalf("members = %q, want a line for alpha/opencode-1", roster)
	}
	if !strings.HasSuffix(roster[index], "opencode  native") {
		t.Errorf("roster line = %q, want a native OpenCode member", roster[index])
	}

	waitFor(t, 60*time.Second, "the TUI registering the session it shows", func() bool {
		return openCodeRead(t, mcpAddr, first) == http.StatusNoContent
	})
	waitFor(t, 60*time.Second, "opencode connecting to crabswarm mcp", func() bool {
		_, answer := server.try(t, http.MethodGet, "/mcp", "", 10*time.Second)
		return strings.Contains(answer, `"connected"`)
	})

	server.takeTurn(t, first, "say hello")
	waitLastStubStatus(t, cfg, opencodeDone, 60*time.Second)
	if log := stubStatus(t, cfg); !slices.Contains(log, opencodeWorking) {
		t.Errorf("cmdman status invocations = %q, want a working report before the done", log)
	}

	// Moving to another session moves the registration with it.
	second := server.createSession(t)
	server.call(t, http.MethodPost, "/tui/select-session",
		fmt.Sprintf(`{"sessionID":%q}`, second))
	waitFor(t, 60*time.Second, "the TUI registering the session it moved to", func() bool {
		return openCodeRead(t, mcpAddr, second) == http.StatusNoContent &&
			openCodeRead(t, mcpAddr, first) == http.StatusNotFound
	})

	runChat(t, cfg, bob, "send", "opencode-1", chatSentText)

	select {
	case <-midTurn:
	case <-time.After(120 * time.Second):
		t.Fatal("the model never received the result of its chat_send")
	}
	if !mock.sawRequest(chatDeliveryNotices[0], `say \"hi\"`) {
		t.Errorf("the chat_send result the model received does not carry the mention read mid-turn")
	}
	waitLastStubStatus(t, cfg, opencodeWorking, 30*time.Second)
	runChat(t, cfg, bob, "send", "opencode-1", "late ping")
	release()

	waitFor(t, 60*time.Second, "the model receiving the late mention", func() bool {
		return mock.sawRequest(chatDeliveryNotices[1], "late ping")
	})
	waitLastStubStatus(t, cfg, opencodeDone, 60*time.Second)

	shown := server.call(t, http.MethodGet, "/session/"+second+"/message", "")
	if !strings.Contains(shown, notice) {
		t.Errorf("the session the TUI shows holds no prompt of the mention's notice")
	}
	left := server.call(t, http.MethodGet, "/session/"+first+"/message", "")
	if strings.Contains(left, notice) {
		t.Errorf("the session the TUI left was prompted with the mention's notice")
	}
	if got := runChat(t, cfg, bob, "read"); !strings.Contains(
		got, "alpha/opencode-1 -> alpha/bob [mentioned you]: hello back") {
		t.Errorf("bob's read = %q, want the chat_send the model made as alpha/opencode-1", got)
	}
	if got := runChat(t, cfg, "tok-oc", "read"); got != "no pending messages\n" {
		t.Errorf("the member's read after the turn = %q, want both mentions handed over", got)
	}
	if log := stubStatus(t, cfg); slices.ContainsFunc(log, func(line string) bool {
		return strings.Contains(line, "tok-srv")
	}) {
		t.Errorf("cmdman status invocations = %q, want none for the server's own token", log)
	}
	if members := memberAddresses(runChat(t, cfg, bob, "members")); slices.Contains(
		members, "alpha/opencode-serve") {
		t.Errorf("members = %v, want the server's own token never to attend", members)
	}
}
