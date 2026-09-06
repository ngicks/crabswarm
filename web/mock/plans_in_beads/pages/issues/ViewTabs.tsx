import { Tabs } from "@ark-ui/react/tabs";
import type { ComponentChildren } from "preact";
import { useLocation } from "preact-iso";
import { ALL_VIEWS, type IssueQuery, type View, encodeIssueQuery } from "@/api/issues.js";
import { sourceHref } from "@/lib/paths.js";

// The view tabs (D14): list, board and graph, switched through `?view=` on
// the source URL. Ark's Tabs owns both the strip and the bodies: each view is
// a Tabs.Content, mounted only while selected (mermaid should not draw a
// hidden graph), skinned as a daisyUI boxed tab strip.
//
// An open issue is not a tab: while one is open the strip shows no selection
// and the detail renders in place of the bodies; picking a tab returns to
// that view with the query string kept.

const VIEW_TITLES: Record<View, string> = { list: "List", board: "Board", graph: "Graph" };

export function ViewTabs({
  sourceId,
  query,
  detail,
  views,
}: {
  sourceId: string;
  query: IssueQuery;
  /** The open issue, rendered instead of the view bodies when set. */
  detail?: ComponentChildren;
  views: Record<View, ComponentChildren>;
}) {
  const loc = useLocation();
  const go = (v: View) => loc.route(sourceHref(sourceId, encodeIssueQuery({ ...query, view: v })));
  const onIssue = detail !== undefined;

  // Lifted tabs on the panel's top border, as the header draws Roots | Issues:
  // the active tab is painted in the panel's base-100 and overlaps the border
  // by 1px (-mb-px), so it reads as the panel's own label rather than as a
  // toggle floating above it. The first tab attaches at the panel's corner, so
  // that corner stays square while it is active.
  const first = !onIssue && query.view === ALL_VIEWS[0];

  return (
    <Tabs.Root value={onIssue ? null : query.view} onValueChange={(d) => go(d.value as View)} lazyMount unmountOnExit>
      <div class="flex flex-wrap items-end gap-3">
        <Tabs.List
          className="tabs tabs-lift tabs-md -mb-px [--tab-bg:var(--color-base-100)] [--tab-border-color:var(--color-base-300)]"
          aria-label="View"
        >
          {ALL_VIEWS.map((v) => (
            <Tabs.Trigger key={v} value={v} className={`tab ${!onIssue && v === query.view ? "tab-active" : ""}`}>
              {VIEW_TITLES[v]}
            </Tabs.Trigger>
          ))}
        </Tabs.List>
        {onIssue && <span class="pb-2 text-xs opacity-60">pick a view to go back</span>}
      </div>
      {onIssue ? (
        <div class="border-t border-base-300 pt-4">{detail}</div>
      ) : (
        <div
          class={`rounded-box border border-base-300 bg-base-100 p-4 shadow-sm ${first ? "rounded-tl-none" : ""}`}
          data-testid="view-panel"
        >
          {ALL_VIEWS.map((v) => (
            <Tabs.Content key={v} value={v} className="space-y-4">
              {views[v]}
            </Tabs.Content>
          ))}
        </div>
      )}
    </Tabs.Root>
  );
}
