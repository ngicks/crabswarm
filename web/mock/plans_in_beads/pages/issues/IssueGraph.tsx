import { useCallback, useEffect, useMemo, useRef } from "preact/hooks";
import { useLocation } from "preact-iso";
import { theme } from "#src/signals/ui.js";
import type { IssueEdge } from "@/api/client.js";
import { type GraphNode, flowchart } from "@/lib/graph.js";
import { runMermaid } from "@/lib/mermaid.js";
import { issueHref } from "@/lib/paths.js";
import { Section } from "./Section.js";

// The dependency neighbourhood on the detail page (D15): a mermaid flowchart
// built from ListDependencies edges and drawn by the same bundled mermaid
// documents use, at natural size inside a zoom / pan viewport. Nodes are
// coloured by status, the current issue in the primary pair, edges styled and
// labelled by type, and a click on a node opens the issue — through the
// rendered SVG's node ids, not mermaid's `click` directive, so securityLevel
// stays strict.
//
// The zoom / pan transform is hand-rolled. mermaid bundles d3-zoom, but
// reaching it means declaring d3-zoom and d3-selection as dependencies of the
// mock; a presentation mock should not force that dependency decision for
// forty lines of arithmetic.
//
// The transform lives in a ref and is written straight to `style`. Keeping it
// in state would re-render the component on every wheel tick and every
// pointermove, which re-runs the mermaid effect and throws away the drawing.

interface Transform {
  x: number;
  y: number;
  k: number;
}

const MIN_K = 0.25;
const MAX_K = 4;
const FIT_MARGIN = 16;
const CLICK_SLOP = 4;
/** A fit below this is too small to read, so a fresh render opens at 1:1 on
 *  the current issue instead of on the whole drawing. */
const READABLE_FIT = 0.6;

export function IssueGraph({
  sourceId,
  nodes,
  edges,
  search,
  title = "Neighbourhood",
  sectionKey,
  testId = "local-graph",
}: {
  sourceId: string;
  nodes: GraphNode[];
  edges: IssueEdge[];
  search: string;
  title?: string;
  /** Section key for the card's anchor id, so the TOC can reach the graph. */
  sectionKey?: string;
  testId?: string;
}) {
  const loc = useLocation();
  const viewport = useRef<HTMLDivElement>(null);
  const ref = useRef<HTMLDivElement>(null);
  const level = useRef<HTMLSpanElement>(null);
  const tf = useRef<Transform>({ x: 0, y: 0, k: 1 });
  // The scale the last fit asked for. Fit itself is never floored — a wide
  // epic must stay whole — and the floor the wheel and the buttons clamp to
  // follows it, so zooming out can always reach the overview and no further.
  const fitK = useRef(1);
  const dragged = useRef(false);
  // Redraws are queued: replacing the nodes while mermaid is still measuring
  // the previous drawing (a node click navigates mid-render) makes it lay
  // out detached elements and log NaN transforms.
  const queue = useRef(Promise.resolve());
  // Read during render so this component subscribes to the theme signal:
  // mermaid bakes the theme into the SVG, so a toggle must redraw.
  const dark = theme.value === "dark";
  const chart = useMemo(() => flowchart(nodes, edges), [nodes, edges]);
  const currentId = useMemo(() => nodes.find((n) => n.current)?.id, [nodes]);

  const clampK = useCallback((k: number) => Math.min(MAX_K, Math.max(Math.min(MIN_K, fitK.current), k)), []);

  const apply = useCallback(() => {
    const { x, y, k } = tf.current;
    if (ref.current) ref.current.style.transform = `translate(${x}px, ${y}px) scale(${k})`;
    if (level.current) level.current.textContent = `${Math.round(k * 100)}%`;
  }, []);

  const reset = useCallback(() => {
    tf.current = { x: 0, y: 0, k: 1 };
    apply();
  }, [apply]);

  /** The transform that puts the whole drawing in the box, centred, and the
   *  scale it needs remembered as the zoom floor. Null before mermaid has
   *  drawn anything. */
  const fitTransform = useCallback((): Transform | null => {
    const box = viewport.current;
    const svg = ref.current?.querySelector("svg");
    if (!box || !svg) return null;
    const rect = svg.getBoundingClientRect();
    // The current transform is still on screen, so divide it back out to get
    // the drawing's natural size.
    const w = rect.width / tf.current.k;
    const h = rect.height / tf.current.k;
    if (w === 0 || h === 0) return null;
    // Never magnify: blowing a two-node graph up to fill the box only blurs
    // its text.
    const k = Math.min(1, (box.clientWidth - 2 * FIT_MARGIN) / w, (box.clientHeight - 2 * FIT_MARGIN) / h);
    fitK.current = k;
    return { k, x: (box.clientWidth - w * k) / 2, y: (box.clientHeight - h * k) / 2 };
  }, []);

  const fit = useCallback(() => {
    const t = fitTransform();
    if (!t) return;
    tf.current = t;
    apply();
  }, [apply, fitTransform]);

  /** Puts the current issue's node in the middle of the box at 1:1. False
   *  when the node is not in the drawing (yet). */
  const centreOnCurrent = useCallback(() => {
    const box = viewport.current;
    const el = ref.current;
    if (!box || !el || currentId === undefined) return false;
    const g = Array.from(el.querySelectorAll<SVGGElement>("g.node")).find((n) => chart.issueIdOf(n.id) === currentId);
    if (!g) return false;
    const boxRect = box.getBoundingClientRect();
    const rect = g.getBoundingClientRect();
    const { x, y, k } = tf.current;
    // Client coordinates back to the drawing's own, so the maths holds
    // whatever transform is on screen.
    const cx = (rect.left + rect.width / 2 - boxRect.left - x) / k;
    const cy = (rect.top + rect.height / 2 - boxRect.top - y) / k;
    tf.current = { k: 1, x: box.clientWidth / 2 - cx, y: box.clientHeight / 2 - cy };
    apply();
    return true;
  }, [apply, chart, currentId]);

  const zoomBy = useCallback(
    (factor: number) => {
      const box = viewport.current;
      if (!box) return;
      const { x, y, k } = tf.current;
      const next = clampK(k * factor);
      // Zoom about the viewport's centre so the buttons keep what you are
      // looking at in view.
      const cx = box.clientWidth / 2;
      const cy = box.clientHeight / 2;
      tf.current = { k: next, x: cx - ((cx - x) / k) * next, y: cy - ((cy - y) / k) * next };
      apply();
    },
    [apply, clampK],
  );

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
      if (stale || !ref.current) return;
      // A neighbourhood small enough to stay legible opens fitted. A wide
      // epic would fit at a quarter size, where no label can be read, so it
      // opens at 1:1 on its own issue instead — the overview is one Fit away.
      const t = fitTransform();
      if (t && t.k >= READABLE_FIT) {
        tf.current = t;
        apply();
      } else if (!centreOnCurrent()) {
        fit();
      }
    });
    return () => {
      stale = true;
    };
  }, [apply, centreOnCurrent, chart, dark, fit, fitTransform]);

  useEffect(() => {
    const box = viewport.current;
    if (!box) return;
    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      const { x, y, k } = tf.current;
      const next = clampK(k * Math.exp(-e.deltaY * 0.002));
      const rect = box.getBoundingClientRect();
      // Keep the point under the cursor pinned.
      const px = e.clientX - rect.left;
      const py = e.clientY - rect.top;
      tf.current = { k: next, x: px - ((px - x) / k) * next, y: py - ((py - y) / k) * next };
      apply();
    };
    box.addEventListener("wheel", onWheel, { passive: false });
    return () => box.removeEventListener("wheel", onWheel);
  }, [apply, clampK]);

  const start = useRef({ x: 0, y: 0, panning: false });

  const onPointerDown = (e: PointerEvent) => {
    if (e.button !== 0) return;
    dragged.current = false;
    start.current = { x: e.clientX, y: e.clientY, panning: true };
  };

  const onPointerMove = (e: PointerEvent) => {
    const s = start.current;
    if (!s.panning) return;
    const dx = e.clientX - s.x;
    const dy = e.clientY - s.y;
    if (!dragged.current) {
      if (Math.hypot(dx, dy) < CLICK_SLOP) return;
      dragged.current = true;
      // Captured only once this is a drag: capturing on pointerdown would
      // retarget the click to the viewport and break node navigation.
      viewport.current?.setPointerCapture(e.pointerId);
      // Inline rather than a class: `cursor-grab` and `cursor-grabbing` have
      // the same specificity, so which one wins would depend on sheet order.
      if (viewport.current) viewport.current.style.cursor = "grabbing";
    }
    tf.current = { ...tf.current, x: tf.current.x + dx, y: tf.current.y + dy };
    start.current = { x: e.clientX, y: e.clientY, panning: true };
    apply();
  };

  const endPan = () => {
    start.current = { ...start.current, panning: false };
    if (viewport.current) viewport.current.style.cursor = "";
  };

  const onClick = (e: MouseEvent) => {
    if (dragged.current) return;
    const node = (e.target as Element).closest<SVGGElement>("g.node");
    if (!node) return;
    const id = chart.issueIdOf(node.id);
    if (id === undefined) return;
    e.preventDefault();
    loc.route(issueHref(sourceId, id, search));
  };

  const onDblClick = (e: MouseEvent) => {
    if ((e.target as Element).closest("g.node")) return;
    if (tf.current.k === 1) fit();
    else reset();
  };

  // The toolbar rides on the title's line; the legend takes the whole next one
  // (`basis-full` inside the strip's flex-wrap), which reads better than
  // letting a wide legend push the buttons onto a third line.
  const strip = (
    <>
      <span class="ml-auto flex items-center gap-1 text-xs opacity-70">
        {/* Written through the ref on every transform, so it carries no JSX children. */}
        <span ref={level} class="w-10 text-right tabular-nums" data-testid="graph-zoom-level" />
        <button type="button" class="btn btn-ghost btn-sm" data-testid="graph-zoom-out" title="zoom out" onClick={() => zoomBy(1 / 1.3)}>
          −
        </button>
        <button type="button" class="btn btn-ghost btn-sm" data-testid="graph-zoom-in" title="zoom in" onClick={() => zoomBy(1.3)}>
          +
        </button>
        <button type="button" class="btn btn-ghost btn-sm" data-testid="graph-zoom-reset" title="actual size" onClick={reset}>
          1:1
        </button>
        <button type="button" class="btn btn-ghost btn-sm" data-testid="graph-zoom-fit" title="fit to the box" onClick={fit}>
          Fit
        </button>
      </span>
      <div class="flex basis-full flex-wrap items-center gap-x-4 gap-y-1 text-xs opacity-70">
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
      </div>
    </>
  );

  return (
    <Section id={sectionKey} title={title} extra={strip} testId={testId}>
      <div
        ref={viewport}
        class="relative h-[22rem] min-h-40 cursor-grab touch-none select-none resize-y overflow-hidden [&_g.node]:cursor-pointer [&_pre.mermaid]:m-0 [&_pre.mermaid]:bg-transparent"
        onClick={onClick}
        onDblClick={onDblClick}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={endPan}
        onPointerCancel={endPan}
        onLostPointerCapture={endPan}
      >
        <div ref={ref} class="w-max origin-top-left" data-testid="graph-canvas" />
      </div>
    </Section>
  );
}
