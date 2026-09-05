// The mock's stand-in for the wire. In the real feature these shapes come from
// IssuesService.ListIssues / GetIssue / ListDependencies (proto package
// ngicks.crabswarm.issues.v1, PLAN.md "Proto") and this module would hold a
// connect-web client, the way web/src/api/client.ts does; here the messages
// are hand-declared and read once out of api/fixtures.json, which
// doc/plan/2026-09-04-plans_in_beads/mock/gen.go renders ahead of time with
// the previewer's own markdown renderer.
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
  /** Closed children, for the epic progress affordance (D14). */
  childClosedCount: number;
  /** bd metadata, verbatim JSON object; "{}" when unset. */
  metadataJson: string;
}

/** issues.v1.IssueComment. */
export interface IssueComment {
  id: string;
  author: string;
  text: RenderedField;
  createdAt: string;
}

/** issues.v1.IssueDependency. `outgoing` is true when this issue is the
 *  edge's `from`: it depends on, is a child of, or was discovered from the
 *  other issue. */
export interface IssueDependency {
  id: string;
  title: string;
  type: string;
  outgoing: boolean;
}

/** issues.v1.IssueEdge: `fromId` is the dependent, the child or the
 *  discoverer; `toId` the blocker, the parent or the origin. */
export interface IssueEdge {
  fromId: string;
  toId: string;
  type: string;
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
  edges: (IssueEdge & { sourceId: string })[];
}

const fixtures = JSON.parse(raw) as Fixtures;

/**
 * Every issue of every source, standing in for what the daemon would hold. A
 * signal, so the simulated WatchIssues push in events.ts can update it and
 * everything reading it re-renders.
 */
export const issues = signal<Issue[]>(fixtures.issues);

/** ListSources: the registered beads databases (D13). */
export function listSources(): Source[] {
  return fixtures.sources;
}

/** ListIssues without the request's filters; api/issues.ts applies those. */
export function listIssues(sourceId: string): Issue[] {
  return issues.value.filter((i) => i.sourceId === sourceId);
}

/** GetIssue: one issue with its rendered fields, children, deps and comments. */
export function getIssue(sourceId: string, issueId: string): Issue | undefined {
  return issues.value.find((i) => i.sourceId === sourceId && i.summary.id === issueId);
}

/** ListDependencies: every edge among the given issues, or every edge of the
 *  source when `issueIds` is empty. One call, however many issues (D15's
 *  graph is built from this, not from per-issue GetIssue calls). */
export function listDependencies(sourceId: string, issueIds: string[]): IssueEdge[] {
  const within = issueIds.length === 0 ? null : new Set(issueIds);
  return fixtures.edges
    .filter((e) => e.sourceId === sourceId)
    .filter((e) => within === null || (within.has(e.fromId) && within.has(e.toId)))
    .map(({ fromId, toId, type }) => ({ fromId, toId, type }));
}
