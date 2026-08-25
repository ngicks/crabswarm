package web

import (
	"bytes"
	"fmt"
	"io/fs"
	"testing"
	"testing/fstest"

	"golang.org/x/sync/errgroup"
)

func TestDistFS_satisfiesFSContract(t *testing.T) {
	if err := fstest.TestFS(DistFS(), "index.html"); err != nil {
		t.Fatal(err)
	}
}

// The embedded archive is decoded lazily through one shared seekable reader and
// zstd decoder, so every served asset is a concurrent ReadAt into that shared
// state. Run with -race.
func TestDistFS_concurrentReadsAreConsistent(t *testing.T) {
	fsys := DistFS()

	var files []string
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking dist: %v", err)
	}
	if len(files) < 3 {
		t.Fatalf("dist has %d files, want at least 3", len(files))
	}
	// Spread across the archive so the reads land in different zstd frames.
	paths := []string{files[0], files[len(files)/2], files[len(files)-1]}

	want := make(map[string][]byte, len(paths))
	for _, p := range paths {
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			t.Fatalf("reading %s: %v", p, err)
		}
		want[p] = data
	}

	var g errgroup.Group
	for range 8 {
		g.Go(func() error {
			for range 20 {
				for _, p := range paths {
					got, err := fs.ReadFile(fsys, p)
					if err != nil {
						return fmt.Errorf("reading %s: %w", p, err)
					}
					if !bytes.Equal(got, want[p]) {
						return fmt.Errorf("reading %s: got %d bytes, want %d", p, len(got), len(want[p]))
					}
				}
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		t.Fatal(err)
	}
}
