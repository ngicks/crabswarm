package poll

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

// fastWaitOption keeps every Wait test on a millisecond budget.
func fastWaitOption() WaitOption {
	return WaitOption{
		Interval:     10 * time.Millisecond,
		Retries:      3,
		ProbeTimeout: time.Second,
	}
}

// debugLogger exercises the debug records Wait emits per failed probe.
func debugLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

func TestWaitUnix(t *testing.T) {
	// The socket path stays short so it fits the sun_path limit.
	sock := filepath.Join(t.TempDir(), "s.sock")
	ln, err := net.Listen("unix", sock)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	// The listener never accepts; the backlog is enough for the dial to succeed.
	target := Target{Kind: KindUnix, Addr: sock}
	assert.NilError(t, Wait(t.Context(), nil, target, fastWaitOption()))
}

func TestWaitFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ready")
	assert.NilError(t, os.WriteFile(path, nil, 0o600))

	target := Target{Kind: KindFile, Addr: path}
	assert.NilError(t, Wait(t.Context(), nil, target, fastWaitOption()))
}

func TestWaitHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	target, err := ParseTarget(srv.URL + "/healthz")
	assert.NilError(t, err)
	assert.NilError(t, Wait(t.Context(), nil, target, fastWaitOption()))
}

func TestWaitGivesUp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "never")

	err := Wait(t.Context(), debugLogger(), Target{Kind: KindFile, Addr: path}, fastWaitOption())
	assert.Assert(t, err != nil)
	assert.Assert(t, errors.Is(err, fs.ErrNotExist), "got %v", err)
	assert.Assert(t, strings.Contains(err.Error(), path), "got %v", err)
}

// flakyServer answers 503 to its first failures requests and 200 afterwards.
// The returned counter holds how many requests it has served.
func flakyServer(t *testing.T, failures int32) (url string, served *atomic.Int32) {
	t.Helper()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) <= failures {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, &calls
}

// stampedServer answers 200 and records how many requests it served and when
// the first one arrived.
func stampedServer(t *testing.T) (url string, served func() (calls int, first time.Time)) {
	t.Helper()

	var (
		mu    sync.Mutex
		calls int
		first time.Time
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		calls++
		if calls == 1 {
			first = time.Now()
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, func() (int, time.Time) {
		mu.Lock()
		defer mu.Unlock()
		return calls, first
	}
}

func TestWaitStartPeriod(t *testing.T) {
	t.Run("nothing is probed before it elapses", func(t *testing.T) {
		const startPeriod = 100 * time.Millisecond

		url, served := stampedServer(t)
		target, err := ParseTarget(url)
		assert.NilError(t, err)

		opts := WaitOption{
			StartPeriod:  startPeriod,
			Interval:     10 * time.Millisecond,
			Retries:      3,
			ProbeTimeout: time.Second,
		}
		start := time.Now()
		assert.NilError(t, Wait(t.Context(), debugLogger(), target, opts))

		// The server answers every request with 200, so the one request it saw
		// is the first probe, and an earlier probe would have stamped an
		// earlier arrival.
		calls, first := served()
		assert.Equal(t, calls, 1)
		assert.Assert(t, first.Sub(start) >= startPeriod,
			"first probe arrived %s after Wait began, want at least %s",
			first.Sub(start), startPeriod)
	})

	t.Run("ctx cancelled during it ends the wait", func(t *testing.T) {
		url, served := stampedServer(t)
		target, err := ParseTarget(url)
		assert.NilError(t, err)

		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancel()

		// The start period outlasts ctx by far, so returning at all proves the
		// sleep honoured the cancellation.
		opts := WaitOption{
			StartPeriod:  time.Minute,
			Interval:     10 * time.Millisecond,
			Retries:      -1,
			ProbeTimeout: time.Second,
		}
		start := time.Now()
		err = Wait(ctx, nil, target, opts)
		elapsed := time.Since(start)

		assert.Assert(t, errors.Is(err, context.DeadlineExceeded), "got %v", err)
		calls, _ := served()
		assert.Equal(t, calls, 0)
		assert.Assert(t, elapsed < time.Second, "Wait took %s", elapsed)
	})

	t.Run("a failure after it spends a retry", func(t *testing.T) {
		url, served := flakyServer(t, 1)
		target, err := ParseTarget(url)
		assert.NilError(t, err)

		opts := WaitOption{
			StartPeriod:  10 * time.Millisecond,
			Interval:     10 * time.Millisecond,
			Retries:      1,
			ProbeTimeout: time.Second,
		}
		assert.Assert(t, Wait(t.Context(), nil, target, opts) != nil)
		assert.Equal(t, served.Load(), int32(1))
	})
}

func TestWaitContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()

	// Retries is unlimited, so only ctx can end this wait.
	opts := WaitOption{Interval: 10 * time.Millisecond, Retries: -1, ProbeTimeout: time.Second}
	target := Target{Kind: KindFile, Addr: filepath.Join(t.TempDir(), "never")}

	start := time.Now()
	err := Wait(ctx, nil, target, opts)
	elapsed := time.Since(start)

	assert.Assert(t, errors.Is(err, context.DeadlineExceeded), "got %v", err)
	assert.Assert(t, errors.Is(err, fs.ErrNotExist), "got %v", err)
	assert.Assert(t, elapsed < time.Second, "Wait took %s", elapsed)
}

func TestWaitUnknownKind(t *testing.T) {
	err := Wait(t.Context(), nil, Target{Kind: Kind(42), Addr: "x"}, fastWaitOption())
	assert.Assert(t, err != nil)
}
