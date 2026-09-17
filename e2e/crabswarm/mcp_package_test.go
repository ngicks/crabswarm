package crabswarm_test

import (
	"bytes"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode"

	"gopkg.in/yaml.v3"
)

// The other half of what the apm package ships. The hook file moves messages;
// the manifest declares the MCP server that attends the room in the first
// place, and since the SessionStart join hook is gone it is the only automatic
// join a consumer gets. apm copies the declaration into `.mcp.json` and
// `.codex/config.toml` as written, so a command line that names a subcommand
// the binary does not have is a server that dies at harness startup with
// nothing to read about why — and the manifest is a text file no compiler ever
// sees.
type mcpPackageManifest struct {
	Dependencies struct {
		MCP []mcpDependency `yaml:"mcp"`
	} `yaml:"dependencies"`
}

// mcpDependency is apm's self-defined MCP server shape. Registry is a
// pointer because the field carries three states: absent means apm resolves the
// name against its registry, and only an explicit `false` means the manifest
// itself says how to start the server.
type mcpDependency struct {
	Name      string   `yaml:"name"`
	Registry  *bool    `yaml:"registry"`
	Transport string   `yaml:"transport"`
	Command   string   `yaml:"command"`
	Args      []string `yaml:"args"`
	EnvVars   []string `yaml:"env_vars"`
}

// readMCPPackageManifest decodes the package's apm.yml out of the checkout
// under test.
func readMCPPackageManifest(t *testing.T) mcpPackageManifest {
	t.Helper()
	path := filepath.Join(repoRoot(), "apm-package", "crabswarm-mcp", "apm.yml")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest %s: %v", path, err)
	}
	var manifest mcpPackageManifest
	if err := yaml.Unmarshal(b, &manifest); err != nil {
		t.Fatalf("decode manifest %s: %v", path, err)
	}
	return manifest
}

// declaredMCPServer returns the single MCP server the package declares,
// failing when it declares none or several: everything below is about that one
// server, and a manifest that grew a second one must not have one of them
// silently picked.
func declaredMCPServer(t *testing.T) mcpDependency {
	t.Helper()
	got := readMCPPackageManifest(t).Dependencies.MCP
	if len(got) != 1 {
		t.Fatalf("manifest declares %d MCP server(s), want exactly the crabswarm one: %v",
			len(got), got)
	}
	return got[0]
}

// What apm needs to render the server into either harness's config, pinned
// field by field: a registry lookup instead of `registry: false` sends apm
// looking for a published server that does not exist, and apm never splits
// `command` on whitespace, so the subcommand has to live in `args` or the
// harness spawns a binary named "crabswarm mcp".
func TestMCPPackage_DeclaresTheServer(t *testing.T) {
	server := declaredMCPServer(t)

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
	if server.Command != "crabswarm" {
		t.Errorf("command = %q, want %q — the binary the hooks and the skill "+
			"already assume on PATH", server.Command, "crabswarm")
	}
	if strings.ContainsFunc(server.Command, unicode.IsSpace) {
		t.Errorf("command = %q carries whitespace; apm rejects that and takes arguments from args",
			server.Command)
	}
	if want := []string{"mcp"}; !slices.Equal(server.Args, want) {
		t.Errorf("args = %v, want %v", server.Args, want)
	}
}

// The three variables the server cannot work without, named so a harness that
// spawns it with a fixed environment whitelist forwards them: two spellings of
// the identity token, and the runtime dir the daemon socket path is derived
// from. A server started without them answers the handshake and then refuses
// every tool, which is the failure this key exists to prevent.
func TestMCPPackage_ForwardsTheServersEnvironment(t *testing.T) {
	server := declaredMCPServer(t)

	want := []string{"CMDMAN_CMD_ID", "CRABSWARM_CHAT_TOKEN", "XDG_RUNTIME_DIR"}
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
	readJSONFile(t, filepath.Join(apmSkillPluginDir("crabswarm-mcp"), ".mcp.json"), &file)
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
// and not the other is a server that finds its token on one harness only.
func TestMCPPackage_PluginDeclaresTheSameServer(t *testing.T) {
	declared := declaredMCPServer(t)
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

// The declared command line, run against this checkout's binary: the manifest
// names a subcommand by string, and a rename anywhere in `cmd/` would leave the
// package shipping a server every consumer's harness fails to start. `--help`
// is the way to ask without handing the process a stdio session it would then
// wait on.
func TestMCPPackage_TheDeclaredCommandExists(t *testing.T) {
	server := declaredMCPServer(t)

	cmd := exec.CommandContext(t.Context(), crabswarmBin, append(server.Args, "--help")...)
	cmd.Env = chatEnviron()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("crabswarm %s --help: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(server.Args, " "), err, stdout.String(), stderr.String())
	}
}
