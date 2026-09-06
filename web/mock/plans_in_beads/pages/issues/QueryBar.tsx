import { Combobox, createListCollection } from "@ark-ui/react/combobox";
import { Portal } from "@ark-ui/react/portal";
import { useEffect, useMemo, useRef, useState } from "preact/hooks";
import type { IssueQuery } from "@/api/issues.js";
import {
  DEFAULT_QUERY,
  QUALIFIERS,
  type SuggestContext,
  type Suggestion,
  parseQuery,
  replaceToken,
  suggest,
  tokenAt,
} from "@/api/query.js";

// The search bar, GitHub style: one text field holding the whole query
// (`is:open label:chat -label:tui foo`), suggestions for the token under the
// caret — qualifier names before the colon, values after it — and Enter to
// apply. liqe parses the text; api/query.ts gives the qualifiers meaning.
//
// The draft is local until Enter (or a suggestion pick, which keeps editing)
// so a half-typed qualifier does not empty the list under the reader. The
// URL's `q` is the applied query; the sidebar widgets edit that same text.
//
// Ark's combobox supplies the listbox behaviour (keyboard, ARIA); the text is
// controlled from here so the suggestion replaces one token, not the input.

export function QueryBar({
  query,
  matches,
  ctx,
  update,
  reset,
}: {
  query: IssueQuery;
  matches: number;
  ctx: SuggestContext;
  update(patch: Partial<IssueQuery>): void;
  reset(): void;
}) {
  const [draft, setDraft] = useState(query.q);
  const [caret, setCaret] = useState(query.q.length);
  const [highlighted, setHighlighted] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  // An outside change (sidebar widget, back button) replaces the draft.
  useEffect(() => {
    setDraft(query.q);
    setCaret(query.q.length);
  }, [query.q]);

  const token = tokenAt(draft, caret);
  const suggestions = useMemo(() => suggest(token.text, ctx), [token.text, ctx]);
  const collection = useMemo(
    () => createListCollection<Suggestion>({ items: suggestions, itemToValue: (s) => s.insert, itemToString: (s) => s.label }),
    [suggestions],
  );
  const parsed = parseQuery(draft);
  const dirty = draft !== query.q;

  const readCaret = () => {
    const el = inputRef.current;
    if (el) setCaret(el.selectionStart ?? el.value.length);
  };
  const apply = () => update({ q: draft.trim() });
  const accept = (insert: string) => {
    const next = replaceToken(draft, token, insert);
    const pos = token.start + insert.length;
    setDraft(next);
    setCaret(pos);
    // Put the caret after what was inserted, once the input has the text.
    requestAnimationFrame(() => {
      const el = inputRef.current;
      if (!el) return;
      el.focus();
      el.setSelectionRange(pos, pos);
    });
  };

  return (
    <div class="space-y-1" data-testid="query-bar">
      <Combobox.Root
        collection={collection}
        inputValue={draft}
        onInputValueChange={(d) => {
          setDraft(d.inputValue);
          // The caret moves with the edit; read it after the value lands.
          requestAnimationFrame(readCaret);
        }}
        value={[]}
        onValueChange={(d) => {
          if (d.value[0] !== undefined) accept(d.value[0]);
        }}
        onHighlightChange={(d) => setHighlighted(d.highlightedValue)}
        selectionBehavior="preserve"
        allowCustomValue
        openOnClick
        className="w-full"
      >
        {/* One daisyUI input box holds the text, the clear mark and the
            Search button, so a single border and focus ring enclose all. */}
        <Combobox.Control className="input input-md w-full gap-1 pr-1">
          <Combobox.Input
            ref={inputRef}
            className="grow bg-transparent font-mono outline-none"
            placeholder="is:open label:chat -label:tui type:epic priority:<2 free text"
            aria-label="Search issues"
            data-testid="query-input"
            onKeyUp={readCaret}
            onClick={readCaret}
            onKeyDown={(e: KeyboardEvent) => {
              // Enter with a highlighted suggestion is Ark's to insert; Enter
              // on plain text applies the query.
              if (e.key === "Enter" && highlighted === null) {
                e.preventDefault();
                apply();
              }
            }}
          />
          <button
            class="btn btn-ghost btn-xs btn-circle"
            onClick={reset}
            title={`clear, back to ${DEFAULT_QUERY}`}
            aria-label="Clear the query"
            data-testid="query-clear"
          >
            ×
          </button>
          <button class={`btn btn-xs ${dirty ? "btn-primary" : "btn-ghost"}`} onClick={apply} data-testid="query-apply">
            Search
          </button>
        </Combobox.Control>
        <Portal>
          <Combobox.Positioner>
            <Combobox.Content className="z-50 max-h-72 w-80 overflow-auto rounded-box border border-base-300 bg-base-100 p-1 text-sm shadow-lg">
              {suggestions.length === 0 ? (
                <div class="px-2 py-1 text-xs opacity-60">No suggestion for this token</div>
              ) : (
                suggestions.map((s) => (
                  <Combobox.Item
                    key={s.insert}
                    item={s}
                    className="flex cursor-pointer items-center justify-between gap-3 rounded px-2 py-1 data-[highlighted]:bg-base-200"
                    data-testid="query-suggestion"
                  >
                    <Combobox.ItemText className="font-mono">{s.label}</Combobox.ItemText>
                    {s.help !== "" && <span class="truncate text-xs opacity-60">{s.help}</span>}
                  </Combobox.Item>
                ))
              )}
            </Combobox.Content>
          </Combobox.Positioner>
        </Portal>
      </Combobox.Root>

      <div class="flex flex-wrap items-center gap-x-3 gap-y-1 px-1 text-xs opacity-70" data-testid="query-status">
        {parsed.error !== "" ? (
          <span class="text-error" data-testid="query-error">
            {parsed.error}
          </span>
        ) : parsed.unknown.length > 0 ? (
          <span class="text-warning" data-testid="query-unknown">
            unknown qualifier{parsed.unknown.length === 1 ? "" : "s"}: {parsed.unknown.join(", ")} — matches nothing
          </span>
        ) : (
          <span>
            {matches} issue{matches === 1 ? "" : "s"}
            {dirty ? ", press Enter to apply" : ""}
          </span>
        )}
        <span class="ml-auto">
          {QUALIFIERS.map((q) => (
            <code key={q.name} class="mr-1 font-mono" title={q.help}>
              {q.name}:
            </code>
          ))}
          <span>free text on title and id · -x negates · AND OR ( )</span>
        </span>
      </div>
    </div>
  );
}
