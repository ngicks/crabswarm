import type * as preact from "preact";
import { useEffect, useMemo, useRef } from "preact/hooks";
import { useLocation } from "preact-iso";
import { theme } from "#src/signals/ui.js";
import type { IssueEdge } from "@/api/client.js";
import { type GraphNode, flowchart } from "@/lib/graph.js";
import { runMermaid } from "@/lib/mermaid.js";
import { issueHref } from "@/lib/paths.js";

// The dependency graph (D14, D15): a mermaid flowchart built from
// ListDependencies edges and drawn by the same bundled mermaid documents use.
// Nodes are coloured by status, edges styled and labelled by type, and a
// click on a node opens the issue — through the rendered SVG's node ids, not
// mermaid's `click` directive, so securityLevel stays strict.
//
// mermaid's layout is static and slows past a few hundred nodes, so the view
// draws the filtered set only and stops at `cap` nodes with a warning.

export const DEFAULT_CAP = 150;

export function IssueGraph({
  sourceId,
  nodes,
  edges,
  search,
  cap = DEFAULT_CAP,
  testId = "issue-graph",
  toolbar,
}: {
  sourceId: string;
  nodes: GraphNode[];
  edges: IssueEdge[];
  search: string;
  cap?: number;
  testId?: string;
  /** Extra controls for the line above the drawing (the view's toggles). */
  toolbar?: preact.ComponentChildren;
}) {
  const loc = useLocation();
  const ref = useRef<HTMLDivElement>(null);
  // Read during render so this component subscribes to the theme signal:
  // mermaid bakes the theme into the SVG, so a toggle must redraw.
  const dark = theme.value === "dark";

  const capped = nodes.length > cap;
  const shown = capped ? nodes.slice(0, cap) : nodes;
  const chart = useMemo(() => flowchart(shown, edges), [shown, edges]);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    // Rebuild the <pre> each time: mermaid replaces it with an SVG, and a
    // redraw (new edges, theme flip) needs the source back.
    el.replaceChildren();
    const pre = document.createElement("pre");
    pre.className = "mermaid";
    pre.textContent = chart.source;
    el.append(pre);
    void runMermaid(el, dark);
  }, [chart, dark]);

  const onClick = (e: MouseEvent) => {
    const node = (e.target as Element).closest<SVGGElement>("g.node");
    if (!node) return;
    const id = chart.issueIdOf(node.id);
    if (id === undefined) return;
    e.preventDefault();
    loc.route(issueHref(sourceId, id, search));
  };

  return (
    <div class="space-y-2" data-testid={testId}>
      {capped && (
        <div class="alert alert-warning text-xs" role="alert" data-testid="graph-cap">
          {nodes.length} issues match; drawing the first {cap}. Narrow the filters to see the rest.
        </div>
      )}
      <div class="flex flex-wrap items-center gap-3 text-xs opacity-70">
        {toolbar}
        <span>
          {shown.length} node{shown.length === 1 ? "" : "s"}, {edges.length} edge{edges.length === 1 ? "" : "s"}
        </span>
        <span>solid: blocks · dotted: parent → child, origin → discovered · line: related</span>
        <span>click a node to open it</span>
      </div>
      {nodes.length === 0 ? (
        <div class="p-3 text-sm opacity-60">Nothing to draw.</div>
      ) : (
        <div
          ref={ref}
          class="overflow-auto rounded-box border border-base-300 bg-base-100 p-3 shadow-sm [&_g.node]:cursor-pointer"
          onClick={onClick}
        />
      )}
    </div>
  );
}
