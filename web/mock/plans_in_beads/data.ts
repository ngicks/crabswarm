// Fixture data layer. In the real feature these shapes come from
// IssuesService.ListIssues / GetIssue (proto package ngicks.crabswarm.issues.v1,
// PLAN.md "Proto"); here they are read once out of fixtures.json, which
// doc/plan/2026-09-04-plans_in_beads/mock/gen.go renders ahead of time with the
// previewer's own markdown renderer.
//
// fixtures.json is imported with ?raw and parsed at startup: a typed JSON
// import would make tsc infer a literal type for a 200 kB file for no gain.
import { signal } from "@preact/signals";
import raw from "./fixtures.json?raw";

/** preview.v1.Heading, as carried by RenderedField.toc. */
export interface Heading {
  level: number;
  text: string;
  id: string;
}

/** issues.v1.RenderedField: markdown rendered to an HTML fragment plus its TOC. */
export interface RenderedField {
  html: string;
  toc: Heading[];
}

/** issues.v1.IssueStatus, in protobuf-JSON enum spelling. */
export type IssueStatus =
  | "ISSUE_STATUS_UNSPECIFIED"
  | "ISSUE_STATUS_OPEN"
  | "ISSUE_STATUS_IN_PROGRESS"
  | "ISSUE_STATUS_BLOCKED"
  | "ISSUE_STATUS_CLOSED";

/** issues.v1.IssueSummary. */
export interface IssueSummary {
  id: string;
  title: string;
  issueType: string;
  status: IssueStatus;
  priority: number;
  labels: string[];
  parentId: string;
  commentCount: number;
  childCount: number;
  createdAt: string;
  updatedAt: string;
}

/** issues.v1.IssueComment. */
export interface IssueComment {
  id: string;
  author: string;
  text: RenderedField;
  createdAt: string;
}

/** issues.v1.IssueDependency. `outgoing` is this issue -> the other one. */
export interface IssueDependency {
  id: string;
  title: string;
  type: string;
  outgoing: boolean;
}

/** issues.v1.Issue, plus sourceId (see MOCK_LIMITS.md "Shape deviations"). */
export interface Issue {
  sourceId: string;
  summary: IssueSummary;
  description: RenderedField;
  design: RenderedField;
  acceptanceCriteria: RenderedField;
  notes: RenderedField;
  metadataJson: string;
  closeReason: RenderedField;
  comments: IssueComment[];
  children: IssueSummary[];
  dependencies: IssueDependency[];
}

/** issues.v1.Source: one registered beads database (D13). */
export interface Source {
  id: string;
  prefix: string;
  beadsPath: string;
  dir: string;
}

interface Fixtures {
  sources: Source[];
  issues: Issue[];
}

const fixtures = JSON.parse(raw) as Fixtures;

export const sources: Source[] = fixtures.sources;

/** Every issue of every source. A signal so "simulate change" can push. */
export const issues = signal<Issue[]>(fixtures.issues);

/** The issue the reader has open, so the simulated push can target it. */
export const openIssueId = signal<string>("");

/** The most recent simulated push, shown in the tab header. */
export const lastSimulated = signal<{ id: string; at: string } | null>(null);

export const ALL_STATUSES: IssueStatus[] = [
  "ISSUE_STATUS_OPEN",
  "ISSUE_STATUS_IN_PROGRESS",
  "ISSUE_STATUS_BLOCKED",
  "ISSUE_STATUS_CLOSED",
];

const STATUS_LABELS: Record<IssueStatus, string> = {
  ISSUE_STATUS_UNSPECIFIED: "unknown",
  ISSUE_STATUS_OPEN: "open",
  ISSUE_STATUS_IN_PROGRESS: "in_progress",
  ISSUE_STATUS_BLOCKED: "blocked",
  ISSUE_STATUS_CLOSED: "closed",
};

const STATUS_BADGES: Record<IssueStatus, string> = {
  ISSUE_STATUS_UNSPECIFIED: "badge-ghost",
  ISSUE_STATUS_OPEN: "badge-info",
  ISSUE_STATUS_IN_PROGRESS: "badge-warning",
  ISSUE_STATUS_BLOCKED: "badge-error",
  ISSUE_STATUS_CLOSED: "badge-success",
};

export function statusLabel(s: IssueStatus): string {
  return STATUS_LABELS[s] ?? s;
}

export function statusBadgeClass(s: IssueStatus): string {
  return STATUS_BADGES[s] ?? "badge-ghost";
}

export function sourceById(id: string): Source | undefined {
  return sources.find((s) => s.id === id);
}

export function findIssue(sourceId: string, issueId: string): Issue | undefined {
  return issues.value.find((i) => i.sourceId === sourceId && i.summary.id === issueId);
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
  for (const i of issues.value) {
    if (i.sourceId !== sourceId) continue;
    for (const l of i.summary.labels) seen.add(l);
  }
  return [...seen].sort();
}

/** ListIssues' contract: newest-updated first, filtered. An empty status
 *  filter means bd's default — open, in progress and blocked. */
export function listIssues(sourceId: string, f: IssueFilter): Issue[] {
  const statuses =
    f.statuses.length > 0
      ? f.statuses
      : (["ISSUE_STATUS_OPEN", "ISSUE_STATUS_IN_PROGRESS", "ISSUE_STATUS_BLOCKED"] as IssueStatus[]);
  const needle = f.search.trim().toLowerCase();
  return issues.value
    .filter((i) => i.sourceId === sourceId)
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

const SIMULATED_SUFFIX = / \(simulated update (\d+)\)$/;

/** Stands in for a WatchIssues push (D8): bump one issue's title and
 *  updated_at in memory. Everything reading `issues` re-renders, and the list
 *  reorders, because it is sorted newest-updated first. */
export function simulateChange(sourceId: string, preferredId: string): string | null {
  const list = issues.value;
  const target =
    list.find((i) => i.sourceId === sourceId && i.summary.id === preferredId) ??
    list.find((i) => i.sourceId === sourceId && i.summary.status === "ISSUE_STATUS_IN_PROGRESS") ??
    list.find((i) => i.sourceId === sourceId);
  if (!target) return null;

  const id = target.summary.id;
  const m = SIMULATED_SUFFIX.exec(target.summary.title);
  const bump = m ? Number(m[1]) + 1 : 1;
  const title = `${target.summary.title.replace(SIMULATED_SUFFIX, "")} (simulated update ${bump})`;
  // A pushed change must land newest, so the reordering of the list is visible.
  // The fixture's own stamps are synthetic and can sit ahead of the wall clock.
  const newest = list.reduce((acc, i) => (i.summary.updatedAt > acc ? i.summary.updatedAt : acc), "");
  const after = Date.parse(newest);
  const at = new Date(Math.max(Date.now(), Number.isNaN(after) ? 0 : after + 60_000)).toISOString();

  issues.value = list.map((i) => {
    if (i.sourceId !== sourceId) return i;
    if (i.summary.id === id) return { ...i, summary: { ...i.summary, title, updatedAt: at } };
    if (i.children.some((c) => c.id === id)) {
      return {
        ...i,
        children: i.children.map((c) => (c.id === id ? { ...c, title, updatedAt: at } : c)),
      };
    }
    return i;
  });
  lastSimulated.value = { id, at };
  return id;
}

export function issueHref(sourceId: string, issueId: string): string {
  return `/issues/${encodeURIComponent(sourceId)}/${encodeURIComponent(issueId)}`;
}

export function sourceHref(sourceId: string): string {
  return `/issues/${encodeURIComponent(sourceId)}`;
}

/** Short "2026-09-04 14:41" stamp; the fixture carries RFC 3339. */
export function shortTime(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`;
}
