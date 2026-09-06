package mermaidlint

import (
	"context"
	"fmt"

	"github.com/ngicks/crabswarm/crabswarm/issues"
)

// Source is what [Sweep] reads a backlog through: the listing it walks and
// the full issue it falls back to for comments. [*issues.Client] satisfies
// it.
type Source interface {
	List(ctx context.Context, f issues.ListFilter) ([]issues.Summary, error)
	Get(ctx context.Context, id string) (*issues.Issue, error)
}

// SweepOptions chooses which issues [Sweep] reads. The zero value takes
// bd's own default listing, which is the open backlog.
type SweepOptions struct {
	// All lints the closed issues alongside the open ones.
	All bool
	// Limit lints only that many issues, the most recently updated first
	// and in any status. Zero lints the whole listing.
	Limit int
}

// allStatuses is every status bd stores. An empty status filter leaves bd's
// own default, which hides the closed issues, so asking for all of them
// means naming them.
var allStatuses = []issues.Status{
	issues.StatusOpen,
	issues.StatusInProgress,
	issues.StatusBlocked,
	issues.StatusDeferred,
	issues.StatusClosed,
}

// filter renders the options as the listing to ask bd for.
func (o SweepOptions) filter() issues.ListFilter {
	f := issues.ListFilter{Limit: o.Limit}
	if o.Limit > 0 {
		// "The most recently touched issues" only means something read
		// across the whole backlog, so a capped sweep drops the status
		// filter along with the default order.
		f.SortByUpdated = true
	}
	if o.All || o.Limit > 0 {
		f.Statuses = allStatuses
	}
	return f
}

// Sweep lints the mermaid diagrams written in the issues src lists.
//
// It reads the listing, which carries the text fields but not the
// comments, so only an issue holding comments is fetched in full and only
// an issue holding a mermaid fence anywhere is linted at all. The
// remaining issues go to [Lint] in one call, and opts reach it unchanged —
// [WithDir] is how a repository's mermaid-lint configuration is brought to
// bear on its issue text.
func Sweep(
	ctx context.Context,
	src Source,
	o SweepOptions,
	opts ...Option,
) ([]Finding, error) {
	summaries, err := src.List(ctx, o.filter())
	if err != nil {
		return nil, fmt.Errorf("listing issues to lint: %w", err)
	}

	var list []issues.Issue
	for _, s := range summaries {
		iss := issues.Issue{Summary: s}
		if s.CommentCount > 0 {
			full, err := src.Get(ctx, s.ID)
			if err != nil {
				return nil, fmt.Errorf("reading issue %s: %w", s.ID, err)
			}
			iss = *full
		}
		if !issueHasFence(iss) {
			continue
		}
		list = append(list, iss)
	}
	if len(list) == 0 {
		return nil, nil
	}
	return Lint(ctx, list, opts...)
}

// issueHasFence reports whether any of the issue's text opens a mermaid
// fence, so that a sweep can drop the issues [Lint] would write no file
// for.
func issueHasFence(iss issues.Issue) bool {
	if HasFence(iss.Description) ||
		HasFence(iss.Design) ||
		HasFence(iss.AcceptanceCriteria) ||
		HasFence(iss.Notes) {
		return true
	}
	for _, c := range iss.Comments {
		if HasFence(c.Text) {
			return true
		}
	}
	return false
}
