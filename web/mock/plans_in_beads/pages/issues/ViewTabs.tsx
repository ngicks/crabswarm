import { Tabs } from "@ark-ui/react/tabs";
import { useLocation } from "preact-iso";
import { ALL_VIEWS, type IssueQuery, type View, encodeIssueQuery } from "@/api/issues.js";
import { sourceHref } from "@/lib/paths.js";

// The view strip above the main column (D14): list, board and graph, switched
// through `?view=` on the source URL. An Ark UI tab list skinned as a daisyUI
// boxed tab strip; while an issue is open the strip stays and picking a view
// returns to the source URL with the filters kept.

const VIEW_TITLES: Record<View, string> = { list: "List", board: "Board", graph: "Graph" };

export function ViewTabs({
  sourceId,
  query,
  onIssue,
}: {
  sourceId: string;
  query: IssueQuery;
  /** Whether an issue detail is open instead of a view. */
  onIssue: boolean;
}) {
  const loc = useLocation();
  const go = (v: View) => loc.route(sourceHref(sourceId, encodeIssueQuery({ ...query, view: v })));

  return (
    <Tabs.Root value={onIssue ? null : query.view} onValueChange={(d) => go(d.value as View)} className="contents">
      <Tabs.List className="tabs tabs-box tabs-sm" role="tablist" aria-label="View">
        {ALL_VIEWS.map((v) => (
          <Tabs.Trigger key={v} value={v} className={`tab ${!onIssue && v === query.view ? "tab-active" : ""}`}>
            {VIEW_TITLES[v]}
          </Tabs.Trigger>
        ))}
      </Tabs.List>
    </Tabs.Root>
  );
}
