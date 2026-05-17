package filetype

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func sampleLeaves() []Config {
	return []Config{
		{
			Ext:      map[string]string{"go": "go"},
			Filename: map[string]string{"go.mod": "go", "go.sum": "go"},
			RootMarkers: map[string][]MarkerGroup{
				"go": {{"go.work", "go.mod"}, {".git"}},
			},
		},
		{
			Ext:      map[string]string{"rs": "rust"},
			Filename: map[string]string{"Cargo.toml": "rust"},
			RootMarkers: map[string][]MarkerGroup{
				"rust": {{"Cargo.toml"}, {".git"}},
			},
		},
		{
			Ext:      map[string]string{"mbt": "moonbit", "mbti": "moonbit"},
			Filename: map[string]string{"moon.pkg": "moonbit", "moon.mod.json": "moonbit"},
			RootMarkers: map[string][]MarkerGroup{
				"moonbit": {{"moon.mod.json"}, {".git"}},
			},
		},
	}
}

func TestDetect_ByExtension(t *testing.T) {
	c := MergeAll(sampleLeaves())
	got, ok := c.Detect("/x/y/foo.go")
	assert.Assert(t, ok)
	assert.Equal(t, got, "go")

	got, ok = c.Detect("/x/y/foo.mbt")
	assert.Assert(t, ok)
	assert.Equal(t, got, "moonbit")
}

func TestDetect_ByFilename(t *testing.T) {
	c := MergeAll(sampleLeaves())
	got, ok := c.Detect("/x/y/Cargo.toml")
	assert.Assert(t, ok)
	assert.Equal(t, got, "rust")
}

func TestDetect_FilenameWinsOverExtension(t *testing.T) {
	c := Config{
		Ext:      map[string]string{"json": "json"},
		Filename: map[string]string{"moon.mod.json": "moonbit"},
	}
	got, ok := c.Detect("/x/moon.mod.json")
	assert.Assert(t, ok)
	assert.Equal(t, got, "moonbit")
}

func TestDetect_LongestExtensionWins(t *testing.T) {
	c := Config{
		Ext: map[string]string{
			"tar.gz": "tarball",
			"gz":     "gzip",
		},
	}
	got, ok := c.Detect("/x/foo.tar.gz")
	assert.Assert(t, ok)
	assert.Equal(t, got, "tarball")

	got, ok = c.Detect("/x/foo.gz")
	assert.Assert(t, ok)
	assert.Equal(t, got, "gzip")
}

func TestDetect_NoMatch(t *testing.T) {
	c := MergeAll(sampleLeaves())
	_, ok := c.Detect("/x/y/README.adoc")
	assert.Assert(t, !ok)
}

// touch creates an empty file at the given path.
func touch(t *testing.T, path string) {
	t.Helper()
	assert.NilError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	assert.NilError(t, err)
	assert.NilError(t, f.Close())
}

func TestFindRoot_FirstGroupWinsOverNearerSecondGroup(t *testing.T) {
	tmp := t.TempDir()
	assert.NilError(t, os.MkdirAll(filepath.Join(tmp, ".git"), 0o755))
	touch(t, filepath.Join(tmp, "proj", "go.mod"))
	touch(t, filepath.Join(tmp, "proj", "pkg", "foo.go"))

	c := Config{
		RootMarkers: map[string][]MarkerGroup{
			"go": {{"go.mod"}, {".git"}},
		},
	}
	root, ok := c.FindRoot("go", filepath.Join(tmp, "proj", "pkg", "foo.go"))
	assert.Assert(t, ok)
	assert.Equal(t, root, filepath.Join(tmp, "proj"))
}

func TestFindRoot_FallsBackToNextGroup(t *testing.T) {
	tmp := t.TempDir()
	assert.NilError(t, os.MkdirAll(filepath.Join(tmp, ".git"), 0o755))
	touch(t, filepath.Join(tmp, "pkg", "foo.go"))

	c := Config{
		RootMarkers: map[string][]MarkerGroup{
			"go": {{"go.mod"}, {".git"}},
		},
	}
	root, ok := c.FindRoot("go", filepath.Join(tmp, "pkg", "foo.go"))
	assert.Assert(t, ok)
	assert.Equal(t, root, tmp)
}

func TestFindRoot_MultipleMarkersInGroup(t *testing.T) {
	tmp := t.TempDir()
	touch(t, filepath.Join(tmp, "proj", "setup.py"))
	touch(t, filepath.Join(tmp, "proj", "src", "main.py"))

	c := Config{
		RootMarkers: map[string][]MarkerGroup{
			"python": {{"pyproject.toml", "setup.py"}},
		},
	}
	root, ok := c.FindRoot("python", filepath.Join(tmp, "proj", "src", "main.py"))
	assert.Assert(t, ok)
	assert.Equal(t, root, filepath.Join(tmp, "proj"))
}

func TestFindRoot_UnknownFiletype(t *testing.T) {
	c := Config{
		RootMarkers: map[string][]MarkerGroup{"go": {{"go.mod"}}},
	}
	_, ok := c.FindRoot("rust", "/x/foo.rs")
	assert.Assert(t, !ok)
}

func TestFindRoot_NoMatch(t *testing.T) {
	tmp := t.TempDir()
	touch(t, filepath.Join(tmp, "pkg", "foo.go"))

	c := Config{
		RootMarkers: map[string][]MarkerGroup{"go": {{"go.mod"}}},
	}
	_, ok := c.FindRoot("go", filepath.Join(tmp, "pkg", "foo.go"))
	assert.Assert(t, !ok)
}

func TestMerge_OverlayWins(t *testing.T) {
	base := Config{
		Ext:      map[string]string{"go": "go"},
		Filename: map[string]string{"go.mod": "go"},
		RootMarkers: map[string][]MarkerGroup{
			"go": {{"go.mod"}},
		},
	}
	overlay := Config{
		Ext: map[string]string{"go": "golang"}, // overrides
		Filename: map[string]string{
			"Cargo.toml": "rust", // adds
		},
		RootMarkers: map[string][]MarkerGroup{
			"go":   {{"go.work"}}, // overrides
			"rust": {{"Cargo.toml"}},
		},
	}
	got := base.Merge(overlay)

	// Overlay wins on conflict.
	assert.Equal(t, got.Ext["go"], "golang")
	assert.DeepEqual(t, got.RootMarkers["go"], []MarkerGroup{{"go.work"}})

	// Base values preserved when not overridden.
	assert.Equal(t, got.Filename["go.mod"], "go")

	// Overlay-only entries added.
	assert.Equal(t, got.Filename["Cargo.toml"], "rust")
	assert.DeepEqual(t, got.RootMarkers["rust"], []MarkerGroup{{"Cargo.toml"}})
}

func TestMergeAll_FoldsLeavesInOrder(t *testing.T) {
	leaves := []Config{
		{Ext: map[string]string{"x": "first"}},
		{Ext: map[string]string{"x": "second"}}, // wins
		{Ext: map[string]string{"y": "third"}},
	}
	got := MergeAll(leaves)
	assert.Equal(t, got.Ext["x"], "second")
	assert.Equal(t, got.Ext["y"], "third")
}

func TestMergeAll_EmptyYieldsZero(t *testing.T) {
	got := MergeAll(nil)
	assert.Assert(t, got.Ext == nil)
	assert.Assert(t, got.Filename == nil)
	assert.Assert(t, got.RootMarkers == nil)
}

func TestConfig_JSONRoundTrip(t *testing.T) {
	in := MergeAll(sampleLeaves())
	data, err := json.Marshal(in)
	assert.NilError(t, err)

	var out Config
	assert.NilError(t, json.Unmarshal(data, &out))
	assert.DeepEqual(t, in, out)
}

func TestLeaf_JSONShapeMatchesUserExample(t *testing.T) {
	// User's example leaf format:
	//   {"ext": {"mbt":"moonbit", "mbti":"moonbit"},
	//    "filename":{"moon.pkg":"moonbit", "moon.mod.json":"moonbit"}}
	raw := `{"ext":{"mbt":"moonbit","mbti":"moonbit"},` +
		`"filename":{"moon.pkg":"moonbit","moon.mod.json":"moonbit"}}`
	var leaf Config
	assert.NilError(t, json.Unmarshal([]byte(raw), &leaf))
	assert.Equal(t, leaf.Ext["mbt"], "moonbit")
	assert.Equal(t, leaf.Ext["mbti"], "moonbit")
	assert.Equal(t, leaf.Filename["moon.pkg"], "moonbit")
	assert.Equal(t, leaf.Filename["moon.mod.json"], "moonbit")
}
