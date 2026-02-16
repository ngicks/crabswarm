// Package tmux implements mux interfaces using the tmux command-line tool.
package tmux

import (
	"context"
	"fmt"
	"strings"

	"github.com/ngicks/crabswarm/pkg/mux"
)

// Config configures a tmux session.
type Config struct {
	// Name is the tmux session name.
	Name string
	// TmuxPath is the path to the tmux binary. Defaults to "tmux".
	TmuxPath string
	// SocketName is the tmux socket name (-L flag). Empty uses the default socket.
	SocketName string
}

// Session is a tmux session implementing mux.Session.
type Session struct {
	id   string
	name string
	exec *executor
}

// Verify interface compliance.
var _ mux.Session = (*Session)(nil)

// New creates a new detached tmux session with the given config.
func New(ctx context.Context, cfg Config) (*Session, error) {
	exec := newExecutor(cfg.TmuxPath, cfg.SocketName)

	out, err := exec.run(ctx, "new-session", "-d", "-s", cfg.Name, "-P", "-F", "#{session_id}")
	if err != nil {
		if strings.Contains(err.Error(), "duplicate session") {
			return nil, mux.ErrSessionExists
		}
		return nil, err
	}

	sess := &Session{
		id:   strings.TrimSpace(out),
		name: cfg.Name,
		exec: exec,
	}

	// Install hooks to rebalance all window panes on client attach/detach.
	// Splits done on detached sessions use default-size (80x24) and get
	// distorted when the window resizes to the real terminal dimensions.
	// Using run-shell to iterate all windows so every window is rebalanced.
	// Array index [100] avoids clobbering user-defined hooks.
	//
	// NOTE: ## escapes # for tmux format expansion in run-shell.
	// All dynamic values are shell-quoted with shellQuote() to prevent injection.
	rebalanceCmd := fmt.Sprintf(
		"run-shell 'for w in $(%s %slist-windows -t %s -F \"##{window_id}\"); do %s %sselect-layout -t \"$w\" tiled; done'",
		shellQuote(exec.tmuxPath), exec.socketFlag(),
		shellQuote(cfg.Name),
		shellQuote(exec.tmuxPath), exec.socketFlag(),
	)
	for _, hook := range []string{"client-attached[100]", "client-detached[100]"} {
		_, err = exec.run(ctx, "set-hook", "-t", cfg.Name, hook, rebalanceCmd)
		if err != nil {
			return nil, err
		}
	}

	return sess, nil
}

// Attach attaches to an existing tmux session.
func Attach(ctx context.Context, cfg Config) (*Session, error) {
	exec := newExecutor(cfg.TmuxPath, cfg.SocketName)

	_, err := exec.run(ctx, "has-session", "-t", cfg.Name)
	if err != nil {
		if strings.Contains(err.Error(), "no server running") ||
			strings.Contains(err.Error(), "can't find session") ||
			strings.Contains(err.Error(), "session not found") {
			return nil, mux.ErrSessionNotFound
		}
		return nil, err
	}

	out, err := exec.run(ctx, "display-message", "-t", cfg.Name, "-p", "#{session_id}")
	if err != nil {
		return nil, err
	}

	return &Session{
		id:   strings.TrimSpace(out),
		name: cfg.Name,
		exec: exec,
	}, nil
}

func (s *Session) Id() string {
	return s.id
}

func (s *Session) Name(ctx context.Context) (string, error) {
	out, err := s.exec.run(ctx, "display-message", "-t", s.name, "-p", "#{session_name}")
	if err != nil {
		return "", err
	}
	return out, nil
}

func (s *Session) NewWindow(ctx context.Context, name string) (mux.Window, error) {
	out, err := s.exec.run(ctx, "new-window", "-t", s.name, "-n", name, "-P", "-F", "#{window_id}")
	if err != nil {
		return nil, err
	}
	return &window{
		id:          strings.TrimSpace(out),
		sessionName: s.name,
		exec:        s.exec,
	}, nil
}

func (s *Session) List(ctx context.Context) ([]mux.Window, error) {
	out, err := s.exec.run(ctx, "list-windows", "-t", s.name, "-F", "#{window_id}\t#{window_index}\t#{window_name}")
	if err != nil {
		return nil, err
	}
	return parseWindows(out, s.name, s.exec), nil
}

func (s *Session) GetAt(ctx context.Context, i int) (mux.Window, error) {
	windows, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	if i < 0 || i >= len(windows) {
		return nil, mux.ErrWindowNotFound
	}
	return windows[i], nil
}

func (s *Session) GetById(ctx context.Context, id string) (mux.Window, error) {
	windows, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, w := range windows {
		if w.Id() == id {
			return w, nil
		}
	}
	return nil, mux.ErrWindowNotFound
}

func (s *Session) Close(ctx context.Context) error {
	_, err := s.exec.run(ctx, "kill-session", "-t", s.name)
	return err
}
