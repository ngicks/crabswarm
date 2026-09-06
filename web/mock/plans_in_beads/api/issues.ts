// Query-like accessors over the fixture "service" in client.ts: the query the
// search bar holds and its spelling in the URL (PLAN.md "SPA routes": `q`
// carries the whole GitHub-style query), and the small decodes the issue
// views need. In the real feature these are the query options of
// web/src/api/issues.ts and the filtering happens in the daemon; here it
// happens in the browser.
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

/** Everything the list URL's query string carries. */
export interface IssueQuery {
  /** The search bar's text (api/query.ts); `is:open` when the URL has no `q`. */
  q: string;
}

export const emptyQuery: IssueQuery = { q: DEFAULT_QUERY };

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

/** Distinct labels of a source, for the Labels button's count and the search
 *  bar's suggestions. */
export function labelsOf(sourceId: string): string[] {
  const seen = new Set<string>();
  for (const i of listIssues(sourceId)) {
    for (const l of i.summary.labels) seen.add(l);
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
  /** Latest updatedAt among the issues carrying the label. */
  updatedAt: string;
}

/** Every label of a source with its issue counts, sorted by name. bd has no
 *  label entity, so a label exists exactly as long as an issue carries it. */
export function labelStats(sourceId: string): LabelStat[] {
  const byName = new Map<string, LabelStat>();
  for (const i of listIssues(sourceId)) {
    const s = i.summary;
    for (const l of s.labels) {
      let stat = byName.get(l);
      if (!stat) {
        stat = { name: l, open: 0, closed: 0, updatedAt: "" };
        byName.set(l, stat);
      }
      if (s.status === "ISSUE_STATUS_CLOSED") stat.closed++;
      else stat.open++;
      if (s.updatedAt > stat.updatedAt) stat.updatedAt = s.updatedAt;
    }
  }
  return [...byName.values()].sort((a, b) => a.name.localeCompare(b.name));
}

/** ListIssues' contract: newest-updated first, filtered by the query text. A
 *  query that does not parse lists nothing; the bar shows the parser's
 *  message. */
export function filterIssues(sourceId: string, q: string): Issue[] {
  const parsed = parseQuery(q);
  if (parsed.error !== "") return [];
  return listIssues(sourceId)
    .filter((i) => matches(parsed.ast, i.summary))
    .slice()
    .sort((a, b) => b.summary.updatedAt.localeCompare(a.summary.updatedAt));
}

/** bd metadata as key=value pairs; D7's `idea_gate_passed` is one of them. */
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
