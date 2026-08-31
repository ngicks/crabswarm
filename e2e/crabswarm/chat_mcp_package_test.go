package crabswarm_test

import (
	"bytes"
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
// the manifest declares the MCP bridge that attends the room in the first
// place, and since the SessionStart join hook is gone it is the only automatic
// join a consumer gets. apm copies the declaration into `.mcp.json` and
// `.codex/config.toml` as written, so a command line that names a subcommand
// the binary does not have is a bridge that dies at harness startup with
// nothing to read about why — and the manifest is a text file no compiler ever
// sees.
type chatPackageManifest struct {
	Dependencies struct {
		MCP []chatMCPDependency `yaml:"mcp"`
	} `yaml:"dependencies"`
}

// chatMCPDependency is apm's self-defined MCP server shape. Registry is a
// pointer because the field carries three states: absent means apm resolves the
// name against its registry, and only an explicit `false` means the manifest
// itself says how to start the server.
type chatMCPDependency struct {
	Name      string   `yaml:"name"`
	Registry  *bool    `yaml:"registry"`
	Transport string   `yaml:"transport"`
	Command   string   `yaml:"command"`
	Args      []string `yaml:"args"`
}

// readChatPackageManifest decodes the package's apm.yml out of the checkout
// under test.
func readChatPackageManifest(t *testing.T) chatPackageManifest {
	t.Helper()
	path := filepath.Join(repoRoot(), "apm-package", "crabswarm-chat", "apm.yml")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest %s: %v", path, err)
	}
	var manifest chatPackageManifest
	if err := yaml.Unmarshal(b, &manifest); err != nil {
		t.Fatalf("decode manifest %s: %v", path, err)
	}
	return manifest
}

// chatDeclaredBridge returns the single MCP server the package declares,
// failing when it declares none or several: everything below is about that one
// server, and a manifest that grew a second one must not have one of them
// silently picked.
func chatDeclaredBridge(t *testing.T) chatMCPDependency {
	t.Helper()
	got := readChatPackageManifest(t).Dependencies.MCP
	if len(got) != 1 {
		t.Fatalf("manifest declares %d MCP server(s), want exactly the chat bridge: %v",
			len(got), got)
	}
	return got[0]
}

// What apm needs to render the bridge into either harness's config, pinned
// field by field: a registry lookup instead of `registry: false` sends apm
// looking for a published server that does not exist, and apm never splits
// `command` on whitespace, so the subcommand has to live in `args` or the
// harness spawns a binary named "crabswarm chat mcp".
func TestChatPackage_DeclaresTheBridge(t *testing.T) {
	bridge := chatDeclaredBridge(t)

	if bridge.Name != "crabswarm-chat" {
		t.Errorf("server name = %q, want %q", bridge.Name, "crabswarm-chat")
	}
	if bridge.Registry == nil || *bridge.Registry {
		t.Errorf("registry = %v, want an explicit false: the manifest starts this server itself",
			bridge.Registry)
	}
	if bridge.Transport != "stdio" {
		t.Errorf("transport = %q, want %q: the harness spawns the bridge as its own subprocess",
			bridge.Transport, "stdio")
	}
	if bridge.Command != "crabswarm" {
		t.Errorf("command = %q, want %q — the binary the hooks and the skill "+
			"already assume on PATH", bridge.Command, "crabswarm")
	}
	if strings.ContainsFunc(bridge.Command, unicode.IsSpace) {
		t.Errorf("command = %q carries whitespace; apm rejects that and takes arguments from args",
			bridge.Command)
	}
	if want := []string{"chat", "mcp"}; !slices.Equal(bridge.Args, want) {
		t.Errorf("args = %v, want %v", bridge.Args, want)
	}
}

// The declared command line, run against this checkout's binary: the manifest
// names a subcommand by string, and a rename anywhere in `cmd/` would leave the
// package shipping a bridge every consumer's harness fails to start. `--help`
// is the way to ask without handing the process a stdio session it would then
// wait on.
func TestChatPackage_TheDeclaredCommandExists(t *testing.T) {
	bridge := chatDeclaredBridge(t)

	cmd := exec.CommandContext(t.Context(), crabswarmBin, append(bridge.Args, "--help")...)
	cmd.Env = chatEnviron()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("crabswarm %s --help: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(bridge.Args, " "), err, stdout.String(), stderr.String())
	}
}
