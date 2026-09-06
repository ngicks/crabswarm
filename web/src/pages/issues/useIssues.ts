// Page hooks over the api layer: the state the issues screens keep while the
// reader is on them.
import { useLocation } from "preact-iso";
import type { IssueSummary } from "@/api/gen/ngicks/crabswarm/issues/v1/issues_service_pb.js";
import { type IssueQuery, encodeIssueQuery, filterIssues, labelsOf, parseIssueQuery, useIssueListing } from "@/api/issues.js";
import { DEFAULT_QUERY } from "@/api/query.js";
import { queryOf } from "@/lib/paths.js";

export interface IssueQueryState {
  query: IssueQuery;
  /** The query string as it stands ("" or "?..."), to carry onto links. */
  search: string;
  /** Rewrites the URL's query string, keeping the path (list or detail). */
  update(patch: Partial<IssueQuery>): void;
  /** Back to the default query. */
  reset(): void;
}

/** The query lives in the URL, not in component state: a filtered list is a
 *  link, and opening an issue then coming back keeps the query. */
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
  /** The source's whole listing, every status: what the state buttons count
   *  over and where the neighbourhood graph finds a node's title and status. */
  listing: IssueSummary[];
  /** Matching issues, newest updated first. */
  rows: IssueSummary[];
  /** Distinct labels of the source, for the Labels button and suggestions. */
  labels: string[];
  isLoading: boolean;
  error: unknown;
}

/** One ListIssues per source, filtered in the browser by the query text. */
export function useIssueList(sourceId: string, query: IssueQuery): IssueListState {
  const { data, isLoading, error } = useIssueListing(sourceId);
  const listing = data?.issues ?? [];
  return {
    listing,
    rows: filterIssues(listing, query.q),
    labels: labelsOf(listing),
    isLoading,
    error,
  };
}
