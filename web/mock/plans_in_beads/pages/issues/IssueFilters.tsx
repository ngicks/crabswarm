import { Combobox, useListCollection } from "@ark-ui/react/combobox";
import { useFilter } from "@ark-ui/react/locale";
import { Portal } from "@ark-ui/react/portal";
import { ToggleGroup } from "@ark-ui/react/toggle-group";
import type { IssueStatus } from "@/api/client.js";
import { ALL_STATUSES, type IssueQuery } from "@/api/issues.js";
import { statusLabel } from "@/lib/format.js";

// Left-pane filter bar, shared by the list, board and graph views (D14): the
// filters ListIssuesRequest carries (statuses, labels), the Plans saved filter
// (`label=plan`) and a title/id search. Every change rewrites the URL's query
// string, so the three views and the detail page all read the same filters.
//
// The status strip and the label picker are Ark UI parts (toggle group,
// combobox) skinned with daisyUI classes, the way the real IssueFilters is
// planned (PLAN.md step 6); the search box and the saved-filter chip are
// plain daisyUI.

interface Props {
  query: IssueQuery;
  labels: string[];
  matches: number;
  update(patch: Partial<IssueQuery>): void;
  reset(): void;
}

export function IssueFilters({ query, labels, matches, update, reset }: Props) {
  return (
    <div class="space-y-2 border-b border-base-300 p-2">
      <input
        type="search"
        class="input input-sm w-full"
        placeholder="Search title or id"
        value={query.search}
        onInput={(e) => update({ search: (e.currentTarget as HTMLInputElement).value })}
      />

      <ToggleGroup.Root
        multiple
        value={query.statuses}
        onValueChange={(d) => update({ statuses: d.value as IssueStatus[] })}
        className="flex flex-wrap gap-1"
        aria-label="Status"
      >
        {ALL_STATUSES.map((s) => (
          <ToggleGroup.Item
            key={s}
            value={s}
            className={`btn btn-xs ${query.statuses.includes(s) ? "btn-primary" : "btn-outline"}`}
            title={s}
          >
            {statusLabel(s)}
          </ToggleGroup.Item>
        ))}
      </ToggleGroup.Root>
      {query.statuses.length === 0 && <div class="px-0.5 text-[11px] opacity-50">no status picked: open, in_progress and blocked</div>}

      <div class="flex flex-wrap items-center gap-1 text-xs">
        <span class="opacity-60">saved</span>
        <button
          class={`btn btn-xs ${query.savedFilter === "plans" ? "btn-secondary" : "btn-outline"}`}
          title="label=plan"
          onClick={() => update({ savedFilter: query.savedFilter === "plans" ? "" : "plans" })}
          data-testid="saved-plans"
        >
          Plans
        </button>
      </div>

      <LabelPicker labels={labels} selected={query.labels} onChange={(v) => update({ labels: v })} />

      <div class="flex items-center justify-between text-[11px] opacity-60">
        <span>
          {matches} issue{matches === 1 ? "" : "s"}, newest updated first
        </span>
        <button class="btn btn-ghost btn-xs" onClick={reset}>
          reset
        </button>
      </div>
    </div>
  );
}

interface LabelItem {
  label: string;
  value: string;
}

// useFilter memoizes on the identity of its props object: an inline literal
// would rebuild the filter every render, and with it the collection, which
// re-syncs the combobox and renders again without end.
const FILTER_OPTIONS = { sensitivity: "base" } as const;

function LabelPicker({
  labels,
  selected,
  onChange,
}: {
  labels: string[];
  selected: string[];
  onChange(v: string[]): void;
}) {
  const filters = useFilter(FILTER_OPTIONS);
  const { collection, filter } = useListCollection<LabelItem>({
    initialItems: labels.map((l) => ({ label: l, value: l })),
    filter: filters.contains,
  });

  return (
    <Combobox.Root
      multiple
      collection={collection}
      value={selected}
      onValueChange={(d) => onChange(d.value)}
      onInputValueChange={(d) => filter(d.inputValue)}
      closeOnSelect={false}
      className="space-y-1"
    >
      {selected.length > 0 && (
        <div class="flex flex-wrap gap-1" data-testid="label-chips">
          {selected.map((l) => (
            <button
              key={l}
              class="badge badge-ghost badge-sm gap-1"
              onClick={() => onChange(selected.filter((x) => x !== l))}
              title={`remove ${l}`}
            >
              {l}
              <span aria-hidden="true">×</span>
            </button>
          ))}
        </div>
      )}
      <Combobox.Control className="join w-full">
        <Combobox.Input className="input input-sm join-item w-full" placeholder="Filter by label" />
        <Combobox.Trigger className="btn btn-sm join-item" aria-label="Open the label list">
          ▾
        </Combobox.Trigger>
      </Combobox.Control>
      <Portal>
        <Combobox.Positioner>
          {/* Not daisyUI's `menu`: it wraps a height-capped list into columns. */}
          <Combobox.Content className="z-50 max-h-64 w-64 overflow-auto rounded-box border border-base-300 bg-base-100 p-1 text-sm shadow-lg">
            <Combobox.Empty className="px-2 py-1 text-xs opacity-60">No label matches</Combobox.Empty>
            {collection.items.map((item) => (
              <Combobox.Item key={item.value} item={item} className="flex cursor-pointer items-center justify-between rounded px-2 py-1 data-[highlighted]:bg-base-200 data-[state=checked]:font-medium">
                <Combobox.ItemText>{item.label}</Combobox.ItemText>
                <Combobox.ItemIndicator>✓</Combobox.ItemIndicator>
              </Combobox.Item>
            ))}
          </Combobox.Content>
        </Combobox.Positioner>
      </Portal>
    </Combobox.Root>
  );
}
