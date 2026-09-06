package issues

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"gotest.tools/v3/assert"
)

// loadListing decodes the recorded whole-source listing, the same fixture the
// fake bd replays, so the filters are exercised against the shape and the
// scale of a real backlog rather than a handful of invented records.
func loadListing(t *testing.T) []Summary {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "list.json"))
	assert.NilError(t, err)
	var listed []Summary
	assert.NilError(t, json.Unmarshal(b, &listed))
	assert.Equal(t, len(listed), 81)
	return listed
}

// summaryIDs is the ids of a result, in its order.
func summaryIDs(listed []Summary) []string {
	out := make([]string, len(listed))
	for i, sum := range listed {
		out[i] = sum.ID
	}
	return out
}

func TestFilterSummaries(t *testing.T) {
	listed := loadListing(t)

	for _, tc := range []struct {
		name   string
		filter ListFilter
		// wantLen is how many of the 81 recorded issues match.
		wantLen int
		// wantHead is the ids the result must start with.
		wantHead []string
	}{
		{
			// 17 of the 81 recorded issues are closed.
			name:     "no filter keeps bd's default and hides closed",
			filter:   ListFilter{},
			wantLen:  64,
			wantHead: []string{"crabswarm-3hp.7"},
		},
		{
			name:     "every status keeps the whole listing in its order",
			filter:   ListFilter{Statuses: allStatuses},
			wantLen:  81,
			wantHead: []string{"crabswarm-lpq.9", "crabswarm-lpq.8", "crabswarm-lpq.7"},
		},
		{
			name:     "one status",
			filter:   ListFilter{Statuses: []Status{StatusInProgress}},
			wantLen:  2,
			wantHead: []string{"crabswarm-3hp.1", "crabswarm-ylc.2"},
		},
		{
			name:    "closed asked for explicitly",
			filter:  ListFilter{Statuses: []Status{StatusClosed}},
			wantLen: 17,
		},
		{
			name:    "several statuses",
			filter:  ListFilter{Statuses: []Status{StatusOpen, StatusInProgress}},
			wantLen: 63,
		},
		{
			name:    "one label",
			filter:  ListFilter{Statuses: allStatuses, Labels: []string{"tui"}},
			wantLen: 7,
		},
		{
			// 7 issues carry tui and 35 carry chat; the 6 here are the ones
			// carrying both, which is what makes the label filter all-of
			// rather than any-of.
			name:    "every label must be carried",
			filter:  ListFilter{Statuses: allStatuses, Labels: []string{"chat", "tui"}},
			wantLen: 6,
		},
		{
			// One of those 6 is closed.
			name:    "labels narrowed further by the default statuses",
			filter:  ListFilter{Labels: []string{"chat", "tui"}},
			wantLen: 5,
		},
		{
			name:    "a label nothing carries",
			filter:  ListFilter{Statuses: allStatuses, Labels: []string{"chat", "nonesuch"}},
			wantLen: 0,
		},
		{
			name:    "parent",
			filter:  ListFilter{Statuses: allStatuses, ParentID: "crabswarm-ylc"},
			wantLen: 8,
		},
		{
			// Two of those 8 children are closed.
			name:    "parent narrowed further by the default statuses",
			filter:  ListFilter{ParentID: "crabswarm-ylc"},
			wantLen: 6,
		},
		{
			name:    "an id nothing hangs under",
			filter:  ListFilter{Statuses: allStatuses, ParentID: "crabswarm-ylc.2"},
			wantLen: 0,
		},
		{
			name:     "sorted by update time, newest first",
			filter:   ListFilter{Statuses: allStatuses, SortByUpdated: true},
			wantLen:  81,
			wantHead: []string{"crabswarm-3hp.1", "crabswarm-ylc.2", "crabswarm-ylc.6"},
		},
		{
			name:     "sorted over the default statuses",
			filter:   ListFilter{SortByUpdated: true},
			wantLen:  64,
			wantHead: []string{"crabswarm-3hp.1", "crabswarm-ylc.2", "crabswarm-3hp"},
		},
		{
			name:     "limit caps the sorted result",
			filter:   ListFilter{Statuses: allStatuses, SortByUpdated: true, Limit: 3},
			wantLen:  3,
			wantHead: []string{"crabswarm-3hp.1", "crabswarm-ylc.2", "crabswarm-ylc.6"},
		},
		{
			name:    "a limit past the result caps nothing",
			filter:  ListFilter{Limit: 500},
			wantLen: 64,
		},
		{
			name: "the clauses narrow together",
			filter: ListFilter{
				Labels:        []string{"step"},
				ParentID:      "crabswarm-3hp",
				SortByUpdated: true,
			},
			wantLen:  7,
			wantHead: []string{"crabswarm-3hp.1", "crabswarm-3hp.7"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := filterSummaries(listed, tc.filter)
			assert.Equal(t, len(got), tc.wantLen)
			if len(tc.wantHead) > 0 {
				assert.DeepEqual(t, summaryIDs(got)[:len(tc.wantHead)], tc.wantHead)
			}
		})
	}
}

func TestFilterSummariesSortsIDAscendingOnATie(t *testing.T) {
	listed := loadListing(t)
	got := summaryIDs(filterSummaries(listed, ListFilter{
		Statuses:      allStatuses,
		SortByUpdated: true,
	}))

	// bd records update times to the second, and these two were recorded in
	// the same one. Ordering them by id is what keeps the board from
	// shuffling between two listings of an unchanged backlog.
	tie := slices.Index(got, "crabswarm-9rf")
	assert.Assert(t, tie >= 0, "the tied pair is not in the result")
	assert.Equal(t, got[tie+1], "crabswarm-sxj")

	all := filterSummaries(listed, ListFilter{Statuses: allStatuses, SortByUpdated: true})
	assert.Assert(t, slices.IsSortedFunc(all, func(a, b Summary) int {
		return b.UpdatedAt.Compare(a.UpdatedAt)
	}), "the result is not ordered newest-updated first")
}

func TestFilterSummariesLeavesTheListingAlone(t *testing.T) {
	listed := loadListing(t)
	before := summaryIDs(listed)

	filterSummaries(listed, ListFilter{Statuses: allStatuses, SortByUpdated: true})
	filterSummaries(listed, ListFilter{Limit: 3})

	// Every caller of one refresh reads the same slice, so a filter that
	// sorted or truncated it in place would reorder somebody else's listing.
	assert.DeepEqual(t, summaryIDs(listed), before)
}

func TestChildCountsFromTheListing(t *testing.T) {
	counts := childCounts(loadListing(t))

	// Hand-counted in testdata/list.json: crabswarm-ylc has eight children,
	// two of them closed, and crabswarm-lpq nine, all closed.
	assert.Equal(t, counts["crabswarm-ylc"], childCount{total: 8, closed: 2})
	assert.Equal(t, counts["crabswarm-lpq"], childCount{total: 9, closed: 9})
	// An issue nothing hangs under is absent rather than zero.
	_, ok := counts["crabswarm-jp7"]
	assert.Assert(t, !ok)
}

func TestChildrenOfKeepsEveryStatus(t *testing.T) {
	children := summaryIDs(childrenOf(loadListing(t), "crabswarm-ylc"))

	// The closed children are in there: an epic's progress is what the list
	// is for.
	assert.DeepEqual(t, children, []string{
		"crabswarm-ylc.8",
		"crabswarm-ylc.7",
		"crabswarm-ylc.6",
		"crabswarm-ylc.5",
		"crabswarm-ylc.4",
		"crabswarm-ylc.3",
		"crabswarm-ylc.2",
		"crabswarm-ylc.1",
	})
	assert.Equal(t, len(childrenOf(loadListing(t), "crabswarm-jp7")), 0)
}

func TestListingEdges(t *testing.T) {
	listed := loadListing(t)

	// 32 of the 81 recorded issues carry edges, 54 between them.
	assert.Equal(t, len(listingEdges(listed, nil)), 54)

	keep := map[string]struct{}{"crabswarm-lpq.9": {}, "crabswarm-lpq": {}}
	assert.DeepEqual(t, listingEdges(listed, keep), []Edge{
		{FromID: "crabswarm-lpq.9", ToID: "crabswarm-lpq", Type: "parent-child"},
	})

	// An empty set is not a nil one: it keeps nothing, which is what a caller
	// drawing no nodes asked for.
	assert.Equal(t, len(listingEdges(listed, map[string]struct{}{})), 0)
}
