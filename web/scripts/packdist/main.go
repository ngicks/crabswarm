// Command packdist packs the vite build output (web/dist) into
// web/dist.tar.zst, the seekable-zstd tar that web/embed.go embeds.
//
// It is run from web/ as the last step of `pnpm build`. The tar headers are
// built by hand rather than through tar.FileInfoHeader so that no on-disk
// metadata (mtime, uid/gid, umask-dependent mode bits) leaks into the archive:
// the blob is committed, so byte-for-byte reproducibility from the same dist is
// what keeps rebuild-and-diff CI checks meaningful.
package main

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"

	seekable "github.com/SaveTheRbtz/zstd-seekable-format-go/pkg"
	"github.com/klauspost/compress/zstd"
)

const (
	distDir = "dist"
	outPath = "dist.tar.zst"

	// Each Write into the seekable writer becomes one independently
	// decompressible zstd frame; 512KiB trades a slightly larger blob for
	// random access that does not decode the whole archive to read one asset.
	frameSize = 512 << 10
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "packdist:", err)
		os.Exit(1)
	}
}

func run() error {
	tarball, err := buildTar(distDir)
	if err != nil {
		return err
	}
	if err := compress(tarball, outPath); err != nil {
		return err
	}
	stat, err := os.Stat(outPath)
	if err != nil {
		return err
	}
	fmt.Printf("packdist: %s -> %s (%d -> %d bytes)\n", distDir, outPath, len(tarball), stat.Size())
	return nil
}

// buildTar walks dir in lexical order and returns a USTAR archive of it whose
// bytes depend only on the file names, modes and contents.
func buildTar(dir string) ([]byte, error) {
	fsys := os.DirFS(dir)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case path == ".":
			return nil
		case d.IsDir():
			return tw.WriteHeader(&tar.Header{
				Format:   tar.FormatUSTAR,
				Typeflag: tar.TypeDir,
				Name:     path + "/",
				Mode:     0o755,
			})
		case d.Type().IsRegular():
			data, err := fs.ReadFile(fsys, path)
			if err != nil {
				return err
			}
			err = tw.WriteHeader(&tar.Header{
				Format:   tar.FormatUSTAR,
				Typeflag: tar.TypeReg,
				Name:     path,
				Mode:     0o644,
				Size:     int64(len(data)),
			})
			if err != nil {
				return err
			}
			_, err = tw.Write(data)
			return err
		default:
			return fmt.Errorf("%s: mode %s: only directories and regular files can be packed", path, d.Type())
		}
	})
	if err != nil {
		return nil, fmt.Errorf("archiving %s: %w", dir, err)
	}
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("archiving %s: %w", dir, err)
	}
	return buf.Bytes(), nil
}

// compress writes tarball to path as a seekable zstd stream: one frame per
// frameSize chunk, followed by the seek table that makes random access work.
func compress(tarball []byte, path string) (err error) {
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return fmt.Errorf("creating zstd encoder: %w", err)
	}
	defer func() { err = errors.Join(err, enc.Close()) }()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()

	// LIFO: the seek table is flushed before the file is closed.
	sw, err := seekable.NewWriter(f, enc)
	if err != nil {
		return fmt.Errorf("creating seekable writer: %w", err)
	}
	defer func() { err = errors.Join(err, sw.Close()) }()

	for rest := tarball; len(rest) > 0; {
		chunk := rest
		if len(chunk) > frameSize {
			chunk = chunk[:frameSize]
		}
		rest = rest[len(chunk):]
		if _, err := sw.Write(chunk); err != nil {
			return fmt.Errorf("compressing %s: %w", path, err)
		}
	}
	return nil
}
