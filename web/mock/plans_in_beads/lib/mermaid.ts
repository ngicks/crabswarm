// The client-side mermaid pass shared by every place that draws a diagram:
// rendered markdown fields (MarkdownField) and the dependency graph
// (IssueGraph, D15). The same pass as DocView's: the SPA's own bundled mermaid, loaded
// on first use, no CDN, securityLevel strict. The theme is baked into the
// SVG, so callers re-run this when the theme flips.

let mermaidMod: Promise<typeof import("mermaid")> | null = null;

/** Draws every `pre.mermaid` / `.mermaid` element under `el` in place. A
 *  diagram that fails to parse keeps its source text visible. */
export async function runMermaid(el: HTMLElement, dark: boolean): Promise<void> {
  const nodes = el.querySelectorAll<HTMLElement>("pre.mermaid, .mermaid");
  if (nodes.length === 0) return;
  mermaidMod ??= import("mermaid");
  try {
    const mermaid = (await mermaidMod).default;
    mermaid.initialize({ startOnLoad: false, theme: dark ? "dark" : "default", securityLevel: "strict" });
    await mermaid.run({ nodes: Array.from(nodes) });
  } catch {
    // leave the raw diagram source visible on failure
  }
}
