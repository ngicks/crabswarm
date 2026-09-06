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
    mermaid.initialize({
      startOnLoad: false,
      theme: dark ? "dark" : "default",
      securityLevel: "strict",
      themeVariables: opts.graph ? { fontSize: "14px", fontFamily: "inherit" } : undefined,
      flowchart: opts.graph ? { useMaxWidth: false, curve: "basis", nodeSpacing: 28, rankSpacing: 56, padding: 10 } : undefined,
    });
    await mermaid.run({ nodes: Array.from(nodes) });
  } catch {
    // leave the raw diagram source visible on failure
  }
}
