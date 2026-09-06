package issues

import (
	"bytes"
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"
)

// stubLister replays one listing per call, repeating the last one once the
// script runs out. It stands in for bd where a test needs the listing to move
// between two polls.
type stubLister struct {
	listings [][]Summary
	calls    int
}

func (s *stubLister) List(_ context.Context, _ ListFilter) ([]Summary, error) {
	i := min(s.calls, len(s.listings)-1)
	s.calls++
	return s.listings[i], nil
}

// at is a fixed instant offset by n minutes, so a test can move one issue's
// update time without caring about the clock.
func at(n int) time.Time {
	return time.Date(2026, 9, 6, 11, 0, 0, 0, time.UTC).Add(time.Duration(n) * time.Minute)
}

func TestPollerDiff(t *testing.T) {
	lister := &stubLister{listings: [][]Summary{
		{
			{ID: "a", UpdatedAt: at(0)},
			{ID: "b", UpdatedAt: at(0)},
			{ID: "gone", UpdatedAt: at(0)},
		},
		{
			{ID: "a", UpdatedAt: at(0)}, // untouched
			{ID: "b", UpdatedAt: at(1)}, // updated
			{ID: "new", UpdatedAt: at(1)},
		},
	}}

	var emitted [][]string
	p := NewPoller(nil, "src-1", lister, time.Minute,
		func(sourceID string, issueIDs []string) {
			assert.Equal(t, sourceID, "src-1")
			emitted = append(emitted, issueIDs)
		})

	// The first poll only records the baseline: everything looks new to a
	// poller that has never listed.
	p.poll(t.Context())
	assert.Equal(t, len(emitted), 0)

	p.poll(t.Context())
	assert.Equal(t, len(emitted), 1)
	// An issue that vanished from the listing changed too: closing one drops
	// it from bd's default view instead of bumping its update time.
	assert.DeepEqual(t, emitted[0], []string{"b", "gone", "new"})

	// A poll that finds the same listing again reports nothing.
	p.poll(t.Context())
	assert.Equal(t, len(emitted), 1)
}

func TestPollerRefreshEmitsOutsideTick(t *testing.T) {
	lister := &stubLister{listings: [][]Summary{
		{{ID: "a", UpdatedAt: at(0)}},
		{{ID: "a", UpdatedAt: at(1)}, {ID: "b", UpdatedAt: at(1)}},
	}}

	var emitted [][]string
	p := NewPoller(nil, "src-1", lister, time.Minute,
		func(sourceID string, issueIDs []string) {
			assert.Equal(t, sourceID, "src-1")
			emitted = append(emitted, issueIDs)
		})

	// A tick records the baseline.
	p.poll(t.Context())
	assert.Equal(t, len(emitted), 0)

	// A refresh between two ticks returns the listing it just read and
	// reports what moved, so nobody has to wait for the next tick to hear
	// about a change this listing already swallowed.
	listed, err := p.Refresh(t.Context())
	assert.NilError(t, err)
	assert.DeepEqual(t, listed, lister.listings[1])
	assert.Equal(t, len(emitted), 1)
	assert.DeepEqual(t, emitted[0], []string{"a", "b"})

	// The refresh moved the baseline, so the next tick finds nothing new.
	p.poll(t.Context())
	assert.Equal(t, len(emitted), 1)
}

func TestPollerRefreshCollapsesCallers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		baseline := []Summary{{ID: "a", UpdatedAt: at(0)}}
		moved := []Summary{{ID: "a", UpdatedAt: at(1)}}

		var calls atomic.Int32
		release := make(chan struct{})
		lister := listerFunc(func(_ context.Context, _ ListFilter) ([]Summary, error) {
			if calls.Add(1) == 1 {
				return baseline, nil
			}
			// Hold the listing open so every caller below joins this one run.
			<-release
			return moved, nil
		})

		var emitted [][]string
		p := NewPoller(nil, "src-1", lister, time.Minute,
			func(_ string, issueIDs []string) { emitted = append(emitted, issueIDs) })

		// Prime the baseline first: a collapsed run of a primed poller emits
		// exactly one event, which an unprimed one could not tell from a run
		// that never collapsed.
		_, err := p.Refresh(t.Context())
		assert.NilError(t, err)

		const callers = 8
		listings := make([][]Summary, callers)
		var eg errgroup.Group
		for i := range callers {
			eg.Go(func() error {
				listed, err := p.Refresh(t.Context())
				listings[i] = listed
				return err
			})
		}

		// Every caller is parked on the run, and the run itself is parked
		// inside the lister, so releasing now cannot let a second listing
		// start.
		synctest.Wait()
		close(release)
		assert.NilError(t, eg.Wait())

		// The baseline plus the one run the burst shared.
		assert.Equal(t, int(calls.Load()), 2)
		assert.Equal(t, len(emitted), 1)
		assert.DeepEqual(t, emitted[0], []string{"a"})
		for _, listed := range listings {
			assert.DeepEqual(t, listed, moved)
		}
	})
}

func TestPollerRefreshOutlivesCancelledCaller(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		baseline := []Summary{{ID: "a", UpdatedAt: at(0)}}
		moved := []Summary{{ID: "a", UpdatedAt: at(1)}}

		var calls atomic.Int32
		release := make(chan struct{})
		lister := listerFunc(func(ctx context.Context, _ ListFilter) ([]Summary, error) {
			if calls.Add(1) == 1 {
				return baseline, nil
			}
			<-release
			// By now the caller that started this run has gone. A listing
			// running under that caller's context would fail here.
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return moved, nil
		})

		var logged bytes.Buffer
		var emitted [][]string
		p := NewPoller(
			slog.New(slog.NewTextHandler(&logged, nil)),
			"src-1",
			lister,
			time.Minute,
			func(_ string, issueIDs []string) { emitted = append(emitted, issueIDs) },
		)

		_, err := p.Refresh(t.Context())
		assert.NilError(t, err)

		// A request that starts the run and then goes away.
		leaverCtx, cancelLeaver := context.WithCancel(t.Context())
		defer cancelLeaver()

		var eg errgroup.Group
		var leaverErr error
		eg.Go(func() error {
			_, leaverErr = p.Refresh(leaverCtx)
			return nil
		})
		// The run it started is parked inside the lister.
		synctest.Wait()

		var joined []Summary
		eg.Go(func() error {
			listed, err := p.Refresh(t.Context())
			joined = listed
			return err
		})
		eg.Go(func() error {
			p.poll(t.Context())
			return nil
		})
		// Both of them are waiting on that same run.
		synctest.Wait()

		// The leaver drops out while the listing is still in progress, so it
		// cannot be the run's result that unblocks it.
		cancelLeaver()
		synctest.Wait()

		close(release)
		assert.NilError(t, eg.Wait())

		assert.ErrorIs(t, leaverErr, context.Canceled)
		// The run outlived the caller that started it: still one listing, and
		// the others read it.
		assert.Equal(t, int(calls.Load()), 2)
		assert.DeepEqual(t, joined, moved)
		assert.Equal(t, len(emitted), 1)
		// A tick that shared the run with a cancelled request reports no
		// failure of its own.
		assert.Equal(t, logged.String(), "")
	})
}

func TestPollerListsEveryStatus(t *testing.T) {
	var filters []ListFilter
	lister := listerFunc(func(_ context.Context, f ListFilter) ([]Summary, error) {
		filters = append(filters, f)
		return nil, nil
	})

	NewPoller(nil, "src-1", lister, time.Minute, func(string, []string) {}).poll(t.Context())

	assert.Equal(t, len(filters), 1)
	assert.DeepEqual(t, filters[0].Statuses, allStatuses)
}

// listerFunc adapts a function to [IssueLister].
type listerFunc func(ctx context.Context, f ListFilter) ([]Summary, error)

func (f listerFunc) List(ctx context.Context, filter ListFilter) ([]Summary, error) {
	return f(ctx, filter)
}

func TestPollerRunStopsOnContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	p := NewPoller(nil, "src-1", &stubLister{listings: [][]Summary{nil}}, time.Millisecond,
		func(string, []string) { t.Error("cancelled poller emitted an event") })
	assert.NilError(t, p.Run(ctx))
}
