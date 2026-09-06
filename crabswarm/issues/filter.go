package issues

import (
	"slices"
	"strings"
)

// The daemon narrows, tallies and flattens a source's issues here rather than
// asking bd for each shape it needs. bd admits one process per database, so a
// screen that asked bd a question per panel would serialize its own panels
// behind each other; one listing per source answers every read instead, and
// these helpers are what turn that listing into the board, an issue's
// children and a dependency graph.
//
// Every listing is shared by the callers that joined the refresh which
// produced it, so nothing here reorders or writes into the slice it is given.

// filterSummaries returns the records of listed matching f, ordered as f asks
// for.
func filterSummaries(listed []Summary, f ListFilter) []Summary {
	out := make([]Summary, 0, len(listed))
	for _, sum := range listed {
		if matchesFilter(sum, f) {
			out = append(out, sum)
		}
	}
	if f.SortByUpdated {
		slices.SortFunc(out, func(a, b Summary) int {
			if c := b.UpdatedAt.Compare(a.UpdatedAt); c != 0 {
				return c
			}
			// bd records update times to the second, so a backlog worked on
			// in one burst holds ties. The id breaks them, which is what
			// keeps two listings of an unchanged source in the same order
			// instead of shuffling the board under the reader.
			return strings.Compare(a.ID, b.ID)
		})
	}
	if f.Limit > 0 && len(out) > f.Limit {
		out = slices.Clip(out[:f.Limit])
	}
	return out
}

// matchesFilter reports whether sum passes every clause of f.
func matchesFilter(sum Summary, f ListFilter) bool {
	if len(f.Statuses) == 0 {
		// An empty status filter is bd's own default listing, which means
		// the live backlog rather than everything ever recorded.
		if sum.Status == StatusClosed {
			return false
		}
	} else if !slices.Contains(f.Statuses, sum.Status) {
		return false
	}
	for _, label := range f.Labels {
		if !slices.Contains(sum.Labels, label) {
			return false
		}
	}
	if f.ParentID != "" && sum.ParentID != f.ParentID {
		return false
	}
	return true
}

// childrenOf returns the issues listed under parent, whatever their status: a
// closed child is still part of an epic's progress.
func childrenOf(listed []Summary, parent string) []Summary {
	var out []Summary
	for _, sum := range listed {
		if sum.ParentID == parent {
			out = append(out, sum)
		}
	}
	return out
}

// childCounts tallies every issue's children by parent. bd's listing carries
// no child count, so the epic progress affordance reads it off the listing
// the request already holds.
func childCounts(listed []Summary) map[string]childCount {
	counts := make(map[string]childCount)
	for _, sum := range listed {
		if sum.ParentID == "" {
			continue
		}
		c := counts[sum.ParentID]
		c.total++
		if sum.Status == StatusClosed {
			c.closed++
		}
		counts[sum.ParentID] = c
	}
	return counts
}

// listingEdges flattens the outgoing edges every record of a listing carries.
// A non-nil keep drops the edges with an end outside it: a caller that named
// the issues it is drawing cannot place an edge pointing at a node it does
// not hold.
func listingEdges(listed []Summary, keep map[string]struct{}) []Edge {
	var out []Edge
	for _, sum := range listed {
		for _, e := range sum.Dependencies {
			if keep != nil {
				if _, ok := keep[e.FromID]; !ok {
					continue
				}
				if _, ok := keep[e.ToID]; !ok {
					continue
				}
			}
			out = append(out, e)
		}
	}
	return out
}
