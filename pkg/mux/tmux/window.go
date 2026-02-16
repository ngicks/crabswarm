package tmux

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ngicks/crabswarm/pkg/mux"
)

// window implements mux.Window for a tmux window.
type window struct {
	id          string
	sessionName string
	exec        *executor
}

var _ mux.Window = (*window)(nil)

func (w *window) Id() string {
	return w.id
}

func (w *window) target() string {
	return w.sessionName + ":" + w.id
}

func (w *window) Index(ctx context.Context) (int, error) {
	out, err := w.exec.run(ctx, "display-message", "-t", w.target(), "-p", "#{window_index}")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(out)
}

func (w *window) Name(ctx context.Context) (string, error) {
	out, err := w.exec.run(ctx, "display-message", "-t", w.target(), "-p", "#{window_name}")
	if err != nil {
		return "", err
	}
	return out, nil
}

// Split adds n panes to the window using a stateless deterministic algorithm.
// Each iteration queries the current pane count to derive the round and step.
func (w *window) Split(ctx context.Context, n int) error {
	paneIDs, err := w.listPaneIDs(ctx)
	if err != nil {
		return err
	}

	numPane := len(paneIDs)

	for range n {
		if numPane == 0 {
			return fmt.Errorf("tmux: window %s has no panes", w.id)
		}

		direction, index := splitTargetPaneIndex(numPane)
		if index >= numPane {
			return fmt.Errorf("tmux: target pane index %d out of range (have %d panes)", index, numPane)
		}

		_, err = w.exec.run(ctx, "split-window", direction, "-t", w.sessionName+":"+w.id+"."+strconv.Itoa(index))
		if err != nil {
			return err
		}

		numPane++
	}
	return nil
}

func (w *window) List(ctx context.Context) ([]mux.Pane, error) {
	out, err := w.exec.run(ctx, "list-panes", "-t", w.target(), "-F", "#{pane_id}\t#{pane_index}")
	if err != nil {
		return nil, err
	}
	return parsePanes(out, w.target(), w.exec), nil
}

func (w *window) GetAt(ctx context.Context, i int) (mux.Pane, error) {
	panes, err := w.List(ctx)
	if err != nil {
		return nil, err
	}
	if i < 0 || i >= len(panes) {
		return nil, mux.ErrPaneNotFound
	}
	return panes[i], nil
}

func (w *window) GetById(ctx context.Context, id string) (mux.Pane, error) {
	panes, err := w.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range panes {
		if p.Id() == id {
			return p, nil
		}
	}
	return nil, mux.ErrPaneNotFound
}

func (w *window) Close(ctx context.Context) error {
	_, err := w.exec.run(ctx, "kill-window", "-t", w.target())
	return err
}

// listPaneIDs returns pane IDs in visual order.
func (w *window) listPaneIDs(ctx context.Context) ([]string, error) {
	out, err := w.exec.run(ctx, "list-panes", "-t", w.target(), "-F", "#{pane_id}")
	if err != nil {
		return nil, err
	}
	return parsePaneIDs(out), nil
}
