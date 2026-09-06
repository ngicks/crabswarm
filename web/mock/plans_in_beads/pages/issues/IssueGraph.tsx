import { useEffect, useMemo, useRef } from "preact/hooks";
import { useLocation } from "preact-iso";
import { theme } from "#src/signals/ui.js";
import type { IssueEdge } from "@/api/client.js";
import { type GraphNode, flowchart } from "@/lib/graph.js";
import { runMermaid } from "@/lib/mermaid.js";
import { issueHref } from "@/lib/paths.js";

// The dependency neighbourhood on the detail page (D15): a mermaid flowchart
// built from ListDependencies edges and drawn by the same bundled mermaid
// documents use, at natural size in a scrolling box. Nodes are coloured by
// status, the current issue in the primary pair, edges styled and labelled
// by type, and a click on a node opens the issue — through the rendered
// SVG's node ids, not mermaid's `click` directive, so securityLevel stays
// strict.

export function IssueGraph({
  sourceId,
  nodes,
  edges,
  search,
  testId = "local-graph",
}: {
  sourceId: string;
  nodes: GraphNode[];
  edges: IssueEdge[];
  search: string;
  testId?: string;
}) {
  const loc = useLocation();
  const ref = useRef<HTMLDivElement>(null);
  // Redraws are queued: replacing the nodes while mermaid is still measuring
  // the previous drawing (a node click navigates mid-render) makes it lay
  // out detached elements and log NaN transforms.
  const queue = useRef(Promise.resolve());
  // Read during render so this component subscribes to the theme signal:
  // mermaid bakes the theme into the SVG, so a toggle must redraw.
  const dark = theme.value === "dark";
  const chart = useMemo(() => flowchart(nodes, edges), [nodes, edges]);

  useEffect(() => {
    let stale = false;
    queue.current = queue.current.then(async () => {
      const el = ref.current;
      if (stale || !el) return;
      // Rebuild the <pre> each time: mermaid replaces it with an SVG, and a
      // redraw (new edges, theme flip) needs the source back.
      el.replaceChildren();
      const pre = document.createElement("pre");
      pre.className = "mermaid";
      pre.textContent = chart.source;
      el.append(pre);
      await runMermaid(el, dark, { graph: true });
    });
    return () => {
      stale = true;
    };
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
    <div class="rounded-box border border-base-300 bg-base-100 shadow-sm" data-testid={testId}>
      <div class="flex flex-wrap items-center gap-x-4 gap-y-1 border-b border-base-300 px-4 py-2 text-xs opacity-70">
        <span>
          {nodes.length} issue{nodes.length === 1 ? "" : "s"}, {edges.length} edge{edges.length === 1 ? "" : "s"}
        </span>
        <span class="flex items-center gap-1">
          <span class="inline-block h-0 w-5 border-t-2 border-current" /> blocks
        </span>
        <span class="flex items-center gap-1">
          <span class="inline-block h-0 w-5 border-t-2 border-dotted border-current" /> parent → child, origin → discovered
        </span>
        <span class="flex items-center gap-1">
          <span class="inline-block h-0 w-5 border-t border-current" /> related
        </span>
        <span class="ml-auto">click a node to open it</span>
      </div>
      <div ref={ref} class="overflow-auto px-4 py-3 [&_g.node]:cursor-pointer [&_pre.mermaid]:m-0 [&_pre.mermaid]:bg-transparent" onClick={onClick} />
    </div>
  );
}
