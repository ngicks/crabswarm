package crabswarm_test

import (
	"net"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	crabmcp "github.com/ngicks/crabswarm/crabswarm/mcp"
)

// A session of the shared server declares how its member is woken in request
// headers, beside the token that names the member: X-Crabswarm-Nudge for the
// delivery, X-Crabswarm-Channel for the channel a Claude Code registered. What
// follows drives `crabswarm mcp --transport http` with sessions that declare
// each, against the real daemon and its stub cmdman.

// startSharedMCP starts one `crabswarm mcp --transport http` against cfg's
// daemon and returns the address it serves MCP at, once it listens.
func startSharedMCP(t *testing.T, cfg string) string {
	t.Helper()
	addr := freeAddr(t)
	cmd := exec.Command(crabswarmBin, "mcp", "--config", cfg,
		"--transport", "http", "--listen", addr)
	cmd.Env = chatEnviron()
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start crabswarm mcp --transport http: %v", err)
	}
	t.Cleanup(func() { stopProcess(t, cmd) })
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if conn, err := net.Dial("tcp", addr); err == nil {
			_ = conn.Close()
			return addr
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("crabswarm mcp never listened on %s", addr)
	return ""
}

// declaringHeaders is a launcher's side of a shared server: every request a
// session makes carries the headers its replica declared.
type declaringHeaders map[string]string

func (h declaringHeaders) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	for k, v := range h {
		req.Header.Set(k, v)
	}
	return http.DefaultTransport.RoundTrip(req)
}

// dialSharedAs opens one session on the shared server at addr, handshaking as
// clientName and carrying headers.
func dialSharedAs(
	t *testing.T, addr, clientName string, headers map[string]string,
) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: clientName, Version: "v0"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{
		Endpoint:   "http://" + addr + crabmcp.HTTPPath,
		HTTPClient: &http.Client{Transport: declaringHeaders(headers)},
	}, nil)
	if err != nil {
		t.Fatalf("connect to the shared server as %s: %v", clientName, err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// A Claude Code replica that declared no waking attends saying so: the roster
// shows none, and a mention is typed at no terminal. The message still waits
// to be read.
func TestChatShared_ASessionThatDeclaredNoneIsNeverTypedAt(t *testing.T) {
	cfg := startChatDaemon(t)
	addr := startSharedMCP(t, cfg)
	dialSharedAs(t, addr, "claude-code", map[string]string{
		crabmcp.TokenHeader: "tok-ana",
		crabmcp.NudgeHeader: "none",
	})
	attendChatBridges(t, cfg, "tok-bob")
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)

	roster := lines(runChat(t, cfg, "tok-bob", "members"))
	slices.Sort(roster)
	want := []string{
		chatBridgeAna + "  agent  done  claude-code  none",
		chatBridgeBob + "  agent  done  other  terminal",
	}
	if !slices.Equal(roster, want) {
		t.Errorf("members = %v, want %v", roster, want)
	}

	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "the migration needs you")
	runChat(t, cfg, "tok-bob", "send", "everyone", "standup in five")
	if keys := stubSendKeys(t, cfg); keys != nil {
		t.Errorf("cmdman send-keys invocations = %q, want none", keys)
	}
	read := runChat(t, cfg, "tok-ana", "read")
	for _, want := range []string{"the migration needs you", "standup in five"} {
		if !strings.Contains(read, want) {
			t.Errorf("read as %s = %q, want it to carry %q", chatBridgeAna, read, want)
		}
	}
}

// A Claude Code replica that registered the fakechat plugin names the plugin's
// port in its own header, so replicas sharing one server each post to their own
// plugin. Its member is native, and a mention goes to that plugin as an upload
// rather than to its terminal.
func TestChatShared_TheDeclaredFakechatPortTakesTheMention(t *testing.T) {
	cfg := startChatDaemon(t)
	addr := startSharedMCP(t, cfg)
	fake := startFakeFakechat(t)
	dialSharedAs(t, addr, "claude-code", map[string]string{
		crabmcp.TokenHeader:   "tok-ana",
		crabmcp.ChannelHeader: "fakechat;port=" + fake.port,
	})
	attendChatBridges(t, cfg, "tok-bob")
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)

	roster := lines(runChat(t, cfg, "tok-bob", "members"))
	if !slices.Contains(roster, chatBridgeAna+"  agent  done  claude-code  native") {
		t.Errorf("members = %v, want %s native", roster, chatBridgeAna)
	}

	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "the migration needs you")
	fake.waitUploads(t, fakechatArrival(chatBridgeBob))
	if keys := stubSendKeys(t, cfg); keys != nil {
		t.Errorf("cmdman send-keys invocations = %q, want none", keys)
	}
}

// The shared server has no way to push a channel event down a session the
// SDK's HTTP handler holds, so a session declaring the direct channel acts as
// nobody, and its tools name the header its launcher has to change.
func TestChatShared_TheDirectChannelIsRefused(t *testing.T) {
	cfg := startChatDaemon(t)
	addr := startSharedMCP(t, cfg)
	attendChatBridges(t, cfg, "tok-bob")
	session := dialSharedAs(t, addr, "claude-code", map[string]string{
		crabmcp.TokenHeader:   "tok-ana",
		crabmcp.ChannelHeader: "direct",
	})

	text, failed := chatToolResult(t, session, "chat_members", nil)
	if !failed {
		t.Fatalf("chat_members succeeded for a session declaring the direct channel: %s", text)
	}
	if !strings.Contains(text, crabmcp.ChannelHeader) {
		t.Errorf("the refusal does not name %s: %s", crabmcp.ChannelHeader, text)
	}
	if out := runChat(t, cfg, "tok-bob", "members"); strings.Contains(out, chatBridgeAna) {
		t.Errorf("the refused session attends anyway:\n%s", out)
	}
}
