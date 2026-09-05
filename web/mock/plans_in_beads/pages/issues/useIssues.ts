// Page hooks over the api layer: the state the issues screens keep while the
// reader is on them. In the real feature these wrap the query options of
// api/issues.ts; here they wrap the fixture accessors, which read the `issues`
// signal, so a simulated push re-renders every caller.
import { type Dispatch, type StateUpdater, useEffect, useState } from "preact/hooks";
import { type Issue, getIssue } from "../../api/client.js";
import { type IssueFilter, emptyFilter, filterIssues, labelsOf } from "../../api/issues.js";
import { openIssueId } from "../../signals/issues.js";

export interface IssueListState {
  filter: IssueFilter;
  setFilter: Dispatch<StateUpdater<IssueFilter>>;
  /** Matching issues, newest updated first. */
  rows: Issue[];
  /** Distinct labels of the source, for the label multi-select. */
  labels: string[];
}

/** Filter state of one source's list, plus what it selects. The caller keys
 *  this per source: the labels are a source's own, so a label selection must
 *  not carry across a switch. */
export function useIssueList(sourceId: string): IssueListState {
  const [filter, setFilter] = useState<IssueFilter>(emptyFilter);
  return {
    filter,
    setFilter,
    rows: filterIssues(sourceId, filter),
    labels: labelsOf(sourceId),
  };
}

/** The open issue, and the record of which one it is: the simulated push
 *  (api/events.ts) targets what the reader is looking at. */
export function useOpenIssue(sourceId: string, issueId: string): Issue | undefined {
  useEffect(() => {
    openIssueId.value = issueId;
    return () => {
      openIssueId.value = "";
    };
  }, [issueId]);

  return getIssue(sourceId, issueId);
}
