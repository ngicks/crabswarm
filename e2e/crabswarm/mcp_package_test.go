package crabswarm_test

import (
	"bytes"
	"maps"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode"

	crabmcp "github.com/ngicks/crabswarm/crabswarm/mcp"
	"github.com/ngicks/crabswarm/crabswarm/mcp/codexproxy"
)

// The MCP server declarations the apm packages ship. The server is what attends
// the room, and nothing else joins a member automatically. apm copies each
// declaration into the harness's config as written, so a command line that
// names a subcommand the binary does not have, or a URL the harness refuses, is
// a server that never starts or never answers, with nothing to read about why —
// and the manifest is a text file no compiler ever sees.
//
// Two packages declare it, one per transport, because apm declares one
// transport for every target of a package. crabswarm-mcp is Claude Code's: a
// stdio `crabswarm mcp` each session spawns. crabswarm-mcp-shared is Codex's
// and OpenCode's: a remote entry pointing at the one
// `crabswarm mcp --transport http` a compose session runs beside its harness
// servers.
const (
	stdioMCPPackage  = "crabswarm-mcp"
	sharedMCPPackage = "crabswarm-mcp-shared"
)

// sharedMCPURL is where crabswarm-mcp-shared points Codex and OpenCode. apm's
// Codex adapter writes a plain http URL only for a loopback host, and the
// harness servers reach the shared server beside them there.
const sharedMCPURL = "http://127.0.0.1:47300/mcp"

// mcpDependency is apm's self-defined MCP server shape. Registry is a
// pointer because the field carries three states: absent means apm resolves the
// name against its registry, and only an explicit `false` means the manifest
// itself says how to reach the server.
type mcpDependency struct {
	Name      string   `yaml:"name"`
	Registry  *bool    `yaml:"registry"`
	Transport string   `yaml:"transport"`
	URL       string   `yaml:"url"`
	Command   string   `yaml:"command"`
	Args      []string `yaml:"args"`
	EnvVars   []string `yaml:"env_vars"`
}

// declaredMCPServer returns the single MCP server the package pkg declares,
// failing when it declares none or several: everything below is about that one
// server, and a manifest that grew a second one must not have one of them
// silently picked.
func declaredMCPServer(t *testing.T, pkg string) mcpDependency {
	t.Helper()
	got := readApmManifest(t, pkg).Dependencies.MCP
	if len(got) != 1 {
		t.Fatalf("%s declares %d MCP server(s), want exactly the crabswarm one: %v",
			pkg, len(got), got)
	}
	return got[0]
}

// What apm needs to render the stdio server into Claude Code's config, pinned
// field by field: a registry lookup instead of `registry: false` sends apm
// looking for a published server that does not exist, and apm never splits
// `command` on whitespace, so the subcommand has to live in `args` or the
// harness spawns a binary named "crabswarm mcp". The package targets Claude
// Code alone: Codex and OpenCode reach the shared server instead.
func TestMCPPackage_DeclaresTheServer(t *testing.T) {
	targets := readApmManifest(t, stdioMCPPackage).Targets
	if want := []string{"claude"}; !slices.Equal(targets, want) {
		t.Errorf("targets = %v, want %v", targets, want)
	}

	server := declaredMCPServer(t, stdioMCPPackage)

	if server.Name != "crabswarm-mcp" {
		t.Errorf("server name = %q, want %q", server.Name, "crabswarm-mcp")
	}
	if server.Registry == nil || *server.Registry {
		t.Errorf("registry = %v, want an explicit false: the manifest starts this server itself",
			server.Registry)
	}
	if server.Transport != "stdio" {
		t.Errorf("transport = %q, want %q: the harness spawns the server as its own subprocess",
			server.Transport, "stdio")
	}
	if server.URL != "" {
		t.Errorf("url = %q, want none: a stdio server is started, not reached", server.URL)
	}
	if server.Command != "crabswarm" {
		t.Errorf("command = %q, want %q — the binary the skill already assumes on PATH",
			server.Command, "crabswarm")
	}
	if strings.ContainsFunc(server.Command, unicode.IsSpace) {
		t.Errorf("command = %q carries whitespace; apm rejects that and takes arguments from args",
			server.Command)
	}
	if want := []string{"mcp"}; !slices.Equal(server.Args, want) {
		t.Errorf("args = %v, want %v", server.Args, want)
	}
}

// The variables the stdio server cannot work without, named so the harness
// hands them to it even where it spawns the server with a fixed environment:
// two spellings of the identity token, and the runtime dir the daemon socket
// path is derived from. A server started without them answers the handshake and then refuses
// every tool, which is the failure this key exists to prevent.
//
// The channel variables are here for a different reason. They say the session
// was launched as one that registered the fakechat plugin, on the loopback port
// that plugin was launched with, so the server can hand it a mention directly;
// a server spawned without them attends as a member the daemon types at
// instead. Dropping the channel variable costs a keystroke nudge rather than a
// refusal, which is why it is listed together with the rest. The port is
// dropped at a higher price: a session that forwards the channel variable
// without it leaves its server looking for the plugin on the default 8787, and
// the member does not attend at all while the plugin is elsewhere.
func TestMCPPackage_ForwardsTheServersEnvironment(t *testing.T) {
	server := declaredMCPServer(t, stdioMCPPackage)

	want := []string{
		"CMDMAN_CMD_ID",
		"CRABSWARM_CHAT_TOKEN",
		"CRABSWARM_CLAUDE_CHANNEL",
		"FAKECHAT_PORT",
		"XDG_RUNTIME_DIR",
	}
	if !slices.Equal(server.EnvVars, want) {
		t.Errorf("env_vars = %v, want %v", server.EnvVars, want)
	}
}

// pluginMCPServer is the server shape in the plugin's `.mcp.json`, which Claude
// Code reads from the skills-directory plugin and never merges anywhere.
type pluginMCPServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// pluginDeclaredServer returns the single server the plugin declares.
func pluginDeclaredServer(t *testing.T) pluginMCPServer {
	t.Helper()
	var file struct {
		Servers map[string]pluginMCPServer `json:"mcpServers"`
	}
	readJSONFile(t, filepath.Join(apmSkillPluginDir(stdioMCPPackage), ".mcp.json"), &file)
	if len(file.Servers) != 1 {
		t.Fatalf("plugin declares %d MCP server(s), want exactly the crabswarm one: %v",
			len(file.Servers), file.Servers)
	}
	server, ok := file.Servers["crabswarm-mcp"]
	if !ok {
		t.Fatalf("plugin server is not named crabswarm-mcp: %v", file.Servers)
	}
	return server
}

// The plugin's `.mcp.json` is the same server apm renders from apm.yml, written
// in Claude Code's own shape: the command line matches, and every variable
// apm.yml asks a harness to forward is an `env` entry Claude Code expands from
// the process environment. One list, two renderings; a variable added to one
// and not the other is a server that finds its token, or its channel, only
// when the other rendering wins.
func TestMCPPackage_PluginDeclaresTheSameServer(t *testing.T) {
	declared := declaredMCPServer(t, stdioMCPPackage)
	plugin := pluginDeclaredServer(t)

	if plugin.Command != declared.Command {
		t.Errorf("plugin command = %q, apm.yml command = %q", plugin.Command, declared.Command)
	}
	if !slices.Equal(plugin.Args, declared.Args) {
		t.Errorf("plugin args = %v, apm.yml args = %v", plugin.Args, declared.Args)
	}
	got := slices.Sorted(maps.Keys(plugin.Env))
	want := slices.Sorted(slices.Values(declared.EnvVars))
	if !slices.Equal(got, want) {
		t.Errorf("plugin env keys = %v, apm.yml env_vars = %v", got, want)
	}
	for k, v := range plugin.Env {
		if v != "${"+k+"}" {
			t.Errorf(
				"plugin env %s = %q, want %q: the value is expanded from the harness's environment",
				k,
				v,
				"${"+k+"}",
			)
		}
	}
}

// runCrabswarmHelp runs the built binary with args and --help, failing when it
// does not exit 0. `--help` is the way to ask whether a command line parses
// without handing the process a session it would then wait on.
func runCrabswarmHelp(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), crabswarmBin, append(args, "--help")...)
	cmd.Env = chatEnviron()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("crabswarm %s --help: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
}

// The declared command line, run against this checkout's binary: the manifest
// names a subcommand by string, and a rename anywhere in `cmd/` would leave the
// package shipping a server every consumer's harness fails to start.
func TestMCPPackage_TheDeclaredCommandExists(t *testing.T) {
	runCrabswarmHelp(t, declaredMCPServer(t, stdioMCPPackage).Args...)
}

// What apm needs to render the shared server into Claude Code's, Codex's and
// OpenCode's config, pinned field by field.
//
// A remote entry names a URL and nothing to start. apm's Codex adapter writes a
// plain http URL only for a loopback host and skips the server otherwise, and
// the path is the one `crabswarm mcp --transport http` serves MCP at.
//
// The name is shared with code that looks the entry up by it: both OpenCode
// plugins read the entry's URL out of OpenCode's config, and
// `crabswarm mcp codex-proxy` stamps each replica's token on the Codex entry of
// that name by default. An entry renamed here alone is one none of them finds.
func TestMCPPackage_SharedDeclaresTheHTTPServer(t *testing.T) {
	targets := readApmManifest(t, sharedMCPPackage).Targets
	if want := []string{"claude", "codex", "opencode"}; !slices.Equal(targets, want) {
		t.Errorf("targets = %v, want %v", targets, want)
	}

	server := declaredMCPServer(t, sharedMCPPackage)

	if server.Name != codexproxy.DefaultServer {
		t.Errorf("server name = %q, want %q, the name codex-proxy stamps its header for",
			server.Name, codexproxy.DefaultServer)
	}
	if server.Registry == nil || *server.Registry {
		t.Errorf("registry = %v, want an explicit false: the manifest says where the server is",
			server.Registry)
	}
	if server.Transport != "streamable-http" {
		t.Errorf("transport = %q, want %q", server.Transport, "streamable-http")
	}
	if server.URL != sharedMCPURL {
		t.Errorf("url = %q, want %q", server.URL, sharedMCPURL)
	}
	if server.Command != "" || server.Args != nil || server.EnvVars != nil {
		t.Errorf("command = %q, args = %v, env_vars = %v; want none: "+
			"the harness reaches this server and starts nothing",
			server.Command, server.Args, server.EnvVars)
	}

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse url %q: %v", server.URL, err)
	}
	if u.Scheme != "http" {
		t.Errorf("url scheme = %q, want http: the server serves no TLS", u.Scheme)
	}
	if ip := net.ParseIP(u.Hostname()); ip == nil || !ip.IsLoopback() {
		t.Errorf("url host = %q, want a loopback address: apm's Codex adapter skips "+
			"a plain http URL on any other host", u.Hostname())
	}
	if u.Path != crabmcp.HTTPPath {
		t.Errorf("url path = %q, want %q, where the server serves MCP", u.Path, crabmcp.HTTPPath)
	}

	for _, plugin := range []string{"opencode.ts", "opencode-tui.ts"} {
		b, err := os.ReadFile(filepath.Join(apmSkillPluginDir(sharedMCPPackage), plugin))
		if err != nil {
			t.Fatalf("read the OpenCode plugin: %v", err)
		}
		if want := `const MCP_ENTRY = "` + server.Name + `"`; !strings.Contains(string(b), want) {
			t.Errorf("%s does not carry %s; it looks the server up by that name", plugin, want)
		}
	}
}

// The command lines the shared package's README launches, run against this
// checkout's binary: the server listening where the declared URL points, and
// the prefix every Codex replica runs its TUI behind.
func TestMCPPackage_TheSharedServerCommandsExist(t *testing.T) {
	u, err := url.Parse(declaredMCPServer(t, sharedMCPPackage).URL)
	if err != nil {
		t.Fatalf("parse the declared url: %v", err)
	}
	runCrabswarmHelp(t, "mcp", "--transport", "http", "--listen", u.Host)
	runCrabswarmHelp(t, "mcp", "codex-proxy")
}
