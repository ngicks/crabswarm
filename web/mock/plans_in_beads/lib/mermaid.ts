// The client-side mermaid pass shared by every place that draws a diagram:
// rendered markdown fields (MarkdownField) and the dependency neighbourhood
// (IssueGraph, D15). The same pass as DocView's: the SPA's own bundled
// mermaid, loaded on first use, no CDN, securityLevel strict. The theme is
// baked into the SVG, so callers re-run this when the theme flips.

let mermaidMod: Promise<typeof import("mermaid")> | null = null;

export interface RunOptions {
  /** Drawing a generated graph rather than a document's diagram: natural
   *  size in a scrolling box instead of shrinking to the container's width
   *  (which makes labels unreadable), rounded edges, a readable font. */
  graph?: boolean;
}

/** Draws every `pre.mermaid` / `.mermaid` element under `el` in place. A
 *  diagram that fails to parse keeps its source text visible. */
export async function runMermaid(el: HTMLElement, dark: boolean, opts: RunOptions = {}): Promise<void> {
  const nodes = el.querySelectorAll<HTMLElement>("pre.mermaid, .mermaid");
  if (nodes.length === 0) return;
  mermaidMod ??= import("mermaid");
  try {
    const mermaid = (await mermaidMod).default;
    // Measuring before the page's fonts are usable sizes every label against
    // the fallback face.
    await document.fonts.ready;
    // The font is resolved here instead of left as `inherit`. Mermaid lays a
    // diagram out in a temporary container on <body> and moves the finished
    // SVG into place, and the SVG lands inside `pre.mermaid`, which Tailwind's
    // preflight puts in the monospace stack. `inherit` therefore measured in
    // the body's sans font and drew in a wider monospace one, so labels were
    // clipped by boxes sized for narrower text ("Plans in beads" drawn as
    // "Plan in bea"). An explicit family makes both steps use the same font.
    // The size is already explicit for the same reason.
    const bodyFont = getComputedStyle(document.body).fontFamily;
    mermaid.initialize({
      startOnLoad: false,
      theme: dark ? "dark" : "default",
      securityLevel: "strict",
      themeVariables: opts.graph ? { fontSize: "14px", fontFamily: bodyFont } : undefined,
      flowchart: opts.graph ? { useMaxWidth: false, curve: "basis", nodeSpacing: 28, rankSpacing: 56, padding: 10 } : undefined,
    });
    await mermaid.run({ nodes: Array.from(nodes) });
  } catch {
    // leave the raw diagram source visible on failure
  }
}
