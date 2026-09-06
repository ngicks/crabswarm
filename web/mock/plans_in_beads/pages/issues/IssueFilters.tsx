import { Combobox, useListCollection } from "@ark-ui/react/combobox";
import { useFilter } from "@ark-ui/react/locale";
import { Portal } from "@ark-ui/react/portal";
import type { IssueQuery } from "@/api/issues.js";
import { tagValues, toggleTag } from "@/api/query.js";

// Left-pane quick filters. They keep no state of their own: the label picker
// reads its `label:` tags out of the search bar's query and toggles tokens in
// it, so the bar, the URL and the picker always say the same thing. The
// Open / Closed / Plans state lives above the table (StateButtons).
//
// The label picker is an Ark UI combobox skinned with daisyUI classes, the
// way the real IssueFilters is planned (PLAN.md step 6).

interface Props {
  query: IssueQuery;
  labels: string[];
  matches: number;
  update(patch: Partial<IssueQuery>): void;
  reset(): void;
}

export function IssueFilters({ query, labels, matches, update, reset }: Props) {
  const q = query.q;
  return (
    <div class="space-y-2 border-b border-base-300 p-2">
      <div class="px-2 text-xs font-semibold uppercase tracking-wide opacity-60">Labels</div>
      <LabelPicker labels={labels} selected={tagValues(q, "label")} onToggle={(l) => update({ q: toggleTag(q, "label", l) })} />

      <div class="flex items-center justify-between text-xs opacity-60">
        <span>
          {matches} issue{matches === 1 ? "" : "s"}, newest updated first
        </span>
        <button class="btn btn-ghost" onClick={reset}>
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
            <button key={l} class="badge badge-ghost gap-1" onClick={() => onToggle(l)} title={`remove label:${l}`}>
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
