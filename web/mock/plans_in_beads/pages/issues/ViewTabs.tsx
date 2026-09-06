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

  return (
    <Tabs.Root
      value={onIssue ? null : query.view}
      onValueChange={(d) => go(d.value as View)}
      lazyMount
      unmountOnExit
      className="space-y-4"
    >
      <div class="flex flex-wrap items-center gap-3">
        <Tabs.List className="tabs tabs-box tabs-sm" aria-label="View">
          {ALL_VIEWS.map((v) => (
            <Tabs.Trigger key={v} value={v} className={`tab ${!onIssue && v === query.view ? "tab-active" : ""}`}>
              {VIEW_TITLES[v]}
            </Tabs.Trigger>
          ))}
        </Tabs.List>
        {onIssue && <span class="text-xs opacity-60">pick a view to go back</span>}
      </div>
      {onIssue
        ? detail
        : ALL_VIEWS.map((v) => (
            <Tabs.Content key={v} value={v} className="space-y-4">
              {views[v]}
            </Tabs.Content>
          ))}
    </Tabs.Root>
  );
}
