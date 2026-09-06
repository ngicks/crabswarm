package issues

import (
	"context"
	"log/slog"
	"slices"
	"time"
)

// defaultPollInterval is how often a [Poller] lists its source when the
// caller passes no [WithPollInterval].
const defaultPollInterval = 10 * time.Second

// IssueLister is the part of [Client] a [Poller] reads its source through.
type IssueLister interface {
	List(ctx context.Context, f ListFilter) ([]Summary, error)
}

// Poller lists one source on an interval and reports the issues that changed
// since the previous listing. bd has no change feed, so the diff of two
// listings — ids that appeared, ids whose update time moved, ids that
// vanished — is what the daemon has to work with.
//
// One Poller serves one source; the [Service] runs one per registered source
// under its errgroup.
type Poller struct {
	logger   *slog.Logger
	sourceID string
	lister   IssueLister
	interval time.Duration
	emit     func(sourceID string, issueIDs []string)

	// seen is the previous listing: issue ID -> update time. primed stays
	// false until one listing succeeded, so the baseline poll reports
	// nothing. Both are touched only from the goroutine running poll.
	seen   map[string]time.Time
	primed bool
}

// NewPoller returns a poller that lists source every interval through lister
// and hands the changed issue ids to emit. A nil logger discards logs; a
// non-positive interval falls back to [defaultPollInterval].
func NewPoller(
	logger *slog.Logger,
	sourceID string,
	lister IssueLister,
	interval time.Duration,
	emit func(sourceID string, issueIDs []string),
) *Poller {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	if interval <= 0 {
		interval = defaultPollInterval
	}
	return &Poller{
		logger:   logger,
		sourceID: sourceID,
		lister:   lister,
		interval: interval,
		emit:     emit,
	}
}

// Run polls until ctx is cancelled, blocking for its whole lifetime. The
// first poll only records the baseline. It always returns nil: the poller
// shares an errgroup with the HTTP server, so a bd invocation that fails —
// a database being rewritten, bd not on PATH yet — must degrade the change
// feed rather than take the daemon down.
func (p *Poller) Run(ctx context.Context) error {
	p.poll(ctx)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

// poll runs one listing and emits the ids that differ from the previous one.
// It is the whole body of a tick, split out so a test can step the poller
// without waiting on a clock.
func (p *Poller) poll(ctx context.Context) {
	// Every status is listed: bd's default listing hides closed issues, so a
	// closed issue would leave the listing silently instead of being
	// reported as a change.
	listed, err := p.lister.List(ctx, ListFilter{Statuses: allStatuses})
	if err != nil {
		if ctx.Err() == nil {
			p.logger.Warn("issues: poll failed", "source", p.sourceID, "err", err)
		}
		return
	}

	current := make(map[string]time.Time, len(listed))
	for _, s := range listed {
		current[s.ID] = s.UpdatedAt
	}
	changed := changedIDs(p.seen, current)
	primed := p.primed
	p.seen, p.primed = current, true

	if !primed || len(changed) == 0 {
		return
	}
	p.emit(p.sourceID, changed)
}

// changedIDs returns the issue ids that appeared, were updated, or vanished
// between two listings, sorted so the emitted event is deterministic.
func changedIDs(prev, current map[string]time.Time) []string {
	var changed []string
	for id, updated := range current {
		was, ok := prev[id]
		if !ok || !was.Equal(updated) {
			changed = append(changed, id)
		}
	}
	for id := range prev {
		if _, ok := current[id]; !ok {
			changed = append(changed, id)
		}
	}
	slices.Sort(changed)
	return changed
}
