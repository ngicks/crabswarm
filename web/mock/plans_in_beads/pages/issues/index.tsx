import { drawerOpen } from "#src/signals/ui.js";
import { type Issue, listDependencies, listSources } from "@/api/client.js";
import { type IssueQuery, sourceById } from "@/api/issues.js";
import { safeDecode, sourceHref } from "@/lib/paths.js";
import { IssueBoard } from "./IssueBoard.js";
import { IssueFilters } from "./IssueFilters.js";
import { IssueGraph } from "./IssueGraph.js";
import { IssueList } from "./IssueList.js";
import { IssueView } from "./IssueView.js";
import { SourceSwitcher } from "./SourceSwitcher.js";
import { ViewTabs } from "./ViewTabs.js";
import { useIssueList, useIssueQuery } from "./useIssues.js";

// The issues screen (PLAN.md "SPA routes"): /issues/{sourceId} shows one of
// the three views (D14) picked by ?view=, /issues/{sourceId}/{issueId} opens
// one issue in the same frame, and / picks a source before either exists.
//
// The left column (bg-base-200, as the file browser's) holds the source
// switcher and the filter bar; the filters live in the query string, so the
// views and the detail page share them. The main column holds the view strip
// and, under it, the view or the open issue.

export function IssuesPage({ sourceId = "", issueId = "" }: { sourceId?: string; issueId?: string }) {
  const id = safeDecode(sourceId);
  const openId = safeDecode(issueId);
  const source = sourceById(id);
  const { query, search, update, reset } = useIssueQuery();
  const { rows, labels } = useIssueList(id, query);

  const side = source ? (
    <IssueFilters query={query} labels={labels} matches={rows.length} update={update} reset={reset} />
  ) : (
    <div class="p-3 text-xs opacity-50">Pick a source to list its issues.</div>
  );

  return (
    <div class="flex min-h-0 flex-1">
      <aside class="hidden w-[320px] shrink-0 flex-col border-r border-base-300 bg-base-200 text-base-content lg:flex">
        <SourceSwitcher activeSourceId={id} />
        {side}
      </aside>

      {drawerOpen.value && (
        <div class="fixed inset-0 z-40 lg:hidden">
          <div
            class="absolute inset-0 bg-black/40"
            onClick={() => {
              drawerOpen.value = false;
            }}
          />
          <aside class="absolute left-0 top-0 flex h-full w-[85%] max-w-[340px] flex-col bg-base-200 shadow-xl">
            <SourceSwitcher activeSourceId={id} />
            {side}
          </aside>
        </div>
      )}

      <main class="min-w-0 flex-1 overflow-auto bg-base-200 p-4 sm:p-6">
        {!source ? (
          <Placeholder text={`No source ${id} is registered.`} />
        ) : (
          <div class="space-y-4">
            <div class="flex flex-wrap items-center gap-3">
              <ViewTabs sourceId={id} query={query} onIssue={openId !== ""} />
              {openId !== "" && (
                <span class="text-xs opacity-60">
                  viewing <span class="font-mono">{openId}</span>; pick a view to go back
                </span>
              )}
            </div>
            {openId !== "" ? (
              <IssueView sourceId={id} issueId={openId} search={search} />
            ) : query.view === "board" ? (
              <IssueBoard sourceId={id} rows={rows} query={query} search={search} update={update} />
            ) : query.view === "graph" ? (
              <GraphView sourceId={id} rows={rows} query={query} search={search} update={update} />
            ) : (
              <IssueList sourceId={id} rows={rows} search={search} />
            )}
          </div>
        )}
      </main>
    </div>
  );
}

// The graph view over the filtered set (D14, D15). Issues no edge touches are
// left out unless asked for: mermaid stacks them in one tall column and most
// of a backlog is unconnected, so drawing them buries the graph.
function GraphView({
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
  const edges = listDependencies(
    sourceId,
    rows.map((i) => i.summary.id),
  );
  const connected = new Set<string>();
  for (const e of edges) {
    connected.add(e.fromId);
    connected.add(e.toId);
  }
  const drawn = query.isolated ? rows : rows.filter((i) => connected.has(i.summary.id));
  const hidden = rows.length - drawn.length;
  return (
    <IssueGraph
      sourceId={sourceId}
      nodes={drawn.map((i) => ({ id: i.summary.id, title: i.summary.title, status: i.summary.status }))}
      edges={edges}
      search={search}
      toolbar={
        <label class="flex cursor-pointer items-center gap-2 opacity-100">
          <input
            type="checkbox"
            class="toggle toggle-xs"
            checked={query.isolated}
            onChange={(e) => update({ isolated: (e.currentTarget as HTMLInputElement).checked })}
          />
          <span data-testid="graph-isolated">
            {query.isolated ? "unconnected issues shown" : `${hidden} unconnected hidden`}
          </span>
        </label>
      }
    />
  );
}

// Landing page at /: the registered sources, before one is chosen.
export function SourcePicker() {
  return (
    <main class="min-h-0 flex-1 overflow-auto bg-base-200 p-6">
      <div class="mx-auto max-w-2xl">
        <h1 class="mb-1 text-2xl font-bold">crabswarm preview — issues</h1>
        <p class="mb-4 text-sm opacity-70">
          One source per repository, keyed by its <code class="font-mono">.beads</code> path (D13).
        </p>
        <ul class="menu w-full">
          {listSources().map((s) => (
            <li key={s.id}>
              <a href={sourceHref(s.id)}>
                <span class="font-medium">{s.prefix}</span>
                <span class="ml-2 truncate text-xs opacity-60">{s.beadsPath}</span>
              </a>
            </li>
          ))}
        </ul>
      </div>
    </main>
  );
}

function Placeholder({ text }: { text: string }) {
  return <div class="p-6 text-sm opacity-60">{text}</div>;
}
