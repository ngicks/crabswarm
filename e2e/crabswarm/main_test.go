package crabswarm_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// crabswarmBin is the path to the built crabswarm binary, set once by TestMain.
var crabswarmBin string

func TestMain(m *testing.M) {
	// Build the crabswarm binary into a temp directory.
	tmp, err := os.MkdirTemp("", "crabswarm-e2e-bin-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	bin := filepath.Join(tmp, "crabswarm")
	build := exec.Command("go", "build", "-o", bin, "./cmd/crabswarm")
	// Build from the repository root.
	build.Dir = repoRoot()
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build crabswarm: %v\n", err)
		os.Exit(1)
	}
	crabswarmBin = bin

	os.Exit(m.Run())
}

func repoRoot() string {
	// Walk up from the test file location to find go.mod.
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("could not find repo root")
		}
		dir = parent
	}
}

// freeAddr reserves an ephemeral loopback port and returns it as host:port. The
// listener is closed before returning so the port is free for the process under
// test to bind. A small race window (another process could grab the port in
// between) is acceptable for e2e: the preview `__serve` command binds a concrete
// port rather than port 0, which would otherwise be unknowable to the caller.
func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve free port: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close reserved listener: %v", err)
	}
	return addr
}

// writeFile writes content to path, failing the test on error.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// waitHealthz polls the preview server's /healthz endpoint until it answers 200
// or the timeout elapses. addr is the server's loopback listen address.
func waitHealthz(ctx context.Context, t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	url := "http://" + addr + "/healthz"
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			t.Fatalf("build healthz request: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("preview server at %s did not become healthy within %s", addr, timeout)
}
