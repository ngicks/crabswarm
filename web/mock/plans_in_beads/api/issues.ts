// Query-like accessors over the fixture "service" in client.ts: the query the
// search bar holds and its spelling in the URL (PLAN.md "SPA routes": `q`
// carries the whole GitHub-style query, `view` and the view options ride
// beside it), and the small decodes the issue views need. In the real feature
// these are the query options of web/src/api/issues.ts and the filtering
// happens in the daemon; here it happens in the browser.
import {
  type Issue,
  type IssueComment,
  type IssueDependency,
  type IssueStatus,
  type IssueSummary,
  type Source,
  listIssues,
  listSources,
} from "./client.js";
import { DEFAULT_QUERY, matches, parseQuery } from "./query.js";

export const ALL_STATUSES: IssueStatus[] = [
  "ISSUE_STATUS_OPEN",
  "ISSUE_STATUS_IN_PROGRESS",
  "ISSUE_STATUS_BLOCKED",
  "ISSUE_STATUS_CLOSED",
];

export function sourceById(id: string): Source | undefined {
  return listSources().find((s) => s.id === id);
}

/** The views one source offers (D14); `list` is the default. */
export type View = "list" | "board" | "graph";
export const ALL_VIEWS: View[] = ["list", "board", "graph"];

/** Everything the list URL's query string carries. */
export interface IssueQuery {
  view: View;
  /** The search bar's text (api/query.ts); `is:open` when the URL has no `q`. */
  q: string;
  /** Board only: swimlanes by parent epic. */
  lanes: boolean;
  /** Graph only: draw issues that no edge touches. Off by default: mermaid
   *  stacks them in one column, and a backlog is mostly unconnected. */
  isolated: boolean;
}

export const emptyQuery: IssueQuery = {
  view: "list",
  q: DEFAULT_QUERY,
  lanes: true,
  isolated: false,
};

/** parseIssueQuery reads the query string (with or without the leading `?`). */
export function parseIssueQuery(search: string): IssueQuery {
  const p = new URLSearchParams(search.startsWith("?") ? search.slice(1) : search);
  const view = p.get("view") ?? "";
  return {
    view: ALL_VIEWS.includes(view as View) ? (view as View) : "list",
    // `q=` present but empty means "everything"; absent means the default.
    q: p.has("q") ? (p.get("q") ?? "") : DEFAULT_QUERY,
    lanes: p.get("lanes") !== "none",
    isolated: p.get("isolated") === "show",
  };
}

/** encodeIssueQuery is parseIssueQuery's inverse; defaults are left out so the
 *  plain list URL stays `/issues/{sourceId}`. Returns "" or "?...". */
export function encodeIssueQuery(q: IssueQuery): string {
  const p = new URLSearchParams();
  if (q.view !== "list") p.set("view", q.view);
  if (q.q !== DEFAULT_QUERY) p.set("q", q.q);
  if (!q.lanes) p.set("lanes", "none");
  if (q.isolated) p.set("isolated", "show");
  // Keep `q=is:open label:a` readable: colons and commas are valid as they are.
  const s = p.toString().replace(/%3A/g, ":").replace(/%2C/g, ",");
  return s === "" ? "" : `?${s}`;
}

/** Distinct labels of a source, for the label picker and the suggestions. */
export function labelsOf(sourceId: string): string[] {
  const seen = new Set<string>();
  for (const i of listIssues(sourceId)) {
    for (const l of i.summary.labels) seen.add(l);
  }
  return [...seen].sort();
}

/** ListIssues' contract: newest-updated first, filtered by the query. A query
 *  that does not parse lists nothing; the bar shows the parser's message. */
export function filterIssues(sourceId: string, q: IssueQuery): Issue[] {
  const parsed = parseQuery(q.q);
  if (parsed.error !== "") return [];
  return listIssues(sourceId)
    .filter((i) => matches(parsed.ast, i.summary))
    .slice()
    .sort((a, b) => b.summary.updatedAt.localeCompare(a.summary.updatedAt));
}

/** The statuses the rows actually have, in canonical order: the board's
 *  columns. A query without closed issues gets no empty closed column. */
export function statusesPresent(rows: Issue[]): IssueStatus[] {
  const seen = new Set(rows.map((i) => i.summary.status));
  return ALL_STATUSES.filter((s) => seen.has(s));
}

/** bd metadata as key=value pairs; D7's `idea_gate` is one of them. */
export function metadataPairs(metadataJson: string): [string, string][] {
  try {
    const obj = JSON.parse(metadataJson || "{}") as Record<string, unknown>;
    return Object.entries(obj).map(([k, v]) => [k, typeof v === "string" ? v : JSON.stringify(v)]);
  } catch {
    return [];
  }
}

/** `Decision:` / `Discussion:` badge, derived from the comment text itself —
 *  the convention (D1) is a text prefix, not a field on the wire. */
export function commentKind(c: IssueComment): "Decision" | "Discussion" | "" {
  const m = /^\s*<p>(Decision|Discussion)\s*:/.exec(c.text.html);
  return m ? (m[1] as "Decision" | "Discussion") : "";
}

/** Epic progress from child status (D14): any issue with children gets it,
 *  not only plans. */
export function progressOf(s: IssueSummary): { closed: number; total: number } | null {
  if (s.childCount === 0) return null;
  return { closed: s.childClosedCount, total: s.childCount };
}

/** How one dependency row reads from this issue's side. */
export function dependencyWording(d: IssueDependency): string {
  switch (d.type) {
    case "blocks":
      return d.outgoing ? "depends on" : "blocks";
    case "parent-child":
      return d.outgoing ? "child of" : "parent of";
    case "discovered-from":
      return d.outgoing ? "discovered from" : "discovered";
    case "related":
      return "related to";
    default:
      return d.outgoing ? `${d.type} →` : `← ${d.type}`;
  }
}
