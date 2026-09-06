import { useEffect, useRef } from "preact/hooks";
import type { RenderedField } from "@/api/gen/ngicks/crabswarm/issues/v1/issues_service_pb.js";
import { openLightbox, openSvgLightbox } from "@/components/Lightbox.js";
import { runMermaid } from "@/lib/mermaid.js";
import { theme } from "@/signals/preferences.js";

// One rendered text field (description, design, acceptance, notes, close
// reason, comment) as the `.markdown-body` article the file browser uses for
// documents — the card around it belongs to the enclosing Section — with the
// same client-side mermaid pass as DocumentView (lib/mermaid.ts): the server
// renders ```mermaid fences to <pre class="mermaid"> and the SPA draws them
// with its own bundled mermaid — no CDN.
//
// Unlike DocumentView this does not typeset MathJax and does not rewrite image
// sources: an issue field is not a file under a root, so there is no document
// directory to resolve a relative image against.

/** Anchor id of a heading inside a field. Description and design are two whole
 *  markdown documents on one page, so their goldmark heading ids can collide;
 *  each field's anchors are namespaced by field name, and the TOC links match. */
export function fieldAnchor(prefix: string, id: string): string {
  return `${prefix}--${id}`;
}

export function MarkdownField({
  field,
  prefix,
  class: className = "p-5 sm:p-6",
}: {
  field: RenderedField;
  prefix: string;
  /** Padding of the body inside its card. Comments sit tighter than a field. */
  class?: string;
}) {
  const ref = useRef<HTMLElement>(null);
  // Read during render so this component subscribes to the theme signal:
  // mermaid bakes the theme into the SVG, so a toggle must re-run it.
  const dark = theme.value === "dark";

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.innerHTML = field.html;
    // goldmark-mermaid's client mode can append a CDN <script>; NoScript is on
    // in the renderer, but strip anything that slipped through anyway.
    el.querySelectorAll("script").forEach((s) => s.remove());
    namespaceAnchors(el, prefix);
    void runMermaid(el, dark);
  }, [field.html, prefix, dark]);

  return <article ref={ref} class={`markdown-body ${className}`} onClick={onArticleClick} />;
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
  // Same branches as DocumentView's handler, so a diagram or an image inside
  // an issue field behaves the way it does in the file browser.
  //
  // A failed render leaves raw text (no <svg>) and falls through.
  const diagram = (e.target as HTMLElement).closest<HTMLElement>("pre.mermaid, .mermaid");
  const svg = diagram?.querySelector<SVGSVGElement>("svg");
  if (svg) {
    e.preventDefault();
    openSvgLightbox(svg);
    return;
  }
  // A linked image falls through so the anchor wins.
  const img = (e.target as HTMLElement).closest<HTMLImageElement>("img");
  if (img && !img.closest("a")) {
    e.preventDefault();
    openLightbox(img.src);
    return;
  }
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
