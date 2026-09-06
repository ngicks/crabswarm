package issues

import (
	"context"
	"testing"
	"time"

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
