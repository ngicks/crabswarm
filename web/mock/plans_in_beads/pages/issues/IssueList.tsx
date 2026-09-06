import type { ComponentChildren } from "preact";
import type { Issue, IssueSummary } from "@/api/client.js";
import { metadataPairs, progressOf } from "@/api/issues.js";
import { shortTime, statusBadgeClass, statusLabel } from "@/lib/format.js";
import { issueHref } from "@/lib/paths.js";

// The list (D14, D20): one table row per matching issue, newest updated
// first, with the affordances every issue gets when it has the data — an epic
// progress bar from child status, metadata chips (so `idea_gate_passed` shows). A
// plan looks like any other epic here; only the Plans button knows the label.

export function IssueList({
  sourceId,
  rows,
  search,
  header,
}: {
  sourceId: string;
  rows: Issue[];
  search: string;
  /** The row above the table: the Open / Closed / Plans buttons. */
  header?: ComponentChildren;
}) {
  return (
    <div class="overflow-x-auto rounded-box border border-base-300 bg-base-100 shadow-sm" data-testid="issue-list">
      {header !== undefined && <div class="border-b border-base-300 bg-base-200/60 px-2 py-1.5">{header}</div>}
      {rows.length === 0 && <div class="p-4 text-sm opacity-60">No issue matches the query.</div>}
      {rows.length > 0 && (
      <table class="table text-sm">
        <thead>
          <tr>
            <th class="w-44">id</th>
            <th class="w-20">type</th>
            <th class="w-28">status</th>
            <th class="w-10">P</th>
            <th>title</th>
            <th class="w-40">updated</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((i) => (
            <Row key={i.summary.id} sourceId={sourceId} s={i.summary} search={search} />
          ))}
        </tbody>
      </table>
      )}
    </div>
  );
}

function Row({ sourceId, s, search }: { sourceId: string; s: IssueSummary; search: string }) {
  const href = issueHref(sourceId, s.id, search);
  const progress = progressOf(s);
  const metadata = metadataPairs(s.metadataJson);
  return (
    <tr class="hover">
      <td class="whitespace-nowrap font-mono text-xs">
        <a class="link" href={href}>
          {s.id}
        </a>
      </td>
      <td>
        <span class="badge badge-outline badge-xs">{s.issueType}</span>
      </td>
      <td>
        <span class={`badge badge-xs ${statusBadgeClass(s.status)}`}>{statusLabel(s.status)}</span>
      </td>
      <td class="text-xs opacity-70">P{s.priority}</td>
      <td class="min-w-64">
        <a class="link link-hover" href={href}>
          {s.title}
        </a>
        <div class="mt-0.5 flex flex-wrap items-center gap-1 text-xs">
          {s.labels.map((l) => (
            <span key={l} class="badge badge-ghost badge-xs">
              {l}
            </span>
          ))}
          {metadata.map(([k, v]) => (
            <span key={k} class="badge badge-outline badge-xs font-mono">
              {k}={v}
            </span>
          ))}
          {s.commentCount > 0 && <span class="opacity-60">{s.commentCount} comments</span>}
        </div>
        {progress && <Progress closed={progress.closed} total={progress.total} />}
      </td>
      <td class="whitespace-nowrap text-xs opacity-70">{shortTime(s.updatedAt)}</td>
    </tr>
  );
}

/** Epic progress: closed children over all children (D14). Shared by the
 *  list rows, the board's cards and lanes, and the detail header. */
export function Progress({ closed, total }: { closed: number; total: number }) {
  return (
    <div class="mt-1 flex items-center gap-2 text-xs opacity-80" data-testid="progress">
      <progress class="progress progress-success h-1.5 w-32" value={closed} max={total} />
      <span>
        {closed}/{total} closed
      </span>
    </div>
  );
}
