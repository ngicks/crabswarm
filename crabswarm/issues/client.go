package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"golang.org/x/sync/singleflight"
)

// bdRunTimeout bounds a run from the moment it is created, which is before
// it queues for the client's one slot: it covers the wait for that slot and
// the bd invocation together. The run is detached from the caller that
// started it — later callers join it, and a caller that gives up leaves it
// running for the others — so no caller's deadline can end it, and it holds
// the slot until it returns. A bd that outlives everyone who wanted its
// output must therefore end on its own.
const bdRunTimeout = 2 * time.Minute

// errAbandoned reports a run whose callers all left before it produced
// anything. The callers that left never see it; they are already returning
// their own ctx.Err(). It reaches only a caller that joined the run in the
// moment between the last caller leaving and the group forgetting the key,
// and that caller runs again.
var errAbandoned = errors.New("bd run abandoned by every caller")

// Client reads one beads database by running bd in a fixed directory. bd
// resolves the database from that directory the same way it does for a
// person typing the command there, so the directory is the only handle the
// caller needs.
//
// A Client runs one bd at a time and shares a run between callers asking
// for the same thing, so it is meant to be kept and used by many
// goroutines rather than built per call.
type Client struct {
	bin    string
	dir    string
	env    []string
	logger *slog.Logger

	// queue holds the single slot every bd run of this client waits for.
	// bd embeds dolt, which admits one process per database at a time: a
	// second bd fails to take the database lock and its driver retries the
	// whole open under an exponential backoff — 500ms, growing by 1.5, up
	// to 5s — so two overlapping runs cost far more than the same two in
	// sequence and are served in whatever order the backoff happens to
	// land. Clients of other databases hold their own slot and are
	// unaffected.
	queue *semaphore.Weighted
	// group collapses callers that ask for the same bd invocation while it
	// is in progress into that one run.
	group singleflight.Group

	mu      sync.Mutex
	flights map[string]*flight // argument list -> the run currently serving it
}

// flight is one bd run together with the callers waiting for it.
type flight struct {
	// ctx bounds the run itself. It lives here, rather than being handed
	// down from a caller, because the run belongs to no single caller: it
	// must survive the first caller giving up and must end when the last
	// one does.
	ctx    context.Context
	cancel context.CancelFunc
	// joined counts the callers still waiting. The run answers to nothing
	// else, so this count is all that says whether anyone still wants it.
	joined int
}

// Option configures a [Client] built by [NewClient].
type Option func(*Client)

// WithBinary overrides the bd executable. The default, "bd", is looked up
// on PATH.
func WithBinary(path string) Option {
	return func(c *Client) { c.bin = path }
}

// WithEnv adds "KEY=VALUE" entries to the environment of every bd
// subprocess, on top of the environment of the running process.
func WithEnv(env ...string) Option {
	return func(c *Client) { c.env = append(c.env, env...) }
}

// WithLogger directs the debug record written for each bd invocation.
func WithLogger(l *slog.Logger) Option {
	return func(c *Client) { c.logger = l }
}

// NewClient returns a Client running bd in dir.
func NewClient(dir string, opts ...Option) *Client {
	c := &Client{
		bin:     defaultBinary,
		dir:     dir,
		queue:   semaphore.NewWeighted(1),
		flights: make(map[string]*flight),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *Client) log() *slog.Logger {
	if c.logger != nil {
		return c.logger
	}
	return slog.Default()
}

// run executes "bd args..." in the client's directory, capturing stdout.
// stdout is returned even when the command fails, because bd reports its
// errors as JSON there. On failure the returned error carries the trimmed
// stderr.
//
// The client runs one bd at a time, and callers asking for the same
// arguments while a run is in progress all read that one run's output. A
// caller whose ctx ends returns right away and leaves the run to the
// others.
func (c *Client) run(ctx context.Context, args ...string) ([]byte, error) {
	// NUL joins the arguments so that ["a b"] and ["a", "b"] cannot key the
	// same run.
	key := strings.Join(args, "\x00")
	for {
		out, err := c.runShared(ctx, key, args)
		if !errors.Is(err, errAbandoned) {
			return out, err
		}
		// The run this caller joined had lost its own callers, in the moment
		// before the group forgot its key; nothing about that says this
		// caller cannot be served. Asking again cannot land on the same run,
		// because the group forgets a key before it hands the result out.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
}

// runShared waits for the shared run of args, or for ctx, whichever comes
// first.
func (c *Client) runShared(ctx context.Context, key string, args []string) ([]byte, error) {
	fl, ch := c.join(ctx, key, args)
	defer c.leave(key, fl)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		// Every caller of the run reads the same stdout slice. They only
		// decode it, so nothing has to copy it.
		out, _ := res.Val.([]byte)
		return out, res.Err
	}
}

// join adds the caller to the run serving key, starting one when none is in
// progress, and returns that run and the channel its result arrives on.
//
// The lookup and the group call happen under one lock, which is what keeps
// the two views of a run in step: the map holds a flight only while its run
// has not reached the deferred finish below, and by then the group still
// knows the key, so a caller that finds a flight joins that same run.
func (c *Client) join(
	ctx context.Context,
	key string,
	args []string,
) (*flight, <-chan singleflight.Result) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fl := c.flights[key]
	if fl == nil {
		// Detached from this caller: being first says nothing about how
		// long the run is wanted, since callers join and leave it.
		runCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), bdRunTimeout)
		fl = &flight{ctx: runCtx, cancel: cancel}
		c.flights[key] = fl
	}
	fl.joined++

	return fl, c.group.DoChan(key, func() (any, error) {
		defer c.finish(key, fl)
		return c.runQueued(fl.ctx, args)
	})
}

// leave drops one caller from the run. When it was the last one the run is
// cancelled: nobody waits for its output any more, so a run still queued
// for the slot never spawns a process and one already running stops holding
// the slot. The map entry goes with it, so the next caller starts a fresh
// run instead of waiting on a cancelled one.
func (c *Client) leave(key string, fl *flight) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fl.joined--
	if fl.joined > 0 {
		return
	}
	if c.flights[key] == fl {
		delete(c.flights, key)
	}
	fl.cancel()
}

// finish retires the run, so that the next caller of the same arguments
// reads bd again instead of this run's output.
func (c *Client) finish(key string, fl *flight) {
	c.mu.Lock()
	if c.flights[key] == fl {
		delete(c.flights, key)
	}
	c.mu.Unlock()
	fl.cancel()
}

// runQueued waits for the client's queue slot and runs bd under ctx, the
// run's own context.
func (c *Client) runQueued(ctx context.Context, args []string) ([]byte, error) {
	if err := c.queue.Acquire(ctx, 1); err != nil {
		if abandoned(ctx) {
			return nil, errAbandoned
		}
		return nil, fmt.Errorf("bd %s: %w", strings.Join(args, " "), err)
	}
	defer c.queue.Release(1)

	cmd := exec.CommandContext(ctx, c.bin, args...)
	cmd.Dir = c.dir
	cmd.Env = append(os.Environ(), c.env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	c.log().Debug("running bd", "args", args, "dir", c.dir)
	if err := cmd.Run(); err != nil {
		if abandoned(ctx) {
			return nil, errAbandoned
		}
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return stdout.Bytes(), fmt.Errorf("bd %s: %w: %s", strings.Join(args, " "), err, msg)
		}
		return stdout.Bytes(), fmt.Errorf("bd %s: %w", strings.Join(args, " "), err)
	}
	return stdout.Bytes(), nil
}

// abandoned reports whether a run's context ended because its last caller
// left. The context is detached from every caller, so nothing else cancels
// it; running out of its own time expires it instead.
func abandoned(ctx context.Context) bool {
	return errors.Is(ctx.Err(), context.Canceled)
}

// ListFilter narrows [Client.List]. The zero value leaves bd's own default
// listing, described on Statuses below.
type ListFilter struct {
	// Statuses keeps only issues in one of these statuses. Empty leaves
	// bd's own default, which reports open, in_progress, blocked and
	// deferred issues and hides closed ones.
	Statuses []Status
	// Labels keeps only issues carrying all of these labels.
	Labels []string
	// ParentID keeps only children of that issue.
	ParentID string
	// Limit caps the result. Zero means no cap.
	Limit int
	// SortByUpdated orders by last update, most recent first, instead of
	// bd's default order.
	SortByUpdated bool
}

// args renders the filter as bd list flags.
func (f ListFilter) args() []string {
	var args []string
	if len(f.Statuses) > 0 {
		ss := make([]string, len(f.Statuses))
		for i, s := range f.Statuses {
			ss[i] = string(s)
		}
		// bd's --status is a single string flag: repeating it overwrites
		// the previous value, so several statuses go in comma-separated.
		args = append(args, "--status", strings.Join(ss, ","))
	}
	for _, l := range f.Labels {
		args = append(args, "--label", l)
	}
	if f.ParentID != "" {
		args = append(args, "--parent", f.ParentID)
	}
	// The limit is always passed. bd's own default caps the result at 50,
	// which would silently truncate a source once its backlog grows past
	// that; "--limit 0" is how bd spells unlimited.
	args = append(args, "--limit", strconv.Itoa(max(f.Limit, 0)))
	if f.SortByUpdated {
		args = append(args, "--sort", "updated")
	}
	return args
}

// List returns the issues matching f.
func (c *Client) List(ctx context.Context, f ListFilter) ([]Summary, error) {
	out, err := c.run(ctx, append([]string{"list", "--json"}, f.args()...)...)
	if err != nil {
		return nil, err
	}
	var summaries []Summary
	if err := json.Unmarshal(out, &summaries); err != nil {
		return nil, fmt.Errorf("decoding bd list: %w", err)
	}
	return summaries, nil
}

// errorReport is the JSON a failing bd subcommand prints on stdout.
type errorReport struct {
	Error string `json:"error"`
}

// Get returns the issue with the given ID, including its comments and the
// issues it depends on.
func (c *Client) Get(ctx context.Context, id string) (*Issue, error) {
	// --id= rather than a positional argument: an ID starting with "-"
	// would otherwise be parsed as one of bd's own global flags.
	out, runErr := c.run(ctx, "show", "--id="+id, "--json", "--include-comments")
	if runErr != nil {
		// A missing ID is reported as JSON on stdout, which decodes into
		// neither the issue array nor a useful message on its own.
		var report errorReport
		if err := json.Unmarshal(out, &report); err == nil && report.Error != "" {
			return nil, fmt.Errorf("bd show %s: %s: %w", id, report.Error, runErr)
		}
		return nil, runErr
	}
	// bd show reports every requested ID, so a single ID still comes back
	// as a one-element array.
	var issues []Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("decoding bd show %s: %w", id, err)
	}
	if len(issues) != 1 {
		return nil, fmt.Errorf("bd show %s: got %d issues, want 1", id, len(issues))
	}
	return &issues[0], nil
}
