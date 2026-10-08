package crabswarm_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The apm packages this repository publishes. A package that targets Claude
// Code ships its wiring as a skills-directory plugin: apm copies
// `.apm/skills/<name>/` to `~/.claude/skills/<name>/` with every file in it,
// and Claude Code loads a skill directory carrying `.claude-plugin/plugin.json`
// as a plugin, reading whatever hooks and MCP servers the directory holds
// instead of settings.json. A Codex hook is a merged hooks file, routed to
// Codex alone by its `codex-` stem. crabswarm-mcp-shared targets Codex and
// OpenCode only, and ships its skill with the OpenCode plugins beside it. All
// of it is text apm copies as written, so its shape is pinned here rather than
// discovered on a consumer's machine.
var apmPackages = []string{"crabswarm-mcp", "crabswarm-mcp-shared", "crabswarm-issues-lint"}

func apmPackageDir(name string) string {
	return filepath.Join(repoRoot(), "apm-package", name)
}

// apmSkillPluginDir is the one skill directory a package ships, named after
// the package so the plugin Claude Code derives from it is `<name>@skills-dir`.
func apmSkillPluginDir(name string) string {
	return filepath.Join(apmPackageDir(name), ".apm", "skills", name)
}

// apmManifest is what these tests read of a package's apm.yml.
type apmManifest struct {
	Version      string   `yaml:"version"`
	Targets      []string `yaml:"targets"`
	Dependencies struct {
		MCP []mcpDependency `yaml:"mcp"`
	} `yaml:"dependencies"`
}

// readApmManifest decodes the apm.yml of the package name out of the checkout
// under test.
func readApmManifest(t *testing.T, name string) apmManifest {
	t.Helper()
	path := filepath.Join(apmPackageDir(name), "apm.yml")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest %s: %v", path, err)
	}
	var manifest apmManifest
	if err := yaml.Unmarshal(b, &manifest); err != nil {
		t.Fatalf("decode manifest %s: %v", path, err)
	}
	return manifest
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

// apmSkillFiles is every file each package's skill directory ships, as a path
// relative to that directory. apm copies the directory as it is, so a file
// added or dropped here is one added to or dropped from every consumer.
var apmSkillFiles = map[string][]string{
	"crabswarm-mcp": {
		".claude-plugin/plugin.json",
		".mcp.json",
		"SKILL.md",
	},
	"crabswarm-mcp-shared": {
		".claude-plugin/plugin.json",
		"SKILL.md",
		"opencode-tui.ts",
		"opencode.ts",
	},
	"crabswarm-issues-lint": {
		".claude-plugin/plugin.json",
		"SKILL.md",
		"hooks/hooks.json",
	},
}

func TestApmPackages_SkillDirectoriesShipTheirFiles(t *testing.T) {
	for _, name := range apmPackages {
		t.Run(name, func(t *testing.T) {
			want, ok := apmSkillFiles[name]
			if !ok {
				t.Fatalf("apmSkillFiles lists nothing for %s", name)
			}
			dir := apmSkillPluginDir(name)
			var got []string
			err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				rel, err := filepath.Rel(dir, path)
				if err != nil {
					return err
				}
				got = append(got, filepath.ToSlash(rel))
				return nil
			})
			if err != nil {
				t.Fatalf("walk %s: %v", dir, err)
			}
			slices.Sort(got)
			if !slices.Equal(got, want) {
				t.Errorf("the skill directory holds %v, want %v", got, want)
			}
		})
	}
}

// apmPluginHookPackages are the packages whose Claude Code plugin ships a hook
// file. crabswarm-issues-lint is one: its whole point is a `Stop` hook that
// blocks a turn on a broken diagram, and Claude Code has no other way to be
// told.
//
// crabswarm-mcp is not. Its member reports what the session is doing off the
// agents listing its own MCP server polls, and its messages arrive either as a
// channel notification or as a line the daemon types into the terminal, so a
// hook there would have nothing left to carry and one thing to get wrong: the
// `Stop` read it used to ship stored `done` whenever the inbox was empty, while
// a subagent running in the background kept the session working.
var apmPluginHookPackages = []string{"crabswarm-issues-lint"}

// Every package that targets Claude Code ships a plugin. apm deploys a skill
// directory only when a SKILL.md sits at its root, and Claude Code turns it into
// a plugin only when the manifest is there and names the directory: a manifest
// name that drifts from the directory is a plugin Claude Code lists under one
// name and apm deploys under another.
//
// The hook file is pinned in both directions. Claude Code loads whatever hooks
// the directory carries on every session, so a file re-added to a package that
// wires none is wiring nobody would notice until the room started disagreeing
// with itself.
func TestApmPackages_ShipAClaudePlugin(t *testing.T) {
	for _, name := range apmPackages {
		pkg := readApmManifest(t, name)
		if !slices.Contains(pkg.Targets, "claude") {
			continue
		}
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
			if manifest.Version != pkg.Version {
				t.Errorf("plugin.json version = %q, apm.yml version = %q; want them equal",
					manifest.Version, pkg.Version)
			}

			if slices.Contains(apmPluginHookPackages, name) {
				hooksObject(t, filepath.Join(dir, "hooks", "hooks.json"))
				return
			}
			if _, err := os.Stat(filepath.Join(dir, "hooks")); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("the plugin carries a hooks directory (%v); %s wires Claude Code none",
					err, name)
			}
		})
	}
}

// chatDeliveryNotices are the two lines the OpenCode plugins announce a
// handed-over message with: the one the server plugin appends to a tool result
// mid-turn, and the one the TUI plugin prompts with when a turn ends. Each is
// pinned by its opening, which is the part an agent recognises.
var chatDeliveryNotices = []string{
	"[crabswarm chat] Messages just arrived. Reply with",
	"[crabswarm chat] Messages arrived while you were working. Act on anything addressed to you",
}

// OpenCode's server plugin delivers mid-turn and its TUI plugin at the end of a
// turn, each announcing what it hands over with its own notice. Both plugins
// run inside OpenCode's own processes, whose stdout the TUI draws on, so
// neither may write there.
func TestApmPackages_OpenCodePluginsAnnounceTheirDelivery(t *testing.T) {
	for plugin, want := range map[string]string{
		"opencode.ts":     chatDeliveryNotices[0],
		"opencode-tui.ts": chatDeliveryNotices[1],
	} {
		b, err := os.ReadFile(filepath.Join(apmSkillPluginDir("crabswarm-mcp-shared"), plugin))
		if err != nil {
			t.Fatalf("read the OpenCode plugin: %v", err)
		}
		if !strings.Contains(string(b), want) {
			t.Errorf("%s does not carry %q", plugin, want)
		}
		if strings.Contains(string(b), "console.log") {
			t.Errorf("%s writes to stdout, which the TUI shares with the screen", plugin)
		}
	}
}

// apmCodexHookPackages are the packages that hand Codex a hook file.
//
// crabswarm-mcp targets Claude Code alone. crabswarm-mcp-shared hooks Codex
// nothing: a Codex hook runs beside the thread, inside the app server every
// replica shares, where the identity is the app server's rather than any
// replica's, so a chat read there would read for the wrong member. A mention
// reaches a Codex replica as a turn the shared MCP server starts instead.
var apmCodexHookPackages = []string{"crabswarm-issues-lint"}

// A package that hooks Codex ships it one file and one file only, under a
// `codex-` stem. apm cannot hand a file to Codex's merge and keep it out of
// Claude Code's: a universal file under `.apm/hooks/` would be merged into
// settings.json beside the plugin, and every Claude Code hook would then run
// twice. A package that hooks Codex nothing ships no `.apm/hooks/` at all.
func TestApmPackages_MergeHooksIntoCodexOnly(t *testing.T) {
	for _, name := range apmPackages {
		t.Run(name, func(t *testing.T) {
			hooksDir := filepath.Join(apmPackageDir(name), ".apm", "hooks")
			if !slices.Contains(apmCodexHookPackages, name) {
				if _, err := os.Stat(hooksDir); !errors.Is(err, fs.ErrNotExist) {
					t.Errorf(".apm/hooks is there (%v); %s wires Codex no hooks", err, name)
				}
				return
			}
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
			hooksObject(t, filepath.Join(hooksDir, "codex-hooks.json"))
		})
	}
}

// apmMirroredHookPackages are the packages whose two hook files are the same
// wiring written twice, which is every package that asks the two harnesses for
// the same thing. The other packages hook Codex nothing, and have no second
// file to compare a plugin's against.
var apmMirroredHookPackages = []string{"crabswarm-issues-lint"}

// The two copies of a mirrored package's wiring are compared whole: a Codex
// file that drifts from the plugin's is a check that runs on one harness and
// silently not on the other.
func TestApmPackages_MirrorTheHooksOntoCodex(t *testing.T) {
	for _, name := range apmMirroredHookPackages {
		t.Run(name, func(t *testing.T) {
			codex := hooksObject(t, filepath.Join(
				apmPackageDir(name), ".apm", "hooks", "codex-hooks.json"))
			plugin := hooksObject(t, filepath.Join(apmSkillPluginDir(name), "hooks", "hooks.json"))
			if !reflect.DeepEqual(codex, plugin) {
				t.Errorf("codex-hooks.json and the plugin's hooks.json wire different hooks:"+
					"\ncodex:  %v\nplugin: %v", codex, plugin)
			}
		})
	}
}
