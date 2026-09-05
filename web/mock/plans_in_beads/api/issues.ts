// Query-like accessors over the fixture "service" in client.ts: the filters
// ListIssuesRequest carries, and the small decodes the issue views need. In the
// real feature these are the query options of web/src/api/issues.ts and the
// filtering happens in the daemon; here it happens in the browser.
import { type Issue, type IssueComment, type IssueStatus, type Source, listIssues, listSources } from "./client.js";

export const ALL_STATUSES: IssueStatus[] = [
  "ISSUE_STATUS_OPEN",
  "ISSUE_STATUS_IN_PROGRESS",
  "ISSUE_STATUS_BLOCKED",
  "ISSUE_STATUS_CLOSED",
];

export function sourceById(id: string): Source | undefined {
  return listSources().find((s) => s.id === id);
}

export interface IssueFilter {
  statuses: IssueStatus[];
  labels: string[];
  plansOnly: boolean;
  search: string;
}

export const emptyFilter: IssueFilter = { statuses: [], labels: [], plansOnly: false, search: "" };

/** Distinct labels of a source, for the label multi-select. */
export function labelsOf(sourceId: string): string[] {
  const seen = new Set<string>();
  for (const i of listIssues(sourceId)) {
    for (const l of i.summary.labels) seen.add(l);
  }
  return [...seen].sort();
}

/** ListIssues' contract: newest-updated first, filtered. An empty status
 *  filter means bd's default — open, in progress and blocked. */
export function filterIssues(sourceId: string, f: IssueFilter): Issue[] {
  const statuses =
    f.statuses.length > 0
      ? f.statuses
      : (["ISSUE_STATUS_OPEN", "ISSUE_STATUS_IN_PROGRESS", "ISSUE_STATUS_BLOCKED"] as IssueStatus[]);
  const needle = f.search.trim().toLowerCase();
  return listIssues(sourceId)
    .filter((i) => statuses.includes(i.summary.status))
    .filter((i) => f.labels.every((l) => i.summary.labels.includes(l)))
    .filter((i) => !f.plansOnly || (i.summary.issueType === "epic" && i.summary.labels.includes("plan")))
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
