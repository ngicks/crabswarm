import { drawerOpen } from "../../../../src/signals/ui.js";
import type { IssueStatus } from "../../api/client.js";
import { ALL_STATUSES, emptyFilter } from "../../api/issues.js";
import { statusBadgeClass, statusLabel } from "../../lib/format.js";
import { issueHref } from "../../lib/paths.js";
import { useIssueList } from "./useIssues.js";

// Left-pane issue list: the filters ListIssuesRequest carries (statuses,
// labels) plus two client-side conveniences — "plans only" (D1's plan
// convention: issue_type epic + the `plan` label) and a title/id search.
// Row shape follows FileTree's: one clickable row per item, the active one
// painted with the primary pair.

const ROW = "flex w-full min-w-0 flex-col gap-0.5 px-2 py-1.5 text-left text-sm cursor-pointer";
const ROW_IDLE = ROW + " hover:bg-base-300/70";
const ROW_SELECTED = ROW + " bg-primary text-primary-content font-medium hover:bg-primary/85";

export function IssueList({ sourceId, activeIssueId }: { sourceId: string; activeIssueId: string }) {
  const { filter, setFilter, rows, labels } = useIssueList(sourceId);

  const toggleStatus = (s: IssueStatus) =>
    setFilter((f) => ({
      ...f,
      statuses: f.statuses.includes(s) ? f.statuses.filter((x) => x !== s) : [...f.statuses, s],
    }));

  return (
    <div class="flex min-h-0 flex-1 flex-col">
      <div class="space-y-2 border-b border-base-300 p-2">
        <input
          type="search"
          class="input input-sm w-full"
          placeholder="Search title or id"
          value={filter.search}
          onInput={(e) => {
            const v = (e.currentTarget as HTMLInputElement).value;
            setFilter((f) => ({ ...f, search: v }));
          }}
        />

        <div class="flex flex-wrap gap-1">
          {ALL_STATUSES.map((s) => {
            const on = filter.statuses.includes(s);
            return (
              <button
                key={s}
                class={`btn btn-xs ${on ? "btn-primary" : "btn-outline"}`}
                onClick={() => toggleStatus(s)}
                title={s}
              >
                {statusLabel(s)}
              </button>
            );
          })}
        </div>

        <label class="flex cursor-pointer items-center gap-2 text-xs">
          <input
            type="checkbox"
            class="toggle toggle-xs"
            checked={filter.plansOnly}
            onChange={(e) => {
              const v = (e.currentTarget as HTMLInputElement).checked;
              setFilter((f) => ({ ...f, plansOnly: v }));
            }}
          />
          <span>plans only (epic + label plan)</span>
        </label>

        <select
          multiple
          size={4}
          class="select select-sm h-auto w-full"
          aria-label="Filter by label"
          onChange={(e) => {
            const picked = Array.from((e.currentTarget as HTMLSelectElement).selectedOptions).map((o) => o.value);
            setFilter((f) => ({ ...f, labels: picked }));
          }}
        >
          {labels.map((l) => (
            <option key={l} value={l} selected={filter.labels.includes(l)}>
              {l}
            </option>
          ))}
        </select>

        <div class="flex items-center justify-between text-[11px] opacity-60">
          <span>
            {rows.length} issue{rows.length === 1 ? "" : "s"}, newest updated first
          </span>
          <button class="btn btn-ghost btn-xs" onClick={() => setFilter(emptyFilter)}>
            reset
          </button>
        </div>
      </div>

      <div data-testid="issue-list" class="min-h-0 flex-1 overflow-auto py-1">
        {rows.length === 0 && <div class="p-3 text-xs opacity-50">No issue matches the filters.</div>}
        {rows.map((i) => {
          const s = i.summary;
          return (
            <a
              key={s.id}
              href={issueHref(sourceId, s.id)}
              class={s.id === activeIssueId ? ROW_SELECTED : ROW_IDLE}
              onClick={() => {
                drawerOpen.value = false;
              }}
            >
              <span class="flex flex-wrap items-center gap-1 text-[11px]">
                <span class="font-mono opacity-70">{s.id}</span>
                <span class="badge badge-outline badge-xs">{s.issueType}</span>
                <span class={`badge badge-xs ${statusBadgeClass(s.status)}`}>{statusLabel(s.status)}</span>
                <span class="opacity-60">P{s.priority}</span>
                {s.commentCount > 0 && (
                  <span class="flex items-center gap-0.5 opacity-60">
                    <CommentIcon />
                    {s.commentCount}
                  </span>
                )}
              </span>
              <span class="truncate">{s.title}</span>
            </a>
          );
        })}
      </div>
    </div>
  );
}

function CommentIcon() {
  return (
    <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
    </svg>
  );
}
