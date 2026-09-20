package supervisor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/pkg/util/poll"
)

// fakeSupervisor answers from the states it was seeded with and records every
// call in order, so a case can assert on the sequence Start drove.
type fakeSupervisor struct {
	// states maps a name to the state Inspect reports. A name missing from it is
	// reported as ErrNotFound.
	states map[string]State
	// ready, when non-empty, is the file Run creates, standing in for the
	// readiness signal a real supervised command would produce.
	ready string
	// runErr, when non-nil, is what Run fails with.
	runErr error

	calls []string
}

func (f *fakeSupervisor) Inspect(_ context.Context, name string) (State, error) {
	f.calls = append(f.calls, "inspect "+name)
	state, ok := f.states[name]
	if !ok {
		return "", fmt.Errorf("fake: %q: %w", name, ErrNotFound)
	}
	return state, nil
}

func (f *fakeSupervisor) Remove(_ context.Context, name string) error {
	f.calls = append(f.calls, "rm "+name)
	delete(f.states, name)
	return nil
}

// Run records the name quoted, so the anonymous case reads as an empty name
// rather than as a stray space.
func (f *fakeSupervisor) Run(_ context.Context, name string, command []string) error {
	f.calls = append(f.calls, fmt.Sprintf("run %q %s", name, strings.Join(command, " ")))
	if f.runErr != nil {
		return f.runErr
	}
	if f.ready != "" {
		if err := os.WriteFile(f.ready, nil, 0o600); err != nil {
			return err
		}
	}
	f.states[name] = StateRunning
	return nil
}

// fastPoll keeps every wait in these tests on a millisecond budget.
func fastPoll() poll.WaitOption {
	return poll.WaitOption{
		Interval:     10 * time.Millisecond,
		Retries:      1,
		ProbeTimeout: time.Second,
	}
}

func TestStartAnonymous(t *testing.T) {
	t.Parallel()

	sup := &fakeSupervisor{states: map[string]State{}}

	err := Start(t.Context(), nil, sup, StartOption{Command: []string{"sleep", "1"}})
	assert.NilError(t, err)

	// No name to collide with, so no inspect and no rm: just the run.
	assert.DeepEqual(t, sup.calls, []string{`run "" sleep 1`})
}

func TestStartNamedAbsent(t *testing.T) {
	t.Parallel()

	sup := &fakeSupervisor{states: map[string]State{}}

	err := Start(t.Context(), nil, sup, StartOption{
		Name:    "web",
		Command: []string{"sleep", "1"},
	})
	assert.NilError(t, err)

	// An absent command must not be rm'd before running.
	assert.DeepEqual(t, sup.calls, []string{"inspect web", `run "web" sleep 1`})
}

func TestStartNamedAlreadyRunning(t *testing.T) {
	t.Parallel()

	sup := &fakeSupervisor{states: map[string]State{"web": StateRunning}}

	err := Start(t.Context(), nil, sup, StartOption{
		Name:    "web",
		Command: []string{"sleep", "1"},
	})
	assert.NilError(t, err)

	assert.DeepEqual(t, sup.calls, []string{"inspect web"})
}

func TestStartNamedExited(t *testing.T) {
	t.Parallel()

	sup := &fakeSupervisor{states: map[string]State{"web": State("exited")}}

	err := Start(t.Context(), nil, sup, StartOption{
		Name:    "web",
		Command: []string{"sleep", "1"},
	})
	assert.NilError(t, err)

	// inspect -> rm (the exited command still holds the name) -> run.
	assert.DeepEqual(t, sup.calls, []string{"inspect web", "rm web", `run "web" sleep 1`})
}

func TestStartEmptyCommand(t *testing.T) {
	t.Parallel()

	sup := &fakeSupervisor{states: map[string]State{}}

	err := Start(t.Context(), nil, sup, StartOption{Name: "web"})
	assert.Assert(t, err != nil)
	assert.Equal(t, len(sup.calls), 0, "nothing may be started")
}

func TestStartRunFails(t *testing.T) {
	t.Parallel()

	sup := &fakeSupervisor{
		states: map[string]State{},
		runErr: errors.New("supervisor refused the run"),
	}

	err := Start(t.Context(), nil, sup, StartOption{
		Name:    "web",
		Command: []string{"sleep", "1"},
	})
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "supervisor refused the run"), "got %v", err)
}

func TestStartPollsTarget(t *testing.T) {
	t.Parallel()

	ready := filepath.Join(t.TempDir(), "ready")
	sup := &fakeSupervisor{states: map[string]State{}, ready: ready}

	// The readiness file only exists once the fake's Run created it, so the wait
	// really rides on the start.
	_, err := os.Stat(ready)
	assert.Assert(t, errors.Is(err, fs.ErrNotExist), "got %v", err)

	target, err := poll.ParseTarget("file://" + ready)
	assert.NilError(t, err)

	err = Start(t.Context(), nil, sup, StartOption{
		Command:     []string{"sleep", "1"},
		Poll:        &target,
		PollOptions: fastPoll(),
	})
	assert.NilError(t, err)
	assert.Equal(t, len(sup.calls), 1)
}

func TestStartPollTargetNeverReady(t *testing.T) {
	t.Parallel()

	sup := &fakeSupervisor{states: map[string]State{}}

	missing := filepath.Join(t.TempDir(), "never")
	target := poll.Target{Kind: poll.KindFile, Addr: missing}

	err := Start(t.Context(), nil, sup, StartOption{
		Command:     []string{"sleep", "1"},
		Poll:        &target,
		PollOptions: fastPoll(),
	})
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), missing), "got %v", err)
	// The command started; only the readiness wait failed.
	assert.Equal(t, len(sup.calls), 1)
}
