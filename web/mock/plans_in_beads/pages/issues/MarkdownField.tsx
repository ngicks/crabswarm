import { useEffect, useRef } from "preact/hooks";
import { theme } from "#src/signals/ui.js";
import type { RenderedField } from "@/api/client.js";
import { runMermaid } from "@/lib/mermaid.js";

// One rendered text field (description, design, acceptance, notes, close
// reason, comment) as the `.markdown-body` card the file browser uses for
// documents, with the same client-side mermaid pass as DocView (lib/mermaid.ts):
// the server renders ```mermaid fences to <pre class="mermaid"> (render.go,
// RenderModeClient) and the SPA draws them with its own bundled mermaid — no CDN.
//
// Unlike DocView this does not typeset MathJax (the mock has no /vendor/mathjax)
// and does not rewrite image sources (no /raw endpoint without a daemon).

/** Anchor id of a heading inside a field. Description and design are two whole
 *  markdown documents on one page, so their goldmark heading ids can collide;
 *  each field's anchors are namespaced by field name, and the TOC links match. */
export function fieldAnchor(prefix: string, id: string): string {
  return `${prefix}--${id}`;
}

export function MarkdownField({ field, prefix }: { field: RenderedField; prefix: string }) {
  const ref = useRef<HTMLElement>(null);
  // Read during render so this component subscribes to the theme signal:
  // mermaid bakes the theme into the SVG, so a toggle must re-run it.
  const dark = theme.value === "dark";

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.innerHTML = field.html;
    // goldmark-mermaid's client mode can append a CDN <script>; NoScript is on
    // in render.go, but strip anything that slipped through anyway.
    el.querySelectorAll("script").forEach((s) => s.remove());
    namespaceAnchors(el, prefix);
    void runMermaid(el, dark);
  }, [field.html, prefix, dark]);

  return (
    <div class="rounded-box border border-base-300 bg-base-100 p-5 shadow-sm sm:p-6">
      <article ref={ref} class="markdown-body" onClick={onArticleClick} />
    </div>
  );
}

function namespaceAnchors(el: HTMLElement, prefix: string): void {
  el.querySelectorAll<HTMLElement>("[id]").forEach((n) => {
    n.id = fieldAnchor(prefix, n.id);
  });
  el.querySelectorAll<HTMLAnchorElement>('a[href^="#"]').forEach((a) => {
    const href = a.getAttribute("href") ?? "";
    a.setAttribute("href", `#${fieldAnchor(prefix, href.slice(1))}`);
  });
}

function onArticleClick(e: MouseEvent): void {
  const a = (e.target as HTMLElement).closest("a");
  if (!a) return;
  const href = a.getAttribute("href") ?? "";
  if (!href.startsWith("#")) return;
  e.preventDefault();
  document.getElementById(decodeURIComponent(href.slice(1)))?.scrollIntoView({
    behavior: "smooth",
    block: "start",
  });
}
