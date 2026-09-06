import { Combobox, useListCollection } from "@ark-ui/react/combobox";
import { useFilter } from "@ark-ui/react/locale";
import { Portal } from "@ark-ui/react/portal";
import { ToggleGroup } from "@ark-ui/react/toggle-group";
import type { IssueStatus } from "@/api/client.js";
import { ALL_STATUSES, type IssueQuery } from "@/api/issues.js";
import { hasTag, tagValues, toggleTag } from "@/api/query.js";
import { statusLabel } from "@/lib/format.js";

// Left-pane quick filters, shared by the list, board and graph views (D14).
// They keep no state of their own: each widget reads its tags out of the
// search bar's query and toggles `field:value` tokens in it, so the bar, the
// URL and the widgets always say the same thing. A `status:` pick drops
// `is:open` / `is:closed`, which would otherwise AND with it.
//
// The status strip and the label picker are Ark UI parts (toggle group,
// combobox) skinned with daisyUI classes, the way the real IssueFilters is
// planned (PLAN.md step 6); the saved-filter chip is plain daisyUI.

interface Props {
  query: IssueQuery;
  labels: string[];
  matches: number;
  update(patch: Partial<IssueQuery>): void;
  reset(): void;
}

export function IssueFilters({ query, labels, matches, update, reset }: Props) {
  const q = query.q;
  const statuses = tagValues(q, "status")
    .map((w) => ALL_STATUSES.find((s) => statusLabel(s) === w))
    .filter((s): s is IssueStatus => s !== undefined);
  const isTags = tagValues(q, "is");
  const setQ = (next: string) => update({ q: next });

  return (
    <div class="space-y-2 border-b border-base-300 p-2">
      <ToggleGroup.Root
        multiple
        value={statuses}
        onValueChange={(d) => {
          // The strip toggles one status at a time; find which one flipped.
          const next = d.value as IssueStatus[];
          const flipped = ALL_STATUSES.find((s) => statuses.includes(s) !== next.includes(s));
          if (flipped) setQ(toggleTag(q, "status", statusLabel(flipped), ["is:open", "is:closed"]));
        }}
        className="flex flex-wrap gap-1"
        aria-label="Status"
      >
        {ALL_STATUSES.map((s) => (
          <ToggleGroup.Item
            key={s}
            value={s}
            className={`btn btn-sm ${statuses.includes(s) ? "btn-primary" : "btn-outline"}`}
            title={`status:${statusLabel(s)}`}
          >
            {statusLabel(s)}
          </ToggleGroup.Item>
        ))}
      </ToggleGroup.Root>
      <div class="px-0.5 text-xs opacity-50">
        {statuses.length > 0
          ? `status:${statuses.map(statusLabel).join(" status:")}`
          : isTags.includes("open")
            ? "is:open — open, in_progress and blocked"
            : isTags.includes("closed")
              ? "is:closed"
              : "any status"}
      </div>

      <div class="flex flex-wrap items-center gap-1 text-xs">
        <span class="opacity-60">saved</span>
        <button
          class={`btn btn-sm ${hasTag(q, "is", "plan") ? "btn-secondary" : "btn-outline"}`}
          title="is:plan"
          onClick={() => setQ(toggleTag(q, "is", "plan"))}
          data-testid="saved-plans"
        >
          Plans
        </button>
      </div>

      <LabelPicker labels={labels} selected={tagValues(q, "label")} onToggle={(l) => setQ(toggleTag(q, "label", l))} />

      <div class="flex items-center justify-between text-xs opacity-60">
        <span>
          {matches} issue{matches === 1 ? "" : "s"}, newest updated first
        </span>
        <button class="btn btn-ghost btn-sm" onClick={reset}>
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
  onToggle,
}: {
  labels: string[];
  selected: string[];
  onToggle(label: string): void;
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
      onValueChange={(d) => {
        // One pick per change; toggle the label that differs.
        const flipped = labels.find((l) => selected.includes(l) !== d.value.includes(l));
        if (flipped !== undefined) onToggle(flipped);
      }}
      onInputValueChange={(d) => filter(d.inputValue)}
      closeOnSelect={false}
      className="space-y-1"
    >
      {selected.length > 0 && (
        <div class="flex flex-wrap gap-1" data-testid="label-chips">
          {selected.map((l) => (
            <button key={l} class="badge badge-ghost badge-sm gap-1" onClick={() => onToggle(l)} title={`remove label:${l}`}>
              {l}
              <span aria-hidden="true">×</span>
            </button>
          ))}
        </div>
      )}
      <Combobox.Control className="join w-full">
        <Combobox.Input className="input input-md join-item w-full" placeholder="Filter by label" />
        <Combobox.Trigger className="btn btn-md join-item" aria-label="Open the label list">
          ▾
        </Combobox.Trigger>
      </Combobox.Control>
      <Portal>
        <Combobox.Positioner>
          {/* Not daisyUI's `menu`: it wraps a height-capped list into columns. */}
          <Combobox.Content className="z-50 max-h-64 w-64 overflow-auto rounded-box border border-base-300 bg-base-100 p-1 text-sm shadow-lg">
            <Combobox.Empty className="px-2 py-1 text-xs opacity-60">No label matches</Combobox.Empty>
            {collection.items.map((item) => (
              <Combobox.Item
                key={item.value}
                item={item}
                className="flex cursor-pointer items-center justify-between rounded px-2 py-1 data-[highlighted]:bg-base-200 data-[state=checked]:font-medium"
              >
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
