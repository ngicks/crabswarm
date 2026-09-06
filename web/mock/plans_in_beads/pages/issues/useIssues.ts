// Page hooks over the api layer: the state the issues screens keep while the
// reader is on them. In the real feature these wrap the query options of
// api/issues.ts; here they wrap the fixture accessors, which read the `issues`
// signal, so a simulated push re-renders every caller.
import { useEffect } from "preact/hooks";
import { useLocation } from "preact-iso";
import { type Issue, getIssue } from "@/api/client.js";
import { type IssueQuery, encodeIssueQuery, filterIssues, labelsOf, parseIssueQuery } from "@/api/issues.js";
import { DEFAULT_QUERY } from "@/api/query.js";
import { queryOf } from "@/lib/paths.js";
import { openIssueId } from "@/signals/issues.js";

export interface IssueQueryState {
  query: IssueQuery;
  /** The query string as it stands ("" or "?..."), to carry onto links. */
  search: string;
  /** Rewrites the URL's query string, keeping the path (list or detail). */
  update(patch: Partial<IssueQuery>): void;
  /** Back to the default query. */
  reset(): void;
}

/** The query lives in the URL (PLAN.md "SPA routes"), not in component
 *  state: a filtered list is a link, and opening an issue then coming back
 *  keeps the query. */
export function useIssueQuery(): IssueQueryState {
  const loc = useLocation();
  const query = parseIssueQuery(queryOf(loc.url));
  const go = (next: IssueQuery) => loc.route(loc.path + encodeIssueQuery(next), true);
  return {
    query,
    search: encodeIssueQuery(query),
    update: (patch) => go({ ...query, ...patch }),
    reset: () => go({ q: DEFAULT_QUERY }),
  };
}

export interface IssueListState {
  /** Matching issues, newest updated first. */
  rows: Issue[];
  /** Distinct labels of the source, for the Labels button and suggestions. */
  labels: string[];
}

export function useIssueList(sourceId: string, query: IssueQuery): IssueListState {
  return {
    rows: filterIssues(sourceId, query.q),
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
