import type { Issue, IssueStatus, IssueSummary } from "@/api/client.js";
import { getIssue } from "@/api/client.js";
import { type IssueQuery, metadataPairs, progressOf, statusesPresent } from "@/api/issues.js";
import { statusBadgeClass, statusLabel } from "@/lib/format.js";
import { issueHref } from "@/lib/paths.js";
import { Progress } from "./IssueList.js";

// The board view (D14): kanban columns by status, optionally split into
// swimlanes by parent epic, each lane headed by the epic's progress. Cards
// link to the detail page; nothing is draggable (editing is a non-goal).
//
// The columns are the statuses the matching issues have — the default query
// `is:open` lists nothing closed, so the closed column only appears when the
// query asks for closed issues.

interface Lane {
  key: string;
  epic: IssueSummary | null;
  rows: Issue[];
}

export function IssueBoard({
  sourceId,
  rows,
  query,
  search,
  update,
}: {
  sourceId: string;
  rows: Issue[];
  query: IssueQuery;
  search: string;
  update(patch: Partial<IssueQuery>): void;
}) {
  const columns = statusesPresent(rows);
  const lanes = query.lanes ? byEpic(sourceId, rows) : [{ key: "all", epic: null, rows }];

  return (
    <div class="space-y-3" data-testid="issue-board">
      <div class="flex items-center gap-3 text-xs">
        <label class="flex cursor-pointer items-center gap-2">
          <input
            type="checkbox"
            class="toggle toggle-xs"
            checked={query.lanes}
            onChange={(e) => update({ lanes: (e.currentTarget as HTMLInputElement).checked })}
          />
          <span>swimlanes by parent epic</span>
        </label>
        <span class="opacity-60">
          {rows.length} card{rows.length === 1 ? "" : "s"} in {columns.length} column{columns.length === 1 ? "" : "s"}
        </span>
      </div>

      {rows.length === 0 && <div class="p-3 text-sm opacity-60">No issue matches the filters.</div>}

      {lanes.map((lane) => (
        <section key={lane.key} class="rounded-box border border-base-300 bg-base-100 p-3 shadow-sm" data-testid="lane">
          {query.lanes && <LaneHeader sourceId={sourceId} epic={lane.epic} count={lane.rows.length} search={search} />}
          <div class="grid gap-3" style={{ gridTemplateColumns: `repeat(${columns.length}, minmax(180px, 1fr))` }}>
            {columns.map((status) => (
              <Column
                key={status}
                sourceId={sourceId}
                status={status}
                rows={lane.rows.filter((i) => i.summary.status === status)}
                search={search}
              />
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}

/** Groups the rows by parent epic, in the rows' own (newest updated) order of
 *  first appearance; issues without a parent form the last lane. The epic is
 *  looked up in the whole source: a lane's epic need not match the filter. */
function byEpic(sourceId: string, rows: Issue[]): Lane[] {
  const lanes = new Map<string, Lane>();
  const orphans: Issue[] = [];
  for (const i of rows) {
    const p = i.summary.parentId;
    if (p === "") {
      orphans.push(i);
      continue;
    }
    let lane = lanes.get(p);
    if (!lane) {
      lane = { key: p, epic: getIssue(sourceId, p)?.summary ?? null, rows: [] };
      lanes.set(p, lane);
    }
    lane.rows.push(i);
  }
  const out = [...lanes.values()];
  if (orphans.length > 0) out.push({ key: "", epic: null, rows: orphans });
  return out;
}

function LaneHeader({
  sourceId,
  epic,
  count,
  search,
}: {
  sourceId: string;
  epic: IssueSummary | null;
  count: number;
  search: string;
}) {
  if (!epic) {
    return <div class="mb-2 text-xs font-semibold uppercase tracking-wide opacity-60">No parent ({count})</div>;
  }
  const progress = progressOf(epic);
  return (
    <div class="mb-2 flex flex-wrap items-center gap-x-3 gap-y-1">
      <a class="link font-mono text-xs" href={issueHref(sourceId, epic.id, search)}>
        {epic.id}
      </a>
      <a class="link link-hover text-sm font-medium" href={issueHref(sourceId, epic.id, search)}>
        {epic.title}
      </a>
      <span class={`badge badge-xs ${statusBadgeClass(epic.status)}`}>{statusLabel(epic.status)}</span>
      {epic.labels.map((l) => (
        <span key={l} class="badge badge-ghost badge-xs">
          {l}
        </span>
      ))}
      <span class="text-[11px] opacity-60">{count} shown</span>
      {progress && <Progress closed={progress.closed} total={progress.total} />}
    </div>
  );
}

function Column({ sourceId, status, rows, search }: { sourceId: string; status: IssueStatus; rows: Issue[]; search: string }) {
  const sorted = rows.slice().sort((a, b) => a.summary.priority - b.summary.priority);
  return (
    <div class="min-w-0 rounded-box bg-base-200 p-2" data-testid={`column-${statusLabel(status)}`}>
      <div class="mb-2 flex items-center justify-between px-1 text-xs">
        <span class={`badge badge-sm ${statusBadgeClass(status)}`}>{statusLabel(status)}</span>
        <span class="opacity-60">{rows.length}</span>
      </div>
      <div class="space-y-2">
        {sorted.map((i) => (
          <Card key={i.summary.id} sourceId={sourceId} s={i.summary} search={search} />
        ))}
      </div>
    </div>
  );
}

function Card({ sourceId, s, search }: { sourceId: string; s: IssueSummary; search: string }) {
  const progress = progressOf(s);
  const metadata = metadataPairs(s.metadataJson);
  return (
    <a
      href={issueHref(sourceId, s.id, search)}
      class="block rounded-box border border-base-300 bg-base-100 p-2 text-sm shadow-sm hover:border-primary"
      data-testid="card"
    >
      <div class="flex flex-wrap items-center gap-1 text-[11px]">
        <span class="font-mono opacity-70">{s.id}</span>
        <span class="badge badge-outline badge-xs">{s.issueType}</span>
        <span class="opacity-60">P{s.priority}</span>
      </div>
      <div class="mt-0.5 leading-snug">{s.title}</div>
      {(s.labels.length > 0 || metadata.length > 0) && (
        <div class="mt-1 flex flex-wrap gap-1 text-[11px]">
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
        </div>
      )}
      {progress && <Progress closed={progress.closed} total={progress.total} />}
    </a>
  );
}
