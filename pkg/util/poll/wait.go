package poll

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"
)

// Budget applied to the zero value of WaitOption.
const (
	defaultInterval     = time.Second
	defaultRetries      = 10
	defaultProbeTimeout = 2 * time.Second
)

// WaitOption tunes Wait. Interval and Retries carry the meaning the Docker
// healthcheck flags of the same names have.
type WaitOption struct {
	// StartPeriod delays the first probe: Wait sleeps for it, interruptible by
	// ctx, and probes nothing until it elapses.
	StartPeriod time.Duration
	// Interval is the cadence of probes, measured from the first one; a probe
	// that outlasts it is followed at once. Zero selects one second.
	Interval time.Duration
	// Retries is the count of consecutive failed probes at which Wait gives up.
	// Zero selects ten; a negative value retries until ctx ends.
	Retries int
	// Client issues the probes for KindHTTP targets. Nil selects a client bounded
	// by ProbeTimeout.
	Client *http.Client
	// ProbeTimeout bounds a single probe attempt, so one stalled dial cannot eat
	// the whole budget. Zero selects two seconds.
	ProbeTimeout time.Duration
}

// Wait probes target until it is ready, ctx ends, or the retry budget runs
// out. The first probe fires once StartPeriod has elapsed and the rest follow
// every Interval. logger may be nil.
func Wait(ctx context.Context, logger *slog.Logger, target Target, opts WaitOption) error {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	switch target.Kind {
	case KindUnix, KindFile, KindHTTP:
	default:
		// A bad Kind is a caller bug, so report it now instead of spending the
		// whole retry budget failing the same way.
		return fmt.Errorf("unsupported target kind %d", int(target.Kind))
	}

	w := newWaiter(target, opts)

	if opts.StartPeriod > 0 {
		timer := time.NewTimer(opts.StartPeriod)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			// No probe has run yet, so there is no last-probe error to report
			// alongside the cancellation.
			return fmt.Errorf("waiting for %s: %w", target.describe(), ctx.Err())
		case <-timer.C:
		}
	}

	// The ticker starts after the start period, because a ticker running during
	// it would buffer a tick and make the second probe follow the first at once.
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	failures := 0
	for {
		err := w.probe(ctx)
		if err == nil {
			logger.Debug("poll target ready", "target", target.describe())
			return nil
		}
		// A probe cut short by ctx fails with the context error; report that as
		// cancellation rather than as one more spent retry.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return waitErr(target, ctxErr, err)
		}
		failures++
		logger.Debug("poll probe failed",
			"target", target.describe(),
			"error", err,
			"failures", failures)

		if w.retries >= 0 && failures >= w.retries {
			return fmt.Errorf("%s not ready after %d failed probes: %w",
				target.describe(), failures, err)
		}

		select {
		case <-ctx.Done():
			return waitErr(target, ctx.Err(), err)
		case <-ticker.C:
		}
	}
}

// waitErr reports why Wait stopped early. Both the reason and the last probe
// error are wrapped, so errors.Is matches either one.
func waitErr(target Target, reason, lastProbe error) error {
	return fmt.Errorf("waiting for %s: %w: last probe: %w",
		target.describe(), reason, lastProbe)
}

// waiter holds the options Wait resolved, so the probe helpers do not thread a
// growing parameter list.
type waiter struct {
	target       Target
	client       *http.Client
	interval     time.Duration
	probeTimeout time.Duration
	retries      int
}

func newWaiter(target Target, opts WaitOption) waiter {
	w := waiter{
		target:       target,
		client:       opts.Client,
		interval:     opts.Interval,
		probeTimeout: opts.ProbeTimeout,
		retries:      opts.Retries,
	}
	if w.interval <= 0 {
		w.interval = defaultInterval
	}
	if w.probeTimeout <= 0 {
		w.probeTimeout = defaultProbeTimeout
	}
	if w.retries == 0 {
		w.retries = defaultRetries
	}
	if w.client == nil {
		w.client = &http.Client{Timeout: w.probeTimeout}
	}
	return w
}

// probe runs one attempt against the target, bounded by the probe timeout.
func (w waiter) probe(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, w.probeTimeout)
	defer cancel()

	switch w.target.Kind {
	case KindUnix:
		return dialUnix(ctx, w.target.Addr)
	case KindFile:
		return statFile(w.target.Addr)
	case KindHTTP:
		return getHTTP(ctx, w.client, w.target.Addr)
	}
	return fmt.Errorf("unsupported target kind %d", int(w.target.Kind))
}

// dialUnix reports whether the socket accepts a connection. It dials through a
// net.Dialer rather than net.Dial, because only DialContext lets the per-probe
// timeout bound the dial.
func dialUnix(ctx context.Context, path string) error {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "unix", path)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

// statFile reports whether the path exists.
func statFile(path string) error {
	_, err := os.Stat(path)
	return err
}

// getHTTP reports whether the URL answers a GET with 2xx.
func getHTTP(ctx context.Context, client *http.Client, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Drain the body so the connection goes back to the pool for the next probe.
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	return nil
}
