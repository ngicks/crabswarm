// The issues surface's data layer: the query options over IssuesService, the
// URL spelling of the search query, and the decodes the views need.
//
// One ListIssues per source asks for every status and the browser evaluates
// the search query over the result (api/query.ts). The request's own statuses
// / labels / parent_id filters stay unused on purpose: the state buttons show
// how many issues each state *would* match, which is three evaluations of one
// listing rather than three round trips.
import { useQuery } from "@tanstack/preact-query";
import { timeValue } from "@/lib/format.js";
import { issuesClient } from "./client.js";
import {
  type IssueComment,
  type IssueEdge,
  IssueStatus,
  type IssueSummary,
  type Source,
} from "./gen/ngicks/crabswarm/issues/v1/issues_service_pb.js";
import { DEFAULT_QUERY, matches, parseQuery } from "./query.js";

/** Every status bd stores. Unspecified is left out: the daemon drops it and
 *  a filter of nothing but unspecified falls back to bd's own default, which
 *  hides closed issues. */
export const ALL_STATUSES: IssueStatus[] = [
  IssueStatus.OPEN,
  IssueStatus.IN_PROGRESS,
  IssueStatus.BLOCKED,
  IssueStatus.DEFERRED,
  IssueStatus.CLOSED,
];

// Query keys are the single source of truth shared between the hooks below and
// the WatchIssues invalidation in events.ts. Everything scoped to one source
// hangs under `["issues", "source", sourceId]`, so an IssuesChanged event that
// names no ids invalidates the whole subtree with one call.
export const qk = {
  sources: () => ["issues", "sources"] as const,
  source: (sourceId: string) => ["issues", "source", sourceId] as const,
  listing: (sourceId: string) => ["issues", "source", sourceId, "listing"] as const,
  issue: (sourceId: string, issueId: string) => ["issues", "source", sourceId, "issue", issueId] as const,
  dependencies: (sourceId: string) => ["issues", "source", sourceId, "deps"] as const,
};

/** The registered beads databases. */
export function useSources() {
  return useQuery({
    queryKey: qk.sources(),
    queryFn: () => issuesClient.listSources({}),
  });
}

/** One source's whole listing, every status, newest updated first. The list,
 *  the state buttons, the labels page and the neighbourhood graph all read
 *  their issues out of this one query. */
export function useIssueListing(sourceId: string) {
  return useQuery({
    queryKey: qk.listing(sourceId),
    queryFn: () => issuesClient.listIssues({ sourceId, statuses: ALL_STATUSES }),
    enabled: sourceId !== "",
  });
}

/** One issue with its rendered fields, children, dependencies and comments. */
export function useIssue(sourceId: string, issueId: string) {
  return useQuery({
    queryKey: qk.issue(sourceId, issueId),
    queryFn: () => issuesClient.getIssue({ sourceId, issueId }),
    enabled: sourceId !== "" && issueId !== "",
  });
}

/** Every dependency edge of a source, in one call.
 *
 *  The whole source rather than one issue's neighbours, because bd reports
 *  only the edges an issue carries — what it depends on, whose child it is,
 *  what it was discovered from. The other direction, what an issue blocks and
 *  what was discovered from it, exists nowhere but in the edges other issues
 *  carry, so a detail page that asked only about its own neighbours could
 *  never see it. Only the detail page reads this; the list and the labels page
 *  never mount it, so no bd subprocess runs for them. */
export function useDependencies(sourceId: string) {
  return useQuery({
    queryKey: qk.dependencies(sourceId),
    queryFn: () => issuesClient.listDependencies({ sourceId, issueIds: [] }),
    enabled: sourceId !== "",
  });
}

export function sourceById(sources: Source[], id: string): Source | undefined {
  return sources.find((s) => s.id === id);
}

/** Everything the list URL's query string carries. */
export interface IssueQuery {
  /** The search bar's text (api/query.ts); `is:open` when the URL has no `q`. */
  q: string;
}

/** parseIssueQuery reads the query string (with or without the leading `?`). */
export function parseIssueQuery(search: string): IssueQuery {
  const p = new URLSearchParams(search.startsWith("?") ? search.slice(1) : search);
  // `q=` present but empty means "everything"; absent means the default.
  return { q: p.has("q") ? (p.get("q") ?? "") : DEFAULT_QUERY };
}

/** encodeIssueQuery is parseIssueQuery's inverse; the default is left out so
 *  the plain list URL stays `/issues/{sourceId}`. Returns "" or "?...". */
export function encodeIssueQuery(q: IssueQuery): string {
  if (q.q === DEFAULT_QUERY) return "";
  const p = new URLSearchParams();
  p.set("q", q.q);
  // Keep `q=is:open label:a` readable: colons and commas are valid as they are.
  return `?${p.toString().replace(/%3A/g, ":").replace(/%2C/g, ",")}`;
}

/** Distinct labels of a listing, for the Labels button's count and the search
 *  bar's suggestions. */
export function labelsOf(listing: IssueSummary[]): string[] {
  const seen = new Set<string>();
  for (const s of listing) {
    for (const l of s.labels) seen.add(l);
  }
  return [...seen].sort();
}

/** One row of the labels page. `open` counts the issues that `is:open`
 *  matches — every status but closed — so the number equals what the list
 *  shows for `is:open label:<name>`. */
export interface LabelStat {
  name: string;
  open: number;
  closed: number;
  /** Latest update among the issues carrying the label, in Unix ms. */
  updatedAt: number;
}

/** Every label of a listing with its issue counts, sorted by name. bd has no
 *  label entity, so a label exists exactly as long as an issue carries it. */
export function labelStats(listing: IssueSummary[]): LabelStat[] {
  const byName = new Map<string, LabelStat>();
  for (const s of listing) {
    const updated = timeValue(s.updatedAt);
    for (const l of s.labels) {
      let stat = byName.get(l);
      if (!stat) {
        stat = { name: l, open: 0, closed: 0, updatedAt: 0 };
        byName.set(l, stat);
      }
      if (s.status === IssueStatus.CLOSED) stat.closed++;
      else stat.open++;
      if (updated > stat.updatedAt) stat.updatedAt = updated;
    }
  }
  return [...byName.values()].sort((a, b) => a.name.localeCompare(b.name));
}

/** The listing narrowed by the query text, newest updated first. A query that
 *  does not parse lists nothing; the bar shows the parser's message. */
export function filterIssues(listing: IssueSummary[], q: string): IssueSummary[] {
  const parsed = parseQuery(q);
  if (parsed.error !== "") return [];
  return listing
    .filter((s) => matches(parsed.ast, s))
    .slice()
    .sort((a, b) => timeValue(b.updatedAt) - timeValue(a.updatedAt));
}

/** bd metadata as key=value pairs. */
export function metadataPairs(metadataJson: string): [string, string][] {
  try {
    const obj = JSON.parse(metadataJson || "{}") as Record<string, unknown>;
    return Object.entries(obj).map(([k, v]) => [k, typeof v === "string" ? v : JSON.stringify(v)]);
  } catch {
    return [];
  }
}

/** `Decision:` / `Discussion:` badge, derived from the comment text itself —
 *  the convention is a text prefix, not a field on the wire. */
export function commentKind(c: IssueComment): "Decision" | "Discussion" | "" {
  const m = /^\s*<p>(Decision|Discussion)\s*:/.exec(c.text?.html ?? "");
  return m ? (m[1] as "Decision" | "Discussion") : "";
}

/** Epic progress from child status: any issue with children gets it, not only
 *  plans. */
export function progressOf(s: IssueSummary): { closed: number; total: number } | null {
  if (s.childCount === 0) return null;
  return { closed: s.childClosedCount, total: s.childCount };
}

/** bd's edge kind for the parent link. */
const PARENT_CHILD = "parent-child";

/** One row of the detail page's dependency table, read off the source's edges
 *  rather than off GetIssue, so both directions of an edge get a row. */
export interface DependencyRow {
  /** The issue at the other end. */
  id: string;
  title: string;
  type: string;
  /** True when this issue is the edge's from side: it depends on, is a child
   *  of, or was discovered from the other one. */
  outgoing: boolean;
}

/** Every edge with `issueId` at either end, in the order the source reports
 *  them. */
export function edgesOf(edges: IssueEdge[], issueId: string): IssueEdge[] {
  return edges.filter((e) => e.fromId === issueId || e.toId === issueId);
}

/** The dependency rows of one issue, both directions, titled from `titleOf`
 *  (the edges carry ids only). Parent-child edges are left out: the parent
 *  link in the header and the children table already carry them. */
export function dependencyRows(
  edges: IssueEdge[],
  issueId: string,
  titleOf: (id: string) => string,
): DependencyRow[] {
  const rows: DependencyRow[] = [];
  for (const e of edgesOf(edges, issueId)) {
    if (e.type === PARENT_CHILD) continue;
    const outgoing = e.fromId === issueId;
    const other = outgoing ? e.toId : e.fromId;
    rows.push({ id: other, title: titleOf(other), type: e.type, outgoing });
  }
  return rows;
}

/** How one dependency row reads from this issue's side. */
export function dependencyWording(d: { type: string; outgoing: boolean }): string {
  switch (d.type) {
    case "blocks":
      return d.outgoing ? "depends on" : "blocks";
    case PARENT_CHILD:
      return d.outgoing ? "child of" : "parent of";
    case "discovered-from":
      return d.outgoing ? "discovered from" : "discovered";
    case "related":
      return "related to";
    default:
      return d.outgoing ? `${d.type} →` : `← ${d.type}`;
  }
}
