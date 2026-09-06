import { useMemo } from "preact/hooks";
import { drawerOpen } from "#src/signals/ui.js";
import { listIssues, listSources } from "@/api/client.js";
import { sourceById } from "@/api/issues.js";
import { safeDecode, sourceHref } from "@/lib/paths.js";
import { IssueFilters } from "./IssueFilters.js";
import { IssueList } from "./IssueList.js";
import { IssueView } from "./IssueView.js";
import { QueryBar } from "./QueryBar.js";
import { SourceSwitcher } from "./SourceSwitcher.js";
import { StateButtons } from "./StateButtons.js";
import { useIssueList, useIssueQuery } from "./useIssues.js";

// The issues screen (PLAN.md "SPA routes"): /issues/{sourceId} lists a
// source the way GitHub's issues page does — the search bar, then Open /
// Closed / Plans over the table — and /issues/{sourceId}/{issueId} opens one
// issue in the same frame; / picks a source before either exists.
//
// The left column (bg-base-200, as the file browser's) holds the source
// switcher and the label picker. The query lives in the URL's `q`, so the
// bar, the buttons, the picker and the detail page share it.

export function IssuesPage({ sourceId = "", issueId = "" }: { sourceId?: string; issueId?: string }) {
  const id = safeDecode(sourceId);
  const openId = safeDecode(issueId);
  const source = sourceById(id);
  const { query, search, update, reset } = useIssueQuery();
  const { rows, labels } = useIssueList(id, query);
  const suggestCtx = useMemo(() => ({ labels, ids: listIssues(id).map((i) => i.summary.id) }), [labels, id]);

  const side = !source ? (
    <div class="p-3 text-xs opacity-50">Pick a source to list its issues.</div>
  ) : openId === "" ? (
    <IssueFilters query={query} labels={labels} matches={rows.length} update={update} reset={reset} />
  ) : null;

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
        ) : openId !== "" ? (
          <div class="space-y-4">
            <a class="link link-hover text-sm opacity-70" href={sourceHref(id, search)}>
              ← back to the list
            </a>
            <IssueView sourceId={id} issueId={openId} search={search} />
          </div>
        ) : (
          <div class="space-y-3">
            <QueryBar query={query} matches={rows.length} ctx={suggestCtx} update={update} reset={reset} />
            <IssueList
              sourceId={id}
              rows={rows}
              search={search}
              header={<StateButtons sourceId={id} q={query.q} setQ={(q) => update({ q })} />}
            />
          </div>
        )}
      </main>
    </div>
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
