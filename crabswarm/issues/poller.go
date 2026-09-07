package issues

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"golang.org/x/sync/singleflight"
)

// defaultPollInterval is how often a [Poller] lists its source when the
// caller passes no [WithPollInterval].
const defaultPollInterval = 10 * time.Second

// refreshKey is the only key a poller's singleflight group uses. One poller
// serves one source and lists all of it, so every caller — a tick, a
// request — asks for the same run.
const refreshKey = "refresh"

// IssueLister is the part of [Client] a [Poller] reads its source through.
type IssueLister interface {
	List(ctx context.Context, f ListFilter) ([]Summary, error)
}

// Poller lists one source — on its interval and on demand through
// [Poller.Refresh] — and reports the issues that changed since the previous
// listing. bd has no change feed, so the diff of two listings — ids that
// appeared, ids whose update time moved, ids that vanished — is what the
// daemon has to work with.
//
// One Poller serves one source; the [Service] runs one per registered source
// under its errgroup.
type Poller struct {
	logger   *slog.Logger
	sourceID string
	lister   IssueLister
	interval time.Duration
	emit     func(sourceID string, issueIDs []string)

	// refresh collapses the callers of [Poller.Refresh] that arrive while a
	// listing is in progress into that one listing.
	refresh singleflight.Group

	// seen is the previous listing: issue ID -> update time. primed stays
	// false until one listing succeeded, so the baseline listing reports
	// nothing. Both are touched only from inside the refresh group above,
	// which runs one listing of its key at a time and, through its own lock,
	// orders each run's writes before the next run's reads.
	seen   map[string]time.Time
	primed bool
}

// NewPoller returns a poller that lists source through lister — every
// interval, and whenever a caller runs [Poller.Refresh] — and hands the
// changed issue ids to emit. A nil logger discards logs; a non-positive
// interval falls back to [defaultPollInterval].
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
// first listing, whether a tick or a [Poller.Refresh] caller runs it, only
// records the baseline. It always returns nil: the poller shares an errgroup
// with the HTTP server, so a bd invocation that fails — a database being
// rewritten, bd not on PATH yet — must degrade the change feed rather than
// take the daemon down.
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

// poll runs one refresh and logs its failure. It is the whole body of a
// tick, split out so a test can step the poller without waiting on a clock.
func (p *Poller) poll(ctx context.Context) {
	if _, err := p.Refresh(ctx); err != nil && ctx.Err() == nil {
		p.logger.Warn("issues: poll failed", "source", p.sourceID, "err", err)
	}
}

// Refresh lists the source now, moves the baseline to that listing and
// returns it. It is the source's one listing: the ticker runs it on
// schedule, a request runs it on demand, and callers that arrive while a
// listing is in progress wait for that one instead of starting another. A
// caller whose ctx ends returns right away and leaves the run to the others.
//
// Every caller of a run receives the same slice, so callers must treat the
// result as read-only.
func (p *Poller) Refresh(ctx context.Context) ([]Summary, error) {
	// A caller whose ctx has already ended, such as the ticker during
	// shutdown, starts nothing: the run below is detached, so without this
	// check it would spawn a bd process nobody is waiting for.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ch := p.refresh.DoChan(refreshKey, func() (any, error) {
		// Detached from the caller that happened to start the run: the run
		// serves every caller, so a request giving up must not fail the
		// ticker and everyone else waiting on it. The listing keeps the
		// caller's values, and the client bounds the bd invocation with a
		// timeout of its own.
		return p.list(context.WithoutCancel(ctx))
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		listed, _ := res.Val.([]Summary)
		return listed, nil
	}
}

// list runs one listing and emits the ids that differ from the previous one.
// One flight runs it, on the goroutine the group spawns for that flight, and
// every caller of the flight reads its result.
func (p *Poller) list(ctx context.Context) ([]Summary, error) {
	// Every status is listed: bd's default listing hides closed issues, so a
	// closed issue would leave the listing silently instead of being
	// reported as a change.
	listed, err := p.lister.List(ctx, ListFilter{Statuses: allStatuses})
	if err != nil {
		return nil, err
	}

	current := make(map[string]time.Time, len(listed))
	for _, s := range listed {
		current[s.ID] = s.UpdatedAt
	}
	changed := changedIDs(p.seen, current)
	primed := p.primed
	p.seen, p.primed = current, true

	// A listing a request asked for emits too: it moves the baseline, so a
	// change only it saw would go unreported by every later tick and the
	// event feed would disagree with the listing everyone reads.
	if primed && len(changed) > 0 {
		p.emit(p.sourceID, changed)
	}
	return listed, nil
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
