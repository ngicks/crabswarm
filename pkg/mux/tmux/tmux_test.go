package tmux

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/ngicks/crabswarm/pkg/mux"
)

// tmuxPath resolves the tmux binary. Fatal (not skip) if not found.
func tmuxPath(t *testing.T) string {
	t.Helper()
	p, err := exec.LookPath("tmux")
	if err != nil {
		t.Fatalf("tmux not found in PATH: %v", err)
	}
	return p
}

// testConfig returns a Config with a unique socket name derived from the test name.
func testConfig(t *testing.T, name string) Config {
	t.Helper()
	socket := strings.ReplaceAll(t.Name(), "/", "_")
	return Config{
		Name:       name,
		TmuxPath:   tmuxPath(t),
		SocketName: socket,
	}
}

// newTestSession creates a new tmux session and registers cleanup to kill the server.
func newTestSession(t *testing.T) *Session {
	t.Helper()
	cfg := testConfig(t, "test")
	ctx := context.Background()

	sess, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	t.Cleanup(func() {
		e := newExecutor(cfg.TmuxPath, cfg.SocketName)
		// kill-server shuts down the entire server on this socket.
		_, _ = e.run(context.Background(), "kill-server")
	})

	return sess
}

// pollCapture polls Capture until output contains match or timeout expires.
func pollCapture(t *testing.T, p mux.Pane, match string, timeout time.Duration) string {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := p.Capture(ctx, 0, 50)
		if err != nil {
			t.Fatalf("Capture: %v", err)
		}
		if strings.Contains(out, match) {
			return out
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("pollCapture: %q not found within %v", match, timeout)
	return ""
}

func TestTmuxNew(t *testing.T) {
	tmuxPath(t)
	cfg := testConfig(t, "newsess")
	ctx := context.Background()

	sess, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		e := newExecutor(cfg.TmuxPath, cfg.SocketName)
		_, _ = e.run(context.Background(), "kill-server")
	})

	if !strings.HasPrefix(sess.Id(), "$") {
		t.Errorf("session ID = %q, want prefix $", sess.Id())
	}

	// Duplicate session should return ErrSessionExists.
	_, err = New(ctx, cfg)
	if !errors.Is(err, mux.ErrSessionExists) {
		t.Errorf("duplicate New: got %v, want %v", err, mux.ErrSessionExists)
	}
}

func TestTmuxAttach(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()
	cfg := testConfig(t, "test")

	attached, err := Attach(ctx, cfg)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	if attached.Id() != sess.Id() {
		t.Errorf("attached ID = %q, want %q", attached.Id(), sess.Id())
	}

	// Nonexistent session should return ErrSessionNotFound.
	badCfg := cfg
	badCfg.Name = "nonexistent"
	_, err = Attach(ctx, badCfg)
	if !errors.Is(err, mux.ErrSessionNotFound) {
		t.Errorf("Attach nonexistent: got %v, want %v", err, mux.ErrSessionNotFound)
	}
}

func TestTmuxSessionName(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()

	name, err := sess.Name(ctx)
	if err != nil {
		t.Fatalf("Name: %v", err)
	}
	if name != "test" {
		t.Errorf("Name = %q, want %q", name, "test")
	}
}

func TestTmuxSessionNewWindow(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()

	w, err := sess.NewWindow(ctx, "mywin")
	if err != nil {
		t.Fatalf("NewWindow: %v", err)
	}
	if !strings.HasPrefix(w.Id(), "@") {
		t.Errorf("window ID = %q, want prefix @", w.Id())
	}
	name, err := w.Name(ctx)
	if err != nil {
		t.Fatalf("window.Name: %v", err)
	}
	if name != "mywin" {
		t.Errorf("window name = %q, want %q", name, "mywin")
	}
}

func TestTmuxSessionList(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()

	// Initially 1 window.
	windows, err := sess.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(windows) != 1 {
		t.Fatalf("initial window count = %d, want 1", len(windows))
	}

	// Add 2 more windows.
	for _, name := range []string{"win1", "win2"} {
		if _, err := sess.NewWindow(ctx, name); err != nil {
			t.Fatalf("NewWindow(%q): %v", name, err)
		}
	}

	windows, err = sess.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(windows) != 3 {
		t.Errorf("window count = %d, want 3", len(windows))
	}
}

func TestTmuxSessionGetAt(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()

	w, err := sess.GetAt(ctx, 0)
	if err != nil {
		t.Fatalf("GetAt(0): %v", err)
	}
	if w.Id() == "" {
		t.Error("GetAt(0) returned empty ID")
	}

	// Out of bounds.
	_, err = sess.GetAt(ctx, 99)
	if !errors.Is(err, mux.ErrWindowNotFound) {
		t.Errorf("GetAt(99): got %v, want %v", err, mux.ErrWindowNotFound)
	}

	// Negative.
	_, err = sess.GetAt(ctx, -1)
	if !errors.Is(err, mux.ErrWindowNotFound) {
		t.Errorf("GetAt(-1): got %v, want %v", err, mux.ErrWindowNotFound)
	}
}

func TestTmuxSessionGetById(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()

	windows, err := sess.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	knownID := windows[0].Id()

	w, err := sess.GetById(ctx, knownID)
	if err != nil {
		t.Fatalf("GetById(%q): %v", knownID, err)
	}
	if w.Id() != knownID {
		t.Errorf("GetById returned ID = %q, want %q", w.Id(), knownID)
	}

	// Unknown ID.
	_, err = sess.GetById(ctx, "@99999")
	if !errors.Is(err, mux.ErrWindowNotFound) {
		t.Errorf("GetById unknown: got %v, want %v", err, mux.ErrWindowNotFound)
	}
}

func TestTmuxSessionClose(t *testing.T) {
	tmuxPath(t)
	cfg := testConfig(t, "closesess")
	ctx := context.Background()

	sess, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		e := newExecutor(cfg.TmuxPath, cfg.SocketName)
		_, _ = e.run(context.Background(), "kill-server")
	})

	if err := sess.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Attach should fail.
	_, err = Attach(ctx, cfg)
	if !errors.Is(err, mux.ErrSessionNotFound) {
		t.Errorf("Attach after Close: got %v, want %v", err, mux.ErrSessionNotFound)
	}
}

func TestTmuxWindowName(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()

	w, err := sess.NewWindow(ctx, "namedwin")
	if err != nil {
		t.Fatalf("NewWindow: %v", err)
	}
	name, err := w.Name(ctx)
	if err != nil {
		t.Fatalf("Name: %v", err)
	}
	if name != "namedwin" {
		t.Errorf("Name = %q, want %q", name, "namedwin")
	}
}

func TestTmuxWindowSplit(t *testing.T) {
	tests := []struct {
		name      string
		splitN    int
		wantPanes int
	}{
		{"split 0 is noop", 0, 1},
		{"split 1 adds 1 pane", 1, 2},
		{"split 3 adds 3 panes", 3, 4},
		{"split 7 adds 7 panes", 7, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess := newTestSession(t)
			ctx := context.Background()

			w, err := sess.GetAt(ctx, 0)
			if err != nil {
				t.Fatalf("GetAt(0): %v", err)
			}

			if err := w.Split(ctx, tt.splitN); err != nil {
				t.Fatalf("Split(%d): %v", tt.splitN, err)
			}

			panes, err := w.List(ctx)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(panes) != tt.wantPanes {
				t.Errorf("pane count = %d, want %d", len(panes), tt.wantPanes)
			}
		})
	}
}

func TestTmuxWindowList(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()

	w, err := sess.GetAt(ctx, 0)
	if err != nil {
		t.Fatalf("GetAt(0): %v", err)
	}

	// Initially 1 pane.
	panes, err := w.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(panes) != 1 {
		t.Fatalf("initial pane count = %d, want 1", len(panes))
	}

	// Split to add 2 more.
	if err := w.Split(ctx, 2); err != nil {
		t.Fatalf("Split(2): %v", err)
	}

	panes, err = w.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(panes) != 3 {
		t.Errorf("pane count = %d, want 3", len(panes))
	}
}

func TestTmuxWindowGetAtAndGetById(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()

	w, err := sess.GetAt(ctx, 0)
	if err != nil {
		t.Fatalf("GetAt(0): %v", err)
	}

	p, err := w.GetAt(ctx, 0)
	if err != nil {
		t.Fatalf("pane GetAt(0): %v", err)
	}
	if p.Id() == "" {
		t.Error("pane GetAt(0) returned empty ID")
	}

	// GetById with known ID.
	p2, err := w.GetById(ctx, p.Id())
	if err != nil {
		t.Fatalf("pane GetById(%q): %v", p.Id(), err)
	}
	if p2.Id() != p.Id() {
		t.Errorf("GetById returned ID = %q, want %q", p2.Id(), p.Id())
	}

	// Out of bounds.
	_, err = w.GetAt(ctx, 99)
	if !errors.Is(err, mux.ErrPaneNotFound) {
		t.Errorf("pane GetAt(99): got %v, want %v", err, mux.ErrPaneNotFound)
	}

	// Unknown ID.
	_, err = w.GetById(ctx, "%99999")
	if !errors.Is(err, mux.ErrPaneNotFound) {
		t.Errorf("pane GetById unknown: got %v, want %v", err, mux.ErrPaneNotFound)
	}
}

func TestTmuxWindowIndex(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()

	// Create extra windows so we have multiple.
	if _, err := sess.NewWindow(ctx, "w1"); err != nil {
		t.Fatalf("NewWindow: %v", err)
	}
	if _, err := sess.NewWindow(ctx, "w2"); err != nil {
		t.Fatalf("NewWindow: %v", err)
	}

	windows, err := sess.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	for i, w := range windows {
		idx, err := w.Index(ctx)
		if err != nil {
			t.Fatalf("windows[%d].Index: %v", i, err)
		}
		// Index should match position in list (tmux default base-index is 0).
		if idx != i {
			t.Errorf("windows[%d].Index = %d, want %d", i, idx, i)
		}
	}
}

func TestTmuxWindowClose(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()

	// Add a window so we can close it without killing the session.
	w, err := sess.NewWindow(ctx, "toclose")
	if err != nil {
		t.Fatalf("NewWindow: %v", err)
	}
	wID := w.Id()

	if err := w.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Should not be in listing anymore.
	_, err = sess.GetById(ctx, wID)
	if !errors.Is(err, mux.ErrWindowNotFound) {
		t.Errorf("GetById after Close: got %v, want %v", err, mux.ErrWindowNotFound)
	}
}

func TestTmuxPaneName(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()

	w, err := sess.GetAt(ctx, 0)
	if err != nil {
		t.Fatalf("GetAt(0): %v", err)
	}
	p, err := w.GetAt(ctx, 0)
	if err != nil {
		t.Fatalf("pane GetAt(0): %v", err)
	}

	name, err := p.Name(ctx)
	if err != nil {
		t.Fatalf("pane Name: %v", err)
	}
	if name == "" {
		t.Error("pane Name returned empty string")
	}
}

func TestTmuxPaneSendKeysAndCapture(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()

	w, err := sess.GetAt(ctx, 0)
	if err != nil {
		t.Fatalf("GetAt(0): %v", err)
	}
	p, err := w.GetAt(ctx, 0)
	if err != nil {
		t.Fatalf("pane GetAt(0): %v", err)
	}

	if err := p.SendKeys(ctx, []string{"echo hello", "Enter"}); err != nil {
		t.Fatalf("SendKeys: %v", err)
	}

	pollCapture(t, p, "hello", 3*time.Second)
}

func TestTmuxPaneSendKeysEmpty(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()

	w, err := sess.GetAt(ctx, 0)
	if err != nil {
		t.Fatalf("GetAt(0): %v", err)
	}
	p, err := w.GetAt(ctx, 0)
	if err != nil {
		t.Fatalf("pane GetAt(0): %v", err)
	}

	_ = sess // keep reference

	if err := p.SendKeys(ctx, nil); err != nil {
		t.Errorf("SendKeys(nil): %v", err)
	}
	if err := p.SendKeys(ctx, []string{}); err != nil {
		t.Errorf("SendKeys([]string{}): %v", err)
	}
}

func TestTmuxPaneCaptureEdge(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()

	w, err := sess.GetAt(ctx, 0)
	if err != nil {
		t.Fatalf("GetAt(0): %v", err)
	}
	p, err := w.GetAt(ctx, 0)
	if err != nil {
		t.Fatalf("pane GetAt(0): %v", err)
	}

	_ = sess

	// Capture(0, 0) should not error.
	_, err = p.Capture(ctx, 0, 0)
	if err != nil {
		t.Errorf("Capture(0, 0): %v", err)
	}
}

func TestTmuxPaneIndex(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()

	w, err := sess.GetAt(ctx, 0)
	if err != nil {
		t.Fatalf("GetAt(0): %v", err)
	}

	if err := w.Split(ctx, 2); err != nil {
		t.Fatalf("Split(2): %v", err)
	}

	panes, err := w.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	_ = sess

	for i, p := range panes {
		idx, err := p.Index(ctx)
		if err != nil {
			t.Fatalf("panes[%d].Index: %v", i, err)
		}
		if idx != i {
			t.Errorf("panes[%d].Index = %d, want %d", i, idx, i)
		}
	}
}

func TestTmuxPaneClose(t *testing.T) {
	sess := newTestSession(t)
	ctx := context.Background()

	w, err := sess.GetAt(ctx, 0)
	if err != nil {
		t.Fatalf("GetAt(0): %v", err)
	}

	// Split to have 2 panes so we can close one.
	if err := w.Split(ctx, 1); err != nil {
		t.Fatalf("Split(1): %v", err)
	}

	panes, err := w.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(panes) != 2 {
		t.Fatalf("pane count = %d, want 2", len(panes))
	}

	closedID := panes[1].Id()
	if err := panes[1].Close(ctx); err != nil {
		t.Fatalf("pane Close: %v", err)
	}

	_ = sess

	// Should not be found.
	_, err = w.GetById(ctx, closedID)
	if !errors.Is(err, mux.ErrPaneNotFound) {
		t.Errorf("GetById after Close: got %v, want %v", err, mux.ErrPaneNotFound)
	}
}
