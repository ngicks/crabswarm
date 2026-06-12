// Package filetype provides filetype detection and project-root resolution.
//
// A [Config] is a self-describing leaf with three maps:
//
//   - Ext: extension (without leading dot) → filetype name.
//   - Filename: exact base name → filetype name.
//   - RootMarkers: filetype name → ordered groups of root markers.
//
// The on-disk default config is a slice of these leaves. At runtime, the
// slice is folded with [MergeAll] into a single merged Config — the same
// shape as a leaf, just bigger. Anything in a later leaf overrides
// anything from an earlier one (overlay semantics, neovim's
// `vim.filetype.add` style).
//
// Detection rules on a merged Config:
//
//  1. Exact filename match wins (most specific).
//  2. Longest-extension match next (`tar.gz` beats `gz`).
package filetype

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
)

// MarkerGroup is one priority tier of root markers. A directory satisfies
// the group when it contains at least one marker filename from the group.
type MarkerGroup []string

// Config is one filetype-configuration leaf. The same shape is also the
// merged big-table the runtime queries.
type Config struct {
	// Ext maps an extension (without leading dot, e.g. "go" or "tar.gz")
	// to a filetype name.
	Ext map[string]string `json:"ext,omitzero" yaml:"ext,omitempty"`
	// Filename maps an exact base name (e.g. "Makefile", "go.mod") to a
	// filetype name.
	Filename map[string]string `json:"filename,omitzero" yaml:"filename,omitempty"`
	// RootMarkers maps a filetype name to an ordered list of marker
	// groups. Groups are tried in order; the first group that finds any
	// marker at any ancestor wins.
	RootMarkers map[string][]MarkerGroup `json:"root_markers,omitzero" yaml:"root_markers,omitempty"`
}

// Detect returns the filetype name for filePath. See package doc for the
// rules.
func (c Config) Detect(filePath string) (string, bool) {
	base := filepath.Base(filePath)

	if name, ok := c.Filename[base]; ok {
		return name, true
	}

	parts := strings.Split(base, ".")
	for i := 1; i < len(parts); i++ {
		candidate := strings.Join(parts[i:], ".")
		if candidate == "" {
			continue
		}
		if name, ok := c.Ext[candidate]; ok {
			return name, true
		}
	}
	return "", false
}

// FindRoot resolves the module root for a (filetype-name, file-path)
// pair. Returns ("", false) when the filetype has no markers configured
// or none of the markers match any ancestor.
func (c Config) FindRoot(name, filePath string) (string, bool) {
	groups, ok := c.RootMarkers[name]
	if !ok || filePath == "" {
		return "", false
	}
	start := filepath.Dir(filePath)
	if start == "" {
		start = "."
	}
	for _, group := range groups {
		if dir, ok := walkUp(start, group); ok {
			return dir, true
		}
	}
	return "", false
}

// Merge returns a Config with c's entries overlaid by overlay's. Overlay
// wins on conflicts across all three maps.
func (c Config) Merge(overlay Config) Config {
	return Config{
		Ext:         mergeMap(c.Ext, overlay.Ext),
		Filename:    mergeMap(c.Filename, overlay.Filename),
		RootMarkers: mergeMap(c.RootMarkers, overlay.RootMarkers),
	}
}

// MergeAll folds a slice of leaves into one Config. Later leaves win on
// conflicts. The empty slice yields the zero Config.
func MergeAll(leaves []Config) Config {
	var out Config
	for _, leaf := range leaves {
		out = out.Merge(leaf)
	}
	return out
}

func mergeMap[K comparable, V any](base, overlay map[K]V) map[K]V {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	out := make(map[K]V, len(base)+len(overlay))
	maps.Copy(out, base)
	maps.Copy(out, overlay)
	return out
}

// walkUp returns the nearest ancestor of start (inclusive) that contains
// any of the markers.
func walkUp(start string, markers MarkerGroup) (string, bool) {
	dir := start
	for {
		for _, m := range markers {
			if _, err := os.Stat(filepath.Join(dir, m)); err == nil {
				return dir, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
