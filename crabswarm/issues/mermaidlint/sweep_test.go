package mermaidlint

import (
	"context"
	"errors"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/crabswarm/issues"
)

// fakeSource is a backlog held in memory. It answers List off the issues'
// summaries, the way bd does — comment text is only reachable through Get —
// and records what it was asked for.
type fakeSource struct {
	backlog []issues.Issue
	listErr error
	getErr  error

	filters []issues.ListFilter
	fetched []string
}

func (f *fakeSource) List(_ context.Context, filter issues.ListFilter) ([]issues.Summary, error) {
	f.filters = append(f.filters, filter)
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]issues.Summary, len(f.backlog))
	for i, iss := range f.backlog {
		s := iss.Summary
		s.CommentCount = len(iss.Comments)
		out[i] = s
	}
	return out, nil
}

func (f *fakeSource) Get(_ context.Context, id string) (*issues.Issue, error) {
	f.fetched = append(f.fetched, id)
	if f.getErr != nil {
		return nil, f.getErr
	}
	for _, iss := range f.backlog {
		if iss.ID == id {
			return &iss, nil
		}
	}
	return nil, errors.New("no such issue")
}

// *issues.Client is what the command sweeps with, so the interface has to
// keep fitting it.
var _ Source = (*issues.Client)(nil)

func TestSweep(t *testing.T) {
	installFakeMermaidLint(t, "report.json")

	src := &fakeSource{backlog: linted()}
	got, err := Sweep(t.Context(), src, SweepOptions{})
	assert.NilError(t, err)

	// The default sweep is bd's own listing: the open backlog, uncapped and
	// in bd's order.
	assert.DeepEqual(t, src.filters, []issues.ListFilter{{}})
	// Every issue in the fixture carries comments, which the listing does
	// not report, so each one is read in full.
	assert.DeepEqual(t, src.fetched, []string{"plan-aaa", "plan-bbb", "plan-ccc"})
	assert.DeepEqual(t, got, lintedFindings)
}

func TestSweepFilters(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts SweepOptions
		want issues.ListFilter
	}{
		{
			name: "default",
			want: issues.ListFilter{},
		},
		{
			name: "all",
			opts: SweepOptions{All: true},
			want: issues.ListFilter{Statuses: allStatuses},
		},
		{
			// A cap means "what was touched most recently", which is read
			// across the whole backlog rather than the open part of it.
			name: "limit",
			opts: SweepOptions{Limit: 20},
			want: issues.ListFilter{Statuses: allStatuses, Limit: 20, SortByUpdated: true},
		},
		{
			name: "all and limit",
			opts: SweepOptions{All: true, Limit: 5},
			want: issues.ListFilter{Statuses: allStatuses, Limit: 5, SortByUpdated: true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A backlog without diagrams keeps mermaid-lint out of the
			// picture: the listing is the whole of what is under test.
			src := &fakeSource{backlog: []issues.Issue{{
				Summary: issues.Summary{ID: "plan-ccc", Description: "prose only"},
			}}}
			got, err := Sweep(t.Context(), src, tc.opts)
			assert.NilError(t, err)
			assert.Equal(t, len(got), 0)
			assert.DeepEqual(t, src.filters, []issues.ListFilter{tc.want})
		})
	}
}

func TestSweepSkipsIssuesWithoutFences(t *testing.T) {
	invocations := installFakeMermaidLint(t, "report.json")

	src := &fakeSource{backlog: []issues.Issue{
		// No comments: the listing already says there is no diagram, so
		// this one is never fetched.
		{Summary: issues.Summary{ID: "plan-ddd", Description: "prose only"}},
		// Comments: what they hold is only visible through Get, and they
		// hold no diagram either.
		{
			Summary:  issues.Summary{ID: "plan-eee", Notes: "prose only"},
			Comments: []issues.Comment{{Text: "prose only"}},
		},
	}}
	got, err := Sweep(t.Context(), src, SweepOptions{})
	assert.NilError(t, err)
	assert.Equal(t, len(got), 0)

	assert.DeepEqual(t, src.fetched, []string{"plan-eee"})
	// Nothing was left to lint, so mermaid-lint never ran.
	assert.Equal(t, len(invocations()), 0)
}

func TestSweepSourceErrors(t *testing.T) {
	boom := errors.New("bd exploded")

	t.Run("list", func(t *testing.T) {
		_, err := Sweep(t.Context(), &fakeSource{listErr: boom}, SweepOptions{})
		assert.ErrorIs(t, err, boom)
	})
	t.Run("get", func(t *testing.T) {
		src := &fakeSource{
			backlog: []issues.Issue{{
				Summary:  issues.Summary{ID: "plan-aaa"},
				Comments: []issues.Comment{{Text: "prose only"}},
			}},
			getErr: boom,
		}
		_, err := Sweep(t.Context(), src, SweepOptions{})
		assert.ErrorIs(t, err, boom)
		assert.ErrorContains(t, err, "plan-aaa")
	})
}
