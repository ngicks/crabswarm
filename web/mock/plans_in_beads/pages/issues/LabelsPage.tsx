import { useEffect, useState } from "preact/hooks";
import { useLocation } from "preact-iso";
import { encodeIssueQuery, labelStats, sourceById } from "@/api/issues.js";
import { tagToken } from "@/api/query.js";
import { shortTime } from "@/lib/format.js";
import { labelsHref, queryOf, safeDecode, sourceHref } from "@/lib/paths.js";
import { IssuesShell } from "./index.js";

// The labels page, GitHub's counterpart to its issue list: every label of one
// source with how many issues carry it.
//
// bd has no label entity and no archive flag — a label exists exactly as long
// as an issue carries it — so this page reads the two states off the issues:
// a label is *active* when at least one open issue carries it, and *archived*
// when only closed issues do.
//
// Both the name filter and the state live in the URL (`q`, `state=archived`),
// the way the list keeps its query, so a filtered page is a link.

type LabelState = "active" | "archived";

export function LabelsPage({ sourceId = "" }: { sourceId?: string }) {
  const id = safeDecode(sourceId);
  const loc = useLocation();
  const params = new URLSearchParams(queryOf(loc.url).slice(1));
  const urlQ = params.get("q") ?? "";
  const state: LabelState = params.get("state") === "archived" ? "archived" : "active";
  const source = sourceById(id);

  const go = (q: string, next: LabelState) => {
    const p = new URLSearchParams();
    if (q !== "") p.set("q", q);
    if (next === "archived") p.set("state", "archived");
    const search = p.toString();
    loc.route(labelsHref(id, search === "" ? "" : `?${search}`), true);
  };

  // The filter is applied as it is typed; the URL follows it (replace, so the
  // back button leaves the page rather than replaying every keystroke).
  //
  // The text is the source of truth once the page is up, and the URL only
  // seeds it: mirroring the URL back into the field would undo a keystroke
  // typed while the previous replace was still in flight.
  const [draft, setDraft] = useState(urlQ);
  useEffect(() => {
    if (draft !== urlQ) go(draft, state);
  }, [draft]);

  const needle = draft.trim().toLowerCase();
  const named = labelStats(id).filter((l) => l.name.toLowerCase().includes(needle));
  const active = named.filter((l) => l.open > 0);
  const archived = named.filter((l) => l.open === 0);
  const rows = state === "archived" ? archived : active;

  return (
    <IssuesShell sourceId={id}>
      {!source ? (
        <div class="p-6 text-sm opacity-60">No source {id} is registered.</div>
      ) : (
        <div class="space-y-4">
          <a class="link link-hover text-sm opacity-70" href={sourceHref(id)}>
            ← back to the list
          </a>
          <h1 class="text-3xl font-semibold">Labels</h1>

          {/* The same daisyUI box the query bar wears, minus the suggestions:
              a label filter is a substring match, not a query language. */}
          <div class="input input-md w-full gap-1 pr-1">
            <input
              class="grow bg-transparent outline-none"
              placeholder="Search all labels"
              aria-label="Search labels"
              data-testid="labels-search"
              value={draft}
              onInput={(e) => setDraft((e.currentTarget as HTMLInputElement).value)}
            />
            <button
              class="btn btn-ghost btn-sm btn-circle"
              onClick={() => setDraft("")}
              title="clear the filter"
              aria-label="Clear the filter"
              data-testid="labels-search-clear"
            >
              ×
            </button>
            <button class="btn btn-ghost btn-sm" onClick={() => go(draft, state)} data-testid="labels-search-apply">
              Search
            </button>
          </div>

          <div class="overflow-x-auto rounded-box border border-base-300 bg-base-100 shadow-sm" data-testid="labels-table">
            <div class="border-b border-base-300 bg-base-200/60 px-2 py-1.5">
              <div class="flex flex-wrap items-center gap-1 text-sm">
                <button
                  class={`btn btn-ghost ${state === "active" ? "font-semibold" : "opacity-70"}`}
                  onClick={() => go(draft, "active")}
                  data-testid="labels-active"
                >
                  <OpenIcon />
                  Active {active.length}
                </button>
                <button
                  class={`btn btn-ghost ${state === "archived" ? "font-semibold" : "opacity-70"}`}
                  onClick={() => go(draft, "archived")}
                  data-testid="labels-archived"
                >
                  <ArchiveIcon />
                  Archived {archived.length}
                </button>
              </div>
            </div>
            <table class="table text-sm">
              <thead>
                <tr>
                  <th class="w-64">label</th>
                  <th class="w-32">open issues</th>
                  <th class="w-32">closed issues</th>
                  <th class="w-40">last updated</th>
                </tr>
              </thead>
              <tbody>
                {rows.length === 0 && (
                  <tr>
                    <td colSpan={4} class="text-sm opacity-60" data-testid="labels-empty">
                      no labels match
                    </td>
                  </tr>
                )}
                {rows.map((l) => (
                  <Row key={l.name} sourceId={id} name={l.name} open={l.open} closed={l.closed} updatedAt={l.updatedAt} />
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </IssuesShell>
  );
}

function Row({
  sourceId,
  name,
  open,
  closed,
  updatedAt,
}: {
  sourceId: string;
  name: string;
  open: number;
  closed: number;
  updatedAt: string;
}) {
  // The links are the list's own query, spelled by the same helpers the bar
  // and the state buttons write with, so the list reads back what it wrote.
  const href = (state: "open" | "closed") =>
    sourceHref(sourceId, encodeIssueQuery({ q: `${tagToken("is", state)} ${tagToken("label", name)}` }));
  return (
    <tr class="hover">
      <td>
        <a class="badge badge-ghost whitespace-nowrap" href={href("open")} data-testid="label-name">
          {name}
        </a>
      </td>
      <td>
        <a class="link link-hover" href={href("open")} data-testid="label-open">
          {open}
        </a>
      </td>
      <td>
        <a class="link link-hover" href={href("closed")} data-testid="label-closed">
          {closed}
        </a>
      </td>
      <td class="whitespace-nowrap text-xs opacity-70">{shortTime(updatedAt)}</td>
    </tr>
  );
}

function OpenIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
      <circle cx="12" cy="12" r="9" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

function ArchiveIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
      <path d="M3 7h18v3H3zM5 10v9h14v-9M10 14h4" />
    </svg>
  );
}
