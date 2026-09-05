// Query-like accessors over the fixture "service" in client.ts: the filters
// ListIssuesRequest carries, their spelling in the URL's query string (PLAN.md
// "SPA routes": status, label, q, filter=plans, plus view), and the small
// decodes the issue views need. In the real feature these are the query
// options of web/src/api/issues.ts and the filtering happens in the daemon;
// here it happens in the browser.
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

export const ALL_STATUSES: IssueStatus[] = [
  "ISSUE_STATUS_OPEN",
  "ISSUE_STATUS_IN_PROGRESS",
  "ISSUE_STATUS_BLOCKED",
  "ISSUE_STATUS_CLOSED",
];

/** bd's default listing: everything that is not closed. */
export const DEFAULT_STATUSES: IssueStatus[] = ["ISSUE_STATUS_OPEN", "ISSUE_STATUS_IN_PROGRESS", "ISSUE_STATUS_BLOCKED"];

export function sourceById(id: string): Source | undefined {
  return listSources().find((s) => s.id === id);
}

/** The views one source offers (D14); `list` is the default. */
export type View = "list" | "board" | "graph";
export const ALL_VIEWS: View[] = ["list", "board", "graph"];

/** Saved filters: `plans` is D14's only one — `label=plan`. */
export type SavedFilter = "" | "plans";

/** Everything the list URL's query string carries. */
export interface IssueQuery {
  view: View;
  statuses: IssueStatus[];
  labels: string[];
  search: string;
  savedFilter: SavedFilter;
  /** Board only: swimlanes by parent epic. */
  lanes: boolean;
  /** Graph only: draw issues that no edge touches. Off by default: mermaid
   *  stacks them in one column, and a backlog is mostly unconnected. */
  isolated: boolean;
}

export const emptyQuery: IssueQuery = {
  view: "list",
  statuses: [],
  labels: [],
  search: "",
  savedFilter: "",
  lanes: true,
  isolated: false,
};

// The status words in the URL are bd's own (open, in_progress, ...), so a
// hand-typed URL reads like a `bd list --status` invocation.
const STATUS_WORDS: [IssueStatus, string][] = [
  ["ISSUE_STATUS_OPEN", "open"],
  ["ISSUE_STATUS_IN_PROGRESS", "in_progress"],
  ["ISSUE_STATUS_BLOCKED", "blocked"],
  ["ISSUE_STATUS_CLOSED", "closed"],
];

function statusFromWord(w: string): IssueStatus | undefined {
  return STATUS_WORDS.find(([, word]) => word === w)?.[0];
}

function wordOfStatus(s: IssueStatus): string {
  return STATUS_WORDS.find(([status]) => status === s)?.[1] ?? "";
}

/** parseIssueQuery reads the query string (with or without the leading `?`). */
export function parseIssueQuery(search: string): IssueQuery {
  const p = new URLSearchParams(search.startsWith("?") ? search.slice(1) : search);
  const view = p.get("view") ?? "";
  const list = (key: string) =>
    (p.get(key) ?? "")
      .split(",")
      .map((s) => s.trim())
      .filter((s) => s !== "");
  return {
    view: ALL_VIEWS.includes(view as View) ? (view as View) : "list",
    statuses: list("status")
      .map(statusFromWord)
      .filter((s): s is IssueStatus => s !== undefined),
    labels: list("label"),
    search: p.get("q") ?? "",
    savedFilter: p.get("filter") === "plans" ? "plans" : "",
    lanes: p.get("lanes") !== "none",
    isolated: p.get("isolated") === "show",
  };
}

/** encodeIssueQuery is parseIssueQuery's inverse; defaults are left out so the
 *  plain list URL stays `/issues/{sourceId}`. Returns "" or "?...". */
export function encodeIssueQuery(q: IssueQuery): string {
  const p = new URLSearchParams();
  if (q.view !== "list") p.set("view", q.view);
  // Statuses in their canonical order, whatever order they were toggled in,
  // so equal filters give equal URLs.
  const statuses = ALL_STATUSES.filter((s) => q.statuses.includes(s));
  if (statuses.length > 0) p.set("status", statuses.map(wordOfStatus).join(","));
  if (q.labels.length > 0) p.set("label", q.labels.slice().sort().join(","));
  if (q.search !== "") p.set("q", q.search);
  if (q.savedFilter !== "") p.set("filter", q.savedFilter);
  if (!q.lanes) p.set("lanes", "none");
  if (q.isolated) p.set("isolated", "show");
  // A comma is a valid query character; keep `status=open,closed` readable.
  const s = p.toString().replace(/%2C/g, ",");
  return s === "" ? "" : `?${s}`;
}

/** Distinct labels of a source, for the label combobox. */
export function labelsOf(sourceId: string): string[] {
  const seen = new Set<string>();
  for (const i of listIssues(sourceId)) {
    for (const l of i.summary.labels) seen.add(l);
  }
  return [...seen].sort();
}

/** The statuses a query actually lists: an empty status filter means bd's
 *  default — open, in progress and blocked — so closed issues only appear
 *  when asked for. The board's columns follow this too. */
export function effectiveStatuses(q: IssueQuery): IssueStatus[] {
  return q.statuses.length > 0 ? ALL_STATUSES.filter((s) => q.statuses.includes(s)) : DEFAULT_STATUSES;
}

/** ListIssues' contract: newest-updated first, filtered. */
export function filterIssues(sourceId: string, q: IssueQuery): Issue[] {
  const statuses = effectiveStatuses(q);
  const needle = q.search.trim().toLowerCase();
  return listIssues(sourceId)
    .filter((i) => statuses.includes(i.summary.status))
    .filter((i) => q.labels.every((l) => i.summary.labels.includes(l)))
    .filter((i) => q.savedFilter !== "plans" || i.summary.labels.includes("plan"))
    .filter(
      (i) =>
        needle === "" ||
        i.summary.title.toLowerCase().includes(needle) ||
        i.summary.id.toLowerCase().includes(needle),
    )
    .slice()
    .sort((a, b) => b.summary.updatedAt.localeCompare(a.summary.updatedAt));
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
