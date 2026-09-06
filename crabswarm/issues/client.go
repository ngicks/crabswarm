package issues

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Client reads one beads database by running bd in a fixed directory. bd
// resolves the database from that directory the same way it does for a
// person typing the command there, so the directory is the only handle the
// caller needs.
type Client struct {
	bin    string
	dir    string
	env    []string
	logger *slog.Logger
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
	c := &Client{bin: defaultBinary, dir: dir}
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
func (c *Client) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, c.bin, args...)
	cmd.Dir = c.dir
	cmd.Env = append(os.Environ(), c.env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	c.log().Debug("running bd", "args", args, "dir", c.dir)
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return stdout.Bytes(), fmt.Errorf("bd %s: %w: %s", strings.Join(args, " "), err, msg)
		}
		return stdout.Bytes(), fmt.Errorf("bd %s: %w", strings.Join(args, " "), err)
	}
	return stdout.Bytes(), nil
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

// Children returns the issues whose parent is id.
func (c *Client) Children(ctx context.Context, id string) ([]Summary, error) {
	return c.List(ctx, ListFilter{ParentID: id})
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
