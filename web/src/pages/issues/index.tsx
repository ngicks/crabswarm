import type { ComponentChildren } from "preact";
import { useEffect, useMemo } from "preact/hooks";
import { useLocation } from "preact-iso";
import { labelsOf, sourceById, useIssueListing, useSources } from "@/api/issues.js";
import { labelsHref, safeDecode, sourceHref } from "@/lib/paths.js";
import { drawerOpen } from "@/signals/navigation.js";
import { IssueList } from "./IssueList.js";
import { IssueView } from "./IssueView.js";
import { QueryBar } from "./QueryBar.js";
import { SourceSwitcher } from "./SourceSwitcher.js";
import { StateButtons } from "./StateButtons.js";
import { useIssueList, useIssueQuery } from "./useIssues.js";

// The issues screen: /issues/{sourceId} lists a source the way GitHub's issues
// page does — the search bar, then Open / Closed / Plans over the table —
// /issues/{sourceId}/{issueId} opens one issue in the same frame,
// /issues/{sourceId}/labels the labels page, and /issues picks a source before
// any of them exists.
//
// The left column (bg-base-200, as the file browser's) holds the source
// switcher and the way into the labels page. The query lives in the URL's `q`,
// so the bar, the buttons and the detail page share it.

export function IssuesPage({ sourceId = "", issueId = "" }: { sourceId?: string; issueId?: string }) {
  const id = safeDecode(sourceId);
  const openId = safeDecode(issueId);
  const sources = useSources();
  const source = sourceById(sources.data?.sources ?? [], id);
  const { query, search, update, reset } = useIssueQuery();
  const { listing, rows, labels, isLoading } = useIssueList(id, query);
  const suggestCtx = useMemo(() => ({ labels, ids: listing.map((s) => s.id) }), [labels, listing]);

  return (
    <IssuesShell sourceId={id}>
      {sources.isLoading ? (
        <Spinner />
      ) : !source ? (
        <Placeholder text={`No source ${id} is registered.`} />
      ) : openId !== "" ? (
        // The back link lives in the detail page's sticky bar, so the detail
        // view owns the whole column.
        <IssueView sourceId={id} issueId={openId} search={search} listing={listing} />
      ) : isLoading ? (
        <Spinner />
      ) : (
        <div class="space-y-3">
          <QueryBar query={query} matches={rows.length} ctx={suggestCtx} update={update} reset={reset} />
          <IssueList
            sourceId={id}
            rows={rows}
            search={search}
            header={<StateButtons listing={listing} q={query.q} setQ={(q) => update({ q })} />}
          />
        </div>
      )}
    </IssuesShell>
  );
}

/** The two columns every issues route draws: the source switcher and the
 *  Labels entry on the left (also as the small-screen drawer), the routed
 *  content on the right. */
export function IssuesShell({ sourceId, children }: { sourceId: string; children: ComponentChildren }) {
  const loc = useLocation();
  const sources = useSources();
  const known = sourceById(sources.data?.sources ?? [], sourceId) !== undefined;
  const onLabels = loc.path === labelsHref(sourceId);
  const { data } = useIssueListing(known ? sourceId : "");
  // Below `lg` the column is an overlay over the page it navigates: picking a
  // source or the labels page has to put it away, or it covers the result.
  useEffect(() => {
    drawerOpen.value = false;
  }, [loc.path]);
  const side = !known ? (
    <div class="p-3 text-xs opacity-50">Pick a source to list its issues.</div>
  ) : (
    <div class="border-b border-base-300 p-2">
      <div class="px-2 pb-1 text-xs font-semibold uppercase tracking-wide opacity-60">Labels</div>
      <a
        class={`btn btn-ghost w-full justify-start gap-1.5 ${onLabels ? "btn-active" : ""}`}
        href={labelsHref(sourceId)}
        data-testid="labels-link"
      >
        <TagIcon />
        Labels {labelsOf(data?.issues ?? []).length}
      </a>
    </div>
  );
  const sidebar = (
    <>
      <SourceSwitcher sources={sources.data?.sources ?? []} activeSourceId={sourceId} />
      {side}
    </>
  );

  return (
    <div class="flex min-h-0 flex-1">
      <aside class="hidden w-[320px] shrink-0 flex-col border-r border-base-300 bg-base-200 text-base-content lg:flex">
        {sidebar}
      </aside>

      {drawerOpen.value && (
        <div class="fixed inset-0 z-40 lg:hidden">
          <div
            class="absolute inset-0 bg-black/40"
            onClick={() => {
              drawerOpen.value = false;
            }}
          />
          <aside class="absolute left-0 top-0 flex h-full w-[85%] max-w-[340px] flex-col bg-base-200 shadow-xl">{sidebar}</aside>
        </div>
      )}

      <main class="min-w-0 flex-1 overflow-auto bg-base-200 p-4 sm:p-6">
        {/* The sidebar is an overlay below `lg`, and the tab header above this
            page is shared with the file browser, so the way into it lives
            here rather than in the header. */}
        <button
          class="btn btn-square btn-ghost btn-sm mb-2 lg:hidden"
          aria-label="Open the source list"
          onClick={() => {
            drawerOpen.value = true;
          }}
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 6h18M3 12h18M3 18h18" />
          </svg>
        </button>
        {children}
      </main>
    </div>
  );
}

function TagIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
      <path d="M3 12V5a2 2 0 0 1 2-2h7l9 9-9 9z" />
      <circle cx="7.5" cy="7.5" r="1.5" />
    </svg>
  );
}

// Landing page at /issues: the registered sources, before one is chosen.
export function SourcePicker() {
  const { data, isLoading, error } = useSources();
  const sources = data?.sources ?? [];
  return (
    <IssuesShell sourceId="">
      <div class="mx-auto max-w-2xl">
        <h1 class="mb-1 text-2xl font-bold">crabswarm preview — issues</h1>
        <p class="mb-4 text-sm opacity-70">
          One source per repository, keyed by its <code class="font-mono">.beads</code> path.
        </p>
        {isLoading && <span class="loading loading-spinner" />}
        {error && <div class="alert alert-error">Failed to load issue sources.</div>}
        {!isLoading && sources.length === 0 && (
          <div class="alert">
            <span>
              No issue source registered yet. Run <code class="font-mono">crabswarm preview --issue .</code> to add one.
            </span>
          </div>
        )}
        <ul class="menu w-full">
          {sources.map((s) => (
            <li key={s.id}>
              <a href={sourceHref(s.id)}>
                <span class="font-medium">{s.prefix}</span>
                <span class="ml-2 truncate text-xs opacity-60">{s.beadsPath}</span>
              </a>
            </li>
          ))}
        </ul>
      </div>
    </IssuesShell>
  );
}

function Spinner() {
  return (
    <div class="p-6">
      <span class="loading loading-spinner" />
    </div>
  );
}

function Placeholder({ text }: { text: string }) {
  return <div class="p-6 text-sm opacity-60">{text}</div>;
}
