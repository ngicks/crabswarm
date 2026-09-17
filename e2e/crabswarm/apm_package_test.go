package crabswarm_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The apm packages this repository publishes. Each ships its Claude Code
// wiring as a skills-directory plugin: apm copies `.apm/skills/<name>/` to
// `~/.claude/skills/<name>/` with every file in it, and Claude Code loads a
// skill directory carrying `.claude-plugin/plugin.json` as a plugin, reading
// its hooks and MCP servers from the directory instead of from settings.json.
// The Codex wiring stays a merged hooks file, routed to Codex alone by its
// `codex-` stem. Both are text apm copies as written, so their shape is pinned
// here rather than discovered on a consumer's machine.
var apmPackages = []string{"crabswarm-mcp", "crabswarm-issues-lint"}

func apmPackageDir(name string) string {
	return filepath.Join(repoRoot(), "apm-package", name)
}

// apmSkillPluginDir is the one skill directory a package ships, named after
// the package so the plugin Claude Code derives from it is `<name>@skills-dir`.
func apmSkillPluginDir(name string) string {
	return filepath.Join(apmPackageDir(name), ".apm", "skills", name)
}

func readJSONFile(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}

// hooksObject returns the `hooks` object of a hook file, decoded loosely so two
// files can be compared whole rather than field by field.
func hooksObject(t *testing.T, path string) map[string]any {
	t.Helper()
	var file struct {
		Hooks map[string]any `json:"hooks"`
	}
	readJSONFile(t, path, &file)
	if len(file.Hooks) == 0 {
		t.Fatalf("%s wires no events", path)
	}
	return file.Hooks
}

// apm deploys a skill directory only when a SKILL.md sits at its root, and
// Claude Code turns it into a plugin only when the manifest is there and names
// the directory: a manifest name that drifts from the directory is a plugin
// Claude Code lists under one name and apm deploys under another.
func TestApmPackages_ShipAClaudePlugin(t *testing.T) {
	for _, name := range apmPackages {
		t.Run(name, func(t *testing.T) {
			dir := apmSkillPluginDir(name)
			if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
				t.Errorf("no SKILL.md at the skill root: %v", err)
			}

			var manifest struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			}
			readJSONFile(t, filepath.Join(dir, ".claude-plugin", "plugin.json"), &manifest)
			if manifest.Name != name {
				t.Errorf("plugin.json name = %q, want the directory name %q", manifest.Name, name)
			}

			var pkg struct {
				Version string `yaml:"version"`
			}
			b, err := os.ReadFile(filepath.Join(apmPackageDir(name), "apm.yml"))
			if err != nil {
				t.Fatalf("read apm.yml: %v", err)
			}
			if err := yaml.Unmarshal(b, &pkg); err != nil {
				t.Fatalf("decode apm.yml: %v", err)
			}
			if manifest.Version != pkg.Version {
				t.Errorf("plugin.json version = %q, apm.yml version = %q; want them equal",
					manifest.Version, pkg.Version)
			}

			hooksObject(t, filepath.Join(dir, "hooks", "hooks.json"))
		})
	}
}

// The OpenCode plugin carries the delivery wording of the hook file, since a
// message announced two different ways on two harnesses is a skill teaching
// the wrong words. The wording is pinned by its opening, which is what an
// agent recognises.
func TestApmPackages_OpenCodePluginSpeaksLikeTheHooks(t *testing.T) {
	plugin, err := os.ReadFile(filepath.Join(apmSkillPluginDir("crabswarm-mcp"), "opencode.ts"))
	if err != nil {
		t.Fatalf("read the OpenCode plugin: %v", err)
	}
	for _, want := range []string{
		"[crabswarm chat] Messages just arrived. Reply with",
		"[crabswarm chat] Messages arrived while you were working. Act on anything addressed to you",
	} {
		if !strings.Contains(string(plugin), want) {
			t.Errorf("opencode.ts does not carry %q", want)
		}
	}
	if strings.Contains(string(plugin), "console.log") {
		t.Error("opencode.ts writes to stdout, which the TUI shares with the screen")
	}
}

// The plugin's hooks file and the Codex hooks file are the same wiring written
// twice, because apm cannot hand one file to Codex's merge and keep it out of
// Claude Code's. A universal file under `.apm/hooks/` would be merged into
// settings.json beside the plugin, and every Claude Code hook would run twice;
// a Codex file that drifts from the plugin's is a member reporting differently
// on the two harnesses.
func TestApmPackages_MergeHooksIntoCodexOnly(t *testing.T) {
	for _, name := range apmPackages {
		t.Run(name, func(t *testing.T) {
			hooksDir := filepath.Join(apmPackageDir(name), ".apm", "hooks")
			entries, err := os.ReadDir(hooksDir)
			if err != nil {
				t.Fatalf("read %s: %v", hooksDir, err)
			}
			var names []string
			for _, e := range entries {
				names = append(names, e.Name())
			}
			if !reflect.DeepEqual(names, []string{"codex-hooks.json"}) {
				t.Fatalf(".apm/hooks holds %v, want exactly [codex-hooks.json]: "+
					"any other stem is merged into Claude Code's settings.json too", names)
			}

			codex := hooksObject(t, filepath.Join(hooksDir, "codex-hooks.json"))
			plugin := hooksObject(t, filepath.Join(apmSkillPluginDir(name), "hooks", "hooks.json"))
			if !reflect.DeepEqual(codex, plugin) {
				t.Errorf("codex-hooks.json and the plugin's hooks.json wire different hooks:"+
					"\ncodex:  %v\nplugin: %v", codex, plugin)
			}
		})
	}
}
